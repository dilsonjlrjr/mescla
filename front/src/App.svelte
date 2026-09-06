<script lang="ts">
  // rf-04: shell reorganizado em 4 telas (T1 Pergunta/T2 Plano/T3 Receitas/
  // T4 Minhas tintas), cada uma com seu próprio cabeçalho+rodapé fixos
  // (Header.svelte é comum; rodapé é por tela). Sem TabBar — a navegação
  // mora no Header (nc-nav) + a marca leva de volta a T1.
  import ToastRegion from './lib/components/ToastRegion.svelte';
  import BrandMark from './lib/components/BrandMark.svelte';
  import PerguntaView from './lib/views/PerguntaView.svelte';
  import PlannerView from './lib/views/PlannerView.svelte';
  import ReceitasView from './lib/views/ReceitasView.svelte';
  import CatalogoView from './lib/views/CatalogoView.svelte';
  import { nav, initNav } from './lib/nav.svelte';
  import { loadCatalog } from './lib/services/catalog';
  import { engineReady } from './lib/services/engine';
  import { initPwa } from './lib/pwa.svelte';

  let booted = $state(false);
  let bootError = $state('');

  initNav();
  initPwa();

  // O catálogo trava o boot (as listas precisam dele); o motor inicializa
  // EM PARALELO sem travar — quem precisar dele aguarda via engineReady().
  void engineReady().catch(e => console.error('Motor de cor não inicializou:', e));
  loadCatalog()
    .then(() => (booted = true))
    .catch(e => {
      console.error('Catálogo não carregou:', e);
      bootError = 'O catálogo não carregou. Verifique a conexão e recarregue.';
    });

  // As 4 telas ficam montadas (display:none) pra preservar estado ao trocar —
  // troca de idioma ou de aba não perde a resposta atual (US-18, NFR-09).
</script>

{#if bootError}
  <div class="boot">
    <BrandMark size={44} />
    <p class="boot-error">{bootError}</p>
    <button class="btn-ghost" onclick={() => location.reload()}>Recarregar</button>
  </div>
{:else if !booted}
  <div class="boot">
    <BrandMark size={52} />
    <p class="boot-word font-display">Mescla AI</p>
  </div>
{:else}
  <!-- Raiz do protótipo (docs/oficial/Mescla AI.html): container-type:
       inline-size é o que faz as unidades `cqi` das telas responderem à
       largura do app, não à viewport (NFR-01, 820–1366px). -->
  <main class="views">
    <div class="view" class:hidden={nav.tab !== 'pergunta'}><PerguntaView /></div>
    <div class="view" class:hidden={nav.tab !== 'plano'}><PlannerView /></div>
    <div class="view" class:hidden={nav.tab !== 'receitas'}><ReceitasView /></div>
    <div class="view" class:hidden={nav.tab !== 'estante'}><CatalogoView /></div>
  </main>
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
    background: var(--color-bg);
  }

  .boot-word {
    font-size: 28px;
    font-weight: 600;
    color: var(--color-text);
    letter-spacing: -0.01em;
    line-height: 1;
    margin-top: 2px;
  }

  .boot-error {
    font-size: 14px;
    color: var(--color-accent-2);
    max-width: 260px;
    text-align: center;
  }

  .views {
    width: 100%;
    height: 100dvh;
    position: relative;
    overflow: hidden;
    container-type: inline-size;
    background: var(--color-bg);
    color: var(--color-text);
    font-family: var(--font-body);
    font-size: 15px;
    line-height: 1.45;
    display: flex;
    flex-direction: column;
    user-select: none;
  }

  .view {
    height: 100%;
    min-height: 0;
  }

  .view.hidden {
    display: none;
  }
</style>
