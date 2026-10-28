---
name: llm-runtime-settings
description: Implement or troubleshoot LLM runtime settings in the Java backend. Use for backend/llm config/providers, AppSettingsStore or app_settings, /api/settings/llm, sentiment/synthesis services, or LLM health changes.
---

# LLM runtime settings

Work as a senior/staff engineer: keep provider settings live-resolved at the call boundary rather than baked into long-lived beans.

## Scope

In scope: LLM backend routing, settings liveness, dynamic `OpenAiChatModel` resolution, model caches, header injection, timeouts, and OpenCode session defaults.

Out of scope (do not generalise): prompt wording or stage orchestration product logic, unrelated Java persistence, and dashboard styling.

## Invariants

- Resolve provider URL, model, and API key from `AppSettingsStore` at call time; never rely on a startup `OpenAiChatModel` bean for settings that can change.
- Rebuild only when a fingerprint of resolved settings changes; keep model construction thread-safe (concurrent map).
- Keep timeout policy split: CPU-bound llama.cpp backends use 2850s and 0 retries; remote OpenAI-compatible backends keep Spring defaults.
- Treat `llm.extra_headers` as JSON and inject defaults for OpenCode endpoints (`x-opencode-session` unless explicitly supplied).
- Correctly distinguish failed/unreachable inference from busy inference; timeouts after `/health` success should be `BUSY`, not failure.
- Keep parsing fallback behavior intact; settings changes must not silently alter prompts.
- Test liveness with settings-write scenarios and stale-bean regression tests.

## Verification

- Run the narrowest relevant backend tests first, then the affected module checks.
- Report checks actually run and any that could not be run.

Preserve unrelated worktree changes.