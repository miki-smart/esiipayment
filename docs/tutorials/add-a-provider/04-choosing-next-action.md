# 4. Choosing the right `NextAction`

This is the section most first-time adapter authors spend the most time
on, and the one this project cares most about you getting right, because
it's the whole point of the exercise: **`NextAction` describes what the
END USER must do next, never what the provider's API is called
internally** ([Invariant I3](../../../spec/02-invariants.md#i3)). There
is no `NextAction` for "call this specific endpoint again" — that's
`Poll`, and which endpoint is a detail your manifest handles, not
something the integrator's checkout UI needs to know.

## The mapping table

[01-domain-model.md](../../../spec/01-domain-model.md#mapping-table-provider-behaviour-nextaction)
has the full table; the short version, from provider-observed behaviour
to the `NextAction` it maps to:

| If the provider's flow does this... | ...emit this `NextAction` |
|---|---|
| Hosted checkout page redirect (Chapa-style) | `RedirectToUrl` |
| 3-D Secure card challenge redirect | `RedirectToUrl` |
| STK push / prompt to a mobile wallet app | `AwaitDevicePush` |
| In-app payment approval notification | `AwaitDevicePush` |
| SMS one-time-password required | `SubmitOtp` |
| QR code returned for scan-to-pay | `DisplayQr` |
| Static or dynamic virtual account for bank transfer | `ShowTransferDetails` |
| USSD code the payer must dial | `DialUssd` |
| Accepted but outcome not yet confirmed | `Poll` |
| Distinct authorize-then-capture step required | `Capture` |
| Fully synchronous, nothing left for the user to do | `None` |

Not exhaustive of every provider UX imaginable, but every Ethiopian PSP
flow this project has encountered so far reduces to one of these.

## Every variant carries its own required payload

Since this repository's `2.0` manifest DSL,
`emit.next_action` is a typed object — `type` plus that variant's own
fields — not a bare label
([01-domain-model.md#nextaction-carries-its-own-payload](../../../spec/01-domain-model.md#nextaction-carries-its-own-payload)).
`esiipayment validate` rejects a step missing a required field for its
variant, or one carrying a field its variant doesn't define. The
required fields, briefly (see the linked table for the full list
including optional fields):

| Variant | Required fields |
|---|---|
| `RedirectToUrl` | `url` |
| `AwaitDevicePush` | `display_ref` |
| `SubmitOtp` | `length` |
| `DisplayQr` | `payload` |
| `ShowTransferDetails` | `account_number`, `institution`, `reference`, `amount` |
| `DialUssd` | `code` |
| `Poll` | `interval_ms` |
| `Capture`, `None` | — (no fields) |

Chapa's `awaiting_redirect` step (chapter 3) shows the simplest case —
one required field, extracted straight from the response:

```yaml
awaiting_redirect:
  emit:
    status: RequiresAction
    next_action:
      type: RedirectToUrl
      url: ${extract.data.checkout_url}
```

The **why** behind this matters more than the mechanics: before this
project's `2.0` DSL, this payload lived in `state` under a
provider-chosen key (`state.checkout_url` for one provider,
`state.payment_url` for another), and an integrator ended up writing
`state.checkout_url ?? state.payment_url` — branching on *which provider
this is*, in practice, even though nothing in their code literally said
`if provider == "chapa"`. Putting the payload on a closed, per-variant
field set on `next_action` itself is what makes "adding a provider never
touches integrator code" actually true. Don't reach for `state` to carry
anything the integrator's UI needs to act on — that's what `next_action`
is for. Reserve `state` for genuine adapter bookkeeping a *later*
operation needs, the way
[`providers/arifpay/manifest.yaml`](../../../providers/arifpay/manifest.yaml)
keeps a provider-assigned `session_id` in `state` because its `sync` flow
needs to address the same session later.

## When nothing in the table fits

If a provider's flow genuinely doesn't match any row — not "I'm not sure
which one," but "none of these describe what the user does here" — that's
worth raising as a spec question before guessing (see
[modify-the-spec](../modify-the-spec/index.md) if you end up being the
one to propose the fix), not a reason to invent a tenth `NextAction`
locally. `NextAction` is closed for the same reason every closed enum in
this project is: a tenth member is a change every language SDK must
implement identically before it's usable anywhere.

Next: [mapping errors](05-mapping-errors.md) — specifically, deciding
which of this provider's failures are `ResolveFirst`.
