<script lang="ts">
  // Catálogo — busca-first, grid de 2 colunas virtualizado (11.932 tintas),
  // filtro de marca em chips horizontais + sheet, detalhe em bottom sheet com
  // o corredor "Mesclar esta cor".
  import Icon from '../components/Icon.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import VirtualList from '../components/VirtualList.svelte';
  import BrandSheet from '../components/BrandSheet.svelte';
  import PaintDetailSheet from '../components/PaintDetailSheet.svelte';
  import { switchTab } from '../nav.svelte';
  import { allManufacturers, allPaints, searchPaints, type Manufacturer, type Paint } from '../services/catalog';
  import { shelf } from '../services/shelf.svelte';
  import { appState } from '../appState.svelte';

  let query = $state('');
  let selectedMfr: Manufacturer | null = $state(null);
  let brandSheetOpen = $state(false);
  let detailPaint: Paint | null = $state(null);

  // Deep-link interno: Mais → Marcas → catálogo já filtrado pela marca.
  $effect(() => {
    if (appState.pendingCatalogMfrId != null) {
      selectedMfr = allManufacturers().find(m => m.id === appState.pendingCatalogMfrId) ?? null;
      appState.pendingCatalogMfrId = null;
      query = '';
    }
  });

  // Debounce leve do campo (o filtro roda sobre 11k itens)
  let debounced = $state('');
  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    const q = query;
    clearTimeout(timer);
    timer = setTimeout(() => (debounced = q), 120);
  });

  let filtered = $derived.by(() => {
    if (debounced.trim()) {
      return searchPaints(debounced, { manufacturerId: selectedMfr?.id, limit: 100000 });
    }
    const all = allPaints();
    return selectedMfr ? all.filter(p => p.manufacturerId === selectedMfr!.id) : all;
  });

  // Grid 2 colunas: a VirtualList trabalha por LINHA — agrupa em pares.
  let rows = $derived.by(() => {
    const out: Paint[][] = [];
    for (let i = 0; i < filtered.length; i += 2) {
      out.push(filtered.slice(i, i + 2));
    }
    return out;
  });

  let shelfMfrs = $derived(allManufacturers().filter(m => shelf.manufacturerIds.includes(m.id)));

  function pickChip(m: Manufacturer | null) {
    selectedMfr = m;
  }

  function mesclarFrom(paint: Paint) {
    detailPaint = null;
    appState.pendingMesclarPaint = paint;
    switchTab('mesclar');
  }
</script>

