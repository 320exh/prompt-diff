<script>
  export let tabs = []
  export let active = ''
  export let onSelect = () => {}
</script>

<nav class="sidebar">
  <div class="brand">
    <div class="brand-mark">pd</div>
    <div>
      <div class="brand-name">prompt-diff</div>
      <div class="brand-sub">workspace</div>
    </div>
  </div>

  <div class="tab-list">
    {#each tabs as tab}
      <button
        class="tab-btn"
        class:active={active === tab.id}
        on:click={() => onSelect(tab.id)}
      >
        <span class="tab-icon">{tab.icon}</span>
        <span>{tab.label}</span>
      </button>
    {/each}
  </div>

  <div class="sidebar-footer">
    <span class="pd-label">CLI parity</span>
    <p>Every capability of the <code>prompt-diff</code> CLI is available here — nothing is CLI-only.</p>
  </div>
</nav>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    width: 220px;
    flex-shrink: 0;
    border-right: 1px solid var(--pd-border);
    background: var(--pd-bg-elevated);
    height: 100vh;
    position: sticky;
    top: 0;
    padding: 1.1rem 0.75rem;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.25rem 0.5rem 1.1rem;
  }
  .brand-mark {
    width: 30px;
    height: 30px;
    border-radius: 8px;
    background: linear-gradient(135deg, var(--pd-accent), #b892ff);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 0.75rem;
    font-family: var(--pd-font-mono);
  }
  .brand-name { font-weight: 700; font-size: 0.9rem; letter-spacing: -0.01em; }
  .brand-sub { font-size: 0.72rem; color: var(--pd-text-faint); }

  .tab-list { display: flex; flex-direction: column; gap: 0.15rem; }

  .tab-btn {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    text-align: left;
    padding: 0.5rem 0.6rem;
    border-radius: 8px;
    border: none;
    background: transparent;
    color: var(--pd-text-muted);
    font-size: 0.83rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 100ms ease, color 100ms ease;
  }
  .tab-btn:hover { background: var(--pd-bg-inset); color: var(--pd-text); }
  .tab-btn.active {
    background: var(--pd-accent-soft);
    color: var(--pd-accent-text);
    font-weight: 600;
  }
  .tab-icon { font-size: 0.95rem; width: 1.1rem; text-align: center; }

  .sidebar-footer {
    margin-top: auto;
    padding: 0.9rem 0.6rem 0.3rem;
    border-top: 1px solid var(--pd-border);
  }
  .sidebar-footer p {
    font-size: 0.72rem;
    color: var(--pd-text-faint);
    line-height: 1.4;
    margin-top: 0.3rem;
  }
  .sidebar-footer code {
    font-family: var(--pd-font-mono);
    background: var(--pd-bg-inset);
    padding: 0.05rem 0.3rem;
    border-radius: 4px;
  }

  @media (max-width: 860px) {
    .sidebar {
      position: static;
      width: 100%;
      height: auto;
      flex-direction: row;
      overflow-x: auto;
      align-items: center;
    }
    .brand { padding-bottom: 0; }
    .tab-list { flex-direction: row; }
    .sidebar-footer { display: none; }
  }
</style>
