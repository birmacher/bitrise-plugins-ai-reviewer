package cmd

import (
	"fmt"

	"github.com/birmacher/bitrise-plugins-ai-reviewer/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the version of this plugin`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Bitrise AI Reviewer Plugin v%s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
