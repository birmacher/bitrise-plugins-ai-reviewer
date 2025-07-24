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

var ciSummaryCmd = &cobra.Command{
	Use:   "ci-summary",
	Short: "Summarize CI failures using AI",
	Long:  `Analyze CI failures and provide summary using AI capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Starting CI Summary Agent 🤖")

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

		llmClient, err := llm.NewLLM(provider, model, llm.WithEnabledTools(llm.EnabledTools{
			GetCIBuildLog:  llm.ToolTypeInitalizer,
			ListDirectory:  llm.ToolTypeHelper,
			GitDiff:        llm.ToolTypeHelper,
			ReadFile:       llm.ToolTypeHelper,
			SearchCodebase: llm.ToolTypeHelper,
			GitBlame:       llm.ToolTypeHelper,
			PostCISummary:  llm.ToolTypeFinalizer,
		}))
		if err != nil {
			return fmt.Errorf("failed to create client for provider: %v", err)
		}

		// Agent usage
		toolsAvailable := llmClient.GetEnabledTools().UseString([]string{
			llm.ToolTypeInitalizer,
			llm.ToolTypeHelper,
			llm.ToolTypeFinalizer,
		})

		// Setup the prompt
		req := llm.Request{
			SystemPrompt: prompt.GetSystemPrompt(settings) + "\n" + prompt.CISummaryToolPrompt(toolsAvailable),
			UserPrompt:   prompt.CISummaryPrompt(ciProvider, buildID, appID, commitHash),
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
	rootCmd.AddCommand(ciSummaryCmd)

	// LLM
	ciSummaryCmd.Flags().StringP("provider", "p", "openai", "LLM provider to use for summarization")
	ciSummaryCmd.Flags().StringP("model", "m", "gpt-4.1", "LLM model to use for summarization")
}
