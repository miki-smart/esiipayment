# 1. Setup

## Pin the spec at a SHA, not a branch

This repository (`esiipayment-spec`) is a git submodule of your SDK
repository. Pin it at a **commit SHA**, not `main` or any other branch.

This matters more than it looks like it should. `main` moving under you
mid-development means a manifest you validated yesterday can silently
stop validating today, for a reason that has nothing to do with any
change you made — a new required field, a stricter check, an added
vector your interpreter now fails. None of that is wrong of the spec
repo to do (see [08-versioning.md](../../../spec/08-versioning.md)); it's
wrong for *your* build to be exposed to it without you choosing when.
Bump the pinned SHA deliberately, read what changed
(`spec/08-versioning.md`'s versioning rules tell you whether it's a
patch/minor/major change and what that implies), and treat the bump
itself as a normal, reviewable commit in your own repository — not
something that happens to you as a side effect of `git submodule update
--remote` on a schedule.

## Repository layout, from your side

You'll read from four directories in this repository, roughly in this
order across the rest of this track:

```text
spec/       Prose. Read 00-overview.md and 01-domain-model.md first;
            everything else you'll come back to per-chapter.
schema/     JSON Schemas for manifest.yaml, metadata.yaml, cassette.yaml,
            capabilities.yaml (native providers), and messages/*.json.
            Useful for editor tooling and as a cross-check, but your
            interpreter's actual validation logic is closed-vocabulary
            checks against spec/01-domain-model.md's enums (see chapter 6),
            not a generic JSON Schema library — see that chapter for why.
vectors/    Data your interpreter's primitives (chapters 2-5) are tested
            against directly, before any manifest exists.
providers/  mock/ is what you replay in chapter 8; chapa/, arifpay/, and
            santimpay/ are real (provisional) manifests you don't touch
            until mock passes completely.
```

You will never write to any of these directories from your SDK
repository. If something here needs to change (a vector gap, an
ambiguity, a real bug), that's a PR against `esiipayment-spec` itself —
see this track's [introduction](index.md#one-thing-every-runtime-eventually-does-writes-a-vector).

## What's conformance-tested, and what isn't

Split your runtime into two halves early, because
[06-conformance.md](../../../spec/06-conformance.md) only mechanically
checks one of them:

- **The interpreter**: everything a manifest and a cassette fully
  determine — expression evaluation, canonical JSON, the step executor,
  error classification, webhook signature verification. Given the same
  manifest, the same cassette, and the same injected clock/UUID seed, the
  interpreter's output is a pure function of its inputs
  ([04-expression-language.md#determinism](../../../spec/04-expression-language.md#determinism)).
  This half is what `esiipayment replay --assert-golden` checks
  byte-for-byte against `expected/*.json`, and what
  `conformance-matrix.yml` in the spec repo runs across every
  `{runtime} × {provider}` pair. If your interpreter is correct, this is
  mechanically provable, not a matter of opinion.
- **The engine**: everything that isn't determined by one manifest run
  in isolation — idempotency-key storage, the persisted payment record,
  concurrent-request handling, real network transport, retry policy,
  credential storage, single-flight token refresh
  ([07-runtime-requirements.md#credential-handling](../../../spec/07-runtime-requirements.md#credential-handling)).
  Nothing in this repository's cassettes exercises concurrency or real
  I/O, so nothing here can be golden-output-compared. Chapter 9 covers
  how to test this half anyway: not byte comparison, but invariant tests
  that assert a specific property (Invariant I4, I5, I6, I8) holds
  under an adversarial fake.

Keep this split real in your code, not just conceptual: your interpreter
should be constructible and testable with no engine present at all (no
database, no HTTP client, no clock but the one you inject). If you find
yourself needing a database connection to unit-test expression
evaluation, the two halves have already blurred together, and chapters 2
through 8 of this track will be harder to follow than they need to be.
