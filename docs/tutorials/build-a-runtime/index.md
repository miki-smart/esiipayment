# Build a runtime

This track is for a developer porting ESIIPayment to a new language:
you know your language; you don't need to know anything about Chapa,
ArifPay, SantimPay, or Ethiopian payment providers generally.
[`providers/mock/`](../../../providers/mock/) is deliberately sufficient
to build and prove an entire runtime before your interpreter ever reads
a real provider's manifest — see
[07-runtime-requirements.md](../../../spec/07-runtime-requirements.md).

Everything here is pseudocode, deliberately language-neutral. Nothing in
this track tells you what class to name something or which HTTP client
library to use; it tells you what has to be true when you're done, and
in what order to build it so each piece is checkable on its own before
the next depends on it.

## Why this order, specifically

The eleven chapters are ordered by **dependency, not by how much progress
feels visible**. Chapter 2 (primitives) produces nothing that looks like
a payment runtime; chapter 8 (replay against `mock`) is the first point
you can show anyone a `PaymentResult`. That's deliberate. Each chapter is:

1. A hard prerequisite for the chapter after it (the expression
   evaluator can't be tested without `Money` and the closed enums it
   returns; the interpreter can't run without the expression evaluator
   and the manifest loader both working; replay can't be asserted without
   canonical JSON being correct first), and
2. **Independently testable with zero infrastructure** — no manifest, no
   HTTP client, no database, often not even a full parser. Chapters 2
   through 6 are pure functions tested against a static vector file you
   already have, sitting in this repository, before you write a single
   line of anything that looks like "the runtime."

Building in visible-progress order instead (say, the interpreter first,
stubbing out expression evaluation as you go) means your first correct
end-to-end run is also the first time any of expression evaluation,
canonical JSON, and error mapping have been tested at all — three
untested subsystems compounding into one debugging session with no way
to isolate which one is wrong. Building in dependency order means that
by the time you write the interpreter (chapter 7), everything it calls
has already been proven correct against this repository's own vectors,
and a bug at that point is almost always in the interpreter's own
control flow, not in a primitive underneath it.

## The chapters

1. [Setup](01-setup.md) — pin the spec, repo layout, what's
   conformance-tested versus what isn't.
2. [Primitives](02-primitives.md) — `Money`, the closed enums, status
   transitions.
3. [The expression evaluator](03-expression-evaluator.md) — extraction,
   interpolation, conditions.
4. [Canonical JSON](04-canonical-json.md) — the byte-exact serialization
   golden output depends on.
5. [Error mapping and webhook verification](05-errors-and-webhooks.md).
6. [The manifest loader](06-manifest-loader.md).
7. [The interpreter](07-interpreter.md) — the step executor.
8. [Replay against `mock` first](08-replay-against-mock.md) — the
   highest-leverage chapter in this track.
9. [The engine](09-the-engine.md) — idempotency, persistence, the
   transport guard.
10. [`conformance-replay` and the conformance matrix](10-conformance-replay.md).
11. [Publishing honestly](11-publishing.md).

Then: [the conformance checklist](checklist.md).

## One thing every runtime eventually does: writes a vector

`vectors/` is the executable definition of the expression language,
canonical JSON, money handling, error classification, webhook
verification, and status transitions — but it was written by people, for
a project with (at the time you read this) a small number of reference
implementations. **You will find a case it doesn't cover.** When you do,
that's not a bug in your runtime to work around quietly; it's a gap in
`vectors/` to fix, in the open, in
[`esiipayment-spec`](https://github.com/miki-smart/esiipayment)
(this repository), not a corner case you handle silently in your own SDK.

A new vector that adds coverage without changing the behaviour any
existing vector documents is an **ordinary change** under
[GOVERNANCE.md](../../../GOVERNANCE.md) — reviewed and merged by a
maintainer, no RFC required, the same class of change as a documentation
clarification. If what you found isn't just missing coverage but an
actual ambiguity in the prose (two readings of
[04-expression-language.md](../../../spec/04-expression-language.md) both
seem defensible, and the vectors don't settle it either), that's worth
raising as an issue before you guess: the next runtime after yours
shouldn't have to rediscover the same ambiguity independently.
