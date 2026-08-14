// Package scaffold writes the starter files `prompt-diff init` (and the web
// workspace's Init action) create in a fresh project.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileNames is the scaffold set, in write order.
var FileNames = []string{"example.prompt", "example.eval.json", ".prompt-diff.yml"}

// Files maps scaffold file name -> content.
var Files = map[string]string{
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

// Write writes the scaffold files into dir ("" = CWD). Existing files are
// skipped unless force is set. Returns the written and skipped names.
func Write(dir string, force bool) (written, skipped []string, err error) {
	for _, name := range FileNames {
		path := name
		if dir != "" {
			path = filepath.Join(dir, name)
		}
		if !force {
			if _, statErr := os.Stat(path); statErr == nil {
				skipped = append(skipped, name)
				continue
			}
		}
		if werr := os.WriteFile(path, []byte(Files[name]), 0o644); werr != nil {
			return written, skipped, fmt.Errorf("writing %s: %w", name, werr)
		}
		written = append(written, name)
	}
	return written, skipped, nil
}
