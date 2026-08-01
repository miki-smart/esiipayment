package checks

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/expr"
	"github.com/esiipayment/esiipayment-spec/tools/validator/internal/model"
)

const requiredSchemaHeader = "# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json"

// ValidateProvider runs every semantic check spec/03-manifest-dsl.md and
// spec/06-conformance.md describe as "esiipayment validate" responsibilities
// against providers/<name>/, dispatching to the DSL (manifest.yaml) or
// native (capabilities.yaml) path depending on which is present. See
// spec/03-manifest-dsl.md#native-providers.
func ValidateProvider(dir string) []Finding {
	if model.IsNativeProvider(dir) {
		return ValidateNativeProvider(dir)
	}
	return validateManifestProvider(dir)
}

// validateManifestProvider is ValidateProvider's DSL (manifest.yaml) path.
// It assumes the manifest/metadata/cassette YAML already parsed
// (schema-shape and additionalProperties: false are enforced during
// loading, via strict decoding: see internal/model/load.go).
func validateManifestProvider(dir string) []Finding {
	var findings []Finding

	header, err := model.FirstLine(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		return append(findings, errf("io", "reading manifest.yaml: %v", err))
	}
	if header != requiredSchemaHeader {
		findings = append(findings, errf("schema-header",
			"manifest.yaml must begin with %q, found %q", requiredSchemaHeader, header))
	}

	m, err := model.LoadManifest(dir)
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

	findings = append(findings, checkRequiredFields(m)...)
	findings = append(findings, checkSpecVersion(m.SpecVersion)...)
	findings = append(findings, checkClosedEnums(m)...)
	findings = append(findings, checkAuth(m)...)
	findings = append(findings, checkOperationsAndCapabilities(m)...)
	findings = append(findings, checkFlows(m)...)
	findings = append(findings, checkNextActionPayloads(m)...)
	findings = append(findings, checkErrors(m)...)
	findings = append(findings, checkWebhook(m)...)
	findings = append(findings, checkInterpolationNamespaces(m)...)
	findings = append(findings, checkAuthNamespaces(m)...)
	findings = append(findings, checkTransforms(m)...)
	findings = append(findings, checkCassetteCoverage(m, cassettes)...)
	if md != nil {
		findings = append(findings, checkMetadata(m, md)...)
	}

	return findings
}

func checkRequiredFields(m *model.Manifest) []Finding {
	var f []Finding
	req := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			f = append(f, errf("required-field", "%s is required and must be non-empty", name))
		}
	}
	req("provider", m.Provider)
	req("spec_version", m.SpecVersion)
	req("display_name", m.DisplayName)
	req("country", m.Country)
	if len(m.Currencies) == 0 {
		f = append(f, errf("required-field", "currencies must declare at least one ISO-4217 code"))
	}
	if len(m.Environments) == 0 {
		f = append(f, errf("required-field", "environments must declare at least one entry"))
	}
	if len(m.Operations) == 0 {
		f = append(f, errf("required-field", "operations must declare at least one entry"))
	}
	if len(m.Flows) == 0 {
		f = append(f, errf("required-field", "flows must declare at least one entry"))
	}
	if len(m.Errors) == 0 {
		f = append(f, errf("required-field", "errors must declare at least one entry (at minimum, the transport: timeout mapping)"))
	}
	return f
}

// checkSpecVersion rejects a manifest (or a native provider's
// capabilities.yaml) declaring a spec_version this reference tool does
// not implement, per spec/08-versioning.md: a runtime must refuse to
// execute a manifest whose spec_version it does not implement, with a
// clear error, rather than a best-effort interpretation.
func checkSpecVersion(specVersion string) []Finding {
	if specVersion == "" {
		// checkRequiredFields (or its native-provider equivalent) already
		// reports the empty case; avoid a duplicate finding here.
		return nil
	}
	if specVersion != model.SupportedSpecVersion {
		return []Finding{errf("unsupported-spec-version",
			"spec_version %q is not implemented by this reference tool (only %q is); see spec/08-versioning.md", specVersion, model.SupportedSpecVersion)}
	}
	return nil
}

