package checks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
)

const requiredCapabilitiesSchemaHeader = "# yaml-language-server: $schema=https://spec.esiipayment.et/schema/capabilities.v1.schema.json"

// ValidateNativeProvider is ValidateProvider's native-implementation
// (capabilities.yaml) path: spec/03-manifest-dsl.md#native-providers. A
// native provider has no flows/errors/webhook for this reference tool to
// interpret, so this checks exactly what's left that's still
// this-repository's responsibility: capabilities.yaml's own shape,
// metadata.yaml, and that cassette/golden-file coverage exists. Whether
// replaying those cassettes actually produces that golden output is the
// native implementation's own per-language conformance suite's job, not
// this tool's — see esiipayment replay's native-provider message.
func ValidateNativeProvider(dir string) []Finding {
	var findings []Finding

	header, err := model.FirstLine(filepath.Join(dir, "capabilities.yaml"))
	if err != nil {
		return append(findings, errf("io", "reading capabilities.yaml: %v", err))
	}
	if header != requiredCapabilitiesSchemaHeader {
		findings = append(findings, errf("schema-header",
			"capabilities.yaml must begin with %q, found %q", requiredCapabilitiesSchemaHeader, header))
	}

	c, err := model.LoadCapabilities(dir)
	if err != nil {
		return append(findings, errf("parse", "%v", err))
	}
	md, err := model.LoadMetadata(dir)
	if err != nil {
		findings = append(findings, errf("parse", "%v", err))
		md = nil
	}
	cassettes, err := model.LoadCassettes(dir)
	if err != nil {
		findings = append(findings, errf("parse", "%v", err))
	}

	findings = append(findings, checkNativeRequiredFields(c)...)
	findings = append(findings, checkSpecVersion(c.SpecVersion)...)
	findings = append(findings, checkNativeClosedEnums(c)...)
	findings = append(findings, checkOperationCassetteCoverage(c.Capabilities.Operations, cassettes)...)
	findings = append(findings, checkNativeGoldenFiles(dir, cassettes)...)
	if md != nil {
		findings = append(findings, checkMetadataAgainstProvider(c.Provider, md)...)
	}

	return findings
}

func checkNativeRequiredFields(c *model.NativeCapabilities) []Finding {
	var f []Finding
	req := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			f = append(f, errf("required-field", "%s is required and must be non-empty", name))
		}
	}
	req("provider", c.Provider)
	req("spec_version", c.SpecVersion)
	req("display_name", c.DisplayName)
	req("country", c.Country)
	if c.Implementation != "native" {
		f = append(f, errf("required-field", "implementation must be exactly \"native\", found %q", c.Implementation))
	}
	if len(c.Currencies) == 0 {
		f = append(f, errf("required-field", "currencies must declare at least one ISO-4217 code"))
	}
	return f
}

func checkNativeClosedEnums(c *model.NativeCapabilities) []Finding {
	var f []Finding
	if !model.CredentialShapes[c.Auth.Shape] {
		f = append(f, errf("closed-enum", "auth.shape %q is not a member of CredentialShape", c.Auth.Shape))
	}
	if c.Auth.Shape == "none" && len(c.Auth.Fields) != 0 {
		f = append(f, errf("auth-shape", "auth.fields must be empty when shape is \"none\""))
	}
	if c.Auth.Shape != "none" && len(c.Auth.Fields) == 0 {
		f = append(f, errf("auth-shape", "auth.fields must declare at least one field when shape is not \"none\""))
	}
	for _, op := range c.Capabilities.Operations {
		if !model.Operations[op] {
			f = append(f, errf("closed-enum", "capabilities.operations contains %q, not a member of Operation", op))
		}
	}
	for _, na := range c.Capabilities.NextActions {
		if !model.NextActions[na] {
			f = append(f, errf("closed-enum", "capabilities.next_actions contains %q, not a member of NextAction", na))
		}
	}
	return f
}

// checkNativeGoldenFiles requires an expected/<name>.json for every
// cassette, and that it is at least well-formed JSON. This reference
// tool has no manifest to replay a native provider's cassettes against,
// so it cannot assert the golden value is correct (that is the native
// implementation's own conformance suite's job), but a missing or
// malformed golden file is still this repository's bug to catch.
func checkNativeGoldenFiles(dir string, cassettes map[string]*model.Cassette) []Finding {
	var f []Finding
	for cassetteFile, cas := range cassettes {
		expectedName := strings.TrimSuffix(cassetteFile, ".yaml") + ".json"
		expectedPath := filepath.Join(dir, "expected", expectedName)
		data, err := os.ReadFile(expectedPath)
		if err != nil {
			f = append(f, errf("missing-golden-file",
				"cassette %q (name: %s) has no matching expected/%s", cassetteFile, cas.Name, expectedName))
			continue
		}
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			f = append(f, errf("malformed-golden-file", "expected/%s is not valid JSON: %v", expectedName, err))
		}
	}
	return f
}
