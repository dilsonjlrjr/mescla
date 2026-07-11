<script lang="ts">
  // Mais — MINHA ESTANTE (toggles laca por marca, a mesma estante da Mesclar)
  // + FERRAMENTAS: Meu estoque, Comparar (lista-âncora), Guia da roda de
  // cores, Exportar estoque CSV e Sobre. Sub-telas empilham camadas de
  // histórico: o back volta pro menu.
  import Icon from '../components/Icon.svelte';
  import BrandMark from '../components/BrandMark.svelte';
  import BottomSheet from '../components/BottomSheet.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import FullScreenSearch from '../components/FullScreenSearch.svelte';
  import StockManager from '../components/StockManager.svelte';
  import { pushLayer, switchTab } from '../nav.svelte';
  import { allManufacturers, paintById, type Paint } from '../services/catalog';
  import { compareToAnchor, stockToCSV, type SearchResult } from '../services/engine';
  import { appState, addToCompare, removeFromCompare } from '../appState.svelte';
  import { shelf, toggleShelf } from '../services/shelf.svelte';
  import { stock } from '../services/stock.svelte';
  import { recents } from '../recents.svelte';
  import { pwa, promptInstall } from '../pwa.svelte';
  import { toast } from '../toast.svelte';
  import { deltaVerdict, deltaIsGood } from '../ui';

  let subView: 'menu' | 'comparar' | 'estoque' = $state('menu');
  // Estante colapsada por padrão: com 35 marcas, a lista inteira empurrava
  // as Ferramentas pra fora da dobra. Mostra só as marcas ativas + expansor.
  let shelfExpanded = $state(false);
  let visibleShelfBrands = $derived(
    shelfExpanded
      ? allManufacturers()
      : allManufacturers().filter(m => shelf.manufacturerIds.includes(m.id))
  );
  let hiddenShelfCount = $derived(allManufacturers().length - visibleShelfBrands.length);
  let deltaSheetOpen = $state(false);
  let aboutOpen = $state(false);
  let searchOpen = $state(false);
  let anchorId: number | null = $state(null);
  let compared: SearchResult[] = $state([]);

  let closeSubLayer: (() => void) | null = null;

  function openSub(view: 'comparar' | 'estoque') {
    if (subView === view) return;
    subView = view;
    closeSubLayer = pushLayer(() => {
      closeSubLayer = null;
      subView = 'menu';
    });
  }

  function backToMenu() {
    closeSubLayer?.();
  }

  // "Tenho outra parecida" (detalhe): abre direto a sub-tela do estoque.
  $effect(() => {
    if (appState.pendingStockPrefill && subView !== 'estoque') {
      openSub('estoque');
    }
  });

  function toggleBrand(id: number) {
    if (navigator.vibrate) navigator.vibrate(10);
    toggleShelf(id);
  }

  // ── Comparar: lista-âncora ──
  let anchor = $derived(anchorId ? paintById(anchorId) : null);

  $effect(() => {
    // âncora padrão: primeira tinta adicionada
    if (appState.compareIds.length > 0 && (anchorId === null || !appState.compareIds.includes(anchorId))) {
      anchorId = appState.compareIds[0];
    }
    if (appState.compareIds.length === 0) {
      anchorId = null;
      compared = [];
    }
  });

  $effect(() => {
    const a = anchorId;
    const ids = [...appState.compareIds];
    if (a === null || ids.length < 2) {
      compared = [];
      return;
    }
    compareToAnchor(a, ids)
      .then(res => (compared = res))
      .catch(() => (compared = []));
  });

  function addPaint(p: Paint) {
    if (!addToCompare(p.id)) {
      toast(appState.compareIds.includes(p.id) ? 'Já está na comparação' : 'Máximo de 6 tintas', 'error');
    }
  }

  function promote(id: number) {
    if (navigator.vibrate) navigator.vibrate(10);
    anchorId = id;
  }

  function remove(id: number) {
    removeFromCompare(id);
  }

  // ── Exportar estoque CSV (mesma rotina do StockManager) ──
  async function exportStockCsv() {
    if (stock.paints.length === 0) {
      toast('O estoque está vazio.', 'error');
      return;
    }
    try {
      const csv = await stockToCSV(stock.paints);
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'meu-estoque-mescla.csv';
      a.click();
      URL.revokeObjectURL(url);
      toast(`${stock.paints.length} ${stock.paints.length === 1 ? 'tinta exportada' : 'tintas exportadas'}.`);
    } catch {
      toast('Não consegui exportar o estoque.', 'error');
    }
  }
</script>

