package cmd

import (
	"fmt"
	"strconv"

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

func init() {
	runsCmd.AddCommand(runsCompareCmd)
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

	s, err := store.Open()
	if err != nil {
		return err
	}
	defer s.Close()

	runs, err := s.ListRuns()
	if err != nil {
		return err
	}
	byID := map[int64]store.Run{}
	for _, r := range runs {
		byID[r.ID] = r
	}
	r1, ok := byID[id1]
	if !ok {
		return fmt.Errorf("run %d not found", id1)
	}
	r2, ok := byID[id2]
	if !ok {
		return fmt.Errorf("run %d not found", id2)
	}

	passRate := func(r store.Run) float64 {
		if r.Total == 0 {
			return 0
		}
		return float64(r.Passed) / float64(r.Total) * 100
	}
	pr1, pr2 := passRate(r1), passRate(r2)

	fmt.Printf("Run %d: %s  %s / %s  pass %d/%d (%.1f%%)\n", r1.ID, r1.CreatedAt, r1.PromptPath, r1.Suite, r1.Passed, r1.Total, pr1)
	fmt.Printf("Run %d: %s  %s / %s  pass %d/%d (%.1f%%)\n", r2.ID, r2.CreatedAt, r2.PromptPath, r2.Suite, r2.Passed, r2.Total, pr2)
	fmt.Println()
	fmt.Printf("Pass rate delta: %+.1f%%\n", pr2-pr1)
	fmt.Printf("Passed delta:    %+d\n", r2.Passed-r1.Passed)
	fmt.Printf("Failed delta:    %+d\n", r2.Failed-r1.Failed)
	fmt.Printf("Total delta:     %+d\n", r2.Total-r1.Total)
	return nil
}
