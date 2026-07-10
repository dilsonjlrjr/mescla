<script lang="ts">
  import Icon from './Icon.svelte';
  import BrandMark from './BrandMark.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface Props {
    currentView: View;
    collapsed: boolean;
    onNavigate: (view: View) => void;
    onToggle: () => void;
    onHelp: () => void;
  }

  let { currentView, collapsed, onNavigate, onToggle, onHelp }: Props = $props();

  // Ordenado pela jornada: a equivalência é o coração do produto.
  const navItems: { id: View; label: string; icon: 'home' | 'grid' | 'building' | 'pipette' | 'swap' | 'flask' | 'wheel' | 'box' }[] = [
    { id: 'home', label: 'Início', icon: 'home' },
    { id: 'mix', label: 'Equivalência', icon: 'flask' },
    { id: 'wheel', label: 'Roda de cor', icon: 'wheel' },
    { id: 'catalog', label: 'Catálogo', icon: 'grid' },
    { id: 'stock', label: 'Meu estoque', icon: 'box' },
    { id: 'color-search', label: 'Buscar cor', icon: 'pipette' },
    { id: 'compare', label: 'Comparar', icon: 'swap' },
    { id: 'manufacturers', label: 'Marcas', icon: 'building' },
  ];
</script>

<aside class="sidebar" class:collapsed>
  <div class="titlebar-spacer" style="--wails-draggable: drag;"></div>

  <div class="sidebar-logo">
    <BrandMark size={30} />
    {#if !collapsed}
      <div>
        <div class="logo-word">Mescla</div>
        <div class="logo-tag">cor certa, qualquer marca</div>
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

  <button class="nav-item help-item" onclick={onHelp} title={collapsed ? 'Guia de uso' : undefined}>
    <span class="nav-icon"><Icon name="info" size={18} /></span>
    {#if !collapsed}
      <span class="nav-label">Guia de uso</span>
    {/if}
  </button>

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
    height: 52px;
    flex-shrink: 0;
  }

  .sidebar-logo {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 4px 16px 18px;
    border-bottom: 1px solid var(--ink-700);
  }

  .logo-word {
    font-family: var(--font-display);
    font-size: 17px;
    font-weight: 700;
    color: var(--paper);
    line-height: 1.15;
    white-space: nowrap;
    letter-spacing: -0.01em;
  }

  .logo-tag {
    font-family: var(--font-mono);
    font-size: 8.5px;
    font-weight: 500;
    letter-spacing: 0.08em;
    color: var(--ink-500);
    white-space: nowrap;
  }

  .sidebar-nav {
    flex: 1;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow-y: auto;
  }

  /* Item horizontal (ícone à esquerda, label à direita) com "costura" de
     laca à esquerda no ativo — o mesmo traço vertical do ícone da marca. */
  .nav-item {
    position: relative;
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    gap: 11px;
    width: 100%;
    padding: 10px 12px;
    border: none;
    border-radius: var(--radius-control);
    background: transparent;
    color: var(--ink-500);
    font-family: var(--font-body);
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .sidebar.collapsed .nav-item {
    justify-content: center;
    padding: 12px 6px;
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

  .nav-item.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 9px;
    bottom: 9px;
    width: 3px;
    border-radius: var(--radius-pill);
    background: var(--lacquer);
  }

  .sidebar.collapsed .nav-item.active::before {
    display: none;
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

  .help-item {
    margin: 0 8px;
    width: calc(100% - 16px);
    border-top: 1px solid var(--ink-700);
    border-radius: 0;
    padding-top: 14px;
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
