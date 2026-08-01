package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
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
	findings = append(findings, checkMessages(repoRoot)...)
	findings = append(findings, checkSnippetReferences(repoRoot)...)
	findings = append(findings, checkUseAProviderTemplate(repoRoot)...)

	return findings
}

// requiredTemplateSections is docs/tutorials/use-a-provider/TEMPLATE.md's
// nine required sections, in the required order (see B5's "the settler
// pattern before adding a second provider" ordering note, and the
// template's own "Checklist for SDK maintainers"). Each is matched as a
// literal "## N. " heading prefix.
var requiredTemplateSections = []string{
	"1. Install and configure one provider",
	"2. Create a payment",
	"3. Handle `NextAction` with a single switch",
	"4. The return handler",
	"5. The webhook handler",
	"6. The settler pattern",
	"7. Adding a second provider",
	"8. Testing with `mock`",
	"9. Going to production",
}

// checkUseAProviderTemplate is the "checklist CI can verify" for
// docs/tutorials/use-a-provider/TEMPLATE.md: every required section
// heading must be present, in order, so an edit can't silently drop a
// section or reorder section 6 (the settler pattern) after section 7 (a
// second provider) — an ordering this template's own prose calls out as
// deliberate, not incidental.
func checkUseAProviderTemplate(repoRoot string) []Finding {
	var f []Finding
	path := filepath.Join(repoRoot, "docs", "tutorials", "use-a-provider", "TEMPLATE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f
		}
		return append(f, errf("io", "reading %s: %v", path, err))
	}
	content := string(data)

	searchFrom := 0
	lastFound := ""
	for _, section := range requiredTemplateSections {
		heading := "## " + section
		idx := strings.Index(content[searchFrom:], heading)
		if idx < 0 {
			f = append(f, errf("missing-template-section",
				"docs/tutorials/use-a-provider/TEMPLATE.md: missing required section %q, or it appears out of order after %q",
				section, lastFound))
			continue
		}
		searchFrom += idx + len(heading)
		lastFound = section
	}
	return f
}

// snippetRefRe matches a pymdownx.snippets inline reference,
// --8<-- "path" or --8<-- "path:section" (see mkdocs.yml's
// pymdownx.snippets config), inside a Markdown file. It deliberately
// does not match the block form (a --8<-- line followed by a list of
// quoted paths) since nothing in this repository's docs uses that form;
// see docs/examples/scenario.md for the form this does check.
var snippetRefRe = regexp.MustCompile(`--8<--\s+"([^"]+)"`)

// snippetSectionRe matches a pymdownx.snippets named-section marker,
// e.g. "# --8<-- [start:collect-request]", regardless of the
// surrounding comment syntax (#, //, <!--, ...): the marker text itself
// is what pymdownx.snippets and this check both key off.
var snippetSectionRe = regexp.MustCompile(`--8<--\s*\[\s*(start|end)\s*:\s*([A-Za-z0-9_-]+)\s*\]`)

// checkSnippetReferences validates every pymdownx.snippets reference
// under docs/ (spec/03-manifest-dsl.md's example-inclusion convention,
// mkdocs.yml's pymdownx.snippets config): the referenced file must
// exist, and if the reference names a section, that file must actually
// contain a matching [start:section]/[end:section] marker pair.
// mkdocs' own check_paths catches the missing-file case at site-build
// time; this catches the missing-or-mismatched-section case, which
// check_paths does not, without needing a full mkdocs build in
// `esiipayment lint`.
func checkSnippetReferences(repoRoot string) []Finding {
	var f []Finding
	docsDir := filepath.Join(repoRoot, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		return f
	}
	sectionsCache := map[string]map[string]map[string]bool{} // file -> section -> {start,end} -> present

	_ = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			f = append(f, errf("io", "reading %s: %v", path, err))
			return nil
		}
		relSelf, _ := filepath.Rel(repoRoot, path)
		for _, m := range snippetRefRe.FindAllStringSubmatch(string(data), -1) {
			ref := m[1]
			target := ref
			section := ""
			if idx := strings.LastIndex(ref, ":"); idx >= 0 {
				target, section = ref[:idx], ref[idx+1:]
			}
			targetPath := filepath.Join(repoRoot, filepath.FromSlash(target))
			targetData, err := os.ReadFile(targetPath)
			if err != nil {
				f = append(f, errf("missing-snippet-target",
					"%s: --8<-- %q references %q, which does not exist", relSelf, ref, target))
				continue
			}
			if section == "" {
				continue
			}
			sections, ok := sectionsCache[target]
			if !ok {
				sections = parseSnippetSections(string(targetData))
				sectionsCache[target] = sections
			}
			marks, ok := sections[section]
			if !ok || !marks["start"] || !marks["end"] {
				f = append(f, errf("missing-snippet-section",
					"%s: --8<-- %q references section %q, which has no matching [start:%s]/[end:%s] marker pair in %q",
					relSelf, ref, section, section, section, target))
			}
		}
		return nil
	})
	return f
}

// parseSnippetSections finds every [start:name]/[end:name] marker in
// content and reports, per section name, which of start/end it saw.
func parseSnippetSections(content string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, m := range snippetSectionRe.FindAllStringSubmatch(content, -1) {
		kind, name := m[1], m[2]
		if out[name] == nil {
			out[name] = map[string]bool{}
		}
		out[name][kind] = true
	}
	return out
}

// messageCatalogue mirrors schema/messages.v1.schema.json for the
// purposes of this check: only the shape checkMessages needs to
// validate, not a full strict decode (message files are plain JSON data,
// not part of the DSL vocabulary these YAML strict-decode rules exist
// for).
type messageCatalogue struct {
	Language                string            `json:"language"`
	LanguageName            string            `json:"language_name"`
	ReviewedByNativeSpeaker bool              `json:"reviewed_by_native_speaker"`
	Messages                map[string]string `json:"messages"`
}

// checkMessages validates every messages/<language>.json
// (schema/messages.v1.schema.json, docs/localization.md): every
// FailureCode member must be present with a non-empty message, and the
// declared `language` must match the file's own name.
func checkMessages(repoRoot string) []Finding {
	var f []Finding
	dir := filepath.Join(repoRoot, "messages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return f
		}
		return append(f, errf("io", "reading %s: %v", dir, err))
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			f = append(f, errf("io", "reading %s: %v", path, err))
			continue
		}
		var mc messageCatalogue
		if err := json.Unmarshal(data, &mc); err != nil {
			f = append(f, errf("parse", "%s: %v", path, err))
			continue
		}
		wantLang := strings.TrimSuffix(e.Name(), ".json")
		if mc.Language != wantLang {
			f = append(f, errf("messages-mismatch", "%s: language %q does not match file name %q", path, mc.Language, wantLang))
		}
		for code := range model.FailureCodes {
			msg, ok := mc.Messages[code]
			if !ok || strings.TrimSpace(msg) == "" {
				f = append(f, errf("missing-message", "%s: no message for FailureCode %q", path, code))
			}
		}
		for code := range mc.Messages {
			if !model.FailureCodes[code] {
				f = append(f, errf("closed-enum", "%s: message key %q is not a member of FailureCode", path, code))
			}
		}
	}
	return f
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
