package cmd

import (
	"os"
	"path/filepath"
	"testing"
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
