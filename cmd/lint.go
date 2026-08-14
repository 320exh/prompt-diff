package cmd

import (
	"fmt"

	"github.com/320exh/prompt-diff/internal/lint"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint <file.prompt> [file2.prompt...]",
	Short: "Static checks on .prompt files (zero API cost)",
	Long: `Check .prompt files for common mistakes: undeclared {{ vars }} used in the
body, declared-but-unused vars, missing frontmatter fields, unknown model
names, and trailing whitespace. Exits nonzero if any file has findings, so it
is safe to use as a CI gate.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	var totalFindings int
	for _, path := range args {
		findings, err := lint.File(path)
		if err != nil {
			return fmt.Errorf("linting %s: %w", path, err)
		}
		if len(findings) == 0 {
			continue
		}
		totalFindings += len(findings)
		fmt.Printf("%s:\n", path)
		for _, f := range findings {
			fmt.Printf("  - %s\n", f)
		}
	}
	if totalFindings == 0 {
		fmt.Printf("%d file(s) OK, no findings\n", len(args))
		return nil
	}
	fmt.Printf("\n%d finding(s) across %d file(s)\n", totalFindings, len(args))
	return fmt.Errorf("lint failed")
}
