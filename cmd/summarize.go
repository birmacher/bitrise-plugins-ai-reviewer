package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/llm"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/prompt"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Summarize code changes using AI",
	Long:  `Analyze code changes and provide summary using AI capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Running AI code review...")

		// Parse settings from command line flags
		settings := parseSettings()
		logger.Debugf("Using settings: %+v", settings)

		codeReviewerName, _ := cmd.Flags().GetString("code-review")
		repo, _ := cmd.Flags().GetString("repo")
		logger.Info("Code review provider:", codeReviewerName)
		logger.Info("Repository:", repo)

		repoTags := strings.Split(repo, "/")
		if len(repoTags) != 2 {
			errMsg := "repository must be in the format 'owner/repo'"
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		repoOwner := repoTags[0]
		repoName := repoTags[1]

		prStr, _ := cmd.Flags().GetString("pr")
		pr, err := strconv.Atoi(prStr)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to parse PR number: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}
		logger.Infof("Pull Request: %d", pr)

		var gitProvider review.Reviewer

		if codeReviewerName != "" {
			gitProvider, err = review.NewReviewer(codeReviewerName)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to create Client for Review Provider: %v", err)
				logger.Errorf(errMsg)
				return errors.New(errMsg)
			}

			err = gitProvider.PostSummaryUnderReview(repoOwner, repoName, pr, common.Summary{}.Header())
			if err != nil {
				errMsg := fmt.Sprintf("Error posting initial review: %v", err)
				logger.Errorf(errMsg)
				return errors.New(errMsg)
			}
		}

		// Get git diff
		commitHash, _ := cmd.Flags().GetString("commit")
		targetBranch, _ := cmd.Flags().GetString("branch")

		git := git.NewClient(git.NewDefaultRunner("."))

		commitHash, err = git.GetCommitHash(commitHash)
		if err != nil {
			errMsg := fmt.Sprintf("Error getting commit hash: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		diff, err := git.GetDiff(commitHash, targetBranch)

		if err != nil {
			errMsg := fmt.Sprintf("Error getting diff with parent: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		// Get the file contents
		fileContent, err := git.GetFileContents(commitHash, targetBranch)
		if err != nil {
			errMsg := fmt.Sprintf("Error getting file contents: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		// Setup LLM client
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model, llm.WithEnabledTools(llm.EnabledTools{
			GetPullRequestDetails: llm.ToolTypeInitalizer,
			GetGitDiff:            llm.ToolTypeHelper,
			ListDirectory:         llm.ToolTypeHelper,
			ReadFile:              llm.ToolTypeHelper,
			SearchCodebase:        llm.ToolTypeHelper,
			GetGitBlame:           llm.ToolTypeHelper,
			PostLineFeedback:      llm.ToolTypeHelper,
			PostSummary:           llm.ToolTypeFinalizer,
		}))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create Client for LLM Provider: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		if gitProvider != nil {
			llmClient.SetGitProvider(&gitProvider)
		}
		llmClient.SetSettings(&settings)

		// Setup the prompt
		req := llm.Request{
			SystemPrompt: prompt.GetSystemPrompt(settings, cmd.Use),
			UserPrompt:   prompt.GetSummarizePrompt(settings, repoOwner, repoName, prStr, commitHash, targetBranch),
		}

		// Send the prompt and get the response
		resp := llmClient.Prompt(req)
		if resp.Error != nil {
			errMsg := fmt.Sprintf("Error getting response from LLM: %v", resp.Error)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		logger.Debug("LLM Response:")
		logger.Debug(resp.Content)

		// Send to the review provider
		if codeReviewerName != "" {
			lineLevel := common.LineLevelFeedback{
				Lines: llmClient.GetLineFeedback(),
			}
			for idx, ll := range lineLevel.Lines {
				// Get the line numbers
				lineNumber, err := common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.FirstLine())
				var lastLineNumber int

				firstLineFound := (err == nil && lineNumber > 0)
				lastLineFound := false
				isMultiline := ll.IsMultiline()

				if !firstLineFound {
					logger.Warnf("Error finding first line '%s' in file %s: %v", ll.FirstLine(), ll.File, err)
				}

				if isMultiline {
					lastLineNumber, err = common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.LastLine())
					lastLineFound = (err == nil && lastLineNumber > 0)
					if !lastLineFound {
						logger.Warnf("Error finding last line '%s' in file %s: %v", ll.LastLine(), ll.File, err)
					}
				}

				if !firstLineFound && !lastLineFound {
					logger.Warnf("Skipping review for file %s, no valid line numbers found in diff", ll.File)
					continue
				}

				if firstLineFound {
					lineLevel.Lines[idx].LineNumber = lineNumber
					if isMultiline && lastLineFound {
						if lastLineNumber > lineLevel.Lines[idx].LineNumber {
							lineLevel.Lines[idx].LastLineNumber = lastLineNumber
						}
					}
				}

				if !firstLineFound && isMultiline && lastLineFound {
					lineLevel.Lines[idx].LineNumber = lastLineNumber
					// Clear suggestion as we have moved the starting line number
					// and it won't be correct anymore
					lineLevel.Lines[idx].Suggestion = ""
				}

				// Get the file content to look up indentation
				if ll.Suggestion != "" {
					suggestionLines := strings.Split(ll.Suggestion, "\n")

					// Get the file content to determine indentation
					fileSource, err := git.GetFileContent(commitHash, ll.File)
					if err != nil {
						errMsg := fmt.Sprintf("Error getting file content for '%s': %v", ll.File, err)
						logger.Errorf(errMsg)
						return errors.New(errMsg)
					}
					fileIndentation := common.GetIndentationString(fileSource)
					logger.Debug("Detected indentation for file '", ll.File, "': '", fileIndentation, "'")

					// Get the file diff to check if the change is for the diff
					fileDiff, err := git.GetDiffForFile(commitHash, ll.File)
					if err != nil {
						errMsg := fmt.Sprintf("Error getting diff for file '%s': %v", ll.File, err)
						logger.Errorf(errMsg)
						return errors.New(errMsg)
					}

					// Check if the line is in the diff and get the original line
					logger.Debug("Checking original line for '", ll.FirstLine(), "' in file '", ll.File, "'")
					originalLine := common.GetFullLine(fileContent, fileDiff, ll.FirstLine())
					if originalLine == "" {
						logger.Warnf("No original line found for '%s' in file '%s'", ll.FirstLine(), ll.File)
						continue
					}
					logger.Debug("Original line found: '", originalLine, "'")

					// Get the base indentation of the original line
					suggestionLines = common.FixIndentation(fileIndentation, originalLine, suggestionLines)
					ll.Suggestion = strings.Join(suggestionLines, "\n")
				}
			}

			err = gitProvider.PostLineFeedback(git, repoOwner, repoName, pr, commitHash, lineLevel)
			if err != nil {
				errMsg := fmt.Sprintf("Error posting line feedback: %v", err)
				logger.Errorf(errMsg)
				return errors.New(errMsg)
			}

			logger.Info("Review posted successfully!")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)

	// LLM
	summarizeCmd.Flags().StringP("provider", "p", "openai", "LLM provider to use for summarization")
	summarizeCmd.Flags().StringP("model", "m", "gpt-4.1", "LLM model to use for summarization")
	// Git
	summarizeCmd.Flags().StringP("commit", "c", "", "Analyze changes in the specified commit's perspective")
	summarizeCmd.Flags().Lookup("commit").NoOptDefVal = "HEAD"
	summarizeCmd.Flags().StringP("branch", "b", "", "Target Branch to merge with")
	// Code Review
	summarizeCmd.Flags().StringP("code-review", "r", "", "Code review provider to use (e.g., github, gitlab)")
	summarizeCmd.Flags().StringP("repo", "", "", "Repository name in the format 'owner/repo' (e.g., 'my-org/my-repo')")
	summarizeCmd.Flags().StringP("pr", "", "", "Pull Request number to post the review to")
}
