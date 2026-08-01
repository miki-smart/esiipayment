# 3. The expression evaluator

[04-expression-language.md](../../../spec/04-expression-language.md) is
short on purpose: "a runtime implementer can finish it in an afternoon."
That brevity is the whole point, and it cuts both ways for you as an
implementer:

> A runtime that supports a superset (its own convenience additions) has
> silently forked the DSL: a manifest written against that superset works
> on one runtime and fails validation on every other.

So the instruction for this chapter is unusual for a tutorial: **do not
add anything.** No extra JSONPath operators because your language's
JSON library makes them free. No extra transforms because you already
have a slugify function lying around. No boolean composition in
conditions because it would obviously be more expressive. If a manifest
author needs something this language can't express, the answer is
[the native escape hatch](../../../spec/03-manifest-dsl.md#native-providers)
or a spec RFC, never a quiet extension in one runtime.

## Extraction: the JSONPath subset

Three forms, nothing else: `$.a.b` (field access), `$.a[0]` (index),
`$.a[*].b` (wildcard projection, exactly one field after the wildcard,
never chained). No recursive descent, no filter expressions, no
arithmetic. [`vectors/expressions/extraction.json`](../../../vectors/expressions/extraction.json)
is the full test:

```text
for each case in vectors/expressions/extraction.json.cases:
    result = extract(document, case.path)
    if case.missing:
        assert result is your runtime's explicit "did not resolve" value
        # never null coerced to a value, never an empty string, never a
        # thrown error here specifically — a step that *requires* the
        # value to resolve fails at the point it's used, not at extraction
    elif case.invalid:
        assert your validator rejects this path as malformed
        # e.g. a bare "$" with no .field/[index] segment at all: distinct
        # from "well-formed path, didn't resolve against this document"
    else:
        assert result == case.expected
```

Pay attention to the null-vs-missing-vs-invalid three-way split this
vector file draws out explicitly: a field present with value `null` is a
**resolved** result (`null` is the value), a field absent entirely (or a
path continuing through a `null` or non-existent intermediate) is
**missing**, and a syntactically malformed path (like a bare `$` with no
navigation segment) is **invalid** — a different failure mode a runtime
may reject earlier, e.g. at manifest-load time, rather than at
extraction time. Three states, not two; conflating any pair of them is
the single easiest way for two runtimes' extraction to disagree on a
edge case neither author thought was worth a manual test.

## Interpolation

`${namespace.path}` or `${transform(namespace.path)}`, in eight
namespaces:
`credentials`, `ctx`, `intent`, `state`, `extract`, `input`, `event`,
`idempotency_key` — plus `auth`, available only where `auth.shape` is
`oauth2_client_credentials` and only inside `auth.apply`/`auth.token`
themselves, never inside `flows`
([03-manifest-dsl.md#auth](../../../spec/03-manifest-dsl.md#auth)).
[`vectors/expressions/interpolation.json`](../../../vectors/expressions/interpolation.json)
covers all nine.

One easy-to-miss detail this vector file calls out under
`whole_value_type_preservation_cases`: a template that is *entirely* one
bare interpolation (`${intent.amount}`, nothing else in the string)
resolves to that value's **own JSON type** — an integer stays an
integer, a boolean stays a boolean — never a stringified form, *unless*
a transform is applied (every transform in the closed set always
produces a string) or the interpolation is embedded inside a larger
literal string (`"ref-${intent.amount}"`, which can only ever be a
string as a whole). This is what keeps
`NextAction.ShowTransferDetails.amount.minor_units`
([01-domain-model.md#nextaction-carries-its-own-payload](../../../spec/01-domain-model.md#nextaction-carries-its-own-payload))
a JSON integer rather than accidentally becoming `"10000"` — get this
wrong and your interpreter's golden output diverges from `expected/*.json`
on any manifest using a nested field like that one, in a way that's easy
to miss until chapter 8.

```text
for each case in vectors/expressions/interpolation.json.cases:
    assert interpolate(case.template, context) == case.expected

for each case in .../whole_value_type_preservation_cases.cases:
    result = interpolate(case.template, ...)
    assert result == case.expected
    assert json_type_of(result) == case.expected_json_type
```

Implement the closed transform set
([03-manifest-dsl.md#field-transforms](../../../spec/03-manifest-dsl.md#field-transforms)) —
`amount_major`, `msisdn_et`, `base64`, `hex`, `sha256_hex`, `upper`,
`lower`, `iso8601` — as eight small pure functions, each independently
testable against the same vector file, plus
[`vectors/money/conversions.json`](../../../vectors/money/conversions.json)
again for `amount_major` specifically (chapter 2). Transforms don't
compose (`${upper(base64(...))}` is not valid); don't build a pipeline
mechanism that would let them.

## Conditions

`==`, `!=`, `>`, `<`, `in` — one comparison, one path, one literal, no
boolean composition.
[`vectors/expressions/conditions.json`](../../../vectors/expressions/conditions.json)
is the test. The recurring theme worth internalizing from that file's
cases, not just passing them individually: **this language never
coerces types.** A numeric field is never `==` or `in` the same digits as
a string; a boolean field is never `==` the string `"true"`. If your
language's native comparison operators do coercion by default (several
dynamically-typed languages' `==` does), you need an explicit
type-and-value check here, not the language's own operator.

Today's manifest schema only ever constructs an equality condition
through `status_map`/`errors[].match.equals`
([03-manifest-dsl.md#errors](../../../spec/03-manifest-dsl.md#errors)) —
nothing in the current DSL surface actually reaches `!=`/`>`/`<`/`in`
through a real manifest field yet. Implement the full condition
evaluator as this chapter describes anyway: it's specified as part of
the expression language itself (forward-provisioned for a schema
extension that would need it), and the vector file tests it directly,
independent of whether today's schema happens to exercise every operator
through a manifest.

## What "done" looks like for this chapter

Three pure functions — `extract`, `interpolate`, `evaluate_condition` —
each passing its vector file completely, still with no manifest loader
and no interpreter. If you find yourself special-casing behavior for a
specific provider's field name here, stop: nothing in this chapter should
know any provider exists.
