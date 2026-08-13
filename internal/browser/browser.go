// Package browser opens a URL in the user's default browser, cross-platform.
package browser

import (
	"os/exec"
	"runtime"
)

// Open launches the given URL in the default browser.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
