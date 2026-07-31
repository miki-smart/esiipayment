# Security Policy

## Scope

This repository is a specification, a set of JSON Schemas, test vectors,
and one validator CLI (`tools/validator`); it does not run a payment
service, hold funds, or process live transactions. The security-relevant
surface here is narrower than a typical application, but not zero:

- **The validator (`tools/validator`)** parses YAML/JSON from provider
  manifests and could in principle have a parsing or resource-exhaustion
  vulnerability. It is not, itself, exposed to untrusted network input;
  it's a local/CI tool run against files in this repository or a PR diff.
- **The spec's security-relevant invariants** (webhook signature
  verification being constant-time over the raw body
  ([Invariant I11](spec/02-invariants.md#i11)), credentials never
  appearing in manifest source
  ([Invariant I9](spec/02-invariants.md#i9))) are things every language
  runtime must get right. A flaw *in the spec itself* here (a vector that
  encodes an insecure comparison as "correct," a schema that would accept
  a literal secret in a manifest field) is a real, in-scope
  vulnerability class for this repository, distinct from a vulnerability
  in any particular runtime's *implementation* of the spec (report those
  to the relevant `esiipayment-<language>` repository instead).
- **A provider manifest that leaks or mishandles a real secret**: per
  Invariant I9, no manifest should ever contain a literal credential
  value. If you find one that does (even in a cassette fixture, which
  should only ever contain clearly-fake test values), treat it as a
  security report, not an ordinary bug, in case the value is real.

## Reporting a Vulnerability

Please report security concerns privately rather than opening a public
issue, using GitHub's private vulnerability reporting for this repository
(**Security** tab → **Report a vulnerability**). If that's not available
to you, open an issue titled only "Security report: see email" with no
details, and a maintainer will follow up with a private contact.

Please include:

- What you found and where (file/line if applicable).
- Why you believe it's security-relevant per the scope above, rather than
  an ordinary correctness bug (ordinary bugs should go through the normal
  issue templates instead).
- A minimal reproduction if applicable (e.g. a manifest snippet that
  validates but shouldn't, or a vector whose "correct" answer is
  actually insecure).

## Response

A maintainer will acknowledge a report within a reasonable time and work
with you on a fix and disclosure timeline before any public discussion of
the specifics. Given this repository's nature (a spec, not a running
service), most fixes will land as a spec/schema/vector correction plus,
if the issue is severe enough to qualify under
[GOVERNANCE.md](GOVERNANCE.md#rfc-process) (e.g. it requires changing a
closed enum or the expression language), an expedited RFC.
