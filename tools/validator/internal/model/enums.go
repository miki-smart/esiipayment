package model

// The closed enums from spec/01-domain-model.md, kept here as the single
// in-code source of truth the validator checks manifests against. If
// these ever need to change, that's an RFC against the spec first
// (spec/08-versioning.md); this file follows, it doesn't lead.

var PaymentStatuses = map[string]bool{
	"RequiresAction": true,
	"Processing":     true,
	"Succeeded":      true,
	"Failed":         true,
	"Canceled":       true,
	"Expired":        true,
}

var TerminalStatuses = map[string]bool{
	"Succeeded": true,
	"Failed":    true,
	"Canceled":  true,
	"Expired":   true,
}

var NextActions = map[string]bool{
	"None":                true,
	"RedirectToUrl":       true,
	"AwaitDevicePush":     true,
	"SubmitOtp":           true,
	"DisplayQr":           true,
	"ShowTransferDetails": true,
	"DialUssd":            true,
	"Poll":                true,
	"Capture":             true,
}

// NextActionFieldSpec is one NextAction variant's closed field set, from
// spec/01-domain-model.md#nextaction-carries-its-own-payload. "type" is
// implicitly required/allowed on every variant and is not repeated here.
type NextActionFieldSpec struct {
	Required []string
	Optional []string
}

var NextActionFields = map[string]NextActionFieldSpec{
	"None": {},
	"RedirectToUrl": {
		Required: []string{"url"},
		Optional: []string{"method"},
	},
	"AwaitDevicePush": {
		Required: []string{"display_ref"},
		Optional: []string{"expires_at"},
	},
	"SubmitOtp": {
		Required: []string{"length"},
		Optional: []string{"hint", "expires_at"},
	},
	"DisplayQr": {
		Required: []string{"payload"},
		Optional: []string{"image_url", "expires_at"},
	},
	"ShowTransferDetails": {
		Required: []string{"account_number", "institution", "reference", "amount"},
		Optional: []string{"account_name", "expires_at"},
	},
	"DialUssd": {
		Required: []string{"code"},
		Optional: []string{"expires_at"},
	},
	"Poll": {
		Required: []string{"interval_ms"},
		Optional: []string{"not_before"},
	},
	"Capture": {},
}

// SupportedSpecVersion is the only manifest-DSL spec_version this
// reference tool implements. Per spec/08-versioning.md, a runtime must
// refuse to execute a manifest whose spec_version it does not implement
// rather than attempting a best-effort interpretation.
const SupportedSpecVersion = "2.0"

// DefaultPollIntervalMs is the fixed interval_ms a runtime must use when it
// synthesizes a Poll action itself rather than reading one from a
// manifest step's emit: the Invariant I4 transport-failure case, and the
// ProviderTimeout/Unknown errors short-circuit
// (spec/03-manifest-dsl.md#how-errors-interacts-with-status_map). See
// spec/01-domain-model.md#why-interval_ms-is-required-even-for-a-runtime-synthesized-poll.
const DefaultPollIntervalMs = 5000

var FailureCodes = map[string]bool{
	"InsufficientFunds":      true,
	"InvalidRecipient":       true,
	"RecipientLimitExceeded": true,
	"SenderLimitExceeded":    true,
	"DuplicateRequest":       true,
	"AuthFailed":             true,
	"AuthorizationDeclined":  true,
	"ProviderUnavailable":    true,
	"ProviderTimeout":        true,
	"InvalidRequest":         true,
	"UnsupportedOperation":   true,
	"Expired":                true,
	"CanceledByUser":         true,
	"Unknown":                true,
}

var RetryClasses = map[string]bool{
	"SafeToRetry": true,
	"DoNotRetry":  true,
	"ResolveFirst": true,
}

// FailureRetryMap is the fixed FailureCode -> RetryClass table from
// vectors/errors/failure-retry-map.json. Kept in sync with that file by
// hand; a future improvement is to load it directly instead of
// duplicating it (see tools/validator/README.md).
var FailureRetryMap = map[string]string{
	"InsufficientFunds":      "DoNotRetry",
	"InvalidRecipient":       "DoNotRetry",
	"RecipientLimitExceeded": "DoNotRetry",
	"SenderLimitExceeded":    "DoNotRetry",
	"DuplicateRequest":       "DoNotRetry",
	"AuthFailed":             "DoNotRetry",
	"AuthorizationDeclined":  "DoNotRetry",
	"ProviderUnavailable":    "SafeToRetry",
	"ProviderTimeout":        "ResolveFirst",
	"InvalidRequest":         "DoNotRetry",
	"UnsupportedOperation":   "DoNotRetry",
	"Expired":                "DoNotRetry",
	"CanceledByUser":         "DoNotRetry",
	"Unknown":                "ResolveFirst",
}

var CredentialShapes = map[string]bool{
	"none":                      true,
	"api_key":                   true,
	"key_pair":                  true,
	"triple_key":                true,
	"oauth2_client_credentials": true,
	"hmac":                      true,
	"mutual_tls":                true,
}

var Operations = map[string]bool{
	"collect": true,
	"sync":    true,
	"payout":  true,
	"cancel":  true,
	"refund":  true,
	"webhook": true,
}

var Transforms = map[string]bool{
	"amount_major": true,
	"msisdn_et":    true,
	"base64":       true,
	"hex":          true,
	"sha256_hex":   true,
	"upper":        true,
	"lower":        true,
	"iso8601":      true,
}

var HTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

var Triggers = map[string]bool{
	"webhook": true, "poll": true, "return": true, "input": true,
}

var WebhookSchemes = map[string]bool{
	"hmac_sha256_hex": true, "hmac_sha256_base64": true, "none": true,
}

var MetadataTiers = map[string]bool{
	"community": true, "certified": true, "official": true,
}

var VerificationStatuses = map[string]bool{
	"provisional": true, "verified": true,
}
