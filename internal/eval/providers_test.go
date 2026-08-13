package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("PD_TEST_KEY", "")
	if got := envOr("PD_TEST_KEY", "def"); got != "def" {
		t.Errorf("envOr empty = %q, want def", got)
	}
	t.Setenv("PD_TEST_KEY", "val")
	if got := envOr("PD_TEST_KEY", "def"); got != "val" {
		t.Errorf("envOr set = %q, want val", got)
	}
}

func TestResolveEndpoint(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "")
	if got := resolveEndpoint("openai"); got != "https://api.openai.com/v1" {
		t.Errorf("openai default = %q", got)
	}
	t.Setenv("OLLAMA_BASE_URL", "http://custom:1234")
	if got := resolveEndpoint("ollama"); got != "http://custom:1234" {
		t.Errorf("ollama endpoint = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("this is a long string", 7); got != "this is…" {
		t.Errorf("truncate long = %q", got)
	}
}

func TestCompletionRouting(t *testing.T) {
	cases := map[string]bool{
		"claude-3-5-sonnet": true,
		"gemini-1.5-pro":    true,
		"gpt-4o":            true,
		"o1-mini":           true,
		"llama3.1:8b":       false,
	}
	for model, wantErrImmediate := range cases {
		fn, resolved, err := completion("http://x", "", model, "sys", "user")
		if err != nil {
			t.Fatalf("completion(%q): %v", model, err)
		}
		if resolved != model {
			t.Errorf("resolved = %q, want %q", resolved, model)
		}
		if fn == nil {
			t.Fatalf("completion(%q): nil provider", model)
		}
		_, cerr := fn(context.Background())
		if wantErrImmediate && cerr == nil {
			t.Errorf("model %q: expected stubbed error, got nil", model)
		}
	}
}

func TestAnthropicCompletionNoKey(t *testing.T) {
	fn := anthropicCompletion("http://x", "", "claude-3-5-sonnet", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "API key") {
		t.Errorf("err = %v, want API key error", err)
	}
}

func TestAnthropicCompletionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-real" {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Errorf("anthropic-version header missing")
		}
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "claude-3-5-sonnet" {
			t.Errorf("model = %s", req.Model)
		}
		_ = json.NewEncoder(w).Encode(anthropicResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "hello from claude"}},
		})
	}))
	defer srv.Close()

	fn := anthropicCompletion(srv.URL, "sk-real", "claude-3-5-sonnet", "sys", "user")
	out, err := fn(context.Background())
	if err != nil {
		t.Fatalf("fn: %v", err)
	}
	if out != "hello from claude" {
		t.Errorf("out = %q", out)
	}
}

func TestAnthropicCompletionAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(anthropicResponse{Error: &struct {
			Message string `json:"message"`
		}{Message: "invalid request"}})
	}))
	defer srv.Close()

	fn := anthropicCompletion(srv.URL, "sk-real", "claude-3-5-sonnet", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("err = %v", err)
	}
}

func TestGeminiCompletionNoKey(t *testing.T) {
	fn := geminiCompletion("http://x", "", "gemini-1.5-pro", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "API key") {
		t.Errorf("err = %v, want API key error", err)
	}
}

func TestGeminiCompletionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/gemini-1.5-pro:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "sk-real" {
			t.Errorf("x-goog-api-key = %q", got)
		}
		_ = json.NewEncoder(w).Encode(geminiResponse{
			Candidates: []struct {
				Content geminiContent `json:"content"`
			}{{Content: geminiContent{Parts: []geminiPart{{Text: "hello from gemini"}}}}},
		})
	}))
	defer srv.Close()

	fn := geminiCompletion(srv.URL, "sk-real", "gemini-1.5-pro", "sys", "user")
	out, err := fn(context.Background())
	if err != nil {
		t.Fatalf("fn: %v", err)
	}
	if out != "hello from gemini" {
		t.Errorf("out = %q", out)
	}
}

func TestOllamaCompletionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "llama3.1:8b" {
			t.Errorf("model = %s", req.Model)
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message msg `json:"message"`
			}{{Message: msg{Role: "assistant", Content: "hello there"}}},
		})
	}))
	defer srv.Close()

	fn := ollamaCompletion(srv.URL, "llama3.1:8b", "sys", "user")
	out, err := fn(context.Background())
	if err != nil {
		t.Fatalf("fn: %v", err)
	}
	if out != "hello there" {
		t.Errorf("out = %q", out)
	}
}

func TestOllamaCompletionHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	fn := ollamaCompletion(srv.URL, "llama3.1:8b", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "provider error 500") {
		t.Errorf("err = %v", err)
	}
}

func TestOllamaCompletionAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{Error: &struct {
			Message string `json:"message"`
		}{Message: "model not found"}})
	}))
	defer srv.Close()

	fn := ollamaCompletion(srv.URL, "missing-model", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "model not found") {
		t.Errorf("err = %v", err)
	}
}

func TestOllamaCompletionNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{})
	}))
	defer srv.Close()

	fn := ollamaCompletion(srv.URL, "llama3.1:8b", "sys", "user")
	_, err := fn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Errorf("err = %v", err)
	}
}

func TestOpenAICompletionSendsAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("auth = %q", got)
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message msg `json:"message"`
			}{{Message: msg{Role: "assistant", Content: "ok"}}},
		})
	}))
	defer srv.Close()

	fn := openAICompletion(srv.URL, "sk-test", "gpt-4o", "sys", "user")
	out, err := fn(context.Background())
	if err != nil {
		t.Fatalf("fn: %v", err)
	}
	if out != "ok" {
		t.Errorf("out = %q", out)
	}
}
