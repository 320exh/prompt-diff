// Package gitutil wraps git plumbing used by prompt-diff.
package gitutil

import (
	"bytes"
	"errors"
	"os/exec"
)

// Show returns the content of a file at a git ref via `git show REF:PATH`.
func Show(ref, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", ref+":"+path)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, errors.New(errb.String())
	}
	return out.Bytes(), nil
}

// Exists reports whether path is tracked at ref.
func Exists(ref, path string) bool {
	_, err := Show(ref, path)
	return err == nil
}