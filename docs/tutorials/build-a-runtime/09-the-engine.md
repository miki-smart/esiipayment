# 9. The engine

Everything up to chapter 8 is deterministic and golden-output-comparable
by construction. The engine isn't: idempotency-key storage, the
persisted payment record, concurrent-request handling, credential
storage and token refresh, retry policy. Nothing in this repository's
cassettes exercises concurrency or real I/O, so nothing here can be
tested by comparing bytes against `expected/*.json`. It needs a
different kind of test: not "does this match a fixture," but "does this
property hold under a fake designed specifically to try to break it."

## Idempotency

Per [Invariant I7](../../../spec/02-invariants.md#i7): a repeated
`collect`/`payout`/`refund` call with the same idempotency key and the
same request payload returns the original result without re-invoking
the interpreter; the same key with a **different** payload is rejected
with `FailureCode.DuplicateRequest`, never silently accepted as an
overwrite. [`vectors/errors/idempotency-conflict.json`](../../../vectors/errors/idempotency-conflict.json)
is data-driven and fits this chapter's engine tests directly, unlike
most of the earlier vector files: it needs your actual persistence layer
behind it, not just a pure function.

```text
for each case in vectors/errors/idempotency-conflict.json.cases:
    result = engine.collect(case.request)   # same idempotency_key as original_request
    if case.expected_behavior == "replay":
        assert result == the original recorded result
        assert no new interpreter invocation happened
    else:  # "reject"
        assert result.failure_code == "DuplicateRequest"
        assert no new interpreter invocation happened
        assert the existing record is unmodified
```

## The persistence port

Define persistence as an interface/port your engine depends on, not a
concrete database. This is what makes the invariant tests below possible
without a real database in your test suite at all — a good sign this
layer is properly separated, the same "independently testable with zero
infrastructure" property chapters 2 through 8 already have.

## Four invariant tests, against four adversarial fakes

Not vectors — you write these, specific to your language and test
framework, each asserting one invariant against a fake built to be
maximally unhelpful in exactly the way that invariant guards against.

**A chaos transport, asserting [Invariant I4](../../../spec/02-invariants.md#i4).**
A fake HTTP transport that randomly (or on a fixed adversarial schedule)
throws a transport fault instead of returning a response. Assert every
resulting `PaymentResult` is `Processing` + `Poll` — never `Failed` —
regardless of when the fault fires relative to the flow's own logic.

```text
transport = ChaosTransport(fault_rate: 1.0)   # always faults
for many_iterations:
    result = engine.collect(fresh_request(), transport)
    assert result.status == "Processing"
    assert result.next_action == {type: "Poll", interval_ms: 5000}
```

**A counting transport, asserting [Invariant I5](../../../spec/02-invariants.md#i5).**
A fake transport that counts how many times it was invoked for one
engine-level operation call. Assert the count is always exactly 1,
regardless of what the response says — the interpreter (chapter 7) never
retries internally, and neither should anything between the engine's
public API and the interpreter. If your engine's own retry policy is
under test too, assert its retries go through the *public* API again
(a new, deliberate top-level attempt), never by looping inside the
single interpreter invocation.

```text
transport = CountingTransport(response: some_response)
engine.collect(request, transport)
assert transport.call_count == 1
```

**A store spy, asserting [Invariant I8](../../../spec/02-invariants.md#i8).**
A fake persistence port that records the *order* of operations it
receives. Assert the payment record (idempotency key, request payload,
initial `Processing` status) is written **before** the first outbound
network call is issued, not after — this ordering is what makes crash
recovery safe: if the process dies after the network call but before
recording a result, the record already exists in `Processing` and a
resumed runtime can `sync` to find the true outcome.

```text
spy = StorePySpy()
engine.collect(request, spy_store: spy, transport: any_transport)
assert spy.operations[0] == "write_record(status=Processing)"
assert spy.operations.index_of("write_record") < spy.operations.index_of("issue_call")
```

**A serialize-and-rehydrate test, asserting [Invariant I6](../../../spec/02-invariants.md#i6).**
Run a flow partway (to a non-terminal `RequiresAction`/`Processing`
state), serialize its `state` object, discard the in-memory engine
instance entirely, construct a fresh one, deserialize `state` back into
it, and continue the flow (e.g. via `sync`). Assert the outcome is
identical to running the same flow without the round-trip. This is what
proves your `state` handling actually survives a process restart or
horizontal scale-out, not just "worked in the same process."

```text
result1 = engine.collect(request)   # -> RequiresAction, some state
serialized = serialize(engine.get_state(result1.idempotency_key))
fresh_engine = new_engine()   # no shared memory with the first one
fresh_engine.restore_state(result1.idempotency_key, deserialize(serialized))
result2 = fresh_engine.sync(result1.idempotency_key)
assert result2 == running the equivalent flow with no restart at all
```

## Credential handling and token refresh

Per [07-runtime-requirements.md#credential-handling](../../../spec/07-runtime-requirements.md#credential-handling):
never log, persist unencrypted, or otherwise expose a credential value
outside the expression context it's injected into for the duration of
the call that needs it. For `auth.shape: oauth2_client_credentials`
specifically, cache the exchanged token and refresh it at `refresh_at` of
its lifetime, and make that refresh **single-flight**: when multiple
concurrent calls observe the cached token needs refreshing, exactly one
of them performs the exchange and the rest wait on that result. This one
is worth its own concurrency test, not just a unit test of the happy
path — several real providers invalidate a token's predecessor the
instant a new one is issued, so N concurrent naive refreshes produce N-1
callers failing with an auth error, a failure mode that only shows up
under real concurrent load.

## What "done" looks like for this chapter

Four passing invariant tests against four purpose-built fakes, plus the
idempotency vector wired to your real persistence layer, plus (if you
support `oauth2_client_credentials`) a concurrency test for single-flight
refresh. None of this is golden-output-compared, and that's expected —
it's a different, necessary kind of correctness the interpreter's byte
comparisons can't reach.
