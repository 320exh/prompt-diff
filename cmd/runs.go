package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/320exh/prompt-diff/internal/report"
	"github.com/320exh/prompt-diff/internal/store"
	"github.com/spf13/cobra"
)

var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List past eval runs from the local SQLite store",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Open()
		if err != nil {
			return err
		}
		defer s.Close()

		runs, err := s.ListRuns()
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			fmt.Println("No eval runs stored yet. Run `prompt-diff eval` first.")
			return nil
		}
		fmt.Printf("%-4s  %-19s  %-16s  %-16s  %5s %5s %5s %5s\n",
			"ID", "Created", "Prompt", "Suite", "Pass", "Fail", "Skip", "Total")
		for _, r := range runs {
			fmt.Printf("%-4d  %-19s  %-16.16s  %-16.16s  %5d %5d %5d %5d\n",
				r.ID, r.CreatedAt, r.PromptPath, r.Suite, r.Passed, r.Failed, r.Skipped, r.Total)
		}
		return nil
	},
}

var runsCompareCmd = &cobra.Command{
	Use:   "compare <run-id-1> <run-id-2>",
	Short: "Compare two stored eval runs: pass-rate delta and totals",
	Args:  cobra.ExactArgs(2),
	RunE:  runRunsCompare,
}

var runsExportOut string
var runsExportFormat string

var runsExportCmd = &cobra.Command{
	Use:   "export <run-id> [run-id...]",
	Short: "Export stored eval runs as a markdown or PDF benchmark report",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRunsExport,
}

func init() {
	runsCmd.AddCommand(runsCompareCmd)
	runsExportCmd.Flags().StringVar(&runsExportOut, "out", "", "write report to file instead of stdout (required for --format pdf)")
	runsExportCmd.Flags().StringVar(&runsExportFormat, "format", "md", "report format: md or pdf")
	runsCmd.AddCommand(runsExportCmd)
}

func loadRunsByID(ids []int64) ([]store.Run, error) {
	s, err := store.Open()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	runs, err := s.ListRuns()
	if err != nil {
		return nil, err
	}
	return report.SelectByID(runs, ids)
}

func runRunsExport(cmd *cobra.Command, args []string) error {
	ids := make([]int64, 0, len(args))
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid run id %q: %w", a, err)
		}
		ids = append(ids, id)
	}

	selected, err := loadRunsByID(ids)
	if err != nil {
		return err
	}

	switch runsExportFormat {
	case "md", "":
		out := report.Markdown(selected)
		if runsExportOut == "" {
			fmt.Print(out)
			return nil
		}
		return os.WriteFile(runsExportOut, []byte(out), 0o644)
	case "pdf":
		if runsExportOut == "" {
			return fmt.Errorf("--format pdf requires --out <file>")
		}
		f, err := os.Create(runsExportOut)
		if err != nil {
			return err
		}
		defer f.Close()
		return report.PDF(selected, f)
	default:
		return fmt.Errorf("unknown --format %q (want md or pdf)", runsExportFormat)
	}
}

func runRunsCompare(cmd *cobra.Command, args []string) error {
	id1, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run id %q: %w", args[0], err)
	}
	id2, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run id %q: %w", args[1], err)
	}

	selected, err := loadRunsByID([]int64{id1, id2})
	if err != nil {
		return err
	}
	r1, r2 := selected[0], selected[1]
	pr1, pr2 := report.PassRate(r1), report.PassRate(r2)

	fmt.Printf("Run %d: %s  %s / %s  pass %d/%d (%.1f%%)\n", r1.ID, r1.CreatedAt, r1.PromptPath, r1.Suite, r1.Passed, r1.Total, pr1)
	fmt.Printf("Run %d: %s  %s / %s  pass %d/%d (%.1f%%)\n", r2.ID, r2.CreatedAt, r2.PromptPath, r2.Suite, r2.Passed, r2.Total, pr2)
	fmt.Println()
	fmt.Printf("Pass rate delta: %+.1f%%\n", pr2-pr1)
	fmt.Printf("Passed delta:    %+d\n", r2.Passed-r1.Passed)
	fmt.Printf("Failed delta:    %+d\n", r2.Failed-r1.Failed)
	fmt.Printf("Total delta:     %+d\n", r2.Total-r1.Total)
	return nil
}
