## What does this change?

## Does this touch a closed enum, the expression language, or the manifest schema's validation behaviour?

- [ ] Yes — link the accepted RFC issue: #
- [ ] No — this is a clarification, correction, or additive vector/doc
      change that doesn't need the RFC process (see
      [GOVERNANCE.md](../../GOVERNANCE.md#rfc-process))

## What's updated

- [ ] `spec/*.md`
- [ ] `schema/*.schema.json`
- [ ] `vectors/*`
- [ ] Version bump per [spec/08-versioning.md](../../spec/08-versioning.md), if applicable

## Impact

Does this invalidate any existing manifest in `providers/`, or change
what `esiipayment validate`/`esiipayment replay` accept? If so, are those
providers updated in this same PR or a tracked follow-up?

## Checklist

- [ ] `esiipayment lint` passes
- [ ] `esiipayment validate` still passes for every existing provider (or the
      PR explains why not, per above)
