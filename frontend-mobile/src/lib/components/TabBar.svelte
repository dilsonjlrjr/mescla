<script lang="ts">
  import Icon from './Icon.svelte';
  import { nav, switchTab, type Tab } from '../nav.svelte';

  const tabs: { id: Tab; label: string; icon: 'droplet' | 'grid' | 'pipette' | 'wheel' | 'dots' }[] = [
    { id: 'mesclar', label: 'Mesclar', icon: 'droplet' },
    { id: 'catalogo', label: 'Catálogo', icon: 'grid' },
    { id: 'cor', label: 'Cor', icon: 'pipette' },
    { id: 'roda', label: 'Roda', icon: 'wheel' },
    { id: 'mais', label: 'Mais', icon: 'dots' },
  ];

  function tap(tab: Tab) {
    if (nav.tab !== tab && navigator.vibrate) navigator.vibrate(10);
    switchTab(tab);
  }
</script>

<nav class="tab-bar" aria-label="Navegação principal">
  {#each tabs as t (t.id)}
    <button
      class="tab-item"
      class:active={nav.tab === t.id}
      onclick={() => tap(t.id)}
      aria-current={nav.tab === t.id ? 'page' : undefined}
    >
      <span class="tab-ico"><Icon name={t.icon} size={22} /></span>
      <span class="tab-label">{t.label}</span>
    </button>
  {/each}
</nav>

<style>
  .tab-bar {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 50;
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    height: calc(var(--tab-bar-h) + var(--safe-bottom));
    padding-bottom: var(--safe-bottom);
    background: var(--ink-900);
    border-top: 1px solid var(--ink-700);
  }

  .tab-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    font-size: 11px;
    font-weight: 500;
    color: var(--ink-500);
    transition: color 0.15s ease;
  }

  /* Pílula atrás do ícone: a aba ativa ganha um "rótulo" de laca — mesma
     linguagem dos botões, sem depender só da cor pra indicar estado. */
  .tab-ico {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 52px;
    height: 27px;
    border-radius: var(--radius-pill);
    transition: background 0.18s ease, color 0.15s ease;
  }

  .tab-item:active {
    color: var(--ink-300);
  }

  .tab-item.active {
    color: var(--ink-100);
    font-weight: 600;
  }

  .tab-item.active .tab-ico {
    background: color-mix(in srgb, var(--lacquer) 15%, transparent);
    color: var(--lacquer-deep);
  }
</style>
