package replay

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/expr"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
)

// Run executes cas against m and returns the resulting PaymentResult,
// mirroring what a conformant runtime must produce for the same manifest
// and cassette (spec/06-conformance.md).
//
// Scope note: this interpreter handles exactly the shape every manifest
// in this repository currently uses: one call (or one inbound webhook)
// per cassette, dispatched by a single status_map/errors check on that
// one response, followed by a chain of plain emit/goto steps to a
// terminal leaf. A manifest whose flow makes a second call after the
// first (multi-round-trip within one operation) or re-dispatches on
// status_map more than once per run is out of scope for this reference
// implementation and will return an error rather than a wrong answer.
func Run(m *model.Manifest, cas *model.Cassette, exponents map[string]int) (*PaymentResult, error) {
	opEntry, ok := m.Operations[cas.Operation]
	if !ok {
		return nil, fmt.Errorf("manifest does not declare operation %q", cas.Operation)
	}
	flow, ok := m.Flows[opEntry.EntryFlow]
	if !ok {
		return nil, fmt.Errorf("operation %q entry_flow %q not found in flows", cas.Operation, opEntry.EntryFlow)
	}
	entryStep, err := FindEntryStep(flow)
	if err != nil {
		return nil, err
	}

	state := map[string]interface{}{}
	for k, v := range cas.State {
		state[k] = v
	}

	// ns (credentials/ctx/intent/idempotency_key) is built unconditionally,
	// including for a webhook operation: intent is simply empty there
	// (the cassette schema does not require it for webhook), and nothing
	// about buildNamespaces assumes a non-webhook operation.
	ns, err := buildNamespaces(m, cas)
	if err != nil {
		return nil, err
	}

	var status string
	var nextAction map[string]interface{}
	var failure *FailureInfo

	if cas.Operation == "webhook" {
		if cas.InboundWebhook == nil {
			return nil, fmt.Errorf("cassette for a webhook operation must set inbound_webhook")
		}
		var doc interface{}
		if err := json.Unmarshal([]byte(cas.InboundWebhook.Body), &doc); err != nil {
			return nil, fmt.Errorf("parsing inbound_webhook.body: %w", err)
		}
		status, nextAction, failure, err = dispatch(m, flow, entryStep, doc, 0, false, state, ns, exponents)
		if err != nil {
			return nil, err
		}
	} else {
		if len(cas.Interactions) == 0 {
			return nil, fmt.Errorf("cassette has no interactions and is not marked transport_failure")
		}
		interaction := cas.Interactions[0]
		if entry := flow.Steps[entryStep]; entry.Call != nil {
			if err := checkRequestMatch(entry.Call, interaction.Request, ns, state, &m.Auth, exponents); err != nil {
				return nil, fmt.Errorf("request assertion failed: %w", err)
			}
		}

		if cas.TransportFailure {
			// Invariant I4: a transport failure always resolves to
			// Processing + Poll. This is fixed regardless of what the
			// manifest's transport: timeout entry declares (validated
			// separately by checks.ValidateProvider), so replay does not
			// need to consult m.Errors here at all. interval_ms is the
			// fixed runtime-synthesized default (spec/01-domain-model.md
			// #why-interval_ms-is-required-even-for-a-runtime-synthesized-poll),
			// since there is no manifest step to read one from.
			status = "Processing"
			nextAction = syntheticPoll()
			return finish(cas, status, nextAction, failure, state), nil
		}

		if interaction.Response == nil {
			return nil, fmt.Errorf("interaction has no response and cassette is not marked transport_failure")
		}
		var doc interface{}
		if err := json.Unmarshal([]byte(interaction.Response.Body), &doc); err != nil {
			return nil, fmt.Errorf("parsing response body: %w", err)
		}
		status, nextAction, failure, err = dispatch(m, flow, entryStep, doc, interaction.Response.Status, true, state, ns, exponents)
		if err != nil {
			return nil, err
		}
	}

	return finish(cas, status, nextAction, failure, state), nil
}

// syntheticPoll is the NextAction.Poll object a runtime builds itself
// when it has no manifest step to read one from (I4 transport failure;
// the ProviderTimeout/Unknown errors short-circuit in dispatch).
func syntheticPoll() map[string]interface{} {
	return map[string]interface{}{"type": "Poll", "interval_ms": int64(model.DefaultPollIntervalMs)}
}

// finish assembles the final PaymentResult.
func finish(cas *model.Cassette, status string, nextAction map[string]interface{}, failure *FailureInfo, state map[string]interface{}) *PaymentResult {
	result := &PaymentResult{
		IdempotencyKey: cas.Seed.IdempotencyKey,
		Operation:      cas.Operation,
		Status:         status,
		State:          state,
		Failure:        failure,
	}
	if nextAction != nil {
		result.NextAction = nextAction
	}
	return result
}

