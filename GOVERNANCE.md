# Governance

This document describes how decisions get made in ESIIPayment: who can
merge what, when a change needs wider review than one maintainer's
approval, and how a provider earns more trust over time. See
[CODEOWNERS](.github/CODEOWNERS) for the mechanical enforcement of the
review split described here.

## Roles

- **Adapter reviewers** (`@esiipayment/adapter-reviewers`): review and
  merge changes under `providers/<name>/` for providers they're
  knowledgeable about. Adapter review does not require programming
  language expertise; it requires knowing (or being willing to check)
  whether a manifest's claims about a specific provider's API are
  plausible, and confirming `esiipayment validate`/`esiipayment replay
  --assert-golden` pass.
- **Maintainers** (`@esiipayment/maintainers`): the core team. Own
  `spec/`, `schema/`, `vectors/`, `tools/`, and governance/process files.
  Maintainers are also the fallback reviewers for `providers/` changes
  when no adapter reviewer is available or when a change is contested.
- **RFC participants**: anyone. The RFC process below is open to any
  contributor, not gated to maintainers; maintainers are responsible for
  deciding RFCs, not for being the only people allowed to propose them.

Team membership is granted by existing maintainers based on sustained,
good-faith contribution; there is no fixed quota or election process at
this project's current size. As the project grows, this document is
itself subject to the RFC process below if that stops being adequate.

## Ordinary changes

Most contributions do not need an RFC:

- A new provider manifest, or a fix to an existing one, that doesn't
  invent a new field, transform, or match kind: reviewed and merged by
  an adapter reviewer per [docs/add-a-provider.md](docs/add-a-provider.md).
- A documentation clarification, a corrected vector that was itself
  wrong, a validator bugfix that doesn't change what a *correct* manifest
  is required to look like: reviewed and merged by a maintainer.
- A new vector file that adds coverage without changing the behaviour any
  existing vector documents.

These follow ordinary PR review: at least one approval from the relevant
CODEOWNERS team, passing CI (`validate.yml`, `no-code.yml`, and for
provider changes, the runtime conformance dispatch in
`conformance-matrix.yml`).

## RFC process

An RFC is required before merging any change that touches:

- A **closed enum** in [spec/01-domain-model.md](spec/01-domain-model.md)
  (`PaymentStatus`, `NextAction`, `FailureCode`, `RetryClass`,
  `CredentialShape`, `Operation`): adding, renaming, or removing a
  member.
- The **expression language**
  ([spec/04-expression-language.md](spec/04-expression-language.md)):
  extraction grammar, interpolation namespaces, condition operators, or
  the closed transform set.
- The shape of **`schema/manifest.v1.schema.json`** (or the metadata/
  cassette schemas) in a way that invalidates previously-valid manifests,
  or changes what `esiipayment validate` accepts.

Why these three and not, say, adding a new provider: each of these is a
change every language runtime must implement identically before it's
usable anywhere (spec/06-conformance.md's whole premise), and a change
that looks small from inside one PR (one new `FailureCode` member, one
new transform) is actually a coordination problem across every SDK repo.
The RFC process is the forcing function that surfaces that coordination
cost before merge, not after five runtimes independently discover it.

### Steps

1. **Open an issue** using the "Spec RFC" issue template, describing the
   problem the current spec can't express, at least one concrete example
   manifest/scenario that motivates it, and a proposed change.
2. **Discussion period.** Maintainers and any interested contributors
   (particularly runtime implementers, since they bear the implementation
   cost) discuss the proposal in the issue. There is no fixed timer:
   small, uncontested RFCs move fast; anything touching a closed enum
   used across every provider gets more scrutiny proportional to its
   blast radius.
3. **Decision.** A maintainer records the decision (accepted, rejected,
   or accepted with changes) directly in the issue, with rationale.
   Rationale is not optional: a rejected RFC should tell the proposer
   *why*, and an accepted one should give future readers of
   `spec/01-domain-model.md` (or wherever the change lands) the same
   "why," matching the style already used throughout this spec.
4. **Implementation.** The PR implementing an accepted RFC updates:
   - the relevant `spec/*.md` file(s),
   - `schema/*.schema.json` if the change affects validation,
   - `vectors/*` if the change affects a fixed table or vector set
     (e.g. a new `FailureCode` needs a `vectors/errors/failure-retry-map.json`
     entry),
   - the version bump per [spec/08-versioning.md](spec/08-versioning.md).
5. **Runtime follow-up.** Once merged, the change is a breaking or
   additive release per spec/08-versioning.md; existing runtimes are not
   required to implement a new closed-enum member or transform
   immediately, but must not silently ignore or mis-map it: see
   [spec/01-domain-model.md](spec/01-domain-model.md)'s note on treating
   an unrecognized value as an adapter mapping bug, not license to
   invent a new one.

## Adapter tiers

A provider's `metadata.yaml` declares a `tier`
(see [schema/metadata.v1.schema.json](schema/metadata.v1.schema.json)),
with automatable criteria so tier isn't a subjective label:

| Tier | Criteria |
|---|---|
| **community** | `esiipayment validate` passes; cassettes exist for at least success, a provider-reported failure, an auth failure, and a transport timeout; `esiipayment replay --assert-golden` passes against them. This is the bar every first provider PR clears; nothing about "community" implies lower quality, only that it hasn't yet had the additional steps below. |
| **certified** | Everything in community, plus: a live run against the provider's real sandbox (not just recorded cassettes) verified by a maintainer or a named adapter reviewer, and at least one named maintainer in `metadata.yaml`'s `maintainers` list who has committed to keeping it current. |
| **official** | Core-team maintained, and the provider itself has acknowledged or engaged with the integration (this is the only tier this repo cannot self-certify; it depends on an external relationship, not just engineering criteria). |

Tier is independent of `verification.status` (`provisional`/`verified`):
tier is about maintenance commitment and depth of testing; verification
status is about whether the manifest's claims about the provider's API
have been checked against current documentation. A manifest can in
principle be `community` tier and `verified`, or (as every real provider
in this repository is today) `community` tier and `provisional`. See
[docs/add-a-provider.md](docs/add-a-provider.md).

## Changes to this document

GOVERNANCE.md itself follows the RFC process above when the change is
structural (adding/removing a tier, changing the closed-enum RFC
trigger list); wording clarifications that don't change who can do what
are an ordinary maintainer-reviewed change.
