# Node service pattern

A practical Node.js observability pattern for lightweight tools and services:

- Load repository-local configuration once near process startup.
- Resolve defaults eagerly so downstream modules work with normalized values.
- Create one session logger per top-level command invocation.
- Write human-readable output to stdout/stderr and structured output to JSONL.
- Record lifecycle markers for command start, success, failure, and exit status.
- Prefer append-only log files to avoid coordination overhead across processes.

This pattern scales well for CLIs, local dev wrappers, and simple background tasks before adopting a centralized log pipeline.
