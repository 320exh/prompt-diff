package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// complete is one model completion.
type complete struct {
	Response string
	Error    string
}

// provider is a function that performs one completion call.
type provider func(ctx context.Context) (string, error)

// completion resolves a model name to its provider adapter.
func completion(endpoint, apiKey, model, sys, user string) (provider, string, error) {
	trim := strings.TrimSpace(model)
	lower := strings.ToLower(trim)
	switch {
	case strings.HasPrefix(lower, "claude"):
		return anthropicCompletion(endpoint, apiKey, model, sys, user), trim, nil
	case strings.HasPrefix(lower, "gemini"):
		return geminiCompletion(endpoint, apiKey, model, sys, user), trim, nil
	case strings.HasPrefix(lower, "gpt"), strings.HasPrefix(lower, "o1"), strings.HasPrefix(lower, "o3"):
		return openAICompletion(endpoint, apiKey, model, sys, user), trim, nil
	default: // llama/qwen/mistral and everything else -> local Ollama
		return ollamaCompletion(endpoint, model, sys, user), trim, nil
	}
}

// openAICompletion targets the /chat/completions endpoint.
func openAICompletion(endpoint, apiKey, model, sys, user string) provider {
	url := strings.TrimSuffix(endpoint, "/") + "/chat/completions"
	req := chatRequest{
		Model:    model,
		Messages: []msg{{Role: "system", Content: sys}, {Role: "user", Content: user}},
	}
	return func(ctx context.Context) (string, error) {
		return postChat(ctx, url, apiKey, req)
	}
}

// anthropicCompletion is provider-adapted for v1: Anthropic's Messages API is
// reachable, but this adapter reports a clear error until a real client with
// the ANTHROPIC_API_KEY is wired in. Local runs fail loudly instead of
// silently guessing at a foreign payload shape.
func anthropicCompletion(endpoint, apiKey, model, sys, user string) provider {
	return func(ctx context.Context) (string, error) {
		if apiKey == "" || apiKey == "sk-none" {
			return "", fmt.Errorf("Anthropic provider requires a valid API key in ANTHROPIC_API_KEY")
		}
		return "", fmt.Errorf("Anthropic provider: direct Messages API integration not yet wired; set ANTHROPIC_API_KEY to enable")
	}
}

// geminiCompletion mirrors the Anthropic adapter for the Gemini API.
func geminiCompletion(endpoint, apiKey, model, sys, user string) provider {
	return func(ctx context.Context) (string, error) {
		if apiKey == "" || apiKey == "sk-none" {
			return "", fmt.Errorf("Gemini provider requires a valid API key in GEMINI_API_KEY")
		}
		return "", fmt.Errorf("Gemini provider: direct v1beta REST integration not yet wired; set GEMINI_API_KEY to enable")
	}
}

// ollamaCompletion targets the local Ollama /api/chat endpoint.
func ollamaCompletion(url, model, sys, user string) provider {
	req := chatRequest{
		Model:    model,
		Messages: []msg{{Role: "system", Content: sys}, {Role: "user", Content: user}},
	}
	return func(ctx context.Context) (string, error) {
		return postChat(ctx, url+"/api/chat", "", req)
	}
}

type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string `json:"model"`
	Messages []msg  `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message msg `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func postChat(ctx context.Context, url, apiKey string, req chatRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sleepCtx(ctx, 50*time.Millisecond)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("provider error %d: %s", resp.StatusCode, truncate(string(data), 160))
	}
	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return "", err
	}
	if cr.Error != nil {
		return "", fmt.Errorf("provider error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}
	return cr.Choices[0].Message.Content, nil
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// resolveEndpoint picks the provider base URL.
func resolveEndpoint(providerName string) string {
	switch providerName {
	case "openai":
		return envOr("OPENAI_BASE_URL", "https://api.openai.com/v1")
	default:
		return envOr(providerName+"_BASE_URL", "")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}