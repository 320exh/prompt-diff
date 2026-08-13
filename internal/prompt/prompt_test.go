package prompt

import (
	"strings"
	"testing"
)

func TestParseFull(t *testing.T) {
	src := `---
name: Customer Support Classifier
version: 2.1.0
models:
  - gpt-4o-mini
  - llama3.1:8b
variables:
  - user_query
  - historical_context
---
You are an expert customer support agent.
Analyze the user query: {{ user_query }}

Context history:
{{ historical_context }}

Return JSON with classification and confidence score.
`
	p, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Name != "Customer Support Classifier" {
		t.Errorf("name = %q", p.Name)
	}
	if p.Version != "2.1.0" {
		t.Errorf("version = %q", p.Version)
	}
	if len(p.Models) != 2 || p.Models[1] != "llama3.1:8b" {
		t.Errorf("models = %v", p.Models)
	}
	if len(p.Variables) != 2 || p.Variables[0] != "user_query" {
		t.Errorf("variables = %v", p.Variables)
	}
	if !strings.Contains(p.Body, "{{ user_query }}") {
		t.Errorf("body missing variable: %q", p.Body)
	}
}

func TestParseCRLFAndBOM(t *testing.T) {
	src := "\xEF\xBB\xBF---\r\nname: X\r\nversion: 1.0.0\r\n---\r\nHello {{ name }}\r\n"
	p, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Name != "X" {
		t.Errorf("name = %q", p.Name)
	}
	if strings.Contains(p.Body, "\r") {
		t.Errorf("body still contains CR: %q", p.Body)
	}
	t.Log("parsed", p.Name, p.Version)
}

func TestParseVariableSetCleaning(t *testing.T) {
	src := `---
name: X
variables:
  - used
  - used
---
Hi {{ used }} and {{ missing }}
`
	p, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// missing is not declared -> not exposed; duplicates are removed
	if len(p.Variables) != 1 || p.Variables[0] != "used" {
		t.Errorf("variables = %v", p.Variables)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("no frontmatter here")); err == nil {
		t.Error("want ErrNoFrontmatter, got nil")
	}
	if _, err := Parse([]byte("---\nname: X\n")); err == nil {
		t.Error("want ErrMalformedFrontmatter, got nil")
	}
}

func TestSections(t *testing.T) {
	p, _ := Parse([]byte("---\nname: X\n---\nOne.\n\nTwo.\n\nThree."))
	sections := p.Sections()
	if len(sections) != 3 || sections[0] != "One." {
		t.Errorf("sections = %v", sections)
	}
}