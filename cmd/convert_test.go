package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/interop"
)

func TestRunConvertPromptToLangChain(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(in, []byte("---\nname: Test\nvariables:\n  - x\n---\nHello {{ x }}.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.json")

	convertIn, convertFrom, convertTo, convertOut = in, "prompt", "langchain", out
	if err := runConvert(convertCmd, nil); err != nil {
		t.Fatalf("runConvert: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var lc interop.LangChain
	if err := json.Unmarshal(data, &lc); err != nil {
		t.Fatal(err)
	}
	if lc.Template != "Hello {x}." {
		t.Errorf("template = %q", lc.Template)
	}
}

func TestRunConvertLangChainToPrompt(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.json")
	if err := os.WriteFile(in, []byte(`{"input_variables":["x"],"template":"Hello {x}.","template_format":"f-string","_type":"prompt"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.prompt")

	convertIn, convertFrom, convertTo, convertOut = in, "langchain", "prompt", out
	if err := runConvert(convertCmd, nil); err != nil {
		t.Fatalf("runConvert: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Hello {{ x }}.") {
		t.Errorf("output = %q", data)
	}
}

func TestRunConvertUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(in, []byte("---\nname: Test\n---\nHi.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	convertIn, convertFrom, convertTo, convertOut = in, "prompt", "bogus", filepath.Join(dir, "out")
	if err := runConvert(convertCmd, nil); err == nil || !strings.Contains(err.Error(), "unknown --to") {
		t.Errorf("err = %v, want unknown-format error", err)
	}
}
