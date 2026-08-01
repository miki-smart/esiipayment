# 1. What's frozen, and why

Three things in this repository are frozen in a stronger sense than
"hard to change" — they're designed to be **closed**, and a proposal
that opens them needs to justify why the closure itself is wrong, not
just why one more member/operator/capability would be convenient.

## The closed enums

`PaymentStatus` (6), `NextAction` (9, each with its own closed field
set), `FailureCode` (14), `RetryClass` (3), `CredentialShape` (7),
`Operation` (6) —
[01-domain-model.md](../../../spec/01-domain-model.md). Every one of
these is closed specifically so every language runtime can implement an
exhaustive `switch`/`match` over it and never need a `default` branch
that silently swallows a value it doesn't recognize. That exhaustiveness
is the mechanism behind [Invariant I12](../../../spec/02-invariants.md#i12)
(integrator code never branches on provider) and
[Invariant I3](../../../spec/02-invariants.md#i3) (`NextAction` describes
user obligations, not provider API details): a checkout UI with nine
`NextAction` branches never needs a tenth when a new provider is added,
*because* the tenth member can't silently appear — adding one is
mechanically forced to go through this repository, an RFC, and every
runtime's own release.

## The twelve invariants

[02-invariants.md](../../../spec/02-invariants.md)'s I1 through I12 are
not implementation advice — each one is enforced by at least one vector
or validator check, traceable back to the rule it protects. They exist
because getting any one of them wrong reintroduces a specific,
previously-identified failure mode (I4's transport-timeout rule exists
specifically to prevent double-charging; I9 exists specifically so a
manifest reviewed in the open never needs to expose a real credential).
Changing the *behaviour* an invariant describes — not clarifying its
wording, but changing what's actually required — is exactly the kind of
change this tutorial's remaining chapters are about.

## The expression language's deliberate weakness

[04-expression-language.md](../../../spec/04-expression-language.md) is
short specifically so "a runtime implementer can finish it in an
afternoon," and it says outright that a runtime implementing a superset
has "silently forked the DSL." No recursive descent in extraction paths.
No boolean composition in conditions. No user-defined transforms. This
isn't an oversight waiting to be completed — it's the load-bearing
design decision that keeps every manifest interpretable identically by
every language runtime, without any of them embedding a general-purpose
expression evaluator. [Chapter 2](02-rule-of-least-power.md) is entirely
about why "the DSL can't express X" is usually not, by itself, a reason
to widen it.

## What this means for you, proposing a change

None of this means these things never change — `NextAction` gaining its
typed payload and `auth` gaining `apply`/`token` were both real changes
to previously-closed surface, in this repository's own history. It means
the bar for changing them is [the RFC process](03-the-rfc-process.md),
not an ordinary PR, and the burden is on the proposal to show the
closure itself, not just the specific member count, is what needs to
move.
