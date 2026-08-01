package model

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// decodeStrict decodes YAML into out, rejecting any key not present in
// out's struct fields. This is what enforces additionalProperties: false
// for the DSL's own named-struct vocabulary; map-typed fields (call.body,
// status_map.values, cassette intent/ctx/credentials, etc.) still accept
// arbitrary keys, which is the deliberate pass-through exception
// documented in spec/03-manifest-dsl.md.
func decodeStrict(data []byte, out interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

// FirstLine returns the first line of a file, used to check for the
// required "# yaml-language-server: $schema=..." header.
func FirstLine(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(data)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimRight(s[:idx], "\r"), nil
	}
	return strings.TrimRight(s, "\r"), nil
}

// LoadManifest reads and strictly decodes providers/<name>/manifest.yaml.
func LoadManifest(providerDir string) (*Manifest, error) {
	path := filepath.Join(providerDir, "manifest.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m Manifest
	if err := decodeStrict(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if m.Auth.Token != nil && m.Auth.Token.Body != nil {
		m.Auth.Token.Body, _ = NormalizeYAMLValue(m.Auth.Token.Body).(map[string]interface{})
	}
	for flowName, flow := range m.Flows {
		for stepName, step := range flow.Steps {
			if step.Call != nil && step.Call.Body != nil {
				step.Call.Body, _ = NormalizeYAMLValue(step.Call.Body).(map[string]interface{})
			}
			if step.Emit != nil && step.Emit.NextAction != nil {
				step.Emit.NextAction, _ = NormalizeYAMLValue(step.Emit.NextAction).(map[string]interface{})
			}
			flow.Steps[stepName] = step
		}
		m.Flows[flowName] = flow
	}
	return &m, nil
}

// IsNativeProvider reports whether providerDir is a native-implementation
// provider (capabilities.yaml present) rather than a manifest.yaml-driven
// one. See spec/03-manifest-dsl.md#native-providers.
func IsNativeProvider(providerDir string) bool {
	_, err := os.Stat(filepath.Join(providerDir, "capabilities.yaml"))
	return err == nil
}

// LoadCapabilities reads and strictly decodes
// providers/<name>/capabilities.yaml.
func LoadCapabilities(providerDir string) (*NativeCapabilities, error) {
	path := filepath.Join(providerDir, "capabilities.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var c NativeCapabilities
	if err := decodeStrict(data, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &c, nil
}

// LoadMetadata reads and strictly decodes providers/<name>/metadata.yaml.
func LoadMetadata(providerDir string) (*Metadata, error) {
	path := filepath.Join(providerDir, "metadata.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var md Metadata
	if err := decodeStrict(data, &md); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &md, nil
}

// LoadCassettes reads and strictly decodes every *.yaml file under
// providers/<name>/cassettes/.
func LoadCassettes(providerDir string) (map[string]*Cassette, error) {
	dir := filepath.Join(providerDir, "cassettes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*Cassette{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	out := map[string]*Cassette{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var c Cassette
		if err := decodeStrict(data, &c); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		if c.Intent != nil {
			c.Intent, _ = NormalizeYAMLValue(c.Intent).(map[string]interface{})
		}
		if c.State != nil {
			c.State, _ = NormalizeYAMLValue(c.State).(map[string]interface{})
		}
		out[e.Name()] = &c
	}
	return out, nil
}
