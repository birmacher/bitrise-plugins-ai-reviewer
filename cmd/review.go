package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review code changes using AI",
	Long:  `Analyze code changes and provide feedback using AI capabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running AI code review...")
		// Implement review logic here
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	// Add flags specific to review command
	reviewCmd.Flags().StringP("pr", "p", "", "Pull request URL or ID to review")
	reviewCmd.Flags().StringP("branch", "b", "", "Branch to review")
}
