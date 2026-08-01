// Command esiipayment is the ESIIPayment reference validator. It is the only
// application code permitted in this repository outside CI workflow
// definitions: see spec/00-overview.md and README.md. It is intended to
// be built as a single static binary and distributed as a container
// image (see tools/validator/Dockerfile) so no contributor needs a Go
// toolchain to validate a manifest.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/catalog"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/checks"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/integrator"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/replay"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "validate":
		err = runValidate(os.Args[2:])
	case "replay":
		err = runReplay(os.Args[2:])
	case "lint":
		err = runLint(os.Args[2:])
	case "catalog":
		err = runCatalog(os.Args[2:])
	case "conformance-integrator":
		err = runConformanceIntegrator(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `esiipayment: ESIIPayment reference validator

Usage:
  esiipayment validate <provider-dir>
      Schema and semantic validation of one provider's manifest.yaml,
      metadata.yaml, and cassettes/.

  esiipayment replay <provider-dir> --assert-golden
      Replays every cassette in <provider-dir>/cassettes/ against its
      manifest.yaml and diffs the canonical output against
      <provider-dir>/expected/<name>.json.

  esiipayment lint [repo-root]
      Repo-wide checks: no code files under providers/, vectors/, or
      schema/; every manifest.yaml carries the required schema header;
      every provider's metadata is complete and consistent. Defaults to
      the current directory as repo-root.

  esiipayment catalog [repo-root]
      Generates the provider catalog and capability matrix as Markdown
      to stdout, from every provider's manifest.yaml/metadata.yaml.
      Defaults to the current directory as repo-root.

  esiipayment conformance-integrator [repo-root]
      Runs the reference integrator (internal/integrator: one generic
      handler over PaymentStatus/NextAction/FailureCode/RetryClass,
      never a provider identifier) against every cassette of every
      manifest.yaml-driven provider, unmodified. Fails if any cassette
      produces an outcome the same generic handler cannot process
      without a provider-specific branch — the machine-checkable form of
      Invariant I12. Defaults to the current directory as repo-root.
      Native providers (spec/03-manifest-dsl.md#native-providers) are
      skipped with a note, the same as esiipayment replay.`)
}

func runValidate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: esiipayment validate <provider-dir>")
	}
	dir := args[0]
	findings := checks.ValidateProvider(dir)
	printFindings(findings)
	if checks.HasErrors(findings) {
		return fmt.Errorf("validation failed for %s", dir)
	}
	return nil
}

func runCatalog(args []string) error {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	out, err := catalog.Generate(root)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func runLint(args []string) error {
	root := "."
	for _, a := range args {
		if a != "--assert-golden" {
			root = a
		}
	}
	findings := checks.LintRepo(root)
	printFindings(findings)
	if checks.HasErrors(findings) {
		return fmt.Errorf("lint failed")
	}
	return nil
}

func runReplay(args []string) error {
	var dir string
	assertGolden := false
	for _, a := range args {
		if a == "--assert-golden" {
			assertGolden = true
			continue
		}
		dir = a
	}
	if dir == "" {
		return fmt.Errorf("usage: esiipayment replay <provider-dir> [--assert-golden]")
	}

	if model.IsNativeProvider(dir) {
		fmt.Printf("%s is a native-implementation provider (capabilities.yaml, no manifest.yaml): this reference tool has no manifest to replay cassettes against. Cassette/golden-file presence is still checked by `esiipayment validate`; producing and verifying golden output against these cassettes is each native implementation's own conformance suite's job. See spec/03-manifest-dsl.md#native-providers.\n", dir)
		return nil
	}

	repoRoot, err := findRepoRoot(dir)
	if err != nil {
		return err
	}
	exponents, err := replay.LoadExponents(repoRoot)
	if err != nil {
		return err
	}

	results, err := replay.RunAll(dir, exponents)
	if err != nil {
		return err
	}

	failed := false
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("FAIL %s: %v\n", r.CassetteFile, r.Err)
			failed = true
			continue
		}
		if !assertGolden {
			fmt.Printf("ok   %s -> %s\n", r.CassetteFile, r.CanonicalJSON)
			continue
		}
		if r.ExpectedJSON == "" {
			fmt.Printf("FAIL %s: no matching expected/%s\n", r.CassetteFile, r.ExpectedFile)
			failed = true
			continue
		}
		if r.CanonicalJSON != r.ExpectedJSON {
			fmt.Printf("FAIL %s: golden mismatch\n  got:      %s\n  expected: %s\n", r.CassetteFile, r.CanonicalJSON, r.ExpectedJSON)
			failed = true
			continue
		}
		fmt.Printf("ok   %s\n", r.CassetteFile)
	}

	if failed {
		return fmt.Errorf("replay failed for %s", dir)
	}
	return nil
}

// runConformanceIntegrator implements `esiipayment conformance-integrator`:
// the reference integrator (internal/integrator) run unmodified against
// every cassette of every manifest.yaml-driven provider under
// <repo-root>/providers/. See usage()'s help text and
// internal/integrator's package doc for what this checks and why.
func runConformanceIntegrator(args []string) error {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	repoRoot, err := findRepoRoot(root)
	if err != nil {
		return err
	}
	exponents, err := replay.LoadExponents(repoRoot)
	if err != nil {
		return err
	}

	providersDir := filepath.Join(repoRoot, "providers")
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", providersDir, err)
	}

	failed := false
	checkedAny := false
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_template" {
			continue
		}
		dir := filepath.Join(providersDir, e.Name())
		if model.IsNativeProvider(dir) {
			fmt.Printf("skip %s: native-implementation provider; see spec/03-manifest-dsl.md#native-providers\n", e.Name())
			continue
		}
		checkedAny = true

		results, err := replay.RunAllResults(dir, exponents)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", e.Name(), err)
			failed = true
			continue
		}
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("FAIL %s/%s: replay error: %v\n", e.Name(), r.CassetteFile, r.Err)
				failed = true
				continue
			}
			outcome, err := integrator.Handle(r.Result)
			if err != nil {
				fmt.Printf("FAIL %s/%s: reference integrator could not handle this result unmodified: %v\n", e.Name(), r.CassetteFile, err)
				failed = true
				continue
			}
			fmt.Printf("ok   %s/%s: %s\n", e.Name(), r.CassetteFile, outcome.Description)
		}
	}

	if !checkedAny {
		return fmt.Errorf("no manifest.yaml-driven providers found under %s", providersDir)
	}
	if failed {
		return fmt.Errorf("conformance-integrator failed: the reference integrator needed provider-specific handling somewhere, breaking Invariant I12's promise")
	}
	return nil
}

func printFindings(findings []checks.Finding) {
	for _, f := range findings {
		fmt.Println(f.String())
	}
	if len(findings) == 0 {
		fmt.Println("no findings")
	}
}

// findRepoRoot walks upward from providerDir looking for a VERSION file,
// which sits at the repository root (spec/08-versioning.md).
func findRepoRoot(providerDir string) (string, error) {
	dir, err := filepath.Abs(providerDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find repository root (a VERSION file) above %s", providerDir)
}
