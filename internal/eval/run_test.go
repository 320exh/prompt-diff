package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/320exh/prompt-diff/internal/prompt"
)

func TestProviderOf(t *testing.T) {
	cases := map[string]string{
		"gpt-4o":            "openai",
		"o1-mini":           "openai",
		"o3-mini":           "openai",
		"CLAUDE-3-5-Sonnet": "anthropic",
		"gemini-1.5-pro":    "gemini",
		"llama3.1:8b":       "ollama",
		"qwen2.5":           "ollama",
	}
	for model, want := range cases {
		if got := providerOf(model); got != want {
			t.Errorf("providerOf(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestKeyFor(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	if got := keyFor("gpt-4o"); got != "sk-none" {
		t.Errorf("keyFor default = %q", got)
	}
	t.Setenv("OPENAI_API_KEY", "sk-real")
	if got := keyFor("gpt-4o"); got != "sk-real" {
		t.Errorf("keyFor set = %q", got)
	}
	if got := keyFor("llama3.1:8b"); got != "" {
		t.Errorf("keyFor ollama = %q, want empty", got)
	}
}

func TestScoreStringContains(t *testing.T) {
	expect := map[string]interface{}{"category": "Billing"}
	if !score(context.Background(), expect, "The category is billing related.") {
		t.Error("expected match on case-insensitive substring")
	}
	if score(context.Background(), expect, "The category is shipping related.") {
		t.Error("expected no match")
	}
}

func TestScoreList(t *testing.T) {
	expect := map[string]interface{}{"category": []interface{}{"Billing", "Refund"}}
	if !score(context.Background(), expect, "This is a Refund case.") {
		t.Error("expected list match")
	}
	if score(context.Background(), expect, "This is a Shipping case.") {
		t.Error("expected list mismatch")
	}
}

func TestScoreOperatorGteLte(t *testing.T) {
	expect := map[string]interface{}{"confidence": map[string]interface{}{"$gte": 0.8}}
	if !score(context.Background(), expect, `{"confidence": 0.95}`) {
		t.Error("expected $gte match")
	}
	if score(context.Background(), expect, `{"confidence": 0.5}`) {
		t.Error("expected $gte mismatch")
	}

	expectLte := map[string]interface{}{"latency": map[string]interface{}{"$lte": 100}}
	if !score(context.Background(), expectLte, "```json\n{\"latency\": 42}\n```") {
		t.Error("expected $lte match inside json block")
	}
}

func TestScoreOperatorIn(t *testing.T) {
	expect := map[string]interface{}{"label": map[string]interface{}{"$in": []interface{}{"Yes", "No"}}}
	if !score(context.Background(), expect, "Answer: Yes") {
		t.Error("expected $in match")
	}
	if score(context.Background(), expect, "Answer: Maybe") {
		t.Error("expected $in mismatch")
	}
}

func TestScoreDefaultEquality(t *testing.T) {
	expect := map[string]interface{}{"x": "42"}
	if !score(context.Background(), expect, "x is 42") {
		t.Error("expected default equality match via ==")
	}
}

func TestScoreOperatorRegex(t *testing.T) {
	expect := map[string]interface{}{"id": map[string]interface{}{"$regex": `^ORD-\d{4}$`}}
	if !score(context.Background(), expect, "ORD-1234") {
		t.Error("expected $regex match")
	}
	if score(context.Background(), expect, "ORD-12") {
		t.Error("expected $regex mismatch")
	}
}

func TestScoreOperatorSchema(t *testing.T) {
	expect := map[string]interface{}{
		"_": map[string]interface{}{
			"$schema": map[string]interface{}{
				"required":   []interface{}{"category", "confidence"},
				"properties": map[string]interface{}{"confidence": map[string]interface{}{"type": "number"}},
			},
		},
	}
	if !score(context.Background(), expect, `{"category": "Billing", "confidence": 0.9}`) {
		t.Error("expected $schema match")
	}
	if score(context.Background(), expect, `{"category": "Billing"}`) {
		t.Error("expected $schema mismatch on missing required field")
	}
	if score(context.Background(), expect, `{"category": "Billing", "confidence": "high"}`) {
		t.Error("expected $schema mismatch on wrong type")
	}
}

func TestScoreOperatorLLMJudge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		content := "FAIL"
		if strings.Contains(req.Messages[len(req.Messages)-1].Content, "good") {
			content = "PASS"
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message msg `json:"message"`
			}{{Message: msg{Role: "assistant", Content: content}}},
		})
	}))
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)
	t.Setenv("PROMPTDIFF_JUDGE_MODEL", "llama3.1:8b")

	expect := map[string]interface{}{"_": map[string]interface{}{"$llm_judge": map[string]interface{}{"criteria": "is polite"}}}
	if !score(context.Background(), expect, "this is a good response") {
		t.Error("expected $llm_judge pass")
	}
	if score(context.Background(), expect, "this is a bad response") {
		t.Error("expected $llm_judge fail")
	}
}

func TestToFloat(t *testing.T) {
	if v, ok := toFloat(3.5); !ok || v != 3.5 {
		t.Errorf("float64: %v %v", v, ok)
	}
	if v, ok := toFloat(3); !ok || v != 3 {
		t.Errorf("int: %v %v", v, ok)
	}
	if v, ok := toFloat("2.5"); !ok || v != 2.5 {
		t.Errorf("string: %v %v", v, ok)
	}
	if v, ok := toFloat(json.Number("7")); !ok || v != 7 {
		t.Errorf("json.Number: %v %v", v, ok)
	}
	if _, ok := toFloat("not-a-number"); ok {
		t.Error("expected failure for non-numeric string")
	}
	if _, ok := toFloat(true); ok {
		t.Error("expected failure for unsupported type")
	}
}

func TestExtractNumericField(t *testing.T) {
	got := extractNumericField("score", `Here is the result: {"score": 7.5, "label": "ok"}`)
	if !got.present || got.value != 7.5 {
		t.Errorf("got = %+v", got)
	}
	missing := extractNumericField("missing", `{"score": 7.5}`)
	if missing.present {
		t.Errorf("expected not present, got %+v", missing)
	}
}

func TestRunEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		content := "category: Billing"
		if req.Model == "llama-fail" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("nope"))
			return
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message msg `json:"message"`
			}{{Message: msg{Role: "assistant", Content: content}}},
		})
	}))
	defer srv.Close()
	t.Setenv("OLLAMA_BASE_URL", srv.URL)

	p := &prompt.Template{Name: "test-prompt", Body: "sys"}
	suite := &Suite{
		Name: "suite",
		Cases: []Case{
			{ID: "c1", Input: "u1", Expect: map[string]interface{}{"category": "Billing"}},
			{ID: "c2", Input: "u2", Expect: map[string]interface{}{"category": "Shipping"}},
		},
	}

	report := Run(context.Background(), p, suite, []string{"llama3.1:8b", "llama-fail"})
	total, passed, failed, skipped := report.Counts()
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if passed != 1 {
		t.Errorf("passed = %d, want 1", passed)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	if skipped != 2 {
		t.Errorf("skipped = %d, want 2", skipped)
	}
}
