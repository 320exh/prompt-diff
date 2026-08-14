// Package uiserver embeds the compiled Svelte dashboard and serves it with a
// JSON API mirroring every CLI capability: diff, lint, eval, compress,
// convert, models, runs (list/export), init scaffolding, and hook install.
package uiserver

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/320exh/prompt-diff/internal/compress"
	"github.com/320exh/prompt-diff/internal/cost"
	embedapi "github.com/320exh/prompt-diff/internal/embed"
	"github.com/320exh/prompt-diff/internal/eval"
	"github.com/320exh/prompt-diff/internal/gitutil"
	"github.com/320exh/prompt-diff/internal/interop"
	"github.com/320exh/prompt-diff/internal/lint"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/promptdiff"
	"github.com/320exh/prompt-diff/internal/report"
	"github.com/320exh/prompt-diff/internal/scaffold"
	"github.com/320exh/prompt-diff/internal/store"
	"github.com/320exh/prompt-diff/internal/tokenizer"
)

//go:embed dist
var distFS embed.FS

// workspaceServer serves the embedded dashboard plus its JSON API.
type workspaceServer struct {
	port    int
	handler http.Handler
}

// New returns a workspaceServer ready to serve on the given port.
func New(port int) *workspaceServer {
	return &workspaceServer{port: port, handler: newMux()}
}

// Serve starts the embedded dashboard server on the given port.
func (s *workspaceServer) Serve() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.handler)
}

// Listen binds a listener for the dashboard, falling back to an OS-assigned
// free port if the requested one is already in use. It returns the listener
// and the port actually bound, without serving yet.
func (s *workspaceServer) Listen() (net.Listener, int, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		ln, err = net.Listen("tcp", ":0")
		if err != nil {
			return nil, 0, err
		}
	}
	return ln, ln.Addr().(*net.TCPAddr).Port, nil
}

// ServeOn serves the dashboard on an already-bound listener.
func (s *workspaceServer) ServeOn(ln net.Listener) error {
	return http.Serve(ln, s.handler)
}

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(mustSub(distFS, "dist"))))
	mux.HandleFunc("/api/tokenize", tokenizeHandler)
	mux.HandleFunc("/api/compare", compareHandler)
	mux.HandleFunc("/api/runs", runsHandler)
	mux.HandleFunc("/api/runs/export", runsExportHandler)
	mux.HandleFunc("/api/diff", diffHandler)
	mux.HandleFunc("/api/lint", lintHandler)
	mux.HandleFunc("/api/models", modelsHandler)
	mux.HandleFunc("/api/eval", evalHandler)
	mux.HandleFunc("/api/compress", compressHandler)
	mux.HandleFunc("/api/convert", convertHandler)
	mux.HandleFunc("/api/init", initHandler)
	mux.HandleFunc("/api/hook", hookHandler)
	return mux
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func tokenizeHandler(w http.ResponseWriter, r *http.Request) {
	body := r.URL.Query().Get("text")
	if body == "" && r.Method == http.MethodPost {
		var req struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		body = req.Text
	}
	n := tokenizer.Encoder.Count(body)
	writeJSON(w, map[string]int{"tokens": n})
}

func compareHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Left  string `json:"left"`
		Right string `json:"right"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	type row struct {
		Side   string `json:"side"`
		Tokens int    `json:"tokens"`
	}
	writeJSON(w, []row{
		{Side: "left", Tokens: tokenizer.Encoder.Count(req.Left)},
		{Side: "right", Tokens: tokenizer.Encoder.Count(req.Right)},
	})
}

func runsHandler(w http.ResponseWriter, r *http.Request) {
	s, err := store.Open()
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	defer s.Close()

	runs, err := s.ListRuns()
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if runs == nil {
		runs = []store.Run{}
	}
	writeJSON(w, runs)
}

// runsExportHandler mirrors `prompt-diff runs export`: ?ids=1,2&format=md|pdf.
func runsExportHandler(w http.ResponseWriter, r *http.Request) {
	var ids []int64
	for _, part := range strings.Split(r.URL.Query().Get("ids"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			httpError(w, http.StatusBadRequest, fmt.Errorf("invalid run id %q", part))
			return
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		httpError(w, http.StatusBadRequest, fmt.Errorf("no run ids given (?ids=1,2)"))
		return
	}

	s, err := store.Open()
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	runs, err := s.ListRuns()
	_ = s.Close()
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	selected, err := report.SelectByID(runs, ids)
	if err != nil {
		httpError(w, http.StatusNotFound, err)
		return
	}

	switch format := r.URL.Query().Get("format"); format {
	case "", "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="prompt-diff-report.md"`)
		_, _ = w.Write([]byte(report.Markdown(selected)))
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="prompt-diff-report.pdf"`)
		if err := report.PDF(selected, w); err != nil {
			httpError(w, http.StatusInternalServerError, err)
		}
	default:
		httpError(w, http.StatusBadRequest, fmt.Errorf("unknown format %q (want md or pdf)", format))
	}
}

// diffHandler mirrors `prompt-diff diff a.prompt b.prompt` on two pasted
// sources, with optional semantic similarity (VOYAGE_API_KEY).
func diffHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Old      string `json:"old"`
		New      string `json:"new"`
		Semantic bool   `json:"semantic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	oldP, err := prompt.Parse([]byte(req.Old))
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("old prompt: %w", err))
		return
	}
	newP, err := prompt.Parse([]byte(req.New))
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("new prompt: %w", err))
		return
	}
	d := promptdiff.Compare(oldP, newP)
	if req.Semantic {
		client := embedapi.NewVoyageClient()
		if err := promptdiff.ApplySemantic(r.Context(), &d, client, oldP, newP); err != nil {
			httpError(w, http.StatusBadGateway, fmt.Errorf("semantic diff: %w", err))
			return
		}
	}
	writeJSON(w, struct {
		Models []string `json:"target_models"`
		promptdiff.Diff
	}{Models: newP.Models, Diff: d})
}

func lintHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	findings := lint.Source([]byte(req.Source))
	if findings == nil {
		findings = []string{}
	}
	writeJSON(w, map[string]interface{}{"findings": findings})
}

// modelsStaleAfterDays mirrors the CLI's `models --check` threshold.
const modelsStaleAfterDays = 90

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	asOf, err := time.Parse("2006-01-02", cost.PricesAsOf)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	age := int(time.Since(asOf).Hours() / 24)
	writeJSON(w, map[string]interface{}{
		"prices_as_of":     cost.PricesAsOf,
		"age_days":         age,
		"stale":            age > modelsStaleAfterDays,
		"stale_after_days": modelsStaleAfterDays,
	})
}

type evalCellJSON struct {
	Model      string `json:"model"`
	CaseID     string `json:"case_id"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	Passed     bool   `json:"passed"`
	Failed     bool   `json:"failed"`
	Skipped    bool   `json:"skipped"`
	DurationMS int64  `json:"duration_ms"`
}

func cellsJSON(cells []eval.Cell) []evalCellJSON {
	out := make([]evalCellJSON, 0, len(cells))
	for _, c := range cells {
		out = append(out, evalCellJSON{
			Model: c.Model, CaseID: c.CaseID, Input: c.Input, Output: c.Output,
			Error: c.Error, Passed: c.Passed, Failed: c.Failed, Skipped: c.Skipped,
			DurationMS: c.Duration.Milliseconds(),
		})
	}
	return out
}

// evalHandler mirrors `prompt-diff eval` on pasted prompt + suite JSON. The
// run is recorded in the same SQLite store the CLI uses.
func evalHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt      string   `json:"prompt"`
		Suite       string   `json:"suite"`
		Models      []string `json:"models"`
		Concurrency int      `json:"concurrency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	p, err := prompt.Parse([]byte(req.Prompt))
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("prompt: %w", err))
		return
	}
	suite, err := eval.ParseSuite([]byte(req.Suite), "workspace suite")
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("suite: %w", err))
		return
	}
	models := req.Models
	if len(models) == 0 {
		models = p.Models
	}
	if len(models) == 0 {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("no models given and none declared in the prompt frontmatter"))
		return
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = eval.DefaultConcurrency
	}
	if concurrency > eval.MaxConcurrency {
		concurrency = eval.MaxConcurrency
	}

	rep := eval.RunWithConcurrency(r.Context(), p, suite, models, concurrency)
	total, passed, failed, skipped := rep.Counts()

	var runID int64
	if s, err := store.Open(); err == nil {
		runID, _ = s.RecordRun("workspace", suite.Name, strings.Join(models, ","), total, passed, failed, skipped)
		_ = s.Close()
	}

	writeJSON(w, map[string]interface{}{
		"suite":   suite.Name,
		"models":  models,
		"total":   total,
		"passed":  passed,
		"failed":  failed,
		"skipped": skipped,
		"run_id":  runID,
		"cells":   cellsJSON(rep.Cells),
	})
}

