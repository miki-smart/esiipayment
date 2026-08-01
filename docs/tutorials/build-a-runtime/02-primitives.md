# 2. Primitives

Everything in this chapter is a value type and a lookup table. No I/O, no
parsing, no manifest. If your language makes this chapter take more than
a day, something about the design is fighting the language rather than
using it; these are deliberately the simplest possible types
([01-domain-model.md](../../../spec/01-domain-model.md)).

## `Money`

```text
Money {
  minor_units: int64   // signed
  currency:    string  // ISO-4217 alphabetic code
}
```

Two rules that are easy to get right in the type and easy to get wrong
in the code that touches it later:

- **`minor_units` is always a 64-bit signed integer.** Never a
  floating-point type, anywhere `Money` flows — not internally, not at
  the JSON boundary. See
  [Invariant I1](../../../spec/02-invariants.md#i1).
- **The minor-unit exponent is per-currency data, not a hardcoded
  constant or platform locale lookup.** Read it from
  [`vectors/money/exponents.json`](../../../vectors/money/exponents.json)
  at startup (or generate a lookup table from it at build time); do not
  ask your language's locale/ICU facilities for it; do not assume every
  currency has exponent 2. `JPY` and Ethiopia's neighbors' zero-decimal
  currencies (`DJF`, `UGX`, `RWF`, ...) have exponent 0; `KWD`/`BHD`/`OMR`/
  `JOD`/`TND` have exponent 3.

Implement exactly one operation beyond the struct itself for now:
`amount_major(money) -> string`, the decimal-string rendering
[03-manifest-dsl.md#field-transforms](../../../spec/03-manifest-dsl.md#field-transforms)
describes. Test it as data, not as examples you invent:

```text
for each case in vectors/money/conversions.json:
    assert amount_major(Money{case.minor_units, case.currency}) == case.major
```

Pay particular attention to the exact-int64-boundary cases in that file
(`math.MinInt64`/`math.MaxInt64`-equivalent in your language). A naive
implementation that computes a value's magnitude as `-n` before dividing
overflows silently on the most negative representable integer, because
that value's magnitude has no positive representation in a same-width
signed integer — this is exactly the kind of bug that only an explicit
boundary vector catches; see that file's own vectors for why it's called
out by name instead of left to be "obviously fine."

## The closed enums

`PaymentStatus` (6 members), `NextAction` (9 members, each a required/
optional field set — see
[01-domain-model.md#nextaction-carries-its-own-payload](../../../spec/01-domain-model.md#nextaction-carries-its-own-payload)),
`FailureCode` (14 members), `RetryClass` (3 members), `CredentialShape`
(7 members), `Operation` (6 members). Represent each as whatever your
language's idiomatic closed-set construct is (an enum, a sealed class
hierarchy, a sum type) — the point is that adding a member must be a
change your compiler or a schema validator forces every call site to
acknowledge, not something a `switch` can silently ignore via a `default`
branch that swallows unrecognized values.

Two things to get right here that aren't obvious from the member list
alone:

- **The `FailureCode -> RetryClass` mapping is fixed and not
  per-adapter-configurable.**
  [`vectors/errors/failure-retry-map.json`](../../../vectors/errors/failure-retry-map.json)
  is the source of truth; load it (or a generated copy of it) rather than
  re-deriving the mapping from the member names. Get
  `ProviderTimeout -> ResolveFirst` and `Unknown -> ResolveFirst` right
  in particular:
  [01-domain-model.md#why-providertimeout-is-not-safetoretry](../../../spec/01-domain-model.md#why-providertimeout-is-not-safetoretry)
  explains why mapping either to `SafeToRetry` is the single
  highest-severity defect class this whole project exists to prevent.
- **An unrecognized value arriving from a provider is an adapter mapping
  bug, never license to invent a new enum member.** Map it to the closest
  existing member (usually `FailureCode.Unknown` or `NextAction.Poll`)
  and log the raw provider value for diagnosis; don't extend your own
  runtime's enum to "help."

## `PaymentStatus` transitions

[`vectors/status/transitions.json`](../../../vectors/status/transitions.json)
is the complete transition matrix: all 36 ordered pairs among the 6
`PaymentStatus` members, each marked legal or illegal. It's already
exhaustive — every pair, not a sample — so there's no judgment call for
your runtime to make here at all:

```text
for each transition in vectors/status/transitions.json.transitions:
    assert your_state_machine.allows(transition.from, transition.to) == transition.legal
```

The one-sentence version of what the matrix encodes: **every transition
out of a terminal status (`Succeeded`, `Failed`, `Canceled`, `Expired`)
is illegal, including to itself.** Your engine (chapter 9) is where this
actually gets enforced against a real persisted record; this chapter is
just making sure the pure state-machine rule is right before anything is
persisted at all.

## What "done" looks like for this chapter

A data-driven test suite, wired to the three vector files above, passing
against nothing but the primitive types themselves — no manifest loader,
no interpreter, no network stack exists yet. If any of these tests need
more than that to run, stop and reconsider what you've coupled together.
