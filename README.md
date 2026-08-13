# prompt-diff 🔍⚡

> A fast, git-native CLI & local Web UI to version, diff tokens/costs, and benchmark LLM system prompts across local and cloud models.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

---

## 💡 Why `prompt-diff`?

Prompt engineering has shifted from basic trial-and-error to core software engineering. However, standard software tooling treats system prompts as plain text strings, ignoring critical factors like **token consumption**, **cost per 1k runs**, **variable binding**, and **behavioral model drift**.

`prompt-diff` treats system prompts as **first-class source code**. It bridges git version control with real-time token/cost diffing and automated multi-model test harness evaluations.

---

## ✨ Key Features

- ⚙️ **Git-Native Prompt Tracking**: Automatically parses `.prompt` templates, identifies template variable injections, and tracks structural evolution across commits.
- 📊 **Semantic & Token Diffing**: Visualizes added/deleted tokens in your terminal and calculates immediate cost impact projections per 1,000 invocations.
- 🧪 **Batch Evaluation Harness**: Runs test matrices across local (Ollama) and OpenAI models, with clean stubs for Anthropic and Gemini until v2.
- 🖥️ **Zero-Config Local Workspace**: Spin up an embedded local web dashboard (`prompt-diff ui`) to tweak variables and compare side-by-side model outputs in real time.
- 📦 **Single-Binary Zero Setup**: Compiles into a single self-contained binary with zero external runtime dependencies.

---

## 🛠️ Architecture Overview

| Component | Stack | Purpose |
| :--- | :--- | :--- |
| **CLI Engine** | Go / Cobra | Lightning-fast execution, sub-millisecond startup, native git tree parsing |
| **Dashboard UI** | Svelte + Tailwind (Vite) | Compiled to static assets and embedded into the binary via `embed.FS` for zero-install local visualization |
| **Data Layer** | SQLite (`modernc.org/sqlite`) | Pure-Go zero-CGO embedded store for local evaluations and runs |
| **Tokenizers** | Embedded `cl100k_base` BPE (`tiktoken` vocab) | Pure-Go byte-level BPE token counting without network calls |

---

## 🚀 Quick Start

### Installation

#### Using Homebrew (macOS / Linux)
```bash
brew install 320exh/tap/prompt-diff
```

#### Using Go
```bash
go install github.com/320exh/prompt-diff@latest
```

#### Pre-built Binaries
Download the latest pre-compiled binary for Linux, macOS, or Windows from the [Releases](https://github.com/320exh/prompt-diff/releases) page.

#### Build from source
```bash
git clone https://github.com/320exh/prompt-diff.git
cd prompt-diff

# build the Go binary (embeds the compiled dashboard UI)
npm --prefix web install && npm --prefix web run build
go build -o prompt-diff .
```

---

## 💻 CLI Usage & Commands

### 1. Inspect & Diff System Prompts

Compare your current prompt draft against `HEAD` or a specific branch/commit:

```bash
# Compare local working prompt against HEAD
prompt-diff diff system_prompt_example.prompt

# Compare against a specific commit or tag
prompt-diff diff prompts/agent_v2.prompt --v1=v1.2.0 --v2=HEAD
```

**Terminal Output Preview:**

```text
Prompt: prompts/agent_v2.prompt
Target Models: gpt-4o, claude-3-5-sonnet

Token Delta: +142 tokens (+18.4%)
Cost Projection (100k invocations):
  - gpt-4o:            $0.35 -> $0.41 (+$0.06)
  - claude-3-5-sonnet:  $0.42 -> $0.50 (+$0.08)

Structural Diffs:
  + Added section: [Output Constraints]
  ~ Modified variable: {{ user_context }} -> {{ augmented_user_profile }}
```

---

### 2. Run Test-Suite Benchmarks

Evaluate prompt changes across a test-case matrix before pushing to production:

```bash
# Run local test matrix using Ollama and Claude
prompt-diff eval --prompt prompts/agent_v2.prompt --tests tests/eval_suite.json \
  --models llama3.1:8b,claude-3-5-sonnet --output report.html

# Reviews this run's results later
prompt-diff runs
```

> The eval harness records every run in a local SQLite store (`runs.db`). See `prompt-diff runs` to list past runs.

---

### 3. Launch Local Web Workspace

Open an instant local playground in your browser:

```bash
prompt-diff ui --port 8080
```

> Opens `http://localhost:8080` with interactive variable tuning, live token counter, and side-by-side stream comparison.

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

## 🗺️ Roadmap

**v1 (implemented):**
- [x] Returns a single self-contained binary (Go + embedded Svelte dashboard)
- [x] On-device token counting via embedded BPE (no network at diff time)
- [x] Git-native diffing: `prompt-diff diff <file> [--v1=ref --v2=ref]`
- [x] SQLite-backed eval history (`prompt-diff runs`)
- [x] `prompt-diff ui` serves the zero-config local dashboard

**v2 (planned):**
- [ ] GitHub Action integration (`prompt-diff-action`) for PR regression checks
- [ ] Automated prompt optimization & auto-compressor node
- [ ] Direct Anthropic / Gemini API clients (currently clear errors)
- [ ] Export benchmark reports to markdown/PDF
- [ ] Native integration with LangChain & LlamaIndex template schemas

---

## 🤝 Contributing

Contributions are warmly welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on setting up a development environment and submitting pull requests.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for details.
