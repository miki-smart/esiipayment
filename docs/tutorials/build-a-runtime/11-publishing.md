# 11. Publish honestly

Everything through chapter 10 is about correctness. This chapter is
about not overstating it once you publish.

## A compliance matrix, not a claim

Publish, prominently, exactly what your runtime has actually proven:
which providers you replay cleanly, which operations each one supports,
and each provider's own `verification.status`
(`provisional`/`verified` — [schema/metadata.v1.schema.json](../../../schema/metadata.v1.schema.json))
and tier (`community`/`certified`/`official` —
[GOVERNANCE.md](../../../GOVERNANCE.md#adapter-tiers)) as recorded in
*this* repository's `metadata.yaml`, not your own restatement of it that
can drift out of sync. A table your users can actually check against
`esiipayment-spec`'s own `providers/*/metadata.yaml` is worth far more
than a paragraph asserting "full conformance."

## `0.x` until a verified provider has been replayed

**Every real provider manifest in this repository — `chapa`, `arifpay`,
`santimpay` — is `verification.status: provisional` as of this writing.**
That's not a defect in those manifests; it's an honest statement that
their endpoint paths, field names, and webhook signature schemes were
written from general recollection of each provider's public API, not
from a current, licensed-to-read confirmation (see each provider's own
`metadata.yaml`, e.g.
[providers/chapa/metadata.yaml](../../../providers/chapa/metadata.yaml)).
Your runtime
replaying `chapa`'s cassettes byte-for-byte proves your **interpreter**
is correct against what that manifest *claims*; it proves nothing about
whether that manifest's claims match the real Chapa API today.

Don't let your own release versioning imply more than that. Stay at
`0.x` (or your ecosystem's equivalent pre-1.0 signal) until at least one
provider your runtime supports has actually moved to
`verification.status: verified` — meaning a maintainer has confirmed that
manifest against current, licensed provider documentation (see
[docs/tutorials/add-a-provider/11-verification-and-tiers.md](../add-a-provider/11-verification-and-tiers.md)) —
or, if you've verified a manifest's
accuracy yourself against a real sandbox account for your own runtime's
benefit, until you've contributed that verification back rather than
just keeping it as private knowledge (this repository's own
verification status is meant to be one shared fact every runtime and
every integrator can rely on, not something each SDK re-establishes for
itself and never reports upstream).

The failure mode this guards against isn't hypothetical: a `1.0.0` SDK
release reads as "production-ready" to anyone evaluating it, and nothing
about interpreter correctness — however real, however thoroughly tested
against `mock` and every vector in this repository — makes a
`provisional` manifest's claims about a real provider's actual API any
more true. Ship the correctness. Don't ship the implied claim that comes
free with a `1.0` version number until the thing that claim is actually
about (a verified provider) exists.

## What "done" looks like for this chapter, and this whole track

A published runtime with: a compliance matrix that names its actual,
checkable state; a pre-1.0 version number until a verified provider
backs it; a working `conformance-replay` entrypoint registered with this
repository's conformance matrix (chapter 10); and, per this track's
[introduction](index.md#one-thing-every-runtime-eventually-does-writes-a-vector),
an open channel back to `esiipayment-spec` for the vector gaps you will
find. Then: [the conformance checklist](checklist.md).
