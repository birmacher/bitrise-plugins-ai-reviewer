package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/llm"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/prompt"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Summarize code changes using AI",
	Long:  `Analyze code changes and provide summary using AI capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running AI code review...")

		// Parse settings from command line flags
		settings := parseSettings()

		codeReviewerName, _ := cmd.Flags().GetString("code-review")
		repo, _ := cmd.Flags().GetString("repo")

		repoTags := strings.Split(repo, "/")
		if len(repoTags) != 2 {
			return errors.New("repository must be in the format 'owner/repo'")
		}
		repoOwner := repoTags[0]
		repoName := repoTags[1]

		prStr, _ := cmd.Flags().GetString("pr")
		pr, err := strconv.Atoi(prStr)
		if err != nil {
			return fmt.Errorf("invalid PR number: %v", err)
		}

		var gitProvider review.Reviewer

		if codeReviewerName != "" {
			gitProvider, err = review.NewReviewer(codeReviewerName)
			if err != nil {
				return fmt.Errorf("failed to create Client for Review Provider: %v", err)
			}

			emptySummary := common.Summary{}
			err = gitProvider.PostSummary(repoOwner, repoName, pr, emptySummary.Header(), emptySummary.InitiatedString())
			if err != nil {
				return fmt.Errorf("error posting review: %v", err)
			}
		}

		// Get git diff
		commitHash, _ := cmd.Flags().GetString("commit")
		targetBranch, _ := cmd.Flags().GetString("branch")

		git := git.NewClient(git.NewDefaultRunner("."))

		commitHash, err = git.GetCommitHash(commitHash)
		if err != nil {
			return fmt.Errorf("error getting commit hash: %v", err)
		}

		diff, err := git.GetDiff(commitHash, targetBranch)

		if err != nil {
			return fmt.Errorf("error getting diff with parent: %v", err)
		}

		// Get the file contents
		fileContent, err := git.GetFileContents(commitHash, targetBranch)
		if err != nil {
			return fmt.Errorf("error getting file contents: %v", err)
		}

		// Get existing review comments
		lineLevelFeedback, err := gitProvider.GetReviewRequestComments(repoOwner, repoName, pr)
		if err != nil {
			return fmt.Errorf("error getting existing review comments: %v", err)
		}

		// Setup LLM client
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model)
		if err != nil {
			return fmt.Errorf("failed to create Client for LLM Provider: %v", err)
		}

		// Setup the prompt
		req := llm.Request{
			SystemPrompt:      prompt.GetSystemPrompt(settings),
			UserPrompt:        prompt.GetSummarizePrompt(settings),
			Diff:              prompt.GetDiffPrompt(diff),
			FileContents:      prompt.GetFileContentPrompt(fileContent),
			LineLevelFeedback: prompt.GetLineLevelFeedbackPrompt(lineLevelFeedback),
		}

		// Send the prompt and get the response
		resp := llmClient.Prompt(req)
		if resp.Error != nil {
			return fmt.Errorf("error getting response: %v", resp.Error)
		}

		fmt.Println("LLM Response:\n", resp.Content)

		// Send to the review provider
		if codeReviewerName != "" {
			summary := common.Summary{}
			if err = json.Unmarshal([]byte(resp.Content), &summary); err != nil {
				return fmt.Errorf("error parsing summary response: %v", err)
			}

			err = gitProvider.PostSummary(repoOwner, repoName, pr, summary.Header(), summary.String(settings))
			if err != nil {
				return fmt.Errorf("error posting review: %v", err)
			}

			lineLevel := common.LineLevelFeedback{}
			if err = json.Unmarshal([]byte(resp.Content), &lineLevel); err != nil {
				return fmt.Errorf("error parsing line-level response: %v", err)
			}

			for idx, ll := range lineLevel.Lines {
				lineNumber, err := common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.FirstLine())
				lastLineNumber := 0

				firstLineFound := (err == nil && lineNumber > 0)
				lastLineFound := false

				if !firstLineFound {
					fmt.Printf("Error finding first line '%s' in file %s: %v\n", ll.FirstLine(), ll.File, err)
				}

				if ll.IsMultiline() {
					lastLineNumber, err = common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.LastLine())
					lastLineFound = (err == nil && lastLineNumber > 0)
					if !lastLineFound {
						fmt.Printf("Error finding last line '%s' in file %s: %v\n", ll.LastLine(), ll.File, err)
					}
				}

				if !firstLineFound && !lastLineFound {
					fmt.Printf("⚠️ Skipping review for file %s, no valid line numbers found in diff\n", ll.File)
					continue
				}

				if firstLineFound {
					lineLevel.Lines[idx].LineNumber = lineNumber
					if ll.IsMultiline() && lastLineFound {
						if lastLineNumber > lineLevel.Lines[idx].LineNumber {
							lineLevel.Lines[idx].LastLineNumber = lastLineNumber
						}
					}
				}

				if !firstLineFound && ll.IsMultiline() && lastLineFound {
					lineLevel.Lines[idx].LineNumber = lastLineNumber
				}
			}

			err = gitProvider.PostLineFeedback(git, repoOwner, repoName, pr, commitHash, lineLevel)
			if err != nil {
				return fmt.Errorf("error posting line feedback: %v", err)
			}

			fmt.Println("Review posted successfully!")
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

func parseSettings() common.Settings {
	return common.WithYamlFile()
}
