package cmd

import (
	"fmt"
	"os"

	"github.com/320exh/prompt-diff/internal/interop"
	"github.com/spf13/cobra"
)

var (
	convertIn   string
	convertFrom string
	convertTo   string
	convertOut  string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert a prompt between .prompt and LangChain/LlamaIndex JSON",
	Long: `Converts a prompt template between prompt-diff's .prompt format and the
JSON prompt-template shapes used by LangChain (PromptTemplate.save()) and
LlamaIndex (PromptTemplate), so prompts authored in those frameworks can be
diffed, evaluated, and compressed through prompt-diff, or vice versa.

--from/--to accept: prompt, langchain, llamaindex

Covers the common single-template shape (f-string "{var}" placeholders);
multi-message ChatPromptTemplate and provider-specific schema variants
aren't supported.`,
	Args: cobra.NoArgs,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVar(&convertIn, "in", "", "input file (required)")
	convertCmd.Flags().StringVar(&convertFrom, "from", "prompt", "input format: prompt, langchain, or llamaindex")
	convertCmd.Flags().StringVar(&convertTo, "to", "", "output format: prompt, langchain, or llamaindex (required)")
	convertCmd.Flags().StringVar(&convertOut, "out", "", "output file (required)")
	_ = convertCmd.MarkFlagRequired("in")
	_ = convertCmd.MarkFlagRequired("to")
	_ = convertCmd.MarkFlagRequired("out")
	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(convertIn)
	if err != nil {
		return fmt.Errorf("reading %q: %w", convertIn, err)
	}

	out, err := interop.Convert(data, convertFrom, convertTo)
	if err != nil {
		return fmt.Errorf("converting %q: %w", convertIn, err)
	}

	if err := os.WriteFile(convertOut, out, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", convertOut, err)
	}
	fmt.Printf("Converted %s (%s) -> %s (%s)\n", convertIn, convertFrom, convertOut, convertTo)
	return nil
}
