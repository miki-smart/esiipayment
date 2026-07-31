## What does this change?

<!--
Runtime SDKs (esiipayment-dotnet, esiipayment-python, esiipayment-node, esiipayment-go,
esiipayment-php) live in their own repositories, not here — a PR against
*this* repo using this template is almost always a change to
tools/validator, or a spec clarification motivated by something you hit
while building a runtime. If you meant to open a PR in your SDK's own
repo, this isn't the right place.
-->

## Motivation

What did you hit while implementing a runtime that motivated this change?
Link the conformance issue if one exists.

## What in spec/ or tools/validator does this affect?

## Checklist

- [ ] If this changes `tools/validator` behaviour, existing manifests in
      `providers/` still validate (or the PR explains which ones now
      correctly fail, and why that's a fix rather than a regression)
- [ ] `esiipayment lint` passes
