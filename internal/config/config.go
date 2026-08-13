// Package config loads optional user overrides for prompt-diff, such as
// negotiated or non-USD model pricing that the built-in price list can't
// know about.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File is the parsed shape of .prompt-diff.yml.
type File struct {
	// PriceOverrides maps model name -> USD price per 1M input tokens,
	// overriding internal/cost's built-in list.
	PriceOverrides map[string]float64 `yaml:"price_overrides"`
}

// Load reads .prompt-diff.yml (or .yaml) from the current directory first,
// falling back to the same filename in the user's home directory. Returns a
// zero-value File, no error, when neither exists.
func Load() (*File, error) {
	paths := []string{".prompt-diff.yml", ".prompt-diff.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".prompt-diff.yml"), filepath.Join(home, ".prompt-diff.yaml"))
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var f File
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, err
		}
		return &f, nil
	}
	return &File{}, nil
}
