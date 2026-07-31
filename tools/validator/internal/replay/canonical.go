package replay

import (
	"bytes"
	"encoding/json"
)

// PaymentResult is the wire shape from spec/01-domain-model.md#paymentresult.
// It is never marshaled directly: ToCanonicalValue converts it to a plain
// map[string]interface{} first, since Go's json.Marshal only sorts map
// keys, not struct field order, and canonical JSON requires every nesting
// level's keys sorted (spec/06-conformance.md).
type PaymentResult struct {
	Failure        *FailureInfo
	IdempotencyKey string
	NextAction     *string
	Operation      string
	State          map[string]interface{}
	Status         string
}

// FailureInfo carries a Failed result's FailureCode/RetryClass pair.
type FailureInfo struct {
	FailureCode string
	RetryClass  string
}

// CanonicalJSON renders v as the canonical JSON defined in
// spec/06-conformance.md: sorted object keys at every level, no
// insignificant whitespace, integers never as floats, no HTML-escaping
// of ordinary characters. v should be built from Go maps/slices/strings/
// int64/bool/nil (e.g. via ToCanonicalValue), not typed structs, so that
// nested object keys are sorted too: Go's json.Marshal already sorts
// map[string]T keys at every nesting level, which is what makes this
// approach sufficient without a hand-rolled recursive key sort.
func CanonicalJSON(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	// enc.Encode appends a trailing newline; canonical JSON has none.
	return string(bytes.TrimRight(buf.Bytes(), "\n")), nil
}

// ToCanonicalValue converts a PaymentResult into a plain
// map[string]interface{} tree so CanonicalJSON's map-key sorting applies
// uniformly, including to the nested State map.
func (r PaymentResult) ToCanonicalValue() map[string]interface{} {
	out := map[string]interface{}{
		"idempotency_key": r.IdempotencyKey,
		"operation":       r.Operation,
		"state":           mapToInterface(r.State),
		"status":          r.Status,
	}
	if r.NextAction != nil {
		out["next_action"] = *r.NextAction
	} else {
		out["next_action"] = nil
	}
	if r.Failure != nil {
		out["failure"] = map[string]interface{}{
			"failure_code": r.Failure.FailureCode,
			"retry_class":  r.Failure.RetryClass,
		}
	} else {
		out["failure"] = nil
	}
	return out
}

func mapToInterface(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}
