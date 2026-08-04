# esiipayment: the ESIIPayment reference validator

This is the one piece of application code this repository permits (see
[spec/00-overview.md](../../spec/00-overview.md) and the repository's own
root README); everything under `providers/`, `vectors/`,
and `schema/` is data, and `no-code.yml` in CI enforces that mechanically.

## What it does

```
esiipayment validate <provider-dir>
    Schema and semantic validation of one provider's manifest.yaml,
    metadata.yaml, and cassettes/: see spec/03-manifest-dsl.md and
    spec/06-conformance.md for what "semantic validation" covers beyond
    what JSON Schema alone can express (flow-graph reachability, the
    errors/status_map precedence rule, capability/emission cross-checks,
    cassette coverage, the closed transform set, position-dependent
    namespace availability).

esiipayment replay <provider-dir> --assert-golden
    Replays every cassette in <provider-dir>/cassettes/ against
    manifest.yaml and diffs the canonical JSON result against
    <provider-dir>/expected/<name>.json.

esiipayment lint [repo-root]
    Repo-wide: no code files under providers/, vectors/, or schema/;
    every manifest.yaml carries the required schema header; every
    provider's metadata is complete and internally consistent.

esiipayment catalog [repo-root]
    Generates the provider catalog and capability matrix as Markdown to
    stdout: what .github/workflows/docs.yml runs so the catalog is
    generated from metadata.yaml/manifest.yaml, never hand-written.

esiipayment conformance-integrator [repo-root]
    Runs the reference integrator (internal/integrator) — one generic
    handler over PaymentStatus/NextAction/FailureCode/RetryClass alone —
    against every cassette of every provider, unmodified. The
    machine-checkable form of Invariant I12: what
    .github/workflows/integrator-promise.yml runs on every change.
```

## Building

```
go build ./cmd/esiipayment
```

`go.sum` is committed, so this needs no `go mod tidy` first and every
build — yours, CI's, and the container image's — verifies the dependency
tree against the same pinned checksums. It was gitignored and regenerated
on each runner until 2026-08-04, which meant nothing ever pinned what the
module proxy served to the tool that gates every provider manifest in this
repository. If you change a dependency, run `go mod tidy` and commit the
resulting `go.mod`/`go.sum`; validate.yml's `test` job fails the build if
they are not already tidy.

Or, without a local Go toolchain, via the container image:

```
docker build -t esiipayment-validator tools/validator
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator validate providers/mock
```

## Scope and honest limitations

As of the `NextAction`-payload/`auth.apply`/native-provider/vectors
changes (spec/08-versioning.md's `2.0.0`), this code **has** actually
been built and run: `go build ./...`, `go vet ./...`, `go test ./...`,
`esiipayment validate`/`esiipayment replay --assert-golden`/`esiipayment
lint`/`esiipayment conformance-integrator` against every provider in this
repository, via `docker build`/`golang:1.22-bookworm` in an environment
with no local Go toolchain but working Docker. That process caught two
real bugs neither manual review nor the earlier JavaScript prototype had
caught (`internal/expr/expr_test.go` regression-tests both): `Extract`
treating a bare `"$"` as resolving to the whole document instead of
rejecting it as an invalid path, and `formatMinorUnits` overflowing on
exactly `math.MinInt64` by negating it before taking its magnitude. Both
are exactly the kind of edge case a byte-for-byte vector
(`vectors/expressions/extraction.json`, `vectors/money/conversions.json`)
exists to surface — treat that as the argument for actually running this
tool (or at minimum its vectors) against any future change here, not for
assuming everything else is now bug-free by extension: this remains a
reference implementation reviewed and tested by one contributor, not a
battle-tested library, and any *new* area of code added here should get
the same build-and-run treatment before being trusted, especially if no
Go toolchain is available in whatever environment made the change.
`go.sum` is intentionally not committed: its cryptographic hashes can't
be hand-verified without fetching the module, and a wrong hand-written
go.sum would fail closed (safely) but confusingly. `go mod tidy` (which
`validate.yml`, the Dockerfile, and CI all run) generates it correctly on
a machine with real module access.

Beyond that, deliberate scope limits (not bugs, but things a fuller
implementation should extend):

- **`replay`'s interpreter** handles exactly the shape every manifest in
  this repository currently uses: one outbound call (or one inbound
  webhook) per cassette, a single `status_map`/`errors` dispatch on that
  one response, then a chain of plain `emit`/`goto` steps to a terminal
  leaf. A flow that makes a second call within one operation, or
  re-dispatches on a fresh `status_map` more than once per run, is out of
  scope and returns an error rather than a wrong answer.
- **Webhook signature verification itself** (HMAC computation and
  constant-time comparison, spec/05-webhooks.md) is not implemented in
  `replay`: a cassette's `inbound_webhook` is fed straight to the flow
  as an already-verified event. A runtime implementation must do this for
  real; this reference tool's job is the DSL/golden-output side, not a
  full security-critical crypto demonstration.
- **Interpolation evaluation** assumes every `call.body`/`call.headers`
  string is either a literal or a *whole-value* `${...}` (optionally
  wrapping a transform), verified true of every manifest in this
  repository at the time of writing, but a future manifest that embeds
  interpolation inside a larger literal string in a body field (rather
  than only in `call.path`, where partial-string interpolation is already
  handled) would need `resolveBody` extended to match `interpolateString`'s
  general substitution.
- **The `FailureCode` → `RetryClass` table** in
  `internal/model/enums.go` is a hand-maintained copy of
  `vectors/errors/failure-retry-map.json`, not loaded from it; a
  worthwhile follow-up is loading the vector file directly (as
  `LoadExponents` already does for `vectors/money/exponents.json`) so
  there is exactly one source of truth on disk instead of two that must
  be kept in sync by hand.
