<script lang="ts">
  // Catálogo (Tintômetro): arquivo de museu — índice No., ordem por matiz,
  // pills de filtro por fabricante e rodapé com contagem mono.
  import { onMount } from 'svelte';
  import Dialog, { Content as DialogContent } from '@smui/dialog';
  import PaintCard from './PaintCard.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import { hueOf, hexOf, contrastOn, paintMatches } from '../ui';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface Props {
    onNavigate: (view: View, paintId?: number) => void;
    initialManufacturer?: string | null;
    initialPaintId?: number | null;
  }

  let { onNavigate, initialManufacturer = null, initialPaintId = null }: Props = $props();

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    productLine: string;
    r: number;
    g: number;
    b: number;
    finishType: string;
    paintType: string;
    coverage: string;
    opacity: string;
    volume: string;
  }

  let paints: Paint[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  // svelte-ignore state_referenced_locally -- captura intencional do valor inicial
  let selectedBrands: string[] = $state(initialManufacturer ? [initialManufacturer] : []);
  let manufacturers: { id: number; name: string }[] = $state([]);
  let selectedPaint: Paint | null = $state(null);
  let dialogOpen = $state(false);
  let sortBy: 'hue' | 'name' | 'code' = $state('hue');

  onMount(async () => {
    try {
      const [allPaints, mfrs] = await Promise.all([
        PaintService.GetAllPaints(),
        PaintService.GetManufacturers(),
      ]);
      paints = allPaints || [];
      manufacturers = mfrs || [];
      // ficha aberta direto da busca global
      if (initialPaintId) {
        const p = paints.find(x => x.id === initialPaintId);
        if (p) openDetail(p);
      }
    } catch (e) {
      console.error('Erro carregando tintas:', e);
    } finally {
      loading = false;
    }
  });

  // Paginação incremental — 11 mil cards de uma vez travam o webview.
  const PAGE = 60;
  let visibleCount = $state(PAGE);

  let filtered = $derived.by(() => {
    let result = paints;
    if (searchQuery) {
      result = result.filter(p => paintMatches(p, searchQuery));
    }
    if (selectedBrands.length > 0) {
      result = result.filter(p => selectedBrands.includes(p.manufacturer));
    }
    const sorted = [...result];
    if (sortBy === 'hue') sorted.sort((a, b) => hueOf(a.r, a.g, a.b) - hueOf(b.r, b.g, b.b));
    else if (sortBy === 'name') sorted.sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'));
    else sorted.sort((a, b) => (a.code || '').localeCompare(b.code || '', 'pt-BR'));
    return sorted;
  });

  $effect(() => {
    // qualquer mudança de filtro/ordem volta pra primeira página
    searchQuery; selectedBrands; sortBy;
    visibleCount = PAGE;
  });

  let visible = $derived(filtered.slice(0, visibleCount));

  function openDetail(paint: Paint) {
    selectedPaint = paint;
    dialogOpen = true;
  }

  function goToRecipe(paint: Paint) {
    dialogOpen = false;
    onNavigate('mix', paint.id);
  }

  function copyHex(p: Paint) {
    navigator.clipboard.writeText(hexOf(p.r, p.g, p.b));
    toast(`${hexOf(p.r, p.g, p.b)} copiado`);
  }

  // Multi-seleção: cada pill liga/desliga; "Todas" limpa o conjunto.
  function pickBrand(name: string) {
    selectedBrands = selectedBrands.includes(name)
      ? selectedBrands.filter(b => b !== name)
      : [...selectedBrands, name];
  }
</script>

<div class="page-container">
  <!-- Header: título + contagem mono à esquerda, pills de filtro à direita -->
  <div class="catalog-head animate-rise">
    <div>
      <h1 class="page-title">Catálogo</h1>
      <p class="catalog-count font-mono">
        {paints.length.toLocaleString('pt-BR')} tintas&nbsp;&nbsp;·&nbsp;&nbsp;<button class="mfr-link" onclick={() => onNavigate('manufacturers')} title="Ver fabricantes">{manufacturers.length} fabricantes</button>
      </p>
    </div>
    <div class="brand-pills">
      <button class="filter-pill" class:active={selectedBrands.length === 0} onclick={() => (selectedBrands = [])}>Todas</button>
      {#each manufacturers as mfr (mfr.id)}
        <button
          class="filter-pill"
          class:active={selectedBrands.includes(mfr.name)}
          aria-pressed={selectedBrands.includes(mfr.name)}
          onclick={() => pickBrand(mfr.name)}
        >{mfr.name}</button>
      {/each}
    </div>
  </div>

  <div class="catalog-search animate-rise" style="animation-delay: 60ms;">
    <input
      type="search"
      bind:value={searchQuery}
      placeholder="Buscar por nome, código ou fabricante"
      aria-label="Buscar tinta"
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
      spellcheck="false"
    />
  </div>

  <!-- Grid -->
  {#if loading}
    <div class="catalog-grid">
      {#each Array(12) as _, i (i)}
        <div>
          <div class="skeleton" style="aspect-ratio: 4/3; margin-bottom: 10px;"></div>
          <div class="skeleton" style="height: 12px; width: 75%; margin-bottom: 6px;"></div>
          <div class="skeleton" style="height: 10px; width: 50%;"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="catalog-grid">
      {#each visible as paint, i (paint.id)}
        <div class="animate-rise" style="animation-delay: {Math.min(i * 15, 180)}ms;">
          <PaintCard {paint} index={i + 1} onclick={() => openDetail(paint)} />
        </div>
      {/each}
    </div>

    {#if visibleCount < filtered.length}
      <div class="load-more">
        <button class="pill-light" onclick={() => (visibleCount += PAGE * 2)}>
          Mostrar mais {Math.min(PAGE * 2, filtered.length - visibleCount)} tintas
        </button>
      </div>
    {/if}

    {#if filtered.length === 0}
      <div class="empty-state">
        <p class="empty-title">Nada com esse nome</p>
        <p class="empty-hint">Tente o código do pote (ex.: 70.951) ou só parte do nome. Ou limpe o filtro de marca.</p>
      </div>
    {/if}

    <!-- Rodapé: contagem à esquerda, ordenação à direita -->
    <div class="catalog-foot">
      <span class="font-mono foot-count">Mostrando {Math.min(visibleCount, filtered.length)} de {filtered.length.toLocaleString('pt-BR')}</span>
      <label class="foot-sort">
        <span class="foot-sort-label">Ordenar:</span>
        <select bind:value={sortBy} aria-label="Ordenar catálogo">
          <option value="hue">matiz</option>
          <option value="name">nome</option>
          <option value="code">código</option>
        </select>
      </label>
    </div>
  {/if}
</div>

<!-- Ficha da tinta -->
<Dialog bind:open={dialogOpen} surface$style="background: var(--papel); border-radius: var(--radius-surface); max-width: 560px; width: 100%;">
  {#if selectedPaint}
    <div class="detail-hero" style="background: rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b});">
      <span class="detail-hero-hex font-mono" style="color: {contrastOn(selectedPaint.r, selectedPaint.g, selectedPaint.b)};">{hexOf(selectedPaint.r, selectedPaint.g, selectedPaint.b)}</span>
      <span class="detail-bottle">
        <PaintBottle r={selectedPaint.r} g={selectedPaint.g} b={selectedPaint.b} size={104} />
      </span>
      <button class="detail-close" style="color: {contrastOn(selectedPaint.r, selectedPaint.g, selectedPaint.b)};" onclick={() => (dialogOpen = false)} aria-label="Fechar">×</button>
    </div>

    <DialogContent>
      <div class="detail-body">
        <h2 class="detail-name font-display">{selectedPaint.name}</h2>
        <p class="detail-sub">{selectedPaint.manufacturer}{selectedPaint.productLine ? ` · ${selectedPaint.productLine}` : ''}</p>

        <div class="detail-grid font-mono">
          <button class="detail-cell click" onclick={() => copyHex(selectedPaint!)} title="Copiar hex">
            <span class="detail-label">Hex · copiar</span>
            <span class="detail-value">{hexOf(selectedPaint.r, selectedPaint.g, selectedPaint.b)}</span>
          </button>
          <div class="detail-cell">
            <span class="detail-label">RGB</span>
            <span class="detail-value">{selectedPaint.r} {selectedPaint.g} {selectedPaint.b}</span>
          </div>
          {#if selectedPaint.code}
            <div class="detail-cell">
              <span class="detail-label">Código</span>
              <span class="detail-value">{selectedPaint.code}</span>
            </div>
          {/if}
          {#if selectedPaint.finishType}
            <div class="detail-cell">
              <span class="detail-label">Acabamento</span>
              <span class="detail-value">{selectedPaint.finishType}</span>
            </div>
          {/if}
          {#if selectedPaint.paintType}
            <div class="detail-cell">
              <span class="detail-label">Tipo</span>
              <span class="detail-value">{selectedPaint.paintType}</span>
            </div>
          {/if}
          {#if selectedPaint.volume}
            <div class="detail-cell">
              <span class="detail-label">Volume</span>
              <span class="detail-value">{selectedPaint.volume}</span>
            </div>
          {/if}
        </div>

        <button class="pill-dark w-full" onclick={() => goToRecipe(selectedPaint!)}>
          Buscar receita equivalente
        </button>
      </div>
    </DialogContent>
  {/if}
</Dialog>

<style>
  .catalog-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 28px;
    margin-bottom: 24px;
  }

  .catalog-count {
    font-size: 12.5px;
    color: var(--text-2);
    margin-top: 4px;
  }

  .mfr-link {
    font: inherit;
    color: inherit;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    text-decoration: underline dotted;
    text-underline-offset: 3px;
  }

  .mfr-link:hover {
    color: var(--laca-deep);
  }

  .brand-pills {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
    max-width: 60%;
  }

  .catalog-search {
    margin-bottom: 32px;
  }

  .catalog-search input {
    width: min(420px, 100%);
    height: 42px;
    padding: 0 16px;
    font: inherit;
    font-size: 13.5px;
  }

  .catalog-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 26px 20px;
  }

  .catalog-grid > * {
    min-width: 0;
  }

  .load-more {
    display: flex;
    justify-content: center;
    margin-top: 32px;
  }

  .catalog-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 40px;
    padding-top: 16px;
    border-top: 1px solid var(--hairline);
  }

  .foot-count {
    font-size: 12px;
    color: var(--text-2);
  }

  .foot-sort {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--grafite);
  }

  .foot-sort-label {
    color: var(--text-2);
  }

  .foot-sort select {
    appearance: none;
    -webkit-appearance: none;
    border: none;
    background: transparent;
    font: inherit;
    font-weight: 700;
    color: var(--grafite);
    cursor: pointer;
    padding-right: 16px;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='8' height='5' viewBox='0 0 8 5'%3E%3Cpath d='M1 1l3 3 3-3' fill='none' stroke='%231a1712' stroke-width='1.4' stroke-linecap='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right center;
  }

  /* Ficha */
  .detail-hero {
    position: relative;
    width: 100%;
    height: 148px;
  }

  .detail-hero-hex {
    position: absolute;
    bottom: 14px;
    left: 20px;
    font-size: 12px;
    opacity: 0.92;
  }

  .detail-bottle {
    position: absolute;
    right: 26px;
    bottom: -1px;
    display: flex;
  }

  .detail-close {
    position: absolute;
    top: 10px;
    left: 14px;
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: none;
    font-size: 22px;
    line-height: 1;
    cursor: pointer;
    opacity: 0.8;
  }

  .detail-close:hover {
    opacity: 1;
  }

  .detail-body {
    padding: 8px 4px 4px;
  }

  .detail-name {
    font-size: 28px;
    font-weight: 720;
    color: var(--grafite);
    margin-bottom: 4px;
  }

  .detail-sub {
    font-size: 13.5px;
    color: var(--text-2);
    margin-bottom: 22px;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0;
    border-top: 1px solid var(--hairline);
    border-bottom: 1px solid var(--hairline);
    margin-bottom: 22px;
  }

  .detail-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 14px 12px 14px 0;
    border: none;
    background: none;
    text-align: left;
    font: inherit;
  }

  .detail-cell.click {
    cursor: pointer;
  }

  .detail-cell.click:hover .detail-value {
    color: var(--laca-deep);
  }

  .detail-label {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.14em;
    color: var(--text-2);
  }

  .detail-value {
    font-size: 13px;
    color: var(--grafite);
  }

  :global(.mdc-dialog__surface) {
    border-radius: var(--radius-surface) !important;
    overflow: hidden;
  }
</style>
