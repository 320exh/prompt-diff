package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	cases := []struct {
		name string
		a, b []float64
		want float64
	}{
		{"identical", []float64{1, 0}, []float64{1, 0}, 1},
		{"orthogonal", []float64{1, 0}, []float64{0, 1}, 0},
		{"opposite", []float64{1, 0}, []float64{-1, 0}, -1},
		{"empty", nil, []float64{1}, 0},
		{"mismatched length", []float64{1, 2}, []float64{1}, 0},
	}
	for _, c := range cases {
		if got := CosineSimilarity(c.a, c.b); got != c.want {
			t.Errorf("%s: CosineSimilarity = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestVoyageEmbedNoKey(t *testing.T) {
	c := &VoyageClient{Endpoint: "http://x", Model: "voyage-3-lite"}
	_, err := c.Embed(context.Background(), []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "VOYAGE_API_KEY") {
		t.Errorf("err = %v, want VOYAGE_API_KEY error", err)
	}
}

func TestVoyageEmbedSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{1, 0}, "index": 0},
				{"embedding": []float64{0, 1}, "index": 1},
			},
		})
	}))
	defer srv.Close()

	c := &VoyageClient{Endpoint: srv.URL, APIKey: "test-key", Model: "voyage-3-lite"}
	vecs, err := c.Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("len(vecs) = %d, want 2", len(vecs))
	}
	if got := CosineSimilarity(vecs[0], vecs[1]); got != 0 {
		t.Errorf("similarity = %v, want 0", got)
	}
}

func TestVoyageEmbedErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"invalid api key"}`))
	}))
	defer srv.Close()

	c := &VoyageClient{Endpoint: srv.URL, APIKey: "bad-key", Model: "voyage-3-lite"}
	_, err := c.Embed(context.Background(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want 401 error", err)
	}
}
