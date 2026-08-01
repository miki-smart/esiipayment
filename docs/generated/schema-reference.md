<!-- GENERATED from schema/*.json property descriptions — do not hand-edit. -->
<!-- Regenerate via this workflow, or the script below, locally. -->

# Schema Reference

## capabilities.v1.schema.json

**ESIIPayment Native Provider Capabilities**
Describes one provider whose behaviour is implemented as native code per language runtime, rather than as a manifest.yaml this repository's DSL can express. See spec/03-manifest-dsl.md#native-providers for the five signals that mean a provider needs this path instead of manifest.yaml. Carries only identity, credential shape, and capability metadata — never call/flow/error mechanics, since those live in each language's native implementation, not here.

| Property | Type | Description |
|---|---|---|
| `provider` | string | Unique slug identifying this provider. Must match the directory name providers/<provider>/. |
| `spec_version` | string | The spec version this provider targets, e.g. "1.0". See spec/08-versioning.md. |
| `implementation` | object | Always the literal "native": marks this provider as implemented as code per language runtime rather than as manifest.yaml. This is the only field that distinguishes this file from a manifest.yaml-driven provider at a glance. |
| `display_name` | string | Human-readable provider name shown in the generated catalog and in credential-input UI. |
| `country` | string | ISO-3166-1 alpha-2 country code of the provider's primary market, e.g. "ET". |
| `currencies` | array | ISO-4217 currency codes this provider can move. Every code listed must have an entry in vectors/money/exponents.json. |
| `auth` | object |  |
| `capabilities` | object |  |

## cassette.v1.schema.json

**ESIIPayment Cassette**
Records HTTP interactions plus the deterministic inputs needed to replay a flow and reproduce byte-identical golden output. See spec/06-conformance.md.

| Property | Type | Description |
|---|---|---|
| `name` | string | Cassette identifier of the form <operation>.<scenario>, e.g. "collect.success". Must have a matching expected/<name>.json in the same provider directory. |
| `operation` | string | The Operation this cassette exercises. Determines which flows entry point is run during replay. |
| `seed` | object |  |
| `interactions` | array | Ordered HTTP request/response pairs. A transport-timeout cassette (see errors[].match.transport == "timeout" in the manifest) declares this as an empty array plus a top-level `transport_failure: true` marker rather than a recorded response (see the `transport_failure` property below). |
| `transport_failure` | boolean | Present and true only on a cassette simulating a transport-level failure (timeout, connection reset) rather than a completed HTTP exchange. When present, interactions describes only the outbound request(s) attempted; no response is recorded because none was received. Required for the mandatory transport-timeout cassette every provider must carry. |
| `intent` | object | The integrator-supplied operation input the flow is run with, e.g. amount, currency, method for collect/payout, an amount for refund. Available at ${intent.*}. Keys mirror the operation's own input shape, not DSL vocabulary, so this object is intentionally not closed by additionalProperties: false (same rationale as call.body in the manifest schema). Required for every operation except webhook, where the inbound event plays this role instead. May be an empty object for an operation whose flow does not reference ${intent.*} at all (e.g. a sync keyed only on ${idempotency_key}). |
| `environment` | string | Name of the manifest's environments entry (spec/03-manifest-dsl.md#environments) this replay resolves call.path and ctx.base_url against, e.g. "sandbox". Required for every operation except webhook, which makes no outbound call of its own. |
| `state` | object | Pre-existing flow state this replay starts from, as if persisted by an earlier operation under the same idempotency key (Invariant I6, spec/02-invariants.md#i6), needed whenever this cassette's operation reads ${state.*} that only a prior operation's collect would have written, e.g. a sync keyed on a provider-assigned session ID rather than the original idempotency key. Omit, or leave empty, for a cassette whose flow starts from empty state (the ordinary case for a fresh collect). |
| `credentials` | object | Fixture values for the credentials namespace, keyed by the manifest's auth.fields names, e.g. the HMAC secret a webhook cassette's signature was actually computed with. These are test fixtures committed to a public repository, never real secrets (Invariant I9, spec/02-invariants.md#i9): a manifest with auth.shape: none omits this entirely; every other manifest's cassettes need enough of a fixture value here to make signature computation and any other ${credentials.*} interpolation reproducible during replay. |
| `ctx` | object | Fixed values for the runtime-supplied ctx namespace beyond base_url (which comes from the environment above), e.g. ctx.webhook_url, ctx.return_url, wherever a manifest's flows interpolate them. Freeform for the same reason intent is: these keys are deployment vocabulary, not DSL vocabulary. Omit, or leave empty, for a cassette whose flow never reads ${ctx.*} beyond path resolution. |
| `inputs` | array | Ordered values fed to the flow at each step reached via triggers: [input], in the order those steps are reached (e.g. an OTP digit string). Available at ${input.<name>} when that step runs. Omit entirely for a cassette whose flow never reaches an input-triggered step. |
| `inbound_webhook` | object | The inbound callback this cassette replays. Required when, and only meaningful when, operation is "webhook": a webhook is provider-initiated, so it has no integrator-supplied intent to replay instead. |

## manifest.v1.schema.json

**ESIIPayment Provider Manifest**
Describes one payment provider's behaviour entirely as data. See spec/03-manifest-dsl.md for the normative prose this schema encodes.

| Property | Type | Description |
|---|---|---|
| `provider` | string | Unique slug identifying this provider. Must match the directory name providers/<provider>/. |
| `spec_version` | string | The manifest DSL version this manifest targets, e.g. "1.0". See spec/08-versioning.md. A runtime must refuse to execute a manifest whose spec_version it does not implement. |
| `display_name` | string | Human-readable provider name shown in the generated catalog and in credential-input UI. |
| `country` | string | ISO-3166-1 alpha-2 country code of the provider's primary market, e.g. "ET". |
| `currencies` | array | ISO-4217 currency codes this provider can move. Every code listed must have an entry in vectors/money/exponents.json. |
| `environments` | object | Named base configurations (typically sandbox and production) that flow steps resolve against. Adapter authors never hardcode a host inside a call step. |
| `auth` | object |  |
| `capabilities` | object |  |
| `operations` | object | Maps each supported Operation to its flows entry point. Every key here must also appear in capabilities.operations, and must have a matching entry under flows. |
| `flows` | object | Named flow state machines. Every operations.*.entry_flow must name a key here. See spec/03-manifest-dsl.md#flows. |
| `errors` | array | Maps provider-specific error signals to the closed FailureCode/RetryClass sets. Must include one entry with match.transport == "timeout" mapped to ProviderTimeout / ResolveFirst. |
| `webhook` | object | Declares how to verify and interpret an inbound webhook callback. Required when capabilities.operations includes "webhook". |

## messages.v1.schema.json

**ESIIPayment FailureCode Message Catalogue**
One language's user-facing message per FailureCode, at messages/<language-code>.json. See docs/localization.md. Every language file must carry exactly the fourteen closed FailureCode members as keys, so no SDK's UI can end up missing a translation the English baseline has.

| Property | Type | Description |
|---|---|---|
| `language` | string | ISO 639-1 (or 639-2 where no 639-1 code exists) language code, matching this file's own name (messages/<language>.json). |
| `language_name` | string | The language's own name for itself (e.g. "አማርኛ" for Amharic), for display in a language picker. |
| `reviewed_by_native_speaker` | boolean | False until a fluent, ideally native, speaker has reviewed this file's messages for naturalness and correctness (not just a machine or non-native draft). See docs/localization.md. |
| `messages` | object | Exactly the fourteen closed FailureCode members (spec/01-domain-model.md#failurecode) mapped to one user-facing sentence each, in this file's language. |

## metadata.v1.schema.json

**ESIIPayment Provider Metadata**
Catalog and provenance information for one provider, stored at providers/<provider>/metadata.yaml. Drives the generated provider catalog and capability matrix (docs.yml) and the adapter-tier and verification-status displays. Separate from manifest.yaml, which is purely technical.

| Property | Type | Description |
|---|---|---|
| `provider` | string | Must match the provider slug in the sibling manifest.yaml and the directory name. |
| `tier` | string | Adapter tier. community: validates, cassettes present, golden output matches. certified: adds a live sandbox run and a named maintainer. official: core-team maintained, provider acknowledged. See GOVERNANCE.md. |
| `description` | string | One or two sentences describing this provider for the generated catalog. |
| `verification` | object |  |
| `maintainers` | array | Named maintainers for this provider. Required to have at least one entry when tier is certified or official. |
| `links` | object | Public links shown in the generated catalog. |
