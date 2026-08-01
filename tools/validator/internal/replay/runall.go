package replay

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
)

// Result is one cassette's replay outcome, ready for the `esiipayment
// replay` command to print or compare against --assert-golden.
type Result struct {
	CassetteFile  string // e.g. "collect.redirect.yaml"
	ExpectedFile  string // e.g. "collect.redirect.json"
	CanonicalJSON string // "" if Err != nil
	ExpectedJSON  string // "" if no matching expected/ file exists
	Err           error
}

// RunAll replays every cassette in providerDir/cassettes/ against
// providerDir/manifest.yaml and, if a matching providerDir/expected/
// file exists, loads it (already required to be canonical JSON itself,
// per spec/06-conformance.md) for comparison.
func RunAll(providerDir string, exponents map[string]int) ([]Result, error) {
	m, err := model.LoadManifest(providerDir)
	if err != nil {
		return nil, err
	}
	cassettes, err := model.LoadCassettes(providerDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for name := range cassettes {
		names = append(names, name)
	}
	sort.Strings(names)

	var results []Result
	for _, name := range names {
		cas := cassettes[name]
		expectedName := strings.TrimSuffix(name, ".yaml") + ".json"
		r := Result{CassetteFile: name, ExpectedFile: expectedName}

		pr, err := Run(m, cas, exponents)
		if err != nil {
			r.Err = err
			results = append(results, r)
			continue
		}
		canonical, err := CanonicalJSON(pr.ToCanonicalValue())
		if err != nil {
			r.Err = err
			results = append(results, r)
			continue
		}
		r.CanonicalJSON = canonical

		expectedPath := filepath.Join(providerDir, "expected", expectedName)
		if data, err := os.ReadFile(expectedPath); err == nil {
			r.ExpectedJSON = strings.TrimRight(string(data), "\n")
		}

		results = append(results, r)
	}
	return results, nil
}

// NamedResult pairs one cassette's file name with its replayed
// PaymentResult (or the error replaying it), for callers (the reference
// integrator: internal/integrator) that need the structured result
// rather than its canonical JSON string.
type NamedResult struct {
	CassetteFile string
	Result       *PaymentResult
	Err          error
}

// RunAllResults is RunAll's structured-result form: same cassette
// loading and ordering, but returns each cassette's *PaymentResult
// directly instead of canonicalizing it to a JSON string.
func RunAllResults(providerDir string, exponents map[string]int) ([]NamedResult, error) {
	m, err := model.LoadManifest(providerDir)
	if err != nil {
		return nil, err
	}
	cassettes, err := model.LoadCassettes(providerDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for name := range cassettes {
		names = append(names, name)
	}
	sort.Strings(names)

	var results []NamedResult
	for _, name := range names {
		pr, err := Run(m, cassettes[name], exponents)
		results = append(results, NamedResult{CassetteFile: name, Result: pr, Err: err})
	}
	return results, nil
}
