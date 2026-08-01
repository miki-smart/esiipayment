# 07: Runtime Requirements

A **runtime** is a language SDK (`esiipayment-dotnet`, `esiipayment-python`,
`esiipayment-node`, `esiipayment-go`, `esiipayment-php`, or any other implementation)
that loads manifests from this repository and executes them. This document
is the checklist a runtime implementer works from: everything here is
required for [conformance](06-conformance.md), regardless of language.

None of this is provider-specific. A runtime implementer needs zero
knowledge of Chapa, ArifPay, or SantimPay's actual APIs to build a
conformant runtime; the `mock` provider ([providers/mock/](../providers/mock/))
is deliberately sufficient to develop and prove a runtime against before
it ever touches a manifest for a real provider.

## Loading and validating manifests

A runtime must load `providers/<name>/manifest.yaml`, validate it against
`schema/manifest.v1.schema.json`, and refuse to execute a manifest that
fails validation. Runtimes are encouraged to reuse the reference validator
(`tools/validator`, distributed as a container image) rather than
reimplementing JSON Schema validation, though re-implementing is not
prohibited as long as it agrees with the reference on every case in this
repository.

## Deterministic execution

Per [04](04-expression-language.md#determinism) and
[06](06-conformance.md#why-determinism-has-to-be-structural-not-disciplinary),
a runtime must accept an injected clock and an injected UUID source and use
*only* those during flow execution and cassette replay, never the
operating system's wall clock or default random source. In production use
(outside of replay), the injected clock/UUID source is simply the real
system clock and a real UUID generator; the requirement is that the
*seam* exists, so tests can substitute a fixed seed.

## Credential handling

A runtime collects credential values from the integrator according to the
manifest's declared `CredentialShape` and its named `auth.fields`, and
injects them into the `credentials` namespace at execution time. Per
[Invariant I9](02-invariants.md#i9), a runtime must never log, persist
unencrypted, or otherwise expose a credential value outside the expression
context it's injected into for the duration of the call that needs it. A
runtime applies `auth.apply` to every outbound call this provider's flows
make; a manifest's own `flows` never repeat this per step (see
[03-manifest-dsl.md#auth](03-manifest-dsl.md#auth)).

### Token refresh must be single-flight

For `auth.shape: oauth2_client_credentials`, a runtime must cache the
token exchanged via `auth.token` and refresh it proactively at
`refresh_at` of its lifetime, per [03-manifest-dsl.md#auth](03-manifest-dsl.md#auth) —
never exchange a fresh token on every call.

Refreshing must be **single-flight**: when multiple concurrent calls
observe that the cached token needs refreshing, exactly one of them
performs the token exchange; the rest wait on that one result rather than
each independently exchanging a new token. This is a correctness
requirement, not a performance optimization. Several real providers
permit only one active session per credential and invalidate the
previous token the instant a new one is issued; a runtime that lets N
concurrent requests each exchange a new token races itself, and the
result is every in-flight call but the last failing with an
authentication error — a failure mode that never shows up in a
single-request unit test and appears only under concurrent load, exactly
where it's hardest to diagnose. A runtime's own internal locking
mechanism for this (mutex, a leader-election pattern, whatever is
idiomatic) is not something this spec mandates; that exactly one token
exchange happens per refresh, regardless of how many concurrent callers
triggered it, is.

## Idempotency and payment record lifecycle

Per [Invariant I7](02-invariants.md#i7) and
[Invariant I8](02-invariants.md#i8), a runtime must:

- Accept an idempotency key on every `collect`/`payout`/`refund` call.
- Persist a payment record (idempotency key, request payload hash, initial
  `Processing` status) **before** issuing the first outbound network call.
- On a repeated call with the same key and identical payload, return the
  existing record's result without re-invoking the manifest's flow.
- On a repeated call with the same key and a **different** payload, reject
  with `FailureCode.DuplicateRequest` without executing anything.

## Flow state persistence

Per [Invariant I6](02-invariants.md#i6), a runtime must persist a flow's
`state` object (everything a manifest's steps wrote via `emit.state`)
opaquely between invocations of the same payment, across process
restarts, horizontal scaling, and arbitrary delay (a payment awaiting
`AwaitDevicePush` approval may resume hours later). A runtime is free to
choose its own storage mechanism (in-memory for tests, a database or cache
in production) as long as state written in one call is available, byte-for-
byte, to the next call for the same idempotency key.

## Retry policy

Per [Invariant I5](02-invariants.md#i5), retry behaviour lives entirely in
the runtime, is configurable by the integrator, and must consult
`RetryClass` (as fixed in
[01-domain-model.md](01-domain-model.md#retryclass) and encoded in
[vectors/errors/failure-retry-map.json](../vectors/errors/failure-retry-map.json))
before retrying any failed operation. A runtime must never retry a
`ResolveFirst` or `DoNotRetry` failure automatically; it may retry a
`SafeToRetry` failure according to its own backoff policy.

## Webhook handling

Per [05-webhooks.md](05-webhooks.md), a runtime must expose a way to
receive an inbound webhook request, hold its raw body for signature
verification (before any JSON parsing), verify it per the manifest's
declared `scheme` using a constant-time comparison, reject requests with a
missing or invalid signature, and otherwise route a verified webhook into
the matching flow's `webhook`-triggered step keyed by whatever correlator
the manifest's flow extracts from the body (per [Invariant I2](02-invariants.md#i2),
this must be a no-op against an already-terminal payment, not an error).

## Public API surface

Per [Invariant I12](02-invariants.md#i12), a runtime's public,
integrator-facing API must express every outcome purely in terms of
`PaymentStatus`, `NextAction`, `FailureCode`, and `RetryClass`, never a
provider identifier meant to be branched on. A runtime may expose the
active provider's name for logging/display purposes, but nothing in its
documented API should invite or require an integrator to switch on it.

## Conformance suite

A runtime must ship a test suite that, for every provider in this
repository:

- Validates the provider's manifest.
- Replays every cassette and asserts byte-identical canonical JSON against
  `expected/`.
- Passes every vector in [vectors/](../vectors/).

This is what `conformance-matrix.yml` invokes across all runtimes for every
manifest change, and what makes "identical behaviour across five languages"
a CI-enforced fact rather than an aspiration.
