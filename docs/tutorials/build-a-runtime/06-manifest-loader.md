# 6. The manifest loader

Parse `manifest.yaml`, `metadata.yaml`, and `cassettes/*.yaml` into
whatever typed structures your language wants to run the interpreter
against, and reject anything [03-manifest-dsl.md](../../../spec/03-manifest-dsl.md)
doesn't allow. Still no HTTP, no persistence — this chapter ends with a
loaded, validated, in-memory manifest and nothing has been executed yet.

## Reject unknown keys

The DSL's own vocabulary (every top-level section, every `flows` step
field, every `auth`/`capabilities`/`emit` key) is closed:
`additionalProperties: false` at every level in
[`schema/manifest.v1.schema.json`](../../../schema/manifest.v1.schema.json).
An unrecognized key is a validation failure, not a silently ignored
typo — this is what keeps a manifest from rotting the moment a
misspelled key stops meaning anything and nobody notices.

You don't strictly need a general JSON Schema validation library to get
this right, and there's a real argument against reaching for one
reflexively: a hand-rolled loader that decodes into your language's own
typed structures, in **strict mode** (reject any key not matching a
known field — most serialization libraries have a "strict"/"deny unknown
fields" mode; use it), gets you the closed-vocabulary check for free from
your type system, at the cost of hand-maintaining the field list instead
of pointing at the schema file. Either approach is legitimate; know
which one you picked and why, since it changes where "the schema" lives
for your runtime — in a `.schema.json` file you interpret generically, or
in your own typed structures that happen to describe the same shape.

The one deliberate exception, at every layer: `call.headers`, `call.body`,
`status_map.values`, and (for a native provider) nothing at all, since a
native provider has no `flows` — these carry the *provider's own*
vocabulary, which this spec has no way to enumerate in advance. Don't
apply strict-unknown-key rejection to these; they're intentionally
open maps.

## `spec_version`

Refuse to execute a manifest whose `spec_version` your interpreter
doesn't implement, with a clear error — never a best-effort
interpretation of an unfamiliar DSL version
([08-versioning.md](../../../spec/08-versioning.md)). Check this as one
of the very first things your loader does, before attempting to
interpret anything else in the file; a manifest targeting a `spec_version`
you don't understand is not safe to partially load.

## The closed enums, again

Every enum value your loader reads off a manifest — `auth.shape`,
`capabilities.operations`, `capabilities.next_actions`, `emit.status`,
`emit.next_action.type`, `errors[].failure_code`, `errors[].retry_class`,
`webhook.verification.scheme` — gets checked against the same closed
sets from chapter 2. Two cross-checks specifically worth getting right
because they catch a whole class of manifest-authoring mistake at load
time rather than at replay time:

- `capabilities.next_actions` must equal **exactly** (not merely be a
  superset of) the union of `next_action.type` values every flow
  actually emits.
- Every `emit.next_action` must supply that variant's required fields
  (per [01-domain-model.md#nextaction-carries-its-own-payload](../../../spec/01-domain-model.md#nextaction-carries-its-own-payload))
  and no field outside that variant's declared set.

## Flow-graph checks

For each `flows` entry: exactly one entry step (the one step nothing
else's `goto`/`status_map.values` ever targets — compute this by
elimination, never by relying on key order, which most languages'
map/dict types don't preserve); every `goto`/`status_map` target names a
real step in the same flow; every step is reachable from the entry step;
a step emitting a terminal `PaymentStatus` never also declares a further
`goto`/`status_map` (Invariant I2). `errors[]` needs exactly one
`transport: timeout` entry, mapped to `ProviderTimeout`/`ResolveFirst`,
and no two entries whose match conditions could both apply to the same
response.

## Position-dependent namespace availability

Chapter 3's `extract`/`event`/`input`/`auth` namespaces aren't available
everywhere. Check statically, at load time, that a manifest never
interpolates a namespace that isn't populated yet at that point in its
flow:

- `extract`/`event` are available anywhere in a flow whose **entry
  step** has a `call` or a `webhook` trigger — not just on the step that
  literally made the call, since a conformant interpreter threads that
  one response through every step in the chain that handles it (chapter
  7). A step reached purely by `goto`/`status_map`, with no `call` of
  its own, can still read `${extract...}` from the entry step's
  response.
- `input` is available only in a step reached via `triggers: [input]`.
- `auth` is available only in `auth.apply`/`auth.token`, and only when
  `auth.shape` is `oauth2_client_credentials` — never inside `flows` at
  all.

## Native providers

A directory with `capabilities.yaml` (and no `manifest.yaml`) is a
[native provider](../../../spec/03-manifest-dsl.md#native-providers):
your loader should recognize this shape and route it away from the
interpreter entirely, since there are no `flows`/`errors`/`webhook` to
execute. Your SDK still needs to know this provider's identity,
`auth.shape`/`auth.fields` (for a generic credential-input form), and
`capabilities` (for the catalog/capability matrix); it's implemented as
hand-written code in your own SDK, held to the same cassette/golden-file
bar as any manifest-driven provider, just not interpreted by the part of
your runtime this track builds in chapters 7-8.

## What "done" looks like for this chapter

A loader that turns `manifest.yaml` + `metadata.yaml` + `cassettes/*.yaml`
into validated in-memory structures, rejecting every malformed manifest
you can construct (start by mutating a working one: rename a key,
delete the `transport: timeout` entry, add an extra `next_action` field,
point a `goto` at a step name that doesn't exist) and accepting every
manifest in this repository, including native ones. Still nothing has
executed a flow.
