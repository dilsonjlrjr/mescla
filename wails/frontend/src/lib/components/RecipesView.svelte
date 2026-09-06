<script lang="ts">
  // T3 — Receitas (rf-04, tela nova). Lista à esquerda (recipes.svelte.ts,
  // fórmula salva — RG-16 "guardar o alvo" fica pendente, ver rf-04 Não faz);
  // detalhe à direita recalcula a fórmula no fabricante escolhido via
  // SuggestRecipeForColor(alvo, fabricante) — funciona igual pra alvo vindo
  // de um pote do catálogo (T1) ou de um pixel do plano da peça (T2), porque
  // usa sempre a cor (hex), nunca o paintId de origem.
  import { onMount } from 'svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type { ManufacturerDTO, EquivalentRecipeDTO } from '../../../bindings/paint-match-ai/models';
  import FormulaRibbon from './FormulaRibbon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import { recipes, removeRecipe, type SavedRecipe } from '../recipes.svelte';
  import { t, decimal } from '../i18n.svelte';
  import { rgbOfHex, computeSuggestedDrops, verdictKey, consequenceKey, isUnreachable } from '../color';

  let manufacturers: ManufacturerDTO[] = $state([]);
  let selectedId: number | null = $state(null);
  let targetMfrId: number = $state(0);
  let result = $state<EquivalentRecipeDTO | null>(null);
  let loading = $state(false);
  let confirmDeleteId: number | null = $state(null);

  interface BrandCompareRow { manufacturerId: number; manufacturerName: string; deltaE: number }
  let brandCompare: BrandCompareRow[] = $state([]);

  onMount(async () => {
    try {
      manufacturers = (await PaintService.GetManufacturers()) || [];
    } catch (e) {
      console.error('Erro carregando fabricantes:', e);
    }
    if (recipes.length > 0) selectRecipe(recipes[0]);
  });

  let selected = $derived(recipes.find(r => r.id === selectedId) ?? null);

  function hexToRgb(hex: string) {
    return rgbOfHex(hex) ?? { r: 138, g: 138, b: 138 };
  }

  function selectRecipe(r: SavedRecipe) {
    selectedId = r.id;
    targetMfrId = r.targetManufacturerId > 0 ? r.targetManufacturerId : (manufacturers[0]?.id ?? 0);
    recompute();
  }

  function onMfrChange(e: Event) {
    targetMfrId = Number((e.currentTarget as HTMLSelectElement).value);
    recompute();
  }

  async function recompute() {
    if (!selected || !targetMfrId) return;
    loading = true;
    result = null;
    brandCompare = [];
    const rgb = hexToRgb(selected.sourceHex);
    try {
      result = await PaintService.SuggestRecipeForColor(rgb.r, rgb.g, rgb.b, targetMfrId);
      const results = await Promise.all(manufacturers.map(async m => {
        try {
          const r = await PaintService.SuggestRecipeForColor(rgb.r, rgb.g, rgb.b, m.id);
          return { manufacturerId: m.id, manufacturerName: m.name, deltaE: r.deltaE };
        } catch {
          return null;
        }
      }));
      brandCompare = results
        .filter((x): x is BrandCompareRow => !!x)
        .sort((a, b) => a.deltaE - b.deltaE);
    } catch (e) {
      console.error('Erro recalculando receita:', e);
      toast('O cálculo falhou. Tente de novo.', 'error');
    } finally {
      loading = false;
    }
  }

  function pickBrandCompare(row: BrandCompareRow) {
    targetMfrId = row.manufacturerId;
    recompute();
  }

  function askDelete(id: number) {
    confirmDeleteId = id;
  }

  function confirmDelete() {
    if (confirmDeleteId === null) return;
    const wasSelected = confirmDeleteId === selectedId;
    removeRecipe(confirmDeleteId);
    confirmDeleteId = null;
    toast(t('recipeDeleted'));
    if (wasSelected) {
      selectedId = recipes[0]?.id ?? null;
      if (selectedId !== null) selectRecipe(recipes[0]);
      else result = null;
    }
  }

  let ingredients = $derived((result?.ingredients ?? []).filter(i => i.percentage > 0.5));
  let drops = $derived(ingredients.length ? computeSuggestedDrops(ingredients.map(i => i.percentage)) : []);
</script>

