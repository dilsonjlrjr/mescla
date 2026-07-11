<script lang="ts">
  // Barra superior — substitui a sidebar. Marca à esquerda, navegação central
  // em pills, busca global (⌘K) e guia à direita. A barra inteira é arrastável
  // (o app não tem titlebar); os controles marcam no-drag.
  import Icon from './Icon.svelte';
  import BrandMark from './BrandMark.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface Props {
    currentView: View;
    onNavigate: (view: View) => void;
    onSearch: () => void;
    onHelp: () => void;
  }

  let { currentView, onNavigate, onSearch, onHelp }: Props = $props();

  // Rótulos curtos: a navegação precisa caber numa linha (regra dura).
  const navItems: { id: View; label: string }[] = [
    { id: 'home', label: 'Início' },
    { id: 'mix', label: 'Equivalência' },
    { id: 'wheel', label: 'Roda' },
    { id: 'catalog', label: 'Catálogo' },
    { id: 'stock', label: 'Estoque' },
    { id: 'color-search', label: 'Cor' },
    { id: 'compare', label: 'Comparar' },
    { id: 'manufacturers', label: 'Marcas' },
  ];

  const isMac = navigator.platform.toLowerCase().includes('mac');
</script>

<header class="topbar" style="--wails-draggable: drag;" class:mac={isMac}>
  <button class="brand" style="--wails-draggable: no-drag;" onclick={() => onNavigate('home')}>
    <BrandMark size={26} />
    <span class="brand-word font-display">Mescla</span>
  </button>

  <nav class="topnav" style="--wails-draggable: no-drag;" aria-label="Navegação principal">
    {#each navItems as item (item.id)}
      <button
        class="topnav-item"
        class:active={currentView === item.id}
        onclick={() => onNavigate(item.id)}
        aria-current={currentView === item.id ? 'page' : undefined}
      >
        {item.label}
      </button>
    {/each}
  </nav>

  <div class="topbar-right" style="--wails-draggable: no-drag;">
    <button class="search-trigger" onclick={onSearch} title="Busca global">
      <Icon name="search" size={15} />
      <span>Buscar</span>
      <kbd class="font-mono">{isMac ? '⌘K' : 'Ctrl K'}</kbd>
    </button>
    <button class="help-btn" onclick={onHelp} aria-label="Guia de uso" title="Guia de uso">
      <Icon name="info" size={17} />
    </button>
  </div>
</header>

<style>
  /* O app.css do desktop não tem reset global de button (SMUI cuida dos
     dele) — este componente zera os próprios. */
  button {
    font: inherit;
    color: inherit;
    border: none;
    background: none;
    padding: 0;
    cursor: pointer;
  }

  .topbar {
    display: flex;
    align-items: center;
    gap: 20px;
    height: 60px;
    padding: 0 20px;
    background: var(--ink-900);
    border-bottom: 1px solid var(--ink-700);
    flex-shrink: 0;
  }

  /* Espaço pros semáforos do macOS (janela sem titlebar) */
  .topbar.mac {
    padding-left: 84px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .brand-word {
    font-size: 17px;
    font-weight: 750;
    color: var(--paper);
    letter-spacing: -0.01em;
  }

  .topnav {
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 0;
    overflow: hidden;
  }

  .topnav-item {
    position: relative;
    padding: 8px 13px;
    border-radius: var(--radius-pill);
    font-size: 13px;
    font-weight: 500;
    color: var(--ink-500);
    white-space: nowrap;
    transition: color 0.15s ease, background 0.15s ease;
  }

  .topnav-item:hover {
    color: var(--ink-100);
    background: var(--ink-800);
  }

  .topnav-item.active {
    color: var(--paper);
    background: var(--ink-800);
    font-weight: 600;
  }

  /* Costura da marca embaixo do item ativo */
  .topnav-item.active::after {
    content: '';
    position: absolute;
    left: 50%;
    bottom: -1px;
    transform: translateX(-50%);
    width: 18px;
    height: 3px;
    border-radius: var(--radius-pill);
    background: var(--lacquer);
  }

  .topbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-left: auto;
    flex-shrink: 0;
  }

  .search-trigger {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 12px;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-pill);
    background: var(--ink-950);
    color: var(--ink-500);
    font-size: 13px;
    transition: border-color 0.15s ease, color 0.15s ease;
  }

  .search-trigger:hover {
    border-color: var(--ink-600);
    color: var(--ink-300);
  }

  .search-trigger kbd {
    font-size: 10.5px;
    padding: 2px 6px;
    border-radius: 5px;
    background: var(--ink-800);
    color: var(--ink-500);
  }

  .help-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: var(--radius-pill);
    color: var(--ink-500);
    transition: color 0.15s ease, background 0.15s ease;
  }

  .help-btn:hover {
    color: var(--ink-100);
    background: var(--ink-800);
  }
</style>
