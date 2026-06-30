<script lang="ts">
  let targetR = $state(180);
  let targetG = $state(120);
  let targetB = $state(60);
  let maxDeltaE = $state(10);
  let maxResults = $state(10);
  let results: any[] = $state([]);
  let searching = $state(false);
  let hasSearched = $state(false);

  async function doSearch() {
    searching = true;
    hasSearched = true;
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
      results = await PaintService.FindSimilar(targetR, targetG, targetB, maxDeltaE, maxResults) || [];
    } catch (e) {
      console.error('Erro buscando similar:', e);
      results = [];
    } finally {
      searching = false;
    }
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
    <h1 class="font-[family-name:var(--font-display)] text-3xl font-bold text-white tracking-tight mb-2">Buscar por Cor</h1>
    <p class="text-[13px]" style="color: var(--color-obsidian-400);">Encontre tintas similares usando Delta E 2000</p>
    <div class="mt-4 h-[1px]" style="background: linear-gradient(90deg, var(--color-amber-glow), transparent 50%); opacity: 0.15;"></div>
  </div>

  <div class="grid grid-cols-[320px_1fr] gap-6">
    <!-- Color Picker Panel -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-[family-name:var(--font-display)] text-sm font-semibold text-white mb-4">Cor Alvo</h3>

      <!-- Preview -->
      <div class="w-full aspect-[16/10] rounded-xl mb-5 relative overflow-hidden"
        style="background: rgb({targetR}, {targetG}, {targetB}); box-shadow: 0 8px 32px rgba(0,0,0,0.4), inset 0 1px 0 rgba(255,255,255,0.1);">
        <div class="absolute bottom-2.5 left-2.5 right-2.5 flex justify-between items-end">
          <span class="text-[11px] font-mono px-2.5 py-1 rounded-lg backdrop-blur-md"
            style="background: rgba(0,0,0,0.5); color: rgba(255,255,255,0.9); border: 1px solid rgba(255,255,255,0.1);">
            RGB({targetR}, {targetG}, {targetB})
          </span>
        </div>
      </div>

      <!-- Sliders -->
      <div class="space-y-5">
        {#each [
          { label: 'Red', value: targetR, color: '#ef4444', setter: (v: number) => targetR = v },
          { label: 'Green', value: targetG, color: '#22c55e', setter: (v: number) => targetG = v },
          { label: 'Blue', value: targetB, color: '#3b82f6', setter: (v: number) => targetB = v },
        ] as channel}
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="text-[11px] font-semibold uppercase tracking-wider" style="color: {channel.color};">{channel.label}</span>
              <span class="text-xs font-mono" style="color: var(--color-obsidian-400);">{channel.value}</span>
            </div>
            <input
              type="range"
              min="0"
              max="255"
              value={channel.value}
              oninput={(e) => channel.setter(parseInt(e.currentTarget.value))}
              class="w-full"
              style="background: linear-gradient(to right, var(--color-obsidian-700), {channel.color});"
            />
          </div>
        {/each}
      </div>

      <!-- Filters -->
      <div class="mt-6 space-y-3">
        <div>
          <label class="text-[11px] font-semibold uppercase tracking-wider mb-1.5 block" style="color: var(--color-obsidian-500);">Delta E Máximo</label>
          <input type="number" bind:value={maxDeltaE} min="1" max="50" step="1" class="artisan-input w-full" />
        </div>
        <div>
          <label class="text-[11px] font-semibold uppercase tracking-wider mb-1.5 block" style="color: var(--color-obsidian-500);">Máx. Resultados</label>
          <input type="number" bind:value={maxResults} min="1" max="50" step="1" class="artisan-input w-full" />
        </div>
      </div>

      <button onclick={doSearch} disabled={searching} class="btn-amber w-full mt-5">
        {searching ? 'Buscando...' : 'Buscar Tintas Similares'}
      </button>
    </div>

    <!-- Results -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if searching}
        <div class="flex items-center justify-center py-20">
          <div class="w-10 h-10 border-2 rounded-full animate-spin" style="border-color: rgba(212,160,83,0.2); border-top-color: var(--color-amber-glow);"></div>
        </div>
      {:else if hasSearched && results.length === 0}
        <div class="text-center py-20 artisan-panel">
          <div class="w-16 h-16 mx-auto mb-4 rounded-2xl flex items-center justify-center" style="background: var(--color-obsidian-800);">
            <svg class="w-8 h-8" style="color: var(--color-obsidian-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <p class="text-sm font-medium" style="color: var(--color-obsidian-500);">Nenhuma tinta similar encontrada</p>
        </div>
      {:else if results.length > 0}
        <div class="space-y-3">
          <div class="text-sm font-medium mb-2" style="color: var(--color-obsidian-400);">
            {results.length} tintas encontradas
          </div>
          {#each results as result, i}
            <div class="artisan-card p-4 flex items-center gap-4 animate-artisan-slide" style="animation-delay: {i * 40}ms;">
              <div class="w-14 h-14 rounded-xl flex-shrink-0 swatch-shimmer"
                style="background: rgb({result.r}, {result.g}, {result.b}); box-shadow: 0 4px 12px rgba(0,0,0,0.3);">
              </div>
              <div class="flex-1 min-w-0">
                <div class="font-semibold text-sm text-white truncate">{result.name}</div>
                <div class="text-[12px]" style="color: var(--color-obsidian-400);">{result.manufacturer}</div>
              </div>
              <div class="text-right flex-shrink-0">
                <div class="text-sm font-mono font-bold {deltaClass(result.deltaE)}">
                  ΔE {result.deltaE.toFixed(1)}
                </div>
                <div class="text-[11px]" style="color: var(--color-obsidian-500);">
                  {result.similarity.toFixed(1)}% similar
                </div>
              </div>
              <div class="w-1.5 h-10 rounded-full flex-shrink-0"
                style="background: {result.deltaE < 3 ? '#2ecc71' : result.deltaE < 6 ? '#f1c40f' : result.deltaE < 10 ? '#e67e22' : '#e74c3c'}; opacity: 0.7;">
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="text-center py-20 artisan-panel">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-obsidian-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
          </svg>
          <p class="text-sm font-medium" style="color: var(--color-obsidian-500);">Ajuste a cor alvo e clique em buscar</p>
        </div>
      {/if}
    </div>
  </div>
</div>
