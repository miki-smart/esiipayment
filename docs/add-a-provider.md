# Adding a provider

This guide has moved to
[docs/tutorials/add-a-provider/](tutorials/add-a-provider/index.md), an
eleven-step tutorial that assumes no programming knowledge and no
toolchain beyond Docker, built around
[the canonical scenario](examples/scenario.md) — a coffee merchant's
order paid through Chapa — as its running example throughout.

Providers are onboarded on one of two tracks, and
[Step 0](tutorials/add-a-provider/00-which-kind.md) is the five-minute
decision between them:

- a **manifest provider** is one YAML file that every runtime already
  knows how to execute — the right answer for almost every provider, and
  what the eleven steps teach;
- a **native provider** is a capability declaration here plus
  hand-written code in each language runtime, for the few providers whose
  APIs the DSL structurally cannot describe (see
  [the five signals](../spec/03-manifest-dsl.md#native-providers)).
  [Step 12](tutorials/add-a-provider/12-the-native-path.md) covers that
  track.

If you're changing a schema, an enum, or the manifest DSL itself rather
than describing one provider's behaviour, that's a different audience
and a different guide:
[docs/tutorials/modify-the-spec/](tutorials/modify-the-spec/index.md).

If you followed a link here from somewhere else, please update it to
point at [docs/tutorials/add-a-provider/](tutorials/add-a-provider/index.md)
directly.
