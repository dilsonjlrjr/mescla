<script lang="ts">
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Select, { Option } from '@smui/select';
  import Dialog, { Content as DialogContent, Title as DialogTitle } from '@smui/dialog';
  import IconButton from '@smui/icon-button';
  import PaintCard from './PaintCard.svelte';

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
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      const [allPaints, mfrs] = await Promise.all([
        wailsjs.GetAllPaints(),
        wailsjs.GetManufacturers(),
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
  <div class="page-header animate-artisan-fade">
    <h1 class="page-title">Catálogo de Tintas</h1>
    <p class="page-subtitle">{filtered.length} de {paints.length} tintas cadastradas</p>
    <div class="page-divider"></div>
  </div>

  <!-- Filters -->
  <div class="flex gap-3 mb-6 animate-artisan-fade" style="animation-delay: 80ms;">
    <div class="flex-1">
      <Textfield
        variant="outlined"
        bind:value={searchQuery}
        label="Buscar por nome, código ou fabricante..."
        style="width: 100%;"
      >
        {#snippet leadingIcon()}
          <span class="material-icons" style="color: var(--color-obsidian-500);">search</span>
        {/snippet}
      </Textfield>
    </div>
    <div style="min-width: 180px;">
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
        <div class="artisan-card overflow-hidden">
          <div class="skeleton w-full" style="aspect-ratio: 4/3; border-radius: 0;"></div>
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
        <div class="animate-artisan-fade" style="animation-delay: {Math.min(i * 20, 200)}ms;">
          <PaintCard {paint} onclick={() => openDetail(paint)} />
        </div>
      {/each}
    </div>

    {#if filtered.length === 0}
      <div style="text-align: center; padding: 80px 0;">
        <div class="stat-icon" style="width: 64px; height: 64px; margin: 0 auto 16px; background: var(--color-obsidian-800); border-radius: 16px; display: flex; align-items: center; justify-content: center;">
          <span class="material-icons" style="color: var(--color-obsidian-600); font-size: 32px;">search_off</span>
        </div>
        <p class="font-medium" style="color: var(--color-obsidian-500);">Nenhuma tinta encontrada</p>
      </div>
    {/if}
  {/if}
</div>

<!-- Detail Dialog -->
<Dialog bind:open={dialogOpen} surface$style="background: var(--color-obsidian-900); border: 1px solid rgba(255,255,255,0.06); border-radius: 16px; max-width: 560px; width: 100%;">
  {#if selectedPaint}
    <!-- Color hero -->
    <div class="color-hero" style="background: linear-gradient(135deg, rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}), rgb({Math.max(0,selectedPaint.r-30)}, {Math.max(0,selectedPaint.g-30)}, {Math.max(0,selectedPaint.b-30)}));">
      <div class="color-hero-overlay"></div>
      <IconButton onclick={() => dialogOpen = false} style="position: absolute; top: 12px; right: 12px; color: rgba(255,255,255,0.7);">
        <span class="material-icons">close</span>
      </IconButton>
    </div>

    <DialogContent>
      <div class="mb-4">
        <h2 class="font-display text-3xl font-bold text-white mb-1">{selectedPaint.name}</h2>
        <p style="font-size: 14px; color: var(--color-obsidian-400);">{selectedPaint.manufacturer} · {selectedPaint.productLine}</p>
      </div>

      <div class="flex items-center gap-2 mb-5">
        <span class="font-mono text-xs font-medium" style="padding: 4px 10px; border-radius: 8px; background: var(--color-obsidian-800); color: var(--color-amber-glow); border: 1px solid rgba(255,255,255,0.06);">{selectedPaint.code}</span>
      </div>

      <div class="grid-2">
        <div class="artisan-card p-3">
          <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">RGB</div>
          <div class="font-mono text-sm text-white">{selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}</div>
        </div>
        {#if selectedPaint.finishType}
          <div class="artisan-card p-3">
            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">Acabamento</div>
            <div class="text-sm text-white">{selectedPaint.finishType}</div>
          </div>
        {/if}
        {#if selectedPaint.paintType}
          <div class="artisan-card p-3">
            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">Tipo</div>
            <div class="text-sm text-white">{selectedPaint.paintType}</div>
          </div>
        {/if}
        {#if selectedPaint.coverage}
          <div class="artisan-card p-3">
            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">Cobertura</div>
            <div class="text-sm text-white">{selectedPaint.coverage}</div>
          </div>
        {/if}
        {#if selectedPaint.opacity}
          <div class="artisan-card p-3">
            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">Opacidade</div>
            <div class="text-sm text-white">{selectedPaint.opacity}</div>
          </div>
        {/if}
        {#if selectedPaint.volume}
          <div class="artisan-card p-3">
            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 0.12em; font-weight: 600; margin-bottom: 4px; color: var(--color-obsidian-500);">Volume</div>
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

  .color-hero-overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(0,0,0,0.3), transparent);
  }

  :global(.mdc-dialog__surface) {
    border-radius: 16px !important;
    overflow: hidden;
  }
</style>
