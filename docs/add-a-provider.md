# Adding a provider

You need no programming language to add a payment provider to ESIIPayment:
a provider is a YAML manifest, a metadata file, and a set of recorded
HTTP cassettes. If you've ever integrated an Ethiopian payment provider's
API (at a job, on a side project, or just reading their docs out of
curiosity), you have what it takes to write one of these in an evening.

This guide walks the whole process. Read [spec/03-manifest-dsl.md](../spec/03-manifest-dsl.md)
alongside it for the full normative reference; this doc is the practical
walkthrough, that one is the source of truth when they disagree.

## 1. Before you start: the clean-room norm

Write this manifest from provider documentation you're licensed to read,
your own sandbox testing, and your own memory of how the integration
behaves. **Do not** copy from, transcribe, or closely paraphrase an
employer's proprietary integration code or internal documentation. See
[CONTRIBUTING.md](../CONTRIBUTING.md#the-clean-room-norm): the adapter
PR template will ask you to affirm this directly, so think about it now
rather than at the end.

## 2. Copy the template

```
cp -r providers/_template providers/<your-provider-slug>
```

`providers/_template/manifest.yaml` is a fully commented skeleton with a
`TODO` at every fill-in point. `providers/mock/manifest.yaml` is a
complete, runnable example covering every `PaymentStatus` and every
`NextAction`. Read it side by side with the template if you want to see
a finished shape, not just a skeleton.

## 3. Fill in `manifest.yaml`

Work through the template's `TODO`s in order. Key decisions you'll make
along the way:

- **`auth.shape`**: pick the closest `CredentialShape`
  ([spec/01-domain-model.md](../spec/01-domain-model.md#credentialshape)).
  Never invent a new one, even if the provider's real scheme is a minor
  variant.
- **`capabilities`**: declare real limits (`min_amount`/`max_amount`),
  not just which operations exist. A boolean tells an integrator a method
  exists; a limit tells them whether their actual use case works.
- **`flows`**: one state machine per operation. The trickiest part for
  most providers is deciding which `NextAction` each non-terminal state
  maps to. Use the table in
  [spec/01-domain-model.md](../spec/01-domain-model.md#nextaction) and
  remember `NextAction` describes what the **end user** must do, never
  what the provider's API is called internally.
- **`errors`**: map every distinguishable provider failure to a
  `FailureCode`/`RetryClass` pair from the fixed table in
  [vectors/errors/failure-retry-map.json](../vectors/errors/failure-retry-map.json).
  The `transport: timeout` entry is mandatory, verbatim, mapped to
  `ProviderTimeout`/`ResolveFirst`: see
  [Invariant I4](../spec/02-invariants.md#i4) for why this is the single
  highest-severity thing to get right.
- **`webhook`** (if this provider has one): verification always runs
  over the *raw* body with a constant-time comparison
  ([spec/05-webhooks.md](../spec/05-webhooks.md)). If you genuinely don't
  know the provider's signature scheme, set `scheme: none` and say so
  plainly in `metadata.yaml`: see step 5. Don't guess a plausible header
  name and present it as fact.

Keep the required first line:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
```

This gets you inline validation and autocomplete in any editor that
supports the `yaml-language-server` convention, with no toolchain
install.

## 4. Write `metadata.yaml`

This is catalog/provenance information, separate from the technical
manifest. The field that matters most:

```yaml
verification:
  status: provisional        # start here, always, unless you've actually
                              # checked every field against current docs
  docs_verified_on: null
  verified_by: null
  notes: >-
    State plainly which fields are unverified and where your knowledge
    came from.
```

**Do not mark something `verified` because it seems plausible.** The
generated provider catalog displays verification status prominently
specifically so nobody builds on a provisional manifest without knowing
it. Moving a manifest from `provisional` to `verified` is itself a
contribution: see step 6.

Set `tier: community`: see [GOVERNANCE.md](../GOVERNANCE.md#adapter-tiers)
for what separates community from certified/official; every first PR
starts at community and that's the expected, unremarkable starting point.

## 5. Record cassettes

Every provider needs, at minimum, cassettes for: **success**, **a
provider-reported failure**, **an auth failure**, and **a transport
timeout**: one per declared operation at least. See
`providers/mock/cassettes/` for real, schema-valid examples across every
operation, and `providers/chapa/cassettes/` for cassettes exercising a
real (if provisional) manifest shape including `http_status`-based error
matching and HMAC webhook signatures.

A cassette fixes everything a replay needs to be deterministic: the
integrator's request (`intent`), the environment/`ctx` values, fixture
`credentials` (never a real secret; see
[Invariant I9](../spec/02-invariants.md#i9)), the clock/UUID seed, and the
recorded HTTP interaction(s). Each cassette needs a matching
`expected/<name>.json`: the exact canonical JSON
([spec/06-conformance.md](../spec/06-conformance.md)) your flow should
produce. Compute this by hand carefully (sorted keys, no insignificant
whitespace, integers never as floats) or by running `esiipayment replay`
without `--assert-golden` first to see what your manifest actually
produces, then reviewing that it's correct before committing it as the
golden file.

## 6. Validate and open the PR

```
esiipayment validate providers/<your-provider>
esiipayment replay providers/<your-provider> --assert-golden
esiipayment lint
```

All three must pass. Open the PR using the **adapter** PR template; it
asks specifically for the judgment calls a validator can't check: which
`NextAction` you chose for each state and why, which errors are
`ResolveFirst`, and whether you used the provider's own idempotency
mechanism. Answer these for real; they're the part of an adapter review
that actually needs a human.

## Moving from provisional to verified

If you have current, licensed access to a provider's documentation (or a
live sandbox account) and can confirm a manifest's specifics are
accurate, update `verification.status: verified`, fill in
`docs_verified_on` and `verified_by`, and open a PR explaining what you
checked. This is exactly as valuable a contribution as writing the
original manifest, arguably more, for a provider a lot of integrators
are about to rely on.
