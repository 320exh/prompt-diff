package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleReport() *Report {
	return &Report{
		Suite:     "suite<1>",
		Prompt:    "My <Prompt>",
		Models:    []string{"gpt-4o", "llama3.1:8b"},
		Timestamp: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Total:     3,
		Failed:    1,
		Cells: []Cell{
			{Model: "gpt-4o", CaseID: "c1", Input: "<in>", Output: "ok", Passed: true},
			{Model: "gpt-4o", CaseID: "c2", Input: "in2", Output: "bad", Failed: true},
			{Model: "llama3.1:8b", CaseID: "c3", Input: "in3", Error: "boom <script>", Skipped: true},
		},
	}
}

func TestCounts(t *testing.T) {
	r := sampleReport()
	total, passed, failed, skipped := r.Counts()
	if total != 3 || passed != 1 || failed != 1 || skipped != 1 {
		t.Errorf("counts = %d/%d/%d/%d", total, passed, failed, skipped)
	}
}

func TestRenderEscapesAndSummarizes(t *testing.T) {
	html := render(sampleReport())
	if strings.Contains(html, "<Prompt>") {
		t.Error("expected prompt name to be HTML-escaped")
	}
	if !strings.Contains(html, "&lt;Prompt&gt;") {
		t.Error("expected escaped prompt name present")
	}
	if strings.Contains(html, "<script>") {
		t.Error("expected error text to be escaped, found raw <script>")
	}
	if !strings.Contains(html, "Passed 1 · Failed 1 · Skipped 1 · Total 3") {
		t.Error("expected summary line in rendered HTML")
	}
}

func TestWriteReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.html")
	if err := WriteReport(path, sampleReport()); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(data), "<!doctype html>") {
		t.Error("expected written file to contain doctype")
	}
}

func TestWriteReportBadPath(t *testing.T) {
	err := WriteReport(filepath.Join(t.TempDir(), "missing-dir", "report.html"), sampleReport())
	if err == nil {
		t.Error("expected error writing to nonexistent directory")
	}
}
