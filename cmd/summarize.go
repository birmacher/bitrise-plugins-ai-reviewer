package cmd

import (
	"encoding/json"
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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running AI code review...")

		codeReviewerName, _ := cmd.Flags().GetString("code-review")
		repo, _ := cmd.Flags().GetString("repo")

		repoTags := strings.Split(repo, "/")
		if len(repoTags) != 2 {
			fmt.Println("Repository must be in the format 'owner/repo'")
			return
		}
		repoOwner := repoTags[0]
		repoName := repoTags[1]

		prStr, _ := cmd.Flags().GetString("pr")
		pr, err := strconv.Atoi(prStr)
		if err != nil {
			fmt.Printf("Invalid PR number: %v\n", err)
			return
		}

		var gitProvider review.Reviewer

		if codeReviewerName != "" {
			gitProvider, err = review.NewReviewer(codeReviewerName)
			if err != nil {
				fmt.Printf("Failed to create Client for Review Provider: %v\n", err)
				return
			}

			err = gitProvider.PostSummary(repoOwner, repoName, pr, common.Summary{})
			if err != nil {
				fmt.Printf("Error posting review: %v\n", err)
				return
			}
		}

		// Get git diff
		commitHash, _ := cmd.Flags().GetString("commit")
		targetBranch, _ := cmd.Flags().GetString("branch")

		git := git.NewClient(git.NewDefaultRunner("."))

		commitHash, err = git.GetCommitHash(commitHash)
		if err != nil {
			fmt.Printf("Error getting commit hash: %v\n", err)
			return
		}

		diff, err := git.GetDiff(commitHash, targetBranch)

		if err != nil {
			fmt.Printf("Error getting diff with parent: %v\n", err)
			return
		}

		// Get the file contents
		fileContent, err := git.GetFileContents(commitHash, targetBranch)
		if err != nil {
			fmt.Printf("Error getting file contents: %v\n", err)
			return
		}

		reviewComments, err := gitProvider.GetReviewRequestComments(repoOwner, repoName, pr)
		if err != nil {
			fmt.Printf("Error getting review comments: %v\n", err)
			return
		}

		// Setup LLM client
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model)
		if err != nil {
			fmt.Printf("Failed to create Client for LLM Provider: %v\n", err)
			return
		}

		// Setup the prompt
		req := llm.Request{
			SystemPrompt:   prompt.GetSystemPrompt(),
			UserPrompt:     prompt.GetSummarizePrompt(),
			Diff:           prompt.GetDiffPrompt(diff),
			FileContents:   prompt.GetFileContentPrompt(fileContent),
			ReviewComments: prompt.GetLineLevelPrompt(reviewComments),
		}

		// Send the prompt and get the response
		resp := llmClient.Prompt(req)
		if resp.Error != nil {
			fmt.Printf("Error getting response: %v\n", resp.Error)
			return
		}

		fmt.Println("LLM Response:\n", resp.Content)

		// Send to the review provider
		if codeReviewerName != "" {
			summary := common.Summary{}
			err = json.Unmarshal([]byte(resp.Content), &summary)
			if err != nil {
				fmt.Printf("Error parsing response: %v\n", err)
				return
			}

			err = gitProvider.PostSummary(repoOwner, repoName, pr, summary)
			if err != nil {
				fmt.Printf("Error posting review: %v\n", err)
				return
			}

			lineLevel := common.LineLevelFeedback{}
			err = json.Unmarshal([]byte(resp.Content), &lineLevel)
			if err != nil {
				fmt.Printf("Error parsing response: %v\n", err)
				return
			}

			for idx, ll := range lineLevel.Lines {
				lineNumber, err := common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.FirstLine())
				if err != nil {
					fmt.Printf("Error getting line number for file %s: %v\n", ll.File, err)
					continue
				}
				lineLevel.Lines[idx].LineNumber = lineNumber

				if ll.IsMultiline() {
					lastLineNumber, err := common.GetLineNumber(ll.File, []byte(fileContent), []byte(diff), ll.LastLine())
					if err != nil {
						fmt.Printf("Error getting line number for file %s: %v\n", ll.File, err)
						continue
					}

					if lastLineNumber > lineLevel.Lines[idx].LineNumber {
						lineLevel.Lines[idx].LastLineNumber = lastLineNumber
					}

				}
			}

			err = gitProvider.PostLineFeedback(git, repoOwner, repoName, pr, commitHash, lineLevel)
			if err != nil {
				fmt.Printf("Error posting line feedback: %v\n", err)
				return
			}

			fmt.Println("Review posted successfully!")
		}
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
