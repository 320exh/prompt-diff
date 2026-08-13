package store

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PROMPT_DIFF_DB", filepath.Join(dir, "runs.db"))
	s, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestDBPathHonorsEnv(t *testing.T) {
	t.Setenv("PROMPT_DIFF_DB", filepath.Join("some", "custom.db"))
	if got := DBPath(); got != filepath.Join("some", "custom.db") {
		t.Errorf("DBPath = %q", got)
	}
}

func TestOpenCreatesTable(t *testing.T) {
	s := testStore(t)
	runs, err := s.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns on fresh db: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected no runs, got %d", len(runs))
	}
}

func TestRecordAndListRuns(t *testing.T) {
	s := testStore(t)

	id1, err := s.RecordRun("prompts/a.prompt", "suite-a", "gpt-4o,llama3.1:8b", 4, 3, 1, 0)
	if err != nil {
		t.Fatalf("RecordRun 1: %v", err)
	}
	id2, err := s.RecordRun("prompts/b.prompt", "suite-b", "gpt-4o", 2, 2, 0, 0)
	if err != nil {
		t.Fatalf("RecordRun 2: %v", err)
	}
	if id1 == 0 || id2 == 0 || id1 == id2 {
		t.Fatalf("expected distinct nonzero ids, got %d, %d", id1, id2)
	}

	runs, err := s.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("len(runs) = %d, want 2", len(runs))
	}
	// newest first
	if runs[0].ID != id2 || runs[1].ID != id1 {
		t.Errorf("order = %d, %d; want %d, %d", runs[0].ID, runs[1].ID, id2, id1)
	}
	if runs[1].PromptPath != "prompts/a.prompt" || runs[1].Suite != "suite-a" {
		t.Errorf("run1 = %+v", runs[1])
	}
	if runs[1].Passed != 3 || runs[1].Failed != 1 || runs[1].Total != 4 {
		t.Errorf("run1 counts = %+v", runs[1])
	}
}

func TestCloseThenUseErrors(t *testing.T) {
	s := testStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := s.ListRuns(); err == nil {
		t.Error("expected error using store after Close")
	}
}
