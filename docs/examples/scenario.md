# The canonical scenario

Every tutorial in this repository — for a runtime implementer
([docs/tutorials/build-a-runtime/](../tutorials/build-a-runtime/)), a
provider or spec contributor
([docs/tutorials/add-a-provider/](../tutorials/add-a-provider/),
[docs/tutorials/modify-the-spec/](../tutorials/modify-the-spec/)), and an
SDK user
([docs/tutorials/use-a-provider/TEMPLATE.md](../tutorials/use-a-provider/TEMPLATE.md)) —
refers back to **this same transaction**, not one it invents for itself.
A runtime implementer, a provider author, and an integrator are all
looking at the same payment from three different angles; using one
scenario everywhere makes that literal, instead of asking the reader to
mentally reconcile three unrelated examples.

Don't restate this narrative in a tutorial. Link here, then show the
part specific to that tutorial's own concern.

## The transaction

| Field | Value |
|---|---|
| Merchant | Highland Coffee Roasters, an online coffee retailer |
| Merchant's own order reference | `INV-2291` |
| Amount | `50000` minor units — **500.00 ETB** |
| Customer | Abebe Kebede, `abebe.kebede@example.com`, `+251911234567` |
| Provider | [`chapa`](../../providers/chapa/) |
| Operation | `collect` |
| Outcome | `RequiresAction` + `RedirectToUrl` to a hosted checkout page |

Highland Coffee Roasters' checkout calls `collect()` once, for order
`INV-2291`, for 500.00 ETB, through Chapa. Chapa's hosted-checkout flow
([providers/chapa/manifest.yaml](../../providers/chapa/manifest.yaml))
doesn't resolve synchronously: it returns a checkout URL the customer
must be redirected to, and the actual outcome (paid or not) arrives later
via `sync` or a webhook. That two-step shape — an immediate
`RequiresAction` response, then a later terminal one — is exactly why
this scenario was chosen over a provider that resolves `collect`
synchronously: it's the one shape every one of this project's tracks
needs to show (the runtime's step executor, the provider author's
`NextAction` choice, and the integrator's `next_action` switch all have
something to do here that a same-request success/failure wouldn't
exercise).

## Where to find the executable version

This scenario isn't only prose. It's recorded as a real,
`esiipayment validate`/`esiipayment replay --assert-golden`-checked
cassette in this repository:

- Cassette:
  [providers/chapa/cassettes/collect.canonical_scenario.yaml](../../providers/chapa/cassettes/collect.canonical_scenario.yaml)
- Golden output:
  [providers/chapa/expected/collect.canonical_scenario.json](../../providers/chapa/expected/collect.canonical_scenario.json)

Every tutorial that shows this scenario's request or result should
snippet-include the relevant region from those two files (see
[03-manifest-dsl.md](../../spec/03-manifest-dsl.md) and this repository's
`mkdocs.yml`, which enables `pymdownx.snippets` for exactly this), not
paste a hand-copied version that can drift from what CI actually
verifies. `esiipayment lint` fails if a tutorial's snippet reference
names a file or a region that doesn't exist — see
[03-manifest-dsl.md](../../spec/03-manifest-dsl.md) and
`.github/workflows/docs.yml`.

## The result an integrator sees

```json
--8<-- "providers/chapa/expected/collect.canonical_scenario.json"
```

Reading this against
[01-domain-model.md#nextaction-carries-its-own-payload](../../spec/01-domain-model.md#nextaction-carries-its-own-payload):
`status` is `RequiresAction`, `next_action.type` is `RedirectToUrl`, and
`next_action.url` is the one field an integrator's UI needs — nothing in
`state` is meant for the integrator to read at all (it's `{}` here: Chapa's
adapter has no bookkeeping to carry into a later step for this
particular flow; contrast with a provider like ArifPay, whose `state`
carries a `session_id` a later `sync` call needs — see
[providers/arifpay/manifest.yaml](../../providers/arifpay/manifest.yaml)).

## How each track uses this scenario

- **Track 1 (build a runtime):** replay `collect.canonical_scenario.yaml`
  against `chapa`'s manifest and confirm your interpreter produces
  `collect.canonical_scenario.json` byte-for-byte, the same way you'd
  confirm it against any of `mock`'s 20 cassettes.
- **Track 2 (add a provider / modify the spec):** this is what a
  `RedirectToUrl`-shaped provider's cassette and manifest section look
  like end to end, worked all the way through instead of left abstract.
- **Track 3 (use a provider):** this is the exact response your
  `switch`/`match` over `NextAction` (the heart of that tutorial) must
  handle for this one call — see
  [docs/tutorials/use-a-provider/TEMPLATE.md](../tutorials/use-a-provider/TEMPLATE.md)'s
  section 3.
