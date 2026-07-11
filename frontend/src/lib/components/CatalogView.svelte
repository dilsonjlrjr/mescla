<script lang="ts">
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Select, { Option } from '@smui/select';
  import Dialog, { Content as DialogContent } from '@smui/dialog';
  import Icon from './Icon.svelte';
  import PaintCard from './PaintCard.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix';

  interface Props {
    onNavigate: (view: View, paintId?: number) => void;
  }

  let { onNavigate }: Props = $props();

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    productLine: string;
    r: number;
    g: number;
    b: number;
    swatchPath: string;
    thumbnail: string;
    imageUrl: string;
    finishType: string;
    paintType: string;
    coverage: string;
    opacity: string;
    volume: string;
  }

  let paints: Paint[] = $state([]);
  let filtered: Paint[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedManufacturer = $state('');
  let manufacturers: { id: number; name: string }[] = $state([]);
  let selectedPaint: Paint | null = $state(null);
  let dialogOpen = $state(false);

  onMount(async () => {
    try {
      const [allPaints, mfrs] = await Promise.all([
        PaintService.GetAllPaints(),
        PaintService.GetManufacturers(),
      ]);
      paints = allPaints || [];
      filtered = paints;
      manufacturers = mfrs || [];
    } catch (e) {
      console.error('Erro carregando tintas:', e);
    } finally {
      loading = false;
    }
  });

  // Paginação incremental — 11 mil cards de uma vez travam o webview.
  const PAGE = 60;
  let visibleCount = $state(PAGE);

  $effect(() => {
    let result = paints;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      result = result.filter(p =>
        p.name.toLowerCase().includes(q) ||
        p.code.toLowerCase().includes(q) ||
        p.manufacturer.toLowerCase().includes(q)
      );
    }
    if (selectedManufacturer) {
      result = result.filter(p => p.manufacturer === selectedManufacturer);
    }
    filtered = result;
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

  function hexOf(p: Paint): string {
    const h = (n: number) => n.toString(16).padStart(2, '0').toUpperCase();
    return `${h(p.r)}${h(p.g)}${h(p.b)}`;
  }

  function copyHex(p: Paint) {
    navigator.clipboard.writeText(`#${hexOf(p)}`);
    toast(`#${hexOf(p)} copiado`);
  }
</script>

<div class="page-container">
  <!-- Header -->
  <div class="page-header animate-rise">
    <h1 class="page-title">Catálogo</h1>
    <p class="page-subtitle">{filtered.length.toLocaleString('pt-BR')} de {paints.length.toLocaleString('pt-BR')} tintas — clique numa tinta pra ver detalhes e pedir a equivalência</p>
    <div class="page-divider"></div>
  </div>

  <!-- Filters -->
  <div class="panel p-3 mb-6 animate-rise" style="animation-delay: 80ms; position: relative; z-index: 1;">
    <div class="flex gap-3">
      <div class="flex-1">
        <Textfield
          variant="outlined"
          bind:value={searchQuery}
          label="Buscar por nome, código ou fabricante..."
          style="width: 100%;"
        >
          {#snippet leadingIcon()}
            <span class="mdc-text-field__icon mdc-text-field__icon--leading" style="color: var(--ink-500); display: flex;"><Icon name="search" size={17} /></span>
          {/snippet}
        </Textfield>
      </div>
      <div style="min-width: 240px;">
        <Select variant="outlined" bind:value={selectedManufacturer} label="Fabricante" style="width: 100%;">
          <Option value="">Todos os fabricantes</Option>
          {#each manufacturers as mfr}
            <Option value={mfr.name}>{mfr.name}</Option>
          {/each}
        </Select>
      </div>
    </div>

    <!-- Filtros ativos como chips-pílula + contagem em etiqueta mono -->
    {#if searchQuery || selectedManufacturer}
      <div class="filter-meta">
        {#if searchQuery}
          <button class="filter-chip" onclick={() => searchQuery = ''} title="Limpar busca">
            "{searchQuery}" <Icon name="close" size={11} />
          </button>
        {/if}
        {#if selectedManufacturer}
          <button class="filter-chip" onclick={() => selectedManufacturer = ''} title="Limpar filtro de marca">
            {selectedManufacturer} <Icon name="close" size={11} />
          </button>
        {/if}
        <span class="count-tag font-mono">{filtered.length.toLocaleString('pt-BR')} de {paints.length.toLocaleString('pt-BR')}</span>
      </div>
    {/if}
  </div>

  <!-- Grid -->
  {#if loading}
    <div class="grid-5">
      {#each Array(15) as _}
        <div class="panel" style="overflow: hidden;">
          <div class="skeleton w-full" style="aspect-ratio: 1/1; border-radius: 0;"></div>
          <div class="p-3" style="display: flex; flex-direction: column; gap: 8px;">
            <div class="skeleton" style="height: 12px; width: 75%;"></div>
            <div class="skeleton" style="height: 10px; width: 50%;"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="grid-5">
      {#each visible as paint, i (paint.id)}
        <div class="animate-rise" style="animation-delay: {Math.min(i * 20, 200)}ms;">
          <PaintCard {paint} onclick={() => openDetail(paint)} />
        </div>
      {/each}
    </div>

    {#if visibleCount < filtered.length}
      <div style="display: flex; flex-direction: column; align-items: center; gap: 8px; margin-top: 28px;">
        <button class="btn-ghost" onclick={() => visibleCount += PAGE * 2}>
          Mostrar mais {Math.min(PAGE * 2, filtered.length - visibleCount)} tintas
        </button>
        <span class="font-mono" style="font-size: 11px; color: var(--ink-500);">exibindo {visibleCount} de {filtered.length.toLocaleString('pt-BR')}</span>
      </div>
    {/if}

    {#if filtered.length === 0}
      <div class="empty-state">
        <div class="empty-icon"><Icon name="search-off" size={40} /></div>
        <p class="empty-title">Nada com esse nome</p>
        <p class="empty-hint">Tente o código do pote (ex.: 70.951) ou só parte do nome — ou limpe o filtro de marca.</p>
      </div>
    {/if}
  {/if}
</div>

<!-- Detail Dialog -->
<Dialog bind:open={dialogOpen} surface$style="background: var(--ink-900); border: 1px solid var(--ink-700); border-radius: var(--radius-surface); max-width: 560px; width: 100%;">
  {#if selectedPaint}
    <!-- Color hero: a cor real da tinta, chapada, com o furo de catálogo -->
    <div class="color-hero" style="background: rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b});">
      <span class="hero-punch"></span>
      <span class="hero-bottle">
        <PaintBottle r={selectedPaint.r} g={selectedPaint.g} b={selectedPaint.b} size={106} label={selectedPaint.code} />
      </span>
      <button class="close-btn" onclick={() => dialogOpen = false} aria-label="Fechar">
        <Icon name="close" size={18} />
      </button>
    </div>

    <DialogContent>
      <div class="mb-4">
        <h2 class="font-display text-3xl font-bold text-white mb-1">{selectedPaint.name}</h2>
        <p style="font-size: 14px; color: var(--ink-500);">{selectedPaint.manufacturer} · {selectedPaint.productLine}</p>
      </div>

      <div class="flex items-center gap-2 mb-5">
        <span class="font-mono text-xs font-medium" style="padding: 4px 12px; border-radius: var(--radius-pill); background: var(--ink-800); color: var(--lacquer-tint);">{selectedPaint.code}</span>
      </div>

      <div class="grid-2">
        <button class="panel p-3 copy-cell" onclick={() => copyHex(selectedPaint!)} title="Copiar código hex">
          <div class="detail-label">Cor · clique pra copiar</div>
          <div class="font-mono text-sm text-white">#{hexOf(selectedPaint)} · {selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}</div>
        </button>
        {#if selectedPaint.finishType}
          <div class="panel p-3">
            <div class="detail-label">Acabamento</div>
            <div class="text-sm text-white">{selectedPaint.finishType}</div>
          </div>
        {/if}
        {#if selectedPaint.paintType}
          <div class="panel p-3">
            <div class="detail-label">Tipo</div>
            <div class="text-sm text-white">{selectedPaint.paintType}</div>
          </div>
        {/if}
        {#if selectedPaint.coverage}
          <div class="panel p-3">
            <div class="detail-label">Cobertura</div>
            <div class="text-sm text-white">{selectedPaint.coverage}</div>
          </div>
        {/if}
        {#if selectedPaint.opacity}
          <div class="panel p-3">
            <div class="detail-label">Opacidade</div>
            <div class="text-sm text-white">{selectedPaint.opacity}</div>
          </div>
        {/if}
        {#if selectedPaint.volume}
          <div class="panel p-3">
            <div class="detail-label">Volume</div>
            <div class="text-sm text-white">{selectedPaint.volume}</div>
          </div>
        {/if}
      </div>

      <button class="btn-primary" style="margin-top: 20px;" onclick={() => goToRecipe(selectedPaint!)}>
        <Icon name="flask" size={17} />
        Encontrar equivalência
      </button>
    </DialogContent>
  {/if}
</Dialog>

<style>
  .color-hero {
    width: 100%;
    height: 128px;
    position: relative;
  }

  /* furo de catálogo — assinatura do swatch chapado */
  .hero-punch {
    position: absolute;
    top: 12px;
    right: 12px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--ink-950);
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.35);
  }

  /* a garrafinha em pé na frente da parede pintada, apoiada na base do hero */
  .hero-bottle {
    position: absolute;
    right: 26px;
    bottom: -1px;
    display: flex;
    filter: drop-shadow(0 4px 10px rgba(0, 0, 0, 0.3));
  }

  /* fechar à esquerda pra não cobrir o furo (e casar com o padrão macOS) */
  .close-btn {
    position: absolute;
    top: 12px;
    left: 12px;
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: 50%;
    background: rgba(15, 13, 18, 0.4);
    color: rgba(255, 255, 255, 0.85);
    cursor: pointer;
  }

  .close-btn:hover {
    background: rgba(15, 13, 18, 0.6);
  }

  .detail-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-weight: 600;
    margin-bottom: 4px;
    color: var(--ink-500);
  }

  .copy-cell {
    font: inherit;
    text-align: left;
    cursor: pointer;
    transition: border-color 0.15s ease;
  }

  .copy-cell:hover {
    border-color: var(--lacquer);
  }

  /* Filtros ativos: pílulas removíveis + etiqueta mono de contagem */
  .filter-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    margin-top: 10px;
  }

  .filter-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 12px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-pill);
    background: var(--ink-850);
    color: var(--ink-300);
    font: inherit;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: border-color 0.15s ease, color 0.15s ease;
  }

  .filter-chip:hover {
    border-color: var(--lacquer);
    color: var(--lacquer-deep);
  }

  .count-tag {
    margin-left: auto;
    font-size: 11px;
    color: var(--ink-500);
  }

  :global(.mdc-dialog__surface) {
    border-radius: var(--radius-surface) !important;
    overflow: hidden;
  }
</style>
