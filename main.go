package main

import (
	"github.com/320exh/prompt-diff/cmd"
	"github.com/spf13/cobra"
)

func main() {
	// Disable cobra's mousetrap splash so double-clicking the binary
	// launches the dashboard instead of showing a "use cmd.exe" notice.
	cobra.MousetrapHelpText = ""
	cmd.Execute()
}
