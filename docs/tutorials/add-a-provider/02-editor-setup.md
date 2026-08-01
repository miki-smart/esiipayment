# 2. Editor setup

Every manifest in this project begins with exactly this line, before
anything else:

```yaml
# yaml-language-server: $schema=https://spec.esiipayment.et/schema/manifest.v1.schema.json
```

This one comment line is a convention several editors already understand
— [VS Code](https://code.visualstudio.com/) with Microsoft's free
[YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)
installed, several JetBrains IDEs out of the box, `vim-lsp` with the
right plugin — and it gets you, with **no other setup at all**:

- Autocomplete for every field name the schema defines.
- Inline red squiggles the moment you type something the schema
  doesn't allow — a typo'd key, a value that isn't one of the closed
  enum members, a required field you forgot.
- Hover documentation pulled directly from the schema's own field
  descriptions.

This matters specifically because you were told in the last chapter that
you don't need to know any programming language — this is what makes
that actually true in practice, not just in principle. Without this
line, you'd be finding out about a mistake only when you run the
validator (chapter 6); with it, your editor tells you immediately,
before you've even saved the file.

`esiipayment lint` (the same tool you'll use in chapter 6) checks that
this exact line is present, verbatim, as the first line of every
`manifest.yaml` — so it's not optional decoration, it's part of what
"a valid manifest" means in this project.

If your editor doesn't support `yaml-language-server` and you don't want
to switch: that's fine, this is a convenience, not a requirement to
write a correct manifest by hand. But if you've never tried it, it's
worth the five minutes of installing the extension before you start
chapter 3 — most of what would otherwise be back-and-forth with the
validator in chapter 6 gets caught here instead, immediately, as you
type.

Next: [copy the template, and walk through every section](03-copy-the-template.md).
