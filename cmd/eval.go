package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/320exh/prompt-diff/internal/eval"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/store"
	"github.com/spf13/cobra"
)

var (
	evalPrompt  string
	evalTests   string
	evalModels  string
	evalOutput  string
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run a test-suite benchmark across models",
	Long: `Evaluate prompt changes across a test-case matrix before pushing to production.

Models are dispatched to their provider by name prefix:
  gpt-*/o1*/o3*      -> OpenAI API       (OPENAI_API_KEY)
  claude-*           -> Anthropic API    (ANTHROPIC_API_KEY)
  gemini-*           -> Google Gemini    (GEMINI_API_KEY)
  llama*, qwen*, ... -> Ollama (local)   (http://localhost:11434)

A standalone HTML report is written to --output.

Examples:
  prompt-diff eval --prompt prompts/agent_v2.prompt --tests tests/eval_suite.json \
    --models llama3.1:8b,claude-3-5-sonnet --output report.html`,
	Args: cobra.NoArgs,
	RunE: runEval,
}

func init() {
	evalCmd.Flags().StringVar(&evalPrompt, "prompt", "", "path to the .prompt template (required)")
	evalCmd.Flags().StringVar(&evalTests, "tests", "", "path to the eval suite JSON (required)")
	evalCmd.Flags().StringVar(&evalModels, "models", "", "comma-separated model list (default: the prompt's frontmatter models)")
	evalCmd.Flags().StringVar(&evalOutput, "output", "report.html", "path to write the HTML report")
	_ = evalCmd.MarkFlagRequired("prompt")
	_ = evalCmd.MarkFlagRequired("tests")
}

func runEval(cmd *cobra.Command, args []string) error {
	src, err := os.ReadFile(evalPrompt)
	if err != nil {
		return fmt.Errorf("reading prompt %q: %w", evalPrompt, err)
	}
	p, err := prompt.Parse(src)
	if err != nil {
		return fmt.Errorf("parsing prompt %q: %w", evalPrompt, err)
	}
	suite, err := eval.LoadSuite(evalTests)
	if err != nil {
		return fmt.Errorf("reading suite %q: %w", evalTests, err)
	}

	models := splitModels(evalModels)
	if len(models) == 0 {
		models = p.Models
	}
	if len(models) == 0 {
		return fmt.Errorf("no models given via --models and none declared in the prompt frontmatter")
	}

	report := eval.Run(cmd.Context(), p, suite, models)
	if err := eval.WriteReport(evalOutput, &report); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	total, passed, failed, skipped := report.Counts()
	fmt.Printf("Suite %q: %d case(s) x %d model(s)\n", suite.Name, len(suite.Cases), len(models))
	fmt.Printf("  passed: %d, failed: %d, skipped: %d, total: %d\n", passed, failed, skipped, total)
	fmt.Printf("Report written to %s\n", evalOutput)

	if s, err := store.Open(); err == nil {
		id, serr := s.RecordRun(evalPrompt, suite.Name, strings.Join(models, ","), total, passed, failed, skipped)
		if serr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not store run in SQLite: %v\n", serr)
		} else {
			fmt.Printf("Run stored in local SQLite store (id %d); see `prompt-diff runs`\n", id)
		}
		_ = s.Close()
	} else {
		fmt.Fprintf(os.Stderr, "warning: could not open SQLite store: %v\n", err)
	}
	return nil
}

func splitModels(csv string) []string {
	var out []string
	for _, m := range strings.Split(csv, ",") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	return out
}
