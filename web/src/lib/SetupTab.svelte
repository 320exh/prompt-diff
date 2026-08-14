<script>
  import { postJSON } from './api.js'

  let initForce = false
  let initResult = null
  let initError = ''
  let initLoading = false

  let hookMaxDelta = 0
  let hookResult = null
  let hookError = ''
  let hookLoading = false

  async function runInit() {
    initLoading = true
    initError = ''
    try {
      initResult = await postJSON('/api/init', { force: initForce })
    } catch (e) {
      initError = String(e.message || e)
      initResult = null
    } finally {
      initLoading = false
    }
  }

  async function runHook() {
    hookLoading = true
    hookError = ''
    try {
      hookResult = await postJSON('/api/hook', { max_delta: Number(hookMaxDelta) || 0 })
    } catch (e) {
      hookError = String(e.message || e)
      hookResult = null
    } finally {
      hookLoading = false
    }
  }
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Setup</h1>
    <p>Scaffold a new project and install the git pre-commit hook — both operate on the directory the server was launched from.</p>
  </header>

  <section class="pd-card pad">
    <span class="pd-label">Init scaffold</span>
    <p class="hint">Writes <code>example.prompt</code>, <code>example.eval.json</code>, and <code>.prompt-diff.yml</code> into the current directory.</p>
    <label class="checkbox">
      <input type="checkbox" bind:checked={initForce} />
      <span>Overwrite existing files (--force)</span>
    </label>
    <button class="pd-btn pd-btn-primary" on:click={runInit} disabled={initLoading}>
      {initLoading ? 'Scaffolding…' : 'Run init'}
    </button>
    {#if initError}<p class="error-text">{initError}</p>{/if}
    {#if initResult}
      <div class="result-list">
        {#each initResult.written as f}<span class="pd-badge pd-badge-good">created {f}</span>{/each}
        {#each initResult.skipped as f}<span class="pd-badge pd-badge-neutral">skipped {f}</span>{/each}
      </div>
    {/if}
  </section>

  <section class="pd-card pad">
    <span class="pd-label">Pre-commit hook</span>
    <p class="hint">Installs a git pre-commit hook that prints the token/cost delta for staged <code>.prompt</code> files.</p>
    <label class="field">
      <span class="field-label">Max token delta (0 = never block the commit)</span>
      <input class="pd-input" type="number" min="0" bind:value={hookMaxDelta} style="max-width: 12rem;" />
    </label>
    <button class="pd-btn pd-btn-primary" on:click={runHook} disabled={hookLoading}>
      {hookLoading ? 'Installing…' : 'Install hook'}
    </button>
    {#if hookError}<p class="error-text">{hookError}</p>{/if}
    {#if hookResult}
      <p class="success-text">Installed at <code>{hookResult.path}</code></p>
    {/if}
  </section>
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; max-width: 640px; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; display: flex; flex-direction: column; gap: 0.7rem; align-items: flex-start; }
  .hint { font-size: 0.82rem; color: var(--pd-text-muted); margin: 0; }
  .hint code { font-family: var(--pd-font-mono); background: var(--pd-bg-inset); padding: 0.05rem 0.3rem; border-radius: 4px; }
  .checkbox { display: flex; align-items: center; gap: 0.5rem; font-size: 0.82rem; color: var(--pd-text-muted); }
  .field { display: flex; flex-direction: column; gap: 0.35rem; width: 100%; }
  .field-label { font-size: 0.78rem; color: var(--pd-text-muted); }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; margin: 0; }
  .success-text { color: var(--pd-good); font-size: 0.85rem; margin: 0; }
  .success-text code { font-family: var(--pd-font-mono); }
  .result-list { display: flex; gap: 0.4rem; flex-wrap: wrap; }
</style>
