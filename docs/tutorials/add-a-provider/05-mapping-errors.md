# 5. Mapping errors

`errors` maps this provider's own failure signals — a body field, an
HTTP status, or a transport-level absence of any response — to the
closed `FailureCode` set, each with its `retry_class`. Chapa's, from
chapter 3:

```yaml
--8<-- "providers/chapa/manifest.yaml:errors-section"
```

Three match kinds, checked in this order against every response your
`flows` receive, before that step's own `status_map` is ever consulted
([03-manifest-dsl.md#errors](../../../spec/03-manifest-dsl.md#errors)):

- **`path`/`equals`** — a body field compared for equality.
- **`http_status`** — the raw HTTP status, independent of the body
  (some providers return a bare `401` on bad credentials or `429` on
  rate limiting with no useful body field to key off of; this is for
  exactly that).
- **`transport: timeout`** — no response was received at all.

## The `FailureCode -> RetryClass` mapping is fixed — you don't choose it

You choose which `FailureCode` a given provider signal maps to; you
**do not** choose what retrying that code means.
[`vectors/errors/failure-retry-map.json`](../../../vectors/errors/failure-retry-map.json)
is the complete, non-configurable table
([Invariant I10](../../../spec/02-invariants.md#i10)); `esiipayment
validate` rejects a manifest whose `retry_class` disagrees with it for a
given `failure_code`. Most of the fourteen codes are unambiguous —
`InsufficientFunds`, `InvalidRecipient`, `AuthFailed`, and most others
are `DoNotRetry`; a plain "the provider's servers had a bad moment
unrelated to this specific request" is `ProviderUnavailable` /
`SafeToRetry`.

## The one you have to get right: `ResolveFirst`

Two `FailureCode`s are `ResolveFirst`, and they're the two you need to
actually think about, not just look up:

- **`ProviderTimeout`** — the mandatory `transport: timeout` entry every
  manifest carries, verbatim, mapped to `ProviderTimeout`/`ResolveFirst`.
- **`Unknown`** — a provider failure your `errors` list doesn't
  recognize at all.

**A timeout is never a failure**, and this is the single highest-severity
thing to get right in this whole tutorial
([Invariant I4](../../../spec/02-invariants.md#i4)). When a request times
out, or the connection resets before any response arrives, you genuinely
don't know what the provider did with it — it may have received the
request, debited the payer, and simply never gotten a response back to
you in time. If that case were mapped to `Failed` /
`DoNotRetry`-or-`SafeToRetry`, a naive caller would see "safe to try
again" and turn one customer-authorized payment into two.
`ResolveFirst` means exactly what it says: before anything else happens
under this idempotency key, the caller must resolve the actual outcome
(a `sync` call, or wait for a webhook) — only then is any retry decision
even meaningful. `Unknown` gets the same treatment for the same reason:
an unrecognized failure is an unconfirmed outcome, not a confirmed safe
one.

Beyond the mandatory timeout entry, ask yourself: **does this provider
have any error condition where you genuinely can't tell from the
response alone whether the payment went through?** If so, that's
`ResolveFirst` too, and you should say so explicitly when you open your
PR (chapter 10 asks this directly). If every failure this provider
reports is unambiguous, that's worth noting as such, not leaving silent
— a provider that always gives a clear answer is a real, useful fact
about it.

## Ambiguity between entries

`esiipayment validate` flags two `errors` entries whose conditions could
both match the same response as an authoring mistake, not something
resolved by whichever one happens to come first in the list. If a
provider genuinely needs both an HTTP status and a body condition to
identify one specific failure, that's a sign you need a cassette
demonstrating the distinction, not a hopeful ordering.

Next: [validating with Docker](06-validating-with-docker.md).
