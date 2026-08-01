# 6. Versioning and deprecation

[08-versioning.md](../../../spec/08-versioning.md) is the full reference;
this is what it means for the change you're actually making.

## Two independent version numbers

Don't conflate these — it's the most common way a spec project confuses
its own contributors:

- **`VERSION`** (repository root, a full SemVer string): the spec's own
  version — `schema/`, `spec/`, the expression language, the invariants,
  as a whole. Major for a closed-enum/expression-language/breaking-schema
  change (always via RFC); minor for a backward-compatible addition;
  patch for a clarification or a validator bugfix that doesn't change
  what a correct manifest looks like.
- **`spec_version`** (declared by every individual manifest, e.g.
  `"2.0"`): the version of the *manifest DSL itself* a given manifest is
  written against — independent of the repository's overall `VERSION`,
  because the DSL can in principle reach a new major version while
  manifests written against the prior one keep validating and running,
  under a runtime that supports both concurrently during a migration
  window.

Bump `VERSION` when your change is the kind
[chapter 4](04-change-classes.md) calls breaking or a notable addition.
Bump `spec_version` specifically when the *DSL shape itself* changes in
a way manifests need to declare which version they target — and prefer
the gradual, dual-version migration path this indirection is designed
for (old-shape manifests keep validating under the version they declare,
a runtime supports both concurrently) over changing everything in one
step. This repository's own `1.0` → `2.0` migration
([08-versioning.md](../../../spec/08-versioning.md#10-20-why-this-migrated-in-one-step-not-gradually))
did the latter — migrating every manifest in one PR rather than
supporting both shapes at once — and says plainly why that was a
deliberate, narrower-than-ideal exception (a small, single-repository
provider set, not a signal that skipping the migration window is the
normal path): don't take it as precedent without the same justification
applying to your change.

## Deprecation

Nothing in this repository currently has a formal deprecation mechanism
beyond the version-bump rules above — there's no deprecated-but-still-valid
enum member convention yet, for instance. If your change needs one (you're
proposing to phase out a closed-enum member or a DSL shape, not just add
a new one), that gap is itself worth raising as part of your RFC
([chapter 3](03-the-rfc-process.md)): propose the deprecation mechanism
as part of the same discussion, rather than assuming one already exists
that your change can slot into.

## Where your PR actually lands

By the time your change merges, it should have touched, together: the
relevant `spec/*.md` prose, `schema/*.schema.json` if validation
behaviour changed, `vectors/*` for anything the change affects
(written **before** the schema change — [chapter 5](05-the-obligation-you-create.md)),
and the `VERSION` (and, if applicable, `spec_version`) bump this chapter
describes. A PR that changes behaviour without updating the version
number it belongs under is missing exactly the signal every downstream
consumer relies on to know whether pulling in this change is safe.
