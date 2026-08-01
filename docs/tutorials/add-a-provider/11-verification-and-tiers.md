# 11. Verification status and tiers

Two independent axes describe a provider manifest in this project, and
conflating them is the most common way a first-time contributor either
undersells or oversells their work.

## `verification.status`: does it match reality?

In `metadata.yaml`:

```yaml
verification:
  status: provisional        # start here, always
  docs_verified_on: null
  verified_by: null
  notes: >-
    State plainly which fields are unverified and where your knowledge
    came from.
```

`provisional` means: this manifest's endpoint paths, field names, and
(especially) webhook signature scheme haven't been confirmed against
current, licensed provider documentation. Start here, unconditionally,
on your first PR — this is not a lesser or embarrassing status. All
three real providers in this repository today (`chapa`, `arifpay`,
`santimpay`) are `provisional`, for exactly this honest reason.

**Never mark something `verified` because it seems plausible.** The
generated provider catalog displays this status prominently specifically
so nobody builds a production integration on an unconfirmed manifest
without knowing it. If you're not certain, say so in `notes` — "I
haven't confirmed the webhook signature header name against current
docs" is a genuinely useful sentence for the next person to read; a
false `verified` is actively harmful.

Moving a manifest from `provisional` to `verified` — because you (or
anyone) later gets current, licensed access to this provider's
documentation, or a live sandbox account, and can confirm the specifics
are accurate — is its own, separately valuable contribution: update the
status, fill in `docs_verified_on` and `verified_by`, and open a PR
explaining specifically what you checked. This is at least as valuable
as writing the original manifest, arguably more so, for any provider a
lot of integrators are about to depend on.

## `tier`: how much ongoing commitment stands behind it

```yaml
tier: community   # certified, official
```

Independent of verification status entirely — see
[GOVERNANCE.md](../../../GOVERNANCE.md#adapter-tiers) for the full
criteria:

| Tier | What it takes |
|---|---|
| `community` | Validates, has cassette coverage for the minimum set, replays cleanly. The bar every first PR clears — nothing about "community" implies lower quality, only that it hasn't yet had the steps below. |
| `certified` | Everything in `community`, plus a live sandbox run verified by a maintainer or adapter reviewer, plus a named maintainer in `metadata.yaml` committed to keeping it current. |
| `official` | Core-team maintained, and the provider itself has acknowledged or engaged with the integration — the one tier this project can't self-certify, since it depends on an external relationship. |

Set `tier: community` on your first PR. There's no reason a first
contribution should claim more, and no penalty for starting there — it's
the expected, unremarkable starting point for every provider in this
project so far.

## The combination that's normal, not a red flag

A provider can be (and, as of this writing, every real provider in this
repository is) `community` tier **and** `provisional` verification, at
the same time. That combination isn't a warning sign — it's simply the
honest, default state of a manifest nobody has yet had the chance to
verify against current documentation or run past a maintainer's live
sandbox check. Read it as "here's exactly what's been confirmed so far,"
not as "this is unfinished work."

That's the whole tutorial. If you found a case none of these eleven
steps or `spec/03-manifest-dsl.md` covers, that's worth raising — either
as a manifest-level question in your PR, or, if it points at an actual
gap in the DSL itself, see
[modify-the-spec](../modify-the-spec/index.md).
