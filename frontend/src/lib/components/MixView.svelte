<script lang="ts">
  import Textfield from '@smui/textfield';
  import Button from '@smui/button';
  import Slider from '@smui/slider';
  import Card, { Content } from '@smui/card';
  import LinearProgress from '@smui/linear-progress';

  let targetR = $state(180);
  let targetG = $state(120);
  let targetB = $state(60);
  let maxPaints = $state(20);
  let recipe: any = $state(null);
  let mixing = $state(false);

  async function doMix() {
    mixing = true;
    try {
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      recipe = await wailsjs.SuggestRecipe(targetR, targetG, targetB, maxPaints);
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

<div class="page-container">
  <div class="page-header animate-artisan-fade">
    <h1 class="page-title">Receita de Mistura</h1>
    <p class="page-subtitle">Descubra a fórmula perfeita para alcançar qualquer cor</p>
    <div class="page-divider"></div>
  </div>

  <div class="mix-layout">
    <!-- Target Color -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-4">Cor Desejada</h3>

      <div class="color-preview" style="width: 100%; aspect-ratio: 16/10; background: rgb({targetR}, {targetG}, {targetB});">
        <div class="color-preview-badge">RGB({targetR}, {targetG}, {targetB})</div>
      </div>

      <div style="margin-top: 20px; display: flex; flex-direction: column; gap: 16px;">
        {#each [
          { label: 'Red', value: targetR, color: '#ef4444', setter: (v: number) => targetR = v },
          { label: 'Green', value: targetG, color: '#22c55e', setter: (v: number) => targetG = v },
          { label: 'Blue', value: targetB, color: '#3b82f6', setter: (v: number) => targetB = v },
        ] as channel}
          <div>
            <div class="flex justify-between mb-2">
              <span style="font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.1em; color: {channel.color};">{channel.label}</span>
              <span class="font-mono" style="font-size: 12px; color: var(--color-obsidian-400);">{channel.value}</span>
            </div>
            <Slider
              min={0}
              max={255}
              step={1}
              bind:value={channel.value}
              style="--mdc-slider-handle-color: {channel.color}; --mdc-slider-focus-handle-color: {channel.color}; --mdc-slider-ink-color: {channel.color};"
            />
          </div>
        {/each}
      </div>

      <div style="margin-top: 16px;">
        <Textfield variant="outlined" bind:value={maxPaints} label="Máx. Tintas" type="number" style="width: 100%;" />
      </div>

      <Button variant="raised" onclick={doMix} disabled={mixing} style="width: 100%; margin-top: 20px; background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm)); color: white; font-weight: 600; border-radius: 12px; height: 48px;">
        {mixing ? 'Calculando...' : 'Sugerir Receita'}
      </Button>
    </div>

    <!-- Recipe Result -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if mixing}
        <div style="display: flex; align-items: center; justify-content: center; padding: 80px 0;">
          <LinearProgress indeterminate style="width: 200px;" />
        </div>
      {:else if recipe}
        <!-- Result Color -->
        <div class="artisan-panel p-5 mb-5">
          <h3 class="font-display text-sm font-semibold text-white mb-4">Cor Resultante</h3>
          <div class="flex gap-4 items-center">
            <div class="swatch-shimmer" style="width: 80px; height: 80px; border-radius: 12px; background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB}); box-shadow: 0 4px 16px rgba(0,0,0,0.3); flex-shrink: 0;"></div>
            <div>
              <div class="font-mono text-sm text-white">RGB({recipe.resultR}, {recipe.resultG}, {recipe.resultB})</div>
              <div class="font-mono font-bold {deltaClass(recipe.deltaE)}" style="margin-top: 4px;">
                ΔE {recipe.deltaE.toFixed(2)}
              </div>
              <div style="font-size: 12px; color: var(--color-obsidian-400); margin-top: 4px;">Método: {recipe.method}</div>
            </div>
          </div>
        </div>

        <!-- Ingredients -->
        <div class="artisan-panel p-5">
          <h3 class="font-display text-sm font-semibold text-white mb-4">Ingredientes ({recipe.ingredients?.length || 0})</h3>

          {#if recipe.ingredients}
            <div style="display: flex; flex-direction: column; gap: 12px;">
              {#each recipe.ingredients as ing, i}
                <Card variant="outlined" class="ingredient-card animate-artisan-slide" style="animation-delay: {i * 40}ms;">
                  <Content>
                    <div class="flex items-center gap-4">
                      <div class="swatch-shimmer" style="width: 44px; height: 44px; border-radius: 10px; background: rgb({ing.r}, {ing.g}, {ing.b}); box-shadow: 0 2px 8px rgba(0,0,0,0.3); flex-shrink: 0;"></div>
                      <div style="flex: 1; min-width: 0;">
                        <div class="font-semibold text-sm text-white truncate">{ing.name}</div>
                      </div>
                      <div style="text-align: right; flex-shrink: 0;">
                        <div class="font-mono font-bold text-white" style="font-size: 16px;">{ing.percentage.toFixed(1)}%</div>
                      </div>
                    </div>

                    <!-- Proportion bar -->
                    <div style="margin-top: 8px; height: 4px; border-radius: 999px; background: var(--color-obsidian-800); overflow: hidden;">
                      <div style="height: 100%; width: {ing.percentage}%; border-radius: 999px; background: linear-gradient(90deg, var(--color-amber-glow), var(--color-amber-warm)); transition: width 0.5s ease;"></div>
                    </div>
                  </Content>
                </Card>
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        <div style="text-align: center; padding: 80px 0;" class="artisan-panel">
          <span class="material-icons" style="color: var(--color-obsidian-600); font-size: 48px;">science</span>
          <p class="font-medium mt-4" style="color: var(--color-obsidian-500);">Ajuste a cor e clique em sugerir receita</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .mix-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 24px;
  }

  :global(.ingredient-card) {
    background: linear-gradient(145deg, rgba(255,255,255,0.035), rgba(255,255,255,0.01));
    border-color: rgba(255, 255, 255, 0.06);
    border-radius: 14px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.ingredient-card:hover) {
    background: linear-gradient(145deg, rgba(255,255,255,0.06), rgba(255,255,255,0.025));
    border-color: rgba(212, 160, 83, 0.15);
    transform: translateY(-1px);
  }
</style>
