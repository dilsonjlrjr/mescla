<script lang="ts">
  // Barra superior (Tintômetro): papel, hairline embaixo. Marca à esquerda
  // (logo gota + Mescla), navegação central em TEXTO com sublinhado laca,
  // pílula de busca ⌘K em mono à direita. Home fica no logo.
  import BrandMark from './BrandMark.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface Props {
    currentView: View;
    onNavigate: (view: View) => void;
    onSearch: () => void;
  }

  let { currentView, onNavigate, onSearch }: Props = $props();

  const navItems: { id: View; label: string }[] = [
    { id: 'mix', label: 'Equivalência' },
    { id: 'catalog', label: 'Catálogo' },
    { id: 'color-search', label: 'Cor' },
    { id: 'stock', label: 'Estoque' },
    { id: 'compare', label: 'Comparar' },
  ];

  // Fabricantes vive dentro de Catálogo; a Roda continua acessível pela busca
  // global — na barra, marca o item "pai" mais próximo.
  let activeNav = $derived(
    currentView === 'manufacturers' ? 'catalog' : currentView === 'wheel' ? 'color-search' : currentView
  );

  const isMac = navigator.platform.toLowerCase().includes('mac');
</script>

<header class="topbar" style="--wails-draggable: drag;" class:mac={isMac}>
  <button class="brand" style="--wails-draggable: no-drag;" onclick={() => onNavigate('home')} aria-label="Início">
    <BrandMark size={26} />
    <span class="brand-word font-display">Mescla</span>
  </button>

  <nav class="topnav" style="--wails-draggable: no-drag;" aria-label="Navegação principal">
    {#each navItems as item (item.id)}
      <button
        class="topnav-item"
        class:active={activeNav === item.id}
        onclick={() => onNavigate(item.id)}
        aria-current={activeNav === item.id ? 'page' : undefined}
      >
        {item.label}
      </button>
    {/each}
  </nav>

  <div class="topbar-right" style="--wails-draggable: no-drag;">
    <button class="search-trigger font-mono" onclick={onSearch} title="Busca global">
      <span class="kbd">{isMac ? '⌘K' : 'Ctrl K'}</span>
      <span>buscar tinta</span>
    </button>
  </div>
</header>

<style>
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
    gap: 24px;
    height: 64px;
    padding: 0 24px;
    background: var(--papel);
    border-bottom: 1px solid var(--hairline);
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
    color: var(--grafite);
    letter-spacing: -0.01em;
  }

  .topnav {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 30px;
    min-width: 0;
    overflow: hidden;
    flex: 1;
  }

  .topnav-item {
    position: relative;
    padding: 21px 0;
    font-size: 13.5px;
    font-weight: 500;
    color: var(--text-2);
    white-space: nowrap;
    transition: color 0.15s ease;
  }

  .topnav-item:hover {
    color: var(--grafite);
  }

  .topnav-item.active {
    color: var(--grafite);
    font-weight: 700;
  }

  /* Sublinhado laca do item ativo */
  .topnav-item.active::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: 11px;
    height: 2px;
    background: var(--laca);
  }

  .topbar-right {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .search-trigger {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    height: 36px;
    padding: 0 16px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-pill);
    background: var(--papel);
    color: var(--text-2);
    font-size: 12px;
    transition: border-color 0.15s ease, color 0.15s ease;
  }

  .search-trigger:hover {
    border-color: var(--grafite);
    color: var(--grafite);
  }

  .search-trigger .kbd {
    font-weight: 600;
    color: var(--grafite);
  }
</style>
