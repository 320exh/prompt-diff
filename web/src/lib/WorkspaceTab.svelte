<script>
  import { onMount } from 'svelte'
  import { postJSON, DEFAULT_PROMPT } from './api.js'

  export let promptText = DEFAULT_PROMPT

  let tokenCount = 0
  let leftVar = ''
  let rightVar = 'User has been with us for 3 years and is on the Premium plan.'
  let comparison = null
  let error = ''

  async function countTokens() {
    try {
      const data = await postJSON('/api/tokenize', { text: promptText })
      tokenCount = data.tokens
      error = ''
    } catch (e) {
      error = String(e.message || e)
    }
  }

  async function compare() {
    try {
      comparison = await postJSON('/api/compare', { left: leftVar, right: rightVar })
      error = ''
    } catch (e) {
      error = String(e.message || e)
    }
  }

  onMount(countTokens)
</script>

<div class="stack pd-fade-in">
  <header class="tab-header">
    <h1>Workspace</h1>
    <p>Draft a prompt, watch the live token count, and try variable values side by side.</p>
  </header>

  <section class="pd-card pad">
    <div class="row-between">
      <span class="pd-label">Prompt source</span>
      <span class="pd-badge pd-badge-neutral">{tokenCount} tokens</span>
    </div>
    <textarea
      bind:value={promptText}
      on:input={countTokens}
      class="pd-textarea"
      style="height: 14rem; margin-top: 0.6rem;"
      spellcheck="false"
    ></textarea>
  </section>

  <section class="pd-card pad">
    <span class="pd-label">Side-by-side variable comparison</span>
    <div class="grid-2" style="margin-top: 0.6rem;">
      <label class="field">
        <span class="field-label">Left value</span>
        <textarea bind:value={leftVar} class="pd-textarea" style="height: 6rem;" spellcheck="false"></textarea>
      </label>
      <label class="field">
        <span class="field-label">Right value</span>
        <textarea bind:value={rightVar} class="pd-textarea" style="height: 6rem;" spellcheck="false"></textarea>
      </label>
    </div>
    <button class="pd-btn pd-btn-primary" style="margin-top: 0.7rem;" on:click={compare}>Compare tokens</button>
    {#if comparison}
      <div class="compare-result">
        {#each comparison as row}
          <span class="pd-badge pd-badge-neutral">{row.side}: <strong>&nbsp;{row.tokens}</strong></span>
        {/each}
      </div>
    {/if}
  </section>

  {#if error}<p class="error-text">{error}</p>{/if}
</div>

<style>
  .stack { display: flex; flex-direction: column; gap: 1rem; max-width: 860px; }
  .tab-header h1 { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em; }
  .tab-header p { color: var(--pd-text-muted); font-size: 0.85rem; margin-top: 0.15rem; }
  .pad { padding: 1rem 1.1rem; }
  .row-between { display: flex; align-items: center; justify-content: space-between; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.8rem; }
  .field { display: flex; flex-direction: column; gap: 0.35rem; }
  .field-label { font-size: 0.78rem; color: var(--pd-text-muted); }
  .compare-result { display: flex; gap: 0.5rem; margin-top: 0.7rem; }
  .error-text { color: var(--pd-bad); font-size: 0.85rem; }
  @media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
