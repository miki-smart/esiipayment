# 1. What a manifest is

A **manifest** is a YAML file,
[`manifest.yaml`](../../../spec/03-manifest-dsl.md), that describes one
payment provider's behaviour entirely as data: what credentials it
needs, what operations it supports, what HTTP requests to make, and how
to interpret the responses. It is not a program. There is no branching
logic beyond a small, fixed set of comparisons; no loops; no functions
you write yourself. Everything a manifest can say is expressed in a
closed, deliberately small vocabulary — described in full in
[03-manifest-dsl.md](../../../spec/03-manifest-dsl.md) — that every
language runtime in this project (a .NET SDK, a Python SDK, a Node SDK,
and others) interprets identically.

## Why data, and not a plugin you'd write in some language

If adding a provider meant writing code, it would mean writing code
*five times* — once per language SDK — or picking one language and
leaving the other four unsupported for this provider indefinitely. Data
means write once, and every SDK that implements this spec correctly
supports your provider automatically, without you touching any of them.

It also means your contribution can be reviewed by someone who knows
this specific provider's API, without that reviewer needing to know
Go, or Python, or C#, or any programming language at all. An adapter
reviewer's job (see [GOVERNANCE.md](../../../GOVERNANCE.md)) is to judge
whether your manifest's *claims about the provider* are plausible and
whether the validator passes — not to read code for bugs, because there
isn't any to read.

## What you'll actually produce

By the end of this tutorial, `providers/<your-provider>/` will contain:

```text
providers/<your-provider>/
  manifest.yaml     # the technical description: auth, flows, errors, webhook
  metadata.yaml      # catalog/provenance info: tier, verification status, maintainers
  cassettes/          # recorded (or sandbox-run) HTTP request/response examples
  expected/            # the exact result each cassette should produce
```

Four files and two directories, all YAML or JSON, nothing else. The
[canonical scenario](../../examples/scenario.md) this tutorial follows —
a coffee merchant's order paid through Chapa — is itself exactly this
shape: see
[`providers/chapa/manifest.yaml`](../../../providers/chapa/manifest.yaml)
and
[`providers/chapa/cassettes/collect.canonical_scenario.yaml`](../../../providers/chapa/cassettes/collect.canonical_scenario.yaml)
for a finished, real example of everything this tutorial is about to
walk you through building from scratch.

Next: [editor setup](02-editor-setup.md), which costs one line and gets
you a lot back for it.