// dispatch evaluates the entry step's errors/status_map against doc, then
// follows the resulting chain of plain emit/goto steps to a terminal
// leaf, patching state along the way (spec/03-manifest-dsl.md's
// emit.state-is-a-patch rule). ns/exponents are threaded through so a
// step's emit.next_action fields can reference any namespace (extract,
// intent, ctx, state, idempotency_key), not only extract.
func dispatch(m *model.Manifest, flow model.Flow, entryStep string, doc interface{}, httpStatus int, checkErrors bool, state map[string]interface{}, ns map[string]interface{}, exponents map[string]int) (status string, nextAction map[string]interface{}, failure *FailureInfo, err error) {
	entry := flow.Steps[entryStep]

	if checkErrors {
		if match := matchError(m.Errors, doc, httpStatus); match != nil {
			// Per spec/03-manifest-dsl.md, a matched error bypasses this
			// step's own emit entirely: state is left as it was.
			if match.FailureCode == "ProviderTimeout" || match.FailureCode == "Unknown" {
				return "Processing", syntheticPoll(), nil, nil
			}
			return "Failed", nil, &FailureInfo{FailureCode: match.FailureCode, RetryClass: match.RetryClass}, nil
		}
	}

	stepName := entryStep
	if entry.StatusMap != nil {
		if err := applyEmitState(entry.Emit, ns, state, doc, exponents); err != nil {
			return "", nil, nil, err
		}
		val, ok := expr.Extract(doc, entry.StatusMap.Path)
		if !ok {
			return "", nil, nil, fmt.Errorf("status_map.path %q did not resolve against the response", entry.StatusMap.Path)
		}
		key := fmt.Sprintf("%v", val)
		target, ok := entry.StatusMap.Values[key]
		if !ok {
			return "", nil, nil, fmt.Errorf("status_map has no entry for extracted value %q at path %q", key, entry.StatusMap.Path)
		}
		stepName = target
	} else {
		if err := applyEmitState(entry.Emit, ns, state, doc, exponents); err != nil {
			return "", nil, nil, err
		}
		if entry.Emit != nil {
			status = entry.Emit.Status
			if na, err := resolveNextAction(entry.Emit.NextAction, ns, state, doc, exponents); err != nil {
				return "", nil, nil, err
			} else if na != nil {
				nextAction = na
			}
		}
		if entry.Goto == "" {
			return status, nextAction, nil, nil
		}
		stepName = entry.Goto
	}

	for {
		step, ok := flow.Steps[stepName]
		if !ok {
			return "", nil, nil, fmt.Errorf("step %q does not exist", stepName)
		}
		if err := applyEmitState(step.Emit, ns, state, doc, exponents); err != nil {
			return "", nil, nil, err
		}
		if step.Emit != nil {
			if step.Emit.Status != "" {
				status = step.Emit.Status
			}
			if na, err := resolveNextAction(step.Emit.NextAction, ns, state, doc, exponents); err != nil {
				return "", nil, nil, err
			} else if na != nil {
				nextAction = na
			}
		}
		if step.Goto != "" {
			stepName = step.Goto
			continue
		}
		return status, nextAction, nil, nil
	}
}

// matchError finds the first errors[] entry (excluding transport, which
// is handled separately) whose match condition is satisfied by doc/httpStatus.
func matchError(errs []model.ErrorMapping, doc interface{}, httpStatus int) *model.ErrorMapping {
	for i := range errs {
		e := &errs[i]
		switch e.Match.Kind() {
		case "http_status":
			if e.Match.HTTPStatus == httpStatus {
				return e
			}
		case "path":
			val, ok := expr.Extract(doc, e.Match.Path)
			if ok && fmt.Sprintf("%v", val) == fmt.Sprintf("%v", e.Match.Equals) {
				return e
			}
		}
	}
	return nil
}

// applyEmitState patches state with a step's emit.state, resolving each
// value (a whole-value ${...} interpolation from any namespace, or a
// literal) via the same resolver call.body and emit.next_action use; see
// resolveBody.
func applyEmitState(e *model.Emit, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) error {
	if e == nil {
		return nil
	}
	for k, template := range e.State {
		val, err := resolveBody(template, ns, state, doc, exponents)
		if err != nil {
			return fmt.Errorf("resolving emit.state.%s: %w", k, err)
		}
		state[k] = val
	}
	return nil
}

// resolveNextAction resolves a step's raw emit.next_action mapping
// ({"type": "...", ...fields}, as decoded from YAML) into a concrete
// object for the final PaymentResult, evaluating every field value the
// same way call.body and emit.state are resolved (resolveBody), which
// also handles the one level of nesting NextAction.ShowTransferDetails's
// amount field needs. raw is nil when the step's emit does not set
// next_action; resolveNextAction returns (nil, nil) in that case.
func resolveNextAction(raw map[string]interface{}, ns map[string]interface{}, state map[string]interface{}, doc interface{}, exponents map[string]int) (map[string]interface{}, error) {
	if raw == nil {
		return nil, nil
	}
	resolved, err := resolveBody(raw, ns, state, doc, exponents)
	if err != nil {
		return nil, fmt.Errorf("resolving emit.next_action: %w", err)
	}
	out, ok := resolved.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("emit.next_action did not resolve to an object")
	}
	return out, nil
}

// FindEntryStep returns the one step in flow that no other step's
// goto/status_map ever names as a target (spec/03-manifest-dsl.md's
// entry-step rule). It errors if there is not exactly one.
func FindEntryStep(flow model.Flow) (string, error) {
	targeted := map[string]bool{}
	for _, step := range flow.Steps {
		if step.Goto != "" {
			targeted[step.Goto] = true
		}
		if step.StatusMap != nil {
			for _, t := range step.StatusMap.Values {
				targeted[t] = true
			}
		}
	}
	var entries []string
	for name := range flow.Steps {
		if !targeted[name] {
			entries = append(entries, name)
		}
	}
	sort.Strings(entries)
	if len(entries) != 1 {
		return "", fmt.Errorf("flow must have exactly one entry step, found %d: %v", len(entries), entries)
	}
	return entries[0], nil
}
