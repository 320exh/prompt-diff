package cmd

import (
	"fmt"

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
