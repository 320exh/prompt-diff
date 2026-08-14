package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/320exh/prompt-diff/internal/prompt"
)

// DefaultConcurrency caps in-flight provider calls when the caller doesn't
// specify one (0 or negative) via RunWithConcurrency.
const DefaultConcurrency = 5

// MaxConcurrency limits in-flight provider calls to prevent excessive memory
// allocation from untrusted inputs.
const MaxConcurrency = 100

// Run executes a suite against each model concurrently and scores results,
// using DefaultConcurrency in-flight calls at a time.
func Run(ctx context.Context, p *prompt.Template, suite *Suite, models []string) Report {
	return RunWithConcurrency(ctx, p, suite, models, DefaultConcurrency)
}

// RunWithConcurrency is Run with an explicit cap on in-flight provider calls
// across all models/cases. Unbounded fan-out (one goroutine per model x case)
// hammers provider rate limits on large suites even with per-call retry, so
// callers should keep this at or below what their API keys' tiers allow.
func RunWithConcurrency(ctx context.Context, p *prompt.Template, suite *Suite, models []string, concurrency int) Report {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	if concurrency > MaxConcurrency {
		concurrency = MaxConcurrency
	}
	sem := make(chan struct{}, concurrency)

	var mu sync.Mutex
	results := []Cell{}
	failed := 0
	total := 0

	for _, m := range models {
		m := m
		var wg sync.WaitGroup
		for ci := range suite.Cases {
			wg.Add(1)
			go func(ci int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				start := time.Now()
				out := callProvider(ctx, m, p.Body, suite.Cases[ci].Input)
				cell := Cell{
					Model:    m,
					CaseID:   suite.Cases[ci].ID,
					Input:    suite.Cases[ci].Input,
					Output:   out.Response,
					Error:    out.Error,
					Duration: time.Since(start),
				}
				switch {
				case out.Error != "":
					cell.Skipped = true
				case score(ctx, suite.Cases[ci].Expect, out.Response):
					cell.Passed = true
				default:
					cell.Failed = true
				}
				mu.Lock()
				defer mu.Unlock()
				results = append(results, cell)
				total++
				if cell.Failed {
					failed++
				}
			}(ci)
		}
		wg.Wait()
	}

	return Report{
		Suite:     suite.Name,
		Prompt:    p.Name,
		Models:    models,
		Timestamp: time.Now(),
		Cells:     results,
		Failed:    failed,
		Total:     total,
	}
}

// callProvider resolves the provider for a model and performs one completion.
func callProvider(ctx context.Context, model, sys, user string) complete {
	endpoint := resolveEndpoint(providerOf(model))
	apiKey := keyFor(model)
	fn, _, err := completion(endpoint, apiKey, model, sys, user)
	if err != nil {
		return complete{Error: err.Error()}
	}
	out, err := fn(ctx)
	if err != nil {
		return complete{Error: err.Error()}
	}
	return complete{Response: out}
}

// Complete performs a single one-off completion against the given model,
// dispatched to its provider by the same name-prefix rules Run uses. It's
// the building block for callers that need one LLM call outside a full eval
// suite run (e.g. prompt compression).
func Complete(ctx context.Context, model, sys, user string) (string, error) {
	out := callProvider(ctx, model, sys, user)
	if out.Error != "" {
		return "", fmt.Errorf("%s", out.Error)
	}
	return out.Response, nil
}

// providerOf maps a model to its provider name.
func providerOf(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.HasPrefix(lower, "gpt"), strings.HasPrefix(lower, "o1"), strings.HasPrefix(lower, "o3"):
		return "openai"
	case strings.HasPrefix(lower, "claude"):
		return "anthropic"
	case strings.HasPrefix(lower, "gemini"):
		return "gemini"
	default:
		return "ollama"
	}
}

func keyFor(model string) string {
	switch providerOf(model) {
	case "openai":
		return envOr("OPENAI_API_KEY", "sk-none")
	case "anthropic":
		return envOr("ANTHROPIC_API_KEY", "sk-none")
	case "gemini":
		return envOr("GEMINI_API_KEY", "sk-none")
	default:
		return ""
	}
}

var (
	jsonBlockRe = regexp.MustCompile("```json\\s*\\n?([\\s\\S]*?)```")
	jsonObjRe   = regexp.MustCompile(`\{[^{}]*\}`)
	kvRe        = regexp.MustCompile(`"([A-Za-z0-9_-]+)"\s*:\s*"([^"]*)"`)
	kvReNum     = regexp.MustCompile(`"([A-Za-z0-9_-]+)"\s*:\s*([0-9.]+)`)
)

// score checks an expected-output map against the model's response text.
func score(ctx context.Context, expect map[string]interface{}, out string) bool {
	trim := strings.TrimSpace(out)
	for k, v := range expect {
		switch t := v.(type) {
		case string:
			if !strings.Contains(strings.ToLower(trim), strings.ToLower(t)) {
				return false
			}
		case []interface{}:
			// explicit list of acceptable values
			if !anyContains(t, trim) {
				return false
			}
		case map[string]interface{}:
			// operator map like { "$in": [...] }, { "$gte": N }, { "$regex": ... },
			// { "$schema": {...} }, or { "$llm_judge": {...} }
			if !evalOp(ctx, k, t, trim) {
				return false
			}
		default:
			if !evalOp(ctx, k, map[string]interface{}{"==": v}, trim) {
				return false
			}
		}
	}
	return true
}

