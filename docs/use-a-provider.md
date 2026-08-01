# Using a provider (for integrators)

This page is for integrators using an ESIIPayment language SDK
(`esiipayment-dotnet`, `esiipayment-python`, `esiipayment-node`, `esiipayment-go`,
`esiipayment-php`) to accept payments, not for contributing to this
repository. If you're looking to add a provider or build a runtime, see
[CONTRIBUTING.md](../CONTRIBUTING.md) instead.

> This repository defines the contract every SDK implements identically:
> it's the right place to understand *what* the SDKs guarantee. For a
> specific SDK's install instructions and language-idiomatic API, see that
> SDK's own repository and documentation.

## The promise

> Integrator code never changes when a provider is added.

You configure which provider a merchant uses (Chapa, ArifPay, SantimPay,
`mock` for testing, or any future provider added to this repository)
through configuration (credentials and a provider name), never by
writing different code per provider. The same `collect` call, against
`mock` in your tests and a real provider in production, returns the same
shape.

## The four things every response is made of

Every SDK call returns some combination of these, and nothing else; see
[spec/01-domain-model.md](../spec/01-domain-model.md) for the full
definitions:

- **`PaymentStatus`**: exactly six values: `RequiresAction`,
  `Processing`, `Succeeded`, `Failed`, `Canceled`, `Expired`. The last
  four are terminal and never change again under the same reference.
- **`NextAction`**: what the end user must do, if anything: redirect to
  a URL, approve a device push, submit an OTP, scan a QR, transfer to a
  displayed account, dial a USSD code, poll again later, or nothing.
  Exactly nine variants, closed: your UI can have exactly one branch per
  variant and never needs a tenth when a new provider is added.
- **`FailureCode`**: why a `Failed` payment failed (only meaningful when
  `status` is `Failed`).
- **`RetryClass`**: whether retrying is safe, unsafe, or requires
  resolving the true outcome first. This mapping is fixed by the spec,
  not something your integration configures.

**Never branch your integration logic on which provider is configured.**
If you find yourself writing `if provider == "chapa"`, something is wrong
: either the SDK is leaking a detail it shouldn't, or the code belongs in
a provider manifest, not your application (see
[Invariant I12](../spec/02-invariants.md#i12)).

## The shape of an integration, illustratively

The exact method names and types are your SDK's, but every ESIIPayment SDK
follows this pattern:

```
result = client.collect(amount: Money(50000, "ETB"), idempotency_key: "INV-2291")

switch result.status:
  case RequiresAction, Processing:
    switch result.next_action.type:
      case RedirectToUrl:       redirect(result.next_action.url)
      case AwaitDevicePush:     show("Approve the prompt sent to " + result.next_action.display_ref)
      case SubmitOtp:           promptForOtp(result.next_action.length)
      case DisplayQr:           renderQr(result.next_action.payload)
      case ShowTransferDetails: showAccount(result.next_action.account_number, result.next_action.amount)
      case DialUssd:            show("Dial " + result.next_action.code)
      case Poll:                scheduleSync(result.next_action.interval_ms)
      case Capture, None:       // no further user action; see spec/01-domain-model.md#nextaction
  case Succeeded:
    // done
  case Failed:
    // inspect result.failure.retry_class before deciding whether to retry
  case Canceled, Expired:
    // terminal, no retry
```

Every field the switch above reads (`.url`, `.display_ref`, `.payload`,
`.code`, ...) is on `next_action` itself, never on `result.state` — see
[01-domain-model.md#nextaction-carries-its-own-payload](../spec/01-domain-model.md#nextaction-carries-its-own-payload).
`state` is opaque adapter bookkeeping your integration never reads.
[docs/tutorials/use-a-provider/TEMPLATE.md](tutorials/use-a-provider/TEMPLATE.md)
and
[docs/examples/integrator/reference-integrator.md](examples/integrator/reference-integrator.md)
work through this exact switch, and the pattern for reaching a terminal
status safely across a return handler, a webhook, and a sweeper (the
**settler pattern**), in full.

## Handling `Processing` + `Poll`

This is not a special case to work around: it's the deliberate shape of
"we don't know the outcome yet," including after a network timeout on
*your* side (see [Invariant I4](../spec/02-invariants.md#i4), which
exists specifically to prevent a naive retry from double-charging a
payer). The correct response is always the same: call `sync()` again
later, or wait for a webhook, and only decide about retrying once you
have a definitive status.

## Retry logic

Never retry a `Failed` result without checking `result.failure.retry_class`:

- `DoNotRetry`: the outcome is settled and retrying won't help (e.g.
  insufficient funds, invalid request).
- `SafeToRetry`: a transient provider-side issue; retrying the same
  idempotency key is safe.
- `ResolveFirst`: the outcome is genuinely unknown (this is what a
  transport timeout maps to). Resolve it via `sync()` before doing
  anything else with this reference.

Your SDK's retry policy (backoff, how many attempts) is something you
configure at the SDK level; it is never a property of any individual
provider.

## Webhooks

Configure your webhook endpoint once, at the SDK/runtime level, using
whatever credential fields the provider's `CredentialShape` requires
(your SDK's setup docs will list them per provider). Verification and
normalization happen inside the SDK: you receive the same
`PaymentStatus`/`NextAction` shape from a webhook callback as from a
direct `collect`/`sync` call, keyed to the same idempotency reference.

## Idempotency keys

Always pass one, and treat it as the single identifier you use to look up
a payment's status later, regardless of which provider processed it.
Reusing a key with the exact same request replays the original result;
reusing it with a *different* request is rejected as a
`DuplicateRequest` error, not silently accepted: see
[Invariant I7](../spec/02-invariants.md#i7).

## Testing offline

Configure the `mock` provider ([providers/mock/](../providers/mock/))
in any non-production environment. It needs no credentials and no
network access, and can produce every `PaymentStatus`/`NextAction`
combination on demand by the `method` value you pass: see
`providers/mock/manifest.yaml`'s comments for the full list of scenario
values. This is the same provider every SDK's own test suite is built
against, so it's exercised at least as thoroughly as any real provider.
