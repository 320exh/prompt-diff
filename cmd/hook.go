package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")

	script := buildHookScript(hookMaxDelta)
	if existing, err := os.ReadFile(hookPath); err == nil && !bytes.Contains(existing, []byte("prompt-diff hook install")) {
		return fmt.Errorf("%s already exists and was not installed by prompt-diff; remove it first or add the check manually", hookPath)
	}
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		return err
	}
	fmt.Printf("Installed pre-commit hook at %s\n", hookPath)
	if hookMaxDelta > 0 {
		fmt.Printf("Commits will be blocked if a staged .prompt file's token delta exceeds %d\n", hookMaxDelta)
	}
	return nil
}

func buildHookScript(maxDelta int) string {
	return fmt.Sprintf(`#!/bin/sh
# Installed by "prompt-diff hook install". Do not edit by hand; re-run the
# install command to update.
staged=$(git diff --cached --name-only --diff-filter=ACM -- '*.prompt')
if [ -z "$staged" ]; then
  exit 0
fi

blocked=0
for f in $staged; do
  echo "prompt-diff: $f"
  prompt-diff diff "$f" --json > /tmp/prompt-diff-hook.json 2>/tmp/prompt-diff-hook.err
  if [ $? -ne 0 ]; then
    cat /tmp/prompt-diff-hook.err >&2
    continue
  fi
  delta=$(grep -o '"token_delta":[^,]*' /tmp/prompt-diff-hook.json | head -1 | cut -d: -f2)
  echo "  token delta: $delta"
  max=%d
  if [ "$max" -gt 0 ] 2>/dev/null; then
    abs=${delta#-}
    if [ "$abs" -gt "$max" ] 2>/dev/null; then
      echo "  BLOCKED: token delta $delta exceeds --max-delta $max" >&2
      blocked=1
    fi
  fi
done
rm -f /tmp/prompt-diff-hook.json /tmp/prompt-diff-hook.err
exit $blocked
`, maxDelta)
}
