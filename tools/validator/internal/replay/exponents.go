package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadExponents reads vectors/money/exponents.json. Per
// spec/01-domain-model.md#money, every runtime (including this reference
// validator) must read the minor-unit exponent table from this file
// rather than hardcoding it or deriving it from locale data.
func LoadExponents(repoRoot string) (map[string]int, error) {
	path := filepath.Join(repoRoot, "vectors", "money", "exponents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var doc struct {
		Exponents map[string]int `json:"exponents"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return doc.Exponents, nil
}
