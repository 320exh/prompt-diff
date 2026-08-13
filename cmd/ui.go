package cmd

import (
	"github.com/320exh/prompt-diff/internal/uiserver"
	"github.com/spf13/cobra"
)

var uiPort int

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the local web workspace",
	Long: `Spin up the embedded local web dashboard for interactive variable
tuning, a live token counter, and side-by-side model output comparison.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return uiserver.New(uiPort).Serve()
	},
}

func init() {
	uiCmd.Flags().IntVar(&uiPort, "port", 8080, "port to bind the dashboard to")
}
