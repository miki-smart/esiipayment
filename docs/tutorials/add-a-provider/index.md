# Add a provider

For almost every provider you need **no programming language** to add a
payment provider to ESIIPayment. A provider is a YAML manifest, a
metadata file, and a set of recorded HTTP examples ("cassettes"). If
you've ever integrated an Ethiopian payment provider's API — at a job, on
a side project, or just reading their public docs closely — you have what
it takes to write one of these.

The only tool you need beyond a text editor is
[Docker](https://docs.docker.com/get-docker/), to run the one piece of
application code this project has: a validator that checks your work.
You will never write or run any other code.

The exception is a small number of providers whose APIs the manifest DSL
structurally cannot describe — one that signs a canonicalized parameter
string, say, or speaks SOAP rather than HTTP+JSON. Those take the
**native** path: a capability declaration here, and hand-written code in
each language runtime.
[Step 0](00-which-kind.md) is the five-minute decision, and it is worth
reading before anything else even though the answer is nearly always
"manifest".

This tutorial refers throughout to
[the canonical scenario](../../examples/scenario.md) — a coffee
merchant's order `INV-2291`, paid via Chapa, resulting in a redirect to a
hosted checkout page — as its running example, so you're seeing the same
transaction the other tracks in this project use.

## The steps

0. [Which kind of provider is yours?](00-which-kind.md) — manifest or
   native. Five minutes, and it saves a rejected PR.
1. [What a manifest is](01-what-a-manifest-is.md) — and why it's data,
   not code.
2. [Editor setup](02-editor-setup.md) — one line gets you autocomplete
   and inline validation for free.
3. [Copy the template, walk every section](03-copy-the-template.md) —
   `auth`, `capabilities`, `operations`, `flows`, `errors`, `webhook`.
4. [Choosing the right `NextAction`](04-choosing-next-action.md).
5. [Mapping errors](05-mapping-errors.md) — and specifically, which ones
   are `ResolveFirst`.
6. [Validating with Docker](06-validating-with-docker.md) — and reading
   what it tells you when you're wrong.
7. [Recording cassettes](07-recording-cassettes.md).
8. [Generating golden files](08-golden-files.md) — and the honest limit
   of what "the build is green" actually proves.
9. [The clean-room rule and the DCO](09-clean-room-and-dco.md).
10. [Opening the PR](10-opening-the-pr.md).
11. [Verification status and tiers](11-verification-and-tiers.md) — what
    `provisional` means, and how a manifest earns `verified`.

And, only if Step 0 sent you there:
[Step 12 — The native path](12-the-native-path.md), which covers what
changes when your provider needs code instead of a manifest, and which of
the eleven steps above still apply to you (most of them do).

Read [spec/03-manifest-dsl.md](../../../spec/03-manifest-dsl.md)
alongside this tutorial for the full normative reference. This tutorial
is the practical walkthrough; that document is the source of truth
whenever the two seem to disagree.
