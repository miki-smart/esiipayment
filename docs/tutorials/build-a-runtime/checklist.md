# Conformance checklist

Tick these off in order — each depends on the one before it, per this
track's [dependency ordering](index.md#why-this-order-specifically).
This is a checklist for yourself, not a substitute for actually running
`esiipayment validate`/`esiipayment replay --assert-golden` (or your own
`conformance-replay` entrypoint) against every provider.

- [ ] **Setup.** `esiipayment-spec` pinned as a submodule at a specific
      commit SHA (not a branch). ([1](01-setup.md))
- [ ] **Primitives.** `Money` is a signed 64-bit integer + currency code,
      never a float, anywhere. Exponents read from
      `vectors/money/exponents.json`, not hardcoded or locale-derived.
      All six closed enums implemented as closed sets. Fixed
      `FailureCode → RetryClass` table loaded from
      `vectors/errors/failure-retry-map.json`, `ProviderTimeout` and
      `Unknown` both `ResolveFirst`. Full `PaymentStatus` transition
      matrix (`vectors/status/transitions.json`) passing, including
      every terminal-state-transitions-nowhere case. ([2](02-primitives.md))
- [ ] **Expression evaluator.** `vectors/expressions/extraction.json`,
      `interpolation.json` (including the `auth` namespace and
      whole-value type preservation), and `conditions.json` all passing,
      with nothing implemented beyond what those vectors and
      `04-expression-language.md` define. ([3](03-expression-evaluator.md))
- [ ] **Canonical JSON.** `vectors/canonical-json/cases.json` passing
      byte-for-byte, including non-ASCII key sorting and the int64
      boundary case. ([4](04-canonical-json.md))
- [ ] **Error mapping and webhooks.** `vectors/errors/match-kinds.json`
      and `timeout-classification.json` passing (including the
      errors-before-status_map precedence rule).
      `vectors/webhook/hmac-sha256.json` passing using a
      constant-time comparison against the *raw* body. ([5](05-errors-and-webhooks.md))
- [ ] **Manifest loader.** Every manifest in `providers/` (mock, chapa,
      arifpay, santimpay) loads and validates; every deliberately-broken
      variant you constructed is rejected with a clear error.
      `spec_version` mismatch is refused, not best-effort interpreted.
      Native providers (`capabilities.yaml`) are recognized and routed
      away from the interpreter. ([6](06-manifest-loader.md))
- [ ] **Interpreter.** A step's `call` is attempted exactly once, always
      (Invariant I5). A transport fault always resolves to `Processing`
      + `Poll{interval_ms: 5000}`, never `Failed` (Invariant I4).
      `emit.state` patches rather than replaces. `emit.next_action`
      resolves through every namespace, not only `extract`. ([7](07-interpreter.md))
- [ ] **Replay against `mock`.** All 20 cassettes in
      `providers/mock/cassettes/` produce byte-identical canonical JSON
      against `providers/mock/expected/`. ([8](08-replay-against-mock.md))
- [ ] **Replay against real providers.** `chapa`, `arifpay`, and
      `santimpay`'s cassettes all pass too, unmodified harness.
      ([8](08-replay-against-mock.md))
- [ ] **Native providers — only if you implement any.** Supporting none is
      fully conformant. For each one you do support: it is reachable
      through the same public client type and the same operation methods
      as a manifest provider (Invariant I12); the shared invariants
      (I2/I4/I7/I8) come from one place both kinds go through, not
      re-derived per provider; its cassettes replay byte-identically to
      its `expected/*.json` in *your* suite, since `esiipayment replay`
      cannot do it for you; and your README states which native providers
      you implement rather than leaving integrators to guess from the
      provider catalog.
      ([spec/07-runtime-requirements.md#native-providers](../../../spec/07-runtime-requirements.md))
- [ ] **The engine.** Idempotency vector
      (`vectors/errors/idempotency-conflict.json`) passing against your
      real persistence layer. Four invariant tests passing: a chaos
      transport (I4), a counting transport (I5), a store spy (I8), and a
      serialize-and-rehydrate test (I6). Single-flight token refresh
      tested under concurrency, if you support
      `oauth2_client_credentials`. ([9](09-the-engine.md))
- [ ] **`conformance-replay` exposed and registered.** Your runtime
      exposes a `conformance-replay`-equivalent entrypoint; your
      repository has a receiver workflow for the
      `esiipayment-conformance-check` dispatch; you've opened (or
      merged) a PR adding your SDK to `conformance-matrix.yml`.
      ([10](10-conformance-replay.md))
- [ ] **Published honestly.** A compliance matrix naming your actual,
      checkable state (which providers, which operations, each
      provider's real `verification.status`/tier from this repository's
      own `metadata.yaml`). Still `0.x` unless a provider you support has
      reached `verification.status: verified`. ([11](11-publishing.md))
- [ ] **You've found at least one vector gap and reported or fixed it
      upstream** — or you haven't yet, and that's fine; it means you
      haven't hit one yet, not that none exist. Keep this box open as a
      standing reminder, not a one-time tick.
