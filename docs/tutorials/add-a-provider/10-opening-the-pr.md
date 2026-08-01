# 10. Opening the PR

Before opening anything, confirm all three of these pass locally
(chapter 6):

```text
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator validate providers/<your-provider>
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator replay providers/<your-provider> --assert-golden
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator lint
```

Open your PR using the repository's **adapter** PR template
(`.github/PULL_REQUEST_TEMPLATE/adapter.md`) — GitHub should offer it to
you automatically for a PR that only touches `providers/`, or you can
select it explicitly. It asks for exactly the things `esiipayment
validate` *cannot* check, because they're judgment calls about this
specific provider, not structural rules:

## NextAction mapping

For each non-terminal state your flows emit, which `NextAction` did you
choose and why? If this provider's behaviour doesn't cleanly match a row
in [chapter 4](04-choosing-next-action.md)'s table, explain your
reasoning rather than leaving it unstated. A validator can confirm
`RedirectToUrl` is a real `NextAction` member with a `url` field; it has
no way to confirm that *redirect* is actually the right choice for what
this provider's flow does. Only you, having read this provider's
behaviour, can answer that.

## Which errors are `ResolveFirst`

Beyond the mandatory `transport: timeout` entry, does this provider have
any error condition where the outcome is genuinely ambiguous from the
response alone? [Chapter 5](05-mapping-errors.md) covers why this
matters; the PR template asks you to state your answer explicitly,
including "none, beyond the mandatory case" if that's genuinely true.

## Idempotency

Does this provider have its own idempotency/deduplication mechanism (an
idempotency key field, a client-reference it deduplicates on)? Does your
manifest actually use it? If the provider has none, that's a real,
worth-stating constraint on what this adapter can guarantee — not
something to paper over by pretending the concern doesn't apply.

## Why these specifically can't be automated

A validator can check that your manifest is *internally consistent* —
every field present, every enum member closed, every flow reachable. It
cannot check whether your `NextAction` choice actually matches this
provider's real behaviour, because that's a claim about the outside
world, verifiable only by someone who's read this provider's
documentation or watched their sandbox actually behave that way. That's
exactly what an adapter reviewer checks, and exactly why the PR template
asks you to state your reasoning rather than just letting CI pass
silently: a reviewer confirming "yes, that reasoning is sound" is a
fundamentally different, more valuable review than a reviewer confirming
"yes, the build is green."

Answer these for real, in your own words, specific to this provider —
not restated boilerplate. This is the part of an adapter review that
actually needs a human, on both sides.

Next: [verification status and tiers](11-verification-and-tiers.md).
