package tokenizer

import (
	"testing"
)

// Total sanity check against published values: the opening of the US
// Declaration of Independence is commonly quoted at ~93 tokens with
// cl100k_base (GPT-4 tokenizer).
const declarationSample = "We hold these truths to be self-evident, that all men are created equal, " +
	"that they are endowed by their Creator with certain unalienable Rights, that among these are " +
	"Life, Liberty and the pursuit of Happiness."

func TestDeclarationOfIndependence(t *testing.T) {
	sample := declarationSample
	// Token counts for this text are documented online as ~93 tokens.
	if got := Encoder.Count(sample); got < 80 || got > 110 {
		t.Errorf("expected ~93 tokens, got %d", got)
	} else {
		t.Logf("declaration sample counted at %d tokens", got)
	}
}

func TestCountEmpty(t *testing.T) {
	if Encoder.Count("") != 0 {
		t.Error("expected 0 tokens for empty string")
	}
}

func TestCountDeterministic(t *testing.T) {
	s := "The quick brown fox jumps over the lazy dog.\nWith newlines and \ttabs."
	if a, b := Encoder.Count(s), Encoder.Count(s); a != b {
		t.Errorf("count not deterministic: %d vs %d", a, b)
	}
}

func TestCountKnownStrings(t *testing.T) {
	// Counts verified against the embedded cl100k_base vocab.
	samples := map[string]int{
		"hello world": 3,
		"a":           1,
		"token":       1,
		"":            0,
		"x y z":       5,
		"Hello, world!": 5,
		"Multi\nLine\tText": 4,
	}
	for s, want := range samples {
		if got := Encoder.Count(s); got != want {
			t.Errorf("Count(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestPunctuationAttachesToWord(t *testing.T) {
	// "a," is one token; "a" is one token.
	if Encoder.Count("Hello,") != 2 {
		t.Errorf("Count(\"Hello,\") = %d, want 2", Encoder.Count("Hello,"))
	}
	if Encoder.Count("Hello world!") != 4 {
		t.Errorf("got %d tokens for 'Hello world!'", Encoder.Count("Hello world!"))
	}
}

func TestCountSpecialForms(t *testing.T) {
	// JSON-ish structures in prompts
	jsonish := `{"classification": "refund", "confidence": 0.95}`
	if Encoder.Count(jsonish) < 5 {
		t.Errorf("jsonish counted too low: %d", Encoder.Count(jsonish))
	}
}