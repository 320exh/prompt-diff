package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a starter .prompt file, eval suite, and .prompt-diff.yml",
	Long: `Writes example.prompt, example.eval.json, and .prompt-diff.yml into the
current directory so a new project has something to run "prompt-diff diff",
"prompt-diff eval", and "prompt-diff lint" against immediately.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite files that already exist")
	rootCmd.AddCommand(initCmd)
}

var initFiles = map[string]string{
	"example.prompt": `---
name: Customer Support Classifier
version: 1.0.0
models:
  - gpt-4o
  - claude-3-5-sonnet
variables:
  - user_context
---
You are an expert customer support agent.
Analyze the user query using the context below.

User context:
{{ user_context }}

Return JSON with the classification and confidence score.
`,
	"example.eval.json": `{
  "name": "example regression suite",
  "cases": [
    {
      "id": "refund-request",
      "input": "I was charged twice this month and want a refund.",
      "expect": {
        "classification": { "$in": ["refund", "billing"] },
        "confidence": { "$gte": 0.5 }
      }
    },
    {
      "id": "empty-query",
      "input": "",
      "expect": {
        "classification": { "$in": ["unspecified", "unknown"] },
        "confidence": { "$lte": 0.5 }
      }
    }
  ]
}
`,
	".prompt-diff.yml": `# prompt-diff config. See https://github.com/320exh/prompt-diff for docs.
# price_overrides maps model name -> USD price per 1M input tokens, for
# negotiated or non-USD pricing the built-in cost table doesn't know about.
price_overrides: {}
`,
}

func runInit(cmd *cobra.Command, args []string) error {
	var written, skipped []string
	for _, name := range []string{"example.prompt", "example.eval.json", ".prompt-diff.yml"} {
		if !initForce {
			if _, err := os.Stat(name); err == nil {
				skipped = append(skipped, name)
				continue
			}
		}
		if err := os.WriteFile(name, []byte(initFiles[name]), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
		written = append(written, name)
	}
	for _, name := range written {
		fmt.Printf("created %s\n", name)
	}
	for _, name := range skipped {
		fmt.Printf("skipped %s (already exists; use --force to overwrite)\n", name)
	}
	if len(written) > 0 {
		fmt.Println("\nNext: prompt-diff lint example.prompt")
	}
	return nil
}
