package cmd

import (
	"fmt"

	"github.com/birmacher/bitrise-plugins-ai-reviewer/git"
	"github.com/birmacher/bitrise-plugins-ai-reviewer/llm"
	"github.com/birmacher/bitrise-plugins-ai-reviewer/prompt"
	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Summarize code changes using AI",
	Long:  `Analyze code changes and provide summary using AI capabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running AI code review...")

		// Get provider flag
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model)
		if err != nil {
			fmt.Printf("Failed to create Client for LLM Provider: %v\n", err)
			return
		}

		commitHash, _ := cmd.Flags().GetString("commit")
		targetBranch, _ := cmd.Flags().GetString("branch")

		git := git.NewClient(git.NewDefaultRunner("."))
		diff, err := git.GetDiff(commitHash, targetBranch)

		if err != nil {
			fmt.Printf("Error getting diff with parent: %v\n", err)
			return
		}

		// Create a request
		req := llm.Request{
			SystemPrompt: prompt.GetSystemPrompt(),
			UserPrompt:   prompt.GetSummarizePrompt(),
			Diff:         prompt.GetDiffPrompt(diff),
		}

		// Send the prompt and get the response
		resp := llmClient.Prompt(req)
		if resp.Error != nil {
			fmt.Printf("Error getting response: %v\n", resp.Error)
			return
		}

		fmt.Println("")
		fmt.Println("Response from LLM:")
		fmt.Println(resp.Content)
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)

	// Add flags specific to review command
	summarizeCmd.Flags().StringP("provider", "p", "openai", "LLM provider to use for summarization")
	summarizeCmd.Flags().StringP("model", "m", "gpt-4o", "LLM model to use for summarization")
	summarizeCmd.Flags().StringP("commit", "c", "", "Analyze changes in the specified commit's perspective")
	summarizeCmd.Flags().Lookup("commit").NoOptDefVal = "HEAD"
	summarizeCmd.Flags().StringP("branch", "b", "", "Target Branch to merge with")
}