<div class="mais">
  {#if subView === 'menu'}
    <header class="head">
      <BrandMark size={26} />
      <h1 class="head-title font-display">Mais</h1>
    </header>

    <p class="section-label" style="margin-top: 18px;">Ferramentas</p>
    <div class="tools">
      <button class="tool-row pressable" onclick={() => openSub('estoque')}>
        <span class="tool-name">Meu estoque</span>
        <span class="tool-value font-mono">{stock.paints.length} {stock.paints.length === 1 ? 'tinta' : 'tintas'}</span>
        <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
      </button>
      <button class="tool-row pressable" onclick={() => openSub('comparar')}>
        <span class="tool-name">Comparar</span>
        <span class="tool-value font-mono">{appState.compareIds.length > 0 ? `${appState.compareIds.length} tintas` : 'lista-âncora'}</span>
        <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
      </button>
      <button class="tool-row pressable" onclick={() => switchTab('roda')}>
        <span class="tool-name">Guia da roda de cores</span>
        <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
      </button>
      <button class="tool-row pressable" onclick={exportStockCsv}>
        <span class="tool-name">Exportar estoque CSV</span>
        <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
      </button>
      {#if pwa.canInstall}
        <button class="tool-row pressable" onclick={promptInstall}>
          <span class="tool-name">Instalar o app</span>
          <span class="tool-value font-mono">offline</span>
          <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
        </button>
      {/if}
      <button class="tool-row pressable" onclick={() => (aboutOpen = true)}>
        <span class="tool-name">Sobre o Mescla</span>
        <span class="tool-value font-mono">v1.0</span>
        <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
      </button>
    </div>

    <p class="section-label" style="margin-top: 26px;">Minha estante</p>
    <p class="shelf-hint">Receitas são calculadas só com as marcas que você tem.</p>

    <div class="shelf-list">
      {#each visibleShelfBrands as m (m.id)}
        {@const on = shelf.manufacturerIds.includes(m.id)}
        <button class="shelf-row pressable" onclick={() => toggleBrand(m.id)} aria-pressed={on}>
          <span class="shelf-name">{m.name}</span>
          <span class="switch" class:on><span class="knob"></span></span>
        </button>
      {/each}
      {#if !shelfExpanded && hiddenShelfCount > 0}
        <button class="shelf-row shelf-expand pressable" onclick={() => (shelfExpanded = true)}>
          <span class="shelf-name" style="color: var(--laca); font-weight: 600;">
            {shelf.manufacturerIds.length > 0 ? `Mostrar todas as ${allManufacturers().length} marcas` : `Escolher marcas (${allManufacturers().length})`}
          </span>
          <span class="tool-chev"><Icon name="chevron-right" size={15} /></span>
        </button>
      {:else if shelfExpanded}
        <button class="shelf-row shelf-expand pressable" onclick={() => (shelfExpanded = false)}>
          <span class="shelf-name" style="color: var(--laca); font-weight: 600;">Mostrar só as minhas</span>
        </button>
      {/if}
    </div>
  {:else if subView === 'comparar'}
    <header class="sub-head">
      <button class="sub-back pressable" onclick={backToMenu}>
        <Icon name="chevron-left" size={16} /> Mais
      </button>
    </header>
    <h1 class="screen-title">Comparar</h1>
    <p class="compare-sub font-mono">todas contra a âncora, por distância de cor</p>

    {#if appState.compareIds.length === 0}
      <div class="empty-state">
        <p class="empty-title">Nenhuma tinta na comparação</p>
        <p class="empty-hint">Adicione pelo Catálogo (detalhe da tinta) ou pelo botão abaixo.</p>
      </div>
    {:else if anchor}
      <!-- Âncora: a pergunta é "qual destas é mais parecida com ELA?" -->
      <div class="anchor">
        <span class="anchor-swatch" style="background: rgb({anchor.r}, {anchor.g}, {anchor.b});"></span>
        <div class="anchor-text">
          <span class="section-label" style="color: var(--laca);">Âncora</span>
          <span class="anchor-name">{anchor.name}</span>
          <span class="anchor-meta font-mono">{anchor.code} · {anchor.manufacturer}</span>
        </div>
        <button class="row-trash pressable" onclick={() => remove(anchor!.id)} aria-label="Remover âncora">
          <Icon name="trash" size={17} />
        </button>
      </div>

      {#if compared.length > 0}
        <p class="compare-caption">da mais próxima à mais distante · toque pra virar âncora</p>
      {/if}
      {#each compared as res (res.paintId)}
        <div class="compare-row">
          <button class="compare-main pressable" onclick={() => promote(res.paintId)}>
            <span class="compare-pair" aria-hidden="true">
              <span style="background: rgb({anchor.r}, {anchor.g}, {anchor.b});"></span>
              <span style="background: rgb({res.r}, {res.g}, {res.b});"></span>
            </span>
            <span class="compare-text">
              <span class="compare-name">{res.name}</span>
              <span class="compare-meta font-mono">{res.code} · {res.manufacturer} · {deltaVerdict(res.deltaE)}</span>
            </span>
            <span class="compare-delta font-mono" class:good={deltaIsGood(res.deltaE)}>{res.deltaE.toFixed(1)}</span>
          </button>
          <button class="row-trash pressable" onclick={() => remove(res.paintId)} aria-label="Remover {res.name}">
            <Icon name="trash" size={17} />
          </button>
        </div>
      {/each}
    {/if}

    {#if appState.compareIds.length < 6}
      <button class="btn-ghost" style="width: 100%; margin-top: 14px;" onclick={() => (searchOpen = true)}>
        + Adicionar tinta à comparação ({appState.compareIds.length}/6)
      </button>
    {/if}
    <button class="delta-link pressable font-mono" onclick={() => (deltaSheetOpen = true)}>o que significa o ΔE?</button>
  {:else if subView === 'estoque'}
    <StockManager onBack={backToMenu} />
  {/if}
</div>

<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} />
<FullScreenSearch
  open={searchOpen}
  onClose={() => (searchOpen = false)}
  onSelect={addPaint}
  recentIds={recents.paintIds}
  placeholder="Tinta pra comparar…"
/>

<!-- Sobre -->
<BottomSheet open={aboutOpen} onClose={() => (aboutOpen = false)}>
  <div class="about">
    <BrandMark size={44} />
    <p class="about-name font-display">Mescla</p>
    <p class="about-tag font-mono">cor certa, qualquer marca · v1.0</p>
    <p class="about-note">
      Equivalência de tintas para pintores de miniaturas. Cores de tela são
      aproximadas: confie no ΔE.
    </p>
    <button class="btn-primary" style="margin-top: 18px;" onclick={() => (aboutOpen = false)}>Fechar</button>
  </div>
</BottomSheet>

<style>
  .mais {
    padding: 0 16px 16px;
  }

  .head {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 56px;
    margin: 0 -16px;
    padding: 8px 16px;
    background: var(--papel);
    border-bottom: 1px solid var(--hairline);
  }

  .head-title {
    font-size: 20px;
    font-weight: 750;
    color: var(--grafite);
  }

  .shelf-hint {
    font-size: 13px;
    color: var(--ink-500);
    margin: 4px 0 6px;
  }

  .shelf-list,
  .tools {
    display: flex;
    flex-direction: column;
  }

  .shelf-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    min-height: 56px;
    padding: 8px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .shelf-name {
    flex: 1;
    min-width: 0;
    font-size: 16px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Toggle: trilho pílula, laca quando on, knob branco */
  .switch {
    flex-shrink: 0;
    width: 46px;
    height: 27px;
    border-radius: var(--radius-pill);
    background: var(--hairline);
    position: relative;
    transition: background 0.18s ease;
  }

  .switch.on {
    background: var(--laca);
  }

  .knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 21px;
    height: 21px;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.18s ease;
  }

  .switch.on .knob {
    transform: translateX(19px);
  }

  .tool-row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 56px;
    padding: 8px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .tool-name {
    flex: 1;
    min-width: 0;
    font-size: 16px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tool-value {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  .tool-chev {
    display: flex;
    color: var(--hairline);
    flex-shrink: 0;
  }

  /* ── Comparar ── */
  .sub-head {
    padding: 8px 0 2px;
  }

  .sub-back {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    min-height: 40px;
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-500);
  }

  .compare-sub {
    font-size: 11px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--ink-500);
    margin: 4px 0 14px;
  }

  .anchor {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 0;
    border-top: 1px solid var(--hairline);
    border-bottom: 1px solid var(--hairline);
    margin-bottom: 10px;
  }

  .anchor-swatch {
    width: 64px;
    height: 56px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
    flex-shrink: 0;
  }

  .anchor-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .anchor-name {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 750;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .anchor-meta {
    font-size: 12px;
    color: var(--ink-500);
  }

  .compare-caption {
    font-size: 12px;
    color: var(--ink-500);
    padding-bottom: 8px;
  }

  .compare-row {
    display: flex;
    align-items: center;
    gap: 6px;
    border-bottom: 1px solid var(--hairline);
  }

  .compare-main {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;
    min-height: 60px;
    padding: 8px 0;
    text-align: left;
  }

  /* amostra dupla: metade âncora, metade candidata — raio 0 (cor chapada) */
  .compare-pair {
    display: inline-flex;
    width: 44px;
    height: 34px;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.12);
    flex-shrink: 0;
  }

  .compare-pair span {
    display: block;
    width: 50%;
    height: 100%;
  }

  .compare-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .compare-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .compare-meta {
    font-size: 11.5px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .compare-delta {
    flex-shrink: 0;
    font-size: 21px;
    font-weight: 600;
    color: var(--grafite);
  }

  .compare-delta.good {
    color: var(--laca);
  }

  .row-trash {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-pill);
    color: var(--ink-500);
    flex-shrink: 0;
  }

  .row-trash:active {
    color: var(--lacquer-deep);
  }

  .delta-link {
    display: block;
    margin: 14px auto 0;
    min-height: 44px; /* alvo de toque */
    padding: 0 12px;
    font-size: 12px;
    color: var(--ink-500);
  }

  /* ── Sobre ── */
  .about {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 8px 0 4px;
  }

  .about-name {
    font-size: 24px;
    font-weight: 750;
    color: var(--grafite);
    margin-top: 10px;
  }

  .about-tag {
    font-size: 11px;
    letter-spacing: 0.1em;
    color: var(--ink-500);
    margin-top: 2px;
  }

  .about-note {
    font-size: 13.5px;
    color: var(--ink-500);
    margin-top: 12px;
    max-width: 280px;
    line-height: 1.5;
  }
</style>
