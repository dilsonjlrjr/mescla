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

<div class="min-h-full p-8 relative z-10">
  <!-- Header -->
  <div class="mb-8 animate-artisan-fade">
    <h1 class="font-[family-name:var(--font-display)] text-3xl font-bold text-white tracking-tight mb-2">Catálogo de Tintas</h1>
    <p class="text-[13px]" style="color: var(--color-obsidian-400);">
      {filtered.length} de {paints.length} tintas cadastradas
    </p>
    <div class="mt-4 h-[1px]" style="background: linear-gradient(90deg, var(--color-amber-glow), transparent 50%); opacity: 0.15;"></div>
  </div>

  <!-- Filters -->
  <div class="flex gap-3 mb-6 animate-artisan-fade" style="animation-delay: 80ms;">
    <div class="flex-1 relative">
      <svg class="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4" style="color: var(--color-obsidian-500);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Buscar por nome, código ou fabricante..."
        class="artisan-input w-full pl-10"
      />
    </div>
    <select bind:value={selectedManufacturer} class="artisan-select min-w-[180px]">
      <option value="">Todos os fabricantes</option>
      {#each manufacturers as mfr}
        <option value={mfr.name}>{mfr.name}</option>
      {/each}
    </select>
  </div>

  <!-- Grid -->
  {#if loading}
    <div class="grid grid-cols-5 gap-4">
      {#each Array(15) as _}
        <div class="artisan-card overflow-hidden">
          <div class="skeleton w-full aspect-[4/3] rounded-none"></div>
          <div class="p-3.5 space-y-2">
            <div class="skeleton h-3 w-3/4"></div>
            <div class="skeleton h-2.5 w-1/2"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="grid grid-cols-5 gap-4">
      {#each filtered as paint, i (paint.id)}
        <div class="animate-artisan-fade" style="animation-delay: {Math.min(i * 20, 200)}ms;">
          <PaintCard {paint} onclick={() => selectedPaint = paint} />
        </div>
      {/each}
    </div>

    {#if filtered.length === 0}
      <div class="text-center py-20">
        <div class="w-16 h-16 mx-auto mb-4 rounded-2xl flex items-center justify-center" style="background: var(--color-obsidian-800);">
          <svg class="w-8 h-8" style="color: var(--color-obsidian-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium" style="color: var(--color-obsidian-500);">Nenhuma tinta encontrada</p>
      </div>
    {/if}
  {/if}
</div>

<!-- Detail Modal -->
{#if selectedPaint}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-md animate-artisan-fade" onclick={() => selectedPaint = null}>
    <div class="artisan-panel p-0 max-w-xl w-full mx-4 animate-artisan-scale overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <!-- Color hero -->
      <div class="w-full h-32 relative"
        style="background: linear-gradient(135deg, rgb({selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}), rgb({Math.max(0,selectedPaint.r-30)}, {Math.max(0,selectedPaint.g-30)}, {Math.max(0,selectedPaint.b-30)}));">
        <div class="absolute inset-0 bg-gradient-to-t from-black/30 to-transparent"></div>
        <button onclick={() => selectedPaint = null} class="absolute top-3 right-3 w-8 h-8 rounded-lg flex items-center justify-center bg-black/30 backdrop-blur-sm text-white/70 hover:text-white hover:bg-black/50 transition-all">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="p-6">
        <div class="mb-4">
          <h2 class="font-[family-name:var(--font-display)] text-2xl font-bold text-white mb-1">{selectedPaint.name}</h2>
          <p class="text-sm" style="color: var(--color-obsidian-400);">{selectedPaint.manufacturer} · {selectedPaint.productLine}</p>
        </div>

        <div class="flex items-center gap-2 mb-5">
          <span class="font-mono text-xs font-medium px-2.5 py-1 rounded-lg" style="background: var(--color-obsidian-800); color: var(--color-amber-glow); border: 1px solid var(--color-glass-border);">{selectedPaint.code}</span>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="artisan-card p-3">
            <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">RGB</div>
            <div class="font-mono text-sm text-white">{selectedPaint.r}, {selectedPaint.g}, {selectedPaint.b}</div>
          </div>
          {#if selectedPaint.finishType}
            <div class="artisan-card p-3">
              <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">Acabamento</div>
              <div class="text-sm text-white">{selectedPaint.finishType}</div>
            </div>
          {/if}
          {#if selectedPaint.paintType}
            <div class="artisan-card p-3">
              <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">Tipo</div>
              <div class="text-sm text-white">{selectedPaint.paintType}</div>
            </div>
          {/if}
          {#if selectedPaint.coverage}
            <div class="artisan-card p-3">
              <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">Cobertura</div>
              <div class="text-sm text-white">{selectedPaint.coverage}</div>
            </div>
          {/if}
          {#if selectedPaint.opacity}
            <div class="artisan-card p-3">
              <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">Opacidade</div>
              <div class="text-sm text-white">{selectedPaint.opacity}</div>
            </div>
          {/if}
          {#if selectedPaint.volume}
            <div class="artisan-card p-3">
              <div class="text-[10px] uppercase tracking-[0.12em] font-semibold mb-1" style="color: var(--color-obsidian-500);">Volume</div>
              <div class="text-sm text-white">{selectedPaint.volume}</div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
