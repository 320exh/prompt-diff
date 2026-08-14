// Package lint runs static checks on .prompt sources: undeclared or unused
// {{ vars }}, missing frontmatter fields, unknown model names, and trailing
// whitespace. Shared by `prompt-diff lint` and the web workspace.
package lint

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/320exh/prompt-diff/internal/cost"
	"github.com/320exh/prompt-diff/internal/prompt"
)

var rawVarRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// File lints the .prompt file at path.
func File(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Source(src), nil
}

// Source lints raw .prompt source. A parse error is itself the only finding.
func Source(src []byte) []string {
	var findings []string

	// Raw scan for {{ var }} usage, independent of prompt.Parse's own
	// declared-only filtering, so we can catch undeclared vars it drops.
	rawUsed := map[string]bool{}
	for _, m := range rawVarRe.FindAllStringSubmatch(string(src), -1) {
		rawUsed[m[1]] = true
	}

	t, err := prompt.Parse(src)
	if err != nil {
		return []string{err.Error()}
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

	return findings
}
