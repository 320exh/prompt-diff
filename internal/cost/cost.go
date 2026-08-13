// Package cost estimates per-model pricing for prompt tokens.
//
// Prices are per 1M input tokens, in USD, approximate list prices as of this
// project's v1. The map is used for README-style "cost impact per 100k runs"
// projections; it is not a substitute for live billing data.
package cost

import (
	"sort"
	"strings"
)

// Price is the USD cost per 1M input tokens.
type Price struct {
	Per1M float64
}

var prices = map[string]float64{
	// OpenAI
	"gpt-4o":         2.50,
	"gpt-4o-mini":    0.15,
	"o1":             15.00,
	"o1-mini":        1.10,
	"o3":             2.00,
	"gpt-4.1":        2.00,
	"gpt-4":          30.00,
	"text-embedding-3-small": 0.02,
	// Anthropic
	"claude-3-5-sonnet": 3.00,
	"claude-sonnet":     3.00,
	"claude-3-5-haiku":  0.80,
	"claude-haiku":      0.80,
	// Google
	"gemini-1.5-pro": 1.25,
	"gemini-2.0-flash": 0.10,
	"gemini-2.5-flash": 0.30,
	// Ollama local (no per-token cost)
	"ollama": 0.00,
}

// Lookup returns the USD per-1M-token price for a model.
// Unlisted models fall back to a neutral default so projections never crash.
func Lookup(model string) float64 {
	if p, ok := prices[strings.ToLower(model)]; ok {
		return p
	}
	lower := strings.ToLower(model)
	// prefix matches for precision variants like gpt-4o-2024-08-06
	keys := make([]string, 0, len(prices))
	for k := range prices {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		if strings.HasPrefix(lower, k) {
			return prices[k]
		}
	}
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") {
		return 2.50
	}
	if strings.HasPrefix(lower, "claude") {
		return 3.00
	}
	if strings.HasPrefix(lower, "gemini") {
		return 1.25
	}
	if strings.HasPrefix(lower, "llama") || strings.HasPrefix(lower, "qwen") || strings.HasPrefix(lower, "mistral") {
		return 0.00
	}
	return 2.00 // unknown cloud model default
}

// Estimated is a per-model cost estimate for a given token count.
type Estimated struct {
	Model  string  `json:"model"`
	Tokens int     `json:"tokens"`
	Per1M  float64 `json:"price_per_1m"`
	Cost   float64 `json:"cost_usd"`
}

// Estimate computes the cost to run count tokens once, per model.
func Estimate(model string, count int) Estimated {
	per1M := Lookup(model)
	return Estimated{
		Model:  model,
		Tokens: count,
		Per1M:  per1M,
		Cost:   float64(count) / 1_000_000 * per1M,
	}
}

// EstimateBulk computes the cost to run each model's tokens n times.
func EstimateBulk(model string, count, invocations int64) float64 {
	return Estimate(model, int(count)).Cost * float64(invocations)
}