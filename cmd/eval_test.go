package cmd

import "testing"

func TestSplitModels(t *testing.T) {
	got := splitModels(" gpt-4o , llama3.1:8b ,,claude-3-5-sonnet")
	want := []string{"gpt-4o", "llama3.1:8b", "claude-3-5-sonnet"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitModelsEmpty(t *testing.T) {
	if got := splitModels(""); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
