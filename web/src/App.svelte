<script>
  import { onMount } from 'svelte'

  let promptText = `---
name: Customer Support Classifier
version: 2.0.0
models:
  - gpt-4o
  - claude-3-5-sonnet
variables:
  - user_context
---
You are an expert customer support agent.
Analyze the user query: {{ user_context }}

Return JSON with classification and confidence score.`
  let tokenCount = 0
  let leftVar = ''
  let rightVar = 'User has been with us for 3 years and is on the Premium plan.'
  let comparison = null
  let error = ''
  let runs = []
  let runsError = ''
  let compareA = null
  let compareB = null

  async function loadRuns() {
    try {
      const res = await fetch('/api/runs')
      runs = await res.json()
      runsError = ''
    } catch (e) {
      runsError = String(e)
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

  async function countTokens() {
    try {
      const res = await fetch('/api/tokenize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: promptText }),
      })
      const data = await res.json()
      tokenCount = data.tokens
    } catch (e) {
      error = String(e)
    }
  }

  async function compare() {
    try {
      const res = await fetch('/api/compare', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          left: leftVar,
          right: rightVar,
        }),
      })
      comparison = await res.json()
      error = ''
    } catch (e) {
      error = String(e)
    }
  }

  onMount(() => {
    countTokens()
    loadRuns()
  })
</script>

<main class="min-h-screen bg-zinc-100 p-6 dark:bg-zinc-900">
  <div class="mx-auto max-w-5xl space-y-6">
    <h1 class="text-2xl font-bold text-zinc-800 dark:text-zinc-100">prompt-diff workspace</h1>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">
      Tune variables, count tokens live, and compare side-by-side.
    </p>

    <section class="rounded-lg bg-white p-4 shadow dark:bg-zinc-800">
      <h2 class="mb-2 font-semibold text-zinc-700 dark:text-zinc-200">Prompt</h2>
      <textarea
        bind:value={promptText}
        on:input={countTokens}
        class="h-48 w-full rounded border border-zinc-300 bg-zinc-50 p-2 font-mono text-sm dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
      ></textarea>
      <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
        Live token count: <strong>{tokenCount}</strong>
      </p>
    </section>

    <section class="rounded-lg bg-white p-4 shadow dark:bg-zinc-800">
      <h2 class="mb-2 font-semibold text-zinc-700 dark:text-zinc-200">Side-by-side comparison</h2>
      <div class="grid grid-cols-2 gap-4">
        <label class="text-sm text-zinc-600 dark:text-zinc-300">
          Left variable value
          <textarea bind:value={leftVar} class="mt-1 h-24 w-full rounded border border-zinc-300 p-2 font-mono text-sm dark:border-zinc-700 dark:bg-zinc-900"></textarea>
        </label>
        <label class="text-sm text-zinc-600 dark:text-zinc-300">
          Right variable value
          <textarea bind:value={rightVar} class="mt-1 h-24 w-full rounded border border-zinc-300 p-2 font-mono text-sm dark:border-zinc-700 dark:bg-zinc-900"></textarea>
        </label>
      </div>
      <button
        on:click={compare}
        class="mt-3 rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
      >
        Compare
      </button>
      {#if comparison}
        <ul class="mt-3 space-y-1 text-sm text-zinc-600 dark:text-zinc-300">
          {#each comparison as row}
            <li>{row.side}: <strong>{row.tokens}</strong> tokens</li>
          {/each}
        </ul>
      {/if}
    </section>

    {#if error}<p class="text-sm text-red-600">{error}</p>{/if}

    <section class="rounded-lg bg-white p-4 shadow dark:bg-zinc-800">
      <h2 class="mb-2 font-semibold text-zinc-700 dark:text-zinc-200">Eval run history</h2>
      <p class="mb-2 text-sm text-zinc-500 dark:text-zinc-400">
        Click two rows to compare them.
      </p>
      {#if runs.length === 0}
        <p class="text-sm text-zinc-500 dark:text-zinc-400">No eval runs stored yet. Run <code>prompt-diff eval</code> first.</p>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="text-zinc-500 dark:text-zinc-400">
                <th class="py-1 pr-3">ID</th>
                <th class="py-1 pr-3">Created</th>
                <th class="py-1 pr-3">Prompt</th>
                <th class="py-1 pr-3">Suite</th>
                <th class="py-1 pr-3">Models</th>
                <th class="py-1 pr-3">Pass</th>
                <th class="py-1 pr-3">Fail</th>
                <th class="py-1 pr-3">Skip</th>
                <th class="py-1 pr-3">Total</th>
                <th class="py-1 pr-3">Rate</th>
              </tr>
            </thead>
            <tbody>
              {#each runs as run}
                <tr
                  class="cursor-pointer border-t border-zinc-100 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-700/50"
                  class:bg-indigo-50={compareA === run.ID || compareB === run.ID}
                  class:dark:bg-indigo-900={compareA === run.ID || compareB === run.ID}
                  on:click={() => toggleCompare(run.ID)}
                >
                  <td class="py-1 pr-3">{run.ID}</td>
                  <td class="py-1 pr-3">{run.CreatedAt}</td>
                  <td class="py-1 pr-3">{run.PromptPath}</td>
                  <td class="py-1 pr-3">{run.Suite}</td>
                  <td class="py-1 pr-3">{run.Models}</td>
                  <td class="py-1 pr-3">{run.Passed}</td>
                  <td class="py-1 pr-3">{run.Failed}</td>
                  <td class="py-1 pr-3">{run.Skipped}</td>
                  <td class="py-1 pr-3">{run.Total}</td>
                  <td class="py-1 pr-3">{passRate(run)}%</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        {#if runDelta}
          <div class="mt-4 rounded border border-indigo-200 bg-indigo-50 p-3 text-sm dark:border-indigo-800 dark:bg-indigo-900/30">
            <p class="mb-1 font-semibold text-zinc-700 dark:text-zinc-200">
              Delta (run {runDelta.a.ID} &rarr; {runDelta.b.ID})
            </p>
            <p class="text-zinc-600 dark:text-zinc-300">Pass rate: {runDelta.passRateDelta > 0 ? '+' : ''}{runDelta.passRateDelta}%</p>
            <p class="text-zinc-600 dark:text-zinc-300">Passed: {runDelta.passedDelta > 0 ? '+' : ''}{runDelta.passedDelta}</p>
            <p class="text-zinc-600 dark:text-zinc-300">Failed: {runDelta.failedDelta > 0 ? '+' : ''}{runDelta.failedDelta}</p>
            <p class="text-zinc-600 dark:text-zinc-300">Total: {runDelta.totalDelta > 0 ? '+' : ''}{runDelta.totalDelta}</p>
          </div>
        {/if}
      {/if}
      {#if runsError}<p class="mt-2 text-sm text-red-600">{runsError}</p>{/if}
    </section>
  </div>
</main>