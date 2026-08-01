# 03: Manifest DSL

A provider manifest is a single YAML file, `providers/<name>/manifest.yaml`,
validated against [schema/manifest.v1.schema.json](../schema/manifest.v1.schema.json).
It is the entire description of how to talk to one payment provider: no
code, in any language, is needed to add or modify one.

Every manifest **must** begin with this exact line, before any other
content:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
```

This gives any editor that supports the `yaml-language-server` convention
(VS Code with the YAML extension, several JetBrains IDEs, `vim-lsp`, etc.)
inline validation and autocomplete against the schema with zero local
toolchain setup: the practical answer to "how does a contributor who
doesn't use your stack still get guardrails." `esiipayment lint` fails a
manifest missing this line.

The schema sets `additionalProperties: false` at every object level of the
DSL's own vocabulary: every top-level section, every `flows` step field,
every `auth`/`capabilities`/`emit` key. An unrecognized key there is a
validation failure, not a silently ignored typo; manifests rot the moment
a misspelled key stops meaning anything and nobody notices.

The one deliberate exception is `flows.*.steps.*.call.headers`,
`call.body`, and `status_map`: these carry the *provider's own* field and
header names and status strings, which this spec has no way to enumerate
in advance and which are not part of the DSL's vocabulary in the first
place. A typo inside a `call.body` value is a functional bug against the
real provider (caught by cassette replay, same as any other wrong
value), not a schema-rot problem, since it was never a key our own tooling
was supposed to recognize. Everywhere else, closure holds.

## Top-level sections

```yaml
provider: chapa                # unique slug, matches the directory name
spec_version: "2.0"             # manifest DSL version this manifest targets
display_name: "Chapa"
country: ET                     # ISO-3166-1 alpha-2
currencies: [ETB]
environments: { ... }
auth: { ... }
capabilities: { ... }
operations: { ... }
flows: { ... }
errors: [ ... ]
webhook: { ... }
```

### `provider`, `spec_version`, `display_name`, `country`, `currencies`

Identity and scope. `provider` is the slug used as the directory name and as
the identifier a runtime loads by. `currencies` is a list of ISO-4217 codes
this provider can move; every code listed must have an entry in
[vectors/money/exponents.json](../vectors/money/exponents.json).

### `environments`

Named base configurations a manifest's `call` steps resolve against
(typically `sandbox` and `production`), e.g. base URLs. Adapter authors
never hardcode a host inside a `flows.call`; they reference the active
environment's `base_url`.

```yaml
environments:
  sandbox:
    base_url: https://sandbox.example-psp.et/api
  production:
    base_url: https://api.example-psp.et/api
```

### `auth`

Declares exactly one `CredentialShape` (see
[01-domain-model.md](01-domain-model.md#credentialshape)), names the
credential fields the runtime must collect from the integrator and expose
under `${credentials.*}`, and (`apply`) where the resolved credential goes
on every outbound call.

```yaml
auth:
  shape: triple_key
  fields:
    - name: secret_key
      description: Secret key issued in the provider merchant dashboard; used as the Bearer token on every request.
    - name: public_key
      description: Public key, sent on client-initialization requests only.
    - name: encryption_key
      description: Used to encrypt card fields when accepting card payments directly.
  apply:
    header: Authorization
    value: "Bearer ${credentials.secret_key}"
```

#### `apply`

Required for every shape except `none`. Exactly one of `header`, `query`,
or `body` names where the credential goes; `value` is the literal value
sent there, built from the same expression grammar as any other manifest
field. This is what lets a provider that authenticates via a custom
header (`X-API-Key`) or a query parameter be expressed at all: earlier
versions of this DSL assumed every credential was an `Authorization`
bearer header, which is simply false for some real providers.

```yaml
apply:
  query: api_key
  value: "${credentials.api_key}"
```

Because `apply` is declared once at the `auth` level, not per-`call`
step, a flow's own `call.headers`/`call.body` never repeat it: the
runtime attaches it to every request this provider's flows make,
automatically.

#### `token`: the mechanism behind `oauth2_client_credentials`

Required if, and only if, `shape` is `oauth2_client_credentials`.
Declaring that shape alone says nothing about *how* to get a token; without
`token`, `oauth2_client_credentials` was a shape with no way to actually
exchange one, which is why every provider needing a real token exchange
(a bank API, most telco-operated wallets) was previously inexpressible.

```yaml
auth:
  shape: oauth2_client_credentials
  fields:
    - name: client_id
      description: OAuth client id issued in the provider developer portal.
    - name: client_secret
      description: OAuth client secret issued alongside client_id.
  token:
    call: { method: POST, path: /v1/oauth/token }
    body:
      grant_type: client_credentials
      client_id: ${credentials.client_id}
      client_secret: ${credentials.client_secret}
    extract:
      access_token: $.access_token
      expires_in: $.expires_in
    refresh_at: 0.8
  apply:
    header: Authorization
    value: "Bearer ${auth.access_token}"
