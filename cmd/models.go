package cmd

import (
	"fmt"
	"time"

	"github.com/320exh/prompt-diff/internal/cost"
	"github.com/spf13/cobra"
)

var modelsCheck bool

// staleAfterDays is how old the built-in price table can get before
// `prompt-diff models --check` starts warning.
const staleAfterDays = 90

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show the built-in cost table and its freshness",
	Args:  cobra.NoArgs,
	RunE:  runModels,
}

func init() {
	modelsCmd.Flags().BoolVar(&modelsCheck, "check", false, "exit nonzero if the price table is stale")
	rootCmd.AddCommand(modelsCmd)
}

func runModels(cmd *cobra.Command, args []string) error {
	asOf, err := time.Parse("2006-01-02", cost.PricesAsOf)
	if err != nil {
		return fmt.Errorf("internal: bad PricesAsOf: %w", err)
	}
	age := int(time.Since(asOf).Hours() / 24)
	fmt.Printf("Built-in prices as of: %s (%d day(s) old)\n", cost.PricesAsOf, age)

	if age > staleAfterDays {
		fmt.Printf("warning: price table is older than %d days; costs may be inaccurate. Check for a newer prompt-diff release.\n", staleAfterDays)
		if modelsCheck {
			return fmt.Errorf("price table stale (%d days old, limit %d)", age, staleAfterDays)
		}
	}

	fmt.Println()
	fmt.Printf("%-28s %s\n", "MODEL", "$/1M TOKENS")
	for _, entry := range cost.List() {
		suffix := ""
		if entry.Overridden {
			suffix = " (override)"
		}
		fmt.Printf("%-28s %.2f%s\n", entry.Model, entry.Per1M, suffix)
	}
	return nil
}
