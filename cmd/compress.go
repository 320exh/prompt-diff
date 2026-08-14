package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/320exh/prompt-diff/internal/eval"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/tokenizer"
	"github.com/spf13/cobra"
)

var (
	compressPrompt  string
	compressModel   string
	compressTests   string
	compressModels  string
	compressMaxLoss float64
	compressOut     string
	compressApply   bool
)

var compressCmd = &cobra.Command{
	Use:   "compress",
	Short: "Auto-compress a prompt to reduce tokens, guarded by its eval suite",
	Long: `Sends the prompt body to an LLM with instructions to rewrite it more
tersely while preserving every instruction and {{ variable }} placeholder,
then reports the token/cost savings.

Pass --tests to guard the rewrite: the compressed prompt is run through the
same eval suite as the original, and the rewrite is rejected (nonzero exit,
nothing written) if its pass rate drops by more than --max-loss percentage
points. Without --tests, compression is unguarded — inspect the diff
yourself before trusting it.

Nothing is written to disk unless --apply is given.`,
	Args: cobra.NoArgs,
	RunE: runCompress,
}

func init() {
	compressCmd.Flags().StringVar(&compressPrompt, "prompt", "", "path to the .prompt template (required)")
	compressCmd.Flags().StringVar(&compressModel, "model", "", "model to use for the compression rewrite (required)")
	compressCmd.Flags().StringVar(&compressTests, "tests", "", "eval suite JSON to guard the rewrite against regressions")
	compressCmd.Flags().StringVar(&compressModels, "models", "", "comma-separated models to score --tests against (default: the prompt's frontmatter models)")
	compressCmd.Flags().Float64Var(&compressMaxLoss, "max-loss", 0, "max allowed pass-rate drop in percentage points (requires --tests)")
	compressCmd.Flags().StringVar(&compressOut, "out", "", "write the compressed prompt here instead of --prompt")
	compressCmd.Flags().BoolVar(&compressApply, "apply", false, "write the compressed prompt to disk (default: report only)")
	_ = compressCmd.MarkFlagRequired("prompt")
	_ = compressCmd.MarkFlagRequired("model")
	rootCmd.AddCommand(compressCmd)
}

const compressInstruction = `You compress LLM system prompts. Rewrite the prompt below to use as few
tokens as possible while preserving every instruction, constraint, and
{{ variable }} placeholder exactly as written (do not rename, add, or
remove any {{ variable }} placeholder). Do not add commentary, headers, or
markdown code fences. Output only the rewritten prompt text.

PROMPT:
`

func runCompress(cmd *cobra.Command, args []string) error {
	src, err := os.ReadFile(compressPrompt)
	if err != nil {
		return fmt.Errorf("reading prompt %q: %w", compressPrompt, err)
	}
	p, err := prompt.Parse(src)
	if err != nil {
		return fmt.Errorf("parsing prompt %q: %w", compressPrompt, err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	rewritten, err := eval.Complete(ctx, compressModel, "", compressInstruction+p.Body)
	if err != nil {
		return fmt.Errorf("compression call to %q failed: %w", compressModel, err)
	}
	compressedBody := strings.TrimSpace(stripCodeFence(rewritten))
	if compressedBody == "" {
		return fmt.Errorf("compression call to %q returned an empty rewrite", compressModel)
	}

	for _, v := range p.Variables {
		if !strings.Contains(compressedBody, "{{ "+v+" }}") && !strings.Contains(compressedBody, "{{"+v+"}}") {
			return fmt.Errorf("rewrite dropped required variable {{ %s }}; rejecting (nothing written)", v)
		}
	}

	before := tokenizer.Encoder.Count(p.Body)
	after := tokenizer.Encoder.Count(compressedBody)
	saved := before - after
	pct := 0.0
	if before > 0 {
		pct = float64(saved) / float64(before) * 100
	}

	fmt.Printf("Tokens: %d -> %d (%+d, %.1f%%)\n", before, after, -saved, -pct)

	if compressTests != "" {
		suite, err := eval.LoadSuite(compressTests)
		if err != nil {
			return fmt.Errorf("reading suite %q: %w", compressTests, err)
		}
		models := splitModels(compressModels)
		if len(models) == 0 {
			models = p.Models
		}
		if len(models) == 0 {
			return fmt.Errorf("no models given via --models and none declared in the prompt frontmatter")
		}

		compressed := &prompt.Template{Name: p.Name, Version: p.Version, Models: p.Models, Variables: p.Variables, Body: compressedBody}

		beforeReport := eval.Run(ctx, p, suite, models)
		afterReport := eval.Run(ctx, compressed, suite, models)

		btotal, bpassed, _, _ := beforeReport.Counts()
		atotal, apassed, _, _ := afterReport.Counts()
		brate, arate := passRateOf(bpassed, btotal), passRateOf(apassed, atotal)

		fmt.Printf("Pass rate: %.1f%% -> %.1f%% (%+.1f%%)\n", brate, arate, arate-brate)

		if arate < brate-compressMaxLoss {
			return fmt.Errorf("rewrite regresses pass rate by %.1f%% (max allowed %.1f%%); rejecting (nothing written)", brate-arate, compressMaxLoss)
		}
	}

	if !compressApply {
		fmt.Println("\n--- compressed prompt (dry run; pass --apply to write) ---")
		fmt.Println(compressedBody)
		return nil
	}

	out := compressOut
	if out == "" {
		out = compressPrompt
	}
	rendered, err := prompt.Render(p, compressedBody)
	if err != nil {
		return fmt.Errorf("rendering compressed prompt: %w", err)
	}
	if err := os.WriteFile(out, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", out, err)
	}
	fmt.Printf("Wrote compressed prompt to %s\n", out)
	return nil
}

func passRateOf(passed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total) * 100
}

// stripCodeFence removes a single leading/trailing ``` fence if the model
// wrapped its output in one despite being told not to.
func stripCodeFence(s string) string {
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

