// Package interop converts between prompt-diff's .prompt format and the
// JSON prompt-template shapes used by LangChain and LlamaIndex, so prompts
// authored in (or exported from) those frameworks can be diffed, evaluated,
// and compressed through prompt-diff too.
//
// Both frameworks serialize templates as an f-string body ("{var}") rather
// than prompt-diff's "{{ var }}" — conversion rewrites between the two. This
// targets the common single-template save/load shape (LangChain's
// PromptTemplate.save()/load_prompt(), LlamaIndex's PromptTemplate); it does
// not cover multi-message ChatPromptTemplate or provider-specific schema
// variants, since those aren't a single stable JSON shape across versions.
package interop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/320exh/prompt-diff/internal/prompt"
)

// LangChain is the PromptTemplate.save()/load_prompt() JSON shape.
type LangChain struct {
	InputVariables []string `json:"input_variables"`
	Template       string   `json:"template"`
	TemplateFormat string   `json:"template_format,omitempty"`
	Type           string   `json:"_type,omitempty"`
}

// LlamaIndex is the PromptTemplate JSON shape (template + declared vars).
type LlamaIndex struct {
	Template     string   `json:"template"`
	TemplateVars []string `json:"template_vars"`
}

var fstringVarRe = regexp.MustCompile(`\{([A-Za-z0-9_.-]+)\}`)

// toFString rewrites {{ var }} placeholders to f-string {var} placeholders.
func toFString(body string) string {
	return varRe.ReplaceAllString(body, "{$1}")
}

var varRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// fromFString rewrites {var} placeholders back to {{ var }}, for every name
// in vars. Braces not matching a declared variable are left untouched so
// literal JSON/curly text in the template body doesn't get mangled.
func fromFString(body string, vars []string) string {
	declared := map[string]bool{}
	for _, v := range vars {
		declared[v] = true
	}
	return fstringVarRe.ReplaceAllStringFunc(body, func(m string) string {
		name := fstringVarRe.FindStringSubmatch(m)[1]
		if !declared[name] {
			return m
		}
		return "{{ " + name + " }}"
	})
}

// ToLangChain converts a prompt-diff template to LangChain's PromptTemplate
// JSON shape.
func ToLangChain(t *prompt.Template) LangChain {
	return LangChain{
		InputVariables: t.Variables,
		Template:       toFString(t.Body),
		TemplateFormat: "f-string",
		Type:           "prompt",
	}
}

// FromLangChain parses LangChain's PromptTemplate JSON shape into a
// prompt-diff template.
func FromLangChain(data []byte) (*prompt.Template, error) {
	var lc LangChain
	if err := json.Unmarshal(data, &lc); err != nil {
		return nil, fmt.Errorf("parsing LangChain prompt: %w", err)
	}
	if lc.Template == "" {
		return nil, fmt.Errorf("LangChain prompt has no template field")
	}
	return &prompt.Template{
		Variables: lc.InputVariables,
		Body:      strings.TrimSpace(fromFString(lc.Template, lc.InputVariables)),
	}, nil
}

// ToLlamaIndex converts a prompt-diff template to LlamaIndex's PromptTemplate
// JSON shape.
func ToLlamaIndex(t *prompt.Template) LlamaIndex {
	return LlamaIndex{
		Template:     toFString(t.Body),
		TemplateVars: t.Variables,
	}
}

// FromLlamaIndex parses LlamaIndex's PromptTemplate JSON shape into a
// prompt-diff template.
func FromLlamaIndex(data []byte) (*prompt.Template, error) {
	var li LlamaIndex
	if err := json.Unmarshal(data, &li); err != nil {
		return nil, fmt.Errorf("parsing LlamaIndex prompt: %w", err)
	}
	if li.Template == "" {
		return nil, fmt.Errorf("LlamaIndex prompt has no template field")
	}
	return &prompt.Template{
		Variables: li.TemplateVars,
		Body:      strings.TrimSpace(fromFString(li.Template, li.TemplateVars)),
	}, nil
}

// Convert parses data as `from` and re-encodes it as `to`. Formats are
// "prompt", "langchain", or "llamaindex". Shared by `prompt-diff convert`
// and the web workspace.
func Convert(data []byte, from, to string) ([]byte, error) {
	var t *prompt.Template
	var err error
	switch from {
	case "prompt":
		t, err = prompt.Parse(data)
	case "langchain":
		t, err = FromLangChain(data)
	case "llamaindex":
		t, err = FromLlamaIndex(data)
	default:
		return nil, fmt.Errorf("unknown source format %q (want prompt, langchain, or llamaindex)", from)
	}
	if err != nil {
		return nil, fmt.Errorf("parsing as %s: %w", from, err)
	}

	switch to {
	case "prompt":
		rendered, rerr := prompt.Render(t, t.Body)
		if rerr != nil {
			return nil, fmt.Errorf("rendering .prompt: %w", rerr)
		}
		return []byte(rendered), nil
	case "langchain":
		return json.MarshalIndent(ToLangChain(t), "", "  ")
	case "llamaindex":
		return json.MarshalIndent(ToLlamaIndex(t), "", "  ")
	default:
		return nil, fmt.Errorf("unknown target format %q (want prompt, langchain, or llamaindex)", to)
	}
}
