# ESIIPayment

**A language-neutral contract for accepting payments from Ethiopian
payment service providers through one uniform interface.**

An integrator writes checkout code once, against six closed statuses,
nine possible next-user-actions, and a handful of operations
(`collect`, `sync`, `payout`, `cancel`, `refund`, `webhook`). Adding a new
provider (or a new programming language) never touches that code:

> **Integrator code never changes when a provider is added.**

This repository is the contract itself: the domain model, the YAML
manifest format that describes a provider's behaviour as data (no code,
in any language), the schemas that validate it, and the test
vectors/cassettes every language runtime must reproduce byte-for-byte.
Language SDKs (`esiipayment-dotnet`, `esiipayment-python`, `esiipayment-node`,
`esiipayment-go`, `esiipayment-php`) are separate repositories that pin this one
as a submodule.

## Two roles, and where each starts

This project only works if these stay separate: see
[spec/00-overview.md](spec/00-overview.md) for why.

| Role | Needs | Start here |
|---|---|---|
| **Adapter author**: describes one provider's behaviour | No programming language. YAML and a text editor. | [docs/add-a-provider.md](docs/add-a-provider.md) |
| **Runtime implementer**: builds a language SDK against this spec | One programming language, zero provider-specific knowledge. | [docs/build-a-runtime.md](docs/build-a-runtime.md) |

Using an SDK to actually accept payments (not contributing here)? See
[docs/use-a-provider.md](docs/use-a-provider.md).

New to both roles and want a worked example before diving into the
reference guides above? [docs/tutorial.md](docs/tutorial.md) walks
through building one small (fictional) adapter and one minimal SDK,
start to finish.

## Repository layout

```
spec/          The normative contract: domain model, invariants, manifest
               DSL, expression language, webhooks, conformance, runtime
               requirements, versioning. Start at spec/00-overview.md.
schema/        JSON Schemas for manifests, metadata, and cassettes.
vectors/       Executable test data every runtime must match exactly:
               money, expressions, status transitions, error/retry
               mapping, webhook signatures, canonical JSON.
providers/     One directory per provider. _template/ is a commented
               skeleton; mock/ is a complete, deterministic reference
               provider with no credentials or network dependency;
               chapa/, arifpay/, santimpay/ are real (provisional;
               see below) provider manifests.
tools/validator/  The one piece of application code this repo permits:
               a Go CLI (esiipayment validate/replay/lint/catalog),
               distributed as a container image so no contributor needs
               a Go toolchain.
docs/          Practical guides for each role (see table above).
.github/       Issue/PR templates per role, CI workflows, CODEOWNERS.
```

## The one governing rule

**No application code, in any programming language, under `providers/`,
`vectors/`, or `schema/`.** A provider is YAML data describing HTTP
behaviour, never a script. This is what lets ten providers work correctly
across five languages without fifty separately-maintained
implementations: see [spec/00-overview.md](spec/00-overview.md) for the
full reasoning. `.github/workflows/no-code.yml` enforces this on every
PR.

## Verification status

**Chapa, ArifPay, and SantimPay's manifests in this repository are
provisional.** They were written from general recollection of each
provider's public API, not from a current, licensed read of their
documentation: endpoint paths, field names, and (especially) webhook
signature schemes are marked inline as unverified, with the specifics of
what's unverified recorded in each provider's `metadata.yaml`. Do not
build production integrations on a provisional manifest without
confirming its specifics yourself; see
[docs/add-a-provider.md](docs/add-a-provider.md#moving-from-provisional-to-verified)
for how a manifest moves from provisional to verified. `providers/mock/`
has no such caveat; it's fully specified by construction, not by
reference to an external API.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the DCO sign-off requirement,
the clean-room norm for writing provider manifests, and where each
contributor role starts. See [GOVERNANCE.md](GOVERNANCE.md) for the RFC
process and adapter tiers.

## License

[Apache 2.0](LICENSE).
