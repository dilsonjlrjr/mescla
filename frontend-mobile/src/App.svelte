<script lang="ts">
  import TabBar from './lib/components/TabBar.svelte';
  import ToastRegion from './lib/components/ToastRegion.svelte';
  import BrandMark from './lib/components/BrandMark.svelte';
  import MesclarView from './lib/views/MesclarView.svelte';
  import CatalogoView from './lib/views/CatalogoView.svelte';
  import CorView from './lib/views/CorView.svelte';
  import MaisView from './lib/views/MaisView.svelte';
  import { nav, initNav } from './lib/nav.svelte';
  import { loadCatalog } from './lib/services/catalog';
  import { engineReady } from './lib/services/engine';
  import { initPwa } from './lib/pwa.svelte';

  let booted = $state(false);
  let bootError = $state('');

  initNav();
  initPwa();

  // O catálogo trava o boot (as listas precisam dele); o motor WASM inicializa
  // EM PARALELO sem travar — quem precisar dele aguarda via engineReady().
  void engineReady().catch(e => console.error('Motor de cor não inicializou:', e));
  loadCatalog()
    .then(() => (booted = true))
    .catch(e => {
      console.error('Catálogo não carregou:', e);
      bootError = 'O catálogo não carregou. Verifique a conexão e recarregue.';
    });

  // As views ficam montadas (display:none) pra preservar estado ao trocar de
  // aba — busca digitada no Catálogo não some ao ir e voltar.
</script>

{#if bootError}
  <div class="boot">
    <BrandMark size={44} />
    <p class="boot-error">{bootError}</p>
    <button class="btn-ghost" onclick={() => location.reload()}>Recarregar</button>
  </div>
{:else if !booted}
  <div class="boot">
    <BrandMark size={44} />
    <p class="boot-tag font-mono">cor certa, qualquer marca</p>
  </div>
{:else}
  <main class="views">
    <div class="view" class:hidden={nav.tab !== 'mesclar'}><MesclarView /></div>
    <div class="view" class:hidden={nav.tab !== 'catalogo'}><CatalogoView /></div>
    <div class="view" class:hidden={nav.tab !== 'cor'}><CorView /></div>
    <div class="view" class:hidden={nav.tab !== 'mais'}><MaisView /></div>
  </main>
  <TabBar />
{/if}

<ToastRegion />

<style>
  .boot {
    position: fixed;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    background: var(--ink-950);
  }

  .boot-tag {
    font-size: 11px;
    letter-spacing: 0.14em;
    color: var(--ink-500);
  }

  .boot-error {
    font-size: 14px;
    color: var(--delta-poor);
    max-width: 260px;
    text-align: center;
  }

  .views {
    height: 100dvh;
    padding-top: var(--safe-top);
    padding-bottom: calc(var(--tab-bar-h) + var(--safe-bottom));
  }

  .view {
    height: 100%;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .view.hidden {
    display: none;
  }
</style>
