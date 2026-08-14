package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

var runsExportCmd = &cobra.Command{
	Use:   "export <run-id> [run-id...]",
	Short: "Export stored eval runs as a markdown benchmark report",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRunsExport,
}

func init() {
	runsCmd.AddCommand(runsCompareCmd)
	runsExportCmd.Flags().StringVar(&runsExportOut, "out", "", "write report to file instead of stdout")
	runsCmd.AddCommand(runsExportCmd)
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

	selected := make([]store.Run, 0, len(ids))
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			return fmt.Errorf("run %d not found", id)
		}
		selected = append(selected, r)
	}

	var b strings.Builder
	b.WriteString("# prompt-diff benchmark report\n\n")
	b.WriteString(fmt.Sprintf("Generated %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("| ID | Created | Prompt | Suite | Models | Pass | Fail | Skip | Total | Pass Rate |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range selected {
		rate := 0.0
		if r.Total > 0 {
			rate = float64(r.Passed) / float64(r.Total) * 100
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %d | %d | %d | %d | %.1f%% |\n",
			r.ID, r.CreatedAt, r.PromptPath, r.Suite, r.Models, r.Passed, r.Failed, r.Skipped, r.Total, rate))
	}

	if len(selected) == 2 {
		r1, r2 := selected[0], selected[1]
		pr := func(r store.Run) float64 {
			if r.Total == 0 {
				return 0
			}
			return float64(r.Passed) / float64(r.Total) * 100
		}
		b.WriteString("\n## Delta (run " + strconv.FormatInt(r1.ID, 10) + " -> " + strconv.FormatInt(r2.ID, 10) + ")\n\n")
		b.WriteString(fmt.Sprintf("- Pass rate: %+.1f%%\n", pr(r2)-pr(r1)))
		b.WriteString(fmt.Sprintf("- Passed: %+d\n", r2.Passed-r1.Passed))
		b.WriteString(fmt.Sprintf("- Failed: %+d\n", r2.Failed-r1.Failed))
		b.WriteString(fmt.Sprintf("- Total: %+d\n", r2.Total-r1.Total))
	}

	out := b.String()
	if runsExportOut == "" {
		fmt.Print(out)
		return nil
	}
	return os.WriteFile(runsExportOut, []byte(out), 0o644)
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
