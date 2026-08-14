<script>
  import Sidebar from './lib/Sidebar.svelte'
  import WorkspaceTab from './lib/WorkspaceTab.svelte'
  import DiffTab from './lib/DiffTab.svelte'
  import LintTab from './lib/LintTab.svelte'
  import EvalTab from './lib/EvalTab.svelte'
  import CompressTab from './lib/CompressTab.svelte'
  import ConvertTab from './lib/ConvertTab.svelte'
  import RunsTab from './lib/RunsTab.svelte'
  import ModelsTab from './lib/ModelsTab.svelte'
  import SetupTab from './lib/SetupTab.svelte'

  const tabs = [
    { id: 'workspace', label: 'Workspace', icon: '✎' },
    { id: 'diff', label: 'Diff', icon: '⇄' },
    { id: 'lint', label: 'Lint', icon: '✓' },
    { id: 'eval', label: 'Eval', icon: '▣' },
    { id: 'compress', label: 'Compress', icon: '↓' },
    { id: 'convert', label: 'Convert', icon: '⇌' },
    { id: 'runs', label: 'Runs', icon: '☰' },
    { id: 'models', label: 'Models', icon: '$' },
    { id: 'setup', label: 'Setup', icon: '⚙' },
  ]

  let active = 'workspace'
</script>

<div class="app-shell">
  <Sidebar {tabs} {active} onSelect={(id) => (active = id)} />
  <main class="main-panel">
    {#if active === 'workspace'}<WorkspaceTab />
    {:else if active === 'diff'}<DiffTab />
    {:else if active === 'lint'}<LintTab />
    {:else if active === 'eval'}<EvalTab />
    {:else if active === 'compress'}<CompressTab />
    {:else if active === 'convert'}<ConvertTab />
    {:else if active === 'runs'}<RunsTab />
    {:else if active === 'models'}<ModelsTab />
    {:else if active === 'setup'}<SetupTab />
    {/if}
  </main>
</div>

<style>
  .app-shell {
    display: flex;
    min-height: 100vh;
  }
  .main-panel {
    flex: 1;
    padding: 1.75rem 2rem 3rem;
    max-width: 1100px;
  }
  @media (max-width: 860px) {
    .app-shell { flex-direction: column; }
    .main-panel { padding: 1.25rem; }
  }
</style>
