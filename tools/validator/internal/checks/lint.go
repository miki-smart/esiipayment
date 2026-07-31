package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// forbiddenCodeExtensions is the governing rule of this repository: no
// application code, in any language, under providers/, vectors/, or
// schema/. See spec/00-overview.md and README.md.
var forbiddenCodeExtensions = []string{
	".cs", ".py", ".ts", ".js", ".go", ".php", ".rb", ".java",
}

// LintRepo runs the repo-wide checks `esiipayment lint` is responsible for:
// no code files under providers/, vectors/, or schema/; every provider's
// manifest.yaml carries the required schema header; and every provider's
// metadata is complete and self-consistent. It returns one Finding list
// per concern, each message prefixed with the file or provider it's
// about, since lint (unlike validate) spans the whole repository rather
// than one provider directory.
func LintRepo(repoRoot string) []Finding {
	var findings []Finding

	findings = append(findings, checkNoCodeFiles(repoRoot)...)
	findings = append(findings, checkAllProviders(repoRoot)...)

	return findings
}

func checkNoCodeFiles(repoRoot string) []Finding {
	var f []Finding
	for _, sub := range []string{"providers", "vectors", "schema"} {
		root := filepath.Join(repoRoot, sub)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			for _, forbidden := range forbiddenCodeExtensions {
				if ext == forbidden {
					rel, _ := filepath.Rel(repoRoot, path)
					f = append(f, errf("no-code-in-data-dirs",
						"%s: application code (%s) is not permitted under providers/, vectors/, or schema/; the only executable code in this repository lives under tools/", rel, ext))
				}
			}
			return nil
		})
	}
	return f
}

func checkAllProviders(repoRoot string) []Finding {
	var f []Finding
	providersDir := filepath.Join(repoRoot, "providers")
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return append(f, errf("io", "reading %s: %v", providersDir, err))
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			// _template is a skeleton with intentional TODOs and is not
			// expected to validate.
			continue
		}
		dir := filepath.Join(providersDir, e.Name())
		for _, finding := range ValidateProvider(dir) {
			finding.Message = fmt.Sprintf("%s: %s", e.Name(), finding.Message)
			f = append(f, finding)
		}
	}
	return f
}
