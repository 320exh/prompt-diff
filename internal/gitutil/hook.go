package gitutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildHookScript returns the pre-commit hook script `prompt-diff hook
// install` writes. With maxDelta > 0 the hook blocks commits whose staged
// .prompt token delta exceeds it.
func BuildHookScript(maxDelta int) string {
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

// InstallHook writes the pre-commit hook into the current repo's hooks dir
// and returns the hook path. It refuses to overwrite a pre-commit hook it
// did not install itself.
func InstallHook(maxDelta int) (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return "", err
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")

	script := BuildHookScript(maxDelta)
	if existing, err := os.ReadFile(hookPath); err == nil && !bytes.Contains(existing, []byte("prompt-diff hook install")) {
		return "", fmt.Errorf("%s already exists and was not installed by prompt-diff; remove it first or add the check manually", hookPath)
	}
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		return "", err
	}
	return hookPath, nil
}
