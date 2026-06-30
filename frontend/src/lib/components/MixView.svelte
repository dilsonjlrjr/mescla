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
</script>

<div class="min-h-full p-8">
  <div class="mb-6 animate-fadeIn">
    <h1 class="text-2xl font-bold text-white tracking-tight mb-1">Sugestão de Mistura</h1>
    <p class="text-sm" style="color: var(--color-surface-400);">Encontre a melhor receita de mistura para alcançar uma cor desejada</p>
  </div>

  <div class="grid grid-cols-3 gap-6">
    <!-- Target Color -->
    <div class="glass rounded-xl p-5 animate-fadeIn" style="animation-delay: 100ms;">
      <h3 class="text-sm font-semibold text-white mb-4">Cor Desejada</h3>

      <div class="w-full aspect-video rounded-xl mb-4 shadow-lg relative overflow-hidden"
        style="background: rgb({targetR}, {targetG}, {targetB}); border: 1px solid rgba(255,255,255,0.1);">
        <div class="absolute bottom-2 left-2">
          <span class="text-xs font-mono px-2 py-1 rounded bg-black/40 text-white backdrop-blur-sm">
            RGB({targetR}, {targetG}, {targetB})
          </span>
        </div>
      </div>

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

      <div class="mt-4">
        <label class="text-xs font-medium text-surface-400 mb-1 block">Tintas disponíveis</label>
        <input type="number" bind:value={maxPaints} min="2" max="45" step="1"
          class="w-full px-3 py-2 rounded-lg text-sm text-white outline-none focus:ring-1 focus:ring-accent-500/30"
          style="background: var(--color-surface-800); border: 1px solid var(--color-glass-border);" />
      </div>

      <button
        onclick={doMix}
        disabled={mixing}
        class="w-full mt-4 py-2.5 rounded-lg text-sm font-semibold text-white transition-all duration-200 disabled:opacity-50"
        style="background: linear-gradient(135deg, var(--color-accent-500), var(--color-accent-600)); box-shadow: 0 4px 16px rgba(232,168,76,0.3);"
      >
        {mixing ? 'Calculando...' : 'Sugerir Mistura'}
      </button>
    </div>

    <!-- Recipe Result -->
    <div class="col-span-2 animate-fadeIn" style="animation-delay: 200ms;">
      {#if recipe && recipe.ingredients && recipe.ingredients.length > 0}
        <div class="space-y-4">
          <!-- Result Preview -->
          <div class="glass rounded-xl p-5">
            <div class="flex items-center gap-4">
              <div class="w-20 h-20 rounded-xl shadow-lg"
                style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB}); border: 1px solid rgba(255,255,255,0.1);">
              </div>
              <div>
                <div class="text-sm font-semibold text-white mb-1">Resultado da Mistura</div>
                <div class="text-xs font-mono" style="color: var(--color-surface-400);">
                  RGB({recipe.resultR}, {recipe.resultG}, {recipe.resultB})
                </div>
                <div class="text-xs mt-1">
                  <span style="color: var(--color-surface-500);">Delta E:</span>
                  <span class="font-mono font-bold ml-1" style="color: {recipe.deltaE < 3 ? 'var(--color-pigment-green)' : recipe.deltaE < 6 ? 'var(--color-accent-400)' : 'var(--color-pigment-red)'};">
                    {recipe.deltaE.toFixed(2)}
                  </span>
                </div>
                <div class="text-xs" style="color: var(--color-surface-600);">
                  Método: {recipe.method}
                </div>
              </div>
            </div>
          </div>

          <!-- Ingredients -->
          <div class="glass rounded-xl p-5">
            <h3 class="text-sm font-semibold text-white mb-4">Ingredientes</h3>
            <div class="space-y-3">
              {#each recipe.ingredients as ing, i}
                <div class="flex items-center gap-4 animate-slideIn" style="animation-delay: {i * 80}ms;">
                  <div class="w-10 h-10 rounded-lg flex-shrink-0 shadow-md"
                    style="background: rgb({ing.r}, {ing.g}, {ing.b}); border: 1px solid rgba(255,255,255,0.1);">
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium text-white truncate">{ing.name}</div>
                  </div>
                  <div class="flex items-center gap-3">
                    <div class="w-32 h-2 rounded-full overflow-hidden" style="background: var(--color-surface-700);">
                      <div class="h-full rounded-full transition-all duration-500"
                        style="width: {ing.percentage}%; background: linear-gradient(to right, var(--color-accent-500), var(--color-accent-400));">
                      </div>
                    </div>
                    <span class="text-sm font-mono font-bold text-accent-400 w-14 text-right">{ing.percentage.toFixed(0)}%</span>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Comparison Bar -->
          <div class="glass rounded-xl p-5">
            <h3 class="text-sm font-semibold text-white mb-3">Comparação Visual</h3>
            <div class="flex gap-4">
              <div class="flex-1">
                <div class="text-xs mb-2" style="color: var(--color-surface-500);">Alvo</div>
                <div class="w-full h-12 rounded-lg"
                  style="background: rgb({targetR}, {targetG}, {targetB}); border: 1px solid rgba(255,255,255,0.1);">
                </div>
              </div>
              <div class="flex items-center">
                <svg class="w-6 h-6" style="color: var(--color-surface-500);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                </svg>
              </div>
              <div class="flex-1">
                <div class="text-xs mb-2" style="color: var(--color-surface-500);">Resultado</div>
                <div class="w-full h-12 rounded-lg"
                  style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB}); border: 1px solid rgba(255,255,255,0.1);">
                </div>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div class="text-center py-16 glass rounded-xl">
          <svg class="w-12 h-12 mx-auto mb-4" style="color: var(--color-surface-600);" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
          </svg>
          <p class="text-sm" style="color: var(--color-surface-500);">Ajuste a cor desejada e clique em sugerir</p>
        </div>
      {/if}
    </div>
  </div>
</div>
