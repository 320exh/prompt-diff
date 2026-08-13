// Package uiserver embeds the compiled Svelte dashboard and serves it with a
// small JSON API for variable tuning and token counting.
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

	"github.com/320exh/prompt-diff/internal/prompt"
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
	return mux
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func tokenizeHandler(w http.ResponseWriter, r *http.Request) {
	body := r.URL.Query().Get("text")
	if body == "" && r.Method == http.MethodPost {
		var req struct{ Text string `json:"text"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		body = req.Text
	}
	n := tokenizer.Encoder.Count(body)
	_ = json.NewEncoder(w).Encode(map[string]int{"tokens": n})
}

func compareHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Left  string `json:"left"`
		Right string `json:"right"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	leftToks := tokenizer.Encoder.Count(req.Left)
	rightToks := tokenizer.Encoder.Count(req.Right)
	type row struct {
		Side   string `json:"side"`
		Tokens int    `json:"tokens"`
	}
	_ = json.NewEncoder(w).Encode([]row{
		{Side: "left", Tokens: leftToks},
		{Side: "right", Tokens: rightToks},
	})
}

type tokenResult struct {
	Tokens int `json:"tokens"`
}

var _ = prompt.Template{}
var _ = strconv.Itoa
var _ = strings.TrimSpace