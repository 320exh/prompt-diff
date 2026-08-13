package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/320exh/prompt-diff/internal/gitutil"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/promptdiff"
	"github.com/spf13/cobra"
)

var (
	diffV1   string
	diffV2   string
	diffJSON bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <file.prompt> [file2.prompt]",
	Short: "Diff a .prompt template against a git revision or another file",
	Long: `Compare a .prompt template against another revision (default HEAD), or
against a second file on disk when given two paths.

Examples:
  prompt-diff diff system_prompt_example.prompt
  prompt-diff diff system_prompt_example.prompt --v1=v1.2.0 --v2=HEAD
  prompt-diff diff a.prompt b.prompt`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffV1, "v1", "HEAD", "older revision (git ref) to compare against")
	diffCmd.Flags().StringVar(&diffV2, "v2", "", "newer revision (git ref); empty means the working copy")
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "print the diff as JSON instead of the human-readable report")
}

func runDiff(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Two-file form: `prompt-diff diff a.prompt b.prompt` compares two files
	// on disk directly, bypassing git refs entirely.
	if len(args) == 2 {
		path2 := args[1]
		oldSrc, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		newSrc, err := os.ReadFile(path2)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path2, err)
		}
		oldP, err := prompt.Parse(oldSrc)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		newP, err := prompt.Parse(newSrc)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path2, err)
		}
		return printDiff(path2, oldP, newP)
	}

	oldSrc, err := promptAtRef(path, diffV1)
	if err != nil {
		return fmt.Errorf("reading %s at %q: %w", path, diffV1, err)
	}
	newSrc, err := promptAtRef(path, diffV2)
	if err != nil {
		return fmt.Errorf("reading %s at %q: %w", path, displayRef(diffV2), err)
	}

	oldP, err := prompt.Parse(oldSrc)
	if err != nil {
		return fmt.Errorf("parsing %s at %q: %w", path, diffV1, err)
	}
	newP, err := prompt.Parse(newSrc)
	if err != nil {
		return fmt.Errorf("parsing %s at %q: %w", path, displayRef(diffV2), err)
	}
	return printDiff(path, oldP, newP)
}

func printDiff(path string, oldP, newP *prompt.Template) error {
	d := promptdiff.Compare(oldP, newP)

	if diffJSON {
		out := struct {
			Prompt string   `json:"prompt"`
			Models []string `json:"target_models"`
			promptdiff.Diff
		}{Prompt: path, Models: newP.Models, Diff: d}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Prompt: %s\n", path)
	fmt.Printf("Target Models: %s\n", strings.Join(newP.Models, ", "))
	fmt.Println()
	fmt.Printf("Token Delta: %+d tokens (%+.1f%%)\n", d.TokenDelta, d.TokenPercent)
	fmt.Println("Cost Projection (100k invocations):")
	for _, c := range d.Costs {
		warn := ""
		if c.Approx {
			warn = "  (~approx tokenizer, no published BPE vocab for this model)"
		}
		fmt.Printf("  - %-19s $%.2f -> $%.2f (+$%.2f)%s\n", c.Model+":", c.Old, c.New, c.Delta, warn)
	}

	if len(d.AddedSections)+len(d.RemovedSections)+len(d.ModifiedVars)+len(d.AddedVars)+len(d.RemovedVars) > 0 {
		fmt.Println()
		fmt.Println("Structural Diffs:")
		for _, s := range d.AddedSections {
			fmt.Printf("  + Added section: [%s]\n", s)
		}
		for _, s := range d.RemovedSections {
			fmt.Printf("  - Removed section: [%s]\n", s)
		}
		for _, v := range d.ModifiedVars {
			fmt.Printf("  ~ Modified variable: {{ %s }} -> {{ %s }}\n", v.From, v.To)
		}
		for _, v := range d.AddedVars {
			fmt.Printf("  + Added variable: {{ %s }}\n", v)
		}
		for _, v := range d.RemovedVars {
			fmt.Printf("  - Removed variable: {{ %s }}\n", v)
		}
	}
	return nil
}

// promptAtRef returns the file content at a git ref, or the working copy when
// ref is empty.
func promptAtRef(path, ref string) ([]byte, error) {
	if ref == "" || ref == "WORKING" {
		return os.ReadFile(path)
	}
	return gitutil.Show(ref, path)
}

func displayRef(ref string) string {
	if ref == "" {
		return "WORKING"
	}
	return ref
}
