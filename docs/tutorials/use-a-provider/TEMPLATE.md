# Using ESIIPayment with `<your-language>`

*This is a template, not a page anyone reads as-is. It lives in
`esiipayment-spec` (this repository) so every language SDK's "use a
provider" tutorial teaches the same nine things, in the same order, for
the same reason — not so every SDK's tutorial reads identically. Copy
this file into your SDK's own documentation, replace every
`<placeholder>` and `TODO` with your language's real idioms, keep the
section order and the section-6-before-section-7 ordering specifically
(see that section), and delete this italic paragraph and the
[checklist](#checklist-for-sdk-maintainers) at the bottom (that part's
for you, the maintainer filling this in — not for your SDK's own
readers).*

This tutorial uses
[the canonical scenario](../../examples/scenario.md) throughout: a
coffee merchant's order `INV-2291`, 500.00 ETB, paid via Chapa, resulting
in a redirect to a hosted checkout page. Every code sample below should
be a real, runnable `<your-language>` example against that same
scenario — not a different example invented per section.

## 1. Install and configure one provider

`TODO`: your package manager's install command, and how to supply
credentials for one provider (Chapa, for this tutorial's running
example). Show the minimum configuration needed to construct a client —
nothing about a second provider yet; that's section 7, deliberately
last among the "getting started" sections.

## 2. Create a payment

The canonical scenario, in your language:

```text
TODO: your language's real syntax, e.g.:
result = client.collect(
    amount: Money(50000, "ETB"),
    idempotency_key: "INV-2291",
    customer: { email: "abebe.kebede@example.com", ... },
)
```

