# Contributing to ESIIPayment

Thank you for considering a contribution. This project deliberately
separates two (really three) kinds of work that have nothing to do with
each other. Find your role below and start there, rather than reading
this whole repository front to back.

| You are... | You need... | Start at |
|---|---|---|
| An **adapter author** describing a payment provider's behaviour | No programming language. YAML and a text editor. | [docs/add-a-provider.md](docs/add-a-provider.md) |
| A **runtime implementer** building a language SDK against this spec | One programming language, no provider-specific knowledge. | [docs/build-a-runtime.md](docs/build-a-runtime.md) |
| A **spec contributor** proposing a change to the contract itself | Familiarity with the existing `spec/` documents. | [GOVERNANCE.md](GOVERNANCE.md#rfc-process) |

If you're an **integrator** using an SDK to accept payments rather than
contributing to this repository, you want
[docs/use-a-provider.md](docs/use-a-provider.md) instead: that's not a
contribution guide, just usage docs, but it's listed here since people
often land on CONTRIBUTING.md looking for it.

New to the adapter-author or runtime-implementer role and want to see the
mechanics worked through end to end before reading the full reference
guides? [docs/tutorial.md](docs/tutorial.md) builds one small fictional
adapter and one minimal SDK, start to finish.

## Before you contribute

### Sign-off, not a CLA

This project uses the [Developer Certificate of Origin](https://developercertificate.org/)
(DCO) instead of a Contributor License Agreement. Every commit must
include a `Signed-off-by` trailer (`git commit -s` adds this
automatically), which is your certification that:

> (a) The contribution was created in whole or in part by me and I have
> the right to submit it under the open source license indicated in the
> file; or
>
> (b) The contribution is based upon previous work that, to the best of
> my knowledge, is covered under an appropriate open source license and I
> have the right under that license to submit that work with
> modifications, whether created in whole or in part by me, under the
> same open source license (unless I am permitted to submit under a
> different license), as indicated in the file; or
>
> (c) The contribution was provided directly to me by some other person
> who certified (a), (b) or (c) and I have not modified it.
>
> (d) I understand and agree that this project and the contribution are
> public and that a record of the contribution (including all personal
> information I submit with it, including my sign-off) is maintained
> indefinitely and may be redistributed consistent with this project or
> the open source license(s) involved.

This matters more than usual for this specific project: a meaningful
fraction of the people best positioned to write a Chapa, ArifPay, or
SantimPay adapter learned that provider's API while integrating it at a
job. The DCO is what lets you contribute that *knowledge* (written fresh,
from documentation you're licensed to read) without it being confused
with contributing your employer's *code*.

### The clean-room norm

Write every provider manifest from provider documentation you are
licensed to read (public API docs, your own test-account observations)
and from your own memory of how the integration behaves. **Never** copy
from, transcribe, or closely paraphrase an employer's proprietary
integration code, internal wiki, or any source you don't have the right
to redistribute. If you're unsure whether something you know came from a
redistributable source, err toward not including it, or check with your
employer first. This is exactly the same discipline used when someone
who has seen a patented or trade-secret algorithm implements a
work-alike from a public spec instead of from memory of the original
source.

The adapter PR template asks you to affirm this directly: see
[docs/add-a-provider.md](docs/add-a-provider.md).

### Code of Conduct

Participation in this project is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## The one rule that applies to every contribution

**No application code, in any programming language, under `providers/`,
`vectors/`, or `schema/`.** Provider behaviour is YAML data; the only
executable code this repository permits is `tools/validator` and CI
workflow definitions. `no-code.yml` enforces this mechanically, but it's
worth understanding *why* before you hit it: see
[spec/00-overview.md](spec/00-overview.md).

## Getting your PR reviewed

See [CODEOWNERS](.github/CODEOWNERS) and [GOVERNANCE.md](GOVERNANCE.md)
for who reviews what, and when a change needs the RFC process instead of
ordinary review. In short: provider changes are reviewed by adapter
reviewers with provider-specific knowledge; changes to `spec/`, `schema/`,
`vectors/`, or `tools/` are reviewed by maintainers; anything touching a
closed enum, the expression language, or the manifest schema's validation
behaviour needs an RFC first, regardless of who'd normally review it.
