package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/prompt"
)

func TestStripCodeFence(t *testing.T) {
	cases := map[string]string{
		"plain text":           "plain text",
		"```\nfenced\n```":     "fenced",
		"```text\nfenced\n```": "fenced",
	}
	for in, want := range cases {
		if got := stripCodeFence(in); got != want {
			t.Errorf("stripCodeFence(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderPrompt(t *testing.T) {
	p := &prompt.Template{Name: "Test", Version: "1.0.0", Models: []string{"gpt-4o"}, Variables: []string{"x"}}
	out, err := renderPrompt(p, "Hello {{ x }}")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("missing frontmatter delimiter: %q", out)
	}
	if !strings.Contains(out, "Hello {{ x }}") {
		t.Errorf("missing body: %q", out)
	}
	if !strings.Contains(out, "name: Test") {
		t.Errorf("missing name: %q", out)
	}
}

func chatServer(t *testing.T, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type resp struct {
			Choices []struct {
				Message msg `json:"message"`
			} `json:"choices"`
		}
		_ = json.NewEncoder(w).Encode(resp{Choices: []struct {
			Message msg `json:"message"`
		}{{Message: msg{Role: "assistant", Content: response}}}})
	}))
}

func TestRunCompressDryRun(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(promptPath, []byte("---\nname: Test\nvariables:\n  - x\n---\nSay hi to {{ x }}, please, if you would be so kind.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := chatServer(t, "Greet {{ x }}.")
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)

	compressPrompt = promptPath
	compressModel = "test-model"
	compressTests = ""
	compressModels = ""
	compressOut = ""
	compressApply = false
	compressMaxLoss = 0

	if err := runCompress(compressCmd, nil); err != nil {
		t.Fatalf("runCompress: %v", err)
	}

	if _, err := os.Stat(promptPath); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(promptPath)
	if !strings.Contains(string(data), "Say hi to") {
		t.Error("dry run must not modify the source prompt")
	}
}

func TestRunCompressApplyWritesFile(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(promptPath, []byte("---\nname: Test\nvariables:\n  - x\n---\nSay hi to {{ x }}, please, if you would be so kind.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "out.prompt")

	srv := chatServer(t, "Greet {{ x }}.")
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)

	compressPrompt = promptPath
	compressModel = "test-model"
	compressTests = ""
	compressModels = ""
	compressOut = outPath
	compressApply = true
	compressMaxLoss = 0
	defer func() { compressApply = false; compressOut = "" }()

	if err := runCompress(compressCmd, nil); err != nil {
		t.Fatalf("runCompress: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Greet {{ x }}") {
		t.Errorf("compressed body missing: %q", data)
	}
}

func TestRunCompressRejectsDroppedVariable(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(promptPath, []byte("---\nname: Test\nvariables:\n  - x\n---\nSay hi to {{ x }}.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := chatServer(t, "Say hi to everyone.")
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)

	compressPrompt = promptPath
	compressModel = "test-model"
	compressTests = ""
	compressModels = ""
	compressOut = ""
	compressApply = false
	compressMaxLoss = 0

	if err := runCompress(compressCmd, nil); err == nil || !strings.Contains(err.Error(), "dropped required variable") {
		t.Errorf("err = %v, want dropped-variable rejection", err)
	}
}

func TestRunCompressRegressionGuard(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "test.prompt")
	if err := os.WriteFile(promptPath, []byte("---\nname: Test\nmodels:\n  - test-model\nvariables:\n  - x\n---\nSay hi to {{ x }}.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testsPath := filepath.Join(dir, "suite.json")
	if err := os.WriteFile(testsPath, []byte(`{"name":"s","cases":[{"id":"c1","input":"hi","expect":{"contains":"hi"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		reply := "hi there" // original prompt path: passes ("contains hi")
		if callCount == 1 {
			reply = "Hi {{ x }}." // the compression rewrite call itself; keeps the placeholder
		} else if callCount > 2 {
			reply = "nope" // compressed-prompt eval run: fails ("contains hi")
		}
		type msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type resp struct {
			Choices []struct {
				Message msg `json:"message"`
			} `json:"choices"`
		}
		_ = json.NewEncoder(w).Encode(resp{Choices: []struct {
			Message msg `json:"message"`
		}{{Message: msg{Role: "assistant", Content: reply}}}})
	}))
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)

	compressPrompt = promptPath
	compressModel = "test-model"
	compressTests = testsPath
	compressModels = ""
	compressOut = ""
	compressApply = false
	compressMaxLoss = 0
	defer func() { compressTests = "" }()

	if err := runCompress(compressCmd, nil); err == nil || !strings.Contains(err.Error(), "regresses pass rate") {
		t.Errorf("err = %v, want regression rejection", err)
	}
}
