# 04: Expression Language

Manifests need a way to pull values out of provider responses, plug values
into requests, and branch on extracted values. ESIIPayment defines exactly
one such language, and it is deliberately small enough that a runtime
implementer can finish it in an afternoon. Every runtime must implement
**exactly** this: nothing more, nothing less. A runtime that supports a
superset (its own convenience additions) has silently forked the DSL: a
manifest written against that superset works on one runtime and fails
validation on every other.

**Where this document and the vectors in
[vectors/expressions/](../vectors/expressions/) disagree, the vectors win.**
Prose is written by people and people are imprecise; the vectors are the
executable definition. If you find a case this document doesn't cover,
check the vectors before guessing, and if the vectors don't cover it
either, that's a spec gap to raise, not a case to improvise a behaviour for.

<a id="extraction--a-jsonpath-subset"></a>

## Extraction: a JSONPath subset

Extraction paths read a value out of a JSON document (a provider response
body, a webhook payload). The supported grammar is intentionally a small
subset of JSONPath:

| Form | Meaning | Example |
|---|---|---|
| `$.a.b` | Object field access, arbitrarily nested | `$.data.checkout_url` |
| `$.a[0]` | Array index access, zero-based | `$.data.items[0]` |
| `$.a[*].b` | Map over every element of an array, projecting field `b` | `$.data.items[*].status` |

Nothing else is supported:

- **No filter expressions** (`$.a[?(@.b == 'x')]`). A manifest that needs to
  find "the item whose status is X" cannot express that as a path; the
  provider response shape has to be simple enough that the step doesn't
  need it, or the flow needs another step.
- **No recursive descent** (`$..b`). Every path names its full route from
  the document root.
- **No scripts, functions, or arithmetic inside a path.** A path only ever
  navigates; it never computes. Computation, where the closed transform set
  allows it, happens in interpolation (below), not extraction.

### Missing-path behaviour

If a path does not resolve (a field is absent, an index is out of range,
or an intermediate node is `null` or not the expected type), extraction
yields an explicit "missing" result, not an error and not a silent `null`
coerced into a string. A manifest step that requires a path to resolve
(most `emit.state` assignments do) must fail loudly if it doesn't; a step
that tolerates absence must say so explicitly rather than relying on a
runtime's ad hoc null-handling. See
[vectors/expressions/extraction.json](../vectors/expressions/extraction.json)
for the exact missing-path vectors every runtime must match: this is one
of the easiest places for runtimes to quietly diverge (one raises, one
returns `null`, one returns an empty string) so it is fully vectorized
rather than left to prose.

## Interpolation

`${...}` interpolates a value into a string field of a manifest (a request
body, header, path segment, or a `state`/`emit` assignment). The braces
contain either a bare namespaced path, or a single transform call wrapping
one:

```
${namespace.path}
${transform(namespace.path)}
```

Namespaces available at interpolation time:

| Namespace | Contents |
|---|---|
| `credentials` | Concrete secret values the runtime injected for this provider (never present in the manifest source itself; [Invariant I9](02-invariants.md#i9)). |
| `ctx` | Runtime-supplied context: active environment's `base_url`, `webhook_url`, and similar deployment-level values. |
| `intent` | The integrator's original request: `amount`, `currency`, `method`, and similar collect/payout inputs. |
| `state` | Values a prior step in this flow wrote via `emit.state` ([Invariant I6](02-invariants.md#i6)). |
| `extract` | Values pulled from the current step's HTTP response or webhook body via an extraction path. |
| `input` | A value the integrator supplied mid-flow in response to a `triggers: [input]` step (e.g. an OTP digit string). |
| `event` | The current inbound webhook's parsed body, available only inside `webhook`-triggered steps. |
| `idempotency_key` | The idempotency key of the current operation. |

Not every namespace is available at every point in a flow: `extract` only
exists once a `call` or `webhook` trigger has produced a response to
extract from, `event` only exists inside a webhook step, `input` only
exists in a step reached via an `input` trigger. `esiipayment validate` checks
statically that a manifest never interpolates a namespace that cannot be
populated yet at that point in its flow.

### Transforms

Interpolation may wrap a path in **exactly one** of the closed transforms
defined in [03-manifest-dsl.md](03-manifest-dsl.md#field-transforms):
`amount_major`, `msisdn_et`, `base64`, `hex`, `sha256_hex`, `upper`,
`lower`, `iso8601`. Transforms do not compose (`${upper(base64(...))}` is
not valid) and there is no mechanism to define a new one in a manifest;
see that section for why.

## Conditions

Conditions are used in `status_map` matching and in `errors` matching.
A condition is exactly one comparison between one extraction path and one
literal:

| Operator | Meaning |
|---|---|
| `==` | Equal |
| `!=` | Not equal |
| `>` | Greater than (numeric) |
| `<` | Less than (numeric) |
| `in` | Path's value is a member of a literal list |

There is no boolean composition (`&&`, `\|\|`), no negation of a compound
expression, and no function calls inside a condition. If a manifest step
seems to need "status is X and amount is over Y," that is two separate
concerns that belong in two separate steps or two separate `errors`
entries, not one compound condition: folding boolean logic into
conditions is exactly the slope that turns a data format into a
programming language reimplemented five times, once per runtime.

## Determinism

Every extraction, interpolation, and condition evaluation must be a pure
function of its inputs (the response/webhook body, and the current
`credentials`/`ctx`/`intent`/`state`/`input`/`idempotency_key` values),
never of ambient system state such as the wall clock or a random number
generator. Where a flow genuinely needs the current time or a fresh
identifier, those are supplied through the same seeded-clock/seeded-UUID
mechanism cassettes use ([03-manifest-dsl.md](03-manifest-dsl.md),
[06-conformance.md](06-conformance.md)), never read directly from the
runtime's operating environment. This is what makes golden-output
comparison meaningful: given the same cassette and the same seed, the
expression language must evaluate identically on every runtime, every time.
