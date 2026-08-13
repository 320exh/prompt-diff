package eval

import (
	"fmt"
	"html"
	"os"
	"strings"
	"time"
)

// Cell is one model x case result.
type Cell struct {
	Model    string
	CaseID   string
	Input    string
	Output   string
	Error    string
	Passed   bool
	Failed   bool
	Skipped  bool
	Duration time.Duration
}

// Report is a full eval run.
type Report struct {
	Suite     string
	Prompt    string
	Models    []string
	Timestamp time.Time
	Cells     []Cell
	Failed    int
	Total     int
}

// Counts summarises the report.
func (r *Report) Counts() (total, passed, failed, skipped int) {
	for _, c := range r.Cells {
		switch {
		case c.Passed:
			passed++
		case c.Failed:
			failed++
		case c.Skipped:
			skipped++
		}
	}
	return r.Total, passed, failed, skipped
}

// WriteReport writes a self-contained HTML report.
func WriteReport(path string, r *Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(render(r))
	return err
}

func render(r *Report) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>` + esc(r.Prompt) + ` — eval report</title>`)
	b.WriteString(`<style>
:root{color-scheme:light dark}
body{font-family:system-ui,sans-serif;max-width:960px;margin:2rem auto;padding:0 1rem}
h1{font-size:1.4rem;margin-bottom:.3rem}
.meta{color:#666;margin-bottom:1rem}
table{width:100%;border-collapse:collapse;font-size:.9rem}
th,td{border:1px solid #ccc;padding:.4rem .6rem;text-align:left;vertical-align:top}
th{background:#f5f5f5}pre{white-space:pre-wrap;margin:0}
.pass{color:#137333}.fail{color:#c5221f}.skip{color:#b06000}
@media (prefers-color-scheme:dark){:root{color-scheme:dark}
body{background:#111;color:#eee}th{background:#222}td{border-color:#444}
.pass{color:#7ed18a}.fail{color:#ff8a80}.skip{color:#ffd54f}}
</style></head><body>`)
	total, passed, failed, skipped := r.Counts()
	b.WriteString(fmt.Sprintf(`<h1>%s</h1>`, esc(r.Prompt)))
	b.WriteString(fmt.Sprintf(`<div class="meta">Suite %s · %s · %d models · %s</div>`,
		esc(r.Suite), r.Timestamp.Format("2006-01-02 15:04:05"), len(r.Models), esc(strings.Join(r.Models, ", "))))
	b.WriteString(fmt.Sprintf(`<div class="meta">Passed %d · Failed %d · Skipped %d · Total %d</div>`,
		passed, failed, skipped, total))
	b.WriteString(`<table><tr><th>Model</th><th>Case</th><th>Input</th><th>Output</th><th>Result</th></tr>`)
	for _, c := range r.Cells {
		cls, label := "skip", "skipped"
		switch {
		case c.Passed:
			cls, label = "pass", "pass"
		case c.Failed:
			cls, label = "fail", "fail"
		}
		detail := `<pre>` + esc(c.Output) + `</pre>`
		if c.Error != "" {
			detail = `<span style="color:#c5221f">` + esc(c.Error) + `</span>`
		}
		b.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td><pre>%s</pre></td><td>%s</td><td class="%s">%s</td></tr>`,
			esc(c.Model), esc(c.CaseID), esc(c.Input), detail, cls, label))
	}
	b.WriteString(`</table></body></html>`)
	return b.String()
}