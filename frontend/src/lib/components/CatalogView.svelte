<script lang="ts">
  import { onMount } from 'svelte';
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

  onMount(async () => {
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
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
</script>

<div class="min-h-full p-8">
  <!-- Header -->
  <div class="mb-6 animate-fadeIn">
    <h1 class="text-2xl font-bold text-white tracking-tight mb-1">Catálogo de Tintas</h1>
    <p class="text-sm" style="color: var(--color-surface-400);">
      {paints.length} tintas cadastradas
    </p>
  </div>

  <!-- Filters -->
  <div class="flex gap-3 mb-6 animate-fadeIn" style="animation-delay: 100ms;">
    <div class="flex-1 relative">
      <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="color: var(--color-surface-500);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Buscar por nome, código ou fabricante..."
        class="w-full pl-10 pr-4 py-2.5 rounded-lg text-sm text-white placeholder-surface-500 outline-none focus:ring-1 focus:ring-accent-500/30 transition-all"
        style="background: var(--color-surface-800); border: 1px solid var(--color-glass-border);"
      />
    </div>
    <select
      bind:value={selectedManufacturer}
      class="px-4 py-2.5 rounded-lg text-sm text-white outline-none appearance-none cursor-pointer min-w-[180px]"
      style="background: var(--color-surface-800); border: 1px solid var(--color-glass-border);"
    >
      <option value="">Todos os fabricantes</option>
      {#each manufacturers as mfr}
        <option value={mfr.name}>{mfr.name}</option>
      {/each}
    </select>
  </div>

  <!-- Grid -->
  {#if loading}
    <div class="grid grid-cols-4 gap-4">
      {#each Array(12) as _}
        <div class="glass rounded-xl p-4 animate-pulse">
          <div class="w-full aspect-square rounded-lg bg-white/5 mb-3"></div>
          <div class="h-4 w-3/4 bg-white/5 rounded mb-2"></div>
          <div class="h-3 w-1/2 bg-white/5 rounded"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="grid grid-cols-4 gap-4">
      {#each filtered as paint, i (paint.id)}
        <div class="animate-fadeIn" style="animation-delay: {Math.min(i * 30, 300)}ms;">
          <PaintCard {paint} onclick={() => selectedPaint = paint} />
        </div>
      {/each}
    </div>

    {#if filtered.length === 0}
      <div class="text-center py-16">
        <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-surface-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <p class="text-sm" style="color: var(--color-surface-500);">Nenhuma tinta encontrada</p>
      </div>
    {/if}
  {/if}
</div>

<!-- Detail Modal -->
{#if selectedPaint}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fadeIn" onclick={() => selectedPaint = null}>
    <div class="glass rounded-2xl p-6 max-w-lg w-full mx-4 animate-scaleIn" onclick={(e) => e.stopPropagation()}>
      <div class="flex items-start justify-between mb-4">
        <div>
          <h2 class="text-lg font-bold text-white">{selectedPaint.name}</h2>
          <p class="text-sm" style="color: var(--color-surface-400);">{selectedPaint.manufacturer} · {selectedPaint.productLine}</p>
        </div>
        <button onclick={() => selectedPaint = null} class="p-1 rounded-lg hover:bg-white/5 transition-colors">
          <svg class="w-5 h-5 text-surface-400" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="flex gap-4 mb-4">
        <div class="w-24 h-24 rounded-xl flex-shrink-0 shadow-lg"
          style="background: rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}); border: 1px solid rgba(255,255,255,0.1);">
        </div>
        <div class="flex-1 space-y-2">
          <div class="flex items-center gap-2">
            <span class="text-xs font-mono px-2 py-0.5 rounded" style="background: var(--color-surface-800); color: var(--color-accent-400);">{selectedPaint.code}</span>
          </div>
          <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
            <div><span style="color: var(--color-surface-500);">RGB:</span> <span class="font-mono text-surface-300">{selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}</span></div>
            {#if selectedPaint.finishType}<div><span style="color: var(--color-surface-500);">Acabamento:</span> <span class="text-surface-300">{selectedPaint.finishType}</span></div>{/if}
            {#if selectedPaint.paintType}<div><span style="color: var(--color-surface-500);">Tipo:</span> <span class="text-surface-300">{selectedPaint.paintType}</span></div>{/if}
            {#if selectedPaint.coverage}<div><span style="color: var(--color-surface-500);">Cobertura:</span> <span class="text-surface-300">{selectedPaint.coverage}</span></div>{/if}
            {#if selectedPaint.opacity}<div><span style="color: var(--color-surface-500);">Opacidade:</span> <span class="text-surface-300">{selectedPaint.opacity}</span></div>{/if}
            {#if selectedPaint.volume}<div><span style="color: var(--color-surface-500);">Volume:</span> <span class="text-surface-300">{selectedPaint.volume}</span></div>{/if}
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
