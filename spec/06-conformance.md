# 06: Conformance

"Every SDK behaves identically" is only a testable claim if there's an
exact, mechanical definition of *identical*. This document defines
canonical JSON (the byte-level output format every runtime must produce),
golden output (the mechanism that compares runtimes against it), and what
it means for a runtime or a provider manifest to be conformant.

## Canonical JSON

Canonical JSON is the exact serialization every runtime must produce for a
given logical value. Two runtimes that agree on the logical result of a
flow but disagree on the canonical bytes have still failed conformance:
byte-identity, not mere logical equivalence, is the bar, because
byte-identity is what a diff can check without a runtime having to embed
another runtime's parser.

Rules, applied recursively to the entire document:

1. **Object keys are sorted lexicographically (byte-wise, by UTF-8 code
   unit)** at every nesting level. `{"b":1,"a":2}` must be emitted as
   `{"a":2,"b":1}`.
2. **No insignificant whitespace.** No spaces after `:` or `,`, no
   trailing newline, no indentation. The entire document is one line.
3. **Integers are never rendered as floats.** A `Money.minor_units` value
   of `10000` must be emitted as `10000`, never `10000.0` or `1e4`. This is
   the JSON-serialization-level enforcement of
   [Invariant I1](02-invariants.md#i1); the rule holds at the model layer
   and must survive serialization too, since a runtime whose internal
   representation is a correct integer but whose JSON encoder helpfully
   "normalizes" it to a float has still produced a non-conformant, and
   arguably worse, output than never having the integer at all.
4. **Timestamps are ISO-8601 in UTC, fixed to second precision, with a
   literal `Z` offset**: `2026-01-15T09:30:00Z`. No fractional seconds, no
   `+00:00` offset notation, no local-timezone rendering. A cassette's
   `seed.clock` is already in this form, and every timestamp a flow emits
   or writes to `state` must round-trip through this exact format.
5. **Strings are emitted with standard JSON escaping** (no unnecessary
   escaping of `/`, no forcing of non-ASCII characters to `\uXXXX`; emit
   UTF-8 directly). Two runtimes that both correctly escape the same
   string must produce identical bytes for it.
6. **No trailing commas, no comments**: this is output JSON, not the YAML
   manifest source; standard strict JSON syntax throughout.

[vectors/canonical-json/](../vectors/canonical-json/) pairs input logical
objects with their exact expected canonical string, covering nested key
sorting, integer-vs-float edge cases, and timestamp formatting. A runtime's
canonicalization routine must match every vector byte-for-byte before it
is trusted to produce golden output at all: this is the prerequisite
layer beneath golden-output comparison, not an optional nicety.

## Golden output and cassette replay

A **cassette** ([schema/cassette.v1.schema.json](../schema/cassette.v1.schema.json))
records a fixed clock, a fixed sequence of UUIDs, a fixed idempotency key,
the integrator-supplied operation input (`intent`, or the inbound event
for a webhook), which named `environment` and `ctx` values to resolve
against, fixture `credentials` values (test fixtures only; never a real
secret, per [Invariant I9](02-invariants.md#i9)) for any provider that
needs them, any pre-existing flow `state` this replay starts from (for an
operation, like some providers' `sync`, whose flow reads state only a
prior operation would have written), and the exact sequence of HTTP
request/response interactions a flow run should produce and receive.
Given:

- one provider's `manifest.yaml`,
- one cassette,
- the cassette's `seed` values injected as the runtime's clock and UUID
  source instead of the operating environment's,

every runtime must:

1. Issue outbound requests matching the cassette's recorded `request`
   entries (method, path, and, where the cassette specifies it, body),
   in order.
2. Feed the corresponding recorded `response` back to the flow as if it
   came from the real provider, never making a real network call during
   replay.
3. Produce a final result that, canonicalized per the rules above, is
   **byte-identical** to the matching `expected/<cassette-name>.json` file
   in the same provider directory.

`esiipayment replay <provider-dir> --assert-golden` performs steps 1-3 against
every cassette in a provider directory and fails on the first byte-level
mismatch, printing a diff. This is the tool a runtime implementer runs
locally, and the tool `conformance-matrix.yml`
([.github/workflows](../.github/workflows/)) runs across every
`{runtime} × {provider}` pair in CI: a manifest change that produces
different (even if arguably "more correct") output for an existing
cassette is a breaking change to that cassette and must ship as a new
cassette or a deliberate, reviewed update to `expected/`, never a silent
drift.

### Why determinism has to be structural, not disciplinary

Golden output only works if replay is fully deterministic. If a runtime
read the wall clock or generated a real random UUID during a flow, the
same cassette would produce different output on every run, and "byte
identical to `expected/`" would be meaningless. This is why
[04-expression-language.md](04-expression-language.md#determinism) requires
every expression evaluation to be a pure function of its inputs, and why
cassettes carry an explicit `seed` rather than letting each runtime free-run
its own clock and RNG during replay: determinism is a structural property
of the format, enforced by construction, not a discipline each runtime
implementer is trusted to maintain independently.

## What "conformant" means

A **runtime** is conformant for a given `spec_version` when it:

- Validates every manifest in this repository against
  `schema/manifest.v1.schema.json` without error,
- Passes every vector in [vectors/](../vectors/) (money, expressions,
  status transitions, error/retry mapping, webhook verification,
  canonical JSON),
- Produces byte-identical golden output, per the rules above, for every
  cassette of every provider in this repository, and
- Implements every requirement in
  [07-runtime-requirements.md](07-runtime-requirements.md).

A **provider manifest** is conformant when `esiipayment validate` passes
against it (schema plus the semantic checks in
[03-manifest-dsl.md](03-manifest-dsl.md)) and `esiipayment replay --assert-golden`
passes against its own cassettes on at least one reference runtime.
Conformance of the manifest does not by itself certify the manifest's
factual accuracy against the real provider's live API; see the
`verification.status` field on every provider's `metadata.yaml` and the
adapter tiers in [GOVERNANCE.md](../GOVERNANCE.md) for that separate axis.
