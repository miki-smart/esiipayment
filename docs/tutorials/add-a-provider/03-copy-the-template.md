# 3. Copy the template, walk every section

```text
cp -r providers/_template providers/<your-provider-slug>
```

[`providers/_template/manifest.yaml`](../../../providers/_template/manifest.yaml)
is a fully commented skeleton with a `TODO` at every fill-in point.
Work through it top to bottom. Every example below is drawn from
[`providers/chapa/manifest.yaml`](../../../providers/chapa/manifest.yaml) —
a real (if provisional) manifest already in this repository, built around
the same [canonical scenario](../../examples/scenario.md) this tutorial
follows — so you're looking at a finished section immediately after
reading what it's for, not just a description of one.

## `auth`

Declares exactly one `CredentialShape`
([01-domain-model.md#credentialshape](../../../spec/01-domain-model.md#credentialshape))
and where the resolved credential goes on every request
([03-manifest-dsl.md#auth](../../../spec/03-manifest-dsl.md#auth)):

```yaml
--8<-- "providers/chapa/manifest.yaml:auth-section"
```

Pick the closest `CredentialShape` to what the provider actually issues
— `api_key` (a single bearer/API key), `key_pair`, `triple_key`,
`oauth2_client_credentials`, `hmac`, or `mutual_tls` — and never invent a
new one, even if the provider's real scheme is a minor variant. `apply`
says where the resolved value goes (a header here; some providers use a
query parameter or a body field instead) so no individual step in
`flows` below has to repeat it.

## `capabilities`

Declares **limits, not just booleans** — a boolean says a method exists;
a limit says whether an integrator's actual use case works:

```yaml
--8<-- "providers/chapa/manifest.yaml:capabilities-section"
```

`next_actions` must equal *exactly* the set of `NextAction` variants your
`flows` (below) actually emit — not a superset "just in case." Chapter 4
is entirely about choosing these correctly.

## `operations`

Maps each operation you support to the `flows` entry that implements it:

```yaml
--8<-- "providers/chapa/manifest.yaml:operations-section"
```

Every key here needs both a matching `flows` entry and cassette coverage
(chapter 7) — you can't declare an operation and skip either.

## `flows`

The actual state machine: what request to send, and what to do with the
response. This is the biggest section and the one most specific to each
provider's own API shape; chapter 4 covers the `NextAction` choice inside
it in detail, and chapter 5 covers `errors`' interaction with it. Chapa's
`collect` flow, hosted-checkout-shaped (initialize returns a checkout
URL; the actual outcome arrives later via `sync` or a webhook):

```yaml
--8<-- "providers/chapa/manifest.yaml:flows-collect-section"
```

A step either `call`s an HTTP endpoint (with `triggers: [return]`) or is
reached via a webhook (`triggers: [webhook]`); it then either `emit`s a
final `status`/`next_action`/`state` patch, or uses `status_map` to
branch to a different next step based on a value extracted from the
response. `${amount_major(intent.amount)}`, `${idempotency_key}`, and
`${ctx.webhook_url}` are all *interpolations* —
[04-expression-language.md](../../../spec/04-expression-language.md) is
the reference for the full `${...}` syntax, but you can get remarkably
far just from reading real examples like this one and the template's own
comments.

## `errors`

Maps this provider's own failure signals to the closed `FailureCode` set:

```yaml
--8<-- "providers/chapa/manifest.yaml:errors-section"
```

Chapter 5 is entirely about getting this section right, in particular
which failures are `ResolveFirst` rather than `DoNotRetry`. The
`transport: timeout` entry at the bottom is **mandatory, verbatim**, in
every manifest: `esiipayment validate` rejects a manifest missing it, or
one that maps it to anything other than `ProviderTimeout`/`ResolveFirst`.

## `webhook`

Required only if `capabilities.operations` includes `webhook`; declares
how to verify an inbound callback
([05-webhooks.md](../../../spec/05-webhooks.md)):

```yaml
--8<-- "providers/chapa/manifest.yaml:webhook-section"
```

Verification always runs over the **raw** body with a **constant-time**
comparison — never a re-serialized form of it, and never a naive
string-equality loop. If you genuinely don't know this provider's
signature scheme (you're not alone; see the `UNVERIFIED` comment in
Chapa's own manifest above), set `scheme: none` and say so plainly in
`metadata.yaml` rather than guessing a plausible-looking header name and
presenting it as fact. `scheme: none` is a real gap, not a mechanism to
lean on for a provider that does sign its callbacks — see
[03-manifest-dsl.md#native-providers](../../../spec/03-manifest-dsl.md#native-providers)
if a provider's actual verification scheme needs something this DSL
can't express (a non-HMAC signing scheme, for instance) rather than
declaring `none` to sidestep it.

## Keep the schema header

The first line stays exactly as chapter 2 described:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
```

Next: [choosing the right `NextAction`](04-choosing-next-action.md) —
the section most first-time adapter authors spend the most time getting
right, and the one this project cares most about you getting right.