func checkClosedEnums(m *model.Manifest) []Finding {
	var f []Finding
	if !model.CredentialShapes[m.Auth.Shape] {
		f = append(f, errf("closed-enum", "auth.shape %q is not a member of CredentialShape", m.Auth.Shape))
	}
	if m.Auth.Shape == "none" && len(m.Auth.Fields) != 0 {
		f = append(f, errf("auth-shape", "auth.fields must be empty when shape is \"none\""))
	}
	if m.Auth.Shape != "none" && len(m.Auth.Fields) == 0 {
		f = append(f, errf("auth-shape", "auth.fields must declare at least one field when shape is not \"none\""))
	}
	for _, op := range m.Capabilities.Operations {
		if !model.Operations[op] {
			f = append(f, errf("closed-enum", "capabilities.operations contains %q, not a member of Operation", op))
		}
	}
	for _, na := range m.Capabilities.NextActions {
		if !model.NextActions[na] {
			f = append(f, errf("closed-enum", "capabilities.next_actions contains %q, not a member of NextAction", na))
		}
	}
	return f
}

// checkAuth validates auth.apply and auth.token: apply is required for
// every shape except "none" and must name exactly one of header/query/
// body; token is required if, and only if, shape is
// oauth2_client_credentials. See spec/03-manifest-dsl.md#auth.
func checkAuth(m *model.Manifest) []Finding {
	var f []Finding
	a := m.Auth

	if a.Shape == "none" {
		if a.Apply != nil {
			f = append(f, errf("auth-shape", "auth.apply must not be set when shape is \"none\""))
		}
	} else if a.Apply == nil {
		f = append(f, errf("auth-shape", "auth.apply is required when shape is not \"none\""))
	} else {
		kind, name := a.Apply.Target()
		if kind == "" {
			f = append(f, errf("malformed-auth-apply", "auth.apply must set exactly one of header, query, or body"))
		} else if name == "" {
			f = append(f, errf("malformed-auth-apply", "auth.apply.%s must not be empty", kind))
		}
		if strings.TrimSpace(a.Apply.Value) == "" {
			f = append(f, errf("required-field", "auth.apply.value is required and must be non-empty"))
		}
	}

	if a.Shape == "oauth2_client_credentials" {
		if a.Token == nil {
			f = append(f, errf("auth-shape", "auth.token is required when shape is oauth2_client_credentials"))
		} else {
			if !model.HTTPMethods[a.Token.Call.Method] {
				f = append(f, errf("closed-enum", "auth.token.call.method %q is not a recognized HTTP method", a.Token.Call.Method))
			}
			if a.Token.RefreshAt <= 0 || a.Token.RefreshAt > 1 {
				f = append(f, errf("range", "auth.token.refresh_at must be > 0 and <= 1, found %v", a.Token.RefreshAt))
			}
		}
	} else if a.Token != nil {
		f = append(f, errf("auth-shape", "auth.token is only meaningful when shape is oauth2_client_credentials"))
	}

	return f
}

func checkOperationsAndCapabilities(m *model.Manifest) []Finding {
	var f []Finding

	opsDeclared := map[string]bool{}
	for _, op := range m.Capabilities.Operations {
		opsDeclared[op] = true
	}
	opsKeyed := map[string]bool{}
	for op := range m.Operations {
		opsKeyed[op] = true
	}
	if !sameStringSet(opsDeclared, opsKeyed) {
		f = append(f, errf("operations-mismatch",
			"capabilities.operations must equal exactly the key set of operations: capabilities has %v, operations has %v",
			sortedKeys(opsDeclared), sortedKeys(opsKeyed)))
	}

	for op, entry := range m.Operations {
		if _, ok := m.Flows[entry.EntryFlow]; !ok {
			f = append(f, errf("dangling-entry-flow",
				"operations.%s.entry_flow %q does not name a flow declared under flows", op, entry.EntryFlow))
		}
	}

	// capabilities.next_actions must equal exactly the union of
	// next_action.type values every flow's steps actually emit.
	emitted := map[string]bool{}
	for _, flow := range m.Flows {
		for _, step := range flow.Steps {
			if step.Emit == nil {
				continue
			}
			if t := step.Emit.NextActionType(); t != "" {
				emitted[t] = true
			}
		}
	}
	declared := map[string]bool{}
	for _, na := range m.Capabilities.NextActions {
		declared[na] = true
	}
	if !sameStringSet(declared, emitted) {
		f = append(f, errf("next-actions-mismatch",
			"capabilities.next_actions must equal exactly the NextAction values flows emit: declared %v, emitted %v",
			sortedKeys(declared), sortedKeys(emitted)))
	}

	return f
}

