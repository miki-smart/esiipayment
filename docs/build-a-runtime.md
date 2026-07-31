# Building a runtime

A runtime is a language SDK (`esiipayment-dotnet`, `esiipayment-python`,
`esiipayment-node`, `esiipayment-go`, `esiipayment-php`, or a new one) that loads
manifests from this repository (pinned as a git submodule) and executes
them. You need one programming language and zero provider-specific
knowledge: `providers/mock/` is deliberately complete and deterministic
enough to build and prove an entire runtime against before you ever touch
a real provider.

This guide is a practical on-ramp. [spec/07-runtime-requirements.md](../spec/07-runtime-requirements.md)
is the normative checklist; read this first to get oriented, then work
from that document as your source of truth.

## 1. Read the spec in this order

1. [spec/01-domain-model.md](../spec/01-domain-model.md): the closed
   enums and `PaymentResult` shape every runtime shares.
2. [spec/03-manifest-dsl.md](../spec/03-manifest-dsl.md): what a
   manifest describes and how a flow executes.
3. [spec/04-expression-language.md](../spec/04-expression-language.md):
   the extraction/interpolation/condition language you need to implement,
   in full, and nothing more.
4. [spec/06-conformance.md](../spec/06-conformance.md): canonical JSON
   and what "golden output" means; this is the bar your runtime is
   actually judged against.
5. [spec/07-runtime-requirements.md](../spec/07-runtime-requirements.md):
   the checklist below, in full.

## 2. Build against `mock` first

`providers/mock/manifest.yaml` has no credentials, makes no real network
calls, and its `collect` operation can produce every `PaymentStatus` and
every `NextAction` on demand by choosing the right `method` value: see
the 20 cassette/`expected` pairs in `providers/mock/cassettes/` and
`providers/mock/expected/`. Get your runtime to:

1. Load and validate `providers/mock/manifest.yaml` against
   `schema/manifest.v1.schema.json`.
2. Replay every cassette in `providers/mock/cassettes/` with the
   cassette's `seed` (clock, UUIDs) injected instead of your runtime's
   ambient clock/RNG.
3. Produce byte-identical canonical JSON against every matching file in
   `providers/mock/expected/`.

Once this passes for `mock`, repeat it for `providers/chapa`,
`providers/arifpay`, and `providers/santimpay`, real (if provisional)
manifests that exercise `http_status`-based error matching and a real
HMAC webhook signature, which `mock` deliberately doesn't need.

## 3. The non-negotiable behaviours

These aren't optional polish; a runtime that gets any of these wrong is
not conformant, regardless of how correct its happy-path output looks:

- **Deterministic replay.** Inject the cassette's clock and UUID
  sequence; never read the system clock or a real RNG during replay.
  This is what makes "byte-identical canonical JSON" a meaningful bar at
  all: see [spec/06-conformance.md](../spec/06-conformance.md).
- **Transport failure → `Processing` + `Poll`, never `Failed`.**
  [Invariant I4](../spec/02-invariants.md#i4) exists specifically to
  prevent double-charging; every cassette named `*.timeout` tests this.
- **Fixed `FailureCode` → `RetryClass` mapping**, read from
  [vectors/errors/failure-retry-map.json](../vectors/errors/failure-retry-map.json)
  (or an exact copy of it), never overridden per-adapter or
  per-integrator config.
- **Idempotency**: same key + same payload replays without a new
  provider call; same key + different payload is
  `FailureCode.DuplicateRequest`, never a silent overwrite. The payment
  record is written *before* the first provider network call: see
  [Invariant I8](../spec/02-invariants.md#i8) for why the ordering
  matters for crash recovery.
- **Webhook verification** over the *raw* body, constant-time comparison,
  missing signature header is a rejection: see
  [spec/05-webhooks.md](../spec/05-webhooks.md) and
  [vectors/webhook/hmac-sha256.json](../vectors/webhook/hmac-sha256.json).
- **No provider-name branching in the public API.** Every outcome is
  `PaymentStatus`/`NextAction`/`FailureCode`/`RetryClass`: see
  [Invariant I12](../spec/02-invariants.md#i12).

## 4. Vectors are the executable spec

Everything in [vectors/](../vectors/) is something your runtime's test
suite should assert against directly: extraction/interpolation/
condition behaviour, the money exponent table, the status transition
matrix, the failure/retry mapping, webhook signatures, canonical JSON.
Where prose and vectors disagree, the vectors win
([spec/04-expression-language.md](../spec/04-expression-language.md)).

## 5. Conformance in CI

Once your runtime passes locally, wire it into `conformance-matrix.yml`'s
expectations (see that workflow's comments in this repo for the
`repository_dispatch` contract your repo needs to implement): your CI
should be able to receive a dispatch naming a provider, check out this
repo's submodule at the given commit, run your conformance suite against
that one provider, and report a commit status back, so a manifest PR
that would break your runtime fails **on the manifest PR**, not later.

## Filing issues

If you hit a case the spec doesn't define precisely enough to implement
identically to another runtime, that's valuable signal: file it with
the "Runtime conformance issue" template. If the fix needs a new closed
enum member, expression language feature, or schema change, it becomes an
RFC per [GOVERNANCE.md](../GOVERNANCE.md#rfc-process); starting with the
conformance issue is still the right first step.
