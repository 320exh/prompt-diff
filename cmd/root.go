package cmd

import (
	"fmt"
	"os"

	"github.com/320exh/prompt-diff/internal/browser"
	"github.com/320exh/prompt-diff/internal/config"
	"github.com/320exh/prompt-diff/internal/cost"
	"github.com/320exh/prompt-diff/internal/uiserver"
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
	// Running with no subcommand (e.g. double-clicking the binary) launches
	// the dashboard directly, so the tool is usable without the CLI.
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := uiserver.New(8080)
		ln, port, err := srv.Listen()
		if err != nil {
			return err
		}
		url := fmt.Sprintf("http://localhost:%d", port)
		fmt.Printf("prompt-diff dashboard running at %s\n", url)
		fmt.Println("Close this window to stop.")
		_ = browser.Open(url)
		return srv.ServeOn(ln)
	},
}

// Execute runs the root command.
func Execute() {
	if cfg, err := config.Load(); err == nil {
		for model, price := range cfg.PriceOverrides {
			cost.SetOverride(model, price)
		}
	}
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
