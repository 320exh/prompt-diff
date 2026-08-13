package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/320exh/prompt-diff/internal/cost"
	"github.com/320exh/prompt-diff/internal/prompt"
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

var rawVarRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

func runLint(cmd *cobra.Command, args []string) error {
	var totalFindings int
	for _, path := range args {
		findings, err := lintFile(path)
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

func lintFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var findings []string

	// Raw scan for {{ var }} usage, independent of prompt.Parse's own
	// declared-only filtering, so we can catch undeclared vars it drops.
	rawUsed := map[string]bool{}
	for _, m := range rawVarRe.FindAllStringSubmatch(string(src), -1) {
		rawUsed[m[1]] = true
	}

	t, err := prompt.Parse(src)
	if err != nil {
		return []string{err.Error()}, nil
	}

	if t.Name == "" {
		findings = append(findings, "missing frontmatter field: name")
	}
	if t.Version == "" {
		findings = append(findings, "missing frontmatter field: version")
	}
	if len(t.Models) == 0 {
		findings = append(findings, "missing frontmatter field: models")
	}

	declared := map[string]bool{}
	for _, v := range t.Variables {
		declared[v] = true
	}
	usedDeclared := map[string]bool{}
	for _, m := range rawVarRe.FindAllStringSubmatch(t.Body, -1) {
		usedDeclared[m[1]] = true
	}

	var undeclared []string
	for name := range rawUsed {
		if !declared[name] {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	for _, name := range undeclared {
		findings = append(findings, fmt.Sprintf("undeclared variable used in body: {{ %s }}", name))
	}

	var unused []string
	for _, name := range t.Variables {
		if !usedDeclared[name] {
			unused = append(unused, name)
		}
	}
	sort.Strings(unused)
	for _, name := range unused {
		findings = append(findings, fmt.Sprintf("declared-but-unused variable: %s", name))
	}

	for _, m := range t.Models {
		if !cost.IsKnown(m) {
			findings = append(findings, fmt.Sprintf("model not in built-in cost table (using a family-default estimate): %s", m))
		}
	}

	for i, line := range strings.Split(t.Body, "\n") {
		if line != strings.TrimRight(line, " \t") {
			findings = append(findings, fmt.Sprintf("trailing whitespace on body line %d (wastes tokens)", i+1))
		}
	}

	return findings, nil
}
