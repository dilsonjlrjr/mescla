<script lang="ts">
  import Icon from './Icon.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix';

  interface Props {
    currentView: View;
    collapsed: boolean;
    onNavigate: (view: View) => void;
    onToggle: () => void;
  }

  let { currentView, collapsed, onNavigate, onToggle }: Props = $props();

  const navItems: { id: View; label: string; icon: 'home' | 'grid' | 'building' | 'pipette' | 'swap' | 'flask' }[] = [
    { id: 'home', label: 'Início', icon: 'home' },
    { id: 'catalog', label: 'Catálogo', icon: 'grid' },
    { id: 'manufacturers', label: 'Fabricantes', icon: 'building' },
    { id: 'color-search', label: 'Buscar Cor', icon: 'pipette' },
    { id: 'compare', label: 'Comparar', icon: 'swap' },
    { id: 'mix', label: 'Mistura', icon: 'flask' },
  ];
</script>

<aside class="sidebar" class:collapsed>
  <div class="titlebar-spacer" style="--wails-draggable: drag;"></div>

  <div class="sidebar-logo">
    <div class="logo-mark"></div>
    {#if !collapsed}
      <div>
        <div class="logo-word">Paint Match</div>
        <div class="logo-tag">AI</div>
      </div>
    {/if}
  </div>

  <nav class="sidebar-nav">
    {#each navItems as item}
      {@const isActive = currentView === item.id}
      <button
        class="nav-item"
        class:active={isActive}
        onclick={() => onNavigate(item.id)}
        title={collapsed ? item.label : undefined}
      >
        <span class="nav-icon"><Icon name={item.icon} size={19} /></span>
        {#if !collapsed}
          <span class="nav-label">{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  {#if !collapsed}
    <div class="sidebar-version">v1.0.0</div>
  {/if}

  <button class="sidebar-toggle" onclick={onToggle} aria-label={collapsed ? 'Expandir menu' : 'Recolher menu'}>
    <span class="toggle-icon" class:flipped={collapsed}><Icon name="chevron-left" size={16} /></span>
  </button>
</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    width: 224px;
    flex-shrink: 0;
    height: 100%;
    background: var(--ink-900);
    border-right: 1px solid var(--ink-700);
    transition: width 0.22s ease;
  }

  .sidebar.collapsed {
    width: 76px;
  }

  .titlebar-spacer {
    height: 30px;
    flex-shrink: 0;
  }

  .sidebar-logo {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 4px 16px 18px;
    border-bottom: 1px solid var(--ink-700);
  }

  .logo-mark {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--lacquer);
    box-shadow: inset 0 -3px 5px rgba(0, 0, 0, 0.25), inset 0 2px 3px rgba(255, 255, 255, 0.35);
  }

  .logo-word {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 600;
    color: var(--paper);
    line-height: 1.2;
    white-space: nowrap;
  }

  .logo-tag {
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 600;
    letter-spacing: 0.22em;
    color: var(--lacquer-deep);
  }

  .sidebar-nav {
    flex: 1;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow-y: auto;
  }

  .nav-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
    padding: 12px 6px 10px;
    border: none;
    border-radius: 10px;
    background: transparent;
    color: var(--ink-500);
    font-family: var(--font-body);
    font-size: 11px;
    font-weight: 500;
    text-align: center;
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .nav-item:hover {
    background: var(--ink-800);
    color: var(--ink-100);
  }

  .nav-item.active {
    background: var(--ink-800);
    color: var(--paper);
    font-weight: 600;
  }

  .nav-icon {
    display: flex;
    flex-shrink: 0;
  }

  .nav-item.active .nav-icon {
    color: var(--lacquer-deep);
  }

  .nav-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  .sidebar-version {
    padding: 12px 16px;
    font-size: 10px;
    font-family: var(--font-mono);
    color: var(--ink-600);
    text-align: center;
  }

  .sidebar-toggle {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 10px;
    border: none;
    border-top: 1px solid var(--ink-700);
    background: transparent;
    color: var(--ink-500);
    cursor: pointer;
  }

  .sidebar-toggle:hover {
    color: var(--ink-100);
  }

  .toggle-icon {
    display: flex;
    transition: transform 0.22s ease;
  }

  .toggle-icon.flipped {
    transform: rotate(180deg);
  }
</style>
