# 8. Generating golden files

Every cassette needs a matching `expected/<name>.json` — the exact
canonical JSON ([06-conformance.md](../../../spec/06-conformance.md))
your manifest's flow should produce for it: sorted object keys at every
level, no insignificant whitespace, integers never as floats, fixed
second-precision UTC timestamps. This is what a runtime's replay tool
byte-compares against.

## How to actually produce one

Don't hand-compute this. Run:

```text
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator replay providers/<your-provider>
```

(without `--assert-golden`) and it prints, for each cassette, the exact
canonical JSON your manifest currently produces. **Read it carefully**,
confirm it's actually correct — the right `status`, the right
`next_action` with the right fields, `state` free of anything the
integrator shouldn't see — and only then save that exact output as the
cassette's `expected/<name>.json`. Once every cassette has a golden
file, re-run with `--assert-golden` to confirm they all still match (they
will, since you just generated them from the same tool — this second run
is mostly to confirm your files are saved correctly, and it's the
command your PR's CI will actually run going forward).

## The honest limit of "the build is green"

Passing `--assert-golden` proves your manifest is **internally
consistent**: given this exact cassette, your flow deterministically
produces this exact output, every time, byte-for-byte. It proves nothing
about whether that output is *actually what the real provider would
return* — that's a claim about the world outside this repository, and a
green build has no way to check it.

This is exactly why the first runtime to replay a manifest is,
unavoidably, **an oracle you have to trust, not a proof**. If your
cassette's recorded response doesn't actually match how the real
provider behaves (because your sandbox test hit an edge case you didn't
realize was one, or you misremembered a field name, or the provider's
docs you built a cassette from were themselves wrong), the golden file
will faithfully encode *that* mistake, and every future check against it
will pass while being wrong in the same way. A green build is a
statement about consistency, not about truth — semantics still need a
human who understands this specific provider to actually read the
`expected/*.json` output and confirm it's the response an integrator
would actually want, not just trust that "the tests pass" settles it.

This is also why `metadata.yaml`'s `verification.status` exists as a
separate axis from "does it validate and replay cleanly" — see chapter
11. A manifest can have a perfectly green build and still be
`provisional`, because those are two different questions: one about
internal consistency, one about whether it matches reality.

Next: [the clean-room rule and the DCO](09-clean-room-and-dco.md) — what
you can and can't write this manifest from.
