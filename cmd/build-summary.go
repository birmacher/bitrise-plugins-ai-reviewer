package cmd

import (
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/ci"
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

		var ciProvider string
		var appID string
		var buildID string
		var commitHash string

		_, isBitrise := os.LookupEnv("BITRISE_IO")
		if isBitrise {
			ciProvider = "bitrise"
			var err error

			if appID, err = ci.GetAppID(); err != nil {
				return err
			}
			if buildID, err = ci.GetBuildID(); err != nil {
				return err
			}
			if commitHash, err = ci.GetCommitHash(); err != nil {
				return err
			}
		}

		if ciProvider == "" {
			return fmt.Errorf("CI provider is not set")
		}
		logger.Info("CI provider:", ciProvider)

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
			UserPrompt:   prompt.GetBuildSummaryPrompt(ciProvider, buildID, appID, commitHash),
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
}
