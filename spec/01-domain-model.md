# 01: Domain Model

This document is the single source of truth for ESIIPayment's value types and
enums. Every language runtime must represent these exactly: same members,
same names (or the idiomatic in-language equivalent, e.g. an enum case named
`RequiresAction` in C# and `requires_action` in Python refer to the same
member), same closedness.

**All enums listed here are closed.** A runtime or adapter must never add a
member. Adding, renaming, or removing a member is a breaking change to the
spec and requires an RFC (see [GOVERNANCE.md](../GOVERNANCE.md)). Treat an
unrecognized value arriving from a provider as a mapping bug in the adapter,
not as license to invent a new enum member: map it to the closest existing
member (usually `FailureCode.Unknown` or `NextAction.Poll`) and let the
runtime log the raw provider value for diagnosis.

## Money

```
Money {
  minor_units: int64   // signed
  currency:    string  // ISO-4217 alphabetic code, e.g. "ETB", "USD"
}
```

- Money is **always** a signed 64-bit integer count of the currency's minor
  unit, plus its ISO-4217 code. There is no floating-point representation of
  money anywhere in the model, in any manifest, in any vector, or in any
  runtime's public API. Floating point cannot represent decimal fractions
  exactly, and a payment system that occasionally loses or gains a santim is
  not a payment system.
- The minor-unit exponent (how many minor units make one major unit) is
  **per-currency, not assumed**. ETB has exponent 2 (1 ETB = 100 santim),
  matching the ISO-4217 default for most currencies, but the exponent is
  data, not a hardcoded constant: some ISO-4217 currencies use 0 or 3.
  Runtimes must read the exponent from
  [vectors/money/exponents.json](../vectors/money/exponents.json) rather than
  from platform locale data, which is inconsistent across languages and
  operating systems.
- Signed: negative `Money` values are legal and are used for refunds and
  reversals in the flow state; adapters never need to encode sign
  information out-of-band.
- Comparison, addition, and subtraction of `Money` are only defined between
  values of the same currency. Cross-currency arithmetic is a runtime error,
  not a silent conversion; this spec has no opinion on FX.

## PaymentStatus

Exactly six members:

