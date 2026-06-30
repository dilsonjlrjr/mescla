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
</script>

<div class="min-h-full p-8">
  <div class="mb-6 animate-fadeIn">
    <h1 class="text-2xl font-bold text-white tracking-tight mb-1">Buscar por Cor</h1>
    <p class="text-sm" style="color: var(--color-surface-400);">Encontre tintas similares a uma cor específica usando Delta E 2000</p>
  </div>

  <div class="grid grid-cols-3 gap-6">
    <!-- Color Picker Panel -->
    <div class="glass rounded-xl p-5 animate-fadeIn" style="animation-delay: 100ms;">
      <h3 class="text-sm font-semibold text-white mb-4">Cor Alvo</h3>

      <!-- Preview -->
      <div class="w-full aspect-video rounded-xl mb-4 shadow-lg relative overflow-hidden"
        style="background: rgb({targetR}, {targetG}, {targetB}); border: 1px solid rgba(255,255,255,0.1);">
        <div class="absolute bottom-2 left-2 right-2 flex justify-between">
          <span class="text-xs font-mono px-2 py-1 rounded bg-black/40 text-white backdrop-blur-sm">
            RGB({targetR}, {targetG}, {targetB})
          </span>
        </div>
      </div>

      <!-- Sliders -->
      <div class="space-y-4">
        {#each [
          { label: 'R', value: targetR, color: '#ef4444', setter: (v: number) => targetR = v },
          { label: 'G', value: targetG, color: '#22c55e', setter: (v: number) => targetG = v },
          { label: 'B', value: targetB, color: '#3b82f6', setter: (v: number) => targetB = v },
        ] as channel}
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs font-semibold" style="color: {channel.color};">{channel.label}</span>
              <span class="text-xs font-mono text-surface-400">{channel.value}</span>
            </div>
            <input
              type="range"
              min="0"
              max="255"
              value={channel.value}
              oninput={(e) => channel.setter(parseInt(e.currentTarget.value))}
              class="w-full h-1.5 rounded-full appearance-none cursor-pointer"
              style="background: linear-gradient(to right, var(--color-surface-700), {channel.color}); accent-color: {channel.color};"
            />
          </div>
        {/each}
      </div>

      <!-- Filters -->
      <div class="mt-5 space-y-3">
        <div>
          <label class="text-xs font-medium text-surface-400 mb-1 block">Delta E Máximo</label>
          <input type="number" bind:value={maxDeltaE} min="1" max="50" step="1"
            class="w-full px-3 py-2 rounded-lg text-sm text-white outline-none focus:ring-1 focus:ring-accent-500/30"
            style="background: var(--color-surface-800); border: 1px solid var(--color-glass-border);" />
        </div>
        <div>
          <label class="text-xs font-medium text-surface-400 mb-1 block">Máx. Resultados</label>
          <input type="number" bind:value={maxResults} min="1" max="50" step="1"
            class="w-full px-3 py-2 rounded-lg text-sm text-white outline-none focus:ring-1 focus:ring-accent-500/30"
            style="background: var(--color-surface-800); border: 1px solid var(--color-glass-border);" />
        </div>
      </div>

      <button
        onclick={doSearch}
        disabled={searching}
        class="w-full mt-4 py-2.5 rounded-lg text-sm font-semibold text-white transition-all duration-200 disabled:opacity-50"
        style="background: linear-gradient(135deg, var(--color-accent-500), var(--color-accent-600)); box-shadow: 0 4px 16px rgba(232,168,76,0.3);"
      >
        {searching ? 'Buscando...' : 'Buscar Tintas Similares'}
      </button>
    </div>

    <!-- Results -->
    <div class="col-span-2 animate-fadeIn" style="animation-delay: 200ms;">
      {#if searching}
        <div class="flex items-center justify-center py-16">
          <div class="w-8 h-8 border-2 border-accent-500/30 border-t-accent-500 rounded-full animate-spin"></div>
        </div>
      {:else if hasSearched && results.length === 0}
        <div class="text-center py-16 glass rounded-xl">
          <p class="text-sm" style="color: var(--color-surface-500);">Nenhuma tinta similar encontrada</p>
        </div>
      {:else if results.length > 0}
        <div class="space-y-3">
          <div class="text-sm font-medium" style="color: var(--color-surface-400);">
            {results.length} tintas encontradas
          </div>
          {#each results as result, i}
            <div class="glass glass-hover rounded-xl p-4 flex items-center gap-4 animate-slideIn" style="animation-delay: {i * 50}ms;">
              <div class="w-14 h-14 rounded-lg flex-shrink-0 shadow-md"
                style="background: rgb({result.r}, {result.g}, {result.b}); border: 1px solid rgba(255,255,255,0.1);">
              </div>
              <div class="flex-1 min-w-0">
                <div class="font-semibold text-sm text-white truncate">{result.name}</div>
                <div class="text-xs" style="color: var(--color-surface-500);">{result.manufacturer}</div>
              </div>
              <div class="text-right flex-shrink-0">
                <div class="text-sm font-mono font-bold" style="color: {result.deltaE < 3 ? 'var(--color-pigment-green)' : result.deltaE < 6 ? 'var(--color-accent-400)' : 'var(--color-pigment-red)'};">
                  ΔE {result.deltaE.toFixed(1)}
                </div>
                <div class="text-xs" style="color: var(--color-surface-500);">
                  {result.similarity.toFixed(1)}% similar
                </div>
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="text-center py-16 glass rounded-xl">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-surface-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <p class="text-sm" style="color: var(--color-surface-500);">Ajuste a cor e clique em buscar</p>
        </div>
      {/if}
    </div>
  </div>
</div>
