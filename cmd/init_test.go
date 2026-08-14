package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunInitCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	initForce = false
	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	for _, name := range []string{"example.prompt", "example.eval.json", ".prompt-diff.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not created: %v", name, err)
		}
	}
}

func TestRunInitSkipsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	if err := os.WriteFile("example.prompt", []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}

	initForce = false
	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	data, err := os.ReadFile("example.prompt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom" {
		t.Errorf("example.prompt was overwritten: %q", data)
	}
}

func TestRunInitForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	if err := os.WriteFile("example.prompt", []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}

	initForce = true
	defer func() { initForce = false }()
	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	data, err := os.ReadFile("example.prompt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "custom" {
		t.Error("example.prompt was not overwritten with --force")
	}
}