func checkFlows(m *model.Manifest) []Finding {
	var f []Finding
	for flowName, flow := range m.Flows {
		if len(flow.Steps) == 0 {
			f = append(f, errf("empty-flow", "flow %q declares no steps", flowName))
			continue
		}

		targeted := map[string]bool{}
		for stepName, step := range flow.Steps {
			if step.Goto != "" {
				if _, ok := flow.Steps[step.Goto]; !ok {
					f = append(f, errf("dangling-goto",
						"flow %q step %q: goto %q does not name a step in this flow", flowName, stepName, step.Goto))
				}
				targeted[step.Goto] = true
			}
			if step.StatusMap != nil {
				for _, target := range step.StatusMap.Values {
					if _, ok := flow.Steps[target]; !ok {
						f = append(f, errf("dangling-status-map-target",
							"flow %q step %q: status_map targets %q, not a step in this flow", flowName, stepName, target))
					}
					targeted[target] = true
				}
			}
			if step.Emit != nil && step.Emit.Status != "" && !model.PaymentStatuses[step.Emit.Status] {
				f = append(f, errf("closed-enum",
					"flow %q step %q: emit.status %q is not a member of PaymentStatus", flowName, stepName, step.Emit.Status))
			}
			if step.Emit != nil && step.Emit.NextAction != nil {
				if t := step.Emit.NextActionType(); t == "" {
					f = append(f, errf("malformed-next-action",
						"flow %q step %q: emit.next_action must have a string \"type\" field", flowName, stepName))
				} else if !model.NextActions[t] {
					f = append(f, errf("closed-enum",
						"flow %q step %q: emit.next_action.type %q is not a member of NextAction", flowName, stepName, t))
				}
			}
			// A step emitting a terminal status must not also declare a
			// further transition: see Invariant I2.
			if step.Emit != nil && model.TerminalStatuses[step.Emit.Status] {
				if step.Goto != "" || step.StatusMap != nil {
					f = append(f, errf("terminal-transitions",
						"flow %q step %q: emits terminal status %q but also declares goto/status_map; a terminal step must not transition further (Invariant I2)",
						flowName, stepName, step.Emit.Status))
				}
			}
			for _, tr := range step.Triggers {
				if !model.Triggers[tr] {
					f = append(f, errf("closed-enum", "flow %q step %q: triggers contains %q, not a recognized trigger", flowName, stepName, tr))
				}
			}
			if step.Call != nil && !model.HTTPMethods[step.Call.Method] {
				f = append(f, errf("closed-enum", "flow %q step %q: call.method %q is not a recognized HTTP method", flowName, stepName, step.Call.Method))
			}
		}

		// Exactly one entry step: the one step nobody targets.
		var entries []string
		for stepName := range flow.Steps {
			if !targeted[stepName] {
				entries = append(entries, stepName)
			}
		}
		sort.Strings(entries)
		if len(entries) == 0 {
			f = append(f, errf("no-entry-step",
				"flow %q: every step is targeted by another step's goto/status_map; there is no entry step", flowName))
		} else if len(entries) > 1 {
			f = append(f, errf("ambiguous-entry-step",
				"flow %q: more than one step is never targeted (%v); a flow must have exactly one entry step", flowName, entries))
		}

		// Reachability: every step should be reachable from the entry
		// step by following goto/status_map edges.
		if len(entries) == 1 {
			reachable := map[string]bool{entries[0]: true}
			queue := []string{entries[0]}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				step := flow.Steps[cur]
				var next []string
				if step.Goto != "" {
					next = append(next, step.Goto)
				}
				if step.StatusMap != nil {
					for _, t := range step.StatusMap.Values {
						next = append(next, t)
					}
				}
				for _, n := range next {
					if !reachable[n] {
						reachable[n] = true
						queue = append(queue, n)
					}
				}
			}
			for stepName := range flow.Steps {
				if !reachable[stepName] {
					f = append(f, errf("unreachable-step", "flow %q step %q is not reachable from entry step %q", flowName, stepName, entries[0]))
				}
			}
		}
	}
	return f
}

