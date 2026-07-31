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
spec_version: "1.0"             # manifest DSL version this manifest targets
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
[01-domain-model.md](01-domain-model.md#credentialshape)) and names the
credential fields the runtime must collect from the integrator and expose
under `${credentials.*}`.

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
```

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
  `NextAction`), and any `state` updates to persist;
- `goto` the name of the next step, or use `status_map` to branch to
  different next steps based on an extracted value: `status_map.path` is
  the extraction path evaluated against this step's response (or webhook
  body), and `status_map.values` maps each value that path might produce
  to the step to go to next.

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
        emit:
          state:
            checkout_url: ${extract.data.checkout_url}

      awaiting_redirect:
        emit:
          status: RequiresAction
          next_action: RedirectToUrl

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
        emit: { status: Processing, next_action: Poll }

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

## Worked example

See [providers/_template/manifest.yaml](../providers/_template/manifest.yaml)
for a fully commented skeleton, and
[providers/mock/manifest.yaml](../providers/mock/manifest.yaml) for a
complete, runnable example covering every `PaymentStatus` and every
`NextAction`.
