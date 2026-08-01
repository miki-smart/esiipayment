package replay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/expr"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
)

// buildNamespaces assembles the credentials/ctx/intent/idempotency_key
// evaluation context for one cassette replay. state is looked up
// separately at evaluation time since it mutates during a run.
func buildNamespaces(m *model.Manifest, cas *model.Cassette) (map[string]interface{}, error) {
	ns := map[string]interface{}{
		"idempotency_key": cas.Seed.IdempotencyKey,
		"credentials":     stringMapToInterface(cas.Credentials),
		"intent":          mapOrEmpty(cas.Intent),
	}
	ctx := stringMapToInterface(cas.Ctx)
	if cas.Environment != "" {
		env, ok := m.Environments[cas.Environment]
		if !ok {
			return nil, fmt.Errorf("cassette environment %q is not declared in manifest environments", cas.Environment)
		}
		ctx["base_url"] = env.BaseURL
	}
	ns["ctx"] = ctx
	return ns, nil
}

func stringMapToInterface(m map[string]string) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func mapOrEmpty(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

// lookupDotPath navigates a plain dot-separated path (no array indices;
// those only ever occur within extract/event paths, which go through
// expr.Extract instead) against a namespace root value.
func lookupDotPath(root interface{}, path string) (interface{}, bool) {
	if path == "" {
		return root, true
	}
	cur := root
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, present := m[part]
		if !present {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// evaluateInterpolation resolves one ${...} occurrence against
// ns/state/doc, applying its transform if any, and returns the value's
// native type (string, int64, bool, ...) when no transform is applied so
// callers that need a typed result (e.g. a NextAction.ShowTransferDetails
// amount.minor_units, which must stay a JSON integer, never become a
// string) get one. A transform always produces a string, since every
// member of the closed transform set (spec/03-manifest-dsl.md#field-transforms)
// is defined to. amount_major is special-cased to read the sibling
// intent.currency field for its exponent, matching how every manifest in
// this repository actually uses it (amount and currency are always
// separate sibling fields, never combined in one interpolation).
//
// doc is the current step's parsed HTTP response or webhook body, needed
// to resolve the "extract" namespace (spec/04-expression-language.md); it
// is nil when evaluating a call's own path/headers/body, since those are
// built before any response exists.
func evaluateInterpolation(interp expr.Interpolation, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) (interface{}, error) {
	var val interface{}
	switch interp.Namespace {
	case "state":
		v, ok := lookupDotPath(state, interp.Path)
		if !ok {
			return nil, fmt.Errorf("state.%s did not resolve", interp.Path)
		}
		val = v
	case "extract":
		v, ok := expr.Extract(doc, "$."+interp.Path)
		if !ok {
			return nil, fmt.Errorf("extract.%s did not resolve", interp.Path)
		}
		val = v
	default:
		root, ok := ns[interp.Namespace]
		if !ok {
			return nil, fmt.Errorf("namespace %q is not available in this context", interp.Namespace)
		}
		v, ok := lookupDotPath(root, interp.Path)
		if !ok {
			return nil, fmt.Errorf("%s.%s did not resolve", interp.Namespace, interp.Path)
		}
		val = v
	}
	if interp.Transform == "" {
		return val, nil
	}
	exponent := 0
	if interp.Transform == "amount_major" {
		intentMap, _ := ns["intent"].(map[string]interface{})
		currency, _ := intentMap["currency"].(string)
		exp, ok := exponents[currency]
		if !ok {
			return "", fmt.Errorf("no minor-unit exponent known for currency %q (vectors/money/exponents.json)", currency)
		}
		exponent = exp
	}
	return expr.ApplyTransform(interp.Transform, val, exponent)
}

// interpolateString replaces every ${...} occurrence in template with its
// evaluated string form: the general case, used for call.path, which
// embeds interpolation inside a larger literal string
// (e.g. "/v1/transaction/verify/${idempotency_key}").
func interpolateString(template string, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) (string, error) {
	out := template
	for _, interp := range expr.FindInterpolations(template) {
		val, err := evaluateInterpolation(interp, ns, state, doc, exponents)
		if err != nil {
			return "", fmt.Errorf("interpolating %s in %q: %w", interp.Raw, template, err)
		}
		out = strings.Replace(out, interp.Raw, fmt.Sprintf("%v", val), 1)
	}
	return out, nil
}

// resolveBody recursively resolves a call.body/emit.next_action template
// (as decoded from YAML into map[string]interface{}/[]interface{}/scalar
// leaves) into concrete values. Every manifest in this repository uses
// body/header/next_action values as whole-value interpolations (never
// partial-string, unlike call.path), so a string leaf that is entirely
// one ${...} resolves directly to that interpolation's (possibly
// non-string) value; any other string, and any non-string scalar (a
// literal integer like a NextAction.Poll interval_ms), is returned as a
// literal.
func resolveBody(body interface{}, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) (interface{}, error) {
	switch v := body.(type) {
	case map[string]interface{}:
		out := map[string]interface{}{}
		for k, vv := range v {
			r, err := resolveBody(vv, ns, state, doc, exponents)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, vv := range v {
			r, err := resolveBody(vv, ns, state, doc, exponents)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case string:
		interps := expr.FindInterpolations(v)
		if len(interps) == 1 && interps[0].Raw == v {
			return evaluateInterpolation(interps[0], ns, state, doc, exponents)
		}
		return v, nil
	default:
		return v, nil
	}
}

// resolveHeaders resolves a call.headers map the same way resolveBody
// resolves string leaves.
func resolveHeaders(headers map[string]string, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range headers {
		interps := expr.FindInterpolations(v)
		if len(interps) == 1 && interps[0].Raw == v {
			val, err := evaluateInterpolation(interps[0], ns, state, doc, exponents)
			if err != nil {
				return nil, err
			}
			out[k] = fmt.Sprintf("%v", val)
			continue
		}
		resolved, err := interpolateString(v, ns, state, doc, exponents)
		if err != nil {
			return nil, err
		}
		out[k] = resolved
	}
	return out, nil
}

// checkRequestMatch reconstructs step's outbound request from ns/state and
// compares it (logically, not byte-for-byte, since YAML/Go maps do not
// preserve key order the way a hand-written cassette body string does)
// against the cassette's recorded expectation. See CanonicalJSON: both
// sides are canonicalized before comparison so key order never matters.
func checkRequestMatch(call *model.Call, req model.Request, ns map[string]interface{}, state map[string]interface{}, auth *model.Auth, exponents map[string]int) error {
	// doc is nil throughout: a call's own path/headers/body are built
	// before this step has received any response, so the extract
	// namespace cannot resolve here (and no manifest in this repository
	// references it in a call step; see checks.checkInterpolationNamespaces).
	gotPath, err := interpolateString(call.Path, ns, state, nil, exponents)
	if err != nil {
		return fmt.Errorf("resolving call.path: %w", err)
	}
	if gotPath != req.Path {
		return fmt.Errorf("path mismatch: manifest resolves to %q, cassette expects %q", gotPath, req.Path)
	}
	if call.Method != req.Method {
		return fmt.Errorf("method mismatch: manifest declares %q, cassette expects %q", call.Method, req.Method)
	}
	{
		gotHeaders, err := resolveHeaders(call.Headers, ns, state, nil, exponents)
		if err != nil {
			return fmt.Errorf("resolving call.headers: %w", err)
		}
		// auth.apply is attached to every call automatically (this is
		// the whole point: a flow step never repeats it), so it is
		// applied here rather than being part of any step's own
		// call.headers. Only the header target has an observable effect
		// on an outbound request in this reference interpreter's scope;
		// query/body targets are schema/structurally validated
		// (checks.checkAuth) but not yet exercised by any cassette in
		// this repository.
		if auth != nil && auth.Apply != nil && auth.Apply.Header != "" {
			val, err := interpolateString(auth.Apply.Value, ns, state, nil, exponents)
			if err != nil {
				return fmt.Errorf("resolving auth.apply.value: %w", err)
			}
			gotHeaders[auth.Apply.Header] = val
		}
		// Lenient: only headers the cassette itself recorded are
		// checked. A recorded cassette need not enumerate every header
		// the manifest declares (some are incidental to the transport,
		// e.g. content-type), so absence in the cassette is not a
		// mismatch, but a present, differing value is.
		for k, want := range req.Headers {
			if got, ok := gotHeaders[k]; ok && got != want {
				return fmt.Errorf("header %q mismatch: manifest resolves to %q, cassette expects %q", k, got, want)
			}
		}
	}
	if req.Body == "" {
		return nil
	}
	gotBody, err := resolveBody(call.Body, ns, state, nil, exponents)
	if err != nil {
		return fmt.Errorf("resolving call.body: %w", err)
	}
	gotCanonical, err := CanonicalJSON(gotBody)
	if err != nil {
		return fmt.Errorf("canonicalizing resolved body: %w", err)
	}
	var wantParsed interface{}
	if err := json.Unmarshal([]byte(req.Body), &wantParsed); err != nil {
		return fmt.Errorf("cassette request.body is not valid JSON: %w", err)
	}
	wantCanonical, err := CanonicalJSON(wantParsed)
	if err != nil {
		return fmt.Errorf("canonicalizing cassette request.body: %w", err)
	}
	if gotCanonical != wantCanonical {
		return fmt.Errorf("request body mismatch:\n  manifest resolves to: %s\n  cassette expects:     %s", gotCanonical, wantCanonical)
	}
	return nil
}
