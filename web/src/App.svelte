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

  onMount(countTokens)
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
  </div>
</main>