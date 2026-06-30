<script lang="ts">
  let targetR = $state(180);
  let targetG = $state(120);
  let targetB = $state(60);
  let maxPaints = $state(20);
  let recipe: any = $state(null);
  let mixing = $state(false);

  async function doMix() {
    mixing = true;
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
      recipe = await PaintService.SuggestRecipe(targetR, targetG, targetB, maxPaints);
    } catch (e) {
      console.error('Erro sugerindo mistura:', e);
      recipe = null;
    } finally {
      mixing = false;
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
    <h1 class="font-[family-name:var(--font-display)] text-3xl font-bold text-white tracking-tight mb-2">Receita de Mistura</h1>
    <p class="text-[13px]" style="color: var(--color-obsidian-400);">Descubra a fórmula perfeita para alcançar qualquer cor</p>
    <div class="mt-4 h-[1px]" style="background: linear-gradient(90deg, var(--color-amber-glow), transparent 50%); opacity: 0.15;"></div>
  </div>

  <div class="grid grid-cols-[320px_1fr] gap-6">
    <!-- Target Color -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-[family-name:var(--font-display)] text-sm font-semibold text-white mb-4">Cor Desejada</h3>

      <div class="w-full aspect-[16/10] rounded-xl mb-5 relative overflow-hidden"
        style="background: rgb({targetR}, {targetG}, {targetB}); box-shadow: 0 8px 32px rgba(0,0,0,0.4), inset 0 1px 0 rgba(255,255,255,0.1);">
        <div class="absolute bottom-2.5 left-2.5">
          <span class="text-[11px] font-mono px-2.5 py-1 rounded-lg backdrop-blur-md"
            style="background: rgba(0,0,0,0.5); color: rgba(255,255,255,0.9); border: 1px solid rgba(255,255,255,0.1);">
            RGB({targetR}, {targetG}, {targetB})
          </span>
        </div>
      </div>

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

      <div class="mt-6">
        <label class="text-[11px] font-semibold uppercase tracking-wider mb-1.5 block" style="color: var(--color-obsidian-500);">Tintas disponíveis</label>
        <input type="number" bind:value={maxPaints} min="2" max="45" step="1" class="artisan-input w-full" />
      </div>

      <button onclick={doMix} disabled={mixing} class="btn-amber w-full mt-5">
        {mixing ? 'Calculando...' : 'Sugerir Mistura'}
      </button>
    </div>

    <!-- Recipe Result -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if recipe && recipe.ingredients && recipe.ingredients.length > 0}
        <div class="space-y-5">
          <!-- Result preview -->
          <div class="artisan-panel p-5">
            <div class="flex items-center gap-5">
              <div class="w-20 h-20 rounded-2xl flex-shrink-0 swatch-shimmer"
                style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB}); box-shadow: 0 6px 20px rgba(0,0,0,0.3);">
              </div>
              <div>
                <div class="font-[family-name:var(--font-display)] text-lg font-semibold text-white mb-1">Resultado da Mistura</div>
                <div class="text-xs font-mono mb-2" style="color: var(--color-obsidian-400);">
                  RGB({recipe.resultR}, {recipe.resultG}, {recipe.resultB})
                </div>
                <div class="flex items-center gap-3">
                  <span class="text-[11px] font-semibold uppercase tracking-wider" style="color: var(--color-obsidian-500);">Delta E:</span>
                  <span class="font-mono text-sm font-bold {deltaClass(recipe.deltaE)}">
                    {recipe.deltaE.toFixed(2)}
                  </span>
                  <span class="text-[10px] px-2 py-0.5 rounded-md font-mono" style="background: var(--color-obsidian-800); color: var(--color-obsidian-500);">
                    {recipe.method}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Ingredients -->
          <div class="artisan-panel p-5">
            <h3 class="font-[family-name:var(--font-display)] text-sm font-semibold text-white mb-4">Ingredientes</h3>
            <div class="space-y-3">
              {#each recipe.ingredients as ing, i}
                <div class="flex items-center gap-4 animate-artisan-slide" style="animation-delay: {i * 60}ms;">
                  <div class="w-11 h-11 rounded-xl flex-shrink-0 swatch-shimmer"
                    style="background: rgb({ing.r}, {ing.g}, {ing.b}); box-shadow: 0 4px 12px rgba(0,0,0,0.3);">
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium text-white truncate">{ing.name}</div>
                  </div>
                  <div class="flex items-center gap-3">
                    <div class="w-36 h-2 rounded-full overflow-hidden" style="background: var(--color-obsidian-700);">
                      <div class="h-full rounded-full transition-all duration-700 ease-out"
                        style="width: {ing.percentage}%; background: linear-gradient(90deg, var(--color-amber-dim), var(--color-amber-glow)); box-shadow: 0 0 8px rgba(212,160,83,0.3);">
                      </div>
                    </div>
                    <span class="text-sm font-mono font-bold w-12 text-right" style="color: var(--color-amber-glow);">{ing.percentage.toFixed(0)}%</span>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Visual comparison -->
          <div class="artisan-panel p-5">
            <h3 class="font-[family-name:var(--font-display)] text-sm font-semibold text-white mb-4">Comparação Visual</h3>
            <div class="flex gap-4 items-end">
              <div class="flex-1">
                <div class="text-[10px] uppercase tracking-wider font-semibold mb-2" style="color: var(--color-obsidian-500);">Alvo</div>
                <div class="w-full h-16 rounded-xl"
                  style="background: rgb({targetR}, {targetG}, {targetB}); box-shadow: 0 4px 16px rgba(0,0,0,0.3);">
                </div>
              </div>
              <div class="flex items-center pb-5">
                <svg class="w-5 h-5" style="color: var(--color-obsidian-500);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                </svg>
              </div>
              <div class="flex-1">
                <div class="text-[10px] uppercase tracking-wider font-semibold mb-2" style="color: var(--color-obsidian-500);">Resultado</div>
                <div class="w-full h-16 rounded-xl"
                  style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB}); box-shadow: 0 4px 16px rgba(0,0,0,0.3);">
                </div>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div class="text-center py-20 artisan-panel">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-obsidian-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
          </svg>
          <p class="text-sm font-medium" style="color: var(--color-obsidian-500);">Ajuste a cor desejada e clique em sugerir</p>
        </div>
      {/if}
    </div>
  </div>
</div>
