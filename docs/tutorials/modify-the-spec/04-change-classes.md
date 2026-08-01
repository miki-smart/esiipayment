# 4. Change classes

Not every change to this repository is the same weight. Knowing which
class a change belongs to tells you which review path it needs — and
misclassifying a breaking change as additive is the single most common
way a "small" PR turns into an incident three other repositories
discover independently.

## Additive: doesn't require an RFC

- A new vector file, or new cases added to an existing one, that adds
  coverage **without changing the behaviour any existing vector
  documents**. This is explicitly called out as ordinary in
  [GOVERNANCE.md](../../../GOVERNANCE.md#ordinary-changes) — the same
  class of change as a documentation clarification.
- A new provider manifest, or a fix to an existing one, that doesn't
  invent a new field, transform, or match kind.
- A new **optional** schema field that every previously-valid manifest,
  unmodified, still satisfies (it's optional; omitting it changes
  nothing for a manifest that doesn't set it).
- A validator bugfix that doesn't change what a *correct* manifest is
  required to look like — fixing a false negative (the validator wrongly
  rejects something it should accept) or a false positive in a
  diagnostic message, not loosening or tightening what's actually valid.

These get ordinary PR review: one approval from the relevant CODEOWNERS
team, passing CI.

## Breaking: requires an RFC first

- Adding, renaming, or removing a member of any closed enum — even one
  member, even one that "obviously" should have been there from the
  start.
- A change to the expression language's grammar, namespaces, operators,
  or transform set.
- A schema change that invalidates a manifest that previously validated,
  or changes what `esiipayment validate` accepts for an existing,
  unmodified field.
- A change to an invariant's actual required behaviour (not its wording).

These need [the RFC process](03-the-rfc-process.md) — no exceptions for
"it's a small addition," since (as
[chapter 5](05-the-obligation-you-create.md) covers) the cost isn't
about the size of the diff in this repository.

## The judgment call in between

Some changes don't fit either bucket cleanly on first look — a new
*required* field on an existing section, for instance, is structurally
"just one field" but breaking in exactly the sense that matters (every
manifest that validated yesterday now needs updating). When in doubt,
treat it as breaking and open an RFC: the cost of an unnecessary RFC
discussion is a few days; the cost of a breaking change that shipped as
an ordinary PR is every downstream runtime and manifest discovering it
broke something, independently, after the fact.