```

- `call` issues the token request the same way a flow step's `call`
  would (method/path, resolved against the active environment's
  `base_url`); `body` is the token request's own body.
- `extract` reads `access_token` and the token's `expires_in` (seconds)
  out of the token response, making `access_token` available at
  `${auth.access_token}` wherever `apply.value` (or anything else) needs
  it.
- `refresh_at` is the fraction of `expires_in` at which the runtime must
  proactively refresh, e.g. `0.8` refreshes with 20% of the token's
  lifetime still remaining, before it actually expires under a request in
  flight. A runtime must perform this refresh **single-flight**: see
  [07-runtime-requirements.md](07-runtime-requirements.md#credential-handling).

Per [Invariant I9](02-invariants.md#i9), a manifest never contains a literal
secret value; only field *names* and *descriptions* the runtime uses to
build a credential-input form.

### `capabilities`

Declares **limits, not just booleans**. A boolean tells an integrator a
method exists; a limit tells them whether their actual use case will work.

```yaml
capabilities:
  operations: [collect, sync, webhook]
  next_actions: [RedirectToUrl, Poll]
  currencies:
    ETB:
      min_amount: 100        # minor units: 1.00 ETB
      max_amount: 100000000  # minor units: 1,000,000.00 ETB
```

`esiipayment validate` cross-checks `capabilities.next_actions` against every
`next_action` a manifest's `flows` actually `emit`: the two sets must be
identical, not merely overlapping. It also cross-checks
`capabilities.operations` against `operations` (every declared operation
must have a corresponding `flows` entry and cassette coverage).

### `operations`

Maps each supported `Operation` (see
[01-domain-model.md](01-domain-model.md#operation)) to the `flows` entry
point that implements it.

```yaml
operations:
  collect:
    entry_flow: collect
  sync:
    entry_flow: sync
  webhook:
    entry_flow: webhook
```

### `flows`

A `flows` entry is a small named state machine. Each flow is a map of named
steps. A step may:

- `call` an HTTP request (method, path, headers, body, all built from the
  expression language in [04-expression-language.md](04-expression-language.md));
- declare `triggers` describing what causes this step to run: `webhook`
  (an inbound callback arrives), `poll` (the runtime calls again on a
  timer), `return` (the initiating call's HTTP response drives the step),
  or `input` (the integrator supplies a value, e.g. an OTP);
- `emit` the resulting `status` (a `PaymentStatus`), `next_action` (a
  typed `NextAction` object, see below), and any `state` updates to
  persist;
- `goto` the name of the next step, or use `status_map` to branch to
  different next steps based on an extracted value: `status_map.path` is
  the extraction path evaluated against this step's response (or webhook
  body), and `status_map.values` maps each value that path might produce
  to the step to go to next.

Because this reference interpreter (and every conformant runtime) keeps
one operation's response/webhook body available to every step in the
chain that handles it, not only the step whose `call` or `webhook`
trigger produced it, a step reached purely by `goto`/`status_map` (no
`call` of its own) may still reference `${extract...}`/`${event...}` from
that same response when building its own `emit.next_action` or
`emit.state`. This is what lets the step that decides the final
`next_action` be a different, more readable step than the one that made
the call.

#### `emit.next_action`

`next_action` is an object, not a bare enum value: a required `type` (one
of the nine `NextAction` members) plus that variant's own required and
optional fields, per the table in
[01-domain-model.md#nextaction-carries-its-own-payload](01-domain-model.md#nextaction-carries-its-own-payload).
Field values follow the same expression grammar as `emit.state`
(interpolation from any available namespace, or a literal), and may
nest one level for `ShowTransferDetails.amount`, which is a full `Money`
value:

```yaml
ra_redirect:
  emit:
    status: RequiresAction
    next_action:
      type: RedirectToUrl
      url: ${extract.data.checkout_url}

ra_transfer:
  emit:
    status: RequiresAction
    next_action:
      type: ShowTransferDetails
      account_number: ${extract.data.account_number}
      institution: ${extract.data.institution}
      reference: ${idempotency_key}
      amount:
        minor_units: ${intent.amount}
        currency: ${intent.currency}

proc_poll:
  emit:
    status: Processing
    next_action:
      type: Poll
      interval_ms: 5000
