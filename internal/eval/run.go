package eval

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/320exh/prompt-diff/internal/prompt"
)

// Run executes a suite against each model concurrently and scores results.
func Run(ctx context.Context, p *prompt.Template, suite *Suite, models []string) Report {
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
				case score(suite.Cases[ci].Expect, out.Response):
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
func score(expect map[string]interface{}, out string) bool {
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
			// operator map like { "$in": [...] } or { "$gte": N }
			if !evalOp(k, t, trim) {
				return false
			}
		default:
			if !evalOp(k, map[string]interface{}{"==": v}, trim) {
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

func evalOp(key string, op map[string]interface{}, out string) bool {
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
		}
	}
	return true
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