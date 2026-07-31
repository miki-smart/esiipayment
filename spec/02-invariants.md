# 02: Invariants

These twelve invariants (I1-I12) hold regardless of provider, language
runtime, or deployment environment. Every one of them is enforced by at
least one conformance vector or validator check; that pointer is given so a
failing test can always be traced back to the rule it exists to protect.

<a id="i1"></a>

## I1: Money is integer minor units

**Rule.** Every monetary amount, everywhere in the model (manifests,
cassettes, expected output, runtime APIs) is a signed 64-bit integer count
of the currency's minor unit plus an ISO-4217 code. No floating-point
representation of money is permitted at any layer.

**Rationale.** Floating point cannot represent common decimal fractions
exactly; a payment system that silently drifts by fractions of a currency
unit under accumulation or serialization is not fit for purpose. Integers
in a fixed, known unit have no such failure mode.

**Enforced by.** [vectors/money/](../vectors/money/) (conversion, rounding,
exponent-table vectors); `schema/manifest.v1.schema.json` types all amount
fields as integers; `esiipayment validate` rejects any manifest field typed or
observed as a JSON float where money is expected.

<a id="i2"></a>

## I2: `PaymentStatus` is closed; terminal states never transition

**Rule.** `PaymentStatus` has exactly the six members in
[01-domain-model.md](01-domain-model.md#paymentstatus). `Succeeded`,
`Failed`, `Canceled`, and `Expired` are terminal: once reached under a given
idempotency key, no further transition is legal, including to a different
terminal state.

**Rationale.** A payment record that can "un-succeed" or flip between two
terminal states is a record that can't be trusted for reconciliation.
Terminality has to be absolute, not a convention adapters are expected to
honour voluntarily.

**Enforced by.** [vectors/status/transitions.json](../vectors/status/transitions.json)
enumerates the full legal transition matrix and marks every
terminal-to-anything transition illegal; runtime conformance suites must
assert their state machine rejects every illegal transition in that file.

<a id="i3"></a>

## I3: `NextAction` is closed; variants describe user obligations

**Rule.** `NextAction` has exactly the nine members in
[01-domain-model.md](01-domain-model.md#nextaction). Each variant names
something the **end user** must do. It never encodes a provider API detail
(e.g. there is no `NextAction` for "call this specific endpoint again";
that's `Poll`, and the endpoint is a runtime implementation detail, not
something the integrator's UI needs to know).

**Rationale.** This is what keeps the integrator's UI code provider-agnostic:
a checkout page can have exactly nine `switch` branches, one per
`NextAction`, and never needs a tenth branch when a new provider is added.

**Enforced by.** `schema/manifest.v1.schema.json` (`emit.next_action` enum);
`esiipayment validate` cross-checks that a manifest's declared
`capabilities.next_actions` matches exactly the set of `next_action` values
its `flows` actually emit.

<a id="i4"></a>

## I4: A transport failure means the outcome is unknown

**Rule.** If an adapter cannot determine whether a provider processed a
request (connection reset, timeout, 5xx with an ambiguous body), the
result **must** be `Processing` + `NextAction.Poll`. It must never be
`Failed`.

**Rationale.** This is the invariant that prevents double-charging. A
`Failed` result signals "safe to try again" to callers and to the
`RetryClass` mapping. If a transport failure produced `Failed` and the
provider had, in fact, received and processed the original request, a
naive retry creates a second real payment. Reporting `Processing` + `Poll`
forces every caller down the same resolution path (query the true status
via `sync` or wait for a webhook) before any retry decision is made,
regardless of what actually happened on the wire.

**Enforced by.** [vectors/errors/](../vectors/errors/) includes a
timeout-classification vector; every provider's cassette set
([03-manifest-dsl.md](03-manifest-dsl.md)) must include a transport-timeout
cassette whose expected output is `Processing` + `Poll`, never `Failed`.
See also [I10](#i10).

<a id="i5"></a>

## I5: Adapters never retry internally; retry policy belongs to the runtime

**Rule.** A manifest's `flows` describe one attempt. Nothing in the manifest
DSL can express "retry this call N times" or backoff policy. Retry decisions
are made by the runtime, using `RetryClass`, and are configurable by the
integrator.

**Rationale.** Retry policy is an operational concern (how aggressive, what
backoff, what jitter, circuit-breaking across providers) that has nothing to
do with any individual provider's API shape, and folding it into manifests
would mean every adapter author re-deciding operational policy per provider.

**Enforced by.** The manifest schema has no retry/backoff vocabulary in
`flows`; `esiipayment lint` flags any manifest attempting to encode retry
behaviour via unsupported fields (schema `additionalProperties: false`
already rejects this structurally).

<a id="i6"></a>

## I6: Adapters are stateless; sequences live in runtime-persisted flow state

**Rule.** A manifest never assumes in-process memory persists between two
calls of the same flow. Anything that must survive between steps (which step
comes next, values extracted from a prior response) is written to an
explicit, serialisable `state` object that the *runtime* persists opaquely
between invocations.

**Rationale.** Real deployments call adapters from stateless request
handlers, scale horizontally, and may resume a flow minutes or days later
(e.g. `AwaitDevicePush` pending an out-of-band approval). An adapter that
assumed continuity of process memory would break under any of those
conditions.

**Enforced by.** `schema/manifest.v1.schema.json` requires `flows` steps to
declare their `emit.state` updates explicitly;
[vectors/expressions/](../vectors/expressions/) includes vectors reading
from the `state` namespace to confirm runtimes round-trip it correctly
across simulated process boundaries.

<a id="i7"></a>

## I7: Idempotency: same key + same payload replays; same key + different payload errors

**Rule.** Every `collect`/`payout`/`refund` call carries an idempotency key.
A repeated call with the same key and the same request payload must return
the original result without re-executing side effects. A repeated call with
the same key but a **different** payload is an error, not a silent
overwrite.

**Rationale.** Idempotency keys exist to make retries safe. If a changed
payload under a reused key were silently accepted, a client-side bug (or a
naive retry-with-different-amount) could cause the wrong payment to be
recorded under a key the caller believes still refers to the original
request.

**Enforced by.** [vectors/errors/](../vectors/errors/) includes an
idempotency-conflict vector mapped to `FailureCode.DuplicateRequest`;
conformance suites assert both the replay case and the conflict case.

<a id="i8"></a>

## I8: The payment record is written before any provider network call

**Rule.** The runtime persists a payment record (idempotency key, request
payload, initial `Processing` status) **before** issuing the first network
call to the provider, not after.

**Rationale.** This ordering, combined with I4, is what makes crash recovery
safe: if the process dies after the network call but before it would have
recorded the result, the record already exists in `Processing` and a
resumed runtime can `sync` to find the true outcome. If the record were
written after the call, a crash in that window would leave no record at
all of a request that may have reached the provider: an unrecoverable
double-payment risk with no key to reconcile against.

**Enforced by.** [07-runtime-requirements.md](07-runtime-requirements.md)
states this as a mandatory runtime behaviour; it is a runtime-implementation
invariant rather than something a manifest or cassette can directly assert,
so conformance here is checked by runtime test suite review, not by the
validator.

<a id="i9"></a>

## I9: Adapters receive credential capabilities, never key material

**Rule.** A manifest's `auth` section declares a `CredentialShape`: the
adapter author picks the shape, never the values. At execution time, the runtime
injects concrete secret values into the expression context's `credentials`
namespace; the manifest only ever references `${credentials.*}` by name. The
manifest source itself never contains a real secret, and the adapter
(manifest) has no code path capable of exfiltrating a credential anywhere
other than the provider call it's declared for.

**Rationale.** Manifests are reviewed in the open, in a public repository.
Nothing about a provider integration's *logic* requires the reviewer, CI, or
any other contributor to ever see a real credential. Keeping credential
*values* entirely outside the manifest and outside this repository is what
makes that possible.

**Enforced by.** `esiipayment lint` scans `providers/` for patterns resembling
live secrets; `schema/manifest.v1.schema.json` only allows
`${credentials.*}` interpolation, never a literal secret-shaped string, in
fields that reach outbound requests.

<a id="i10"></a>

## I10: Every failure carries a fixed retry classification

**Rule.** Every `FailureCode` an adapter can emit maps to exactly one
`RetryClass`, per the fixed table in
[01-domain-model.md](01-domain-model.md#retryclass). No adapter, provider,
or integrator configuration may override that mapping.

**Rationale.** If retry safety were configurable per adapter, an adapter
author's mistake (or a well-meaning "just retry everything" override) could
reintroduce the double-charge risk I4 exists to prevent. Fixing the mapping
centrally means the single highest-severity failure mode in this system
(treating an unknown outcome as safe to retry) cannot be reintroduced by any
one contributor's local decision.

**Enforced by.** [vectors/errors/failure-retry-map.json](../vectors/errors/failure-retry-map.json)
is the canonical mapping; `esiipayment validate` rejects any manifest `errors`
entry whose declared `retry_class` disagrees with this table for its mapped
`failure_code`.

<a id="i11"></a>

## I11: Webhook verification is constant-time over the raw body

**Rule.** Inbound webhook signature verification (HMAC or equivalent) must
compare against the **raw, unparsed** request body (not a re-serialized
form of it) using a constant-time comparison function.

**Rationale.** Comparing against a re-serialized body is fragile: any
whitespace or key-ordering difference between what the provider signed and
what got re-serialized breaks verification for legitimate requests.
Non-constant-time comparison of a signature is a timing side-channel that
leaks information about the correct signature to an attacker attempting
forgery.

**Enforced by.** [vectors/webhook/](../vectors/webhook/) includes real
precomputed HMAC signatures over raw bodies, tampered-body vectors that must
fail verification, and missing-header vectors; see
[05-webhooks.md](05-webhooks.md).

<a id="i12"></a>

## I12: Integrator code never branches on provider name

**Rule.** Nothing in the public runtime API exposes a provider identifier in
a way that's meant to be switched on by integrator code. All integrator-
visible outcomes are expressed purely in terms of `PaymentStatus`,
`NextAction`, `FailureCode`, and `RetryClass`.

**Rationale.** This is the promise stated in
[00-overview.md](00-overview.md): adding a provider must never require
touching integrator code. The moment integrator code contains
`if provider == "chapa"`, that promise is broken for every future provider
added, because the integrator now has an implicit dependency on the current
provider roster.

**Enforced by.** This is primarily a design discipline enforced through
runtime API review rather than an automated vector; runtime conformance
suites should include a test that constructs checkout logic against `mock`
alone and asserts it requires no changes when re-run against any other
provider's cassettes.