```

`esiipayment validate` rejects a step whose `next_action` is missing a
required field for its `type`, or that names a field its `type` does not
define — the same closure the rest of the DSL vocabulary gets. Note what
this replaces: routing a provider's own field name (`checkout_url`,
`payment_url`, whatever that provider calls it) into `emit.state` under a
manifest-author-chosen key, which is exactly the pattern that leaked
provider-specific keys to the integrator; see
[Invariant I3](02-invariants.md#i3) and
[Invariant I12](02-invariants.md#i12). `state` remains for genuine
adapter bookkeeping a *later* operation needs (e.g. a provider-assigned
session id `sync` must address), never for data the integrator's UI reads.

A flow's **entry step** (the step a runtime executes first when the
operation is invoked) is not named explicitly anywhere in the manifest.
It is the one step in the flow that no other step's `goto` or
`status_map.values` ever names as a target. A flow must have exactly one
such step; `esiipayment validate` computes this the same way (by elimination,
not by relying on YAML key order, which most languages' map/dict types do
not preserve) and rejects a flow with zero such steps (every step is
somebody's target; there's no entry point) or more than one (an
ambiguous flow with two unreachable-from-each-other starting points).

`emit.state` is a **patch, not a replacement**: the keys a step's
`emit.state` names are merged into the flow's accumulated state object;
any key written by an earlier step in the same run and not mentioned by
the current step's `emit.state` is left untouched. A flow with three steps
that each write one state key ends with all three keys present at the
end, not just the last one written. This is what lets an `initialize` step
record something every later step in the flow can still read (e.g. which
scenario a response corresponds to) without every subsequent step having
to re-declare it.

```yaml
flows:
  collect:
    steps:
      initialize:
        call:
          method: POST
          path: /v1/transaction/initialize
          body:
            amount: ${amount_major(intent.amount)}
            currency: ${intent.currency}
            tx_ref: ${idempotency_key}
            callback_url: ${ctx.webhook_url}
        triggers: [return]
        status_map:
          path: $.status
          values:
            success: awaiting_redirect
            failed: failed_terminal
        emit: {}

      awaiting_redirect:
        emit:
          status: RequiresAction
          next_action:
            type: RedirectToUrl
            url: ${extract.data.checkout_url}

      failed_terminal:
        emit:
          status: Failed

  sync:
    steps:
      query:
        call:
          method: GET
          path: /v1/transaction/verify/${state.tx_ref}
        triggers: [return]
        status_map:
          path: $.data.status
          values:
            success: succeeded
            failed: failed_terminal
            pending: still_processing
        emit: {}

      succeeded:
        emit: { status: Succeeded }
      failed_terminal:
        emit: { status: Failed }
      still_processing:
        emit:
          status: Processing
          next_action: { type: Poll, interval_ms: 5000 }

  webhook:
    steps:
      receive:
        triggers: [webhook]
        status_map:
          path: $.status
          values:
            success: succeeded
            failed: failed_terminal
        emit: {}
      succeeded:
        emit: { status: Succeeded }
      failed_terminal:
        emit: { status: Failed }
```

`goto` is used for unconditional transitions; `status_map` is used when the
next step depends on a value extracted from a response or webhook payload.
Every `goto`/`status_map` target must name a real step in the same flow;
`esiipayment validate` checks this exhaustively, including that every step is
reachable and that every flow's terminal steps `emit` one of the four
terminal `PaymentStatus` values.

### `errors`

A list mapping provider-specific error signals to the closed `FailureCode`
set, each with its `retry_class` (which must agree with the fixed table in
[01-domain-model.md](01-domain-model.md#retryclass);
[Invariant I10](02-invariants.md#i10)).

```yaml
errors:
  - match:
      path: $.data.status
      equals: insufficient_balance
    failure_code: InsufficientFunds
    retry_class: DoNotRetry

  - match:
      http_status: 401
    failure_code: AuthFailed
    retry_class: DoNotRetry

  - match:
      transport: timeout
    failure_code: ProviderTimeout
    retry_class: ResolveFirst
