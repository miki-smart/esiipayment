# 8. Replay against `mock` first

This is the highest-leverage chapter in this track. `mock`
([providers/mock/](../../../providers/mock/)) is not a real payment
provider — it has no credentials, makes no real network calls, and is
completely deterministic by construction: every `PaymentStatus` and every
`NextAction` variant can be produced on demand by choosing the right
`method` value on the request. It exists so you can build and prove an
entire interpreter before a single real provider's manifest is involved.

## The harness

```text
function replay(provider_dir):
    manifest = load(provider_dir/manifest.yaml)     # chapter 6
    for cassette in provider_dir/cassettes/*.yaml:
        result = run(manifest, cassette.operation, cassette)   # chapter 7,
                                                                # fed the cassette's
                                                                # recorded response
                                                                # instead of a real call
        canonical = canonicalize(result)                        # chapter 4
        expected = read(provider_dir/expected/<cassette-name>.json)
        assert canonical == expected   # byte-for-byte, not "logically equal"
```

Two things this harness must do that are easy to gloss over:

- **Assert the outbound request matches the cassette's recorded
  expectation before feeding back the recorded response** (method, path,
  and, where the cassette specifies it, body) — replaying a cassette
  isn't just "return this response," it's confirming your interpreter
  would have sent the right request to get it.
- **Inject the cassette's `seed` (clock, UUID sequence) instead of your
  runtime's real clock/UUID source.** [04-expression-language.md#determinism](../../../spec/04-expression-language.md#determinism)
  and [06-conformance.md](../../../spec/06-conformance.md#why-determinism-has-to-be-structural-not-disciplinary)
  are both explicit that this has to be a structural seam in your
  interpreter, not a discipline you trust yourself to maintain — if
  anything in the call path reads the real wall clock or generates a
  real random UUID during replay, the same cassette produces different
  output on different runs, and "byte-identical to `expected/`" stops
  being a meaningful claim.

## What `mock`'s 20 cassettes cover

Every `PaymentStatus`/`NextAction` combination this project defines, plus
the mandatory transport-timeout case, across every operation `mock`
supports (`collect`, `sync`, `payout`, `cancel`, `refund`, `webhook`).
Once all 20 pass byte-for-byte, you have independently exercised: every
step-executor code path from chapter 7, every `NextAction` variant's
field-resolution logic (including the nested `ShowTransferDetails.amount`
object and literal-vs-extracted fields — see `collect.transfer.yaml`),
the `errors`-vs-`status_map` precedence, and the transport-timeout →
`Processing`+`Poll` rule, all without a single real provider's manifest
involved.

## Then the real providers are nearly free

Once `mock` is fully green, replay `chapa`, `arifpay`, and `santimpay`'s
cassettes the same way, with the same harness, unmodified. If your
interpreter is correct against `mock`, these should mostly just pass —
they exercise a subset of what `mock` already covers (hosted-checkout
`RedirectToUrl` + `Poll`, mainly), plus one thing `mock` can't: real
`auth.apply` header injection (see
[03-manifest-dsl.md#auth](../../../spec/03-manifest-dsl.md#auth)) — `mock`'s
`auth.shape` is `none`, so it never exercises credential attachment at
all. If a real-provider cassette fails and `mock` didn't catch it, that's
almost always in one of two places: `auth.apply` resolution (confirm the
resolved header/query/body value against the cassette's recorded
request), or a field this specific provider's manifest uses that no
`mock` cassette happens to exercise the same way.

## When a mismatch isn't obvious

A byte-level diff between your canonical output and `expected/*.json`
tells you *where* the strings differ, not *why*. Work outward from the
diff position: a key-ordering difference means chapter 4; a value that's
a string where it should be an integer (or vice versa) means chapter 3's
type-preservation rule; a missing field on a `next_action` object means
chapter 6's required-field check didn't catch a bad manifest at load
time, or chapter 7's `resolve_next_action` isn't resolving a nested
field. Isolate which chapter's function actually produced the wrong
value before assuming the bug is "in the interpreter" generally — by
this point every earlier chapter has its own vector-backed test suite
specifically so you can rule each one in or out in isolation, rather than
re-debugging all of them at once every time replay fails.

## What "done" looks like for this chapter

`esiipayment replay providers/mock --assert-golden`'s equivalent, in your
own runtime, passing all 20 cassettes byte-for-byte — then `chapa`,
`arifpay`, and `santimpay` passing too. This is the first point in the
whole track where you have something that looks like a working
interpreter end to end; everything before this was proving the pieces
underneath it.
