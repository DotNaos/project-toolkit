# Observability roadmap

Suggested incremental path beyond the MVP:

1. **Session logging**
    - JSONL files per invocation
    - lifecycle and command output events
2. **Agent execution telemetry**
    - stream adapter events as they happen
    - attach thread and tool metadata
3. **Quality-of-life tooling**
    - add `repo-kit logs tail` or summary commands
    - rotate or prune old local sessions
4. **Centralized export**
    - optional OTLP or HTTP export for CI/runtime environments
    - map local JSONL schema to a formal event contract
5. **Metrics and tracing**
    - command duration histograms
    - correlation ids across subprocesses and remote agent calls

Keep the local-first JSONL format simple so future ingestion remains boring in the best possible way.
