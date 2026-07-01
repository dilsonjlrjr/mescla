<script lang="ts">
  import Textfield from '@smui/textfield';
  import Button from '@smui/button';
  import Slider from '@smui/slider';
  import LinearProgress from '@smui/linear-progress';
  import Icon from './Icon.svelte';
  import DeltaBadge from './DeltaBadge.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

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
      results = await PaintService.FindSimilar(targetR, targetG, targetB, maxDeltaE, maxResults) || [];
    } catch (e) {
      console.error('Erro buscando similar:', e);
      results = [];
    } finally {
      searching = false;
    }
  }

</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Buscar cor</h1>
    <p class="page-subtitle">Monte a cor exata e descubra qual tinta chega mais perto dela</p>
    <div class="page-divider"></div>
  </div>

  <div class="search-layout">
    <!-- Color Picker Panel -->
    <div class="panel p-5 animate-rise" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-4">Cor alvo</h3>

      <!-- Preview -->
      <div class="color-preview" style="width: 100%; aspect-ratio: 16/10; background: rgb({targetR}, {targetG}, {targetB});">
        <div class="color-preview-badge">RGB({targetR}, {targetG}, {targetB})</div>
      </div>

      <!-- Sliders -->
      <div style="margin-top: 20px; display: flex; flex-direction: column; gap: 16px;">
        {#each [
          { label: 'Red', value: targetR, color: 'var(--chan-r)', setter: (v: number) => targetR = v },
          { label: 'Green', value: targetG, color: 'var(--chan-g)', setter: (v: number) => targetG = v },
          { label: 'Blue', value: targetB, color: 'var(--chan-b)', setter: (v: number) => targetB = v },
        ] as channel}
          <div>
            <div class="flex justify-between mb-2">
              <span style="font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.1em; color: {channel.color};">{channel.label}</span>
              <span class="font-mono" style="font-size: 12px; color: var(--ink-400);">{channel.value}</span>
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
        <Textfield variant="outlined" bind:value={maxDeltaE} label="Delta E máximo" type="number" style="width: 100%;" />
        <Textfield variant="outlined" bind:value={maxResults} label="Máx. resultados" type="number" style="width: 100%;" />
      </div>

      <Button variant="raised" onclick={doSearch} disabled={searching} style="width: 100%; margin-top: 20px; background: var(--lacquer); color: white; font-weight: 600; border-radius: 8px; height: 46px;">
        {searching ? 'Buscando…' : 'Buscar tintas similares'}
      </Button>
    </div>

    <!-- Results -->
    <div class="animate-rise" style="animation-delay: 160ms;">
      {#if searching}
        <div style="display: flex; align-items: center; justify-content: center; padding: 80px 0;">
          <LinearProgress indeterminate style="width: 200px;" />
        </div>
      {:else if hasSearched && results.length === 0}
        <div class="panel empty-state">
          <span class="empty-icon"><Icon name="search-off" size={40} /></span>
          <p class="empty-title">Nenhuma tinta dentro do limite</p>
          <p class="empty-hint">Aumente o "ΔE máximo" (12 já mostra cores visivelmente diferentes) e busque de novo.</p>
        </div>
      {:else if results.length > 0}
        <div style="display: flex; flex-direction: column; gap: 10px;">
          <div class="text-sm font-medium mb-2" style="color: var(--ink-400);">
            {results.length} tintas encontradas
          </div>
          {#each results as result, i}
            <div class="panel result-row animate-slide" style="animation-delay: {i * 40}ms;">
              <PaintBottle r={result.r} g={result.g} b={result.b} size={52} />
              <div style="flex: 1; min-width: 0;">
                <div class="font-semibold text-sm text-white truncate">{result.name}</div>
                <div style="font-size: 12px; color: var(--ink-500);">{result.manufacturer}</div>
              </div>
              <DeltaBadge
                deltaE={result.deltaE}
                size="sm"
                pair={{ r1: targetR, g1: targetG, b1: targetB, r2: result.r, g2: result.g, b2: result.b }}
              />
            </div>
          {/each}
        </div>
      {:else}
        <div class="panel empty-state">
          <span class="empty-icon"><Icon name="palette" size={40} /></span>
          <p class="empty-title">Monte a cor nos controles ao lado</p>
          <p class="empty-hint">Arraste os canais R, G e B até a cor do preview bater com o que você procura.</p>
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

  .result-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 16px;
    transition: border-color 0.15s ease;
  }

  .result-row:hover {
    border-color: var(--ink-600);
  }
</style>