func checkErrors(m *model.Manifest) []Finding {
	var f []Finding
	timeoutCount := 0
	for i, em := range m.Errors {
		switch em.Match.Kind() {
		case "transport":
			if em.Match.Transport != "timeout" {
				f = append(f, errf("closed-enum", "errors[%d].match.transport %q is not a recognized transport match (only \"timeout\" is defined)", i, em.Match.Transport))
				break
			}
			timeoutCount++
			if em.FailureCode != "ProviderTimeout" || em.RetryClass != "ResolveFirst" {
				f = append(f, errf("timeout-mapping-fixed",
					"errors[%d]: the transport: timeout entry must map to ProviderTimeout / ResolveFirst, found %s / %s (Invariant I4)",
					i, em.FailureCode, em.RetryClass))
			}
		case "http_status":
			if em.Match.HTTPStatus < 100 || em.Match.HTTPStatus > 599 {
				f = append(f, errf("range", "errors[%d].match.http_status %d is not a valid HTTP status code", i, em.Match.HTTPStatus))
			}
		case "path":
			// path/equals shape is structurally checked by the strict
			// decoder already; nothing further to check statically
			// without a cassette to evaluate the path against.
		default:
			f = append(f, errf("malformed-match", "errors[%d].match must be exactly one of path/equals, http_status, or transport", i))
		}

		if !model.FailureCodes[em.FailureCode] {
			f = append(f, errf("closed-enum", "errors[%d].failure_code %q is not a member of FailureCode", i, em.FailureCode))
			continue
		}
		if !model.RetryClasses[em.RetryClass] {
			f = append(f, errf("closed-enum", "errors[%d].retry_class %q is not a member of RetryClass", i, em.RetryClass))
			continue
		}
		if want := model.FailureRetryMap[em.FailureCode]; want != em.RetryClass {
			f = append(f, errf("retry-class-mismatch",
				"errors[%d]: failure_code %s must map to retry_class %s per the fixed table (vectors/errors/failure-retry-map.json), found %s (Invariant I10)",
				i, em.FailureCode, want, em.RetryClass))
		}
	}
	if timeoutCount == 0 {
		f = append(f, errf("missing-timeout-mapping", "errors must include exactly one entry with match.transport: timeout"))
	} else if timeoutCount > 1 {
		f = append(f, errf("duplicate-timeout-mapping", "errors declares %d entries with match.transport: timeout, expected exactly one", timeoutCount))
	}

	// Ambiguity check: no two entries should be able to match the same
	// response: approximated here by flagging exact duplicate (path,
	// equals) pairs and duplicate http_status values, which is the
	// mechanically checkable subset of the rule in
	// spec/03-manifest-dsl.md#how-errors-interacts-with-status_map.
	seenHTTPStatus := map[int]bool{}
	seenPathEquals := map[string]bool{}
	for i, em := range m.Errors {
		switch em.Match.Kind() {
		case "http_status":
			if seenHTTPStatus[em.Match.HTTPStatus] {
				f = append(f, errf("ambiguous-error-match", "errors[%d]: http_status %d is matched by more than one entry", i, em.Match.HTTPStatus))
			}
			seenHTTPStatus[em.Match.HTTPStatus] = true
		case "path":
			key := fmt.Sprintf("%s==%v", em.Match.Path, em.Match.Equals)
			if seenPathEquals[key] {
				f = append(f, errf("ambiguous-error-match", "errors[%d]: %s is matched by more than one entry", i, key))
			}
			seenPathEquals[key] = true
		}
	}

	return f
}

