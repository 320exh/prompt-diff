package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Suite is an eval test-case matrix.
type Suite struct {
	Name   string `json:"name"`
	Models []string `json:"models"`
	Cases  []Case `json:"cases"`
}

// Case is a single expected output.
type Case struct {
	ID     string                 `json:"id"`
	Input  string                 `json:"input"`
	Expect map[string]interface{} `json:"expect"`
}

// LoadSuite reads a suite JSON file.
func LoadSuite(path string) (*Suite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Suite
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Name == "" {
		s.Name = strings.TrimSuffix(path, ".json")
	}
	return &s, nil
}

// Validate checks a suite for obvious problems.
func (s *Suite) Validate() error {
	if len(s.Cases) == 0 {
		return fmt.Errorf("suite has no test cases")
	}
	return nil
}