package cmd

import (
	"fmt"

	"github.com/320exh/prompt-diff/internal/gitutil"
	"github.com/spf13/cobra"
)

var hookMaxDelta int

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage the prompt-diff git pre-commit hook",
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a pre-commit hook that prints token/cost delta for staged .prompt files",
	Long: `Installs a git pre-commit hook that runs "prompt-diff diff" against HEAD for
every staged .prompt file, printing the token/cost delta. With --max-delta,
the commit is blocked when any file's token delta exceeds the threshold.`,
	Args: cobra.NoArgs,
	RunE: runHookInstall,
}

func init() {
	hookInstallCmd.Flags().IntVar(&hookMaxDelta, "max-delta", 0, "block the commit if a staged .prompt file's token delta exceeds this (0 = never block)")
	hookCmd.AddCommand(hookInstallCmd)
	rootCmd.AddCommand(hookCmd)
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	hookPath, err := gitutil.InstallHook(hookMaxDelta)
	if err != nil {
		return err
	}
	fmt.Printf("Installed pre-commit hook at %s\n", hookPath)
	if hookMaxDelta > 0 {
		fmt.Printf("Commits will be blocked if a staged .prompt file's token delta exceeds %d\n", hookMaxDelta)
	}
	return nil
}
