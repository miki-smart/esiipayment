# Step 0 — Which kind of provider is yours?

There are two ways a provider gets onboarded, and this page is the whole
decision. It takes about five minutes and saves everyone a rejected PR.

- A **manifest provider** is a `manifest.yaml`: plain data describing what
  your provider does. Every runtime already knows how to execute it, so
  one file supports .NET, Python, JavaScript and every future runtime at
  once. No programming, no code review in any language.
  [providers/chapa/manifest.yaml](../../../providers/chapa/manifest.yaml)
  is the worked example.
- A **native provider** is a `capabilities.yaml` plus hand-written code in
  each language runtime that supports it. It exists for providers the DSL
  structurally cannot describe.
  [providers/telebirr/capabilities.yaml](../../../providers/telebirr/capabilities.yaml)
  is the worked example.

Both are first-class. Neither is a second-class citizen or a "temporary"
state. But they are not equal in cost, and the cost falls on different
people.

## Start from the manifest, always

Write a manifest unless you have a specific reason you can name. Not
because the native path is discouraged, but because of who pays for each:

| | Manifest provider | Native provider |
|---|---|---|
| What you write | One YAML file | `capabilities.yaml` + code, per language |
| Programming needed | None | Yes, in each runtime's language |
| Runtimes that get your provider | All of them, immediately | Only the ones someone implements it in |
| Who verifies your cassettes replay correctly | `esiipayment replay`, in this repo's CI | Each runtime's own conformance suite |
| Cost of a future spec change | Absorbed by runtimes | Partly yours, in every runtime |
| Review bar | Does the manifest match the provider's API? | That, plus: is the native path actually justified? |

The asymmetry in row 3 is the one that matters most. A manifest for a
provider is a contribution to everyone; a native provider is a
contribution to whichever runtimes get around to it. Today that is .NET.
If you write a native provider and nobody implements it in Python, then
as far as a Python shop is concerned your provider does not exist.

So: an awkward manifest beats an elegant native implementation. If your
manifest needs an extra step, three more `errors` entries, or a
status-mapping arrangement that feels inelegant — ship it. Inelegance in
a manifest costs nothing to any runtime. Code costs every runtime,
forever.

## The five signals that mean native

You need the native path when your provider needs at least one of these.
This is the same list as
[spec/03-manifest-dsl.md#the-five-signals](../../../spec/03-manifest-dsl.md),
which is normative; this page just makes it concrete.

**1. Branching on a parsed intermediate value.** `status_map` and `errors`
compare exactly one extracted path against one literal. If deciding your
next step genuinely requires combining two fields, testing a numeric
range, or computing something — not just reading one value — you cannot
express it.

*Not this signal:* "I need to check two different fields in two different
error cases." That is two `errors` entries, which is fine.

**2. Canonicalisation before signing.** Your provider requires a
signature over something other than the raw body: sorted parameters, a
provider-specific concatenation, a re-serialized form. This is Telebirr's
case — it signs RSASSA-PSS over its `biz_content` flattened one level up,
non-string values dropped, keys sorted, joined as `k=v` pairs.

*Not this signal:* an HMAC over the raw request or webhook body. The DSL
handles that; see [spec/05-webhooks.md](../../../spec/05-webhooks.md).

**2b. A `next_action` your runtime has to compute.** A close relative
worth naming separately, because it is easy to miss until late: the DSL
emits a `NextAction` by interpolating values out of a response. If the
value the payer needs does not exist in any response — Telebirr's
checkout URL has to be *assembled* from a `prepay_id` and then signed
with a second signature — no amount of manifest is going to produce it.

**3. Session or cookie state.** Your provider's API requires carrying an
opaque cookie jar or session object between calls. The DSL's `state` is a
flat object whose every key you name, which cannot hold an opaque blob
you never see the inside of.

*Not this signal:* a provider-assigned session *id* you store and send
back later. That is exactly what `state` is for; see
[providers/arifpay/manifest.yaml](../../../providers/arifpay/manifest.yaml).

**4. A non-HTTP channel.** SOAP, gRPC, a vendor SDK you must link
against, a USSD gateway's own protocol. Every `call` in the DSL is one
HTTP request; if there is no HTTP request to describe, there is nothing
for `call` to say.

*Not this signal:* an unusual content type, or form-encoded bodies over
HTTP.

**5. Status vocabularies needing more than equality.** Classifying one
outcome requires a regex, a numeric threshold, or reading more than one
field at once.

*Not this signal:* a provider with fifteen distinct status strings. That
is fifteen `status_map` entries — verbose, and completely fine.

## "The DSL can't express my provider"

This is almost always the wrong question, and it has a specific failure
mode: reaching for a new DSL feature. Adding a transform or a condition
operator to fit one provider means every runtime, in every language,
forever, must implement it identically — for one provider's quirk. That
is why the transform set is closed and why the answer to a genuinely
inexpressible provider is the native path rather than a bigger DSL.

If you find yourself drafting an RFC to add a canonicalisation transform,
read [modify-the-spec/02-rule-of-least-power](../modify-the-spec/02-rule-of-least-power.md)
first. The native escape hatch exists precisely so the DSL does not have
to grow.

## Decide, then go

- **Manifest** (the overwhelmingly common case): continue to
  [Step 1 — What a manifest is](01-what-a-manifest-is.md) and work
  through all eleven steps.
- **Native**, and you can name which of the five signals applies: read
  [Step 12 — The native path](12-the-native-path.md), which tells you
  what changes and which of the eleven steps still apply to you (most of
  them do).
- **Unsure?** Open a
  [new-provider issue](https://github.com/miki-smart/esiipayment/issues/new/choose)
  describing the awkward part before you write anything. Being talked out
  of a native implementation is a good outcome; discovering halfway
  through a PR that your manifest was viable all along is not.