```

Three match kinds exist:

- **`path`/`equals`**: an ordinary body-content match, evaluated with the
  extraction language from
  [04-expression-language.md](04-expression-language.md).
- **`http_status`**: a reserved match kind for a failure signalled purely
  by HTTP status code, independent of what the body says. Real providers
  routinely return a bare 401 on bad credentials or a 429 on rate-limiting
  with no useful body field to key off of; `http_status` is for exactly
  that case, and is checked against the response actually received, not
  against any extracted value.
- **`transport: timeout`**: a reserved match kind used to declare how
  this provider's transport-timeout case is classified; every manifest
  must include exactly one such entry, and it must always resolve to
  `ProviderTimeout` / `ResolveFirst`.

A single response is expected to match **at most one** `errors` entry.
`esiipayment validate` treats two entries whose conditions could both match
the same response (e.g. a `path`/`equals` entry and an `http_status` entry
both plausibly true of the same recorded response in a cassette) as an
authoring ambiguity to be flagged, not something resolved by an implicit
ordering: an adapter author who needs both an HTTP status and a body
condition to identify one specific failure should cassette-verify that
distinction rather than relying on match order.

#### How `errors` interacts with `status_map`

`errors` is evaluated **before** a `call` step's own `status_map`, against
that same step's response (or, for `match.transport: timeout`, against the
absence of one). If a response matches an `errors` entry, the flow
short-circuits to that entry's outcome and `status_map` is never
consulted for this response. Concretely:

- A matched entry whose `failure_code` is **`ProviderTimeout`** or
  **`Unknown`** always short-circuits to `Processing` + `NextAction.Poll`
  (never `Failed`) per [Invariant I4](02-invariants.md#i4). This is the
  one case where a matched error does not simply become `Failed`.
- A matched entry with any other `failure_code` short-circuits to
  `Failed`, carrying that `failure_code` and its fixed `retry_class`.
- A response that matches no `errors` entry falls through to the step's
  own `status_map` exactly as shown in the worked example above.
- An `errors` match bypasses this step's own `emit` entirely: none of
  its `status`, `next_action`, or `state` is applied. The flow's state
  is left exactly as it was before this step ran; a step whose `call`
  matched an `errors` entry has, by definition, not produced state worth
  keeping from that attempt.

This ordering is what keeps "how do we classify a provider error" and
"what's the next step in the happy/expected path" as two independent
concerns an adapter author reasons about separately, rather than forcing
every `status_map` entry to also encode error classification inline.

### `webhook`

Declares how to verify and interpret an inbound callback; see
[05-webhooks.md](05-webhooks.md) for the full signature-verification model.

```yaml
webhook:
  verification:
    scheme: hmac_sha256_hex
    signature_header: Chapa-Signature
    secret: ${credentials.secret_key}
  body_format: json
```

## Field transforms

Transforms are a **closed named set**, applied as `${transform(path)}`
inside interpolation:

| Transform | Behaviour |
|---|---|
| `amount_major` | Converts an integer minor-unit `Money` amount to a major-unit decimal string per the currency's exponent (e.g. `10000` ETB → `"100.00"`). |
| `msisdn_et` | Normalizes an Ethiopian phone number to the provider-agnostic canonical form (`2519XXXXXXXX`). |
| `base64` | Base64-encodes a string. |
| `hex` | Hex-encodes bytes. |
| `sha256_hex` | SHA-256 hash, hex-encoded. |
| `upper` | Uppercases a string. |
| `lower` | Lowercases a string. |
| `iso8601` | Formats a timestamp as ISO-8601 UTC. |

There is no mechanism for a manifest to define a new transform or express
arbitrary logic. If a provider needs a transform not on this list, that is
a spec RFC (see [GOVERNANCE.md](../GOVERNANCE.md)), not a manifest-local
extension. The moment manifests could carry arbitrary expressions, the
"language-neutral data, not code" property collapses: every runtime would
need to embed a general-purpose expression evaluator, which is exactly the
five-times-reimplemented-scripting-language outcome this project exists to
avoid.

## Native providers

Every provider *should* be a manifest. The DSL above is deliberately
small (no boolean composition in conditions, no scripting, a closed
transform set), which is the entire point: a manifest is data any runtime
can execute identically, with no per-language reimplementation risk. But
"deliberately small" means some real provider, eventually, needs
something the DSL cannot express. When that happens, the answer is a
**native provider**, never a DSL feature added to fit one provider's
quirk.

### The five signals

A provider needs the native path — rather than a slightly-awkward but
still-expressible manifest — when it needs at least one of:

1. **Branching on a parsed intermediate value.** `status_map` and
   `errors` support exactly one extraction path compared against one
   literal (or, for `errors`, an HTTP status). A provider whose next step
   genuinely depends on combining two fields, a numeric range, or a
   computed value has no manifest-expressible way to decide that.
2. **Canonicalisation before signing.** Some providers require a request
   or webhook signature computed over a canonical form that isn't simply
   "the raw body" (sorted query parameters, a provider-specific
   concatenation rule, whitespace-sensitive re-serialization). The closed
   transform set has no general canonicalisation transform, deliberately
   (see [Field transforms](#field-transforms) above); a provider that
   truly needs one is exactly the native case, not a reason to add one.
3. **Session or cookie state.** The DSL's `state` is a flat, explicit,
   serializable object a manifest author names every key of. A provider
   whose API requires carrying an opaque cookie jar or session object
   across calls has no way to express that as manifest data.
4. **A non-HTTP channel.** Every `call` in this DSL is one HTTP request.
   A provider reached over SOAP, gRPC, a vendor SDK, or a USSD gateway's
   own binary protocol has nothing for `call` to describe.
5. **Status vocabularies needing more than equality comparison.** A
   provider whose success/failure signal requires a regex match, a
   numeric threshold, or inspecting more than one field to classify one
   outcome exceeds what `status_map`/`errors`
   ([How errors interacts with status_map](#how-errors-interacts-with-status_map))
   can express.

If none of these apply, express the provider as a manifest, even if it
takes a few more `errors` entries or an extra step than feels elegant: an
awkward-but-correct manifest costs nothing extra to any runtime, while a
DSL feature added to fit one provider's quirk is a feature every runtime,
forever, must now implement identically (see
[Field transforms](#field-transforms) above, and
[GOVERNANCE.md](../GOVERNANCE.md)'s RFC process, which exists precisely
to make that cost visible before merge). "The DSL can't express my
provider" is the wrong-question shape almost every time; the answer is
usually the native path below, not a new DSL feature. See
[docs/tutorials/modify-the-spec/](../docs/tutorials/modify-the-spec/) for
this same rule from the spec-contributor's side.

### The shape of a native provider

A native provider directory has no `manifest.yaml`. In its place:

```text
providers/<name>/
  capabilities.yaml   # identity + capabilities, NOT flows/errors/webhook
  metadata.yaml        # unchanged: same schema as any other provider
  cassettes/            # unchanged: same cassette.v1.schema.json shape
  expected/             # unchanged: golden PaymentResult JSON per cassette
