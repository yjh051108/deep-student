---
name: cards
description: Generate Anki flashcards from documents or notes.
version: 0.1
tier: builtin
---

# Anki Cards

Generate high-quality Anki flashcards from any text source.

## When to use
- The user asks to "make cards from this document"
- The user wants batch generation into a specific deck
- The user wants to send cards to Anki

## Inputs
- `text` (required): source text
- `deck` (required): target deck name
- `template` (optional): template ID; default = "default"
- `batch` (optional): cards per chunk; default = 5

## Output
- A `*anki.Job` with cards and status.

## Steps
1. Split source into chunks
2. Call LLM to generate cards
3. Persist to VFS as `vfs://flashcard/<job-id>`
4. Optionally export to .apkg
