<script lang="ts">
  // Cabeçalho fixo (rf-04, NFR-02/CA16): marca à esquerda (leva a T1), nav
  // central com as 4 telas (Plano da peça/Catálogo/Receitas/Minhas tintas —
  // T1 "Pergunta e resposta" mora atrás da marca, como o "início" de sempre),
  // seletor de idioma (RG-22/US-18) e busca global (⌘K) à direita.
  import BrandMark from './BrandMark.svelte';
  import { t, i18n, LANGS, setLang, type Lang } from '../i18n.svelte';

  type View =
    | 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare'
    | 'mix' | 'wheel' | 'stock' | 'planner' | 'receitas';

  interface Props {
    currentView: View;
    onNavigate: (view: View) => void;
    onSearch: () => void;
  }

  let { currentView, onNavigate, onSearch }: Props = $props();

  const navItems: { id: View; key: 'navPlano' | 'navCatalogo' | 'navReceitas' | 'navTintas' }[] = [
    { id: 'planner', key: 'navPlano' },
    { id: 'catalog', key: 'navCatalogo' },
    { id: 'receitas', key: 'navReceitas' },
    { id: 'stock', key: 'navTintas' },
  ];

  // Fabricantes vive dentro de Minhas tintas (aba); a Roda/Comparar/Equivalência
  // continuam acessíveis pela busca global — na barra, marcam o item "pai".
  let activeNav = $derived(
    currentView === 'manufacturers' ? 'stock'
    : currentView === 'wheel' || currentView === 'color-search' || currentView === 'mix' || currentView === 'home' ? 'home'
    : currentView
  );

  const isMac = navigator.platform.toLowerCase().includes('mac');

  function onLangChange(e: Event) {
    setLang((e.currentTarget as HTMLSelectElement).value as Lang);
  }
</script>

<header class="topbar view-fixed" style="--wails-draggable: drag;" class:mac={isMac}>
  <button class="brand" style="--wails-draggable: no-drag;" onclick={() => onNavigate('home')} aria-label="Mescla">
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
        {t(item.key)}
      </button>
    {/each}
  </nav>

  <div class="topbar-right" style="--wails-draggable: no-drag;">
    <button class="search-trigger font-mono" onclick={onSearch} title="Busca global">
      <span class="kbd">{isMac ? '⌘K' : 'Ctrl K'}</span>
    </button>

    <div class="lang-select-wrap">
      <select class="lang-select font-mono" value={i18n.lang} onchange={onLangChange} aria-label="Idioma">
        {#each LANGS as l (l.code)}
          <option value={l.code}>{l.code.toUpperCase()}</option>
        {/each}
      </select>
    </div>
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
    gap: var(--space-6);
    height: 64px;
    padding: 0 var(--space-6);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-divider);
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
    min-height: 44px;
  }

  .brand-word {
    font-size: 17px;
    font-weight: 750;
    color: var(--color-text);
    letter-spacing: -0.01em;
  }

  .topnav {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-8);
    min-width: 0;
    overflow: hidden;
    flex: 1;
  }

  .topnav-item {
    position: relative;
    min-height: 44px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    font-size: 13.5px;
    font-weight: 500;
    color: color-mix(in srgb, var(--color-text) 66%, transparent);
    white-space: nowrap;
    transition: color 0.15s ease;
  }

  .topnav-item:hover {
    color: var(--color-text);
  }

  .topnav-item.active {
    color: var(--color-text);
    font-weight: 700;
  }

  /* Sublinhado de acento do item ativo */
  .topnav-item.active::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: 11px;
    height: 2px;
    background: var(--color-accent);
  }

  .topbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  .search-trigger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    height: 44px;
    padding: 0 14px;
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-pill);
    background: var(--color-bg);
    color: color-mix(in srgb, var(--color-text) 66%, transparent);
    font-size: 12px;
    transition: border-color 0.15s ease, color 0.15s ease;
  }

  .search-trigger:hover {
    border-color: var(--color-accent);
    color: var(--color-text);
  }

  .lang-select-wrap {
    display: flex;
  }

  .lang-select {
    min-height: 44px;
    height: 44px;
    padding: 0 10px;
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-pill);
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .lang-select:hover {
    border-color: var(--color-accent);
  }
</style>
