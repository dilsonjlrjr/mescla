<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '@smui/button';
  import Card, { Content } from '@smui/card';
  import Checkbox from '@smui/checkbox';
  import FormField from '@smui/form-field';
  import List, { Item, Text, Graphic } from '@smui/list';

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
    if (de < 3) return '#2ecc71';
    if (de < 6) return '#f1c40f';
    if (de < 10) return '#e67e22';
    return '#e74c3c';
  }
</script>

<div class="page-container">
  <div class="page-header animate-artisan-fade">
    <h1 class="page-title">Comparar Tintas</h1>
    <p class="page-subtitle">Selecione até 6 tintas para comparar lado a lado</p>
    <div class="page-divider"></div>
  </div>

  <div class="compare-layout">
    <!-- Selection Panel -->
    <div class="artisan-panel p-5 animate-artisan-fade" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-2">Selecionar Tintas</h3>
      <div style="font-size: 11px; font-weight: 500; margin-bottom: 16px; color: var(--color-obsidian-500);">
        {selectedIDs.length}/6 selecionadas
      </div>

      <!-- Selected chips -->
      {#if selectedIDs.length > 0}
        <div class="flex gap-2 mb-4 flex-wrap">
          {#each getSelectedPaints() as paint}
            <button class="selected-chip" onclick={() => togglePaint(paint.id)}>
              <span class="chip-swatch" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
              <span class="truncate" style="max-width: 70px;">{paint.name}</span>
              <span class="material-icons" style="font-size: 14px; color: var(--color-obsidian-400);">close</span>
            </button>
          {/each}
        </div>
      {/if}

      <!-- Paint list -->
      <List twoLine style="max-height: 400px; overflow-y: auto;">
        {#each allPaints as paint (paint.id)}
          {@const isSelected = selectedIDs.includes(paint.id)}
          <Item
            href="javascript:void(0)"
            onclick={() => togglePaint(paint.id)}
            selected={isSelected}
            style="border-radius: 8px; margin: 2px 0; {isSelected ? 'background: rgba(212,160,83,0.08); border: 1px solid rgba(212,160,83,0.15);' : ''}"
          >
            <Graphic>
              <span style="width: 16px; height: 16px; border-radius: 4px; background: rgb({paint.r}, {paint.g}, {paint.b}); border: 1px solid rgba(255,255,255,0.08); display: block;"></span>
            </Graphic>
            <Text>
              <span style="color: {isSelected ? 'white' : 'var(--color-obsidian-300)'}; font-size: 13px;">{paint.name}</span>
              <span style="color: var(--color-obsidian-500); font-size: 11px; font-family: var(--font-mono);">{paint.code}</span>
            </Text>
          </Item>
        {/each}
      </List>

      <Button variant="raised" onclick={doCompare} disabled={selectedIDs.length < 2 || comparing} style="width: 100%; margin-top: 16px; background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm)); color: white; font-weight: 600; border-radius: 12px; height: 44px;">
        {comparing ? 'Comparando...' : 'Comparar Selecionadas'}
      </Button>
    </div>

    <!-- Results -->
    <div class="animate-artisan-fade" style="animation-delay: 160ms;">
      {#if results.length > 0}
        <div style="display: flex; flex-direction: column; gap: 20px;">
          <div class="text-sm font-medium" style="color: var(--color-obsidian-400);">
            {results.length} comparações
          </div>

          <!-- Visual comparison -->
          <div class="artisan-panel p-5">
            <div class="flex gap-3">
              {#each getSelectedPaints() as paint}
                <div style="flex: 1;">
                  <div class="swatch-shimmer" style="width: 100%; aspect-ratio: 1; border-radius: 12px; background: rgb({paint.r}, {paint.g}, {paint.b}); box-shadow: 0 4px 16px rgba(0,0,0,0.3);"></div>
                  <div style="text-align: center; margin-top: 10px;">
                    <div style="font-size: 12px; font-weight: 500; color: white; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{paint.name}</div>
                    <div style="font-size: 10px; font-family: var(--font-mono); color: var(--color-obsidian-500);">rgb({paint.r},{paint.g},{paint.b})</div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Delta E Results -->
          {#each results as result, i}
            <Card variant="outlined" class="result-card animate-artisan-slide" style="animation-delay: {i * 50}ms;">
              <Content>
                <div class="flex items-center gap-4">
                  <div style="flex: 1;">
                    <div class="text-sm font-medium text-white">{result.name}</div>
                    <div style="font-size: 12px; color: var(--color-obsidian-400);">{result.manufacturer}</div>
                  </div>
                  <div style="text-align: right;">
                    <div class="font-mono font-bold {deltaClass(result.deltaE)}" style="font-size: 18px;">
                      ΔE {result.deltaE.toFixed(2)}
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
          <span class="material-icons" style="color: var(--color-obsidian-600); font-size: 48px;">compare_arrows</span>
          <p class="font-medium mt-4" style="color: var(--color-obsidian-500);">Selecione tintas e clique em comparar</p>
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

  .selected-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px 4px 6px;
    border-radius: 8px;
    background: var(--color-obsidian-700);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.08);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .selected-chip:hover {
    background: var(--color-obsidian-600);
    transform: scale(1.05);
  }

  .chip-swatch {
    width: 12px;
    height: 12px;
    border-radius: 3px;
    flex-shrink: 0;
  }
</style>
