<script>
  import { onMount } from 'svelte'
  import { getJSON } from './api.js'

  let info = null
  let error = ''

  async function load() {
    try {
      info = await getJSON('/api/models')
      error = ''
    } catch (e) {
      error = String(e.message || e)
    }
  }

  onMount(load)
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Models</h1>
    <p>Freshness of the built-in cost table used for token-cost projections.</p>
  </header>

  <section class="pd-card pad">
    {#if error}<p class="error-text">{error}</p>{/if}
    {#if info}
      <div class="metric-row">
        <div class="metric">
          <span class="pd-label">Prices as of</span>
          <span class="metric-value">{info.prices_as_of}</span>
        </div>
        <div class="metric">
          <span class="pd-label">Age</span>
          <span class="metric-value">{info.age_days} day{info.age_days === 1 ? '' : 's'}</span>
        </div>
        <div class="metric">
          <span class="pd-label">Status</span>
          {#if info.stale}
            <span class="pd-badge pd-badge-warn">stale (&gt; {info.stale_after_days}d)</span>
          {:else}
            <span class="pd-badge pd-badge-good">fresh</span>
          {/if}
        </div>
      </div>
      {#if info.stale}
        <p class="warn-text">Price table is older than {info.stale_after_days} days; costs may be inaccurate. Check for a newer prompt-diff release.</p>
      {/if}
    {/if}
  </section>

  <section class="pd-card pad">
    <h2 class="table-title">Cost table</h2>
    {#if info}
      <table class="pd-table">
        <thead>
          <tr><th>Model</th><th>$ / 1M tokens</th><th></th></tr>
        </thead>
        <tbody>
          {#each info.models as m}
            <tr>
              <td class="mono">{m.model}</td>
              <td class="mono">{m.price_per_1m.toFixed(2)}</td>
              <td>{#if m.overridden}<span class="pd-badge pd-badge-neutral">override</span>{/if}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; max-width: 760px; }
  .table-title { font-size: 0.95rem; font-weight: 700; margin-bottom: 0.6rem; }
  .pd-table { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .pd-table th { text-align: left; color: var(--pd-text-faint); font-weight: 600; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.03em; padding: 0.3rem 0.5rem; border-bottom: 1px solid var(--pd-border); }
  .pd-table td { padding: 0.35rem 0.5rem; border-bottom: 1px solid var(--pd-border); }
  .mono { font-family: var(--pd-font-mono); }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .metric-row { display: flex; gap: 2rem; flex-wrap: wrap; }
  .metric { display: flex; flex-direction: column; gap: 0.3rem; }
  .metric-value { font-size: 1.15rem; font-weight: 700; font-family: var(--pd-font-mono); }
  .warn-text { margin-top: 0.8rem; font-size: 0.83rem; color: var(--pd-warn); }
</style>
