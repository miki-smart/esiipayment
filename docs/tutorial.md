# Tutorial: your first adapter and your first SDK

This is a hands-on companion to [docs/add-a-provider.md](add-a-provider.md)
and [docs/build-a-runtime.md](build-a-runtime.md): those two are the
normative reference guides; this one is a worked example you can actually
follow start to finish (roughly 30-45 minutes for Part 1, an afternoon for
Part 2). By the end you'll have a small, real (if fictional) provider
manifest that validates and replays cleanly, and a minimal runtime that
proves, with a passing test, that it can execute that manifest and
produce byte-identical canonical JSON.

Part 2 uses C# as the illustrative language. Nothing about the approach
is C#-specific: swap in your own language's YAML and JSON libraries and
the same steps apply.

## Part 1: build an adapter ("Telebirr")

No programming language needed for this part. A text editor and the
`esiipayment` CLI (built once, from `tools/validator`; see that
directory's README) are all you need.

We'll build a manifest loosely modeled on Telebirr, Ethio Telecom's
mobile money service, because a familiar real-world name keeps the
exercise grounded. The endpoint shape, field names (`biz_content`,
`merch_order_id`, `trade_status`, `total_amount`, `trans_currency`), and
terminal status values (`PAY_SUCCESS`, `PAY_FAILED`) below are drawn from
Telebirr's real H5/Fabric payment-gateway vocabulary — reconstructed from
the developer portal's documented method names and two independently
maintained open-source client libraries, **not confirmed first-hand
against the provider**. The auth scheme (a plain `api_key` standing in
for Telebirr's real RSA key-pair + fabric-token + signed-request flow),
the intermediate `PAY_PENDING` status, and the single-step flow shape are
still simplified/invented for teaching purposes. This is still **not** a
verified description of Telebirr's actual API — compare
`providers/telebirr/metadata.yaml` for the real (also still
`provisional`) in-progress contribution this repo is building, and
`providers/santimpay/metadata.yaml` for what an honest "unverified" note
looks like on a real contribution in general.

Because a real `providers/telebirr/` is already in progress in this
repo, this tutorial deliberately builds its throwaway example under a
different directory — `providers/telebirr-tutorial` — so following along
never collides with or overwrites that real work. Delete
`providers/telebirr-tutorial/` when you're done; it's a learning
exercise, not a contribution (leave `providers/telebirr/` alone — that
one's real).

### Step 1: copy the template

```
cp -r providers/_template providers/telebirr-tutorial
```

### Step 2: identity, environment, auth

Replace the top of `providers/telebirr-tutorial/manifest.yaml`:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
provider: telebirr-tutorial
spec_version: "1.0"
display_name: "Telebirr (tutorial illustration, simplified/unverified)"
country: ET
currencies: [ETB]

environments:
  sandbox:
    base_url: https://sandbox.telebirr.test/api
  production:
    base_url: https://api.telebirr.test/api

auth:
  shape: api_key
  fields:
    - name: api_key
      description: API key issued in the Telebirr merchant dashboard; sent as a bearer token.
```

The real Telebirr Fabric gateway doesn't actually use a bare API key: it's
closer to a `key_pair` shape plus a token exchange (a `fabricAppId` +
`appSecret` call gets a short-lived bearer token, and every request past
that is RSA-PSS-signed with a merchant private key). We simplify to
`api_key` here so the exercise stays inside its 30-45 minute budget; see
`spec/03-manifest-dsl.md#auth`'s `key_pair` shape and the
`oauth2_client_credentials` `token:` block for the pieces you'd reach for
to model the real thing.

### Step 3: capabilities

Telebirr's `collect` is synchronous most of the time, but can report
"pending" for a payer still confirming on their end. That gives us a
realistic reason to need both a terminal outcome and `NextAction.Poll`:

```yaml
capabilities:
  operations: [collect]
  next_actions: [Poll]
  currencies:
    ETB:
      min_amount: 100
      max_amount: 100000000

operations:
  collect:
    entry_flow: collect
```

### Step 4: the flow

```yaml
flows:
  collect:
    steps:
      initialize:
        call:
          method: POST
          path: /payment/v1/merchant/preOrder
          headers:
            Authorization: "Bearer ${credentials.api_key}"
          body:
            biz_content:
              merch_order_id: ${idempotency_key}
              title: "Tutorial order"
              total_amount: ${amount_major(intent.amount)}
              trans_currency: ${intent.currency}
        triggers: [return]
        status_map:
          path: $.biz_content.trade_status
          values:
            PAY_SUCCESS: succeeded
            PAY_PENDING: proc_poll
        emit: {}

      succeeded:
        emit:
          status: Succeeded

      proc_poll:
        emit:
          status: Processing
          next_action: Poll
```

`/payment/v1/merchant/preOrder`, the `biz_content` wrapper, and
`merch_order_id`/`total_amount`/`trans_currency` are the real Telebirr
Fabric gateway's vocabulary for this call (its real request also carries
`timestamp`, `nonce_str`, `sign`, and `sign_type` fields we're omitting
here, since we're not implementing the signing step). `PAY_SUCCESS` is
likewise a real, confirmed `trade_status` value (its siblings are
`PAY_FAILED` and `PAY_CANCEL`); `PAY_PENDING` is **not** — public
documentation didn't surface an intermediate status literal for this
provider, so it's invented purely to give this exercise a reason to need
`NextAction.Poll`.

Notice `capabilities.next_actions: [Poll]` matches exactly the one
`next_action` this flow emits (`succeeded` emits none, since it's
terminal). `esiipayment validate` checks this cross-reference exactly,
not just "declared is a superset of emitted."

### Step 5: errors

Every manifest needs the mandatory transport-timeout entry, plus whatever
distinguishable failures the provider reports. We'll model a declined
payment and a bad API key:

```yaml
errors:
  - match:
      path: $.biz_content.trade_status
      equals: PAY_FAILED
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

Recall the precedence rule from
[spec/03-manifest-dsl.md](../spec/03-manifest-dsl.md#how-errors-interacts-with-status_map):
`errors` is checked *before* `status_map`, against the same response.
That's why `initialize`'s `status_map` above only needs to handle
`PAY_SUCCESS`/`PAY_PENDING`: `PAY_FAILED` and a bad key never reach it.

Real Telebirr distinguishes a third terminal status, `PAY_CANCEL` (the
payer backed out), which we're leaving out to keep this exercise the same
size as the original three-entry `errors` list — mapping it to its own
`FailureCode` (check `vectors/errors/failure-retry-map.json` for the
closest match) is a reasonable extension once you've got this passing.

No webhook section: this tutorial version of Telebirr doesn't have one,
so we leave it out entirely (only required when
`capabilities.operations` includes `"webhook"`).

### Step 6: metadata.yaml

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/metadata.v1.schema.json
provider: telebirr-tutorial
tier: community
description: >-
  Tutorial-only, simplified manifest used to teach the manifest DSL
  (docs/tutorial.md), loosely modeled on Telebirr. Not a verified
  description of Telebirr's real API; see providers/telebirr/ for this
  repo's actual (also unverified) Telebirr contribution.
verification:
  status: provisional
  docs_verified_on: null
  verified_by: null
  notes: >-
    Endpoint path, field names, and terminal status values are drawn from
    Telebirr's real Fabric gateway vocabulary (via public developer-portal
    references and third-party client libraries, not confirmed
    first-hand); the auth scheme, the pending status, and the single-step
    flow shape are simplified/invented for this exercise.
maintainers: []
links: {}
```

### Step 7: cassettes

Four minimum, per [docs/add-a-provider.md](add-a-provider.md): success, a
provider-reported failure, an auth failure, a transport timeout. Create
`providers/telebirr-tutorial/cassettes/collect.success.yaml`:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/cassette.v1.schema.json
name: collect.success
operation: collect
seed:
  clock: "2026-01-15T09:30:00Z"
  uuid: []
  idempotency_key: "TELEBIRR-COLLECT-SUCCESS-0001"
environment: sandbox
credentials:
  api_key: "fixture-not-a-real-key"
intent:
  amount: 5000
  currency: ETB
interactions:
  - request:
      method: POST
      path: /payment/v1/merchant/preOrder
      headers: { Authorization: "Bearer fixture-not-a-real-key" }
      body: '{"biz_content":{"merch_order_id":"TELEBIRR-COLLECT-SUCCESS-0001","title":"Tutorial order","total_amount":"50.00","trans_currency":"ETB"}}'
    response:
      status: 200
      headers: { content-type: application/json }
      body: '{"biz_content":{"trade_status":"PAY_SUCCESS"}}'
```

and `providers/telebirr-tutorial/expected/collect.success.json`
(canonical: sorted keys, no whitespace):

```json
{"failure":null,"idempotency_key":"TELEBIRR-COLLECT-SUCCESS-0001","next_action":null,"operation":"collect","state":{},"status":"Succeeded"}
```

Repeat the pattern for the other three, changing only what needs to
change:

- **`collect.declined.yaml`**: response body
  `{"biz_content":{"trade_status":"PAY_FAILED"}}`, expected
  `status: Failed`, `failure: {"failure_code":"InsufficientFunds","retry_class":"DoNotRetry"}`.
- **`collect.auth_failed.yaml`**: response `status: 401`, body
  `{"message":"Unauthorized"}` (deliberately *no* `"biz_content"` field,
  so it can't also match the `declined` entry; an ambiguous match is
  exactly what `esiipayment validate` flags), expected `Failed` /
  `AuthFailed` / `DoNotRetry`.
- **`collect.timeout.yaml`**: `transport_failure: true`, no `response`
  block at all, expected `Processing` / `Poll` / `failure: null`.

### Step 8: validate

```
cd tools/validator
go build -o esiipayment ./cmd/esiipayment
./esiipayment validate ../../providers/telebirr-tutorial
./esiipayment replay ../../providers/telebirr-tutorial --assert-golden
```

Both must pass before you'd consider this ready for review in a real
adapter PR (it won't be, since it's fictional, but this is the exact loop
you'll run for a real provider).

> The Go validator hasn't been build-tested in every environment this
> spec was authored in; if `go build` surfaces something, that's useful
> signal, so file it as a runtime-conformance issue.

### What you just learned

- The manifest DSL is genuinely just data: nothing here required a
  programming language.
- `errors` short-circuits `status_map`, and a matched error means that
  step's `emit` never runs.
- `capabilities` fields (`next_actions`, `operations`) are exact-match
  contracts against what a manifest's flows actually do, not aspirational
  declarations.
- Every cassette needs a byte-exact `expected/` golden file: this is the
  mechanism that keeps five language runtimes honest against each other.
- Grounding an invented example in a real provider's actual field names
  and status vocabulary (not just its name) makes the exercise more
  useful without requiring you to implement its full complexity (we still
  skipped Telebirr's real signing step entirely).

When you're done, `rm -rf providers/telebirr-tutorial`: it was a sandbox.
Leave `providers/telebirr/` — the real, still-in-progress contribution —
alone.

## Part 2: build a minimal SDK, in C#

This part builds *just enough* of a runtime to load one manifest, replay
one cassette, and prove byte-identical canonical output, not the full
conformance suite [docs/build-a-runtime.md](build-a-runtime.md) describes.
Treat this as the first rung of that ladder, not the whole ladder.

### Step 1: scaffold

```
dotnet new console -n Tutorial.Runtime -o Tutorial.Runtime
cd Tutorial.Runtime
dotnet add package YamlDotNet
```

### Step 2: confirm strict YAML decoding, empirically

Don't take anyone's word (including this tutorial's) for how a YAML
library handles an unrecognized key: this is exactly the mechanism that
enforces `additionalProperties: false`, so verify it yourself first:

```csharp
using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;

var deserializer = new DeserializerBuilder()
    .WithNamingConvention(UnderscoredNamingConvention.Instance)
    .Build();

class Simple { public string Name { get; set; } = ""; }

var yaml = "name: test\nunknown_field: oops\n";
try
{
    deserializer.Deserialize<Simple>(yaml);
    Console.WriteLine("Unknown keys are silently ignored; you'll need extra work for strictness.");
}
catch (YamlDotNet.Core.YamlException ex)
{
    Console.WriteLine("Unknown keys throw by default: " + ex.Message);
}
```

Confirmed while writing this tutorial (YamlDotNet 18.1.0, default
`DeserializerBuilder().Build()`, no `.IgnoreUnmatchedProperties()`
called): **it throws.** That's exactly the strictness you want: as long
as you never call `.IgnoreUnmatchedProperties()`, an unrecognized manifest
key becomes a hard failure automatically.

### Step 3: a minimal model

Just enough to run one `collect` flow with one `call` -> `status_map` ->
terminal `emit` chain (skip `errors`, `webhook`, transforms, multi-step
chains for now; those are exactly what you'd add next, following
`spec/03-manifest-dsl.md` section by section):

```csharp
class ManifestDoc
{
    public string Provider { get; set; } = "";
    public Dictionary<string, FlowDoc> Flows { get; set; } = new();
}
class FlowDoc { public Dictionary<string, StepDoc> Steps { get; set; } = new(); }
class StepDoc { public StatusMapDoc? StatusMap { get; set; } public EmitDoc? Emit { get; set; } }
class StatusMapDoc { public string Path { get; set; } = ""; public Dictionary<string, string> Values { get; set; } = new(); }
class EmitDoc { public string? Status { get; set; } public string? NextAction { get; set; } }

class CassetteDoc
{
    public string Name { get; set; } = "";
    public string Operation { get; set; } = "";
    public SeedDoc Seed { get; set; } = new();
    public InteractionDoc Interaction { get; set; } = new();
}
class SeedDoc { public string IdempotencyKey { get; set; } = ""; }
class InteractionDoc { public ResponseDoc Response { get; set; } = new(); }
class ResponseDoc { public int Status { get; set; } public string Body { get; set; } = ""; }
```

(This is a simplified single-interaction cassette shape for the tutorial;
the real schema in `schema/cassette.v1.schema.json` allows an
`interactions` list, `intent`, `environment`, `credentials`, and more; add
those as you graduate to real cassettes.)

### Step 4: canonical JSON, correctly

**The one place C# will surprise you if you assume Go-like behaviour**:
`System.Text.Json` does not sort dictionary/object keys the way Go's
`encoding/json` does automatically. You must sort keys yourself, at every
nesting level, and you should disable the default HTML-safe escaping
(`JavaScriptEncoder.UnsafeRelaxedJsonEscaping`) or you'll emit
`&`-style escapes canonical JSON doesn't want:

```csharp
using System.Text.Json;
using System.Text.Encodings.Web;

record PaymentResult(string IdempotencyKey, string Operation, string Status, string? NextAction);

static class Canonical
{
    public static string ToJson(PaymentResult r)
    {
        var opts = new JsonSerializerOptions
        {
            Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
            WriteIndented = false,
        };
        // Insert keys in already-sorted order: .NET preserves insertion
        // order for Dictionary<string,TValue>, it does not re-sort them.
        var obj = new Dictionary<string, object?>
        {
            ["failure"] = null,
            ["idempotency_key"] = r.IdempotencyKey,
            ["next_action"] = r.NextAction,
            ["operation"] = r.Operation,
            ["status"] = r.Status,
        };
        return JsonSerializer.Serialize(obj, opts);
    }
}
```

Once you add `state` (a nested object) to `PaymentResult`, apply the same
discipline recursively: build it as a `Dictionary<string, object?>` with
keys inserted in sorted order, not a plain C# object serialized via
reflection (whose property order follows declaration order, not
alphabetical order).

### Step 5: the interpreter

```csharp
static class Interpreter
{
    public static PaymentResult Run(ManifestDoc manifest, CassetteDoc cassette)
    {
        var flow = manifest.Flows["collect"];
        var entry = flow.Steps["initialize"];

        using var doc = JsonDocument.Parse(cassette.Interaction.Response.Body);
        var field = entry.StatusMap!.Path.Replace("$.", "");
        var extracted = doc.RootElement.GetProperty(field).GetString()!;

        var targetStepName = entry.StatusMap.Values[extracted];
        var targetStep = flow.Steps[targetStepName];

        return new PaymentResult(
            cassette.Seed.IdempotencyKey,
            cassette.Operation,
            targetStep.Emit!.Status!,
            targetStep.Emit.NextAction
        );
    }
}
```

This is deliberately the simplest possible slice: no `errors[]`
precedence, no entry-step-by-elimination (it just hardcodes
`"initialize"`), no `emit.state` patching, no transport-timeout handling.
Each of those is a real rule from `spec/03-manifest-dsl.md` you'll add as
you extend this; see the "where to go from here" list at the end.

### Step 6: prove it against one cassette

```csharp
var manifestYaml = """
provider: telebirr
flows:
  collect:
    steps:
      initialize:
        status_map:
          path: $.status
          values:
            success: succeeded
      succeeded:
        emit:
          status: Succeeded
""";

var cassetteYaml = """
name: collect.success
operation: collect
seed:
  idempotency_key: "TELEBIRR-COLLECT-SUCCESS-0001"
interaction:
  response:
    status: 200
    body: '{"status":"success"}'
""";

var manifest = deserializer.Deserialize<ManifestDoc>(manifestYaml);
var cassette = deserializer.Deserialize<CassetteDoc>(cassetteYaml);
var result = Interpreter.Run(manifest, cassette);

Console.WriteLine(Canonical.ToJson(result));
// {"failure":null,"idempotency_key":"TELEBIRR-COLLECT-SUCCESS-0001","next_action":null,"operation":"collect","status":"Succeeded"}
```

This exact code was written and run while preparing this tutorial;
`dotnet run` prints that line character-for-character. Once you have this
passing, wire it into an actual xUnit test project (`dotnet new xunit`)
and assert the string equality there instead of eyeballing console
output.

### Where to go from here

This minimal interpreter handles one cassette's happy path. To reach real
conformance against `providers/mock/` (20 cassettes, every
`PaymentStatus`/`NextAction` combination) you still need, in roughly this
order:

1. **The entry-step rule.** A flow's first step is never named explicitly:
   it's whichever step no other step's `goto`/`status_map.values` ever
   targets. Compute it by elimination instead of hardcoding `"initialize"`.
2. **`errors[]` precedence.** Check `errors` against the response *before*
   `status_map`; a match bypasses that step's `emit` entirely (state stays
   as it was).
3. **Transport failures.** No response at all resolves to `Processing` +
   `Poll`, always, per Invariant I4: this is fixed behaviour, not
   something you read out of the manifest at replay time.
4. **`emit.state` as a patch**, not a replacement: later steps add to
   what earlier steps in the same run already wrote.
5. **The full expression language** (`spec/04-expression-language.md`):
   the JSONPath subset with array/wildcard support, `${transform(...)}`
   application (`amount_major` needs `vectors/money/exponents.json` for
   the currency's exponent; don't hardcode 2), and namespace resolution
   for `intent`/`ctx`/`credentials`/`state`.
6. **Every vector in `vectors/`** as literal test cases: they're the
   executable spec, not illustrative examples.

At that point you're building the real thing: go to
[docs/build-a-runtime.md](build-a-runtime.md) and work from
`spec/07-runtime-requirements.md` as your checklist.
