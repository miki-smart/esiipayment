# 08: Versioning

Three independent things carry version numbers in this project, and
conflating them is the most common way a spec project confuses its own
contributors: the **spec** itself (this repository as a whole), the
**manifest DSL** a given manifest is written against, and an individual
**provider manifest's** own revision. This document defines how each moves.

## The spec (this repository)

The repository's own version lives in [`VERSION`](../VERSION) at the root
(currently `1`) and follows SemVer for the repository as a whole
(`schema/`, `spec/`, the expression language, the invariants). A runtime or
tooling that pins `esiipayment-spec` as a submodule pins this version.

- **Major**: a breaking change to a closed enum
  ([01-domain-model.md](01-domain-model.md)), the manifest schema in a way
  that invalidates previously-valid manifests, or the expression language.
  Always goes through the RFC process in
  [GOVERNANCE.md](../GOVERNANCE.md); see below.
- **Minor**: a backward-compatible addition: a new optional schema field,
  a new provider, a new vector file, a new transform (itself an RFC, since
  the transform set is closed, but additive and non-breaking to existing
  manifests once accepted).
- **Patch**: documentation clarification, a corrected vector that was
  itself wrong, a validator bugfix that doesn't change what a *correct*
  manifest is required to look like.

## `spec_version` (the manifest DSL)

Every manifest declares `spec_version: "1.0"`
([03-manifest-dsl.md](03-manifest-dsl.md)): the version of the *manifest
DSL* (schema shape, expression language, flow semantics) it's written
against, independent of the repository's own overall `VERSION`. This
indirection exists because the DSL can, in principle, reach a new major
version while providers written against the prior version continue to
validate and run unmodified; a runtime is expected to support more than
one `spec_version` concurrently during a migration window rather than
forcing every provider manifest to migrate in lockstep with every spec
release.

A runtime must refuse to execute a manifest whose `spec_version` it does
not implement, with a clear error, rather than attempting a best-effort
interpretation: silently guessing at semantics for a DSL version a
runtime wasn't built against is exactly the kind of cross-runtime
divergence [06-conformance.md](06-conformance.md) exists to prevent.

## A provider manifest's own revision

An individual provider's `manifest.yaml` and `metadata.yaml` change over
time as the real provider's API evolves or an adapter author fixes a
mapping. These changes are tracked by this repository's ordinary git
history and the manifest's own `metadata.yaml` fields (see
[docs/add-a-provider.md](../docs/add-a-provider.md)): there is no separate
per-provider semantic version number. A change to a provider manifest that
alters its cassette-derived golden output is, by definition, required to
ship an updated cassette and `expected/` fixture in the same PR
([06-conformance.md](06-conformance.md)), which is what keeps a provider
manifest's history auditable without needing its own version scheme
layered on top of git's.

## Breaking changes require an RFC

Anything touching a closed enum
([01-domain-model.md](01-domain-model.md)), the expression language
([04-expression-language.md](04-expression-language.md)), or the shape of
`schema/manifest.v1.schema.json` in a way that invalidates existing
manifests goes through the RFC process defined in
[GOVERNANCE.md](../GOVERNANCE.md), regardless of how small it looks. A
one-member addition to `FailureCode` looks tiny from inside a single PR;
from outside, it's five runtimes that all need to add a case to the same
switch statement before the addition is actually usable, and every
existing manifest's `errors` list is a candidate for revisiting against
the new option. The RFC process is the forcing function that makes that
coordination happen before the change merges, not after five runtimes
independently discover it broke something.

## Compatibility guarantee

Within a major `VERSION`, this repository guarantees:

- Every manifest that validated against `schema/manifest.v1.schema.json`
  continues to validate.
- Every conformant runtime continues to produce the same golden output for
  every existing cassette.
- No closed enum member is renamed or removed.

This is the guarantee that lets a language SDK pin a `esiipayment-spec`
submodule at a given major version and trust that pulling in new patch and
minor releases (new providers, new vectors, clarified docs) never breaks
its existing conformance suite.
