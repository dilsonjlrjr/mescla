<script lang="ts">
  // Catálogo — lista agrupada por fabricante (labels mono como fichas de
  // arquivo), VIRTUALIZADA (11.932 tintas: headers e linhas viram "rows" de
  // altura fixa na mesma VirtualList). Busca-first, filtro de marca no sheet,
  // detalhe em bottom sheet com o corredor "Gerar fórmula equivalente".
  import Icon from '../components/Icon.svelte';
  import BrandMark from '../components/BrandMark.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import VirtualList from '../components/VirtualList.svelte';
  import BrandSheet from '../components/BrandSheet.svelte';
  import PaintDetailSheet from '../components/PaintDetailSheet.svelte';
  import { switchTab } from '../nav.svelte';
  import { allManufacturers, allPaints, searchPaints, type Manufacturer, type Paint } from '../services/catalog';
  import { stock } from '../services/stock.svelte';
  import { appState } from '../appState.svelte';

  let query = $state('');
  let selectedMfr: Manufacturer | null = $state(null);
  let brandSheetOpen = $state(false);
  let detailPaint: Paint | null = $state(null);

  // Deep-link interno: catálogo já filtrado pela marca.
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

  // Selo "no estoque": match por fabricante + código.
  let stockKeys = $derived(
    new Set(stock.paints.filter(sp => sp.code !== '').map(sp => `${sp.manufacturer}|${sp.code}`))
  );

  // Agrupamento por fabricante em "rows" virtuais de altura única:
  // header = ficha `{FABRICANTE} · {n}`, item = linha de tinta.
  type Row = { kind: 'header'; label: string; count: number } | { kind: 'paint'; p: Paint };

  let rows = $derived.by(() => {
    const out: Row[] = [];
    let currentMfr = '';
    let headerIdx = -1;
    for (const p of filtered) {
      if (p.manufacturer !== currentMfr) {
        currentMfr = p.manufacturer;
        headerIdx = out.length;
        out.push({ kind: 'header', label: p.manufacturer, count: 0 });
      }
      (out[headerIdx] as { kind: 'header'; label: string; count: number }).count++;
      out.push({ kind: 'paint', p });
    }
    return out;
  });

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
    <div class="cat-head">
      <span class="cat-brand">
        <BrandMark size={26} />
        <h1 class="cat-title font-display">Catálogo</h1>
      </span>
      <span class="cat-count font-mono">{filtered.length.toLocaleString('pt-BR')}</span>
    </div>

    <input
      bind:value={query}
      type="search"
      class="cat-search"
      placeholder="buscar por nome, código ou cor…"
      aria-label="Buscar no catálogo"
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
      spellcheck="false"
      enterkeyhint="search"
    />

    <div class="cat-chips">
      <button class="cat-chip pressable" class:active={selectedMfr === null} onclick={() => (brandSheetOpen = true)}>
        {selectedMfr ? selectedMfr.name : 'Todas'}
        <Icon name="chevron-down" size={14} />
      </button>
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
      <VirtualList {rows} rowHeight={64}>
        {#snippet row(item: Row)}
          {#if item.kind === 'header'}
            <div class="cat-group section-label">{item.label} · {item.count.toLocaleString('pt-BR')}</div>
          {:else}
            {@const p = item.p}
            <button class="cat-row pressable" onclick={() => (detailPaint = p)}>
              <span class="cat-row-swatch" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
              <PaintBottle r={p.r} g={p.g} b={p.b} size={42} />
              <span class="cat-row-text">
                <span class="cat-row-name">{p.name}</span>
                <span class="cat-row-meta font-mono">
                  {p.code}{#if stockKeys.has(`${p.manufacturer}|${p.code}`)}<span class="cat-row-stock">&nbsp;·&nbsp;no estoque</span>{/if}
                </span>
              </span>
              <span class="cat-row-chev"><Icon name="chevron-right" size={16} /></span>
            </button>
          {/if}
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
    padding: 8px 16px 12px;
    background: var(--papel);
    border-bottom: 1px solid var(--hairline);
  }

  .cat-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 48px;
  }

  .cat-brand {
    display: inline-flex;
    align-items: center;
    gap: 10px;
  }

  .cat-title {
    font-size: 20px;
    font-weight: 750;
    color: var(--grafite);
  }

  .cat-count {
    font-size: 13px;
    color: var(--ink-500);
  }

  .cat-search {
    width: 100%;
    min-height: 50px;
    padding: 0 14px;
    font: inherit;
    font-size: 16px;
    background: var(--bancada);
  }

  .cat-chips {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    margin-top: 10px;
    padding-bottom: 2px;
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
    border-radius: var(--radius-pill);
    border: 1px solid var(--hairline);
    background: var(--papel);
    font-size: 13.5px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .cat-chip.active {
    background: var(--grafite);
    border-color: var(--grafite);
    color: var(--papel);
  }

  .cat-chip.clear {
    color: var(--lacquer-deep);
    border-style: dashed;
  }

  .cat-list {
    flex: 1;
    min-height: 0;
  }

  /* Ficha de arquivo do fabricante — alinhada na base da "row" virtual. */
  .cat-group {
    display: flex;
    align-items: flex-end;
    height: 100%;
    padding: 0 16px 8px;
  }

  .cat-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    height: 100%;
    padding: 0 16px;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .cat-row-swatch {
    width: 48px;
    height: 42px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
    flex-shrink: 0;
  }

  .cat-row-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .cat-row-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cat-row-meta {
    font-size: 12px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cat-row-stock {
    color: var(--laca);
    font-weight: 600;
  }

  .cat-row-chev {
    display: flex;
    color: var(--hairline);
    flex-shrink: 0;
  }
</style>
