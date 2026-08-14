# prompt-diff 🔍⚡

> A fast, git-native CLI & local Web UI to version, diff tokens/costs, and benchmark LLM system prompts across local and cloud models.

[![CI](https://github.com/320exh/prompt-diff/actions/workflows/ci.yml/badge.svg)](https://github.com/320exh/prompt-diff/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Homebrew tap](https://img.shields.io/badge/homebrew-320exh%2Fprompt--diff-orange.svg)](https://github.com/320exh/homebrew-prompt-diff)

---

## 💡 Why `prompt-diff`?

Prompt engineering has shifted from basic trial-and-error to core software engineering. However, standard software tooling treats system prompts as plain text strings, ignoring critical factors like **token consumption**, **cost per 1k runs**, **variable binding**, and **behavioral model drift**.

`prompt-diff` treats system prompts as **first-class source code**. It bridges git version control with real-time token/cost diffing and automated multi-model test harness evaluations.

---

## ✨ Key Features

- ⚙️ **Git-Native Prompt Tracking**: Automatically parses `.prompt` templates, identifies template variable injections, and tracks structural evolution across commits.
- 📊 **Semantic & Token Diffing**: Visualizes added/deleted tokens in your terminal and calculates immediate cost impact projections per 1,000 invocations.
- 🧪 **Batch Evaluation Harness**: Runs test matrices across local (Ollama), OpenAI, Anthropic, and Gemini models, with `contains`/list/`$in`/`$gte`/`$lte`/`$regex`/`$schema`/`$llm_judge` assertion operators.
- 🖥️ **Zero-Config Local Workspace**: Spin up an embedded local web dashboard (`prompt-diff ui`) to tweak variables and compare side-by-side model outputs in real time. Running the binary with no arguments (or double-clicking it on Windows) launches the dashboard directly.
- 📦 **Single-Binary Zero Setup**: Compiles into a single self-contained binary with zero external runtime dependencies.

---

## 🛠️ Architecture Overview

| Component | Stack | Purpose |
| :--- | :--- | :--- |
| **CLI Engine** | Go / Cobra | Lightning-fast execution, sub-millisecond startup, native git tree parsing |
| **Dashboard UI** | Svelte + Tailwind (Vite) | Compiled to static assets and embedded into the binary via `embed.FS` for zero-install local visualization |
| **Data Layer** | SQLite (`modernc.org/sqlite`) | Pure-Go zero-CGO embedded store for local evaluations and runs |
| **Tokenizers** | Embedded `cl100k_base` + `o200k_base` BPE (`tiktoken` vocabs), auto-selected per model | Pure-Go byte-level BPE token counting without network calls |

---

## 🚀 Quick Start

### Installation

#### Using Homebrew (macOS & Linux)
Prebuilt binaries via the [`320exh/homebrew-prompt-diff`](https://github.com/320exh/homebrew-prompt-diff) tap:
```bash
brew install 320exh/prompt-diff/prompt-diff
```

#### Pre-built binary (Linux, without Homebrew)
Download `prompt-diff-linux-amd64` or `prompt-diff-linux-arm64` from the [latest release](https://github.com/320exh/prompt-diff/releases/latest), then:
```bash
chmod +x prompt-diff-linux-* && sudo mv prompt-diff-linux-* /usr/local/bin/prompt-diff
```

#### Pre-built binary (Windows)
Download `prompt-diff-windows-amd64.exe` or `prompt-diff-windows-arm64.exe` from the [latest release](https://github.com/320exh/prompt-diff/releases/latest), then add it to your `PATH` (or run it in place). Double-clicking the `.exe` also works — it boots the dashboard directly instead of printing CLI usage.

#### Using Go (any platform)
```bash
go install github.com/320exh/prompt-diff@latest
```

#### Build from source
```bash
git clone https://github.com/320exh/prompt-diff.git
cd prompt-diff

# build the Go binary (embeds the compiled dashboard UI)
make build
# equivalent to:
#   npm --prefix web install && npm --prefix web run build
#   go build -o prompt-diff .
```

---

## 💻 CLI Usage & Commands

### 0. Scaffold a New Project

Starting from scratch? `init` drops a starter `.prompt` file, eval suite, and config so the rest of the commands below have something to run against immediately:

```bash
prompt-diff init
# created example.prompt
# created example.eval.json
# created .prompt-diff.yml

prompt-diff lint example.prompt
```

Existing files are left untouched unless you pass `--force`.

---

### 1. Inspect & Diff System Prompts

Compare your current prompt draft against `HEAD` or a specific branch/commit:

```bash
# Compare local working prompt against HEAD
prompt-diff diff system_prompt_example.prompt

# Compare against a specific commit or tag
prompt-diff diff prompts/agent_v2.prompt --v1=v1.2.0 --v2=HEAD

# Machine-readable output, e.g. for a CI job posting a PR comment
prompt-diff diff prompts/agent_v2.prompt --json

# Semantic similarity via Voyage AI embeddings (needs VOYAGE_API_KEY)
prompt-diff diff prompts/agent_v2.prompt --semantic
```

**Terminal Output Preview:**

```text
Prompt: prompts/agent_v2.prompt
Target Models: gpt-4o, claude-3-5-sonnet

Semantic Similarity: 0.947 (cosine, Voyage AI embeddings)
Token Delta: +142 tokens (+18.4%)
Cost Projection (100k invocations):
  - gpt-4o:            $0.35 -> $0.41 (+$0.06)
      with prompt caching: $0.18 -> $0.21 (+$0.03)
  - claude-3-5-sonnet:  $0.42 -> $0.50 (+$0.08)
      with prompt caching: $0.04 -> $0.05 (+$0.01)
    (caching assumes the whole prompt is cached: 1st call pays the cache-write price, the other 99,999 pay the cache-read price; models without a modeled discount are omitted)

Structural Diffs:
  + Added section: [Output Constraints]
  ~ Modified variable: {{ user_context }} -> {{ augmented_user_profile }}
```

---

### 2. Run Test-Suite Benchmarks

Evaluate prompt changes across a test-case matrix before pushing to production:

```bash
# Run local test matrix using Ollama and OpenAI
prompt-diff eval --prompt prompts/agent_v2.prompt --tests tests/eval_suite.json \
  --models llama3.1:8b,gpt-4o-mini --output report.html

# Reviews this run's results later
prompt-diff runs
```

> The eval harness records every run in a local SQLite store (`runs.db`). See `prompt-diff runs` to list past runs.
> In-flight provider calls are capped at 5 by default (`--concurrency N` to change) to avoid tripping rate limits on large suites.

---

### 3. Launch Local Web Workspace

Open an instant local playground in your browser:

```bash
prompt-diff ui --port 8080
```

> Opens `http://localhost:8080` with interactive variable tuning, live token counter, and side-by-side stream comparison.

---

### 4. Configuration

Drop a `.prompt-diff.yml` in the repo root (or `~/.prompt-diff.yml` for a global default) to override built-in per-model pricing — useful for negotiated rates or non-USD projections:

```yaml
price_overrides:
  gpt-4o: 2.10          # USD per 1M input tokens
  claude-sonnet-4-5: 2.75
```

---

## 📁 `.prompt` File Format

`prompt-diff` supports standard frontmatter metadata for declaring variables, model targets, and system rules:

```yaml
---
name: Customer Support Classifier
version: 2.1.0
models:
  - gpt-4o-mini
  - llama3.1:8b
variables:
  - user_query
  - historical_context
---
You are an expert customer support agent.
Analyze the user query: {{ user_query }}

Context history:
{{ historical_context }}

Return JSON with classification and confidence score.
```

---

## 🤝 Contributing

Contributions are warmly welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on setting up a development environment and submitting pull requests.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 🔒 Security

Provider API keys are read from environment variables only, never persisted. See [SECURITY.md](SECURITY.md) for details and how to report vulnerabilities.

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for details.
