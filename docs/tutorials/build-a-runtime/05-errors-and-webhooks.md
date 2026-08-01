# 5. Error mapping and webhook verification

Two independent subsystems, grouped into one chapter because both are
pure-ish functions over vector data, and both are done once you've
passed their vectors — no manifest, no interpreter, no real network
call, needed for either.

## Error mapping

You already loaded the fixed `FailureCode -> RetryClass` table in
chapter 2. This chapter is the three **match kinds** a manifest's
`errors[]` list uses to decide *which* `FailureCode` applies to a given
response, and how they interact with a step's own `status_map`
([03-manifest-dsl.md#errors](../../../spec/03-manifest-dsl.md#errors)):

- **`path`/`equals`**: an extraction path (chapter 3) compared for
  equality against a literal.
- **`http_status`**: matched against the raw HTTP status code, entirely
  independent of the response body — including when the body is absent
  or unparseable.
- **`transport`**: matched when no response was received at all (a
  timeout or connection reset), not against anything in a response,
  because there is no response.

[`vectors/errors/match-kinds.json`](../../../vectors/errors/match-kinds.json)
exercises all three and, more importantly, the **precedence rule**:
`errors` is checked before a step's own `status_map`, against the same
response, and a match short-circuits that step's `emit` entirely — no
`status`, no `next_action`, no `state` from that step applies, even if
the body would also have matched a `status_map` key. A response matching
no `errors` entry is the only case that falls through to `status_map`.
[`vectors/errors/failure-code-cases.json`](../../../vectors/errors/failure-code-cases.json)
gives you one worked example per `FailureCode`, if you want a case to
reason from for each.

The one case worth its own paragraph:
[Invariant I4](../../../spec/02-invariants.md#i4) — a `transport: timeout`
match, or the transport-level absence of any response, must **always**
resolve to `Processing` + `NextAction.Poll`, never `Failed`, regardless
of what a manifest's own `transport: timeout` entry declares (that entry
only pins the `FailureCode`/`RetryClass` pair reported alongside it; the
`PaymentStatus`/`NextAction` outcome for this specific case is fixed by
the spec, not by any manifest).
[`vectors/errors/timeout-classification.json`](../../../vectors/errors/timeout-classification.json)
is the vector for this specific rule.

```text
for each case in vectors/errors/match-kinds.json.cases:
    match = find_matching_error(manifest_errors_under_test, case.http_status, case.body, case.transport_event)
    assert match_index_of(match) == case.expected_match_index
    assert (match.failure_code if match else null) == case.expected_failure_code
```

## Webhook verification

[05-webhooks.md](../../../spec/05-webhooks.md) and
[Invariant I11](../../../spec/02-invariants.md#i11): verify against the
**raw, unparsed request body** — never a re-serialized form of it, since
any whitespace or key-ordering difference between what the provider
signed and what got re-serialized breaks verification for a legitimate
request — using a **constant-time comparison** for the signature itself.
Almost every mainstream language ships a constant-time byte-compare
function in its standard crypto library already (`hmac.compare_digest`,
`crypto.timingSafeEqual`, `subtle.ConstantTimeCompare`, or equivalent);
use it. Do not write your own equality loop for this comparison, even a
"careful" one — timing side-channel resistance is easy to defeat by
accident and hard to verify by eye.

[`vectors/webhook/hmac-sha256.json`](../../../vectors/webhook/hmac-sha256.json)
covers: a valid hex signature, a valid base64 signature, a tampered body
(signature computed over different bytes than what's received), a
missing signature header, a reordered-but-logically-equal body (the case
that specifically proves you're comparing raw bytes, not re-serialized
JSON), a signature computed with the wrong algorithm entirely, malformed
signature encodings (invalid hex/base64), and `scheme: none` always
trivially verifying (only meaningful for a provider with no signing
mechanism, like `mock`; a red flag if declared for any real provider —
see that vector file's own `scheme_none_always_verifies` case).

```text
for each case in vectors/webhook/hmac-sha256.json (each named case):
    result = verify(case.scheme, case.secret, case.received_body ?? raw_body, case.signature_header, case.signature)
    assert result == case.expected_result
```

A missing signature header is a rejection, never an implicit pass. A
signature value that fails to decode per its scheme (not valid hex, not
valid base64) is rejected the same way an incorrect-but-well-formed
signature is — a decoding failure is not a third outcome, and must never
short-circuit past the comparison into an accept.

## What "done" looks like for this chapter

Two functions: `classify_error(manifest_errors, response) -> FailureCode
| null` and `verify_webhook(scheme, secret, raw_body, header_value) ->
bool`, each passing their vector files completely. Still no manifest
loader, no interpreter, no real HTTP.
