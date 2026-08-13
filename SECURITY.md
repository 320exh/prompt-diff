# Security Policy

## Reporting a vulnerability

Please report security issues privately via GitHub's [private vulnerability reporting](https://github.com/320exh/prompt-diff/security/advisories/new) instead of opening a public issue. You should get a response within a few days.

## API keys and credentials

`prompt-diff eval` sends prompts to whatever model providers you target. Provider credentials are read from environment variables only — they are never written to disk, logged, or embedded in `.prompt` files:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`

Local Ollama runs (`llama3.1:8b`, etc.) require no key and never leave your machine.

Eval results, including model responses, are stored locally in `runs.db` (SQLite) and are not transmitted anywhere by `prompt-diff` itself. Treat `runs.db` and `report.html` as potentially containing sensitive output from your test cases if your prompts/tests include real user data.

## Scope

`prompt-diff` is a local CLI/dashboard. `prompt-diff ui` currently binds to all interfaces (`:<port>`), not just localhost, and has no authentication. Do not run it on a machine reachable from an untrusted network, and don't forward/expose the port publicly.
