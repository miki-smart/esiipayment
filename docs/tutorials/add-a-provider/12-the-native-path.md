# Step 12 — The native path

You are here because [Step 0](00-which-kind.md) ended in "native": your
provider needs something the manifest DSL structurally cannot express,
and you can name which of the five signals applies. This page is the
whole difference between the two tracks.

Read [spec/03-manifest-dsl.md#native-providers](../../../spec/03-manifest-dsl.md)
alongside this — that section is normative, this is the walkthrough.

## What changes, and what doesn't

Most of this tutorial still applies to you unchanged:

| Step | Applies to a native provider? |
|---|---|
| [2. Editor setup](02-editor-setup.md) | Yes — point the schema line at `capabilities.v1.schema.json` instead |
| [3. Copy the template](03-copy-the-template.md) | Replaced by "copy `_template-native/`" below |
| [4. Choosing `NextAction`](04-choosing-next-action.md) | Yes, entirely — your code emits these |
| [5. Mapping errors](05-mapping-errors.md) | Yes, entirely — your code decides `failure_code`/`retry_class`, and `ResolveFirst` still means what it means |
| [6. Validating with Docker](06-validating-with-docker.md) | Yes, with one difference (below) |
| [7. Recording cassettes](07-recording-cassettes.md) | Yes, same schema, same requirement |
| [8. Golden files](08-golden-files.md) | Yes — but *you* assert them, see below |
| [9. Clean room and DCO](09-clean-room-and-dco.md) | Yes, and it matters more: you are writing code |
| [10. Opening the PR](10-opening-the-pr.md) | Yes, plus the extra review bar below |
| [11. Verification and tiers](11-verification-and-tiers.md) | Yes, unchanged |

What changes is exactly three things: the file you write here, who
verifies your cassettes, and the fact that you also have code to write
somewhere else.

## 1. Write `capabilities.yaml`, not `manifest.yaml`

Copy [providers/_template-native/](../../../providers/_template-native/)
to `providers/<your-provider>/` and fill in every TODO. The template
walks each field.

`capabilities.yaml` carries only **what your provider can do**: identity,
country, currencies, the credential *shape* and field names, the
operations and `NextAction` variants you support, and per-currency
limits. Plus one literal line, `implementation: native`, which is what
routes this whole repository — validate, replay, lint, the catalog, and
every runtime — down the native path.

It carries no `environments`, `flows`, `errors` or `webhook` section.
Those describe **how a runtime talks to your provider**, and for a native
provider that is code, not data. Base URLs included: they live in your
implementation's configuration, not here. Adding any of those keys is a
validation error, not a shortcut.

One consequence worth internalising: `capabilities.next_actions` is a
promise this repository cannot check. For a manifest, `esiipayment
validate` cross-checks the declared variants against the flows that emit
them. There are no flows here, so nothing catches a lie except your own
conformance suite.

## 2. Your code goes in the runtime's repository, never here

This repository contains no application code beyond `tools/validator`,
and that rule is not relaxed for native providers.
[.github/workflows/no-code.yml](../../../.github/workflows/no-code.yml)
enforces it against your provider directory exactly as it does against
every other. `providers/<name>/` holds `capabilities.yaml`,
`metadata.yaml`, `cassettes/` and `expected/` — nothing else.

Your implementation lives in each language SDK's own repository. For .NET
that is `esiipayment-dotnet`, where `Esiipayment.Providers.Telebirr` is
the worked example: a native client with its own signing, token
exchange and response mapping, plus its own tests.

That means a native provider PR is normally **two** PRs: one here (data),
one in a runtime repository (code). Open the data one first — if the
native path itself gets pushed back, you will not have written the code
for nothing.

## 3. Conformance becomes your job

For a manifest provider, this repository's CI proves your cassettes
replay to your golden files. For a native provider it cannot: there is no
manifest describing your provider's logic for `esiipayment replay` to
interpret. So:

```console
$ esiipayment validate providers/<your-provider>
no findings

$ esiipayment replay providers/<your-provider> --assert-golden
providers/<your-provider> is a native-implementation provider ...
```

`validate` checks everything that is still this repository's business:
`capabilities.yaml`'s shape, closed enums, `metadata.yaml`, cassette
coverage for every declared operation, and a well-formed
`expected/<name>.json` for every cassette. `replay` prints an
informational message and exits 0 — deliberately, so CI needs no
special-casing.

The verification it can no longer do is yours to supply, at the same bar:
**the same cassettes, the same `expected/*.json`, the same
[spec/06-conformance.md](../../../spec/06-conformance.md) byte-identity
requirement, asserted by your language's own test runner.** Not
"equivalent" output. Byte-identical canonical JSON.

In .NET that is a single theory over the provider's cassette directory;
see `TelebirrConformanceTests` in `esiipayment-dotnet`. If your runtime
has no such harness yet, write it before writing the provider — a native
provider whose goldens are never asserted is indistinguishable from one
that is quietly wrong.

### About `request.body` in your cassettes

Record it. A cassette that pins the exact wire body asserts request
construction, not merely response mapping, and that is most of its value.

Omit it only when the body genuinely cannot be reproduced from fixtures
committed here — in practice, when it carries a signature over a private
key that must not be published in a public repository. Telebirr's
cassettes omit bodies for exactly that reason and
[say so explicitly](../../../providers/telebirr/metadata.yaml). If you
omit them, state why in `metadata.yaml` and cover request construction
in your own unit tests instead. An unasserted request body is a real
gap; disclose it rather than letting green CI imply coverage you do not
have.

## 4. The obligation your implementation carries

Integrator code must not be able to tell your provider is native. That is
[Invariant I12](../../../spec/02-invariants.md): every outcome is
expressed in `PaymentStatus` / `NextAction` / `FailureCode` /
`RetryClass`, and an integrator switching on those must never need a
provider-specific branch. Concretely, in whichever runtime you implement:

- Your provider must be reachable through the **same public surface** as
  a manifest-driven one — the same client interface, the same method
  names, the same result type. If an application has to hold your
  provider in a differently-typed variable, the native path has leaked
  into integrator code and I12 is broken.
- The invariants are not optional for you. A transport timeout must not
  surface as `Failed` (I4). The idempotency-key record must be persisted
  before your first network call (I8). A duplicate key with a different
  payload must be `DuplicateRequest` (I7). A runtime that gives you a
  native-provider base class has usually done this part for you —
  in .NET, `NativePaymentClient` carries all three, so inherit it rather
  than reimplementing them.
- Your credentials come from the `auth.fields` you declared, and secrets
  never appear in this repository (I9).

## 5. The extra review bar

A native-provider PR is reviewed for one thing a manifest PR is not:
**is the native path justified?** State in `metadata.yaml`, specifically
enough to be checked, which of the five signals applies and why.

- Checkable: "Every request past the token exchange must carry an
  RSASSA-PSS signature over biz_content flattened one level up and
  key-sorted into `k=v` pairs — canonicalisation before signing, signal 2."
- Not checkable, and the most common reason a native PR is sent back:
  "The DSL is not flexible enough for this provider."

If a reviewer can see a manifest that would have worked, you will be
asked to write it instead. That is the process functioning correctly:
they are saving every other runtime the cost of your provider.
