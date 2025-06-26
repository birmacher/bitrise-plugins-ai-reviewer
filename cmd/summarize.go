package cmd

import (
	"fmt"
	"os"

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

		// Get API key from environment
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("OPENAI_API_KEY environment variable not set")
			return
		}

		// Get provider flag
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		var llmClient llm.LLM
		var err error
		switch provider {
		case "openai":
			// Create a new OpenAI model
			llmClient, err = llm.NewOpenAI(apiKey,
				llm.WithModel(model),
				llm.WithMaxTokens(4000),
				llm.WithAPITimeout(60),
			)
		default:
			fmt.Printf("Unsupported provider: %s\n", provider)
			return
		}

		if err != nil {
			fmt.Printf("Failed to create Client for LLM Provider: %v\n", err)
			return
		}

		fmt.Println("")
		fmt.Println("Using LLM Provider:", provider)
		fmt.Println("With Model:", model)

		git := git.NewClient(git.NewDefaultRunner("."))
		diff := ""

		commitFlag := cmd.Flags().Changed("commit")
		if commitFlag {
			commit_hash, _ := cmd.Flags().GetString("commit")
			if commit_hash == "" {
				commit_hash, err = git.GetCurrentCommitHash()
				if err != nil {
					fmt.Printf("Error getting current commit hash: %v\n", err)
					return
				}
			}

			diff, err = git.GetDiffWithParent(commit_hash)
			if err != nil {
				fmt.Printf("Error getting diff with parent: %v\n", err)
				return
			}
		}

		if diff == "" {
			fmt.Println("No diff found. Please provide a commit hash or ensure there are changes to summarize.")
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
	summarizeCmd.Flags().StringP("commit", "c", "", "Analyze changes in the specified commit (optional, uses current commit if not provided)")
	summarizeCmd.Flags().Lookup("commit").NoOptDefVal = "HEAD"
	// summarizeCmd.Flags().StringP("pr", "p", "", "Pull request URL or ID to review")
	// summarizeCmd.Flags().StringP("branch", "b", "", "Branch to review")
}
