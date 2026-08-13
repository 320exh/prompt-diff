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

// anthropicCompletion targets the Anthropic Messages API directly.
func anthropicCompletion(endpoint, apiKey, model, sys, user string) provider {
	url := strings.TrimSuffix(endpoint, "/") + "/messages"
	req := anthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		System:    sys,
		Messages:  []anthropicMsg{{Role: "user", Content: user}},
	}
	return func(ctx context.Context) (string, error) {
		if apiKey == "" || apiKey == "sk-none" {
			return "", fmt.Errorf("Anthropic provider requires a valid API key in ANTHROPIC_API_KEY")
		}
		return postAnthropic(ctx, url, apiKey, req)
	}
}

// geminiCompletion targets the Gemini generateContent REST endpoint.
func geminiCompletion(endpoint, apiKey, model, sys, user string) provider {
	url := strings.TrimSuffix(endpoint, "/") + "/models/" + model + ":generateContent"
	req := geminiRequest{
		Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: user}}}},
	}
	if sys != "" {
		req.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: sys}}}
	}
	return func(ctx context.Context) (string, error) {
		if apiKey == "" || apiKey == "sk-none" {
			return "", fmt.Errorf("Gemini provider requires a valid API key in GEMINI_API_KEY")
		}
		return postGemini(ctx, url, apiKey, req)
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

// Anthropic Messages API shapes: https://platform.claude.com/docs/en/api/messages
type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func postAnthropic(ctx context.Context, url, apiKey string, req anthropicRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var ar anthropicResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Anthropic provider error %d: %s", resp.StatusCode, truncate(string(data), 160))
		}
		return "", err
	}
	if ar.Error != nil {
		return "", fmt.Errorf("Anthropic provider error: %s", ar.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic provider error %d: %s", resp.StatusCode, truncate(string(data), 160))
	}
	var text strings.Builder
	for _, block := range ar.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("Anthropic provider returned no text content")
	}
	return text.String(), nil
}

// Gemini generateContent shapes: https://ai.google.dev/api/generate-content
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func postGemini(ctx context.Context, url, apiKey string, req geminiRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", apiKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var gr geminiResponse
	if err := json.Unmarshal(data, &gr); err != nil {
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Gemini provider error %d: %s", resp.StatusCode, truncate(string(data), 160))
		}
		return "", err
	}
	if gr.Error != nil {
		return "", fmt.Errorf("Gemini provider error: %s", gr.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini provider error %d: %s", resp.StatusCode, truncate(string(data), 160))
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini provider returned no candidates")
	}
	var text strings.Builder
	for _, part := range gr.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
	}
	return text.String(), nil
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
	case "anthropic":
		return envOr("ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1")
	case "gemini":
		return envOr("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta")
	default:
		return envOr(strings.ToUpper(providerName)+"_BASE_URL", "")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}