```

`capabilities.yaml` ([schema/capabilities.v1.schema.json](../schema/capabilities.v1.schema.json))
carries exactly the parts of a manifest that describe *what the provider
can do*, not *how a runtime talks to it*: `provider`, `spec_version`,
`display_name`, `country`, `currencies`, `auth.shape`/`auth.fields` (for
generic credential-form generation only — a native implementation applies
credentials and performs any token exchange in code, not through
`auth.apply`/`auth.token`), and `capabilities` (`operations`,
`next_actions`, per-currency limits). It always sets one literal field
that marks it as native:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/capabilities.v1.schema.json
provider: examplebank
spec_version: "1.0"
implementation: native
display_name: "Example Bank"
country: ET
currencies: [ETB]
auth:
  shape: mutual_tls
  fields:
    - name: client_cert
      description: Client certificate issued by the bank for mutual TLS.
capabilities:
  operations: [collect, sync]
  next_actions: [RedirectToUrl, Poll]
  currencies:
    ETB: { min_amount: 100, max_amount: 100000000 }
```

There is no `environments`, `flows`, `errors`, or `webhook` section: those
describe DSL execution mechanics, and a native provider's actual HTTP (or
non-HTTP) behaviour is hand-written code in each language runtime, not
data in this repository. Per the governing no-application-code rule (see
[00-overview.md](00-overview.md)), that code lives in each language SDK's
own repository (`esiipayment-dotnet`, `esiipayment-python`, ...) — never
here. This repository holds only `capabilities.yaml`, `metadata.yaml`,
`cassettes/`, and `expected/` for a native provider, exactly as it does
for a manifest-driven one; `.github/workflows/no-code.yml`'s no-code rule
applies to a native provider's directory identically.

### What conformance means for a native provider

`esiipayment validate` validates `capabilities.yaml` against its schema
and cross-checks it the same way a manifest's `capabilities` section is
checked (closed enums, cassette coverage per declared operation), and
requires an `expected/<name>.json` for every cassette. What it *cannot*
do is what it does for a manifest: interpret the provider's own logic
and assert the cassette replays to that golden output, because there is
no manifest describing that logic to interpret. That verification is
each native implementation's own conformance suite's responsibility —
same cassettes, same expected `PaymentResult` JSON, same
[06-conformance.md](06-conformance.md) byte-identity bar, just asserted
by that language's own test runner instead of `esiipayment replay`.
`esiipayment replay providers/<native-provider>` prints an informational
message and exits successfully rather than erroring, so CI does not need
to special-case native providers to stay green.

## Worked example

See [providers/_template/manifest.yaml](../providers/_template/manifest.yaml)
for a fully commented skeleton, and
[providers/mock/manifest.yaml](../providers/mock/manifest.yaml) for a
complete, runnable example covering every `PaymentStatus` and every
`NextAction`.