func checkWebhook(m *model.Manifest) []Finding {
	var f []Finding
	needsWebhook := false
	for _, op := range m.Capabilities.Operations {
		if op == "webhook" {
			needsWebhook = true
		}
	}
	if needsWebhook && m.Webhook == nil {
		f = append(f, errf("missing-webhook-section", "capabilities.operations includes \"webhook\" but no top-level webhook section is declared"))
		return f
	}
	if m.Webhook == nil {
		return f
	}
	if !model.WebhookSchemes[m.Webhook.Verification.Scheme] {
		f = append(f, errf("closed-enum", "webhook.verification.scheme %q is not a recognized scheme", m.Webhook.Verification.Scheme))
	}
	if m.Webhook.Verification.Scheme != "none" {
		if m.Webhook.Verification.SignatureHeader == "" {
			f = append(f, errf("required-field", "webhook.verification.signature_header is required unless scheme is \"none\""))
		}
		if m.Webhook.Verification.Secret == "" {
			f = append(f, errf("required-field", "webhook.verification.secret is required unless scheme is \"none\""))
		}
	}
	if m.Webhook.BodyFormat != "json" {
		f = append(f, errf("closed-enum", "webhook.body_format %q is not recognized (only \"json\" is defined)", m.Webhook.BodyFormat))
	}
	return f
}

// checkInterpolationNamespaces implements the position-dependent
// availability rules from spec/04-expression-language.md: `extract` and
// `event` are only meaningful once this step's own response/webhook body
// exists; `input` only inside a step reached via triggers: [input].
func checkInterpolationNamespaces(m *model.Manifest) []Finding {
	var f []Finding
	for flowName, flow := range m.Flows {
		// extract/event availability is flow-wide, not per-step: this
		// reference interpreter (and every conformant runtime) threads
		// one operation's response/webhook body through every step in
		// the chain that handles it, not only the step whose own call or
		// webhook trigger produced it (see spec/03-manifest-dsl.md#flows).
		// So a step reached purely by goto/status_map, with no call of
		// its own, may still read ${extract...}/${event...} from the
		// entry step's response once that entry step actually is a call
		// or webhook trigger.
		flowHasResponse, flowHasEvent := false, false
		if entryName, ok := entryStepOf(flow); ok {
			entry := flow.Steps[entryName]
			flowHasResponse = entry.Call != nil || containsTrigger(entry.Triggers, "webhook")
			flowHasEvent = containsTrigger(entry.Triggers, "webhook")
		}

		for stepName, step := range flow.Steps {
			hasInput := containsTrigger(step.Triggers, "input")

			var strings_ []string
			if step.Call != nil {
				strings_ = append(strings_, step.Call.Path)
				for _, v := range step.Call.Headers {
					strings_ = append(strings_, v)
				}
				expr.WalkStrings(step.Call.Body, func(s string) { strings_ = append(strings_, s) })
			}
			if step.Emit != nil {
				for _, v := range step.Emit.State {
					strings_ = append(strings_, v)
				}
				expr.WalkStrings(step.Emit.NextAction, func(s string) { strings_ = append(strings_, s) })
			}

			for _, s := range strings_ {
				for _, interp := range expr.FindInterpolations(s) {
					switch interp.Namespace {
					case "extract":
						if !flowHasResponse {
							f = append(f, errf("namespace-unavailable",
								"flow %q step %q: %s references the extract namespace, but this flow's entry step has no call or webhook trigger to extract from",
								flowName, stepName, interp.Raw))
						}
					case "event":
						if !flowHasEvent {
							f = append(f, errf("namespace-unavailable",
								"flow %q step %q: %s references the event namespace, only available in a flow whose entry step has triggers: [webhook]",
								flowName, stepName, interp.Raw))
						}
					case "input":
						if !hasInput {
							f = append(f, errf("namespace-unavailable",
								"flow %q step %q: %s references the input namespace, only available in a step with triggers: [input]",
								flowName, stepName, interp.Raw))
						}
					case "credentials", "ctx", "intent", "state", "idempotency_key":
						// Always available; no static check needed.
					default:
						f = append(f, errf("unknown-namespace",
							"flow %q step %q: %s references unknown namespace %q", flowName, stepName, interp.Raw, interp.Namespace))
					}
				}
			}
		}
	}
	return f
}

