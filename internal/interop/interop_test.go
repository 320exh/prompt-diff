package interop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/prompt"
)

func TestToLangChainRoundTrip(t *testing.T) {
	orig := &prompt.Template{
		Name:      "Support Classifier",
		Variables: []string{"user_context"},
		Body:      "You are a support agent.\nContext: {{ user_context }}",
	}
	lc := ToLangChain(orig)
	if lc.Template != "You are a support agent.\nContext: {user_context}" {
		t.Errorf("template = %q", lc.Template)
	}
	if lc.TemplateFormat != "f-string" {
		t.Errorf("template_format = %q", lc.TemplateFormat)
	}

	data, err := json.Marshal(lc)
	if err != nil {
		t.Fatal(err)
	}
	back, err := FromLangChain(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Body != orig.Body {
		t.Errorf("round-tripped body = %q, want %q", back.Body, orig.Body)
	}
	if len(back.Variables) != 1 || back.Variables[0] != "user_context" {
		t.Errorf("round-tripped variables = %v", back.Variables)
	}
}

func TestFromLangChainMissingTemplate(t *testing.T) {
	_, err := FromLangChain([]byte(`{"input_variables": []}`))
	if err == nil || !strings.Contains(err.Error(), "no template") {
		t.Errorf("err = %v, want missing-template error", err)
	}
}

func TestFromLangChainLeavesUndeclaredBracesAlone(t *testing.T) {
	back, err := FromLangChain([]byte(`{"input_variables": ["x"], "template": "Use {x} not {y}."}`))
	if err != nil {
		t.Fatal(err)
	}
	if back.Body != "Use {{ x }} not {y}." {
		t.Errorf("body = %q", back.Body)
	}
}

func TestToLlamaIndexRoundTrip(t *testing.T) {
	orig := &prompt.Template{
		Variables: []string{"query"},
		Body:      "Answer the query: {{ query }}",
	}
	li := ToLlamaIndex(orig)
	if li.Template != "Answer the query: {query}" {
		t.Errorf("template = %q", li.Template)
	}

	data, err := json.Marshal(li)
	if err != nil {
		t.Fatal(err)
	}
	back, err := FromLlamaIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Body != orig.Body {
		t.Errorf("round-tripped body = %q, want %q", back.Body, orig.Body)
	}
}

func TestFromLlamaIndexMissingTemplate(t *testing.T) {
	_, err := FromLlamaIndex([]byte(`{"template_vars": []}`))
	if err == nil || !strings.Contains(err.Error(), "no template") {
		t.Errorf("err = %v, want missing-template error", err)
	}
}
