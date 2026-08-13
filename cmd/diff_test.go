package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/prompt"
)

func TestDisplayRef(t *testing.T) {
	if got := displayRef(""); got != "WORKING" {
		t.Errorf("displayRef(\"\") = %q", got)
	}
	if got := displayRef("v1.2.0"); got != "v1.2.0" {
		t.Errorf("displayRef(v1.2.0) = %q", got)
	}
}

func TestPromptAtRefWorkingCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.prompt")
	if err := os.WriteFile(path, []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, ref := range []string{"", "WORKING"} {
		data, err := promptAtRef(path, ref)
		if err != nil {
			t.Fatalf("promptAtRef(%q): %v", ref, err)
		}
		if string(data) != "body" {
			t.Errorf("data = %q", data)
		}
	}
}

func TestPromptAtRefMissingWorkingCopy(t *testing.T) {
	if _, err := promptAtRef(filepath.Join(t.TempDir(), "nope.prompt"), ""); err == nil {
		t.Error("expected error for missing working-copy file")
	}
}

func TestPrintDiffSemanticNoKey(t *testing.T) {
	t.Setenv("VOYAGE_API_KEY", "")
	oldP, err := prompt.Parse([]byte("---\nmodels: [gpt-4o]\n---\nhello"))
	if err != nil {
		t.Fatal(err)
	}
	newP, err := prompt.Parse([]byte("---\nmodels: [gpt-4o]\n---\nhello world"))
	if err != nil {
		t.Fatal(err)
	}
	err = printDiff("x.prompt", oldP, newP, true)
	if err == nil || !strings.Contains(err.Error(), "VOYAGE_API_KEY") {
		t.Errorf("err = %v, want VOYAGE_API_KEY error", err)
	}
}
