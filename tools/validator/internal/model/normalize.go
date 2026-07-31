package model

// NormalizeYAMLValue recursively converts any map[interface{}]interface{}
// found within v into map[string]interface{}, leaving everything else
// unchanged.
//
// Why this exists: fields like Call.Body decode through YAML's generic
// interface{} path for nested objects (Body's declared type is
// map[string]interface{}, but a nested object inside it, e.g. ArifPay's
// beneficiaries block, has no more specific declared type to guide the
// decoder). Different versions/configurations of yaml.v3 have decoded
// bare interface{} maps as either map[string]interface{} or
// map[interface{}]interface{} depending on version history; rather than
// assert which this module's pinned version does from memory, every
// caller that walks a YAML-decoded generic value (internal/expr's
// WalkStrings, internal/replay's resolveBody) should run it through this
// function first so both shapes behave identically either way.
func NormalizeYAMLValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			out[k] = NormalizeYAMLValue(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			key, ok := k.(string)
			if !ok {
				key = toString(k)
			}
			out[key] = NormalizeYAMLValue(vv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, vv := range t {
			out[i] = NormalizeYAMLValue(vv)
		}
		return out
	default:
		return v
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "" // non-string map keys never occur in this DSL's YAML
}
