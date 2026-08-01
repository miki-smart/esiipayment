# 9. The clean-room rule and the DCO

Read this chapter before you write a single field, not after — it's
easier to stay inside these rules from the start than to retroactively
check whether you did.

## The clean-room norm

Write your manifest from:

- Provider documentation you are **licensed to read** — public API
  docs, developer portals, anything publicly published.
- Your own **sandbox testing** — what you actually observed a real
  sandbox account do.
- Your own **memory** of how the integration behaves, from having built
  it before.

**Never** copy from, transcribe, or closely paraphrase an employer's
proprietary integration code, an internal wiki page, or any source you
don't have the right to redistribute. If you're unsure whether something
you know came from a redistributable source, err toward leaving it out,
or check with your employer first.

This matters more for this project than it might for a typical open
source contribution, precisely because of who's best positioned to write
these manifests: a meaningful fraction of the people who know a real
Ethiopian PSP's API well learned it while integrating it at a job. The
DCO below (not a copyright assignment, not a CLA) is specifically what
lets you contribute that *knowledge* — written fresh, from what you're
licensed to read — without it being confused with contributing your
employer's *code*. The same discipline applies here as when someone
who's seen a patented or trade-secret algorithm implements a work-alike
from a public specification instead of from memory of the original
proprietary source: same destination, deliberately different path to
get there.

If you're unsure whether a specific detail is safe to include, the
practical test is: *could I have written this after reading only the
provider's public docs and testing my own sandbox account, with no
memory of anything my employer's internal systems ever showed me?* If
the honest answer is no, leave that detail out and mark it `UNVERIFIED`
instead (chapter 11) — an honest gap is always better than a detail that
shouldn't be here at all.

## The DCO: sign off on every commit

This project uses the
[Developer Certificate of Origin](https://developercertificate.org/)
(DCO), not a Contributor License Agreement. Every commit needs a
`Signed-off-by` trailer — `git commit -s` adds it automatically — which
certifies that the contribution is yours to submit, or that you have the
right to submit it under this project's license. See
[CONTRIBUTING.md](../../../CONTRIBUTING.md#sign-off-not-a-cla) for the
full certification text.

The adapter PR template (chapter 10) asks you to affirm the clean-room
norm directly, as its own checkbox — not because the DCO sign-off alone
doesn't already cover it, but because a specific, deliberate question
("did this come from documentation you're licensed to read, or your own
testing — not an employer's proprietary source?") gets a more honest
answer than a general legal certification most people sign without
rereading closely. Answer it as the specific, direct question it is.

Next: [opening the PR](10-opening-the-pr.md).
