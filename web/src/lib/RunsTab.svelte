<script>
  import { onMount } from 'svelte'
  import { getJSON } from './api.js'

  let runs = []
  let error = ''
  let compareA = null
  let compareB = null

  async function loadRuns() {
    try {
      runs = await getJSON('/api/runs')
      error = ''
    } catch (e) {
      error = String(e.message || e)
    }
  }

  function passRate(run) {
    return run.Total > 0 ? ((run.Passed / run.Total) * 100).toFixed(1) : '0.0'
  }

  function toggleCompare(id) {
    if (compareA === id) { compareA = null; return }
    if (compareB === id) { compareB = null; return }
    if (compareA === null) { compareA = id; return }
    if (compareB === null) { compareB = id; return }
    compareA = id
    compareB = null
  }

  $: runDelta = (() => {
    if (compareA === null || compareB === null) return null
    const a = runs.find((r) => r.ID === compareA)
    const b = runs.find((r) => r.ID === compareB)
    if (!a || !b) return null
    return {
      a, b,
      passRateDelta: (Number(passRate(b)) - Number(passRate(a))).toFixed(1),
      passedDelta: b.Passed - a.Passed,
      failedDelta: b.Failed - a.Failed,
      totalDelta: b.Total - a.Total,
    }
  })()

  $: selectedIds = [compareA, compareB].filter((id) => id !== null)

  function exportUrl(format) {
    const ids = selectedIds.length ? selectedIds : runs.map((r) => r.ID)
    return `/api/runs/export?ids=${ids.join(',')}&format=${format}`
  }

  onMount(loadRuns)
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Runs</h1>
    <p>Stored eval-run history. Click two rows to compare, or export selected (or all) runs as a report.</p>
  </header>

  <section class="pd-card pad">
    {#if error}<p class="error-text">{error}</p>{/if}
    {#if runs.length === 0 && !error}
      <p class="empty-text">No eval runs stored yet. Run one from the Eval tab or <code>prompt-diff eval</code>.</p>
    {:else}
      <div class="row-between">
        <span class="pd-label">{runs.length} stored run{runs.length === 1 ? '' : 's'}</span>
        <div class="row-gap">
          <a class="pd-btn" href={exportUrl('md')} download>Export markdown</a>
          <a class="pd-btn" href={exportUrl('pdf')} download>Export PDF</a>
        </div>
      </div>
      <div class="table-scroll">
        <table class="pd-table">
          <thead>
            <tr>
              <th>ID</th><th>Created</th><th>Prompt</th><th>Suite</th><th>Models</th>
              <th>Pass</th><th>Fail</th><th>Skip</th><th>Total</th><th>Rate</th>
            </tr>
          </thead>
          <tbody>
            {#each runs as run}
              <tr
                class="run-row"
                class:selected={compareA === run.ID || compareB === run.ID}
                on:click={() => toggleCompare(run.ID)}
              >
                <td>{run.ID}</td>
                <td>{run.CreatedAt}</td>
                <td class="truncate">{run.PromptPath}</td>
                <td class="truncate">{run.Suite}</td>
                <td class="truncate">{run.Models}</td>
                <td>{run.Passed}</td>
                <td>{run.Failed}</td>
                <td>{run.Skipped}</td>
                <td>{run.Total}</td>
                <td>{passRate(run)}%</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if runDelta}
        <div class="delta-box">
          <span class="pd-label">Delta (run {runDelta.a.ID} → {runDelta.b.ID})</span>
          <div class="delta-grid">
            <span>Pass rate: <strong>{runDelta.passRateDelta > 0 ? '+' : ''}{runDelta.passRateDelta}%</strong></span>
            <span>Passed: <strong>{runDelta.passedDelta > 0 ? '+' : ''}{runDelta.passedDelta}</strong></span>
            <span>Failed: <strong>{runDelta.failedDelta > 0 ? '+' : ''}{runDelta.failedDelta}</strong></span>
            <span>Total: <strong>{runDelta.totalDelta > 0 ? '+' : ''}{runDelta.totalDelta}</strong></span>
          </div>
        </div>
      {/if}
    {/if}
  </section>
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .row-between { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.6rem; flex-wrap: wrap; gap: 0.5rem; }
  .row-gap { display: flex; gap: 0.5rem; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  .empty-text { color: var(--pd-text-muted); font-size: 0.85rem; }
  .empty-text code { font-family: var(--pd-font-mono); background: var(--pd-bg-inset); padding: 0.05rem 0.3rem; border-radius: 4px; }
  .table-scroll { overflow-x: auto; }
  .pd-table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  .pd-table th { text-align: left; color: var(--pd-text-faint); font-weight: 600; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.03em; padding: 0.35rem 0.6rem; border-bottom: 1px solid var(--pd-border); white-space: nowrap; }
  .pd-table td { padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--pd-border); white-space: nowrap; }
  .truncate { max-width: 12rem; overflow: hidden; text-overflow: ellipsis; }
  .run-row { cursor: pointer; transition: background 100ms ease; }
  .run-row:hover { background: var(--pd-bg-inset); }
  .run-row.selected { background: var(--pd-accent-soft); }
  .delta-box {
    margin-top: 1rem;
    padding: 0.8rem 1rem;
    border-radius: 8px;
    background: var(--pd-accent-soft);
    border: 1px solid var(--pd-border);
  }
  .delta-grid { display: flex; gap: 1.5rem; flex-wrap: wrap; margin-top: 0.4rem; font-size: 0.85rem; color: var(--pd-text-muted); }
  .delta-grid strong { color: var(--pd-text); }
</style>
