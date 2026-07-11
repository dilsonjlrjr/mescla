<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '@smui/button';
  import Icon from './Icon.svelte';
  import DeltaBadge from './DeltaBadge.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

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

      <Button variant="raised" onclick={doCompare} disabled={selectedIDs.length < 2 || comparing} style="width: 100%; margin-top: 16px; background: var(--lacquer); color: white; font-weight: 600; border-radius: var(--radius-pill); height: 44px;">
        {comparing ? 'Comparando…' : 'Comparar selecionadas'}
      </Button>
    </div>

    <!-- Results -->
    <div class="animate-rise" style="animation-delay: 160ms;">
      {#if results.length > 0}
        <div style="display: flex; flex-direction: column; gap: 20px;">
          <div class="font-mono" style="font-size: 11.5px; color: var(--ink-500);">
            {results.length} comparações
          </div>

          <!-- Veredito visual: as cores COLADAS, junção limpa — nada por cima -->
          <div class="panel p-5">
            <div class="compare-strip" style="grid-template-columns: repeat({getSelectedPaints().length}, 1fr);">
              {#each getSelectedPaints() as paint}
                <div class="compare-band" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></div>
              {/each}
            </div>
            <div class="compare-labels" style="grid-template-columns: repeat({getSelectedPaints().length}, 1fr);">
              {#each getSelectedPaints() as paint}
                <div class="compare-label">
                  <div class="compare-name">{paint.name}</div>
                  <div class="compare-meta font-mono">rgb({paint.r},{paint.g},{paint.b})</div>
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
              <DeltaBadge deltaE={result.deltaE} />
            </div>
          {/each}
        </div>
      {:else}
        <div class="panel" style="text-align: center; padding: 80px 0;">
          <span style="color: var(--ink-500); display: flex; justify-content: center;"><Icon name="swap" size={40} /></span>
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
    padding: 4px 10px 4px 6px;
    border-radius: var(--radius-pill);
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
    color: var(--ink-500);
  }

  /* Faixa de comparação: metades adjacentes, sem gap — a junção fica limpa */
  .compare-strip {
    display: grid;
    height: 120px;
    border-radius: var(--radius-surface);
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.25);
  }

  .compare-band {
    min-width: 0;
  }

  /* Etiquetas ABAIXO de cada faixa, alinhadas às colunas de cor */
  .compare-labels {
    display: grid;
    gap: 0;
    margin-top: 10px;
  }

  .compare-label {
    min-width: 0;
    padding: 0 6px;
    text-align: center;
  }

  .compare-name {
    font-size: 12px;
    font-weight: 500;
    color: var(--paper);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .compare-meta {
    font-size: 10px;
    color: var(--ink-500);
  }
</style>