func anyContains(items []interface{}, out string) bool {
	for _, item := range items {
		if s, ok := item.(string); ok && strings.Contains(strings.ToLower(out), strings.ToLower(s)) {
			return true
		}
	}
	return false
}

func evalOp(ctx context.Context, key string, op map[string]interface{}, out string) bool {
	for opName, raw := range op {
		switch opName {
		case "$in":
			if arr, ok := raw.([]interface{}); ok {
				return anyContains(arr, out)
			}
			return false
		case "$gte", "$lte":
			want, ok := toFloat(raw)
			if !ok {
				return false
			}
			got := extractNumericField(key, out)
			if !got.present {
				return false
			}
			if opName == "$gte" {
				return got.value >= want
			}
			return got.value <= want
		case "==":
			s, _ := raw.(string)
			return strings.Contains(strings.ToLower(out), strings.ToLower(s))
		case "$regex":
			pattern, _ := raw.(string)
			re, err := regexp.Compile(pattern)
			if err != nil {
				return false
			}
			return re.MatchString(out)
		case "$schema":
			spec, ok := raw.(map[string]interface{})
			if !ok {
				return false
			}
			return matchesSchema(spec, out)
		case "$llm_judge":
			spec, _ := raw.(map[string]interface{})
			return llmJudge(ctx, spec, out)
		}
	}
	return true
}

// matchesSchema checks a JSON-ish output against a lightweight schema:
// { "required": ["field", ...], "properties": { "field": { "type": "string" } } }.
func matchesSchema(spec map[string]interface{}, out string) bool {
	obj, ok := extractJSONObject(out)
	if !ok {
		return false
	}
	if req, ok := spec["required"].([]interface{}); ok {
		for _, r := range req {
			name, _ := r.(string)
			if _, present := obj[name]; !present {
				return false
			}
		}
	}
	if props, ok := spec["properties"].(map[string]interface{}); ok {
		for name, propSpec := range props {
			ps, ok := propSpec.(map[string]interface{})
			if !ok {
				continue
			}
			wantType, _ := ps["type"].(string)
			if wantType == "" {
				continue
			}
			val, present := obj[name]
			if !present {
				continue
			}
			if !jsonTypeMatches(wantType, val) {
				return false
			}
		}
	}
	return true
}

func jsonTypeMatches(want string, val interface{}) bool {
	switch want {
	case "string":
		_, ok := val.(string)
		return ok
	case "number":
		_, ok := val.(float64)
		return ok
	case "integer":
		f, ok := val.(float64)
		return ok && f == float64(int64(f))
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "array":
		_, ok := val.([]interface{})
		return ok
	case "object":
		_, ok := val.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// extractJSONObject pulls the first JSON object out of a model response,
// preferring a fenced ```json``` block over a bare {...} substring.
func extractJSONObject(out string) (map[string]interface{}, bool) {
	candidate := out
	if m := jsonBlockRe.FindStringSubmatch(out); len(m) > 1 {
		candidate = m[1]
	} else if start := strings.Index(out, "{"); start >= 0 {
		if end := strings.LastIndex(out, "}"); end > start {
			candidate = out[start : end+1]
		}
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(candidate)), &obj); err != nil {
		return nil, false
	}
	return obj, true
}

// llmJudge asks a grader model whether the output satisfies given criteria.
// spec: { "criteria": "...", "model": "gpt-4o-mini" (optional) }.
func llmJudge(ctx context.Context, spec map[string]interface{}, out string) bool {
	criteria, _ := spec["criteria"].(string)
	if criteria == "" {
		return false
	}
	model, _ := spec["model"].(string)
	if model == "" {
		model = envOr("PROMPTDIFF_JUDGE_MODEL", "gpt-4o-mini")
	}
	sys := "You are a strict grader. Respond with exactly one word: PASS or FAIL."
	user := fmt.Sprintf("Criteria: %s\n\nOutput to grade:\n%s", criteria, out)
	verdict := callProvider(ctx, model, sys, user)
	if verdict.Error != "" {
		return false
	}
	return strings.Contains(strings.ToUpper(verdict.Response), "PASS")
}

type numericField struct {
	present bool
	value   float64
}

// extractNumericField pulls a numeric value for key from JSON-ish output.
func extractNumericField(key, out string) numericField {
	if m := jsonBlockRe.FindStringSubmatch(out); len(m) > 1 {
		out = m[1]
	} else if m := jsonObjRe.FindStringSubmatch(out); len(m) > 0 && strings.Contains(out, "\""+key+"\"") {
		out = m[0]
	}
	for _, re := range []*regexp.Regexp{kvReNum, kvRe} {
		if m := re.FindStringSubmatch(out); len(m) == 3 && strings.EqualFold(m[1], key) {
			if v, err := strconv.ParseFloat(m[2], 64); err == nil {
				return numericField{true, v}
			}
		}
	}
	return numericField{}
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	}
	return 0, false
}