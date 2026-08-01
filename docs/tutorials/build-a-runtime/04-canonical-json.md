# 4. Canonical JSON

This chapter has to come before chapter 8 (replay against `mock`) for a
structural reason, not just a pedagogical one: golden-output comparison
*is* canonical-JSON comparison. If your serializer isn't proven correct
first, a replay failure in chapter 8 could be a real interpreter bug or
just a formatting difference, and you won't be able to tell which
without stepping through it by hand every time. Prove serialization
first, and a chapter 8 mismatch is unambiguously about logic, not
formatting.

## The rules

[06-conformance.md#canonical-json](../../../spec/06-conformance.md#canonical-json)
states six rules; the two most likely to bite a naive implementation:

- **Integers are never rendered as floats.** A JSON encoder that
  "helpfully" normalizes an integer-valued number to `10000.0` or `1e4`
  has silently violated [Invariant I1](../../../spec/02-invariants.md#i1)
  at the serialization layer, even if the in-memory value was a correct
  integer the whole time. Several general-purpose JSON libraries do this
  by default for anything that passed through a floating-point-typed
  intermediate at any point — check yours specifically, don't assume.
- **Object keys are sorted byte-wise (by UTF-8 code unit) at every
  nesting level**, independently per object — not just at the top level,
  and not by your language's default locale-aware string comparison,
  which can disagree with byte-wise order for non-ASCII keys. A `Money`
  value's own two keys are one level of this you'll hit immediately;
  [`vectors/canonical-json/cases.json`](../../../vectors/canonical-json/cases.json)'s
  non-ASCII-key cases are there because this is exactly the kind of thing
  that passes every test written with only English field names and fails
  the first time a manifest or a payer's name uses Amharic script.

Also: no insignificant whitespace anywhere (the entire document is one
line), standard JSON string escaping (forward slash unescaped, non-ASCII
emitted as literal UTF-8 rather than forced to `\uXXXX`, control
characters/quotes/backslashes still escaped per ordinary JSON rules —
"no HTML-escaping" doesn't mean "no escaping"), and timestamps rendered
as fixed second-precision UTC with a literal `Z` (`2026-01-15T09:30:00Z`,
never fractional seconds, never a `+00:00` offset form).

## The test

```text
for each case in vectors/canonical-json/cases.json.cases:
    assert your_canonicalize(case.input) == case.expected
```

Build `your_canonicalize` to operate on your language's own generic
JSON-value representation (whatever your JSON library decodes into —
a map/dict of string to value, recursively), not on a typed
`PaymentResult` struct directly. That's what lets the same function
canonicalize a `PaymentResult`, a nested `Money`, and a bare test value
from this vector file identically, and it's what chapter 7's step
executor will eventually feed a `PaymentResult` through unchanged.

## What "done" looks like for this chapter

One function, `canonicalize(value) -> string`, passing every case in
`vectors/canonical-json/cases.json` byte-for-byte — including the
int64-boundary case (cross-check against chapter 2's `Money` work: the
same boundary value should canonicalize identically whether it arrived
as a `Money.minor_units` or a bare test integer). Nothing here has read a
manifest or a cassette yet.
