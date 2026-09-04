<script lang="ts">
  import { nav, switchTab, type Tab } from '../nav.svelte';

  // 4 abas do Tintômetro (a Roda vive em Mais → "Guia da roda de cores").
  const tabs: { id: Tab; label: string }[] = [
    { id: 'mesclar', label: 'Mesclar' },
    { id: 'catalogo', label: 'Catálogo' },
    { id: 'cor', label: 'Cor' },
    { id: 'mais', label: 'Mais' },
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
      {t.label}
    </button>
  {/each}
</nav>

<style>
  /* Dock flutuante em pílula (papel + sombra suave): a navegação é um objeto
     na bancada, não uma parede. Item ativo = pílula LACA com texto branco. */
  .tab-bar {
    position: fixed;
    bottom: calc(12px + var(--safe-bottom));
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    display: flex;
    gap: 4px;
    padding: 8px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: var(--radius-pill);
    box-shadow: 0 12px 32px rgba(26, 23, 18, 0.16);
    max-width: calc(100vw - 24px);
  }

  .tab-item {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 44px;
    padding: 0 18px;
    border-radius: var(--radius-pill);
    font-size: 14px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
    transition: background 0.18s ease, color 0.15s ease;
  }

  .tab-item:active {
    color: var(--lacquer-deep);
  }

  .tab-item.active {
    background: var(--laca);
    color: #ffffff;
  }
</style>