<div class="view-shell">
  <div class="recipes-body view-scroll">
    <aside class="recipes-list">
      <p class="label-mono list-title">{t('savedRecipes')}</p>
      {#if recipes.length === 0}
        <div class="empty-state">
          <p class="empty-title">{t('noRecipesYet')}</p>
          <p class="empty-hint">{t('noRecipesYetHint')}</p>
        </div>
      {:else}
        {#each recipes as r (r.id)}
          <button class="recipe-row" class:selected={selectedId === r.id} onclick={() => selectRecipe(r)}>
            <div class="recipe-ribbon">
              <FormulaRibbon segments={r.ingredients.map(i => ({ ...hexToRgb(i.hex), code: i.code, percentage: i.percentage }))} height={26} ruler={false} labels={false} />
            </div>
            <div class="recipe-id">
              <span class="recipe-name">{r.sourceName}</span>
              <span class="recipe-target font-mono">{t('reproduceIn')} {r.targetManufacturer}</span>
            </div>
            <span class="recipe-delta font-mono">{decimal(r.deltaE, 1)}</span>
            <span class="recipe-del" onclick={(e) => { e.stopPropagation(); askDelete(r.id); }} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && askDelete(r.id)} aria-label={t('del')}>×</span>
          </button>
        {/each}
      {/if}
    </aside>

    <section class="recipe-detail">
      {#if !selected}
        <p class="pick-hint">{t('pickRecipe')}</p>
      {:else}
        <div class="detail-head">
          <div>
            <h1 class="detail-title font-display">{selected.sourceName}</h1>
            <p class="detail-sub font-mono">{selected.sourceHex}</p>
          </div>
          <select class="mfr-select" value={targetMfrId} onchange={onMfrChange} aria-label={t('reproduceIn')}>
            {#each manufacturers as m (m.id)}
              <option value={m.id}>{m.name}</option>
            {/each}
          </select>
        </div>

        {#if loading}
          <p class="calc-line font-mono">{t('calculating')}</p>
        {:else if result}
          {#if isUnreachable(result.deltaE)}
            <div class="notice-card mb-6">
              <div class="notice-title">{t('unreachT')}</div>
              <p class="notice-text">{t('unreachB', { pool: t('poolBrand', { brand: result.targetManufacturer }), d: decimal(result.deltaE, 1) })}</p>
            </div>
          {/if}

          <p class="headline font-display">
            {ingredients.length <= 1 ? t('hPote') : t('hMix', { n: ingredients.length })}
          </p>
          <p class="consequence">{t(consequenceKey(result.deltaE))}</p>

          <div class="pair-row">
            <div class="pair-item">
              <span class="swatch-flat pair-swatch" style="background: rgb({result.sourceR}, {result.sourceG}, {result.sourceB});"></span>
              <span class="label-mono">{t('target')}</span>
            </div>
            <div class="pair-item">
              <span class="swatch-flat pair-swatch" style="background: rgb({result.resultR}, {result.resultG}, {result.resultB});"></span>
              <span class="label-mono">{t('mixture')}</span>
            </div>
            <div class="pair-delta">
              <span class="delta-reading pair-delta-n" class:good={result.deltaE < 2}>{decimal(result.deltaE, 1)}</span>
              <span class="delta-verdict" class:good={result.deltaE < 2}>{t(verdictKey(result.deltaE))}</span>
            </div>
          </div>

          <FormulaRibbon segments={ingredients.map(i => ({ r: i.r, g: i.g, b: i.b, code: i.code || i.name, percentage: i.percentage }))} />

          <div class="steps-list">
            {#each ingredients as ing, idx (ing.paintId + '-' + idx)}
              <div class="step-row">
                <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={40} />
                <div class="step-text">
                  <span class="step-name">{ing.name}</span>
                  <span class="step-meta font-mono">{ing.code ? `${ing.code} · ` : ''}{result.targetManufacturer}</span>
                </div>
                <span class="step-drops font-mono">{drops[idx]} {drops[idx] === 1 ? t('drop') : t('drops')}</span>
                <span class="step-pct font-display">{Math.round(ing.percentage)}%</span>
              </div>
            {/each}
          </div>

          {#if brandCompare.length > 0}
            <div class="compare-block">
              <p class="label-mono compare-title">{t('sameOtherBrands')}</p>
              {#each brandCompare as row (row.manufacturerId)}
                <button class="compare-row" onclick={() => pickBrandCompare(row)}>
                  <span class="compare-name">{row.manufacturerName}</span>
                  <span class="delta-reading compare-delta" class:good={row.deltaE < 2}>{decimal(row.deltaE, 1)}</span>
                </button>
              {/each}
            </div>
          {/if}
        {/if}
      {/if}
    </section>
  </div>
</div>

{#if confirmDeleteId !== null}
  <div class="confirm-overlay">
    <div class="confirm-dialog">
      <div class="notice-title">{t('delRecipeT')}</div>
      <p class="notice-text">{t('delRecipeB')}</p>
      <div class="confirm-actions">
        <button class="pill-light" onclick={() => (confirmDeleteId = null)}>{t('cancel')}</button>
        <button class="pill-dark" onclick={confirmDelete}>{t('del')}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .recipes-body {
    display: grid;
    grid-template-columns: minmax(280px, 34%) minmax(0, 1fr);
    gap: 0;
  }

  @media (max-width: 900px) {
    .recipes-body { grid-template-columns: 1fr; }
  }

  .recipes-list {
    padding: var(--space-6);
    border-right: 1px solid var(--color-divider);
  }

  .list-title {
    padding-bottom: var(--space-3);
    border-bottom: 1px solid var(--color-divider);
    margin-bottom: var(--space-2);
    display: block;
  }

  .recipe-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    min-height: 44px;
    padding: var(--space-3) var(--space-2);
    border: none;
    border-bottom: 1px solid var(--color-divider);
    background: transparent;
    cursor: pointer;
    text-align: left;
    font: inherit;
    color: var(--color-text);
  }

  .recipe-row:hover, .recipe-row.selected {
    background: color-mix(in srgb, var(--color-accent) 12%, transparent);
  }

  .recipe-ribbon {
    width: 90px;
    flex-shrink: 0;
  }

  .recipe-id {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .recipe-name {
    font-size: 13.5px;
    font-weight: 640;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recipe-target {
    font-size: 10.5px;
    color: var(--text-2);
  }

  .recipe-delta {
    font-size: 16px;
    flex-shrink: 0;
  }

  .recipe-del {
    flex-shrink: 0;
    min-width: 44px;
    min-height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-3);
    cursor: pointer;
    font-size: 16px;
  }

  .recipe-del:hover {
    color: var(--color-danger);
  }

  .recipe-detail {
    padding: var(--space-8) var(--space-6);
    max-width: 720px;
  }

  .pick-hint {
    color: var(--text-2);
    font-size: 14px;
  }

  .detail-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }

  .detail-title {
    font-size: 1.8rem;
    font-weight: 720;
    color: var(--color-text);
  }

  .detail-sub {
    font-size: 12px;
    color: var(--text-2);
    margin-top: var(--space-1);
  }

  .mfr-select {
    min-height: 44px;
    padding: 0 var(--space-4);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-pill);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }

  .calc-line {
    color: var(--text-2);
    font-size: 12.5px;
  }

  .headline {
    font-size: 1.4rem;
    font-weight: 700;
    color: var(--color-text);
    margin-bottom: var(--space-2);
  }

  .consequence {
    font-size: 13.5px;
    color: var(--text-2);
    margin-bottom: var(--space-6);
  }

  .pair-row {
    display: flex;
    align-items: flex-end;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }

  .pair-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .pair-swatch {
    width: 72px;
    height: 56px;
  }

  .pair-delta {
    margin-left: var(--space-4);
    display: flex;
    flex-direction: column;
  }

  .pair-delta-n {
    font-size: 34px;
  }

  .steps-list {
    margin-top: var(--space-6);
    display: flex;
    flex-direction: column;
  }

  .step-row {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) 0;
    border-bottom: 1px solid var(--color-divider);
  }

  .step-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .step-name {
    font-size: 14px;
    font-weight: 640;
    color: var(--color-text);
  }

  .step-meta {
    font-size: 11px;
    color: var(--text-2);
  }

  .step-drops {
    font-size: 12px;
    color: var(--text-2);
    flex-shrink: 0;
  }

  .step-pct {
    font-size: 22px;
    font-weight: 700;
    min-width: 60px;
    text-align: right;
    flex-shrink: 0;
  }

  .compare-block {
    margin-top: var(--space-8);
    border-top: 1px solid var(--color-divider);
    padding-top: var(--space-4);
  }

  .compare-title {
    margin-bottom: var(--space-3);
  }

  .compare-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    min-height: 44px;
    padding: var(--space-2);
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--color-text);
    cursor: pointer;
    font: inherit;
  }

  .compare-row:hover {
    background: color-mix(in srgb, var(--color-accent) 12%, transparent);
  }

  .compare-name {
    flex: 1;
    text-align: left;
    font-size: 13.5px;
    font-weight: 600;
  }

  .compare-delta {
    font-size: 16px;
  }

  .empty-state {
    padding: var(--space-6) 0;
  }

  .empty-title {
    font-size: 14px;
    font-weight: 700;
    color: var(--color-text);
    margin-bottom: var(--space-2);
  }

  .empty-hint {
    font-size: 12.5px;
    color: var(--text-2);
    line-height: 1.55;
  }
</style>
