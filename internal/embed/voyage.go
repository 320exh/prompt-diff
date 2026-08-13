// Package embed provides text-embedding clients for semantic prompt diffing.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client embeds text into a vector for semantic comparison.
type Client interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// VoyageClient calls the Voyage AI embeddings API.
// https://docs.voyageai.com/reference/embeddings-api
type VoyageClient struct {
	Endpoint string
	APIKey   string
	Model    string
}

// NewVoyageClient builds a client from VOYAGE_API_KEY / VOYAGE_BASE_URL /
// VOYAGE_MODEL env vars, defaulting to the voyage-3-lite model.
func NewVoyageClient() *VoyageClient {
	return &VoyageClient{
		Endpoint: envOr("VOYAGE_BASE_URL", "https://api.voyageai.com/v1"),
		APIKey:   os.Getenv("VOYAGE_API_KEY"),
		Model:    envOr("VOYAGE_MODEL", "voyage-3-lite"),
	}
}

type voyageRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"`
}

type voyageResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error string `json:"detail"`
}

// Embed returns one vector per input text, in the same order as texts.
func (c *VoyageClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("semantic diff requires a Voyage AI API key in VOYAGE_API_KEY (get one at https://dashboard.voyageai.com)")
	}
	body, err := json.Marshal(voyageRequest{Input: texts, Model: c.Model, InputType: "document"})
	if err != nil {
		return nil, err
	}
	url := strings.TrimSuffix(c.Endpoint, "/") + "/embeddings"

	var resp *http.Response
	backoff := 500 * time.Millisecond
	for attempt := 1; attempt <= 4; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
		resp, err = http.DefaultClient.Do(httpReq)
		if err == nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			break
		}
		if attempt == 4 {
			break
		}
		if err == nil {
			resp.Body.Close()
		}
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
		backoff *= 2
	}
	if resp == nil {
		return nil, fmt.Errorf("Voyage embeddings request failed with no response")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Voyage embeddings error %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var vr voyageResponse
	if err := json.Unmarshal(data, &vr); err != nil {
		return nil, err
	}
	if len(vr.Data) != len(texts) {
		return nil, fmt.Errorf("Voyage embeddings: expected %d vectors, got %d", len(texts), len(vr.Data))
	}
	out := make([][]float64, len(texts))
	for _, d := range vr.Data {
		out[d.Index] = d.Embedding
	}
	return out, nil
}

// CosineSimilarity returns the cosine similarity of a and b, in [-1, 1].
// Returns 0 if either vector is empty or zero-length.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
