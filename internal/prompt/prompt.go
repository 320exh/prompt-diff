// Package prompt parses .prompt templates.
//
// A .prompt file is YAML frontmatter followed by freeform text:
//
//	---
//	name: Customer Support Classifier
//	version: 2.1.0
//	models:
//	  - gpt-4o-mini
//	variables:
//	  - user_query
//	---
//	You are an expert customer support agent.
//	Analyze the user query: {{ user_query }}
package prompt

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template is a parsed .prompt file.
type Template struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Models      []string `yaml:"models"`
	Variables   []string `yaml:"variables"`
	Body        string   `yaml:"-"`
	RawVariable string   `yaml:"variableSet"` // any additional user keys are ignored
}

// ErrNoFrontmatter is returned when a file has no `---` delimiter.
var ErrNoFrontmatter = errors.New("no frontmatter found (expected a leading `---` block)")

// ErrMalformedFrontmatter is returned when the frontmatter block is not terminated.
var ErrMalformedFrontmatter = errors.New("malformed frontmatter: missing closing `---`")

var varRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// Parse splits frontmatter from the body, parses the YAML, and extracts every
// {{ variable }} referenced in the body.
func Parse(src []byte) (*Template, error) {
	s := string(bytes.TrimPrefix(src, []byte{0xEF, 0xBB, 0xBF}))
	s = strings.ReplaceAll(s, "\r\n", "\n")

	if !strings.HasPrefix(s, "---\n") {
		return nil, ErrNoFrontmatter
	}
	rest := s[len("---\n"):]
	if !strings.Contains(rest, "\n---\n") && !strings.HasPrefix(rest, "---\n") {
		// allow a lone closing delimiter at the very end
		if !strings.HasSuffix(rest, "\n---") {
			return nil, ErrMalformedFrontmatter
		}
	}

	// Split on the first closing `---` line.
	sep := strings.Index(rest, "\n---\n")
	var fm, body string
	if sep >= 0 {
		fm, body = rest[:sep], rest[sep+len("\n---\n"):]
	} else {
		body = ""
		fm = rest
	}

	var t Template
	if err := yaml.Unmarshal([]byte(fm), &t); err != nil {
		return nil, err
	}
	t.Body = strings.TrimSpace(strings.TrimRight(body, "\n "))

	declared := map[string]bool{}
	for _, v := range t.Variables {
		declared[v] = true
	}
	// Only surface {{ var }} names declared in frontmatter (ignore unknown).
	seen := map[string]bool{}
	for _, m := range varRe.FindAllStringSubmatch(t.Body, -1) {
		name := m[1]
		if declared[name] && !seen[name] {
			seen[name] = true
			t.Variables = append(t.Variables, name)
		}
	}
	// De-duplicate in case frontmatter repeats a name.
	var out []string
	have := map[string]bool{}
	for _, v := range t.Variables {
		if !have[v] {
			have[v] = true
			out = append(out, v)
		}
	}
	t.Variables = out
	return &t, nil
}

// Sections splits the body into paragraph-sized sections on blank lines.
func (t *Template) Sections() []string {
	var out []string
	for _, para := range strings.Split(t.Body, "\n\n") {
		para = strings.TrimSpace(para)
		if para != "" {
			out = append(out, para)
		}
	}
	return out
}