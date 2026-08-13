package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSuiteNamed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "suite.json")
	body := `{"name": "My Suite", "models": ["gpt-4o"], "cases": [{"id": "c1", "input": "hi", "expect": {"x": "y"}}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSuite(path)
	if err != nil {
		t.Fatalf("LoadSuite: %v", err)
	}
	if s.Name != "My Suite" {
		t.Errorf("name = %q", s.Name)
	}
	if len(s.Cases) != 1 || s.Cases[0].ID != "c1" {
		t.Errorf("cases = %+v", s.Cases)
	}
}

func TestLoadSuiteDefaultsNameFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unnamed.json")
	if err := os.WriteFile(path, []byte(`{"cases": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSuite(path)
	if err != nil {
		t.Fatalf("LoadSuite: %v", err)
	}
	if s.Name == "" || s.Name == "unnamed.json" {
		t.Errorf("name = %q, want .json suffix trimmed", s.Name)
	}
}

func TestLoadSuiteMissingFile(t *testing.T) {
	if _, err := LoadSuite(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSuiteBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSuite(path); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestSuiteValidate(t *testing.T) {
	empty := &Suite{}
	if err := empty.Validate(); err == nil {
		t.Error("expected error for suite with no cases")
	}
	withCase := &Suite{Cases: []Case{{ID: "c1"}}}
	if err := withCase.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
