# Phase 5: LLM, news, sentiment, synthesis

Goal: same request construction, validation, fallback and degraded-stage semantics. LLM output itself is non-deterministic and is tested with recorded responses.

## Scope
OpenAI-compatible client with per-provider settings reload, backends (local, Pi SSH, OpenAI, Ollama, GPUHub, Laya), prompt loaders, structured output validation, sentiment analyzer and gate, synthesis, 7 news sources, filtering, PDF extraction, Pi llama.cpp lifecycle over SSH, pi-agent subprocess, accuracy and evaluation jobs.

## Steps
1. Ledger rows per source, backend and job.
2. Record real responses for each provider and news source (secrets removed).
3. Red tests: prompt rendering, request bodies, validation and rejection, retry, circuit breaker, timeout, fallback to keyword sentiment, `stage_degraded` and metrics emission.
4. Implement clients, parsers (fuzz the parsers), SSH lifecycle with context timeouts, subprocess handling.
5. Accuracy and evaluation jobs against copies of data; compare outputs.
6. Failure drills: provider down, malformed JSON, timeouts, SSH failure.

## Gate G5 exit criteria
All recorded-response suites green; degraded and fallback behavior and metrics match; no secret in fixtures.
