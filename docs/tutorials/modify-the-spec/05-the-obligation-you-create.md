# 5. The obligation you create

A one-member addition to `FailureCode` looks tiny from inside a single
PR against this repository. From outside, it's every
`esiipayment-<language>` runtime needing to add a case to the same
`switch`/`match` statement before the addition is actually usable
anywhere, and every existing provider manifest's `errors` list becoming
a candidate for revisiting against the new option. That's the real
weight [GOVERNANCE.md](../../../GOVERNANCE.md#rfc-process) is naming when
it calls this repository "a coordination point": the RFC process is the
forcing function that surfaces that N-runtime cost *before* merge, not
after five runtimes independently discover it.

## Add the vector before the schema change

When your accepted RFC touches behaviour a vector file could describe,
write that vector **first**, against the *new* intended behaviour,
before you change `schema/*.schema.json` or any prose. This orders the
work so the executable definition of "correct" exists before anything
that has to conform to it does, and it gives every runtime implementer
something concrete to build against the moment the schema change lands
— not a description they have to translate into test cases themselves,
each potentially slightly differently.

## The cross-runtime matrix has to stay green

[`conformance-matrix.yml`](../../../.github/workflows/conformance-matrix.yml)
fans out to every registered SDK repository on every manifest/schema/spec
change, specifically so a change that breaks a runtime fails on *this*
repository's PR, not later, silently, in that runtime's own repository.
A breaking change is not done merging when this repository's own CI is
green — it's done when every runtime that was passing before your change
either still passes, or has a clear, tracked path to updating (the
"existing runtimes are not required to implement a new closed-enum
member immediately, but must not silently ignore or mis-map it" rule
from [chapter 3](03-the-rfc-process.md) is exactly the boundary here: a
runtime doesn't have to catch up instantly, but it can't pretend the
change didn't happen either).

If you're proposing a breaking change and can influence more than one
runtime (your own, plus others you have some relationship with), it's
worth coordinating the update timing before merging here — not required
by process, but the difference between "the matrix goes red for a day
while everyone updates in a coordinated way" and "the matrix goes red
indefinitely because nobody knew to look."

## What this chapter is not saying

This isn't an argument against ever making breaking changes — this
repository has made several (see
[08-versioning.md](../../../spec/08-versioning.md)'s own account of the
`1.0` → `2.0` manifest DSL migration). It's an argument for making the
real cost visible to yourself, deliberately, before you propose one —
which is exactly what the RFC template's "impact on existing runtimes
and manifests" question ([chapter 3](03-the-rfc-process.md)) is there to
force you to write down, not skip past.
