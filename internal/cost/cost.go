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

// PricesAsOf is the date the built-in price table was last updated. Bump it
// whenever prices map changes so `prompt-diff models --check` can warn when
// it goes stale.
const PricesAsOf = "2026-08-13"

var prices = map[string]float64{
	// OpenAI
	"gpt-5":           1.25,
	"gpt-5-mini":      0.25,
	"gpt-5-nano":      0.05,
	"gpt-4o":          2.50,
	"gpt-4o-mini":     0.15,
	"gpt-4.1":         2.00,
	"gpt-4.1-mini":    0.40,
	"gpt-4.1-nano":    0.10,
	"o1":              15.00,
	"o1-mini":         1.10,
	"o3":              2.00,
	"o3-mini":         1.10,
	"o4-mini":         1.10,
	"gpt-4":           30.00,
	"gpt-3.5-turbo":   0.50,
	"text-embedding-3-small": 0.02,
	// Anthropic
	"claude-opus-4":       15.00,
	"claude-sonnet-4-5":   3.00,
	"claude-sonnet-4":     3.00,
	"claude-3-7-sonnet":   3.00,
	"claude-3-5-sonnet":   3.00,
	"claude-haiku-4-5":    1.00,
	"claude-3-5-haiku":    0.80,
	"claude-3-opus":       15.00,
	"claude-opus":         15.00,
	"claude-sonnet":       3.00,
	"claude-haiku":        0.80,
	// Google
	"gemini-1.5-pro":   1.25,
	"gemini-2.0-flash": 0.10,
	"gemini-2.5-pro":   1.25,
	"gemini-2.5-flash": 0.30,
	// Ollama local (no per-token cost)
	"ollama": 0.00,
}

// overrides holds user-supplied prices (e.g. negotiated rates, non-USD
// conversions) that take precedence over the built-in price list. Set via
// SetOverride, typically from a loaded config file.
var overrides = map[string]float64{}

// SetOverride pins model's price, overriding the built-in list.
func SetOverride(model string, per1M float64) {
	overrides[strings.ToLower(model)] = per1M
}

// IsKnown reports whether model has an exact or override entry in the price
// table (as opposed to falling back to a prefix or family-default guess).
func IsKnown(model string) bool {
	lower := strings.ToLower(model)
	if _, ok := overrides[lower]; ok {
		return true
	}
	_, ok := prices[lower]
	return ok
}

// Lookup returns the USD per-1M-token price for a model.
// Unlisted models fall back to a neutral default so projections never crash.
func Lookup(model string) float64 {
	lower := strings.ToLower(model)
	if p, ok := overrides[lower]; ok {
		return p
	}
	if p, ok := prices[lower]; ok {
		return p
	}
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

// cacheMultiplier holds a provider family's published prompt-caching
// discount: Write applies to the first invocation (cache miss, plus the
// provider's cache-write premium where one exists), Read applies to every
// subsequent invocation (cache hit).
//
// Sources (list prices, checked 2026-08-13):
//   - Anthropic: cache writes cost 1.25x the base input price (5-minute TTL),
//     cache reads cost 0.1x. https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
//   - OpenAI: cached input tokens are billed at 0.5x the base input price,
//     with no separate write premium (caching is automatic).
//     https://platform.openai.com/docs/guides/prompt-caching
//
// Gemini and any other family are deliberately left unmodeled: Gemini's
// caching has a separate hourly storage fee that this per-token model can't
// represent without risking a misleading number, so IsCacheSupported returns
// false for it rather than guessing.
type cacheMultiplier struct {
	Write float64
	Read  float64
}

var cacheMultipliers = map[string]cacheMultiplier{
	"claude": {Write: 1.25, Read: 0.1},
	"gpt":    {Write: 1.0, Read: 0.5},
	"o1":     {Write: 1.0, Read: 0.5},
	"o3":     {Write: 1.0, Read: 0.5},
	"o4":     {Write: 1.0, Read: 0.5},
}

// familyCacheMultiplier returns the cache pricing multipliers for model's
// provider family, and whether caching is modeled for it at all.
func familyCacheMultiplier(model string) (cacheMultiplier, bool) {
	lower := strings.ToLower(model)
	for prefix, m := range cacheMultipliers {
		if strings.HasPrefix(lower, prefix) {
			return m, true
		}
	}
	return cacheMultiplier{}, false
}

// IsCacheSupported reports whether prompt-caching cost is modeled for model's
// provider family (see cacheMultipliers doc comment for why some are excluded).
func IsCacheSupported(model string) bool {
	_, ok := familyCacheMultiplier(model)
	return ok
}

// EstimateBulkCached computes the cost to run count prompt tokens across
// invocations calls, assuming the whole prompt is cached: the first call
// pays the cache-write price, every subsequent call pays the cheaper
// cache-read price. Returns (0, false) when model's family has no modeled
// caching discount — callers should treat that as "not applicable", not
// "free".
func EstimateBulkCached(model string, count, invocations int64) (float64, bool) {
	mult, ok := familyCacheMultiplier(model)
	if !ok || invocations <= 0 {
		return 0, ok
	}
	per1M := Lookup(model)
	base := float64(count) / 1_000_000 * per1M
	total := base*mult.Write + base*mult.Read*float64(invocations-1)
	return total, true
}