<div class="cat">
  <div class="cat-top">
    <h1 class="screen-title">Catálogo</h1>
    <p class="cat-count font-mono">{filtered.length.toLocaleString('pt-BR')} tintas</p>

    <div class="cat-search">
      <Icon name="search" size={18} />
      <input
        bind:value={query}
        type="search"
        placeholder="Nome, código ou marca…"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        enterkeyhint="search"
      />
      {#if query}
        <button class="cat-clear pressable" onclick={() => (query = '')} aria-label="Limpar busca">
          <Icon name="close" size={16} />
        </button>
      {/if}
    </div>

    <div class="cat-chips">
      <button class="cat-chip pressable" class:active={selectedMfr === null} onclick={() => (brandSheetOpen = true)}>
        {selectedMfr ? selectedMfr.name : 'Todas'}
        <Icon name="chevron-down" size={14} />
      </button>
      {#each shelfMfrs as m (m.id)}
        {#if selectedMfr?.id !== m.id}
          <button class="cat-chip pressable" onclick={() => pickChip(m)}>{m.name}</button>
        {/if}
      {/each}
      {#if selectedMfr}
        <button class="cat-chip clear pressable" onclick={() => pickChip(null)}>
          <Icon name="close" size={13} />
          limpar filtro
        </button>
      {/if}
    </div>
  </div>

  <div class="cat-list">
    {#if filtered.length === 0}
      <div class="empty-state">
        <span class="empty-icon"><Icon name="search-off" size={36} /></span>
        <p class="empty-title">Nada com esse nome</p>
        <p class="empty-hint">Tente o código do pote (ex.: 70.951) ou limpe o filtro de marca.</p>
      </div>
    {:else}
      <VirtualList {rows} rowHeight={172}>
        {#snippet row(pair: Paint[])}
          <div class="cat-row">
            {#each pair as p (p.id)}
              <button class="cat-card pressable" onclick={() => (detailPaint = p)}>
                <span class="cat-card-swatch">
                  <span class="cat-card-color" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
                  <span class="cat-card-bottle"><PaintBottle r={p.r} g={p.g} b={p.b} size={68} /></span>
                </span>
                <span class="cat-card-name">{p.name}</span>
                <span class="cat-card-meta"><span class="font-mono">{p.code}</span> · {p.manufacturer}</span>
              </button>
            {/each}
            {#if pair.length === 1}
              <span aria-hidden="true"></span>
            {/if}
          </div>
        {/snippet}
      </VirtualList>
    {/if}
  </div>
</div>

<BrandSheet
  open={brandSheetOpen}
  onClose={() => (brandSheetOpen = false)}
  mode="single"
  selectedId={selectedMfr?.id ?? null}
  onPick={pickChip}
/>
<PaintDetailSheet paint={detailPaint} onClose={() => (detailPaint = null)} onMesclar={mesclarFrom} />

<style>
  .cat {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .cat-top {
    flex-shrink: 0;
    padding: 16px 16px 10px;
    background: var(--ink-950);
    border-bottom: 1px solid var(--ink-700);
  }

  .cat-count {
    font-size: 12.5px;
    color: var(--ink-500);
    margin: 2px 0 12px;
  }

  .cat-search {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 52px;
    padding: 0 14px;
    border: 1px solid var(--ink-600);
    border-radius: 12px;
    background: var(--ink-850);
    color: var(--ink-500);
  }

  .cat-search:focus-within {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .cat-search input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    box-shadow: none;
    background: transparent;
    font: inherit;
    font-size: 16px;
    color: var(--ink-100);
  }

  .cat-clear {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    color: var(--ink-500);
    flex-shrink: 0;
  }

  .cat-chips {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    margin-top: 10px;
    padding-bottom: 4px;
    scrollbar-width: none;
  }

  .cat-chips::-webkit-scrollbar {
    display: none;
  }

  .cat-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 40px;
    padding: 0 14px;
    border-radius: 999px;
    border: 1px solid var(--ink-700);
    background: var(--ink-900);
    font-size: 13.5px;
    font-weight: 600;
    color: var(--ink-300);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .cat-chip.active {
    background: var(--paper);
    border-color: var(--paper);
    color: var(--ink-950);
  }

  .cat-chip.clear {
    color: var(--lacquer-deep);
    border-style: dashed;
  }

  .cat-list {
    flex: 1;
    min-height: 0;
  }

  .cat-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    padding: 5px 16px;
    height: 100%;
  }

  /* Cartela de tinta: cor chapada honesta (a tinta real, sem lavar) com o
     furo de catálogo da assinatura, código como etiqueta impressa abaixo. */
  .cat-card {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-surface);
    background: var(--ink-900);
    overflow: hidden;
    text-align: left;
    min-width: 0;
  }

  /* Topo do card: sample da cor chapada + a garrafinha do lado. */
  .cat-card-swatch {
    position: relative;
    display: grid;
    grid-template-columns: 1.35fr 1fr;
    height: 96px;
    flex-shrink: 0;
  }

  .cat-card-color {
    display: block;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink-100) 12%, transparent);
  }

  .cat-card-bottle {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--ink-850);
    border-left: 1px solid var(--ink-700);
  }

  /* furo de catálogo — sempre sobre a COR */
  .cat-card-swatch::after {
    content: '';
    position: absolute;
    top: 6px;
    left: 6px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--ink-950);
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.35);
  }

  .cat-card-name {
    font-size: 13.5px;
    font-weight: 600;
    color: var(--ink-100);
    padding: 9px 11px 0;
    line-height: 1.25;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .cat-card-meta {
    font-size: 11px;
    color: var(--ink-500);
    padding: 3px 11px 9px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
