## Provider

<!-- e.g. chapa -->

## Kind

<!--
Delete the line that doesn't apply:

- manifest (manifest.yaml) — the ordinary case
- native (capabilities.yaml + implementation: native)

If native, you MUST name which of the five signals in
spec/03-manifest-dsl.md#the-five-signals forces it, specifically enough
that a reviewer can check the claim. "Requires canonicalisation before
signing: every request carries an RSASSA-PSS signature over sorted k=v
parameters (signal 2)" is checkable. "The DSL isn't flexible enough" is
not, and is the most common reason a native PR is sent back. See
docs/tutorials/add-a-provider/00-which-kind.md.
-->

## What changed

<!-- New provider, new operation support, fixed field/endpoint, new cassette scenario, etc. -->

## The judgment calls (please answer these — automation can't check them)

### NextAction mapping

For each of this provider's non-terminal states your manifest's flows —
or, for a native provider, your implementation — emit, which `NextAction`
did you choose and why? (See the mapping table in
[spec/01-domain-model.md](../../spec/01-domain-model.md#nextaction) — if
this provider's behaviour doesn't cleanly match a row in that table,
explain your reasoning here rather than guessing silently.)

<!--
e.g.
- Hosted checkout redirect -> RedirectToUrl (standard case)
- STK push prompt to the customer's phone -> AwaitDevicePush
-->

### Which errors are `ResolveFirst`?

Beyond the mandatory `transport: timeout` entry (always `ProviderTimeout`
/ `ResolveFirst`), does this provider have any error condition where the
outcome is genuinely ambiguous — where you can't tell from the response
alone whether the payment went through? List them and why they're
`ResolveFirst` rather than `DoNotRetry`/`SafeToRetry`. If none, say so
explicitly (a provider that always gives an unambiguous answer is worth
noting as such).

### Idempotency

Does this provider have its own idempotency/deduplication mechanism (an
idempotency key field, a client-reference field it deduplicates on)? If
so, does your adapter use it (e.g. `tx_ref: ${idempotency_key}` in a
manifest, or the equivalent in native code)? If the provider has no such
mechanism, say so — that's a real constraint on what this adapter can
guarantee, not an oversight to silently paper over.

## Verification status

- [ ] `providers/<name>/metadata.yaml` accurately reflects verification
      status — if you have **not** checked this manifest's specifics
      against current provider documentation, `verification.status` must
      be `provisional` with `docs_verified_on`/`verified_by` both `null`
      and a `notes` field explaining what's unverified. Do not mark
      something `verified` because it seems plausible.
- [ ] Knowledge of this provider's API came from documentation I'm
      licensed to read and/or my own testing — not from an employer's
      proprietary source (see the clean-room norm in
      [CONTRIBUTING.md](../../CONTRIBUTING.md)).

## Checklist

- [ ] `esiipayment validate providers/<name>` passes
- [ ] `esiipayment replay providers/<name> --assert-golden` passes
      (a native provider gets an informational skip here — see the native
      checklist below, which replaces what this would have proved)
- [ ] Cassettes cover, at minimum: success, a provider-reported failure,
      an auth failure, and a transport timeout
- [ ] Every operation in `capabilities.operations` has cassette coverage
- [ ] `esiipayment lint` passes

## Native providers only

Delete this whole section for a manifest provider. See
[docs/tutorials/add-a-provider/12-the-native-path.md](../../docs/tutorials/add-a-provider/12-the-native-path.md).

- [ ] `metadata.yaml` names which of the five signals forces the native
      path, specifically enough for a reviewer to check
- [ ] `providers/<name>/` contains only `capabilities.yaml`,
      `metadata.yaml`, `cassettes/` and `expected/` — no code, no
      `environments`/`flows`/`errors`/`webhook`
- [ ] The implementation PR (in a runtime repository, e.g.
      `esiipayment-dotnet`) is linked here, and its conformance suite
      replays *these* cassettes to *these* `expected/*.json` byte-for-byte
- [ ] The implementation reaches integrators through the same public
      client surface as a manifest-driven provider, so no integrator code
      branches on which kind it is (Invariant I12)
- [ ] `request.body` is recorded in every cassette, or `metadata.yaml`
      states why it cannot be and where request construction is asserted
      instead
