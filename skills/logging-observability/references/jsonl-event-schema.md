# JSONL event schema

Recommended baseline event shape for local CLI session logging:

- `timestamp`: ISO-8601 UTC string
- `sessionId`: UUID for the current CLI session
- `source`: logical producer such as `cli`, `stdout`, `stderr`, or `dev-wrapper`
- `eventType`: stable machine-readable event name
- `level`: `debug`, `info`, `warn`, or `error`
- `cwd`: current working directory
- `gitRoot`: detected Git root or `null`
- `skillId`: optional skill identifier for skill-backed commands
- `command`: optional command string for wrapped shell/process execution
- `message`: optional human-readable detail
- `payload`: optional JSON object/array/primitive for structured metadata

Guidelines:

1. Keep each event self-contained so single-line grep remains useful.
2. Prefer stable `eventType` values over parsing free-form text.
3. Reserve `payload` for structured details that do not fit fixed top-level keys.
4. Do not emit secrets, tokens, or full environment dumps.
