# esiipayment: the ESIIPayment reference validator

This is the one piece of application code this repository permits (see
[spec/00-overview.md](../../spec/00-overview.md) and the root
[README.md](../../README.md)); everything under `providers/`, `vectors/`,
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
```

## Building

```
go mod tidy   # populates go.sum, deliberately not committed, see below
go build ./cmd/esiipayment
```

Or, without a local Go toolchain, via the container image:

```
docker build -t esiipayment-validator tools/validator
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator validate providers/mock
```

## Scope and honest limitations

This validator was written and reviewed carefully, but **the environment
it was authored in has no Go toolchain available, so none of this code
has actually been compiled or run**: there is no `go build` / `go test`
confirmation behind it, only careful manual review, cross-checked where
possible against equivalent logic re-implemented and executed in
JavaScript during development (the flow-dispatch and error-matching logic
here mirrors a Node.js prototype that *was* run against every cassette in
this repository and confirmed to produce the committed `expected/*.json`
files). Treat this as a solid first draft that needs `go build ./...`,
`go vet ./...`, and a run of `esiipayment validate`/`esiipayment replay
--assert-golden` against every provider in this repository as its actual
first test, not as already-proven-correct code. `go.sum` is intentionally
not committed for the same reason: its cryptographic hashes can't be
hand-verified without fetching the module, and a wrong hand-written
go.sum would fail closed (safely) but confusingly. `go mod tidy` (which
`validate.yml` and the Dockerfile both run) generates it correctly on a
machine with real module access.

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