Show the actual shape of `result` your SDK returns for this call
(`status: RequiresAction`, `next_action: {type: "RedirectToUrl", url:
"..."}`— see
[providers/chapa/expected/collect.canonical_scenario.json](../../../providers/chapa/expected/collect.canonical_scenario.json)
for the exact canonical values). Always pass an idempotency key
explicitly here, and say plainly that reusing one with a different
request is rejected, not silently overwritten
([Invariant I7](../../../spec/02-invariants.md#i7)).

## 3. Handle `NextAction` with a single switch

**This is the heart of the tutorial.** Show the complete switch over all
nine `NextAction` variants, in your language's real idiomatic
switch/match/pattern-matching construct — not a partial example with
"...and so on for the rest." The reference version every language
derives from:

--8<-- "docs/examples/integrator/reference-integrator.md:handle-next-action-switch"

Translate this directly into `<your-language>`'s real syntax, using your
SDK's actual `PaymentResult`/`NextAction` types. State the promise
explicitly, in your own words, right after the switch — don't let the
reader infer it: **adding a provider later requires no change to this
function.** This switch was written once, for Chapa in this scenario,
and it will handle ArifPay, SantimPay, and every future provider without
being touched, because every provider's manifest emits the same nine
`NextAction` shapes and nothing here reads a provider-specific field or
a `state` key. If your language's real implementation of this switch
ever needs a provider-specific branch, that's a bug in your SDK to fix,
not a limitation of this pattern to work around here.

## 4. The return handler

When the payer's browser returns to your site (from a `RedirectToUrl`,
typically), **never trust query parameters the user's browser sent** —
they're editable by anyone with the URL bar, including the payer
themselves before submission ever completes. Re-fetch the payment by its
idempotency key/id (`sync()`, or your SDK's equivalent) and act on *that*
authoritative result, not on `?status=success` or similar in the return
URL. Show this explicitly:

```text
TODO: your language's real syntax, e.g.:
function handle_return(request):
    payment_id = request.query["payment_id"]   # fine to trust: just an identifier
    result = client.sync(payment_id)            # never trust request.query["status"]
    handle(result)                               # section 3's switch
```

## 5. The webhook handler

Verify the inbound signature (your SDK does this for you, using the
manifest's declared `webhook.verification` scheme — see
[05-webhooks.md](../../../spec/05-webhooks.md)), **acknowledge the
webhook immediately** (return 2xx as soon as verification passes, before
doing any slow work), and process the actual status update
**asynchronously**, off the request/response cycle. A provider that
doesn't get a fast acknowledgment will retry the delivery, sometimes
aggressively; slow synchronous processing inside the webhook handler
itself is a common, avoidable source of duplicate deliveries.

## 6. The settler pattern

**Read this section before section 7, even though section 7 sounds like
the more exciting next step for actually shipping.** This ordering is
deliberate, not incidental: integrators who learn the settler pattern
last tend to treat it as an afterthought to bolt onto working code, and
it does not work bolted on.

You now have **three independent triggers** that can each observe a
payment reach a terminal status: the return handler (section 4), the
webhook handler (section 5), and — you'll need to build this one
yourself — a **sweeper**, a periodic job that re-`sync()`s any payment
still stuck in `Processing` past a reasonable window (covering the case
where the payer never returns and the webhook never arrives, or arrives
late). All three can fire for the same payment, in any order, and **any
one of them can win the race**.

The fix is **one idempotent function** — call it `settle(payment_id)` —
that every trigger calls, never three separate code paths that each
try to "finish" the order themselves:

```text
TODO: your language's real syntax, e.g.:
function settle(payment_id):
    result = client.sync(payment_id)
    if not is_terminal(result.status):
        return   # not done yet; whichever trigger runs next will check again
    if already_settled(payment_id):    # your own idempotent-settlement bookkeeping
        return                          # a second trigger arriving after the first already won
    if result.status == "Succeeded":
        fulfill_order(payment_id)
    mark_settled(payment_id)

# every trigger calls the same function:
function handle_return(request):     settle(extract_payment_id(request))
function handle_webhook(event):       settle(event.payment_id)
function sweep_stuck_payments():      for id in payments_stuck_in_processing(): settle(id)
```

The most common way to double-ship an order is treating the return
handler, the webhook handler, and the sweeper as three separate places
that each fulfill the order directly. Whichever of the three fires
first does the real work; the other two, arriving later for the same
already-terminal payment, must be safe, cheap no-ops — which is exactly
what routing all three through one idempotent `settle()` guarantees, and
three independent "finish the order" code paths do not.

## 7. Adding a second provider

Only now, with the settler pattern already in place, add ArifPay (or
SantimPay) alongside Chapa. Show that this is **purely configuration** —
a second credential set, a provider name — and that **nothing in
sections 2 through 6 changes**: the same `collect()` call shape, the
same section-3 switch, the same return handler, the same webhook
handler, the same `settle()` function, now serving two providers
identically. This is the demonstration, not just the assertion, of this
whole project's core promise.

## 8. Testing with `mock`

Configure the `mock` provider ([providers/mock/](../../../providers/mock/))
in any non-production environment: no credentials, no network access
required. Show how your SDK's config selects it, and that it can produce
every `PaymentStatus`/`NextAction` combination on demand (see
`providers/mock/manifest.yaml`'s comments for the full scenario list) —
useful for testing every branch of section 3's switch without needing
sandbox access to a real provider at all.

## 9. Going to production

What changes, explicitly: real credentials (from secure config/secrets
management, never hardcoded), a durable persisted store instead of
whatever section 8 used for testing, a publicly reachable webhook URL
registered with the real provider, and — say this plainly, linking to
the provider's own `metadata.yaml` in `esiipayment-spec` — this
provider's current `verification.status`. A `provisional` manifest means
its specifics haven't been confirmed against current provider
documentation; going to production against one is a real risk decision
to make consciously, not to discover later.

---

## Checklist for SDK maintainers

Before publishing your filled-in version of this tutorial:

- [ ] All nine sections present, **in this order** — section 6 (the
      settler) before section 7 (a second provider), specifically.
- [ ] Every code sample is real, running `<your-language>` code against
      this repository's `mock` provider or the canonical scenario — not
      pseudocode left over from this template.
- [ ] Section 3's switch is complete over all nine `NextAction`
      variants, translated faithfully from
      [docs/examples/integrator/reference-integrator.md](../../examples/integrator/reference-integrator.md),
      and states the "adding a provider requires no change here" promise
      explicitly.
- [ ] Section 6 shows one `settle()` function called from all three
      triggers (return handler, webhook handler, sweeper) — not three
      separate fulfillment code paths.
- [ ] Section 9 links to this repository's own provider `metadata.yaml`
      files for verification status, rather than restating (and risking
      going stale relative to) that status in your own SDK's docs.
- [ ] Every normative claim (what `Invariant I7` requires, what a
      `RetryClass` means) links back to `esiipayment-spec`'s own `spec/`
      documents rather than restating them in your own words that can
      drift from the source of truth.
