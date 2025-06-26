package cmd

import (
	"fmt"
	"os"

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
		var llm model.LLM
		var err error
		switch provider {
		case "openai":
			// Create a new OpenAI model
			llm, err = model.NewOpenAI(apiKey,
				model.WithModel(model),
				model.WithMaxTokens(4000),
				model.WithAPITimeout(60),
			)
		default:
			fmt.Printf("Unsupported provider: %s\n", provider)
			return
		}

		if err != nil {
			fmt.Printf("Failed to create Client for LLM Provider: %v\n", err)
			return
		}

		// Create a request
		req := model.Request{
			SystemPrompt: prompt.GetSystemPrompt(),
			UserPrompt:   prompt.GetSummarizePrompt(),
			Diff:         prompt.GetDiffPrompt(""),
		}

		// Send the prompt and get the response
		resp := llm.Prompt(req)
		if resp.Error != nil {
			fmt.Printf("Error getting response: %v\n", resp.Error)
			return
		}

		fmt.Println("Response from LLM:")
		fmt.Println(resp.Content)
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)

	// Add flags specific to review command
	summarizeCmd.Flags().StringP("provider", "p", "openai", "LLM provider to use for summarization")
	summarizeCmd.Flags().StringP("model", "m", "gpt-4o", "LLM model to use for summarization")
	summarizeCmd.Flags().StringP("pr", "p", "", "Pull request URL or ID to review")
	summarizeCmd.Flags().StringP("branch", "b", "", "Branch to review")
}
