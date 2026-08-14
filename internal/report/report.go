// Package report renders stored eval runs as markdown or PDF benchmark
// reports. Shared by `prompt-diff runs export` and the web workspace.
package report

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/320exh/prompt-diff/internal/store"
	"github.com/go-pdf/fpdf"
)

// PassRate is a run's pass percentage in [0, 100].
func PassRate(r store.Run) float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Passed) / float64(r.Total) * 100
}

// Markdown renders the selected runs as a markdown benchmark report. Exactly
// two runs also get a delta section.
func Markdown(selected []store.Run) string {
	var b strings.Builder
	b.WriteString("# prompt-diff benchmark report\n\n")
	b.WriteString(fmt.Sprintf("Generated %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("| ID | Created | Prompt | Suite | Models | Pass | Fail | Skip | Total | Pass Rate |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range selected {
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %d | %d | %d | %d | %.1f%% |\n",
			r.ID, r.CreatedAt, r.PromptPath, r.Suite, r.Models, r.Passed, r.Failed, r.Skipped, r.Total, PassRate(r)))
	}

	if len(selected) == 2 {
		r1, r2 := selected[0], selected[1]
		b.WriteString("\n## Delta (run " + strconv.FormatInt(r1.ID, 10) + " -> " + strconv.FormatInt(r2.ID, 10) + ")\n\n")
		b.WriteString(fmt.Sprintf("- Pass rate: %+.1f%%\n", PassRate(r2)-PassRate(r1)))
		b.WriteString(fmt.Sprintf("- Passed: %+d\n", r2.Passed-r1.Passed))
		b.WriteString(fmt.Sprintf("- Failed: %+d\n", r2.Failed-r1.Failed))
		b.WriteString(fmt.Sprintf("- Total: %+d\n", r2.Total-r1.Total))
	}
	return b.String()
}

// PDF renders the selected runs as a PDF benchmark report to w.
func PDF(selected []store.Run, w io.Writer) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "prompt-diff benchmark report", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 8, "Generated "+time.Now().Format("2006-01-02 15:04:05"), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	headers := []string{"ID", "Created", "Prompt", "Suite", "Models", "Pass", "Fail", "Skip", "Total", "Rate"}
	widths := []float64{10, 30, 30, 25, 30, 15, 15, 15, 15, 15}
	pdf.SetFont("Helvetica", "B", 9)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, h, "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 9)
	for _, r := range selected {
		row := []string{
			strconv.FormatInt(r.ID, 10), r.CreatedAt, r.PromptPath, r.Suite, r.Models,
			strconv.Itoa(r.Passed), strconv.Itoa(r.Failed), strconv.Itoa(r.Skipped), strconv.Itoa(r.Total),
			fmt.Sprintf("%.1f%%", PassRate(r)),
		}
		for i, v := range row {
			pdf.CellFormat(widths[i], 8, v, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	if len(selected) == 2 {
		r1, r2 := selected[0], selected[1]
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, fmt.Sprintf("Delta (run %d -> %d)", r1.ID, r2.ID), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, fmt.Sprintf("Pass rate: %+.1f%%", PassRate(r2)-PassRate(r1)), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Passed: %+d", r2.Passed-r1.Passed), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Failed: %+d", r2.Failed-r1.Failed), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Total: %+d", r2.Total-r1.Total), "", 1, "L", false, 0, "")
	}

	return pdf.Output(w)
}

// SelectByID returns the stored runs with the given ids, in order.
func SelectByID(runs []store.Run, ids []int64) ([]store.Run, error) {
	byID := map[int64]store.Run{}
	for _, r := range runs {
		byID[r.ID] = r
	}
	selected := make([]store.Run, 0, len(ids))
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("run %d not found", id)
		}
		selected = append(selected, r)
	}
	return selected, nil
}
