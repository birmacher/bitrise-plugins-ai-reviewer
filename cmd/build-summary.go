package cmd

import (
	"errors"
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

		ci, _ := cmd.Flags().GetString("code-review")
		appID, _ := cmd.Flags().GetString("app-id")
		buildID, _ := cmd.Flags().GetString("build-id")
		logger.Info("CI provider:", ci)

		// Setup LLM client
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")

		llmClient, err := llm.NewLLM(provider, model)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create Client for LLM Provider: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		// Setup the prompt
		req := llm.Request{
			SystemPrompt: prompt.GetSystemPrompt(settings, cmd.Use),
			UserPrompt:   prompt.GetBuildSummaryPrompt(buildID, appID),
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
	buildSummaryCmd.Flags().StringP("app-id", "", "", "Build ID to summarize")
	buildSummaryCmd.Flags().StringP("build-id", "", "", "Build ID to summarize")
}
