<script>
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  const DEFAULT_SUITE = `{
  "name": "example regression suite",
  "cases": [
    {
      "id": "refund-request",
      "input": "I was charged twice this month and want a refund.",
      "expect": { "classification": { "$in": ["refund", "billing"] } }
    }
  ]
}`

  let promptSrc = DEFAULT_PROMPT
  let suiteSrc = DEFAULT_SUITE
  let modelsCsv = ''
  let concurrency = 5
  let result = null
  let error = ''
  let loading = false

  async function runEval() {
    loading = true
    error = ''
    try {
      const models = modelsCsv.split(',').map((m) => m.trim()).filter(Boolean)
      result = await postJSON('/api/eval', { prompt: promptSrc, suite: suiteSrc, models, concurrency })
    } catch (e) {
      error = String(e.message || e)
      result = null
    } finally {
      loading = false
    }
  }
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Eval</h1>
    <p>Run a test-case matrix across models. Results are stored in the same local SQLite history as the CLI.</p>
  </header>

  <div class="grid-2">
    <section class="pd-card pad">
      <span class="pd-label">Prompt</span>
      <textarea bind:value={promptSrc} class="pd-textarea" style="height: 11rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    </section>
    <section class="pd-card pad">
      <span class="pd-label">Eval suite (JSON)</span>
      <textarea bind:value={suiteSrc} class="pd-textarea" style="height: 11rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    </section>
  </div>

  <section class="pd-card pad">
    <div class="grid-3">
      <label class="field">
        <span class="field-label">Models (comma-separated, default: frontmatter)</span>
        <input class="pd-input" bind:value={modelsCsv} placeholder="gpt-4o-mini, claude-3-5-sonnet" />
      </label>
      <label class="field">
        <span class="field-label">Concurrency</span>
        <input class="pd-input" type="number" min="1" bind:value={concurrency} />
      </label>
      <div class="field field-btn">
        <button class="pd-btn pd-btn-primary" on:click={runEval} disabled={loading}>
          {loading ? 'Running…' : 'Run eval'}
        </button>
      </div>
    </div>
  </section>

  {#if error}<p class="error-text">{error}</p>{/if}

  {#if result}
    <section class="pd-card pad">
      <div class="metric-row">
        <span class="pd-badge pd-badge-good">{result.passed} passed</span>
        <span class="pd-badge pd-badge-bad">{result.failed} failed</span>
        <span class="pd-badge pd-badge-neutral">{result.skipped} skipped</span>
        <span class="pd-badge pd-badge-neutral">{result.total} total</span>
        {#if result.run_id}<span class="pd-badge pd-badge-neutral">stored as run #{result.run_id}</span>{/if}
      </div>
      <table class="pd-table">
        <thead><tr><th>Model</th><th>Case</th><th>Result</th><th>Output</th></tr></thead>
        <tbody>
          {#each result.cells as c}
            <tr>
              <td>{c.model}</td>
              <td>{c.case_id}</td>
              <td>
                {#if c.passed}<span class="pd-badge pd-badge-good">pass</span>
                {:else if c.failed}<span class="pd-badge pd-badge-bad">fail</span>
                {:else}<span class="pd-badge pd-badge-warn">skip</span>{/if}
              </td>
              <td class="output-cell">{c.error || c.output}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.8rem; }
  .grid-3 { display: grid; grid-template-columns: 2fr 1fr auto; gap: 0.8rem; align-items: end; }
  .field { display: flex; flex-direction: column; gap: 0.35rem; }
  .field-label { font-size: 0.78rem; color: var(--pd-text-muted); }
  .field-btn { justify-content: flex-end; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .metric-row { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 0.7rem; }
  .pd-table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  .pd-table th { text-align: left; color: var(--pd-text-faint); font-weight: 600; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.03em; padding: 0.3rem 0.5rem; border-bottom: 1px solid var(--pd-border); }
  .pd-table td { padding: 0.4rem 0.5rem; border-bottom: 1px solid var(--pd-border); vertical-align: top; }
  .output-cell { font-family: var(--pd-font-mono); font-size: 0.76rem; max-width: 32rem; white-space: pre-wrap; word-break: break-word; color: var(--pd-text-muted); }
  @media (max-width: 720px) { .grid-2, .grid-3 { grid-template-columns: 1fr; } }
</style>
