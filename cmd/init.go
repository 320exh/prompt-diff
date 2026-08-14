package cmd

import (
	"fmt"

	"github.com/320exh/prompt-diff/internal/scaffold"
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

func runInit(cmd *cobra.Command, args []string) error {
	written, skipped, err := scaffold.Write("", initForce)
	if err != nil {
		return err
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
