<script>
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  let content = DEFAULT_PROMPT
  let from = 'prompt'
  let to = 'langchain'
  let output = ''
  let error = ''
  let loading = false
  let copied = false

  const FORMATS = [
    { value: 'prompt', label: '.prompt' },
    { value: 'langchain', label: 'LangChain' },
    { value: 'llamaindex', label: 'LlamaIndex' },
  ]

  async function runConvert() {
    loading = true
    error = ''
    copied = false
    try {
      const data = await postJSON('/api/convert', { content, from, to })
      output = data.output
    } catch (e) {
      error = String(e.message || e)
      output = ''
    } finally {
      loading = false
    }
  }

  async function copyOutput() {
    if (!output) return
    await navigator.clipboard.writeText(output)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Convert</h1>
    <p>Convert between prompt-diff's <code>.prompt</code> format and the LangChain / LlamaIndex single-template JSON shape.</p>
  </header>

  <div class="grid-2">
    <section class="pd-card pad">
      <div class="row-between">
        <span class="pd-label">Input</span>
        <select class="pd-select format-select" bind:value={from}>
          {#each FORMATS as f}<option value={f.value}>{f.label}</option>{/each}
        </select>
      </div>
      <textarea bind:value={content} class="pd-textarea" style="height: 16rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    </section>
    <section class="pd-card pad">
      <div class="row-between">
        <span class="pd-label">Output</span>
        <div class="row-gap">
          <select class="pd-select format-select" bind:value={to}>
            {#each FORMATS as f}<option value={f.value}>{f.label}</option>{/each}
          </select>
          {#if output}<button class="pd-btn" on:click={copyOutput}>{copied ? 'Copied ✓' : 'Copy'}</button>{/if}
        </div>
      </div>
      <pre class="output-block">{output || '—'}</pre>
    </section>
  </div>

  <button class="pd-btn pd-btn-primary" on:click={runConvert} disabled={loading} style="align-self: flex-start;">
    {loading ? 'Converting…' : 'Convert'}
  </button>

  {#if error}<p class="error-text">{error}</p>{/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .tab-header code { font-family: var(--pd-font-mono); background: var(--pd-bg-inset); padding: 0.05rem 0.3rem; border-radius: 4px; }
  .pad { padding: 1rem 1.1rem; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.8rem; }
  .row-between { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
  .row-gap { display: flex; align-items: center; gap: 0.5rem; }
  .format-select { width: auto; padding: 0.3rem 0.5rem; font-size: 0.78rem; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .output-block {
    margin-top: 0.5rem;
    background: var(--pd-bg-inset);
    border: 1px solid var(--pd-border);
    border-radius: 8px;
    padding: 0.8rem;
    font-family: var(--pd-font-mono);
    font-size: 0.8rem;
    white-space: pre-wrap;
    word-break: break-word;
    height: 16rem;
    overflow-y: auto;
  }
  @media (max-width: 760px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
