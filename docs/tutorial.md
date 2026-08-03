# Tutorial: add a payment provider, then build a runtime

This is the hands-on guide. You can follow it with no prior knowledge of
this project, and every command in it has been run exactly as written.

There are two independent parts. **You almost certainly only need one of
them.** Skip to whichever fits:

- **[Part 1 — Add a payment provider](#part-1--add-a-payment-provider)**
  You know a payment company's API and want this project to support it.
  No programming required. About 45 minutes.
- **[Part 2 — Build a runtime](#part-2--build-a-runtime)**
  You want this project to work in a programming language it doesn't
  support yet. You need to know that language well. A few days.

---

## Read this first

Five minutes that will make both parts make sense.

### The problem this project solves

Say you run an online shop in Ethiopia and want to accept payments. You
pick a payment company — Chapa, say — read their documentation, and write
code that talks to them.

Six months later you want to also accept Telebirr. Different company,
different API. So you write that code too. Now your checkout page has
two ways of doing everything, and an `if` statement choosing between
them. Add a third provider and it gets worse. Switch programming
languages and you start over from zero.

Every shop in the country is writing this same code, separately, for the
same handful of providers.

### The idea

Split the problem in two:

> **What a provider does** is a fact about that provider. Write it down
> once, as plain data, in a file everyone shares.
>
> **How to follow those instructions** is a programming problem. Solve
> it once per programming language.

Think of a restaurant. A recipe card says "sauté the onions for five
minutes." That card is the same whether the cook is in Addis or Toronto.
The recipe doesn't know how to cook — it just describes. The cook doesn't
know this particular dish in advance — they just follow cards.

- The **recipe card** is a `manifest.yaml`. Part 1 teaches you to write one.
- The **cook** is a runtime. Part 2 teaches you to build one.

One recipe card works for every cook, in every language. That's the whole
trick.

### The words you need

Only eight. Every other term in this project is built from these.

| Word | Plain meaning |
|---|---|
| **Provider** / **PSP** | A payment company. Chapa, Telebirr, ArifPay. (PSP = "payment service provider".) |
| **Manifest** | One YAML file describing one provider's behaviour. The recipe card. No code in it — just data. |
| **Runtime** | A library, in one programming language, that reads manifests and does what they say. The cook. |
| **Integrator** | A person building a shop who uses a runtime. The customer eating the food. |
| **Flow** | A sequence of steps for one task, like "start a payment". Written inside a manifest. |
| **Cassette** | A recording of a real conversation with a provider — the request sent, the response received. Like a tape recording of a phone call, so you can replay it later without calling again. |
| **Golden file** | The exact result a cassette should produce. If replaying the cassette gives something different, something broke. |
| **Idempotency key** | A unique ID for one payment attempt. Send the same request twice with the same key, and you get charged once, not twice. |

### Two rules that explain most of the design

**Rule 1: money is always whole numbers.**
150 birr is stored as `15000` — the number of santim. Never `150.00`.
Computers can't store decimals like `0.1` exactly, and a payment system
that loses a santim now and then isn't a payment system.

**Rule 2: "we don't know" is never treated as "it failed."**
If you send a payment request and the network dies before you hear back,
you genuinely don't know whether the customer was charged. Guessing
"failed" and retrying could charge them twice. So this project has a
specific answer for that case, and Part 1 and Part 2 both enforce it.

---

## Part 1 — Add a payment provider

**Who this is for:** you know how a payment company's API works, either
from their documentation or from having integrated it before.

**What you need:** a text editor and [Docker](https://docs.docker.com/get-docker/).
Nothing else. No programming language, no compiler.

**What you'll have at the end:** a complete provider description that
passes every automated check, and a real understanding of how to write
one for the provider *you* care about.

We'll build a fictional provider called **Selam Pay**. Fictional so
nothing here is a wrong claim about a real company — but shaped exactly
like the most common real pattern in Ethiopia (Chapa and ArifPay both
work this way): you send the payment details, they give you back a web
page URL, your customer pays there, then you check whether it worked.

### Step 0 — Get the tool

One command, run from the repository root. It builds a small program that
checks your work.

```sh
docker build -t esiipayment-validator tools/validator
```

Takes a minute the first time. To confirm it worked:

```sh
docker run --rm esiipayment-validator --help
```

You should see a list of commands: `validate`, `replay`, `lint`,
`catalog`, `conformance-integrator`.

> **Windows note.** If you're using Git Bash and get an error mentioning
> `C:/Program Files/Git/work`, run `export MSYS_NO_PATHCONV=1` first. Git
> Bash mangles paths that start with `/`. This bites everyone once.

To save typing, set up a shortcut. Run this from the repository root:

```sh
alias esiipayment='MSYS_NO_PATHCONV=1 docker run --rm -v "$PWD:/work" -w /work esiipayment-validator'
```

Now `esiipayment validate ...` works like a normal command. (The alias
lasts until you close the terminal.)

### Step 1 — Interview the provider

Before writing anything, answer these seven questions about your
provider. Everything in the manifest comes from these answers. Here are
Selam Pay's:

| Question | Selam Pay's answer |
|---|---|
| 1. What are its web addresses, for testing and for real? | `https://sandbox.selampay.example/api` and `https://api.selampay.example/api` |
| 2. How do you prove who you are? | One secret key, sent as `Authorization: Bearer <key>` |
| 3. What can it do? | Start a payment; check a payment's status |
| 4. How do you start a payment? | `POST /v1/payments` with amount, currency, your reference, and a callback URL |
| 5. What does it reply? | `{"data": {"state": "pending", "payment_id": "...", "checkout_url": "..."}}` |
| 6. How does the customer actually pay? | You send them to `checkout_url` in their browser |
| 7. What can go wrong, and how would you tell? | Not enough money (`reason: insufficient_funds`); bad key (HTTP 401); the network dies |

If you can answer these for your provider, you can write its manifest.
If you can't, that's the real work — go read their docs first.

### Step 2 — Copy the template

```sh
cp -r providers/_template providers/selampay-tutorial
```

The folder name must match the name you'll put inside the file.

> **Heads up:** we're using `selampay-tutorial` because it's practice, not
> a real contribution. When you do this for real, use the provider's
> actual name: `providers/chapa`, `providers/telebirr`.

Your folder needs four things, and we'll fill them in that order:

```
providers/selampay-tutorial/
  manifest.yaml     what the provider does          <- the main file
  metadata.yaml     who wrote this and how sure they are
  cassettes/        recorded conversations
  expected/         what each recording should produce
```

### Step 3 — Identity

Open `providers/selampay-tutorial/manifest.yaml`, delete everything, and
start fresh. First, who this provider is:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
provider: selampay-tutorial
spec_version: "2.0"
display_name: "Selam Pay (tutorial example)"
country: ET
currencies: [ETB]

environments:
  sandbox:
    base_url: https://sandbox.selampay.example/api
  production:
    base_url: https://api.selampay.example/api
```

Line by line:

- **The first line is required** and must be exactly that. It tells your
  editor where the rules live, so you get red squiggles and autocomplete
  as you type. Install the YAML extension for VS Code and you'll see
  mistakes immediately instead of at validation time. A file missing this
  line fails the checks.
- **`provider`** must match the folder name. Lowercase.
- **`spec_version`** is the version of *this format*, not of your
  provider. Leave it at `"2.0"`.
- **`environments`** are the base web addresses. You'll never write a
  full URL anywhere else — steps say `/v1/payments` and the runtime
  sticks the right base on the front, depending on whether the shop is
  testing or live.

### Step 4 — Authentication

How the runtime proves it's allowed to call this provider:

```yaml
auth:
  shape: api_key
  fields:
    - name: secret_key
      description: Secret key from the Selam Pay merchant dashboard, sent as a bearer token on every request.
  apply:
    header: Authorization
    value: "Bearer ${credentials.secret_key}"
```

Three parts:

- **`shape`** — pick one of exactly seven: `none`, `api_key`, `key_pair`,
  `triple_key`, `oauth2_client_credentials`, `hmac`, `mutual_tls`. Pick
  the closest one. Don't invent a new one, even if your provider's scheme
  is slightly different — the whole point is that every runtime can build
  one credential form that works for everybody.
- **`fields`** — the *names* of the secrets a shop must supply.
- **`apply`** — where the secret goes on every outgoing request. Say it
  once here, and every step below gets it automatically.

> ### 🔒 Never put a real secret in this file
>
> Notice we wrote `${credentials.secret_key}` — a *placeholder*, not a
> key. Manifests live in a public repository. The actual key comes from
> the shop's own configuration at run time and never touches this file.
>
> The `${...}` syntax means "fill this in later." You'll see it a lot.

### Step 5 — Capabilities

What this provider can do, and its limits:

```yaml
capabilities:
  operations: [collect, sync]
  next_actions: [RedirectToUrl, Poll]
  currencies:
    ETB:
      min_amount: 100
      max_amount: 100000000

operations:
  collect:
    entry_flow: collect
  sync:
    entry_flow: sync
```

**`operations`** — there are exactly six possible, and you list the ones
your provider supports:

| Operation | Meaning |
|---|---|
| `collect` | Take money from a customer |
| `sync` | Ask "did that payment work?" |
| `payout` | Send money to someone |
| `cancel` | Call off a payment that hasn't finished |
| `refund` | Give money back |
| `webhook` | Receive a notification the provider sends you |

Selam Pay does the first two.

**`next_actions`** — what the *customer* might have to do. There are
exactly nine possibilities in the whole system:

| Action | What the customer does |
|---|---|
| `None` | Nothing — just wait |
| `RedirectToUrl` | Go to a web page and pay there |
| `AwaitDevicePush` | Approve a prompt on their phone |
| `SubmitOtp` | Type in a code sent by SMS |
| `DisplayQr` | Scan a QR code |
| `ShowTransferDetails` | Manually transfer to an account number |
| `DialUssd` | Dial something like `*127#` |
| `Poll` | Nothing — the *shop* checks again shortly |
| `Capture` | Nothing — the shop finalises a held payment |

**Why exactly nine?** Because a shop's checkout page can then handle all
nine and be finished forever. Adding a tenth provider never adds a tenth
case. This is the promise the whole project exists to keep.

Pick based on **what the customer physically does**, never on what the
provider calls it internally. Selam Pay sends customers to a web page, so
that's `RedirectToUrl`. When we check status and it's not done yet, the
shop must check again — that's `Poll`.

> **This list must match exactly.** Not "at least" — exactly. If you
> declare `Poll` but never actually produce it, validation fails. It's a
> promise, and it's checked.

**`min_amount` / `max_amount`** are in santim: `100` = 1.00 birr. (Rule
1 — whole numbers only.)

### Step 6 — The collect flow

Here's where it gets interesting. A **flow** is a set of named **steps**.
Each step either calls the provider, or decides what to tell the shop, or
both.

```yaml
flows:
  collect:
    steps:
      start_payment:
        call:
          method: POST
          path: /v1/payments
          body:
            amount: ${amount_major(intent.amount)}
            currency: ${intent.currency}
            reference: ${idempotency_key}
            callback_url: ${ctx.webhook_url}
        triggers: [return]
        status_map:
          path: $.data.state
          values:
            pending: send_user_to_checkout
        emit:
          state:
            payment_id: ${extract.data.payment_id}

      send_user_to_checkout:
        emit:
          status: RequiresAction
          next_action:
            type: RedirectToUrl
            url: ${extract.data.checkout_url}
```

Read it as a story:

**Step `start_payment`:** send a POST to `/v1/payments`. Build the body
from four placeholders:

| Placeholder | Where the value comes from |
|---|---|
| `${amount_major(intent.amount)}` | The shop's amount (`15000` santim), converted to `"150.00"` because this provider wants decimals. `amount_major` is a built-in converter. |
| `${intent.currency}` | The shop's currency — `"ETB"` |
| `${idempotency_key}` | The unique ID for this payment attempt |
| `${ctx.webhook_url}` | The shop's own callback address |

`triggers: [return]` means "this step runs when the reply arrives."

**`status_map`** is the decision. It says: look at `$.data.state` in the
reply — that means "the `state` field inside the `data` object." If it
says `pending`, go to the step named `send_user_to_checkout`.

**`emit.state`** saves something for later. We save `payment_id` because
the sync flow will need it to ask about *this* payment. Think of `state`
as a sticky note the runtime keeps attached to the payment.

**Step `send_user_to_checkout`:** no call — it just reports the outcome.
Status `RequiresAction` means "not done, the customer must do something."
And the something is: go to this URL.

Three things worth understanding properly:

> **Where does the flow start?**
> Nowhere does it say "start here." The starting step is worked out by
> elimination: it's the one step no other step points to. Here,
> `send_user_to_checkout` is pointed at by `status_map`, so
> `start_payment` must be the start.
>
> Why do it this way? Because YAML doesn't reliably preserve ordering
> across programming languages, so "the first one" isn't a safe rule.
> This is why a flow must have **exactly one** step nothing points at —
> zero means no entry point, two means an ambiguous one, and both fail
> validation.

> **Why does the URL go in `next_action` and not in `state`?**
> This is the most important design rule in the project, so it's worth
> being explicit.
>
> Every provider names its checkout URL differently — `checkout_url`,
> `payment_url`, `toPayUrl`. If we dumped that into the general-purpose
> `state` bag, every shop would end up writing
> `state.checkout_url ?? state.payment_url ?? state.toPayUrl`. That's the
> provider-specific code the project exists to eliminate, just hidden.
>
> So: `next_action.url` is a **fixed name** every provider maps onto.
> `state` is only for private bookkeeping the *shop never reads* — like
> `payment_id` above.

> **Why is the amount converted?**
> The shop always speaks in whole santim (Rule 1). Providers vary — some
> want `"150.00"`, some want `15000`. `amount_major` bridges that. There
> are exactly eight such converters (`amount_major`, `msisdn_et`,
> `base64`, `hex`, `sha256_hex`, `upper`, `lower`, `iso8601`) and you
> can't add your own. Deliberately: the moment manifests can run
> arbitrary logic, every runtime has to reimplement that logic
> identically, and they won't.

### Step 7 — The sync flow

"Did that payment work?"

```yaml
  sync:
    steps:
      check_payment:
        call:
          method: GET
          path: /v1/payments/${state.payment_id}
        triggers: [return]
        status_map:
          path: $.data.state
          values:
            paid: paid
            failed: payment_failed
            pending: still_waiting
        emit: {}

      paid:
        emit:
          status: Succeeded

      payment_failed:
        emit:
          status: Failed

      still_waiting:
        emit:
          status: Processing
          next_action:
            type: Poll
            interval_ms: 5000
```

Note `${state.payment_id}` in the path — that's the sticky note from the
collect flow being read back.

Three outcomes, three steps. There are exactly six statuses in the whole
system:

| Status | Finished? | Meaning |
|---|---|---|
| `RequiresAction` | no | Waiting on the customer |
| `Processing` | no | In flight — including "we genuinely don't know" |
| `Succeeded` | **yes** | Money confirmed |
| `Failed` | **yes** | Confirmed not to have worked |
| `Canceled` | **yes** | Called off deliberately |
| `Expired` | **yes** | Ran out of time |

The four marked "yes" are **final**. Once a payment reaches one, it can
never change again — not even to a different final status. A payment that
can un-succeed is a payment you can't do accounting with.

> **Where's "unknown"?** There isn't one, on purpose. An unknown outcome
> is `Processing` + `Poll` — "not finished, check again." Adding an
> `Unknown` status would force every shop to write a special branch for
> it, and that branch's correct behaviour is always "check again"
> anyway. This is Rule 2, in the type system.

### Step 8 — Errors

What can go wrong and what it means:

```yaml
errors:
  - match:
      path: $.data.reason
      equals: insufficient_funds
    failure_code: InsufficientFunds
    retry_class: DoNotRetry

  - match:
      http_status: 401
    failure_code: AuthFailed
    retry_class: DoNotRetry

  - match:
      transport: timeout
    failure_code: ProviderTimeout
    retry_class: ResolveFirst
```

Each entry pairs a way of recognising a problem with what it means.
There are three ways to recognise one: a field in the reply
(`path`/`equals`), an HTTP status code (`http_status`), or the network
dying (`transport: timeout`).

`failure_code` comes from a fixed list of fourteen. `retry_class` says
whether retrying is safe, and there are three:

- **`SafeToRetry`** — go ahead, nothing happened.
- **`DoNotRetry`** — it definitively failed. Retrying won't help.
- **`ResolveFirst`** — **we don't know what happened.** Find out before
  doing anything else.

> ### ⚠️ The single most important line in this file
>
> ```yaml
>   - match:
>       transport: timeout
>     failure_code: ProviderTimeout
>     retry_class: ResolveFirst
> ```
>
> **Every manifest must have this, exactly.** It's Rule 2 again.
>
> A timeout means your request went out and no answer came back. The
> provider may have received it, charged the customer, and just failed to
> reply in time. If you call that "failed" and retry, you charge the
> customer twice.
>
> `ResolveFirst` means: before doing *anything* else with this payment,
> go find out what actually happened. Then decide.
>
> You don't get to choose the pairing — `failure_code` → `retry_class` is
> fixed for all fourteen codes ([the table][retry-table]) and validation
> rejects any disagreement. Retry safety is too important to leave to
> each contributor's judgement.

[retry-table]: ../spec/01-domain-model.md#retryclass

**Errors are checked before `status_map`.** If a reply matches an error
entry, that's the answer, and the step's own `emit` is skipped entirely.
That's why `start_payment`'s `status_map` only handles `pending` — a
declined payment never reaches it.

**Only one error entry may match any given reply.** Two entries that
could both match the same response is an authoring bug, and validation
flags it. (This is a real bug we hit against Chapa's live API: their 401
response *also* contains `"status":"failed"`, so an error entry matching
on that field caught the auth failure too, and reported a bad API key as
a validation error.)

### Step 9 — Say how sure you are

`metadata.yaml` records who wrote this and how much to trust it:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/metadata.v1.schema.json
provider: selampay-tutorial
tier: community
description: >-
  A fictional hosted-checkout provider used by docs/tutorial.md to teach the
  manifest format. Not a real payment provider.
verification:
  status: provisional
  docs_verified_on: null
  verified_by: null
  notes: >-
    Entirely fictional. Selam Pay does not exist; this manifest is a
    teaching example in docs/tutorial.md, shaped after the hosted-checkout
    pattern that Chapa and ArifPay both really use. Nothing here was or
    could be verified against a real provider.
maintainers: []
links: {}
```

**This file matters more than it looks.** `status: provisional` means "I
wrote this from memory or from docs I haven't rechecked." `verified`
means someone confirmed it against current official documentation.

Be honest here. Someone will build a real shop on your manifest. Writing
`verified` because it seems probably right is how a shop loses money.
Most manifests in this repository are `provisional` and say so plainly —
including the real Telebirr one, which openly documents that it omits a
signing step it can't express.

### Step 10 — Record cassettes

A **cassette** is a recording: what request should go out, what reply
comes back. Replaying one lets anyone check the manifest without
credentials, without a network, and get the identical result every time.

You need at minimum four: a success, a provider-reported failure, an auth
failure, and a timeout. Plus coverage for each operation you declared.

`cassettes/collect.success.yaml`:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/cassette.v1.schema.json
name: collect.success
operation: collect
seed:
  clock: "2026-01-15T09:30:00Z"
  uuid: []
  idempotency_key: "SELAM-COLLECT-SUCCESS-0001"
environment: sandbox
ctx:
  webhook_url: "https://myshop.example/webhooks/selampay"
credentials:
  secret_key: "sk_test_fixture-not-a-real-key"
intent:
  amount: 15000
  currency: ETB
interactions:
  - request:
      method: POST
      path: /v1/payments
      headers: { Authorization: "Bearer sk_test_fixture-not-a-real-key" }
      body: '{"amount":"150.00","currency":"ETB","reference":"SELAM-COLLECT-SUCCESS-0001","callback_url":"https://myshop.example/webhooks/selampay"}'
    response:
      status: 200
      headers: { content-type: application/json }
      body: '{"data":{"state":"pending","payment_id":"pay_abc123","checkout_url":"https://sandbox.selampay.example/pay/pay_abc123"}}'
```

**`seed`** is what makes replay reproducible. Instead of the real clock
and real random IDs, the runtime is handed these fixed values. Same
input every time means same output every time — which is the only way
"the answer must be exactly this" can be a meaningful test.

**`credentials`** here are fake test values, never real ones.

Notice the request body has `"amount":"150.00"` — the converted form.
The cassette records what *actually goes on the wire*.

The timeout cassette is the odd one — no reply at all:

```yaml
name: collect.timeout
operation: collect
seed:
  clock: "2026-01-15T09:30:00Z"
  uuid: []
  idempotency_key: "SELAM-COLLECT-TIMEOUT-0001"
environment: sandbox
ctx:
  webhook_url: "https://myshop.example/webhooks/selampay"
credentials:
  secret_key: "sk_test_fixture-not-a-real-key"
intent:
  amount: 15000
  currency: ETB
transport_failure: true
interactions:
  - request:
      method: POST
      path: /v1/payments
      headers: { Authorization: "Bearer sk_test_fixture-not-a-real-key" }
      body: '{"amount":"150.00","currency":"ETB","reference":"SELAM-COLLECT-TIMEOUT-0001","callback_url":"https://myshop.example/webhooks/selampay"}'
```

`transport_failure: true`, and no `response:` block.

The sync cassette needs the sticky note to already exist, which is what
`state:` is for:

```yaml
name: sync.succeeded
operation: sync
seed:
  clock: "2026-01-15T09:35:00Z"
  uuid: []
  idempotency_key: "SELAM-COLLECT-SUCCESS-0001"
environment: sandbox
credentials:
  secret_key: "sk_test_fixture-not-a-real-key"
intent: {}
state:
  payment_id: "pay_abc123"
interactions:
  - request:
      method: GET
      path: /v1/payments/pay_abc123
      headers: { Authorization: "Bearer sk_test_fixture-not-a-real-key" }
    response:
      status: 200
      headers: { content-type: application/json }
      body: '{"data":{"state":"paid","payment_id":"pay_abc123"}}'
```

The remaining two (`collect.declined.yaml`, `collect.auth_failed.yaml`)
follow the same shape — see the [appendix](#appendix-the-complete-selam-pay-files).

### Step 11 — Write the golden files

For each cassette, one file in `expected/` saying exactly what it should
produce. `expected/collect.success.json`:

```json
{"failure":null,"idempotency_key":"SELAM-COLLECT-SUCCESS-0001","next_action":{"type":"RedirectToUrl","url":"https://sandbox.selampay.example/pay/pay_abc123"},"operation":"collect","state":{"payment_id":"pay_abc123"},"status":"RequiresAction"}
```

**The formatting is not cosmetic.** Keys in alphabetical order, no
spaces, no line breaks, all on one line. Every runtime in every language
must produce these exact bytes. Byte-for-byte comparison is what makes
"all our runtimes agree" checkable by a machine rather than a promise.

The other four:

```json
{"failure":{"failure_code":"InsufficientFunds","retry_class":"DoNotRetry"},"idempotency_key":"SELAM-COLLECT-DECLINED-0001","next_action":null,"operation":"collect","state":{},"status":"Failed"}
```
```json
{"failure":{"failure_code":"AuthFailed","retry_class":"DoNotRetry"},"idempotency_key":"SELAM-COLLECT-AUTH-0001","next_action":null,"operation":"collect","state":{},"status":"Failed"}
```
```json
{"failure":null,"idempotency_key":"SELAM-COLLECT-TIMEOUT-0001","next_action":{"interval_ms":5000,"type":"Poll"},"operation":"collect","state":{},"status":"Processing"}
```
```json
{"failure":null,"idempotency_key":"SELAM-COLLECT-SUCCESS-0001","next_action":null,"operation":"sync","state":{"payment_id":"pay_abc123"},"status":"Succeeded"}
```

Look closely at the timeout one: `"status":"Processing"`, `"failure":null`,
and a `Poll` action. **Not** `Failed`. Rule 2, now enforced by a test.

Also notice `"state":{}` on the failures. When an error matches, the
step's `emit` is skipped entirely — including its state — so nothing was
saved.

### Step 12 — Check your work

Two commands. This is the moment of truth.

```sh
esiipayment validate providers/selampay-tutorial
```

```
no findings
```

That's a pass. It checked the file structure, that every step target
exists, that exactly one entry step exists, that your declared
`next_actions` match what you actually produce, that every retry class
agrees with the fixed table, that no two error entries collide, and more.

```sh
esiipayment replay providers/selampay-tutorial --assert-golden
```

```
ok   collect.auth_failed.yaml
ok   collect.declined.yaml
ok   collect.success.yaml
ok   collect.timeout.yaml
ok   sync.succeeded.yaml
```

Every cassette produced its golden file exactly. **Your manifest works.**

### When it doesn't pass

Real messages you're likely to see:

| Message | What it means | Fix |
|---|---|---|
| `flow "collect" has 2 entry steps` | Two steps that nothing points to | Point one at the other, or merge them |
| `flow "collect" has no entry step` | Everything is pointed at — probably a loop | Find the step that shouldn't be a target |
| `capabilities.next_actions declares [X] but flows emit [Y]` | Your promise doesn't match reality | Make the lists identical |
| `retry_class DoNotRetry disagrees with the fixed table` | You paired a code with the wrong class | Look it up in [the table][retry-table] |
| `goto target "foo" is not a step` | Typo in a step name | Check spelling |
| `two errors entries could match the same response` | Ambiguous error matching | Make the conditions mutually exclusive |
| `golden mismatch` + a diff | Replay produced something different | Read the diff. Usually a wrong field path or a missing `state` key |

For a golden mismatch, the diff shows expected vs actual. Trust the
diff — if the actual output is right, update the golden file; if not,
fix the manifest.

### Clean up

```sh
rm -rf providers/selampay-tutorial
```

It was practice. When you do this for real, keep the folder and open a
pull request — see [docs/tutorials/add-a-provider/10-opening-the-pr.md](tutorials/add-a-provider/10-opening-the-pr.md).

### What you now know

- A provider is described as **data**. You wrote no code.
- The nine `NextAction`s and six statuses are the entire vocabulary a
  shop ever sees. That's what makes adding providers free for them.
- Fixed field names (`next_action.url`) are what keep provider
  differences from leaking out through the back door.
- Errors are checked before the happy path, and a match cancels the
  step's own output.
- A timeout is never a failure.
- Every claim is checked, byte for byte.

**Next:** [docs/tutorials/add-a-provider/](tutorials/add-a-provider/index.md)
covers the parts a fictional example can't — recording cassettes from
real sandbox traffic, the clean-room rules for writing about an API you
integrated at work, and the review process.

---

## Part 2 — Build a runtime

**Who this is for:** you want this project to work in a language it
doesn't support yet.

**What you need:** solid knowledge of one programming language, and a
YAML and JSON library for it. **You do not need to know anything about
any payment provider.** That's the point of the split.

**What you'll have at the end:** a library that can execute any
conformant manifest, proven against the reference provider.

This part walks through how the C# runtime
([esiipayment-dotnet](https://github.com/miki-smart/esiipayment-dotnet))
was actually built — the order, the reasoning, and the things that went
wrong. It's C# in the examples, but nothing about the approach is
C#-specific.

### Your test subject: the mock provider

Before anything else, meet [`providers/mock/`](../providers/mock/).

It's a fake provider that exists solely so runtime authors have something
to build against. It has no credentials, needs no network, and is
completely predictable. Its trick: the caller picks a scenario by name,
and the mock's flow routes on it.

```yaml
status_map:
  path: $.data.scenario
  values:
    succeeded: succeeded
    redirect: ra_redirect
    otp: ra_otp
    qr: ra_qr
    poll: proc_poll
    # ... one per scenario
```

It ships **20 cassettes** covering every status and every one of the nine
actions. Get all 20 replaying byte-identically and your runtime is
genuinely working. Build against this, not against a real provider.

### The order that works

Build in dependency order, and **prove each layer before starting the
next**. This matters more than usual here, because errors compound: a
subtly wrong JSON writer produces mismatches in the flow executor that
look like flow bugs and cost hours.

```
1. Canonical JSON        no dependencies       ~200 lines
2. Expressions           needs nothing         ~600 lines
3. Manifest loading      needs expressions     ~700 lines
4. Flow execution        needs all above       ~500 lines
   ── replay all 20 mock cassettes ──          the milestone
5. Storage, credentials, webhooks              ~600 lines
6. Public API                                  ~200 lines
```

Roughly 2,800 lines for a complete runtime. Smaller than it sounds,
because the manifest format is deliberately tiny.

Every layer has ready-made test data in [`vectors/`](../vectors/) — input
and required output, as JSON. Don't invent your own tests for these
layers; the vectors *are* the specification.

### Stage 1 — Canonical JSON

**Goal:** turn a result into bytes, the same bytes every runtime
produces.

**Test data:** `vectors/canonical-json/`

Four rules:

1. Object keys sorted alphabetically, at every level of nesting
2. No whitespace at all — no spaces after `:` or `,`, no newline at the end
3. Whole numbers never printed as decimals: `10000`, never `10000.0`
4. Times as `2026-01-15T09:30:00Z` — UTC, whole seconds, literal `Z`

Start here because everything else is graded against its output.

**The C# trap that will get you:** `System.Text.Json` does *not* sort
keys — a `Dictionary` keeps insertion order. And its default text encoder
escapes far more than JSON requires (`+` becomes `\u002B`), so the bytes
won't match. You must sort yourself, and set the encoder explicitly:

```csharp
var options = new JsonWriterOptions
{
    Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping,  // minimal escaping
    Indented = false,
};
```

Despite the name, `UnsafeRelaxedJsonEscaping` is correct here — "unsafe"
means "don't add HTML-injection escaping," which is exactly right for
JSON that isn't being pasted into a web page.

Your language will have its own version of this trap. Find it now, with
the vectors, rather than later.

→ [`Serialization/CanonicalJson.cs`](https://github.com/miki-smart/esiipayment-dotnet/blob/main/src/Esiipayment.Core/Serialization/CanonicalJson.cs)

### Stage 2 — Expressions

**Goal:** evaluate the `${...}` placeholders.

**Test data:** `vectors/expressions/`, `vectors/money/`

Four small pieces:

**Extraction** — `$.data.checkout_url` pulls a value out of a JSON reply.
A deliberately tiny subset of JSONPath: dotted names, `[0]` indexes,
`[*]` wildcards. No filters, no recursive descent, no expressions. Small
enough to hand-write in an afternoon; don't reach for a library, because
libraries implement *more* than this subset and you'd accept manifests
other runtimes reject.

**Interpolation** — filling `${...}` from named sources: `intent` (what
the shop asked for), `credentials`, `ctx`, `state`, `extract` (the
current reply), `event` (a webhook body), `idempotency_key`, `auth`.

**Transforms** — the eight converters. `amount_major` is the one with a
real trap: **the number of decimal places is per-currency, not always
two.** Read it from `vectors/money/exponents.json`. Hardcoding `2` passes
every ETB test and silently corrupts other currencies.

**Conditions** — comparisons for `status_map` and `errors`. Just `==`,
`!=`, `>`, `<`, `in`. No `and`/`or`.

> **The rule underneath all of this:** evaluation must be a pure
> function of its inputs. No reading the clock, no random numbers. Your
> runtime takes a clock and an ID generator as *parameters* — in
> production they're the real ones; in tests they're the cassette's fixed
> seed. Without that seam, replay isn't reproducible and none of the
> golden tests mean anything.

→ [`Expressions/`](https://github.com/miki-smart/esiipayment-dotnet/tree/main/src/Esiipayment.Core/Expressions)

### Stage 3 — Manifest loading

**Goal:** read a manifest, and reject bad ones clearly.

Three sub-steps, in order:

**YAML → JSON tree.** Convert straight into your JSON library's node
type. Don't deserialize into typed classes yet, and don't round-trip
through a string — you'd lose the distinction between `1` and `1.0`,
which Rule 1 cares about.

**Validate against the schema.** `schema/manifest.v1.schema.json` already
encodes every structural rule. Use a real JSON Schema library
(C#: `JsonSchema.Net`). Hand-writing the equivalent checks means drifting
from the schema the moment it's updated.

**Semantic checks** — the things a schema can't express:

- `spec_version` is one you support. If not, **refuse with a clear
  error**. Never guess at a format version you weren't built for.
- Exactly one entry step per flow (the elimination rule from Part 1).
- `capabilities.next_actions` exactly equals what the flows emit.
- Every `retry_class` matches the fixed table — asserted against your own
  hardcoded copy, not read from the manifest.
- Every `goto` and `status_map` target names a real step.

Do these at load time, not first-execution time. A shop should learn its
manifest is broken at startup, not during a customer's checkout.

→ [`Manifests/ManifestLoader.cs`](https://github.com/miki-smart/esiipayment-dotnet/blob/main/src/Esiipayment.Core/Manifests/ManifestLoader.cs),
  [`ManifestValidator.cs`](https://github.com/miki-smart/esiipayment-dotnet/blob/main/src/Esiipayment.Core/Manifests/ManifestValidator.cs)

### Stage 4 — The flow executor

**Goal:** actually run a flow. This is the heart.

The loop, in order — **the order is the specification**:

```
1. Work out the entry step (or, for sync/webhook, the triggered step)
2. Loop:
   a. If the step has a `call`, build the request and send it
   b. Check `errors[]` FIRST, before anything else
      - matched, and code is ProviderTimeout or Unknown
            -> Processing + Poll(5000). Stop.          <- Rule 2
      - matched, any other code
            -> Failed with that code. Stop.
      - either way: skip this step's `emit` entirely, leave state alone
   c. No error matched: apply `emit` (status, next_action, state)
   d. `emit.state` MERGES into existing state, it does not replace it
   e. Follow `status_map` or `goto` to the next step, or stop
```

Four details that are easy to get wrong and each cost real debugging
time:

**Errors before `status_map`, always.** And a match skips the step's
`emit` completely — no status, no state. A step whose call failed
produced nothing worth keeping.

**Timeout is hardcoded, not read from the manifest.** When the network
dies there's no response to read an interval from, so the spec fixes it:
`Processing` + `Poll` with `interval_ms: 5000`. Same in every runtime,
because it's part of a byte-compared output.

**State merges.** Three steps each writing one key means three keys at
the end, not one.

**The response stays available.** A step reached by `goto` — with no
`call` of its own — can still read `${extract...}` from the response the
*previous* step received. That's what lets `send_user_to_checkout` read
`checkout_url` in Part 1.

**Make the network a seam.** One interface, two implementations:

```csharp
public interface IProviderTransport
{
    Task<TransportOutcome> SendAsync(ResolvedRequest request, CancellationToken ct);
}
```

`HttpProviderTransport` makes real calls. `CassetteProviderTransport`
replays a recording and never touches the network. The executor can't
tell them apart — which is exactly why replay is trustworthy.

→ [`Flows/FlowExecutor.cs`](https://github.com/miki-smart/esiipayment-dotnet/blob/main/src/Esiipayment.Core/Flows/FlowExecutor.cs)

### Stage 5 — Prove it against mock

**This is the milestone.** Everything before was preparation.

Write one test that finds every cassette and checks it:

```csharp
public static IEnumerable<object[]> MockCassettes() =>
    Directory.EnumerateFiles(
        Path.Combine(SpecRepoRoot, "providers", "mock", "cassettes"), "*.yaml")
        .Select(path => new object[] { Path.GetFileNameWithoutExtension(path) });

[Theory]
[MemberData(nameof(MockCassettes))]
public void Cassette_replays_to_its_golden_file(string cassetteName)
{
    var providerDir = Path.Combine(SpecRepoRoot, "providers", "mock");
    var manifest = ManifestLoader.Load(
        File.ReadAllText(Path.Combine(providerDir, "manifest.yaml")));
    var cassette = CassetteLoader.Load(
        File.ReadAllText(Path.Combine(providerDir, "cassettes", $"{cassetteName}.yaml")));

    // The cassette's seed replaces the real clock and ID generator.
    var result = new FlowExecutor(manifest, new CassetteProviderTransport(cassette))
        .Execute(cassette);

    var expected = File.ReadAllText(
        Path.Combine(providerDir, "expected", $"{cassetteName}.json")).Trim();

    Assert.Equal(expected, CanonicalJson.Serialize(result));   // byte for byte
}
```

Discovery-based, so new cassettes become new test cases with no code
change. Run it:

```sh
dotnet test
```

```
Passed!  - Failed: 0, Passed: 202, Skipped: 0, Total: 202
```

Those 202 include all 20 mock cassettes plus every vector file.

**Work through the failures one at a time, in this order:**

1. **`collect.succeeded`** — the simplest path. If this fails, the
   problem is basic wiring.
2. **`collect.redirect`** — first one with a `next_action` payload.
3. **`collect.timeout`** — must give `Processing` + `Poll`. If you get
   `Failed`, your error handling has the most dangerous bug in the
   system.
4. **`collect.auth_error`, `collect.declined`** — error matching and
   `emit` skipping.
5. **The seven other action variants** — `otp`, `qr`, `transfer`, `ussd`,
   `device_push`, `capture`, `none`. Payload shapes.
6. **`sync.*`, `webhook.*`, `payout.*`, `cancel.*`, `refund.*`** — the
   other operations.

When all 20 pass, point the same test at every other provider directory.
They should pass with zero new code — that's the proof your runtime is
provider-agnostic, which is the entire claim.

### Stage 6 — The rest

Needed for a real runtime, but not for replay:

**Payment records and idempotency.** Save a record — key, a hash of the
request, status `Processing` — **before** the first network call. If the
process dies mid-call, the record exists and you can find out what
happened. Written after, a crash leaves no trace of a request that may
have charged someone.

Same key + same request = return the saved result, don't call again.
Same key + *different* request = reject it. That's a caller bug, not
something to silently accept.

**Credentials.** Never log them, never store them in plain text, never
let them into flow state (which gets persisted). For OAuth providers,
cache the token and refresh it early — and make the refresh
**single-flight**: when ten calls notice the token is stale, exactly one
fetches a new one and the others wait. Several real providers allow only
one active token, so ten concurrent refreshes means nine broken requests.
That bug never shows up in single-threaded tests.

**Webhooks.** Verify the signature over the **raw bytes**, before parsing
JSON. Re-serializing changes whitespace and key order, and the signature
won't match. Compare in constant time (`CryptographicOperations.FixedTimeEquals`
in .NET) — a normal `==` leaks how much of a forged signature was right.
A missing signature header is a rejection, not a pass.

→ [`Persistence/`](https://github.com/miki-smart/esiipayment-dotnet/tree/main/src/Esiipayment.Core/Persistence),
  [`Credentials/`](https://github.com/miki-smart/esiipayment-dotnet/tree/main/src/Esiipayment.Core/Credentials),
  [`Webhooks/`](https://github.com/miki-smart/esiipayment-dotnet/tree/main/src/Esiipayment.Core/Webhooks)

### Stage 7 — The public API

What shops actually call. The rule: **never expose a provider identifier
they're meant to branch on.**

```csharp
var result = await client.CollectAsync(orderId, intent);

switch (result.Status)
{
    case PaymentStatus.RequiresAction: /* show result.NextAction */ break;
    case PaymentStatus.Succeeded:      /* done */                   break;
    case PaymentStatus.Failed:         /* show result.Failure */     break;
    // ...
}
```

The shop picks a provider by name (they have to — it's their business
decision). But nothing in the *result* says which one it was. Every
outcome is expressed in the six statuses, nine actions, fourteen failure
codes, and three retry classes.

The test: could someone add a new provider without touching this shop's
code? If yes, you've kept the promise.

### What replay can't catch

Worth knowing before you call it done.

When we pointed the C# runtime at Chapa's **real** API — after all 20
mock cassettes were green — we found four genuine bugs immediately:

- **Two in the manifest.** Chapa returns HTTP 400 for validation errors,
  not the 200 the manifest assumed. And Chapa's 401 body *also* contains
  `"status":"failed"`, so an error entry matching that field caught the
  auth failure too — reporting a bad API key as a validation error.
- **Two in the runtime.** Our HTTP transport attached a JSON body to
  every request including GETs; Chapa answers a GET-with-a-body with an
  HTML error page. Which then hit the second bug: a non-JSON response
  crashed with a raw parser exception from deep inside the interpreter.

**None of these were reachable through cassette replay**, because a
cassette hands back a recorded response without ever building a real HTTP
request. Replay proves your interpreter is correct. It cannot prove your
HTTP layer is, or that the manifest matches reality.

Do both. Replay for correctness, one real sandbox call for reality.

### Checklist

Your runtime is conformant when:

- [ ] Every manifest in `providers/` loads without error
- [ ] Every file in `vectors/` passes
- [ ] Every cassette of every provider replays byte-identically
- [ ] A clock and ID source can be injected
- [ ] Timeouts produce `Processing` + `Poll`, never `Failed`
- [ ] Records are saved before the first network call
- [ ] Repeated keys replay; conflicting ones are rejected
- [ ] Webhook signatures verified on raw bytes, constant-time
- [ ] OAuth refresh is single-flight
- [ ] Unsupported `spec_version` is refused, not guessed at
- [ ] No provider identifier in the public result type

The full reference list is
[spec/07-runtime-requirements.md](../spec/07-runtime-requirements.md).

---

## Appendix: the complete Selam Pay files

Every file from Part 1, in full. These are the exact contents that
produced `no findings` and five `ok` lines.

<details>
<summary><strong>manifest.yaml</strong></summary>

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
provider: selampay-tutorial
spec_version: "2.0"
display_name: "Selam Pay (tutorial example)"
country: ET
currencies: [ETB]

environments:
  sandbox:
    base_url: https://sandbox.selampay.example/api
  production:
    base_url: https://api.selampay.example/api

auth:
  shape: api_key
  fields:
    - name: secret_key
      description: Secret key from the Selam Pay merchant dashboard, sent as a bearer token on every request.
  apply:
    header: Authorization
    value: "Bearer ${credentials.secret_key}"

capabilities:
  operations: [collect, sync]
  next_actions: [RedirectToUrl, Poll]
  currencies:
    ETB:
      min_amount: 100
      max_amount: 100000000

operations:
  collect:
    entry_flow: collect
  sync:
    entry_flow: sync

flows:
  collect:
    steps:
      start_payment:
        call:
          method: POST
          path: /v1/payments
          body:
            amount: ${amount_major(intent.amount)}
            currency: ${intent.currency}
            reference: ${idempotency_key}
            callback_url: ${ctx.webhook_url}
        triggers: [return]
        status_map:
          path: $.data.state
          values:
            pending: send_user_to_checkout
        emit:
          state:
            payment_id: ${extract.data.payment_id}

      send_user_to_checkout:
        emit:
          status: RequiresAction
          next_action:
            type: RedirectToUrl
            url: ${extract.data.checkout_url}

  sync:
    steps:
      check_payment:
        call:
          method: GET
          path: /v1/payments/${state.payment_id}
        triggers: [return]
        status_map:
          path: $.data.state
          values:
            paid: paid
            failed: payment_failed
            pending: still_waiting
        emit: {}

      paid:
        emit:
          status: Succeeded

      payment_failed:
        emit:
          status: Failed

      still_waiting:
        emit:
          status: Processing
          next_action:
            type: Poll
            interval_ms: 5000

errors:
  - match:
      path: $.data.reason
      equals: insufficient_funds
    failure_code: InsufficientFunds
    retry_class: DoNotRetry

  - match:
      http_status: 401
    failure_code: AuthFailed
    retry_class: DoNotRetry

  - match:
      transport: timeout
    failure_code: ProviderTimeout
    retry_class: ResolveFirst
```

</details>

<details>
<summary><strong>cassettes/collect.declined.yaml</strong></summary>

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/cassette.v1.schema.json
name: collect.declined
operation: collect
seed:
  clock: "2026-01-15T09:30:00Z"
  uuid: []
  idempotency_key: "SELAM-COLLECT-DECLINED-0001"
environment: sandbox
ctx:
  webhook_url: "https://myshop.example/webhooks/selampay"
credentials:
  secret_key: "sk_test_fixture-not-a-real-key"
intent:
  amount: 15000
  currency: ETB
interactions:
  - request:
      method: POST
      path: /v1/payments
      headers: { Authorization: "Bearer sk_test_fixture-not-a-real-key" }
      body: '{"amount":"150.00","currency":"ETB","reference":"SELAM-COLLECT-DECLINED-0001","callback_url":"https://myshop.example/webhooks/selampay"}'
    response:
      status: 200
      headers: { content-type: application/json }
      body: '{"data":{"state":"rejected","reason":"insufficient_funds"}}'
```

</details>

<details>
<summary><strong>cassettes/collect.auth_failed.yaml</strong></summary>

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/cassette.v1.schema.json
name: collect.auth_failed
operation: collect
seed:
  clock: "2026-01-15T09:30:00Z"
  uuid: []
  idempotency_key: "SELAM-COLLECT-AUTH-0001"
environment: sandbox
ctx:
  webhook_url: "https://myshop.example/webhooks/selampay"
credentials:
  secret_key: "sk_test_wrong-key"
intent:
  amount: 15000
  currency: ETB
interactions:
  - request:
      method: POST
      path: /v1/payments
      headers: { Authorization: "Bearer sk_test_wrong-key" }
      body: '{"amount":"150.00","currency":"ETB","reference":"SELAM-COLLECT-AUTH-0001","callback_url":"https://myshop.example/webhooks/selampay"}'
    response:
      status: 401
      headers: { content-type: application/json }
      body: '{"message":"Invalid API key"}'
```

Note the 401 body has no `data.reason` field — so it can't accidentally
match the `insufficient_funds` error entry too. Keeping error cases
distinguishable is part of writing them.

</details>

---

## Where to go next

| You are | Go to |
|---|---|
| Adding a real provider | [docs/tutorials/add-a-provider/](tutorials/add-a-provider/index.md) |
| Building a real runtime | [docs/tutorials/build-a-runtime/](tutorials/build-a-runtime/index.md) |
| Changing the format itself | [docs/tutorials/modify-the-spec/](tutorials/modify-the-spec/index.md) |
| Just using an SDK in a shop | [docs/use-a-provider.md](use-a-provider.md) |

The reference documents, when you need exact rules:
[domain model](../spec/01-domain-model.md) ·
[invariants](../spec/02-invariants.md) ·
[manifest format](../spec/03-manifest-dsl.md) ·
[expressions](../spec/04-expression-language.md) ·
[webhooks](../spec/05-webhooks.md) ·
[conformance](../spec/06-conformance.md) ·
[runtime requirements](../spec/07-runtime-requirements.md)
