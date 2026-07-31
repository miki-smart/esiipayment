# 00: Overview

## What ESIIPayment is

ESIIPayment is a language-neutral contract for accepting payments from Ethiopian
payment service providers (Chapa, ArifPay, SantimPay, telecom wallets, bank
gateways, and others) through one uniform interface. An integrator writes
checkout code once, against six closed enums and a handful of operations.
Adding a new provider (or a new programming language) never touches that
code.

This repository, `esiipayment-spec`, is the contract itself: the domain model,
the manifest DSL that describes a provider's behaviour as data, the
expression language manifests are allowed to use, the schemas that validate
both, and the vectors and cassettes that every language runtime must
reproduce byte-for-byte. It contains no application code. Language SDKs
(`esiipayment-dotnet`, `esiipayment-python`, `esiipayment-node`, `esiipayment-go`,
`esiipayment-php`, and others) are separate repositories that pin this one as a
git submodule and implement a runtime against it.

## The promise

> Integrator code never changes when a provider is added.

An integrator calls `collect(amount, currency, method)` against whichever
provider a merchant has configured. The response is always one of six
`PaymentStatus` values and one of nine `NextAction` variants, never a
provider-specific shape, never a provider-specific error string requiring a
`switch` on provider name. See [Invariant I12](02-invariants.md#i12).

## Two contributor roles

ESIIPayment deliberately separates two kinds of work that have nothing to do
with each other:

- **Adapter authors** describe a provider's HTTP behaviour as a `manifest.yaml`
  plus recorded HTTP cassettes. They need no programming language; many will
  have integrated a provider at their day job and can contribute that
  knowledge in an evening. Start at
  [docs/add-a-provider.md](../docs/add-a-provider.md).
- **Runtime implementers** write an interpreter, in one language, for the
  manifest format: load a manifest, validate it, execute its flows, verify
  its output against the golden cassettes. They need no knowledge of any
  specific provider. Start at
  [docs/build-a-runtime.md](../docs/build-a-runtime.md).

Keeping these disjoint is the reason the project scales. If providers were
implemented per-language, N providers across M languages would require N×M
adapters, and a single provider API change would need M separate PRs from M
different people before behaviour reconverged across languages. With
manifests, it's N adapters plus M runtimes, and a manifest change is
correct everywhere the moment every runtime's golden tests pass against it.

## How the pieces fit together

```
                     ┌─────────────────────┐
                     │   esiipayment-spec      │   ← this repo: the contract
                     │  schema / spec /    │
                     │  vectors / cassettes│
                     └──────────┬──────────┘
                                │ git submodule
              ┌─────────────────┼─────────────────┐
              │                 │                 │
      ┌───────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐
      │ esiipayment-dotnet│  │ esiipayment-python│  │  esiipayment-go  │  ...
      │  (runtime)    │  │  (runtime)    │  │  (runtime)   │
      └───────────────┘  └───────────────┘  └──────────────┘
```

A runtime loads `providers/<name>/manifest.yaml`, validates it against
`schema/manifest.v1.schema.json`, and executes its `flows` using the
expression language defined in [04](04-expression-language.md). Given the
same manifest, the same cassette, and the same frozen clock/UUID seed, every
runtime must emit byte-identical canonical JSON (defined in
[06-conformance.md](06-conformance.md)). That determinism is what makes
"every SDK behaves identically" a testable claim rather than a slogan.

## Reading order

| Doc | Purpose |
|---|---|
| [01-domain-model.md](01-domain-model.md) | The closed enums and value types every runtime shares |
| [02-invariants.md](02-invariants.md) | The twelve rules that must hold regardless of provider or language |
| [03-manifest-dsl.md](03-manifest-dsl.md) | How a provider's behaviour is described as data |
| [04-expression-language.md](04-expression-language.md) | The minimal extraction/interpolation/condition language manifests may use |
| [05-webhooks.md](05-webhooks.md) | Inbound webhook verification and normalisation |
| [06-conformance.md](06-conformance.md) | Canonical JSON, golden output, and what "conformant" means |
| [07-runtime-requirements.md](07-runtime-requirements.md) | What every language SDK must implement and guarantee |
| [08-versioning.md](08-versioning.md) | SemVer rules for the spec, schema, and manifests |

## Non-goals

- This repo does not implement a payment gateway, a checkout UI, or a
  hosted service. It defines a contract that others implement against.
- This repo does not certify that any specific provider integration is
  production-correct. See the verification-status requirement in
  [docs/add-a-provider.md](../docs/add-a-provider.md): provider manifests
  are marked `provisional` until a maintainer has verified them against
  current, licensed-to-read provider documentation.