// checkAuthNamespaces applies the position-dependent availability rules
// (spec/04-expression-language.md) to auth.apply.value and auth.token's
// own fields, which evaluate outside any flow run: only credentials and
// ctx are ever available there, plus auth itself (the token-exchange
// result), and only when shape is oauth2_client_credentials. extract,
// event, input, intent, state, and idempotency_key never apply here,
// since none of them are populated until a flow actually runs.
func checkAuthNamespaces(m *model.Manifest) []Finding {
	var f []Finding
	a := m.Auth

	var strings_ []string
	if a.Apply != nil {
		strings_ = append(strings_, a.Apply.Value)
	}
	if a.Token != nil {
		strings_ = append(strings_, a.Token.Call.Path)
		expr.WalkStrings(a.Token.Body, func(s string) { strings_ = append(strings_, s) })
	}

	for _, s := range strings_ {
		for _, interp := range expr.FindInterpolations(s) {
			switch interp.Namespace {
			case "credentials", "ctx":
				// Always available here.
			case "auth":
				if a.Shape != "oauth2_client_credentials" {
					f = append(f, errf("namespace-unavailable",
						"auth: %s references the auth namespace, only available when shape is oauth2_client_credentials", interp.Raw))
				}
			default:
				f = append(f, errf("namespace-unavailable",
					"auth: %s references namespace %q, which is never available in auth.apply/auth.token (only credentials, ctx, and, for oauth2_client_credentials, auth)", interp.Raw, interp.Namespace))
			}
		}
	}
	return f
}

func checkTransforms(m *model.Manifest) []Finding {
	var f []Finding
	for flowName, flow := range m.Flows {
		for stepName, step := range flow.Steps {
			var strings_ []string
			if step.Call != nil {
				strings_ = append(strings_, step.Call.Path)
				for _, v := range step.Call.Headers {
					strings_ = append(strings_, v)
				}
				expr.WalkStrings(step.Call.Body, func(s string) { strings_ = append(strings_, s) })
			}
			if step.Emit != nil {
				for _, v := range step.Emit.State {
					strings_ = append(strings_, v)
				}
				expr.WalkStrings(step.Emit.NextAction, func(s string) { strings_ = append(strings_, s) })
			}
			for _, s := range strings_ {
				for _, interp := range expr.FindInterpolations(s) {
					if interp.Transform != "" && !model.Transforms[interp.Transform] {
						f = append(f, errf("closed-transform-set",
							"flow %q step %q: %s uses transform %q, not a member of the closed transform set", flowName, stepName, interp.Raw, interp.Transform))
					}
				}
			}
		}
	}
	return f
}

func checkCassetteCoverage(m *model.Manifest, cassettes map[string]*model.Cassette) []Finding {
	return checkOperationCassetteCoverage(m.Capabilities.Operations, cassettes)
}

// checkOperationCassetteCoverage is checkCassetteCoverage's operations-list
// form, shared with the native-provider path (checks/native.go), which has
// no *model.Manifest to read capabilities.operations off of.
func checkOperationCassetteCoverage(operations []string, cassettes map[string]*model.Cassette) []Finding {
	var f []Finding
	covered := map[string]bool{}
	for _, c := range cassettes {
		covered[c.Operation] = true
	}
	for _, op := range operations {
		if !covered[op] {
			f = append(f, errf("missing-cassette-coverage",
				"operation %q is declared in capabilities.operations but no cassette in cassettes/ has operation: %s", op, op))
		}
	}
	return f
}

func checkMetadata(m *model.Manifest, md *model.Metadata) []Finding {
	return checkMetadataAgainstProvider(m.Provider, md)
}

