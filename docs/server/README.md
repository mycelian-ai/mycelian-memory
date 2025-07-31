# Synapse Documentation

This folder collects reference and design material for the Synapse project. Files can be large, so they are **not** automatically loaded into every AI prompt.

## How to bring a document into context

1. Inside the chat, mention the file with the tag syntax:

   `@docs/<relative-path>.md`

   Example:

   `@docs/reference/api-documentation.md`

2. The agent will attach the referenced file (or the specific section if you add a `#heading`) to the conversation context.

3. Keep files under ~300 lines where possible; split oversized docs into smaller ones.

## Directory layout

```
docs/
  reference/            # API specs, protocol buffers, OpenAPI, etc.
  design/               # Design documents and milestones
  adr/                  # Architecture Decision Records (see memory-bank/decisionLog.md)
  README.md             # (this file)
```

For guidance on structuring docs and size limits, see the Memory Bank rule: `.cursor/rules/external-docs.mdc`. 