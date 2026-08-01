# The reference integrator

This is one piece of logic, described once, in language-neutral
pseudocode, that two different things in this project derive from:

1. **`docs/tutorials/use-a-provider/TEMPLATE.md`'s section 3** ("Handle
   `NextAction` with a single switch") — the tutorial every language
   SDK's own docs fill in with their idioms — snippet-includes the
   switch below directly, so the tutorial can't quietly drift from what
   this page actually says.
2. **`esiipayment conformance-integrator`**
   (`tools/validator/internal/integrator/`) — the machine-checkable form
   of the same logic, run against every cassette of every provider in
   this repository, in CI, on every change
   (`.github/workflows/integrator-promise.yml`). See
   [Invariant I12](../../../spec/02-invariants.md#i12).

The point of writing it down once, here, is that "the tutorial's example
integrator" and "the thing CI actually checks" are the same claim, not
two descriptions that happen to agree today and can drift apart later.
If you change one, change the other, and check they still say the same
thing.

## What it is

A function, `handle(payment_result)`, that decides what to do next using
**only** `PaymentStatus`, `NextAction`, `FailureCode`, and `RetryClass`
— never a provider identifier, never a key read out of `state`. It is
exhaustive over the closed sets it switches on: every branch below is
required, and encountering a value with no matching branch is the
integrator-facing form of the same promise violation
`conformance-integrator` treats as a hard failure.

<!-- --8<-- [start:handle-next-action-switch] -->
```text
function handle(payment_result):
    switch payment_result.status:

        case "Succeeded":
            return terminal("mark the order paid, fulfill it")

        case "Failed":
            assert payment_result.failure is not null
            return terminal(
                "surface the decline to the payer using failure.failure_code; "
                + "do not retry automatically unless failure.retry_class == SafeToRetry"
            )

        case "Canceled":
            return terminal("release any held inventory; the payer may start a new payment")

        case "Expired":
            return terminal("same handling as Canceled: a new payment, never a retry of this one")

        case "RequiresAction", "Processing":
            return handle_next_action(payment_result.next_action)

        default:
            fail("unrecognized PaymentStatus — this is exactly the promise violation "
                 + "esiipayment conformance-integrator exists to catch")


function handle_next_action(next_action):
    switch next_action.type:

        case "None":
            return "wait: nothing further required of the user yet"

        case "RedirectToUrl":
            assert next_action.url is not null
            return "redirect the payer to " + next_action.url

        case "AwaitDevicePush":
            assert next_action.display_ref is not null
            return "tell the payer to approve the push sent to " + next_action.display_ref

        case "SubmitOtp":
            assert next_action.length is a number
            return "prompt the payer for a " + next_action.length + "-digit code"

        case "DisplayQr":
            assert next_action.payload is not null
            return "render a QR code for " + next_action.payload

        case "ShowTransferDetails":
            assert next_action.account_number, next_action.institution,
                   next_action.reference, next_action.amount are all not null
            return "show the payer these manual transfer details"

        case "DialUssd":
            assert next_action.code is not null
            return "tell the payer to dial " + next_action.code

        case "Poll":
            assert next_action.interval_ms is a number
            return "schedule a sync retry after interval_ms"

        case "Capture":
            return "invoke the capture operation to finalize funds already authorized"

        default:
            fail("unrecognized NextAction.type — same promise violation as above")
```
<!-- --8<-- [end:handle-next-action-switch] -->

## The promise this makes concrete

Every branch above reads only fields the closed `NextAction` payload
shape defines
([01-domain-model.md#nextaction-carries-its-own-payload](../../../spec/01-domain-model.md#nextaction-carries-its-own-payload)).
Nothing here is keyed to which provider produced the result. That's not
an accident of this particular example being simple — it's the specific
thing this whole exercise is testing: **adding a tenth provider, with a
manifest nobody who wrote this switch has ever seen, requires no change
to this function.** If you find yourself wanting to add a
`case provider == "chapa"` anywhere in a real implementation of this
switch, that's the signal something upstream (the manifest, or a `state`
field being read instead of a `next_action` field) has leaked a detail
this switch was never supposed to need.

## Where the Go embodiment lives

[`tools/validator/internal/integrator/integrator.go`](../../../tools/validator/internal/integrator/integrator.go)
is this exact logic, in Go, run by `esiipayment conformance-integrator`
against every cassette of every provider in this repository. Read it
side by side with the pseudocode above if you want to see a concrete,
executed version — the two should never materially disagree; if you spot
a place where they do, that's a bug in one of them to fix, not a
deliberate difference to document.
