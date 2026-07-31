// Package expr implements just enough of the expression language from
// spec/04-expression-language.md for the validator's static checks and
// the replay interpreter: finding ${...} interpolations in manifest
// strings, evaluating JSONPath-subset extraction against a parsed JSON
// document, and evaluating the closed transform set.
package expr

import (
	"encoding/base64"
	"encoding/hex"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// interpolationRe matches ${transform(namespace.path)} or bare
// ${namespace.path}. These are two distinct alternatives, not one
// pattern with an optional group, because a bare target may itself
// contain dots (${extract.data.checkout_url}) while a transform name
// never does: a single identifier class shared between "the transform
// name" and "the whole bare target" can't express that. Alternative 1
// captures (transform, target) into groups 1 and 2; alternative 2
// captures the bare target into group 3.
var interpolationRe = regexp.MustCompile(
	`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\(([a-zA-Z_][a-zA-Z0-9_.]*)\)\}|\$\{([a-zA-Z_][a-zA-Z0-9_.]*)\}`)

// Interpolation is one ${...} occurrence found in a manifest string.
type Interpolation struct {
	Raw       string // the full ${...} text
	Transform string // "" if this is a bare ${namespace.path}
	Namespace string // e.g. "intent", "extract", "credentials"
	Path      string // the path after the namespace, e.g. "amount" for "intent.amount"
}

// FindInterpolations returns every ${...} occurrence in s.
func FindInterpolations(s string) []Interpolation {
	matches := interpolationRe.FindAllStringSubmatch(s, -1)
	var out []Interpolation
	for _, m := range matches {
		raw := m[0]
		var transform, target string
		if m[1] != "" {
			transform = m[1]
			target = m[2]
		} else {
			target = m[3]
		}
		namespace := target
		path := ""
		if idx := strings.IndexByte(target, '.'); idx >= 0 {
			namespace = target[:idx]
			path = target[idx+1:]
		}
		out = append(out, Interpolation{Raw: raw, Transform: transform, Namespace: namespace, Path: path})
	}
	return out
}

// WalkStrings recursively visits every string leaf in v, which may be a
// map[string]interface{}, a []interface{}, or a scalar: the shape
// call.body and similar pass-through fields decode into.
func WalkStrings(v interface{}, fn func(string)) {
	switch t := v.(type) {
	case string:
		fn(t)
	case map[string]interface{}:
		for _, vv := range t {
			WalkStrings(vv, fn)
		}
	case []interface{}:
		for _, vv := range t {
			WalkStrings(vv, fn)
		}
	}
}

// Extract evaluates a JSONPath-subset path (spec/04-expression-language.md)
// against a parsed JSON document (the result of encoding/json.Unmarshal
// into interface{}). ok is false if the path does not resolve.
func Extract(doc interface{}, path string) (value interface{}, ok bool) {
	if !strings.HasPrefix(path, "$") {
		return nil, false
	}
	rest := path[1:]
	cur := doc
	for len(rest) > 0 {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			end := 0
			for end < len(rest) && (isIdentByte(rest[end])) {
				end++
			}
			if end == 0 {
				return nil, false
			}
			field := rest[:end]
			rest = rest[end:]
			m, isMap := cur.(map[string]interface{})
			if !isMap {
				return nil, false
			}
			v, present := m[field]
			if !present {
				return nil, false
			}
			cur = v
		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, false
			}
			idxStr := rest[1:end]
			rest = rest[end+1:]
			arr, isArr := cur.([]interface{})
			if !isArr {
				return nil, false
			}
			if idxStr == "*" {
				// Wildcard projection: continue only if this is the last
				// segment (spec/04's supported grammar is $.a[*].b, i.e.
				// the wildcard is followed by exactly one more field).
				if !strings.HasPrefix(rest, ".") {
					return nil, false
				}
				rest = rest[1:]
				end2 := 0
				for end2 < len(rest) && isIdentByte(rest[end2]) {
					end2++
				}
				field := rest[:end2]
				var results []interface{}
				for _, el := range arr {
					m, isMap := el.(map[string]interface{})
					if !isMap {
						return nil, false
					}
					v, present := m[field]
					if !present {
						return nil, false
					}
					results = append(results, v)
				}
				return results, true
			}
			idx, err := strconv.Atoi(idxStr)
			if err != nil || idx < 0 || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func isIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// ApplyTransform applies one of the closed transforms from
// spec/03-manifest-dsl.md#field-transforms to a value already extracted
// or interpolated as a plain string/number.
func ApplyTransform(name string, value interface{}, exponent int) (string, error) {
	switch name {
	case "amount_major":
		n, err := toInt64(value)
		if err != nil {
			return "", err
		}
		return formatMinorUnits(n, exponent), nil
	case "msisdn_et":
		s := fmt.Sprintf("%v", value)
		return normalizeMsisdnEt(s), nil
	case "base64":
		return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", value))), nil
	case "hex":
		return hex.EncodeToString([]byte(fmt.Sprintf("%v", value))), nil
	case "sha256_hex":
		sum := sha256.Sum256([]byte(fmt.Sprintf("%v", value)))
		return hex.EncodeToString(sum[:]), nil
	case "upper":
		return strings.ToUpper(fmt.Sprintf("%v", value)), nil
	case "lower":
		return strings.ToLower(fmt.Sprintf("%v", value)), nil
	case "iso8601":
		return toISO8601(value)
	default:
		return "", fmt.Errorf("unknown transform %q", name)
	}
}

func toInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %v to integer", value)
	}
}

// formatMinorUnits renders an integer minor-unit amount as a major-unit
// decimal string with exactly `exponent` fractional digits (0 means no
// decimal point at all), matching vectors/money/conversions.json.
func formatMinorUnits(minorUnits int64, exponent int) string {
	if exponent == 0 {
		return strconv.FormatInt(minorUnits, 10)
	}
	neg := minorUnits < 0
	n := minorUnits
	if neg {
		n = -n
	}
	div := int64(1)
	for i := 0; i < exponent; i++ {
		div *= 10
	}
	whole := n / div
	frac := n % div
	fracStr := strconv.FormatInt(frac, 10)
	for len(fracStr) < exponent {
		fracStr = "0" + fracStr
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%s", sign, whole, fracStr)
}

// normalizeMsisdnEt normalizes an Ethiopian phone number to 251XXXXXXXXX,
// per the vectors in vectors/expressions/interpolation.json.
func normalizeMsisdnEt(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == '+' {
			continue
		}
	}
	digits := b.String()
	switch {
	case strings.HasPrefix(digits, "251") && len(digits) == 12:
		return digits
	case strings.HasPrefix(digits, "0") && len(digits) == 10:
		return "251" + digits[1:]
	default:
		return digits
	}
}

func toISO8601(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		// Already a timestamp string; assume it is well-formed and
		// re-emit at second precision with a literal Z, per
		// spec/06-conformance.md's canonical timestamp rule.
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return "", err
		}
		return t.UTC().Format("2006-01-02T15:04:05Z"), nil
	case int64:
		return time.Unix(v, 0).UTC().Format("2006-01-02T15:04:05Z"), nil
	case int:
		return time.Unix(int64(v), 0).UTC().Format("2006-01-02T15:04:05Z"), nil
	case float64:
		return time.Unix(int64(v), 0).UTC().Format("2006-01-02T15:04:05Z"), nil
	default:
		return "", fmt.Errorf("cannot convert %v to a timestamp", value)
	}
}
