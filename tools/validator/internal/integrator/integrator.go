// Package integrator is the reference integrator program A5/GOVERNANCE's
// "integrator promise" describes: one generic handler over PaymentResult
// that expresses every outcome purely in terms of PaymentStatus,
// NextAction, FailureCode, and RetryClass, per Invariant I12
// (spec/02-invariants.md#i12). It is run unmodified against every
// provider's cassette replay output; the promise it checks is exactly
// "adding a provider never requires touching integrator code."
//
// This is the machine-checkable form of the same reference integrator
// described, in language-neutral pseudocode, at
// docs/examples/integrator/ (spec/03-manifest-dsl.md's Part B tutorials
// derive from that pseudocode; this file is its Go embodiment for this
// repository's own CI).
package integrator

import (
	"fmt"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/replay"
)

// Outcome is what the reference integrator decided to do with one
// PaymentResult: purely descriptive, used only to report what happened in
// `esiipayment conformance-integrator`'s output.
type Outcome struct {
	// Terminal is true if pr.Status is one of the four terminal
	// PaymentStatus values (spec/01-domain-model.md#paymentstatus).
	Terminal bool
	// Description is a short human-readable summary of what the
	// reference integrator's generic switch did with this result, e.g.
	// "terminal: Succeeded" or "show redirect to https://...".
	Description string
}

// Handle is the reference integrator's entire decision surface: an
// exhaustive switch over the six closed PaymentStatus values and, for a
// non-terminal one, the nine closed NextAction variants. It never reads
// pr.State, never branches on which provider produced pr, and never
// falls through to a default case that would silently accept an
// unrecognized value — an unhandled status or next_action type is
// exactly the promise violation this program exists to catch, so it is
// reported as an error, not swallowed.
func Handle(pr *replay.PaymentResult) (Outcome, error) {
	switch pr.Status {
	case "Succeeded":
		return Outcome{Terminal: true, Description: "terminal: Succeeded — mark the order paid, fulfill it"}, nil
	case "Failed":
		if pr.Failure == nil {
			return Outcome{}, fmt.Errorf("status Failed must carry a non-null failure (failure_code/retry_class), got none")
		}
		if !model.RetryClasses[pr.Failure.RetryClass] {
			return Outcome{}, fmt.Errorf("failure.retry_class %q is not a member of the closed RetryClass set", pr.Failure.RetryClass)
		}
		return Outcome{Terminal: true, Description: fmt.Sprintf("terminal: Failed (%s, %s) — surface the decline, do not retry a DoNotRetry/ResolveFirst failure automatically", pr.Failure.FailureCode, pr.Failure.RetryClass)}, nil
	case "Canceled":
		return Outcome{Terminal: true, Description: "terminal: Canceled — release any held inventory, let the payer retry as a new payment"}, nil
	case "Expired":
		return Outcome{Terminal: true, Description: "terminal: Expired — same handling as Canceled: a new payment, never a retry of this one"}, nil
	case "RequiresAction", "Processing":
		desc, err := handleNextAction(pr.NextAction)
		if err != nil {
			return Outcome{}, err
		}
		return Outcome{Terminal: false, Description: desc}, nil
	default:
		return Outcome{}, fmt.Errorf("status %q is not a member of the closed PaymentStatus set: this is exactly the promise violation this program exists to catch", pr.Status)
	}
}

// handleNextAction is the nine-branch switch Invariant I3 promises never
// needs a tenth case: every field it reads is one of NextAction's own
// closed, per-variant fields (spec/01-domain-model.md#nextaction-carries-its-own-payload),
// never a key from pr.State.
func handleNextAction(na map[string]interface{}) (string, error) {
	if na == nil {
		return "", fmt.Errorf("a non-terminal status must carry a non-null next_action")
	}
	t, _ := na["type"].(string)
	switch t {
	case "None":
		return "wait: nothing further required of the user yet", nil
	case "RedirectToUrl":
		url, ok := na["url"].(string)
		if !ok || url == "" {
			return "", fmt.Errorf("RedirectToUrl requires a non-empty url field")
		}
		return "redirect the payer to " + url, nil
	case "AwaitDevicePush":
		ref, ok := na["display_ref"].(string)
		if !ok || ref == "" {
			return "", fmt.Errorf("AwaitDevicePush requires a non-empty display_ref field")
		}
		return "tell the payer to approve the push sent to " + ref, nil
	case "SubmitOtp":
		if _, ok := numeric(na["length"]); !ok {
			return "", fmt.Errorf("SubmitOtp requires a numeric length field")
		}
		return "prompt the payer for an OTP", nil
	case "DisplayQr":
		payload, ok := na["payload"].(string)
		if !ok || payload == "" {
			return "", fmt.Errorf("DisplayQr requires a non-empty payload field")
		}
		return "render a QR code for " + payload, nil
	case "ShowTransferDetails":
		for _, field := range []string{"account_number", "institution", "reference"} {
			if s, ok := na[field].(string); !ok || s == "" {
				return "", fmt.Errorf("ShowTransferDetails requires a non-empty %s field", field)
			}
		}
		if _, ok := na["amount"].(map[string]interface{}); !ok {
			return "", fmt.Errorf("ShowTransferDetails requires an amount object")
		}
		return "show the payer manual transfer details", nil
	case "DialUssd":
		code, ok := na["code"].(string)
		if !ok || code == "" {
			return "", fmt.Errorf("DialUssd requires a non-empty code field")
		}
		return "tell the payer to dial " + code, nil
	case "Poll":
		if _, ok := numeric(na["interval_ms"]); !ok {
			return "", fmt.Errorf("Poll requires a numeric interval_ms field")
		}
		return "schedule a sync retry after interval_ms", nil
	case "Capture":
		return "invoke the capture operation to finalize funds already authorized", nil
	default:
		return "", fmt.Errorf("next_action.type %q is not a member of the closed NextAction set: this is exactly the promise violation this program exists to catch", t)
	}
}

func numeric(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
