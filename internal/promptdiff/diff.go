// Package promptdiff computes structural and token diffs between two prompt
// versions, with README-matching output.
package promptdiff

import (
	"context"
	"math"
	"strings"

	"github.com/320exh/prompt-diff/internal/cost"
	"github.com/320exh/prompt-diff/internal/embed"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/tokenizer"
)

// Diff is the result of comparing two versions of a prompt.
type Diff struct {
	TokenDelta      int         `json:"token_delta"`
	TokenPercent    float64     `json:"token_percent"`
	Costs           []CostLine  `json:"costs"`
	AddedSections   []string    `json:"added_sections,omitempty"`
	RemovedSections []string    `json:"removed_sections,omitempty"`
	ModifiedVars    []VarChange `json:"modified_vars,omitempty"`
	AddedVars       []string    `json:"added_vars,omitempty"`
	RemovedVars     []string    `json:"removed_vars,omitempty"`
	// SemanticSimilarity is the cosine similarity, in [-1, 1], between the
	// old and new prompt body embeddings. Nil unless semantic diffing ran.
	SemanticSimilarity *float64 `json:"semantic_similarity,omitempty"`
}

// CostLine is one model row of the cost projection.
type CostLine struct {
	Model string  `json:"model"`
	Old   float64 `json:"old"`
	New   float64 `json:"new"`
	Delta float64 `json:"delta"`
	// Approx is true when Model has no known BPE vocabulary, so the token
	// counts behind Old/New/Delta are approximated with cl100k_base rather
	// than the model's real tokenizer.
	Approx bool `json:"approx"`
}

// VarChange records a variable that was renamed.
type VarChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Compare diffs oldP -> newP.
func Compare(oldP, newP *prompt.Template) Diff {
	d := Diff{}
	oldToks := tokenizer.Encoder.Count(oldP.Body)
	newToks := tokenizer.Encoder.Count(newP.Body)
	d.TokenDelta = newToks - oldToks
	if oldToks > 0 {
		d.TokenPercent = float64(d.TokenDelta) / float64(oldToks) * 100
	} else {
		d.TokenPercent = 0
	}

	// Unified model set for the cost projection, preserving frontmatter order.
	models := orderedUnion(newP.Models, oldP.Models)
	const invocations = 100_000
	for _, m := range models {
		mOldToks, exact := tokenizer.CountForModel(m, oldP.Body)
		mNewToks, _ := tokenizer.CountForModel(m, newP.Body)
		oldCost := cost.EstimateBulk(m, int64(mOldToks), invocations)
		newCost := cost.EstimateBulk(m, int64(mNewToks), invocations)
		d.Costs = append(d.Costs, CostLine{
			Model:  m,
			Old:    round2(oldCost),
			New:    round2(newCost),
			Delta:  round2(newCost - oldCost),
			Approx: !exact,
		})
	}

	// Structural diff on sections.
	oldSections := sectionKeys(oldP)
	newSections := sectionKeys(newP)
	for _, s := range diffAdded(oldSections, newSections) {
		d.AddedSections = append(d.AddedSections, s)
	}
	for _, s := range diffRemoved(oldSections, newSections) {
		d.RemovedSections = append(d.RemovedSections, s)
	}

	// Structural diff on variables.
	oldVars := []string{}
	for _, v := range oldP.Variables {
		oldVars = append(oldVars, v)
	}
	newVars := []string{}
	for _, v := range newP.Variables {
		newVars = append(newVars, v)
	}
	oldSet := map[string]bool{}
	for _, v := range oldVars {
		oldSet[v] = true
	}
	newSet := map[string]bool{}
	for _, v := range newVars {
		newSet[v] = true
	}
	// Detect renames: a removed var whose Value appears verbatim in new body.
	// We approximate a rename when a removed variable name no longer appears
	// in the body and a new variable was introduced with a similar role.
	for _, v := range newVars {
		if !oldSet[v] {
			d.AddedVars = append(d.AddedVars, v)
		}
	}
	for _, v := range oldVars {
		if !newSet[v] {
			// try to pair with an added var
			matched := false
			for _, nv := range d.AddedVars {
				if sameRole(v, nv) {
					d.ModifiedVars = append(d.ModifiedVars, VarChange{From: v, To: nv})
					matched = true
					break
				}
			}
			if !matched {
				d.RemovedVars = append(d.RemovedVars, v)
			}
		}
	}
	return d
}

// ApplySemantic embeds oldP.Body and newP.Body with client and sets
// d.SemanticSimilarity to their cosine similarity.
func ApplySemantic(ctx context.Context, d *Diff, client embed.Client, oldP, newP *prompt.Template) error {
	vecs, err := client.Embed(ctx, []string{oldP.Body, newP.Body})
	if err != nil {
		return err
	}
	sim := embed.CosineSimilarity(vecs[0], vecs[1])
	d.SemanticSimilarity = &sim
	return nil
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func orderedUnion(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sectionKeys(p *prompt.Template) map[string]bool {
	out := map[string]bool{}
	for _, s := range p.Sections() {
		out[firstLine(s)] = true
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func diffAdded(oldSet, newSet map[string]bool) []string {
	var out []string
	for k := range newSet {
		if !oldSet[k] {
			out = append(out, k)
		}
	}
	return out
}

func diffRemoved(oldSet, newSet map[string]bool) []string {
	var out []string
	for k := range oldSet {
		if !newSet[k] {
			out = append(out, k)
		}
	}
	return out
}

// sameRole heuristically pairs a removed variable with an added one.
//
// Two names are "same role" when they share a significant contiguous
// substring, or share a common initial token. "user_context" and
// "augmented_user_profile" both contain "user", so they pair; unrelated
// names like "auth" and "temperature" do not.
func sameRole(a, b string) bool {
	clean := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", " ")
		s = strings.ReplaceAll(s, "-", " ")
		return s
	}
	ca := strings.Fields(clean(a))
	cb := strings.Fields(clean(b))
	if len(ca) == 0 || len(cb) == 0 {
		return false
	}
	// any shared word
	for _, wa := range ca {
		for _, wb := range cb {
			if wa == wb && len(wa) >= 3 {
				return true
			}
		}
	}
	// or a shared contiguous run of >= 5 chars
	return longestCommonSubstring(clean(a), clean(b)) >= 5
}

func longestCommonSubstring(a, b string) int {
	if len(a) > len(b) {
		a, b = b, a
	}
	best := 0
	for i := 0; i < len(a); i++ {
		for j := i + 1; j <= len(a); j++ {
			if j-i <= best {
				continue
			}
			if strings.Contains(b, a[i:j]) {
				best = j - i
			}
		}
	}
	return best
}