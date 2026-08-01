# 10. Expose `conformance-replay`, and register with the conformance matrix

By now your own runtime can do everything `esiipayment validate` and
`esiipayment replay --assert-golden` do (chapters 6-8), in your own
language. This chapter is about making that runnable *by other people's
CI*, not just your own.

## Expose it

Give your SDK a command (a CLI entrypoint, an exported function, whatever
is idiomatic — this repository's own reference validator calls its
equivalent `esiipayment replay <provider-dir> --assert-golden`; yours
doesn't need the same name) that:

1. Takes a provider directory (or provider name, resolved against a
   pinned `esiipayment-spec` submodule checkout).
2. Validates that provider's manifest.
3. Replays every cassette and asserts byte-identical canonical JSON
   against `expected/`.
4. Exits non-zero, with a clear diff, on the first mismatch.

This is exactly chapter 8's harness, exposed as a stable entrypoint
something outside your own test suite can invoke — specifically, the
receiver workflow below.

## Register with `esiipayment-spec`'s conformance matrix

[`.github/workflows/conformance-matrix.yml`](../../../.github/workflows/conformance-matrix.yml),
in this repository, fans out to every language SDK on a
`{sdk} × {provider}` matrix whenever `providers/`, `schema/`, or `spec/`
changes on `main` — so a manifest change that breaks *your* runtime fails
loudly on the manifest PR itself, not silently, later, in your own
repository. This repo only controls the **dispatch** side; your SDK
repository has to implement the **receiver**. The contract, verbatim
from that workflow's own header comment:

> Each entry in `sdks` must have a workflow listening for a
> `repository_dispatch` event of type `esiipayment-conformance-check`,
> whose payload is `{"provider": "<name>", "sha": "<this repo's commit>"}`.
> That workflow is expected to:
>
> 1. update its `esiipayment-spec` submodule to the given sha,
> 2. run its own runtime's conformance suite against
>    `providers/<provider>` (validate the manifest, replay every
>    cassette, assert golden output),
> 3. report a commit status (or check run) back on this repo's sha,
>    named e.g. `"conformance/<sdk>/<provider>"`, so branch protection on
>    this repo can require it.

Building the receiver workflow is: listen for that dispatch event,
`git submodule update` to the given SHA, invoke the `conformance-replay`
entrypoint you exposed above against the named provider, and post a
commit status back to *this* repository's SHA (not your own) using
whatever your CI platform's cross-repository status API is (a GitHub
Actions workflow in your SDK repo posting to `esiipayment/esiipayment-spec`
needs a token with permission to do that — see
`peter-evans/repository-dispatch`'s own receiving-side documentation for
the common pattern, or your CI platform's equivalent).

Then open a PR against `esiipayment-spec` adding your SDK's repository
name to `conformance-matrix.yml`'s `sdks` list. As of this writing, none
of the five placeholder entries there
(`esiipayment-dotnet`/`-python`/`-node`/`-go`/`-php`) have a real
repository with the receiver implemented yet — dispatches to them
currently fail with "repository not found," and that's called out
explicitly in the workflow file as expected, not something blocking
merges on this repo. If you're building one of those five, this is where
your work turns that placeholder into a real, checked entry; if you're
building a runtime in a language not on that list at all, this is where
you add it.

## What "done" looks like for this chapter

A `conformance-replay` entrypoint your SDK exposes, a receiver workflow
in your own repository that responds to `esiipayment-conformance-check`
dispatches correctly, and (once you're ready to have this repo's own CI
depend on you) a merged PR adding your SDK to `conformance-matrix.yml`.
