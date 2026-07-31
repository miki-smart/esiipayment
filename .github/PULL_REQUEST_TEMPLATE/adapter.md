## Provider

<!-- e.g. chapa -->

## What changed

<!-- New provider, new operation support, fixed field/endpoint, new cassette scenario, etc. -->

## The judgment calls (please answer these — automation can't check them)

### NextAction mapping

For each of this provider's non-terminal states your manifest's flows
emit, which `NextAction` did you choose and why? (See the mapping table in
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
so, does your manifest use it (e.g. `tx_ref: ${idempotency_key}`)? If the
provider has no such mechanism, say so — that's a real constraint on
what this adapter can guarantee, not an oversight to silently paper over.

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
- [ ] Cassettes cover, at minimum: success, a provider-reported failure,
      an auth failure, and a transport timeout
- [ ] Every operation in `capabilities.operations` has cassette coverage
- [ ] `esiipayment lint` passes