// checkMetadataAgainstProvider is checkMetadata's provider-slug form,
// shared with the native-provider path (checks/native.go), which has no
// *model.Manifest to read the provider slug off of.
func checkMetadataAgainstProvider(provider string, md *model.Metadata) []Finding {
	var f []Finding
	if md.Provider != provider {
		f = append(f, errf("metadata-mismatch", "metadata.yaml provider %q does not match provider %q", md.Provider, provider))
	}
	if !model.MetadataTiers[md.Tier] {
		f = append(f, errf("closed-enum", "metadata.yaml tier %q is not a recognized tier", md.Tier))
	}
	if !model.VerificationStatuses[md.Verification.Status] {
		f = append(f, errf("closed-enum", "metadata.yaml verification.status %q is not recognized", md.Verification.Status))
	}
	if md.Verification.Status == "provisional" {
		if md.Verification.DocsVerifiedOn != nil || md.Verification.VerifiedBy != nil {
			f = append(f, errf("verification-consistency", "verification.status is \"provisional\" but docs_verified_on/verified_by are set; they must be null until status is \"verified\""))
		}
	}
	if md.Verification.Status == "verified" {
		if md.Verification.DocsVerifiedOn == nil || md.Verification.VerifiedBy == nil {
			f = append(f, errf("verification-consistency", "verification.status is \"verified\" but docs_verified_on/verified_by are not both set"))
		}
	}
	if (md.Tier == "certified" || md.Tier == "official") && len(md.Maintainers) == 0 {
		f = append(f, errf("tier-requires-maintainer", "tier %q requires at least one named maintainer", md.Tier))
	}
	return f
}

// checkNextActionPayloads validates emit.next_action against the closed,
// per-variant field set in model.NextActionFields
// (spec/01-domain-model.md#nextaction-carries-its-own-payload): every
// required field for the declared type must be present, and no field
// outside that variant's required+optional set may appear. The type
// itself being a recognized NextAction member is already checked by
// checkFlows; this only runs once that holds.
func checkNextActionPayloads(m *model.Manifest) []Finding {
	var f []Finding
	for flowName, flow := range m.Flows {
		for stepName, step := range flow.Steps {
			if step.Emit == nil || step.Emit.NextAction == nil {
				continue
			}
			t := step.Emit.NextActionType()
			spec, ok := model.NextActionFields[t]
			if !ok {
				// Not a recognized NextAction member at all; checkFlows
				// already reported this, so avoid a duplicate finding.
				continue
			}
			allowed := map[string]bool{"type": true}
			for _, name := range spec.Required {
				allowed[name] = true
			}
			for _, name := range spec.Optional {
				allowed[name] = true
			}
			for _, name := range spec.Required {
				if _, present := step.Emit.NextAction[name]; !present {
					f = append(f, errf("next-action-missing-field",
						"flow %q step %q: next_action.type %q requires field %q",
						flowName, stepName, t, name))
				}
			}
			var extra []string
			for key := range step.Emit.NextAction {
				if !allowed[key] {
					extra = append(extra, key)
				}
			}
			sort.Strings(extra)
			for _, key := range extra {
				f = append(f, errf("next-action-unknown-field",
					"flow %q step %q: next_action.type %q does not define a field %q",
					flowName, stepName, t, key))
			}
		}
	}
	return f
}

// entryStepOf returns the one step in flow that no other step's
// goto/status_map ever targets, mirroring the entry-step computation in
// checkFlows and internal/replay.FindEntryStep. ok is false if flow does
// not have exactly one such step (checkFlows reports that separately).
func entryStepOf(flow model.Flow) (name string, ok bool) {
	targeted := map[string]bool{}
	for _, step := range flow.Steps {
		if step.Goto != "" {
			targeted[step.Goto] = true
		}
		if step.StatusMap != nil {
			for _, target := range step.StatusMap.Values {
				targeted[target] = true
			}
		}
	}
	var entries []string
	for stepName := range flow.Steps {
		if !targeted[stepName] {
			entries = append(entries, stepName)
		}
	}
	if len(entries) != 1 {
		return "", false
	}
	return entries[0], true
}

func containsTrigger(triggers []string, want string) bool {
	for _, t := range triggers {
		if t == want {
			return true
		}
	}
	return false
}

func sameStringSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
