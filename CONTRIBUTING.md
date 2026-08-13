# Contributing to `prompt-diff`

First off, thanks for taking the time to contribute! 🎉

`prompt-diff` treats LLM system prompts as first-class source code. We welcome all kinds of contributions — new features, bug fixes, documentation, benchmark suites, and `.prompt` template examples.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Feature Requests](#feature-requests)
  - [Pull Requests](#pull-requests)
- [`.prompt` Template Guidelines](#prompt-template-guidelines)
- [Testing](#testing)
- [Style Guide](#style-guide)
- [License](#license)

## Code of Conduct

Be kind, constructive, and respectful. Harassment and other exclusionary behaviour will not be tolerated.

## Getting Started

### Prerequisites

- **Go 1.22+** — CLI engine and single-binary build
- **Node.js 20+** — Svelte dashboard UI development
- *(Optional)* **Ollama** or **Llama.cpp** — local model evaluation

### Development Setup

1. Fork the repository and clone your fork:

   ```bash
   git clone https://github.com/yourusername/prompt-diff.git
   cd prompt-diff
   ```

2. Build the CLI binary:

   ```bash
   go build -o prompt-diff .
   ```

3. For UI work, install the Svelte toolchain and rebuild the embedded assets:

   ```bash
   cd ui
   npm install
   npm run build
   ```

   The compiled UI is embedded into the binary via `embed.FS`.

## Project Structure

```
prompt-diff/
├── cmd/                  # CLI entrypoints (root, diff, eval, ui)
├── internal/             # Core logic: parse, diff, tokenize, cost, eval
├── ui/                   # Svelte + Tailwind dashboard sources
├── prompts/              # Example .prompt templates
├── tests/                # Evaluation test-case matrices (eval_suite.json)
└── README.md
```

## How to Contribute

### Reporting Bugs

Open an issue and include:

- The command you ran and the full error output
- Your `prompt-diff` version (`prompt-diff --version`)
- A minimal `.prompt` file (and commits, if relevant) that reproduces the problem

### Feature Requests

Open an issue describing the problem you're solving and the behaviour you'd expect. If the feature involves model cost or token accounting, outline how it should be measured.

### Pull Requests

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please make sure your PR description references any related issue, and that all tests pass before requesting review.

## `.prompt` Template Guidelines

Every `.prompt` template must declare its metadata in YAML frontmatter:

```yaml
---
name: Customer Support Classifier
version: 2.1.0
models:
  - gpt-4o-mini
variables:
  - user_query
---
```

- `name` — human-readable identifier
- `version` — semantic version, bumped on behavioural changes
- `models` — target models for cost projection and evaluation
- `variables` — every `{{ variable }}` used in the body must be listed

## Testing

- Run the Go test suite:

  ```bash
  go test ./...
  ```

- Add or extend test matrices in `tests/eval_suite.json` when you change eval behaviour.

## Style Guide

- Format Go code with `gofmt` and check with `go vet ./...`
- Keep the CLI output stable and `diff`-friendly — a change that reorders output is a breaking change
- Use conventional commit-style messages (`feat:`, `fix:`, `docs:`, `test:`)

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).