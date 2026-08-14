package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/store"
)

func TestRunRunsExport(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("PROMPT_DIFF_DB", filepath.Join(dir, "runs.db"))
	defer os.Unsetenv("PROMPT_DIFF_DB")

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	id1, err := s.RecordRun("example.prompt", "example.eval.json", "claude-3-haiku", 10, 8, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.RecordRun("example.prompt", "example.eval.json", "claude-3-haiku", 10, 9, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	outPath := filepath.Join(dir, "report.md")
	runsExportOut = outPath
	defer func() { runsExportOut = "" }()

	args := []string{
		strconv.FormatInt(id1, 10),
		strconv.FormatInt(id2, 10),
	}
	if err := runRunsExport(runsExportCmd, args); err != nil {
		t.Fatalf("runRunsExport: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	report := string(data)
	if !strings.Contains(report, "# prompt-diff benchmark report") {
		t.Errorf("missing report header: %q", report)
	}
	if !strings.Contains(report, "## Delta (run") {
		t.Errorf("missing delta section for two runs: %q", report)
	}
	if !strings.Contains(report, "claude-3-haiku") {
		t.Errorf("missing model column: %q", report)
	}
}

func TestRunRunsExportPDF(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("PROMPT_DIFF_DB", filepath.Join(dir, "runs.db"))
	defer os.Unsetenv("PROMPT_DIFF_DB")

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	id1, err := s.RecordRun("example.prompt", "example.eval.json", "claude-3-haiku", 10, 8, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	outPath := filepath.Join(dir, "report.pdf")
	runsExportOut = outPath
	runsExportFormat = "pdf"
	defer func() { runsExportOut = ""; runsExportFormat = "md" }()

	if err := runRunsExport(runsExportCmd, []string{strconv.FormatInt(id1, 10)}); err != nil {
		t.Fatalf("runRunsExport pdf: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Errorf("output does not look like a PDF: %q", data[:min(20, len(data))])
	}
}

func TestRunRunsExportPDFRequiresOut(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("PROMPT_DIFF_DB", filepath.Join(dir, "runs.db"))
	defer os.Unsetenv("PROMPT_DIFF_DB")

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	id1, err := s.RecordRun("example.prompt", "example.eval.json", "claude-3-haiku", 10, 8, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	runsExportOut = ""
	runsExportFormat = "pdf"
	defer func() { runsExportFormat = "md" }()

	if err := runRunsExport(runsExportCmd, []string{strconv.FormatInt(id1, 10)}); err == nil {
		t.Error("expected error when --format pdf used without --out")
	}
}

func TestRunRunsExportUnknownID(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("PROMPT_DIFF_DB", filepath.Join(dir, "runs.db"))
	defer os.Unsetenv("PROMPT_DIFF_DB")

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	runsExportOut = ""
	if err := runRunsExport(runsExportCmd, []string{"999"}); err == nil {
		t.Error("expected error for unknown run id")
	}
}
