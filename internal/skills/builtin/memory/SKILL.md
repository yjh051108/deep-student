---
name: memory
description: Long-term persistent memory with decay and privacy mode.
version: 0.1
tier: builtin
---

# Smart Memory

Inspired by mem0 / memU.

## When to use
- End of a session
- User asks to "remember" something

## Steps
1. Extract facts from conversation
2. Compare to existing memory
3. LLM decides ADD / UPDATE / APPEND / DELETE / NONE
4. Apply idempotently
5. Aggregate into user profile
