<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '@smui/button';
  import Icon from './Icon.svelte';

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
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      allPaints = await wailsjs.GetAllPaints() || [];
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
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      results = await wailsjs.CompareColors(selectedIDs) || [];
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

  function deltaColor(de: number): string {
    if (de < 3) return 'var(--delta-excellent)';
    if (de < 6) return 'var(--delta-good)';
    if (de < 10) return 'var(--delta-fair)';
    return 'var(--delta-poor)';
  }
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Comparar tintas</h1>
    <p class="page-subtitle">Selecione até 6 tintas para comparar lado a lado</p>
    <div class="page-divider"></div>
  </div>

  <div class="compare-layout">
    <!-- Selection Panel -->
    <div class="panel p-5 animate-rise" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-2">Selecionar tintas</h3>
      <div style="font-size: 11px; font-weight: 500; margin-bottom: 16px; color: var(--ink-500);">
        {selectedIDs.length}/6 selecionadas
      </div>

      <!-- Selected chips -->
      {#if selectedIDs.length > 0}
        <div class="flex gap-2 mb-4 flex-wrap">
          {#each getSelectedPaints() as paint}
            <button class="selected-chip" onclick={() => togglePaint(paint.id)}>
              <span class="chip-swatch-sm" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
              <span class="truncate" style="max-width: 70px;">{paint.name}</span>
              <Icon name="close" size={12} />
            </button>
          {/each}
        </div>
      {/if}

      <!-- Paint list -->
      <div class="pick-list">
        {#each allPaints as paint (paint.id)}
          {@const isSelected = selectedIDs.includes(paint.id)}
          <button class="pick-item" class:selected={isSelected} onclick={() => togglePaint(paint.id)}>
            <span class="chip-swatch-sm" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
            <span class="pick-text">
              <span class="pick-name">{paint.name}</span>
              <span class="pick-code">{paint.code}</span>
            </span>
          </button>
        {/each}
      </div>

      <Button variant="raised" onclick={doCompare} disabled={selectedIDs.length < 2 || comparing} style="width: 100%; margin-top: 16px; background: var(--lacquer); color: white; font-weight: 600; border-radius: 8px; height: 44px;">
        {comparing ? 'Comparando…' : 'Comparar selecionadas'}
      </Button>
    </div>

    <!-- Results -->
    <div class="animate-rise" style="animation-delay: 160ms;">
      {#if results.length > 0}
        <div style="display: flex; flex-direction: column; gap: 20px;">
          <div class="text-sm font-medium" style="color: var(--ink-400);">
            {results.length} comparações
          </div>

          <!-- Visual comparison -->
          <div class="panel p-5">
            <div class="flex gap-3">
              {#each getSelectedPaints() as paint}
                <div style="flex: 1;">
                  <div class="swatch-flat" style="width: 100%; aspect-ratio: 1; background: rgb({paint.r}, {paint.g}, {paint.b});"></div>
                  <div style="text-align: center; margin-top: 10px;">
                    <div style="font-size: 12px; font-weight: 500; color: var(--paper); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{paint.name}</div>
                    <div style="font-size: 10px; font-family: var(--font-mono); color: var(--ink-500);">rgb({paint.r},{paint.g},{paint.b})</div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Delta E Results -->
          {#each results as result, i}
            <div class="panel result-row animate-slide" style="animation-delay: {i * 50}ms;">
              <div style="flex: 1;">
                <div class="text-sm font-medium text-white">{result.name}</div>
                <div style="font-size: 12px; color: var(--ink-500);">{result.manufacturer}</div>
              </div>
              <div style="text-align: right;">
                <div class="font-mono font-bold {deltaClass(result.deltaE)}" style="font-size: 18px;">
                  ΔE {result.deltaE.toFixed(2)}
                </div>
                <div style="font-size: 11px; color: var(--ink-500);">
                  {result.similarity.toFixed(1)}% similar
                </div>
              </div>
              <div style="width: 4px; height: 36px; border-radius: 999px; flex-shrink: 0; background: {deltaColor(result.deltaE)};"></div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="panel" style="text-align: center; padding: 80px 0;">
          <span style="color: var(--ink-600); display: flex; justify-content: center;"><Icon name="swap" size={40} /></span>
          <p class="font-medium mt-4" style="color: var(--ink-500);">Selecione tintas e clique em comparar</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .compare-layout {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: 24px;
  }

  .result-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 16px;
  }

  .selected-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px 4px 6px;
    border-radius: 6px;
    background: var(--ink-700);
    color: var(--paper);
    border: none;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s ease;
  }

  .selected-chip:hover {
    background: var(--ink-600);
  }

  .chip-swatch-sm {
    width: 12px;
    height: 12px;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .pick-list {
    max-height: 400px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .pick-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 10px;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s ease, border-color 0.15s ease;
  }

  .pick-item:hover {
    background: var(--ink-850);
  }

  .pick-item.selected {
    background: rgba(232, 84, 44, 0.08);
    border-color: rgba(232, 84, 44, 0.25);
  }

  .pick-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .pick-name {
    font-size: 13px;
    color: var(--ink-300);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pick-item.selected .pick-name {
    color: var(--paper);
  }

  .pick-code {
    font-size: 10.5px;
    font-family: var(--font-mono);
    color: var(--ink-600);
  }
</style>
