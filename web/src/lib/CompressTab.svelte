<script>
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  let promptSrc = DEFAULT_PROMPT
  let model = ''
  let suiteSrc = ''
  let modelsCsv = ''
  let maxLoss = 0
  let result = null
  let error = ''
  let loading = false
  let copied = false

  async function runCompress() {
    loading = true
    error = ''
    copied = false
    try {
      const models = modelsCsv.split(',').map((m) => m.trim()).filter(Boolean)
      result = await postJSON('/api/compress', {
        prompt: promptSrc, model, suite: suiteSrc, models, max_loss: Number(maxLoss) || 0,
      })
    } catch (e) {
      error = String(e.message || e)
      result = null
    } finally {
      loading = false
    }
  }

  async function copyRendered() {
    if (!result?.rendered) return
    await navigator.clipboard.writeText(result.rendered)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Compress</h1>
    <p>Rewrite a prompt to use fewer tokens via an LLM. Optionally guard against pass-rate regressions with an eval suite.</p>
  </header>

  <section class="pd-card pad">
    <span class="pd-label">Prompt</span>
    <textarea bind:value={promptSrc} class="pd-textarea" style="height: 11rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
  </section>

  <section class="pd-card pad">
    <div class="grid-3">
      <label class="field">
        <span class="field-label">Model (required)</span>
        <input class="pd-input" bind:value={model} placeholder="claude-3-5-sonnet" />
      </label>
      <label class="field">
        <span class="field-label">Guard models (default: frontmatter)</span>
        <input class="pd-input" bind:value={modelsCsv} placeholder="gpt-4o-mini" />
      </label>
      <label class="field">
        <span class="field-label">Max pass-rate loss (pts)</span>
        <input class="pd-input" type="number" min="0" bind:value={maxLoss} />
      </label>
    </div>
    <label class="field" style="margin-top: 0.7rem;">
      <span class="field-label">Eval suite JSON (optional — guards the rewrite)</span>
      <textarea bind:value={suiteSrc} class="pd-textarea" style="height: 6rem;" spellcheck="false" placeholder="paste an eval suite to guard the rewrite"></textarea>
    </label>
    <button class="pd-btn pd-btn-primary" style="margin-top: 0.7rem;" on:click={runCompress} disabled={loading || !model}>
      {loading ? 'Compressing…' : 'Compress'}
    </button>
  </section>

  {#if error}<p class="error-text">{error}</p>{/if}

  {#if result}
    <section class="pd-card pad">
      <div class="metric-row">
        <div class="metric">
          <span class="pd-label">Tokens</span>
          <span class="metric-value">{result.tokens_before} → {result.tokens_after}</span>
        </div>
        {#if result.pass_rate_before !== undefined}
          <div class="metric">
            <span class="pd-label">Pass rate</span>
            <span class="metric-value">{result.pass_rate_before.toFixed(1)}% → {result.pass_rate_after.toFixed(1)}%</span>
          </div>
        {/if}
      </div>
      {#if result.rejected}
        <p class="error-text" style="margin-top: 0.7rem;">Rejected: {result.rejected}</p>
      {/if}
      <div class="row-between" style="margin-top: 0.9rem;">
        <span class="pd-label">Compressed body</span>
        {#if result.rendered}
          <button class="pd-btn" on:click={copyRendered}>{copied ? 'Copied ✓' : 'Copy .prompt file'}</button>
        {/if}
      </div>
      <pre class="output-block">{result.body}</pre>
    </section>
  {/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .grid-3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 0.8rem; }
  .field { display: flex; flex-direction: column; gap: 0.35rem; }
  .field-label { font-size: 0.78rem; color: var(--pd-text-muted); }
  .row-between { display: flex; align-items: center; justify-content: space-between; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .metric-row { display: flex; gap: 2rem; flex-wrap: wrap; }
  .metric { display: flex; flex-direction: column; gap: 0.2rem; }
  .metric-value { font-size: 1.15rem; font-weight: 700; font-family: var(--pd-font-mono); }
  .output-block {
    margin-top: 0.5rem;
    background: var(--pd-bg-inset);
    border: 1px solid var(--pd-border);
    border-radius: 8px;
    padding: 0.8rem;
    font-family: var(--pd-font-mono);
    font-size: 0.82rem;
    white-space: pre-wrap;
    word-break: break-word;
  }
  @media (max-width: 760px) { .grid-3 { grid-template-columns: 1fr; } }
</style>
