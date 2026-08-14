// Package compress rewrites a prompt body more tersely via an LLM, guarding
// against dropped {{ variable }} placeholders. Shared by `prompt-diff
// compress` and the web workspace.
package compress

import (
	"context"
	"fmt"
	"strings"

	"github.com/320exh/prompt-diff/internal/eval"
	"github.com/320exh/prompt-diff/internal/prompt"
)

// Instruction is the system-side rewrite instruction sent ahead of the body.
const Instruction = `You compress LLM system prompts. Rewrite the prompt below to use as few
tokens as possible while preserving every instruction, constraint, and
{{ variable }} placeholder exactly as written (do not rename, add, or
remove any {{ variable }} placeholder). Do not add commentary, headers, or
markdown code fences. Output only the rewritten prompt text.

PROMPT:
`

// Rewrite asks model to compress p's body and returns the compressed body.
// It rejects empty rewrites and rewrites that drop a declared variable.
func Rewrite(ctx context.Context, model string, p *prompt.Template) (string, error) {
	rewritten, err := eval.Complete(ctx, model, "", Instruction+p.Body)
	if err != nil {
		return "", fmt.Errorf("compression call to %q failed: %w", model, err)
	}
	body := strings.TrimSpace(StripCodeFence(rewritten))
	if body == "" {
		return "", fmt.Errorf("compression call to %q returned an empty rewrite", model)
	}
	for _, v := range p.Variables {
		if !strings.Contains(body, "{{ "+v+" }}") && !strings.Contains(body, "{{"+v+"}}") {
			return "", fmt.Errorf("rewrite dropped required variable {{ %s }}; rejecting (nothing written)", v)
		}
	}
	return body, nil
}

// StripCodeFence removes a single leading/trailing ``` fence if the model
// wrapped its output in one despite being told not to.
func StripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	lines = lines[1:]
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
