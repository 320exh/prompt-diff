<script>
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  let oldSrc = DEFAULT_PROMPT
  let newSrc = DEFAULT_PROMPT.replace('gpt-4o', 'gpt-4o-mini')
  let semantic = false
  let result = null
  let error = ''
  let loading = false

  async function runDiff() {
    loading = true
    error = ''
    try {
      result = await postJSON('/api/diff', { old: oldSrc, new: newSrc, semantic })
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
    <h1>Diff</h1>
    <p>Compare two prompt versions: token/cost delta, structural changes, and optional semantic similarity.</p>
  </header>

  <div class="grid-2">
    <section class="pd-card pad">
      <span class="pd-label">Old version</span>
      <textarea bind:value={oldSrc} class="pd-textarea" style="height: 12rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    </section>
    <section class="pd-card pad">
      <span class="pd-label">New version</span>
      <textarea bind:value={newSrc} class="pd-textarea" style="height: 12rem; margin-top: 0.5rem;" spellcheck="false"></textarea>
    </section>
  </div>

  <div class="row">
    <label class="checkbox">
      <input type="checkbox" bind:checked={semantic} />
      <span>Semantic similarity (Voyage AI, requires <code>VOYAGE_API_KEY</code>)</span>
    </label>
    <button class="pd-btn pd-btn-primary" on:click={runDiff} disabled={loading}>
      {loading ? 'Diffing…' : 'Run diff'}
    </button>
  </div>

  {#if error}<p class="error-text">{error}</p>{/if}

  {#if result}
    <section class="pd-card pad result">
      <div class="metric-row">
        <div class="metric">
          <span class="pd-label">Token delta</span>
          <span class="metric-value">{result.token_delta > 0 ? '+' : ''}{result.token_delta} <small>({result.token_percent > 0 ? '+' : ''}{result.token_percent.toFixed(1)}%)</small></span>
        </div>
        {#if result.semantic_similarity !== undefined}
          <div class="metric">
            <span class="pd-label">Semantic similarity</span>
            <span class="metric-value">{result.semantic_similarity.toFixed(3)}</span>
          </div>
        {/if}
      </div>

      {#if result.costs?.length}
        <div class="subsection">
          <span class="pd-label">Cost projection (100k invocations)</span>
          <table class="pd-table">
            <thead><tr><th>Model</th><th>Old</th><th>New</th><th>Delta</th></tr></thead>
            <tbody>
              {#each result.costs as c}
                <tr>
                  <td>{c.model}{c.approx ? ' *' : ''}</td>
                  <td>${c.old.toFixed(2)}</td>
                  <td>${c.new.toFixed(2)}</td>
                  <td class:pos={c.delta > 0} class:neg={c.delta < 0}>{c.delta > 0 ? '+' : ''}${c.delta.toFixed(2)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if result.added_sections?.length || result.removed_sections?.length || result.modified_vars?.length || result.added_vars?.length || result.removed_vars?.length}
        <div class="subsection">
          <span class="pd-label">Structural diff</span>
          <ul class="diff-list">
            {#each result.added_sections || [] as s}<li class="add">+ section: {s}</li>{/each}
            {#each result.removed_sections || [] as s}<li class="rem">- section: {s}</li>{/each}
            {#each result.modified_vars || [] as v}<li class="mod">~ {'{{'} {v.from} {'}}'} → {'{{'} {v.to} {'}}'}</li>{/each}
            {#each result.added_vars || [] as v}<li class="add">+ {'{{'} {v} {'}}'}</li>{/each}
            {#each result.removed_vars || [] as v}<li class="rem">- {'{{'} {v} {'}}'}</li>{/each}
          </ul>
        </div>
      {/if}
    </section>
  {/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.8rem; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
  .checkbox { display: flex; align-items: center; gap: 0.5rem; font-size: 0.82rem; color: var(--pd-text-muted); }
  .checkbox code { font-family: var(--pd-font-mono); background: var(--pd-bg-inset); padding: 0.05rem 0.3rem; border-radius: 4px; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .metric-row { display: flex; gap: 2rem; flex-wrap: wrap; }
  .metric { display: flex; flex-direction: column; gap: 0.2rem; }
  .metric-value { font-size: 1.3rem; font-weight: 700; font-family: var(--pd-font-mono); }
  .metric-value small { font-size: 0.8rem; font-weight: 500; color: var(--pd-text-muted); }
  .subsection { margin-top: 1.1rem; }
  .pd-table { width: 100%; border-collapse: collapse; font-size: 0.82rem; margin-top: 0.5rem; }
  .pd-table th { text-align: left; color: var(--pd-text-faint); font-weight: 600; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.03em; padding: 0.3rem 0.5rem; border-bottom: 1px solid var(--pd-border); }
  .pd-table td { padding: 0.4rem 0.5rem; border-bottom: 1px solid var(--pd-border); font-family: var(--pd-font-mono); }
  .pd-table td.pos { color: var(--pd-bad); }
  .pd-table td.neg { color: var(--pd-good); }
  .diff-list { list-style: none; display: flex; flex-direction: column; gap: 0.3rem; margin-top: 0.5rem; font-family: var(--pd-font-mono); font-size: 0.82rem; }
  .diff-list .add { color: var(--pd-good); }
  .diff-list .rem { color: var(--pd-bad); }
  .diff-list .mod { color: var(--pd-warn); }
  @media (max-width: 720px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
