# 6. Validating with Docker

`esiipayment` is a small command-line tool
([`tools/validator/`](../../../tools/validator/)) that checks your
manifest against everything this tutorial has described so far, and
more that would be tedious to check by eye. You don't need a programming
language toolchain to run it — only [Docker](https://docs.docker.com/get-docker/).

## Build it once

From the root of your clone of this repository:

```text
docker build -t esiipayment-validator tools/validator
```

This downloads a small Go build environment, compiles the tool, and
throws the build environment away, leaving you with one image named
`esiipayment-validator`. You only need to do this once (or again later,
if `tools/validator/` itself changes upstream and you pull that update).

## Run it against your provider

```text
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator validate providers/<your-provider>
```

(On Windows PowerShell, replace `$PWD` with `${PWD}`, or use `%cd%` in
`cmd.exe`.) This mounts your repository into the container and runs
`esiipayment validate` against your provider's directory. Two possible
outcomes:

```text
no findings
```

means your manifest is structurally valid — every closed enum member
recognized, every flow reachable, the mandatory timeout entry present,
every `next_action` carrying its required fields, and everything else
this tutorial has walked through. That's necessary, not sufficient: it
doesn't yet mean your manifest correctly describes the *real* provider's
behaviour — chapters 7 and 8 are what actually exercise that.

Or you'll see one or more lines like:

```text
[ERROR] missing-timeout-mapping: errors must include exactly one entry with match.transport: timeout
[ERROR] next-actions-mismatch: capabilities.next_actions must equal exactly the NextAction values flows emit: declared [RedirectToUrl], emitted [Poll, RedirectToUrl]
```

## Reading the output

Each line is `[SEVERITY] rule-name: message`. The `rule-name` (like
`missing-timeout-mapping` or `next-actions-mismatch` above) is a stable,
short identifier for the *kind* of problem — useful for searching this
project's docs or asking for help, since "next-actions-mismatch" is a
more useful search term than the full sentence around it. The message
itself tells you exactly what's wrong and usually exactly what's
missing: the second example above is telling you a flow emits `Poll`
somewhere that `capabilities.next_actions` doesn't declare — either add
`Poll` there, or you have a step emitting a `NextAction` you didn't mean
to.

Fix one error, re-run, repeat. If you set up your editor as chapter 2
described, most of these will already have been caught before you ever
ran the container at all — this command is your safety net, not your
primary feedback loop.

## `esiipayment lint`

Once `validate` is clean, also run:

```text
docker run --rm -v "$PWD":/repo -w /repo esiipayment-validator lint
```

This is the repository-wide check (every provider, not just yours, plus
a few checks specific to the repo as a whole — the required schema
header, for instance). It should already be clean if `validate` was, but
it's part of what your PR needs to pass (chapter 10), so it's worth
running locally rather than finding out from CI.

Next: [recording cassettes](07-recording-cassettes.md) — the part where
your manifest starts getting tested against something that looks like
real provider behaviour.
