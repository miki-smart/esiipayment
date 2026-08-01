# Localization

Every `FailureCode` ([spec/01-domain-model.md](../spec/01-domain-model.md#failurecode))
needs a user-facing message eventually — something an integrator's
checkout UI can show the payer when a payment fails. Left to each SDK,
that message ends up hardcoded in English five separate times, once per
language runtime, drifting independently. [`messages/`](../messages/)
is the shared, machine-readable fix: one message catalogue per language,
keyed by the same closed `FailureCode` set every runtime already uses,
so every SDK reads from the same source instead of inventing its own English
strings.

## What's here

```text
messages/
  en.json   English (baseline)
  am.json   አማርኛ (Amharic)
  om.json   Afaan Oromoo (Afaan Oromo)
  ti.json   ትግርኛ (Tigrinya)
```

Each file ([schema/messages.v1.schema.json](../schema/messages.v1.schema.json))
carries exactly the fourteen `FailureCode` members as keys, one sentence
each, plus a `reviewed_by_native_speaker` flag.

A runtime is expected to look up `messages/<locale>.json[failure_code]`
to render a failure to the payer, falling back to `en.json` for any
locale it doesn't ship a translation for. This repository is the source
of the message *text*; how a runtime bundles, loads, or caches these
files is its own concern, the same way manifest interpretation is.

## Verification status — please read before using these in production

**`am.json`, `om.json`, and `ti.json` are machine-drafted and have not
been reviewed by a fluent or native speaker of Amharic, Afaan Oromo, or
Tigrinya respectively.** `reviewed_by_native_speaker: false` in each file
says this explicitly, the same way a provider manifest's
`verification.status: provisional` says its endpoint details are
unconfirmed (see [add-a-provider.md](add-a-provider.md)) — this is the
same honesty norm applied to language instead of API behaviour. Do not
ship these three files to real payers without a native speaker
confirming the wording is natural, unambiguous, and not simply a stilted
literal translation. `en.json` is the one file in this set written
directly, not translated, and is marked `reviewed_by_native_speaker: true`
on that basis.

If you can review or correct `am.json`, `om.json`, or `ti.json`: that is
a genuinely high-value, no-programming-language-required contribution.
Open a PR editing the relevant file's `messages` values and flip
`reviewed_by_native_speaker` to `true` once you're confident in the
wording; no other file needs to change. This follows the same ordinary
PR review path as any other change in [GOVERNANCE.md](../GOVERNANCE.md)
(no RFC needed: correcting a message's wording doesn't change the
`FailureCode` set or any schema shape).

## Adding a language

1. Copy `messages/en.json` to `messages/<ISO-639-1-or-639-2-code>.json`.
2. Translate every value in `messages`; keep every key exactly as-is
   (the fourteen `FailureCode` members are closed —
   [spec/01-domain-model.md](../spec/01-domain-model.md#failurecode) —
   and `schema/messages.v1.schema.json` rejects a file missing one or
   carrying an extra key).
3. Set `language` to the file's own code and `language_name` to the
   language's name for itself.
4. Set `reviewed_by_native_speaker` honestly: `false` unless a fluent
   speaker has actually reviewed the wording, not merely run it through
   translation software.

No RFC is required to add a new language file: it's purely additive data,
the same class of change as a new vector file (see
[spec/08-versioning.md](../spec/08-versioning.md)).
