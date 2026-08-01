# Build a runtime

This guide has moved to
[docs/tutorials/build-a-runtime/](tutorials/build-a-runtime/index.md), an
eleven-chapter tutorial ordered by dependency (each chapter is a
prerequisite for, and independently testable before, the next) rather
than by visible progress, ending in a
[conformance checklist](tutorials/build-a-runtime/checklist.md) covering
everything this page used to describe in one pass and considerably more:
`vectors/expressions/`'s and `vectors/canonical-json/`'s edge cases,
the engine-level invariant tests (a chaos transport, a counting
transport, a store spy, a serialize-and-rehydrate test), and the exact
`repository_dispatch` contract `conformance-matrix.yml` expects from your
runtime's receiver workflow.

If you followed a link here from somewhere else, please update it to
point at [docs/tutorials/build-a-runtime/](tutorials/build-a-runtime/index.md)
directly.
