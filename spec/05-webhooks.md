# 05: Webhooks

Most Ethiopian PSPs confirm a payment's outcome asynchronously, via an
inbound HTTP callback, rather than (or in addition to) the synchronous
response to the initiating call. A manifest's `webhook` section declares
how to verify that an inbound request genuinely came from the provider and
how to turn its body into the same `status`/`next_action`/`state` shape
every other step produces.

## Verification

```yaml
webhook:
  verification:
    scheme: hmac_sha256_hex
    signature_header: Chapa-Signature
    secret: ${credentials.secret_key}
  body_format: json
```

`scheme` names one of a closed set of verification schemes:

| Scheme | Behaviour |
|---|---|
| `hmac_sha256_hex` | HMAC-SHA256 over the raw request body, secret from `credentials`, compared against a hex-encoded signature in `signature_header`. |
| `hmac_sha256_base64` | Same, but the signature is base64-encoded. |
| `none` | No signature verification is possible for this provider (must be justified in `metadata.yaml` verification notes; this is a red flag on a real, non-`mock` provider and should be rare). |

Field transforms in [03-manifest-dsl.md](03-manifest-dsl.md#field-transforms)
(`hex`, `base64`, `sha256_hex`) exist in part so a manifest can express
these schemes declaratively without inventing a webhook-specific mechanism.

### The rule that matters most here

Per [Invariant I11](02-invariants.md#i11), verification **must**:

1. Compute the signature over the **raw bytes of the request body exactly
   as received**, before any JSON parsing, key reordering, or whitespace
   normalization. A runtime that verifies against `json.dumps(parsed_body)`
   will reject legitimate webhooks the moment the provider's JSON
   serialization differs from the runtime's in any whitespace or key-order
   detail, and will *pass* a forged request if an attacker can produce a
   reparsed body with the same logical content but a different raw
   encoding along with a signature computed the same broken way. Runtimes
   must capture and hold the raw body specifically for this comparison,
   separately from whatever they hand to the JSON parser for extraction.
2. Compare the computed signature against the received one using a
   **constant-time** comparison (e.g. `hmac.compare_digest` in Python,
   `crypto/subtle.ConstantTimeCompare` in Go, `hash_equals` in PHP;
   every mainstream language ships one; a runtime must use it, not `==`).
   A non-constant-time comparison leaks, via response timing, how many
   leading bytes of an attacker's guess matched the real signature, which
   is a textbook forgery side-channel.
3. Reject the request (map to a `FailureCode` your runtime surfaces as a
   webhook-processing error, never silently drop or silently accept) when
   the signature header is **missing entirely**: a missing header is not
   "unverified, proceed anyway," it's a rejection.

[vectors/webhook/](../vectors/webhook/) contains real precomputed HMAC
signatures (both hex and base64) over fixed raw bodies, a tampered-body
case that must fail verification, and a missing-header case that must be
rejected; every runtime's conformance suite runs against all of them.

## Body format and normalization

`body_format` tells the runtime how to parse the (already-verified) body
for extraction purposes: currently only `json` is defined. Once verified,
the parsed body becomes the `event` namespace
([04-expression-language.md](04-expression-language.md)) for the
`webhook`-triggered flow step, which then extracts fields from it and
`status_map`s exactly like any other step reading a `call` response.

## How a webhook step fits into a flow

A `webhook` operation's `flows` entry point is triggered by
`triggers: [webhook]`, not by a `call`. There is no outbound HTTP request
for this step: the inbound request itself is the input. Everything else
about the step (extraction from `event`, `status_map` to the next step,
`emit`) works exactly as it does elsewhere in the manifest DSL, which is
the point: webhooks are not a special case bolted onto the flow model,
they're one more trigger kind within the same state machine, and a runtime
implementer who has already built `call`/`return` handling has almost all
of the machinery `webhook` needs.

## Webhooks do not replace polling

A provider that supports webhooks is not thereby exempt from also
supporting `sync`. Webhook delivery is not guaranteed (providers drop
retries, integrators misconfigure endpoints, networks partition) and
[Invariant I4](02-invariants.md#i4) already commits this spec to treating
unresolved outcomes as `Processing` + `Poll` rather than assuming a webhook
will eventually arrive. Every provider manifest that declares `webhook`
capability must still declare and cassette-cover `sync` as an independent,
authoritative way to resolve the same payment.

## Idempotency of webhook delivery

Providers may (and do) redeliver the same webhook more than once, on their
own retry schedule, not the runtime's. Because [Invariant I2](02-invariants.md#i2)
makes terminal `PaymentStatus` values non-transitioning, processing the
same webhook body twice for a payment already in a terminal state must be
a no-op that returns the existing recorded outcome, never an error and
never a second side effect. A manifest does not need to express this
explicitly: it falls out of I2 plus the runtime's obligation
([07-runtime-requirements.md](07-runtime-requirements.md)) to treat every
inbound webhook as a status update against an existing idempotency-keyed
record, not as a new event to append to a log.
