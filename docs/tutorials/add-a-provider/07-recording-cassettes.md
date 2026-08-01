# 7. Recording cassettes

A **cassette** is a fixed recording of one HTTP exchange (or a webhook
delivery), plus everything else needed to replay it deterministically:
the integrator's request (`intent`), the clock/UUID seed, fixture
`credentials`, and the recorded request/response bytes
([03-manifest-dsl.md](../../../spec/03-manifest-dsl.md),
[06-conformance.md](../../../spec/06-conformance.md)). It's what proves
your manifest actually produces the response shape it claims to, instead
of being a plausible-looking guess nobody has run.

## Where the recording comes from

Run your manifest's flow against your own **sandbox** account for this
provider — never production, never a real payer's real payment — and
record what actually comes back. If you don't have sandbox access for
this provider, a cassette built from the provider's own published
example responses in their documentation is an acceptable starting
point, but say so plainly in `metadata.yaml`'s `verification.notes`
(chapter 11): a cassette built from documented examples is a different,
lower-confidence thing than one built from an observed real response,
and integrators relying on this manifest deserve to know which.

## Scrub secrets before committing

**Never commit a real credential value.** Every `credentials` field in a
cassette is a fixture — a value your manifest's expression language can
interpolate during replay, never a real secret
([Invariant I9](../../../spec/02-invariants.md#i9)). Chapa's cassettes
use `"CHASECK_TEST-fixture-not-a-real-key"`; yours should use an
obviously-fake placeholder in the same spirit. Before committing, read
back through every recorded request and response body specifically
looking for: your real API key or secret, a real customer's phone
number/email/name if your sandbox test happened to use one, and any
other value your sandbox account holder wouldn't want public. Replace
each with an equally-shaped fake value — same format, same rough length,
obviously not real.

## The required minimum set

Every provider needs, at minimum, cassettes covering:

- **Success** — the ordinary happy path for at least one operation.
- **A provider-reported failure** — something your `errors` list maps to
  a `FailureCode` via `path`/`equals` or `http_status`.
- **An auth failure** — a bad/expired credential, however this provider
  signals that.
- **A transport timeout** — `transport_failure: true`, no response
  recorded at all; see chapter 5 for why this one specifically must
  resolve to `Processing`+`Poll`, never `Failed`.

One per declared operation at minimum, not just for `collect`.
[`providers/mock/cassettes/`](../../../providers/mock/cassettes/) has 20
real, schema-valid examples covering every `PaymentStatus`/`NextAction`
combination across every operation, if you want more shapes to compare
against than the minimum set requires.
[`providers/chapa/cassettes/`](../../../providers/chapa/cassettes/) shows
this exact minimum set for a real (provisional) provider:
`collect.canonical_scenario.yaml` and `collect.redirect.yaml` (success,
two different scenarios), `collect.invalid_request.yaml` (provider
failure), `collect.auth_failed.yaml` (auth failure), and
`collect.timeout.yaml` (transport timeout), plus `sync.succeeded.yaml`
and `webhook.succeeded.yaml` for the other two operations it declares.

## What a cassette needs to fix

Everything a replay needs to be byte-identical every time it runs: the
`seed` (a fixed clock and UUID sequence — never let a cassette read your
runtime's real clock), `environment` (which of the manifest's
`environments` entries this replay resolves against), `intent` (the
integrator-supplied request), `ctx` (deployment-level values like
`webhook_url`/`return_url`), `credentials` (fixture values, per above),
and the `interactions` themselves (the exact request your manifest
should produce, and the exact response to feed back).

Next: [generating golden files](08-golden-files.md) — the other half of
what makes a cassette meaningful.
