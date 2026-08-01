# 7. The interpreter

This is the step executor: given a loaded manifest, an operation, and one
HTTP response (or its absence), produce a `PaymentResult`. Everything
before this chapter was a pure function over static data; this chapter
is the first place your code makes (or, during replay, simulates) an
actual network call.

## The shape every manifest in this repository uses today

One outbound call (or one inbound webhook) per operation invocation,
dispatched by a single `errors`/`status_map` check against that one
response, followed by a chain of plain `emit`/`goto` steps to a terminal
leaf. Build the interpreter to this shape first — it's what every
cassette in this repository, `mock` included, actually exercises. A
manifest whose flow makes a *second* call within one operation, or
re-dispatches on a fresh `status_map` more than once per run, is out of
scope for this shape and should be a clear "not yet supported" error in
your interpreter rather than a silent wrong answer, until you have a
concrete case that needs it.

## The two lines that carry the safety

Everything else in this chapter is bookkeeping around these two rules.
Get them right and most of what makes this system trustworthy follows;
get either wrong and no amount of correct expression evaluation saves
you.

**One.** A step's `call` is attempted **exactly once**. Nothing in this
interpreter retries, backs off, or reissues a request under any
condition — that's [Invariant I5](../../../spec/02-invariants.md#i5),
and retry policy belongs entirely to the engine (chapter 9), configurable
by the integrator, informed by `RetryClass`. If you find yourself adding
a retry loop inside the step executor "just for transient errors," stop:
that decision doesn't belong at this layer, and making it here is
exactly how two runtimes end up disagreeing about how many times a
provider got called for the same idempotency key.

**Two.** A transport fault — no response received at all: a timeout, a
connection reset, anything that means you don't know what the provider
did with the request — resolves to **`Processing` + `NextAction.Poll`**,
never `Failed`, unconditionally:

```text
try:
    response = issue_the_one_call(step.call)
except TransportFault:
    return PaymentResult{
        status: "Processing",
        next_action: { type: "Poll", interval_ms: 5000 },
        state: unchanged,
        failure: null,
    }
```

`interval_ms: 5000` here is fixed, not a choice your runtime makes: since
there's no manifest step to read an interval from for a fault the
interpreter itself synthesizes, the spec pins the value so golden output
stays deterministic
([01-domain-model.md#why-interval_ms-is-required-even-for-a-runtime-synthesized-poll](../../../spec/01-domain-model.md#why-interval_ms-is-required-even-for-a-runtime-synthesized-poll)).
The same fixed `Processing`+`Poll{interval_ms: 5000}` outcome applies
when a response *was* received but matched an `errors[]` entry mapped to
`FailureCode.ProviderTimeout` or `FailureCode.Unknown` — see chapter 5;
that's the "we got a response but couldn't confirm the outcome" case,
handled identically to "we got no response at all," for the same reason:
[Invariant I4](../../../spec/02-invariants.md#i4) exists specifically to
prevent a caller ever inferring "safe to retry" from an outcome that
might already have succeeded on the provider's side.

## The rest of the dispatch loop

```text
function run(manifest, operation, cassette_or_live_request) -> PaymentResult:
    flow = manifest.flows[manifest.operations[operation].entry_flow]
    entry_step = the one step nothing else's goto/status_map targets   # chapter 6
    state = cassette.pre_existing_state ?? {}

    if entry_step has a `call`:
        response = issue_the_one_call(entry_step.call)   # or: TransportFault, see above

    doc = parse(response.body)   # or the inbound webhook body, for a webhook operation

    if errors_match := classify_error(manifest.errors, response, doc):   # chapter 5
        if errors_match.failure_code in {ProviderTimeout, Unknown}:
            return PaymentResult{status: Processing, next_action: {type: Poll, interval_ms: 5000}, ...}
        return PaymentResult{status: Failed, failure: errors_match, state: unchanged}
        # an errors match bypasses this step's own emit entirely —
        # status/next_action/state all come from nowhere else; the flow's
        # state is left exactly as it was before this step ran

    step = entry_step
    loop:
        apply_emit_state(step.emit, state, doc)   # patch, not replace — see below
        if step.emit.status: status = step.emit.status
        if step.emit.next_action: next_action = resolve_next_action(step.emit.next_action, doc, state, ...)
        if step.status_map:
            step = flow.steps[step.status_map.values[extract(doc, step.status_map.path)]]
            continue
        if step.goto:
            step = flow.steps[step.goto]
            continue
        break

    return PaymentResult{idempotency_key, operation, status, next_action, state, failure: null}
```

Three details in that sketch worth their own explanation:

- **`emit.state` is a patch, not a replacement.** The keys a step's
  `emit.state` names are merged into the flow's accumulated state; a key
  written by an earlier step and not mentioned by the current step is
  left untouched. This is what lets an entry step record something a
  later step in the same run still needs to read, without every
  subsequent step re-declaring it — see
  [Invariant I6](../../../spec/02-invariants.md#i6).
- **The same response document (`doc`) is available to every step in the
  chain, not only the step whose `call` produced it.** This is why a
  step reached purely by `goto`/`status_map` can still resolve
  `${extract...}` in its own `emit.next_action` — see chapter 6's
  namespace-availability note. Thread `doc` through the whole dispatch
  loop, not just into the entry step's own handling.
- **`resolve_next_action` needs every namespace, not only `extract`.**
  `NextAction.ShowTransferDetails.reference: ${idempotency_key}` and
  `amount: { minor_units: ${intent.amount}, currency: ${intent.currency}
  }` are both realistic, both correct, and neither is an `extract`
  reference — resolve `emit.next_action`'s fields (and, while you're at
  it, `emit.state`'s values) through the *same* general interpolation
  resolver from chapter 3, not a narrower one that only understands
  `${extract...}`. This also gets you nested-object resolution
  (`ShowTransferDetails.amount`) for free, since a general resolver
  recurses through a map/object the same way regardless of which
  manifest field it's resolving.

## What "done" looks like for this chapter

A function that takes a manifest, an operation, and one HTTP response
(or a simulated transport fault) and returns a `PaymentResult` matching
this shape. You don't have cassettes wired up yet — that's chapter 8 —
but you should be able to hand-construct one fake response per
`NextAction` variant and confirm each one produces the right shape,
including the two safety rules above under a hand-injected transport
fault.
