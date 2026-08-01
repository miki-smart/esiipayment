# 2. The rule of least power

The manifest DSL is deliberately **not** Turing-complete. No loops, no
user-defined functions, no boolean composition, no arithmetic beyond the
closed transform set. This is a design principle with a name — the
[rule of least power](https://www.w3.org/2001/tag/doc/leastPower.html):
choose the least powerful language adequate for a given purpose, because
a less powerful language is easier to analyze, easier to implement
identically everywhere, and impossible to accidentally turn into a
general-purpose scripting environment nobody decided to build on
purpose.

## Why "the DSL can't express my provider" isn't the end of the analysis

Every time someone hits a real DSL limitation, there are two possible
conclusions, and it's tempting to reach for the wrong one first: "the
DSL needs a new feature" or "this provider needs the native escape
hatch." The DSL-feature path feels smaller from inside one PR — one more
transform, one more operator — but it's the more expensive path
project-wide: every feature added to the DSL is a feature every
conformant runtime must now implement identically, forever, whether or
not most providers ever use it. The native-provider path
([03-manifest-dsl.md#native-providers](../../../spec/03-manifest-dsl.md#native-providers))
costs exactly one language's worth of hand-written code, for exactly the
one provider that actually needs it, with no obligation on any other
runtime or any other provider.

## The five signals that mean "native," not "new DSL feature"

Directly from
[03-manifest-dsl.md#the-five-signals](../../../spec/03-manifest-dsl.md#the-five-signals):
branching on a parsed intermediate value, canonicalisation before
signing, session or cookie state, a non-HTTP channel, or a status
vocabulary needing more than equality comparison. If a provider's real
need matches one of these, that's a strong signal the right answer is
the native path, not a schema change — and it's worth checking against
this list explicitly before opening an RFC, since "my provider needs X"
frequently turns out to be one of these five in disguise.

## When a DSL change is actually the right call

Sometimes it genuinely is — the DSL has grown before (typed
`NextAction` payloads, `auth.apply`/`auth.token`) when the previous shape
was actively wrong for *every* provider using a given pattern, not just
inconvenient for one. The test that separates a legitimate DSL change
from scope creep: does this serve a *pattern* real across multiple
providers (or realistically anticipated to be), expressible within the
DSL's existing power level (still no loops, no user functions, no
arbitrary computation) — or does it exist to accommodate one specific
provider's specific quirk? The first is what
[the RFC process](03-the-rfc-process.md) exists to evaluate seriously.
The second is what the native escape hatch is for.