// compressHandler mirrors `prompt-diff compress` on a pasted prompt. Instead
// of writing to disk, it returns the compressed body and the re-rendered
// .prompt file for the user to copy or download.
func compressHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt  string   `json:"prompt"`
		Model   string   `json:"model"`
		Suite   string   `json:"suite"`
		Models  []string `json:"models"`
		MaxLoss float64  `json:"max_loss"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if req.Model == "" {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("model is required"))
		return
	}
	p, err := prompt.Parse([]byte(req.Prompt))
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("prompt: %w", err))
		return
	}

	compressedBody, err := compress.Rewrite(r.Context(), req.Model, p)
	if err != nil {
		httpError(w, http.StatusBadGateway, err)
		return
	}

	before := tokenizer.Encoder.Count(p.Body)
	after := tokenizer.Encoder.Count(compressedBody)

	resp := map[string]interface{}{
		"body":          compressedBody,
		"tokens_before": before,
		"tokens_after":  after,
	}

	if strings.TrimSpace(req.Suite) != "" {
		suite, err := eval.ParseSuite([]byte(req.Suite), "workspace suite")
		if err != nil {
			httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("suite: %w", err))
			return
		}
		models := req.Models
		if len(models) == 0 {
			models = p.Models
		}
		if len(models) == 0 {
			httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("no models given and none declared in the prompt frontmatter"))
			return
		}
		compressed := &prompt.Template{Name: p.Name, Version: p.Version, Models: p.Models, Variables: p.Variables, Body: compressedBody}
		beforeReport := eval.Run(r.Context(), p, suite, models)
		afterReport := eval.Run(r.Context(), compressed, suite, models)
		btotal, bpassed, _, _ := beforeReport.Counts()
		atotal, apassed, _, _ := afterReport.Counts()
		brate, arate := rateOf(bpassed, btotal), rateOf(apassed, atotal)
		resp["pass_rate_before"] = brate
		resp["pass_rate_after"] = arate
		if arate < brate-req.MaxLoss {
			resp["rejected"] = fmt.Sprintf("rewrite regresses pass rate by %.1f%% (max allowed %.1f%%)", brate-arate, req.MaxLoss)
			writeJSON(w, resp)
			return
		}
	}

	if rendered, err := prompt.Render(p, compressedBody); err == nil {
		resp["rendered"] = rendered
	}
	writeJSON(w, resp)
}

func rateOf(passed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total) * 100
}

// convertHandler mirrors `prompt-diff convert` on pasted content.
func convertHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		From    string `json:"from"`
		To      string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	out, err := interop.Convert([]byte(req.Content), req.From, req.To)
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, map[string]string{"output": string(out)})
}

// initHandler mirrors `prompt-diff init`, scaffolding into the directory the
// server was launched from.
func initHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	var req struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	written, skipped, err := scaffold.Write("", req.Force)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if written == nil {
		written = []string{}
	}
	if skipped == nil {
		skipped = []string{}
	}
	writeJSON(w, map[string][]string{"written": written, "skipped": skipped})
}

// hookHandler mirrors `prompt-diff hook install` in the server's CWD repo.
func hookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	var req struct {
		MaxDelta int `json:"max_delta"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	hookPath, err := gitutil.InstallHook(req.MaxDelta)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]string{"path": hookPath})
}
