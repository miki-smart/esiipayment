# Modify the spec

This tutorial is for someone changing `spec/`, `schema/`, the expression
language, or the manifest DSL itself — not for adding or fixing a
provider manifest (that's [add-a-provider](../add-a-provider/index.md))
and not for building a language runtime (that's
[build-a-runtime](../build-a-runtime/index.md)). It assumes you're
already familiar with the existing `spec/` documents; if you're not,
start with [00-overview.md](../../../spec/00-overview.md).

The theme running through all six sections: **this repository is a
coordination point for every language SDK that implements it.** A change
that looks small from inside one PR — one new enum member, one new
transform, one relaxed validation rule — is, from outside this
repository, a change every SDK must implement identically before it's
usable anywhere. This tutorial is mostly about making that cost visible
to yourself before you propose a change, not about discouraging you from
proposing one.

1. [What's frozen, and why](01-whats-frozen.md).
2. [The rule of least power](02-rule-of-least-power.md) — why the DSL
   deliberately can't do more, and why "it can't express my provider" is
   usually not a reason to add a DSL feature.
3. [The RFC process](03-the-rfc-process.md), using the `spec-rfc.yml`
   issue template.
4. [Change classes](04-change-classes.md) — additive versus breaking,
   and what each actually requires.
5. [The obligation you create](05-the-obligation-you-create.md) — every
   change ripples to every runtime.
6. [Versioning and deprecation](06-versioning-and-deprecation.md).
