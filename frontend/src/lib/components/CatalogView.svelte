<script lang="ts">
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Select, { Option } from '@smui/select';
  import Dialog, { Content as DialogContent } from '@smui/dialog';
  import Icon from './Icon.svelte';
  import PaintCard from './PaintCard.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

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
  });

  function openDetail(paint: Paint) {
    selectedPaint = paint;
    dialogOpen = true;
  }
</script>

<div class="page-container">
  <!-- Header -->
  <div class="page-header animate-rise">
    <h1 class="page-title">Catálogo de tintas</h1>
    <p class="page-subtitle">{filtered.length} de {paints.length} tintas cadastradas</p>
    <div class="page-divider"></div>
  </div>

  <!-- Filters -->
  <div class="panel p-3 flex gap-3 mb-6 animate-rise" style="animation-delay: 80ms; position: relative; z-index: 1;">
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
      {#each filtered as paint, i (paint.id)}
        <div class="animate-rise" style="animation-delay: {Math.min(i * 20, 200)}ms;">
          <PaintCard {paint} onclick={() => openDetail(paint)} />
        </div>
      {/each}
    </div>

    {#if filtered.length === 0}
      <div style="text-align: center; padding: 80px 0;">
        <div style="color: var(--ink-600); display: flex; justify-content: center; margin-bottom: 16px;"><Icon name="search-off" size={40} /></div>
        <p class="font-medium" style="color: var(--ink-500);">Nenhuma tinta encontrada</p>
      </div>
    {/if}
  {/if}
</div>

<!-- Detail Dialog -->
<Dialog bind:open={dialogOpen} surface$style="background: var(--ink-900); border: 1px solid var(--ink-700); border-radius: 10px; max-width: 560px; width: 100%;">
  {#if selectedPaint}
    <!-- Color hero -->
    <div class="color-hero" style="background: rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b});">
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
        <span class="font-mono text-xs font-medium" style="padding: 4px 10px; border-radius: 5px; background: var(--ink-800); color: var(--lacquer-tint);">{selectedPaint.code}</span>
      </div>

      <div class="grid-2">
        <div class="panel p-3">
          <div class="detail-label">RGB</div>
          <div class="font-mono text-sm text-white">{selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}</div>
        </div>
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
    </DialogContent>
  {/if}
</Dialog>

<style>
  .color-hero {
    width: 100%;
    height: 128px;
    position: relative;
  }

  .close-btn {
    position: absolute;
    top: 12px;
    right: 12px;
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

  :global(.mdc-dialog__surface) {
    border-radius: 10px !important;
    overflow: hidden;
  }
</style>
