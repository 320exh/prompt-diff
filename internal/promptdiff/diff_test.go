package promptdiff

import (
	"testing"

	"github.com/320exh/prompt-diff/internal/prompt"
)

func compareTemplates(t *testing.T, oldSrc, newSrc string) Diff {
	t.Helper()
	oldP, err := prompt.Parse([]byte(oldSrc))
	if err != nil {
		t.Fatalf("parse old: %v", err)
	}
	newP, err := prompt.Parse([]byte(newSrc))
	if err != nil {
		t.Fatalf("parse new: %v", err)
	}
	return Compare(oldP, newP)
}

const baseSrc = `---
name: X
version: 1.0.0
models:
  - gpt-4o
  - claude-3-5-sonnet
variables:
  - user_query
---
You are an agent.
Handle this: {{ user_query }}
`

func TestCompareSections(t *testing.T) {
	newSrc := baseSrc + "\n\n[Output Constraints]\nReturn JSON only.\n"
	d := compareTemplates(t, baseSrc, newSrc)
	if len(d.AddedSections) != 1 || d.AddedSections[0] != "[Output Constraints]" {
		t.Errorf("added sections = %v", d.AddedSections)
	}
	if d.TokenDelta <= 0 {
		t.Errorf("token delta = %d, want > 0", d.TokenDelta)
	}
}

func TestCompareVariableRename(t *testing.T) {
	oldSrc := `---
name: X
variables: [user_context]
---
Context: {{ user_context }}
`
	newSrc := `---
name: X
variables: [augmented_user_profile]
---
Context: {{ augmented_user_profile }}
`
	d := compareTemplates(t, oldSrc, newSrc)
	if len(d.ModifiedVars) != 1 {
		t.Fatalf("modified vars = %v, want 1", d.ModifiedVars)
	}
	if d.ModifiedVars[0].From != "user_context" || d.ModifiedVars[0].To != "augmented_user_profile" {
		t.Errorf("rename = %v", d.ModifiedVars)
	}
}

func TestCompareCostsIncludeBothModels(t *testing.T) {
	d := compareTemplates(t, baseSrc, baseSrc+"\n\n[Extra]\nMore text here to add tokens.\n")
	if len(d.Costs) < 2 {
		t.Fatalf("costs = %v", d.Costs)
	}
	// Delta should be positive for each model.
	for _, c := range d.Costs {
		if c.Delta <= 0 {
			t.Errorf("cost %s delta = %+.2f, want > 0", c.Model, c.Delta)
		}
	}
	if d.TokenDelta == 0 {
		t.Error("token delta should be non-zero after adding text")
	}
}

func TestCompareIdentical(t *testing.T) {
	d := compareTemplates(t, baseSrc, baseSrc)
	if d.TokenDelta != 0 {
		t.Errorf("identical prompts should have 0 token delta, got %d", d.TokenDelta)
	}
	if len(d.AddedSections)+len(d.RemovedSections)+len(d.ModifiedVars) != 0 {
		t.Errorf("identical prompts should have no structural diffs: %+v", d)
	}
}