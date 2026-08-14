<script>
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  let source = DEFAULT_PROMPT
  let findings = null
  let error = ''
  let loading = false

  async function runLint() {
    loading = true
    error = ''
    try {
      const data = await postJSON('/api/lint', { source })
      findings = data.findings
    } catch (e) {
      error = String(e.message || e)
      findings = null
    } finally {
      loading = false
    }
  }
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Lint</h1>
    <p>Static checks, zero API cost: undeclared/unused variables, missing frontmatter, unknown models, trailing whitespace.</p>
  </header>

  <section class="pd-card pad">
    <span class="pd-label">.prompt source</span>
    <textarea bind:value={source} class="pd-textarea" style="height: 14rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    <button class="pd-btn pd-btn-primary" style="margin-top: 0.7rem;" on:click={runLint} disabled={loading}>
      {loading ? 'Linting…' : 'Run lint'}
    </button>
  </section>

  {#if error}<p class="error-text">{error}</p>{/if}

  {#if findings}
    <section class="pd-card pad">
      {#if findings.length === 0}
        <span class="pd-badge pd-badge-good">✓ No findings</span>
      {:else}
        <span class="pd-badge pd-badge-warn">{findings.length} finding{findings.length === 1 ? '' : 's'}</span>
        <ul class="finding-list">
          {#each findings as f}<li>{f}</li>{/each}
        </ul>
      {/if}
    </section>
  {/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; max-width: 720px; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .finding-list { list-style: none; display: flex; flex-direction: column; gap: 0.4rem; margin-top: 0.7rem; font-size: 0.83rem; }
  .finding-list li { padding: 0.45rem 0.6rem; background: var(--pd-warn-soft); color: var(--pd-warn); border-radius: 6px; }
</style>
