# 3. The RFC process

[GOVERNANCE.md#rfc-process](../../../GOVERNANCE.md#rfc-process) is the
full normative process; this chapter is how to actually use it.

## When you need one

Before merging any change touching: a **closed enum**
([01-domain-model.md](../../../spec/01-domain-model.md)), the
**expression language** (extraction grammar, interpolation namespaces,
condition operators, the closed transform set), or the shape of
`schema/manifest.v1.schema.json` (or the metadata/cassette/capabilities
schemas) **in a way that invalidates previously-valid manifests, or
changes what `esiipayment validate` accepts**. If your change doesn't
touch any of those — a new vector, a documentation clarification, a new
optional field nothing existing must set — it's an ordinary PR instead;
see [chapter 4](04-change-classes.md) for exactly where that line falls.

## Open the issue

Use the **"Spec RFC"** issue template
(`.github/ISSUE_TEMPLATE/spec-rfc.yml`). It asks for four things,
deliberately in this order:

1. **What area this touches** — a closed enum, the expression language,
   the manifest schema's validation behaviour, or something else you
   think is RFC-worthy anyway.
2. **What can't the current spec express** — the concrete problem,
   *not yet the solution*. Naming the actual gap before proposing a fix
   is what lets a reviewer sanity-check the problem independently of
   whatever solution you have in mind, rather than only ever evaluating
   your proposed answer.
3. **A concrete motivating example** — a real (or realistic) provider
   manifest or scenario that actually needs this. An RFC argued in the
   abstract is much harder to evaluate than one anchored to "here's the
   specific manifest section that can't be written today."
4. **Impact on existing runtimes and manifests** — additive (existing
   manifests keep validating, existing runtimes keep working unmodified)
   or breaking, and specifically what every `esiipayment-<language>`
   runtime would need to change. This is the question that surfaces the
   real cost — see [chapter 5](05-the-obligation-you-create.md).

## What happens next

Discussion has no fixed timer — a small, uncontested RFC moves fast; one
touching a closed enum every provider manifest could reference gets
scrutiny proportional to its blast radius, especially from runtime
implementers, since they bear the implementation cost. A maintainer
records the decision (accepted, rejected, or accepted with changes)
**directly in the issue, with rationale** — a rejected RFC should tell
you why, in the same style [01-domain-model.md](../../../spec/01-domain-model.md)
already gives every rule it states ("Why there is no `Indeterminate`" is
exactly this kind of rationale, written down once so it doesn't need
re-litigating every time someone proposes reintroducing it).

## If it's accepted

The implementing PR updates, together, in one change: the relevant
`spec/*.md` file(s), `schema/*.schema.json` if validation behaviour
changes, `vectors/*` for any new fixed table or vector set the change
introduces or affects, and the version bump per
[chapter 6](06-versioning-and-deprecation.md). Existing runtimes aren't
required to implement a new closed-enum member or transform
immediately, but must not silently ignore or mis-map it — see
[01-domain-model.md](../../../spec/01-domain-model.md)'s note on
treating an unrecognized value as a mapping bug, never license to invent
a new one locally.
