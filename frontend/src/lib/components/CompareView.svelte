<script lang="ts">
  import { onMount } from 'svelte';

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  let allPaints: Paint[] = $state([]);
  let selectedIDs: number[] = $state([]);
  let results: any[] = $state([]);
  let loading = $state(true);
  let comparing = $state(false);

  onMount(async () => {
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
      allPaints = await PaintService.GetAllPaints() || [];
    } catch (e) {
      console.error('Erro:', e);
    } finally {
      loading = false;
    }
  });

  function togglePaint(id: number) {
    if (selectedIDs.includes(id)) {
      selectedIDs = selectedIDs.filter(x => x !== id);
    } else if (selectedIDs.length < 6) {
      selectedIDs = [...selectedIDs, id];
    }
  }

  async function doCompare() {
    if (selectedIDs.length < 2) return;
    comparing = true;
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
      results = await PaintService.CompareColors(selectedIDs) || [];
    } catch (e) {
      console.error('Erro comparando:', e);
      results = [];
    } finally {
      comparing = false;
    }
  }

  function getSelectedPaints(): Paint[] {
    return selectedIDs.map(id => allPaints.find(p => p.id === id)).filter(Boolean) as Paint[];
  }

  function deltaClass(de: number): string {
    if (de < 3) return 'delta-excellent';
    if (de < 6) return 'delta-good';
    if (de < 10) return 'delta-fair';
    return 'delta-poor';
  }
</script>

<div class="min-h-full p-8 relative z-10">
  <div class="mb-8 animate-artisan-fade">
    <h1 class="font-[family-name:var(--font-display)] text-3xl font-bold text-white tracking-tight mb-2">Comparar Tintas</h1>
    <p class="text-[13px]" style="color: var(--color-obsidian-400);">Selecione até 6 tintas para comparar lado a lado</p>
    <div class="mt-4 h-[1px]" style="background: linear-gradient(90deg, var(--color-amber-glow), transparent 50%); opacity: 0.15;"></div>
  </div>

  <div class="grid grid-cols-[300px_1fr] gap-6">
    <!-- Selection Panel -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-[family-name:var(--font-display)] text-sm font-semibold text-white mb-2">Selecionar Tintas</h3>
      <div class="text-[11px] font-medium mb-4" style="color: var(--color-obsidian-500);">
        {selectedIDs.length}/6 selecionadas
      </div>

      <!-- Selected chips -->
      {#if selectedIDs.length > 0}
        <div class="flex gap-1.5 mb-4 flex-wrap">
          {#each getSelectedPaints() as paint}
            <button onclick={() => togglePaint(paint.id)}
              class="flex items-center gap-1.5 px-2 py-1 rounded-lg text-[11px] font-medium text-white transition-all hover:scale-105"
              style="background: var(--color-obsidian-700); border: 1px solid rgba(255,255,255,0.08);">
              <span class="w-3 h-3 rounded-sm flex-shrink-0" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
              <span class="truncate max-w-[70px]">{paint.name}</span>
              <svg class="w-3 h-3 text-[var(--color-obsidian-400)]" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          {/each}
        </div>
      {/if}

      <!-- Paint list -->
      <div class="space-y-0.5 max-h-[400px] overflow-y-auto pr-1">
        {#each allPaints as paint (paint.id)}
          {@const isSelected = selectedIDs.includes(paint.id)}
          <button
            onclick={() => togglePaint(paint.id)}
            class="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150 text-[12px]
              {isSelected
                ? 'bg-[rgba(212,160,83,0.08)] border border-[rgba(212,160,83,0.15)]'
                : 'hover:bg-white/[0.02] border border-transparent'}"
          >
            <span class="w-4 h-4 rounded flex-shrink-0" style="background: rgb({paint.r}, {paint.g}, {paint.b}); border: 1px solid rgba(255,255,255,0.08);"></span>
            <span class="flex-1 truncate" style="color: {isSelected ? 'white' : 'var(--color-obsidian-300)'};">{paint.name}</span>
            <span class="font-mono text-[10px]" style="color: var(--color-obsidian-600);">{paint.code}</span>
          </button>
        {/each}
      </div>

      <button onclick={doCompare} disabled={selectedIDs.length < 2 || comparing} class="btn-amber w-full mt-4">
        {comparing ? 'Comparando...' : 'Comparar Selecionadas'}
      </button>
    </div>

    <!-- Results -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if results.length > 0}
        <div class="space-y-5">
          <div class="text-sm font-medium" style="color: var(--color-obsidian-400);">
            {results.length} comparações
          </div>

          <!-- Visual comparison -->
          <div class="artisan-panel p-5">
            <div class="flex gap-3">
              {#each getSelectedPaints() as paint}
                <div class="flex-1">
                  <div class="w-full aspect-square rounded-xl swatch-shimmer"
                    style="background: rgb({paint.r}, {paint.g}, {paint.b}); box-shadow: 0 4px 16px rgba(0,0,0,0.3);">
                  </div>
                  <div class="text-center mt-2.5">
                    <div class="text-[12px] font-medium text-white truncate">{paint.name}</div>
                    <div class="text-[10px] font-mono" style="color: var(--color-obsidian-500);">rgb({paint.r},{paint.g},{paint.b})</div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Delta E Results -->
          {#each results as result, i}
            <div class="artisan-card p-4 flex items-center gap-4 animate-artisan-slide" style="animation-delay: {i * 50}ms;">
              <div class="flex-1">
                <div class="text-sm font-medium text-white">{result.name}</div>
                <div class="text-[12px]" style="color: var(--color-obsidian-400);">{result.manufacturer}</div>
              </div>
              <div class="text-right">
                <div class="text-lg font-mono font-bold {deltaClass(result.deltaE)}">
                  ΔE {result.deltaE.toFixed(2)}
                </div>
                <div class="text-[11px]" style="color: var(--color-obsidian-500);">
                  {result.similarity.toFixed(1)}% similar
                </div>
              </div>
              <div class="w-1.5 h-10 rounded-full"
                style="background: {result.deltaE < 3 ? '#2ecc71' : result.deltaE < 6 ? '#f1c40f' : result.deltaE < 10 ? '#e67e22' : '#e74c3c'}; opacity: 0.7;">
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="text-center py-20 artisan-panel">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-obsidian-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
          </svg>
          <p class="text-sm font-medium" style="color: var(--color-obsidian-500);">Selecione tintas e clique em comparar</p>
        </div>
      {/if}
    </div>
  </div>
</div>
