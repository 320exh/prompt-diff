package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/320exh/prompt-diff/internal/gitutil"
	"github.com/320exh/prompt-diff/internal/prompt"
	"github.com/320exh/prompt-diff/internal/promptdiff"
	"github.com/spf13/cobra"
)

var (
	diffV1 string
	diffV2 string
)

var diffCmd = &cobra.Command{
	Use:   "diff <file.prompt>",
	Short: "Diff a .prompt template against a git revision",
	Long: `Compare a .prompt template against another revision (default HEAD).

Examples:
  prompt-diff diff system_prompt.prompt
  prompt-diff diff system_prompt.prompt --v1=v1.2.0 --v2=HEAD`,
	Args: cobra.ExactArgs(1),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffV1, "v1", "HEAD", "older revision (git ref) to compare against")
	diffCmd.Flags().StringVar(&diffV2, "v2", "", "newer revision (git ref); empty means the working copy")
}

func runDiff(cmd *cobra.Command, args []string) error {
	path := args[0]

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

	d := promptdiff.Compare(oldP, newP)

	fmt.Printf("Prompt: %s\n", path)
	fmt.Printf("Target Models: %s\n", strings.Join(newP.Models, ", "))
	fmt.Println()
	fmt.Printf("Token Delta: %+d tokens (%+.1f%%)\n", d.TokenDelta, d.TokenPercent)
	fmt.Println("Cost Projection (100k invocations):")
	for _, c := range d.Costs {
		fmt.Printf("  - %-19s $%.2f -> $%.2f (+$%.2f)\n", c.Model+":", c.Old, c.New, c.Delta)
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
