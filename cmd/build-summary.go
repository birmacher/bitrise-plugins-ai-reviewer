package cmd

import (
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/llm"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/prompt"
	"github.com/spf13/cobra"
)

var buildSummaryCmd = &cobra.Command{
	Use:   "build-summary",
	Short: "Summarize build failures using AI",
	Long:  `Analyze build failures and provide summary using AI capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Running AI build summary...")

		// Parse settings from command line flags
		settings := parseSettings()
		logger.Debugf("Using settings: %+v", settings)

		ci, _ := cmd.Flags().GetString("ci")
		appID, _ := cmd.Flags().GetString("app-id")
		buildID, _ := cmd.Flags().GetString("build-id")
		commitHash, _ := cmd.Flags().GetString("commit")
		logger.Info("CI provider:", ci)

		// Setup LLM client
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model)
		if err != nil {
			return fmt.Errorf("failed to create client for provider: %v", err)
		}

		// Setup the prompt
		req := llm.Request{
			SystemPrompt: prompt.GetSystemPrompt(settings, cmd.Use),
			UserPrompt:   prompt.GetBuildSummaryPrompt(ci, buildID, appID, commitHash),
		}

		// Send the prompt and get the response
		resp := llmClient.Prompt(req)
		if resp.Error != nil {
			return fmt.Errorf("error getting response from provider: %v", resp.Error)
		}

		logger.Debug("LLM Response:")
		logger.Debug(resp.Content)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildSummaryCmd)

	// LLM
	buildSummaryCmd.Flags().StringP("provider", "p", "openai", "LLM provider to use for summarization")
	buildSummaryCmd.Flags().StringP("model", "m", "gpt-4.1", "LLM model to use for summarization")
	// CI
	buildSummaryCmd.Flags().StringP("ci", "", "bitrise", "CI provider to use for build summary")
	buildSummaryCmd.Flags().StringP("app-id", "", "", "App ID for the build")
	buildSummaryCmd.Flags().StringP("build-id", "", "", "Build ID to summarize")
	buildSummaryCmd.Flags().StringP("commit", "c", "", "Analyze changes in the specified commit's perspective")
	buildSummaryCmd.Flags().Lookup("commit").NoOptDefVal = "HEAD"
}