| Member | Terminal? | Meaning |
|---|---|---|
| `RequiresAction` | no | The end user must do something before the payment can proceed (redirect, OTP, USSD, etc.; see `NextAction`). |
| `Processing` | no | The payment is in flight. This includes the case where the outcome is genuinely unknown, e.g. after a transport failure; see [Invariant I4](02-invariants.md#i4). |
| `Succeeded` | **yes** | The payment completed and funds are confirmed. |
| `Failed` | **yes** | The payment was confirmed not to complete, and will not be retried under this idempotency key. |
| `Canceled` | **yes** | The end user or integrator explicitly canceled the payment. |
| `Expired` | **yes** | The payment window elapsed before completion. |

`Succeeded`, `Failed`, `Canceled`, and `Expired` are **terminal**: once a
payment reaches one of these four states under a given idempotency key, it
must never transition to any other state, including another terminal one.
A runtime that observes a provider report a different terminal state for
the same key after the fact has caught a provider bug or a replay-attack
attempt, not a legitimate transition, and must treat it as a fatal
consistency error rather than overwrite the record.

### Why there is no `Indeterminate`

Every other payment library eventually invents an `Indeterminate` or
`Unknown` status for "we don't know what happened yet," usually after a
network timeout. ESIIPayment deliberately has no such member.

An unknown outcome is `Processing` plus `NextAction.Poll`. This is not a
simplification for its own sake: an `Indeterminate` status is a second
return type in disguise. Every integrator who receives it has to write a
special branch that doesn't fit the `Succeeded`/`Failed`/"do something and
come back" shape the rest of the API follows, and that branch's correct
behaviour is always "poll again later" anyway. Folding the unknown case into
`Processing` + `Poll` means callers only ever need to handle one shape:
either the payment needs the user to do something, or it needs the
integrator to check back. There is no third thing to special-case, and no
temptation to short-circuit the resolution path with a guess. See
[Invariant I4](02-invariants.md#i4) for the transport-failure case this is
built for.

## NextAction

Exactly nine members. Each describes what the **end user** must do next,
never what the provider's API requires internally (that belongs in the
manifest's `flows`, not in the value the integrator sees).

| Member | End-user obligation |
|---|---|
| `None` | Nothing further; wait for the payment to resolve with no user action (rare; most non-terminal states pair with a concrete action). |
| `RedirectToUrl` | Visit a URL (hosted checkout page, 3-D Secure challenge, etc.). |
| `AwaitDevicePush` | Approve a push/prompt already sent to a registered device (e.g. mobile money STK push, banking app approval). |
| `SubmitOtp` | Enter a one-time code sent via SMS or app. |
| `DisplayQr` | Scan a QR code with a wallet or banking app. |
| `ShowTransferDetails` | Perform a manual bank/wallet transfer to displayed account details (virtual account, static account number). |
| `DialUssd` | Dial a USSD string on the registered phone. |
| `Poll` | No user action; the integrator must check status again after a delay. Paired with `Processing` for in-flight or indeterminate outcomes. |
| `Capture` | A separate, explicit capture step is required to finalize funds already authorized. |

### `NextAction` carries its own payload

`NextAction` is not a bare label. Every non-`null` `next_action` on a
`PaymentResult` is a **typed object** with a discriminating `type` field
(one of the nine members above) plus that variant's own fields:

```json
{"type": "RedirectToUrl", "url": "https://pay.example/checkout/abc123"}
```

The fields below are **required** or **optional** per variant, closed:
a manifest may not invent an additional field on a variant, and a runtime
must never expose provider-specific data on `next_action` under any other
key. This is the fix for the failure mode this design otherwise invites:
without a fixed payload shape, one provider emits `checkout_url` and
another `payment_url` for the same `RedirectToUrl` case, both stuffed into
the opaque `state` bag, and an integrator ends up writing
`state.checkout_url ?? state.payment_url` — branching on provider identity
in practice even though [Invariant I12](02-invariants.md#i12) forbids it
in principle. A manifest author maps the provider's own field name to the
canonical one on the left-hand side once, in the manifest; every
integrator-facing consumer reads the same key regardless of provider.

| Variant | Required fields | Optional fields |
|---|---|---|
| `None` | — | — |
| `RedirectToUrl` | `url: string` | `method: string` (`GET` or `POST`; the HTTP method the redirect target expects, default `GET`) |
| `AwaitDevicePush` | `display_ref: string` (a value safe to show the user identifying which device/session the push went to, e.g. a masked phone number) | `expires_at: string` (ISO-8601 timestamp) |
| `SubmitOtp` | `length: integer` (number of digits/characters the user must enter) | `hint: string` (e.g. "sent to 251911***567"), `expires_at: string` |
| `DisplayQr` | `payload: string` (the raw QR content to render) | `image_url: string` (a provider-hosted pre-rendered QR image, if available instead of/alongside rendering `payload` client-side), `expires_at: string` |
| `ShowTransferDetails` | `account_number: string`, `institution: string`, `reference: string` (value the payer must enter as the transfer memo/reference so the provider can match it), `amount: Money` | `account_name: string`, `expires_at: string` |
| `DialUssd` | `code: string` (the full USSD string to dial, e.g. `*899*1*0001#`) | `expires_at: string` |
| `Poll` | `interval_ms: integer` (suggested delay before the next `sync`) | `not_before: string` (ISO-8601 timestamp; do not poll before this time) |
| `Capture` | — | — |

`expires_at`, where present, is always an ISO-8601 UTC timestamp in the
canonical form from [06-conformance.md](06-conformance.md#canonical-json).
`amount` on `ShowTransferDetails` is a full `Money` value (`minor_units` +
`currency`), never a bare number, for the same reason [I1](02-invariants.md#i1)
requires it everywhere else.

A manifest's `flows` populate these fields via `emit.next_action` (see
[03-manifest-dsl.md](03-manifest-dsl.md#flows)); `esiipayment validate`
rejects a step that emits a variant missing one of its required fields, or
carrying a field that variant does not define.

#### Why `interval_ms` is required, even for a runtime-synthesized `Poll`

Most `Poll` actions come from a manifest step's own `emit`, which supplies
`interval_ms` like any other field. The one exception is the `Poll` a
runtime synthesizes itself rather than reads from a manifest: the
transport-failure case ([Invariant I4](02-invariants.md#i4)) and the
`ProviderTimeout`/`Unknown` error short-circuit
([03-manifest-dsl.md](03-manifest-dsl.md#how-errors-interacts-with-status_map)).
Neither has a provider-declared interval to read. A runtime must use a
fixed default of **5000** (five seconds) in both cases rather than
inventing its own value or its own backoff policy at this layer: retry
*policy* is the runtime's to configure per [Invariant I5](02-invariants.md#i5),
but this specific field is part of a byte-comparable golden output, so the
one value used when no manifest supplies one has to be fixed by the spec,
not left to each runtime's discretion.

### Mapping table: provider behaviour → `NextAction`

Adapter authors use this table to decide which variant a given provider flow
step should `emit`. It is not exhaustive of every possible provider UX, but
every Ethiopian PSP flow this project has encountered so far reduces to one
of these.

| Provider-observed behaviour | `NextAction` |
|---|---|
| Hosted checkout page (Chapa-style redirect URL) | `RedirectToUrl` |
| 3-D Secure card challenge redirect | `RedirectToUrl` |
| STK push / prompt to a mobile wallet app | `AwaitDevicePush` |
| In-app payment approval notification | `AwaitDevicePush` |
| SMS one-time-password required to confirm | `SubmitOtp` |
| QR code returned for scan-to-pay | `DisplayQr` |
| Static or dynamic virtual account number issued for bank transfer | `ShowTransferDetails` |
| USSD code the payer must dial from their handset | `DialUssd` |
| Provider has accepted the request but has not yet confirmed outcome | `Poll` |
| Provider requires a distinct authorize-then-capture call | `Capture` |
| Fully synchronous success/failure with nothing left for the user to do | `None` |

## FailureCode

Exactly fourteen members:

| Member | Meaning |
|---|---|
| `InsufficientFunds` | Payer's account/wallet lacked sufficient balance. |
| `InvalidRecipient` | The receiving account/merchant identifier was invalid (payout direction). |
| `RecipientLimitExceeded` | The recipient has hit a provider- or regulator-imposed limit. |
| `SenderLimitExceeded` | The payer has hit a provider- or regulator-imposed limit (daily/transaction cap, KYC tier). |
| `DuplicateRequest` | The provider rejected the request as a duplicate of a prior one. |
| `AuthFailed` | The adapter's own credentials were rejected (bad API key, expired token, bad signature). |
| `AuthorizationDeclined` | The payer's issuer/wallet declined authorization (distinct from `InsufficientFunds` when the provider reports a decline without a specific reason). |
| `ProviderUnavailable` | The provider's system was unreachable or returned a server error unrelated to this specific request. |
| `ProviderTimeout` | The request was sent but no response was received before the transport deadline. |
| `InvalidRequest` | The request was malformed per the provider's own validation (not an ESIIPayment manifest bug). |
| `UnsupportedOperation` | The requested operation is not supported by this provider/method combination. |
| `Expired` | The payment window elapsed. |
| `CanceledByUser` | The payer explicitly canceled. |
| `Unknown` | The provider reported a failure the adapter cannot map to any of the above; the raw provider code must still be preserved in flow state for diagnostics. |

## RetryClass

Exactly three members: `SafeToRetry`, `DoNotRetry`, `ResolveFirst`.

The `FailureCode → RetryClass` mapping is **fixed by this repository** and
is not configurable per adapter, per provider, or per integrator. An adapter
declares which `FailureCode` a given provider error maps to; this spec alone
decides what retrying that code means.

| FailureCode | RetryClass |
|---|---|
| `InsufficientFunds` | `DoNotRetry` |
| `InvalidRecipient` | `DoNotRetry` |
| `RecipientLimitExceeded` | `DoNotRetry` |
| `SenderLimitExceeded` | `DoNotRetry` |
| `DuplicateRequest` | `DoNotRetry` |
| `AuthFailed` | `DoNotRetry` |
| `AuthorizationDeclined` | `DoNotRetry` |
| `ProviderUnavailable` | `SafeToRetry` |
| `ProviderTimeout` | **`ResolveFirst`** |
| `InvalidRequest` | `DoNotRetry` |
| `UnsupportedOperation` | `DoNotRetry` |
| `Expired` | `DoNotRetry` |
| `CanceledByUser` | `DoNotRetry` |
| `Unknown` | **`ResolveFirst`** |

This exact mapping is also encoded machine-readably in
[vectors/errors/failure-retry-map.json](../vectors/errors/failure-retry-map.json)
and every runtime's conformance suite must assert its retry logic reads
from that table (or an exact copy of it) rather than hardcoding a
divergent mapping.

### Why `ProviderTimeout` is not `SafeToRetry`

This is the single highest-severity bug class this system is designed to
prevent, so it is called out explicitly rather than left to be inferred
from the table: **a timeout means the request may or may not have been
processed by the provider.** The provider may have received the request,
debited the payer, and simply failed to return a response before the
adapter's deadline. Blindly retrying turns one customer-authorized payment
into two. `ResolveFirst` means: before taking any further action under this
idempotency key, the caller must resolve the actual outcome (via a status
query / `sync` operation, or a webhook) and only then decide whether a
retry is even meaningful. The same reasoning applies to `Unknown`: an
unmapped provider failure is an unknown outcome, not a confirmed one, and
must not be treated as safely retryable by default.

Mapping `ProviderTimeout` to `SafeToRetry` in any runtime or adapter is
considered a critical defect regardless of how it got there; see
[Invariant I4](02-invariants.md#i4) and
[Invariant I10](02-invariants.md#i10).

## CredentialShape

Exactly seven members, closed:

| Member | Shape |
|---|---|
| `none` | No credentials required (e.g. the `mock` provider). |
| `api_key` | A single bearer/API key sent as a header or query param. |
| `key_pair` | A public/private key pair (e.g. a merchant ID plus a private signing key). |
| `triple_key` | Three distinct credential values used together (some Ethiopian PSPs issue a merchant key, a public key, and a secret key as separate values). |
| `oauth2_client_credentials` | OAuth2 client-credentials grant; adapter exchanges client id/secret for a bearer token. |
| `hmac` | A shared secret used to compute a request or webhook signature; never sent over the wire itself. |
| `mutual_tls` | Client-certificate authentication at the TLS layer. |

Adapter authors pick exactly one `CredentialShape` per provider in `auth`
and never invent a new shape, even if a provider's real scheme is a minor
variant of one of these. This closed set is what lets every runtime
generate a uniform credential-input form, validate credential completeness
generically, and handle secret storage the same way regardless of which
provider is configured; see
[Invariant I9](02-invariants.md#i9) (adapters receive credential
*capabilities*, never key material, in the runtime; `CredentialShape` only
describes what shape of secret the integrator must supply out of band).

## Operation

Exactly six members: `collect`, `sync`, `payout`, `cancel`, `refund`,
`webhook`.

| Member | Meaning |
|---|---|
| `collect` | Initiate collecting a payment from a payer. |
| `sync` | Query/poll the current status of a previously initiated payment. |
| `payout` | Send funds to a recipient (disbursement). |
| `cancel` | Explicitly cancel a non-terminal payment. |
| `refund` | Reverse a previously succeeded `collect`. |
| `webhook` | Receive and interpret an inbound provider callback. |

A provider manifest declares which operations it supports via
`capabilities`, and must supply a `flows` entry and cassette coverage (see
[03-manifest-dsl.md](03-manifest-dsl.md)) for every operation it declares.

## PaymentResult

Every flow execution (replaying a cassette or running for real) produces
exactly this shape. This is the object [canonical JSON](06-conformance.md)
rules apply to, and the object every `expected/<name>.json` file is.

```
PaymentResult {
  idempotency_key: string
  operation:        Operation
  status:           PaymentStatus
  next_action:      NextAction | null   // NextAction is {type, ...fields}; see above
  state:            object   // the flow's accumulated state; {} if empty
  failure:          null | { failure_code: FailureCode, retry_class: RetryClass }
}
```

- `next_action` is `null` for a **terminal** `status` (`Succeeded`,
  `Failed`, `Canceled`, `Expired`): a terminal outcome has nothing next by
  definition, which is a distinct concept from `NextAction.None` (that
  variant pairs with a still-open `RequiresAction`/`Processing` status
  where there's deliberately nothing for the user to do yet, not with a
  finished payment). For a **non-terminal** `status`, `next_action` is
  always present and is the typed `{type, ...fields}` object for one of
  the nine `NextAction` members, per the payload table above.
- `failure` is non-null if and only if `status` is `Failed`, and carries
  exactly the `FailureCode`/`RetryClass` pair the matching manifest
  `errors` entry declared (per [Invariant I10](02-invariants.md#i10)). It
  is `null` for every other status, including the other three terminal
  ones: `Canceled` and `Expired` are not failures in the retry-logic
  sense and carry no `FailureCode`.
- `state` reflects whatever the flow's steps wrote via `emit.state`
  ([Invariant I6](02-invariants.md#i6)); it is `{}`, not omitted, when no
  step wrote anything. `state` is opaque adapter bookkeeping only (e.g. a
  provider-assigned session id a later `sync` call needs to address the
  same transaction) — it never carries data the integrator is meant to
  read to build their UI. Anything the end user needs is on `next_action`
  instead; see [Invariant I12](02-invariants.md#i12).
- A runtime's public API may shape its own language-idiomatic result type
  around this data (a discriminated union, a class hierarchy, whatever is
  idiomatic) as long as the same information is present; `PaymentResult`
  is the wire/golden-output shape cassettes and conformance tests compare
  against, not a mandated in-memory representation.
