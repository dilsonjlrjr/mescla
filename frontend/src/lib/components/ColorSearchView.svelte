<script lang="ts">
  import Textfield from '@smui/textfield';
  import Button from '@smui/button';
  import Slider from '@smui/slider';
  import Card, { Content } from '@smui/card';
  import LinearProgress from '@smui/linear-progress';

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
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      results = await wailsjs.FindSimilar(targetR, targetG, targetB, maxDeltaE, maxResults) || [];
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

  function deltaColor(de: number): string {
    if (de < 3) return '#2ecc71';
    if (de < 6) return '#f1c40f';
    if (de < 10) return '#e67e22';
    return '#e74c3c';
  }
</script>

<div class="page-container">
  <div class="page-header animate-artisan-fade">
    <h1 class="page-title">Buscar por Cor</h1>
    <p class="page-subtitle">Encontre tintas similares usando Delta E 2000</p>
    <div class="page-divider"></div>
  </div>

  <div class="search-layout">
    <!-- Color Picker Panel -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-4">Cor Alvo</h3>

      <!-- Preview -->
      <div class="color-preview" style="width: 100%; aspect-ratio: 16/10; background: rgb({targetR}, {targetG}, {targetB});">
        <div class="color-preview-badge">RGB({targetR}, {targetG}, {targetB})</div>
      </div>

      <!-- Sliders -->
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

      <!-- Filters -->
      <div style="margin-top: 24px; display: flex; flex-direction: column; gap: 12px;">
        <Textfield variant="outlined" bind:value={maxDeltaE} label="Delta E Máximo" type="number" style="width: 100%;" />
        <Textfield variant="outlined" bind:value={maxResults} label="Máx. Resultados" type="number" style="width: 100%;" />
      </div>

      <Button variant="raised" onclick={doSearch} disabled={searching} style="width: 100%; margin-top: 20px; background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm)); color: white; font-weight: 600; border-radius: 12px; height: 48px;">
        {searching ? 'Buscando...' : 'Buscar Tintas Similares'}
      </Button>
    </div>

    <!-- Results -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if searching}
        <div style="display: flex; align-items: center; justify-content: center; padding: 80px 0;">
          <LinearProgress indeterminate style="width: 200px;" />
        </div>
      {:else if hasSearched && results.length === 0}
        <div style="text-align: center; padding: 80px 0;" class="artisan-panel">
          <span class="material-icons" style="color: var(--color-obsidian-600); font-size: 48px;">search_off</span>
          <p class="font-medium mt-4" style="color: var(--color-obsidian-500);">Nenhuma tinta similar encontrada</p>
        </div>
      {:else if results.length > 0}
        <div style="display: flex; flex-direction: column; gap: 12px;">
          <div class="text-sm font-medium mb-2" style="color: var(--color-obsidian-400);">
            {results.length} tintas encontradas
          </div>
          {#each results as result, i}
            <Card variant="outlined" class="result-card animate-artisan-slide" style="animation-delay: {i * 40}ms;">
              <Content>
                <div class="flex items-center gap-4">
                  <div class="swatch-shimmer" style="width: 56px; height: 56px; border-radius: 12px; flex-shrink: 0; background: rgb({result.r}, {result.g}, {result.b}); box-shadow: 0 4px 12px rgba(0,0,0,0.3);"></div>
                  <div style="flex: 1; min-width: 0;">
                    <div class="font-semibold text-sm text-white truncate">{result.name}</div>
                    <div style="font-size: 12px; color: var(--color-obsidian-400);">{result.manufacturer}</div>
                  </div>
                  <div style="text-align: right; flex-shrink: 0;">
                    <div class="font-mono font-bold {deltaClass(result.deltaE)}">
                      ΔE {result.deltaE.toFixed(1)}
                    </div>
                    <div style="font-size: 11px; color: var(--color-obsidian-500);">
                      {result.similarity.toFixed(1)}% similar
                    </div>
                  </div>
                  <div style="width: 6px; height: 40px; border-radius: 999px; flex-shrink: 0; background: {deltaColor(result.deltaE)}; opacity: 0.7;"></div>
                </div>
              </Content>
            </Card>
          {/each}
        </div>
      {:else}
        <div style="text-align: center; padding: 80px 0;" class="artisan-panel">
          <span class="material-icons" style="color: var(--color-obsidian-600); font-size: 48px;">palette</span>
          <p class="font-medium mt-4" style="color: var(--color-obsidian-500);">Ajuste a cor alvo e clique em buscar</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .search-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 24px;
  }

  :global(.result-card) {
    background: linear-gradient(145deg, rgba(255,255,255,0.035), rgba(255,255,255,0.01));
    border-color: rgba(255, 255, 255, 0.06);
    border-radius: 14px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.result-card:hover) {
    background: linear-gradient(145deg, rgba(255,255,255,0.06), rgba(255,255,255,0.025));
    border-color: rgba(212, 160, 83, 0.15);
    transform: translateY(-1px);
  }
</style>
