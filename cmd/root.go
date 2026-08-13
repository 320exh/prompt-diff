package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags; default for dev builds.
var Version = "v1.0.2"

var rootCmd = &cobra.Command{
	Use:   "prompt-diff",
	Short: "Version, diff, and benchmark LLM system prompts",
	Long: `prompt-diff is a fast, git-native CLI & local Web UI to version, diff
tokens/costs, and benchmark LLM system prompts across local and cloud models.

It treats system prompts as first-class source code: parse .prompt templates,
diff token/cost impact across git commits, and run multi-model test matrices.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(runsCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
