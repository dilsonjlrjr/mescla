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
</script>

<div class="min-h-full p-8">
  <div class="mb-6 animate-fadeIn">
    <h1 class="text-2xl font-bold text-white tracking-tight mb-1">Comparar Tintas</h1>
    <p class="text-sm" style="color: var(--color-surface-400);">Selecione até 6 tintas para comparar lado a lado</p>
  </div>

  <div class="grid grid-cols-3 gap-6">
    <!-- Selection Panel -->
    <div class="glass rounded-xl p-5 animate-fadeIn" style="animation-delay: 100ms;">
      <h3 class="text-sm font-semibold text-white mb-3">Selecionar Tintas</h3>
      <div class="text-xs mb-3" style="color: var(--color-surface-500);">
        {selectedIDs.length}/6 selecionadas
      </div>

      <!-- Selected -->
      {#if selectedIDs.length > 0}
        <div class="flex gap-2 mb-4 flex-wrap">
          {#each getSelectedPaints() as paint}
            <button onclick={() => togglePaint(paint.id)}
              class="flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium text-white transition-colors hover:bg-white/10"
              style="background: var(--color-surface-700);">
              <span class="w-3 h-3 rounded-sm flex-shrink-0" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
              <span class="truncate max-w-[80px]">{paint.name}</span>
              <svg class="w-3 h-3 text-surface-400" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          {/each}
        </div>
      {/if}

      <!-- Paint List -->
      <div class="space-y-1 max-h-[400px] overflow-y-auto pr-1">
        {#each allPaints as paint (paint.id)}
          <button
            onclick={() => togglePaint(paint.id)}
            class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-left transition-colors text-xs
              {selectedIDs.includes(paint.id) ? 'bg-accent-500/10 border border-accent-500/20' : 'hover:bg-white/[0.03] border border-transparent'}"
          >
            <span class="w-4 h-4 rounded-sm flex-shrink-0" style="background: rgb({paint.r}, {paint.g}, {paint.b}); border: 1px solid rgba(255,255,255,0.1);"></span>
            <span class="flex-1 truncate text-surface-300">{paint.name}</span>
            <span class="text-surface-600 font-mono">{paint.code}</span>
          </button>
        {/each}
      </div>

      <button
        onclick={doCompare}
        disabled={selectedIDs.length < 2 || comparing}
        class="w-full mt-4 py-2.5 rounded-lg text-sm font-semibold text-white transition-all duration-200 disabled:opacity-40"
        style="background: linear-gradient(135deg, var(--color-accent-500), var(--color-accent-600)); box-shadow: 0 4px 16px rgba(232,168,76,0.3);"
      >
        {comparing ? 'Comparando...' : 'Comparar'}
      </button>
    </div>

    <!-- Results -->
    <div class="col-span-2 animate-fadeIn" style="animation-delay: 200ms;">
      {#if results.length > 0}
        <div class="space-y-4">
          <div class="text-sm font-medium" style="color: var(--color-surface-400);">
            {results.length} comparações
          </div>

          <!-- Visual Comparison -->
          <div class="glass rounded-xl p-5">
            <div class="flex gap-3">
              {#each getSelectedPaints() as paint}
                <div class="flex-1">
                  <div class="w-full aspect-square rounded-xl mb-2 shadow-lg"
                    style="background: rgb({paint.r}, {paint.g}, {paint.b}); border: 1px solid rgba(255,255,255,0.1);">
                  </div>
                  <div class="text-center">
                    <div class="text-xs font-medium text-white truncate">{paint.name}</div>
                    <div class="text-[10px] font-mono" style="color: var(--color-surface-500);">rgb({paint.r},{paint.g},{paint.b})</div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Delta E Results -->
          {#each results as result, i}
            <div class="glass glass-hover rounded-xl p-4 flex items-center gap-4 animate-slideIn" style="animation-delay: {i * 60}ms;">
              <div class="flex-1">
                <div class="text-sm font-medium text-white">{result.name}</div>
                <div class="text-xs" style="color: var(--color-surface-500);">{result.manufacturer}</div>
              </div>
              <div class="text-right">
                <div class="text-lg font-mono font-bold" style="color: {result.deltaE < 3 ? 'var(--color-pigment-green)' : result.deltaE < 6 ? 'var(--color-accent-400)' : 'var(--color-pigment-red)'};">
                  ΔE {result.deltaE.toFixed(2)}
                </div>
                <div class="text-xs" style="color: var(--color-surface-500);">
                  {result.similarity.toFixed(1)}% similar
                </div>
              </div>
              <div class="w-2 h-8 rounded-full"
                style="background: {result.deltaE < 3 ? 'var(--color-pigment-green)' : result.deltaE < 6 ? 'var(--color-accent-400)' : 'var(--color-pigment-red)'}; opacity: 0.6;">
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="text-center py-16 glass rounded-xl">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-surface-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
          </svg>
          <p class="text-sm" style="color: var(--color-surface-500);">Selecione tintas e clique em comparar</p>
        </div>
      {/if}
    </div>
  </div>
</div>
