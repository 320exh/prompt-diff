package compress

import "testing"

func TestStripCodeFence(t *testing.T) {
	cases := map[string]string{
		"plain text":           "plain text",
		"```\nfenced\n```":     "fenced",
		"```text\nfenced\n```": "fenced",
	}
	for in, want := range cases {
		if got := StripCodeFence(in); got != want {
			t.Errorf("StripCodeFence(%q) = %q, want %q", in, got, want)
		}
	}
}
