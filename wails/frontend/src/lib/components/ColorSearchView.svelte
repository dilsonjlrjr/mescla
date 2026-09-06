<script lang="ts">
  // T1 — Pergunta e resposta (rf-04). Busca única (nome/código/hex) → alvo →
  // resposta em 2 colunas (manchete+consequência+par ΔE00 | pote pronto OU
  // fórmula com fita de proporção e gotas ajustáveis). RG-12: o ajuste manual
  // de gotas recalcula localmente (color.ts), sem novo request ao solver.
  //
  // Limitação herdada da API exposta pelo Wails (declarada, não escondida):
  // não existe binding para "melhor mistura cross-brand com o catálogo
  // inteiro" nem para "mistura só com o estoque" quando o alvo é uma cor
  // livre (hex sem pote correspondente) — só SuggestEquivalentFromStock, que
  // exige um paintId de catálogo. Nesses dois casos a tela mostra o melhor
  // POTE ÚNICO disponível no pool (via nearest-neighbour sobre o catálogo) e
  // avisa a limitação (stockOnlyNoFormula), em vez de fingir uma fórmula.
  import { onMount } from 'svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type {
    PaintDTO, ManufacturerDTO, UserPaintDTO, EquivalentRecipeDTO,
  } from '../../../bindings/paint-match-ai/models';
  import FormulaRibbon from './FormulaRibbon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import { saveRecipe } from '../recipes.svelte';
  import { sessionHistory, pushHistory, type HistoryEntry } from '../sessionHistory.svelte';
  import { t, decimal } from '../i18n.svelte';
  import {
    rgbToLab, deltaE2000, mixDropsLinear, computeSuggestedDrops, dropsToMl,
    verdictKey, consequenceKey, isUnreachable, hexOfRgb, rgbOfHex, type RGB,
  } from '../color';

  type View =
    | 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare'
    | 'mix' | 'wheel' | 'stock' | 'planner' | 'receitas';
  interface NavOpts { stockPrefillPaintId?: number }
  interface Props { onNavigate?: (view: View, opts?: number | NavOpts) => void }
  let { onNavigate }: Props = $props();

  interface Target { paintId?: number; name: string; r: number; g: number; b: number; hex: string }

  let allPaints: PaintDTO[] = $state([]);
  let manufacturers: ManufacturerDTO[] = $state([]);
  let userPaints: UserPaintDTO[] = $state([]);
  let loading = $state(true);

  let query = $state('');
  let inputFocused = $state(false);
  let target: Target | null = $state(null);
  let askedLabel = $state('');

  let onlyStock = $state(false);
  let brandFilter: number | 'all' = $state('all');
  let unit: 'drops' | 'ml' = $state('drops');
  let wantMixAnyway = $state(false);

  type ResultKind = 'none' | 'pote' | 'formula' | 'empty-pool' | 'empty-stock' | 'no-result' | 'error';
  let resultKind = $state<ResultKind>('none');
  let loadingResult = $state(false);
  let errorMsg = $state('');

  let potePaint = $state<{ paintId: number; name: string; manufacturer: string; code: string; r: number; g: number; b: number; deltaE: number } | null>(null);
  let formula = $state<EquivalentRecipeDTO | null>(null);
  let fromStock = $state(false);
  let manualDrops: number[] = $state([]);

  interface BrandCompareRow { manufacturerId: number; manufacturerName: string; deltaE: number; kind: 'pote' | 'mix' }
  let brandCompare: BrandCompareRow[] = $state([]);

  onMount(async () => {
    try {
      const [paints, mfrs, stock] = await Promise.all([
        PaintService.GetAllPaints(),
        PaintService.GetManufacturers(),
        PaintService.GetUserPaints(),
      ]);
      allPaints = paints || [];
      manufacturers = mfrs || [];
      userPaints = stock || [];
    } catch (e) {
      console.error('Erro carregando catálogo:', e);
      toast(String(e), 'error');
    } finally {
      loading = false;
    }
  });

  let stockKeys = $derived(
    new Set(userPaints.filter(p => p.code.trim()).map(p => `${p.manufacturer.toLowerCase()}|${p.code.trim().toLowerCase()}`))
  );
  let stockById = $derived(new Map(userPaints.map(p => [p.id, p])));
  let paintById = $derived(new Map(allPaints.map(p => [p.id, p])));

  function mfrName(id: number): string {
    return manufacturers.find(m => m.id === id)?.name ?? '';
  }
  function mfrIdByName(name: string): number {
    return manufacturers.find(m => m.name === name)?.id ?? 0;
  }

  let suggestOpen = $derived(inputFocused && !loading);
  let suggestions = $derived.by(() => {
    const q = query.trim().toLowerCase();
    const pool = q
      ? allPaints.filter(p => p.name.toLowerCase().includes(q) || p.code.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q))
      : allPaints;
    return pool.slice(0, 8);
  });

  let swatchGrid = $derived(allPaints.slice(0, 60));

  // ── Resolver o alvo a partir do texto digitado (US-01) ──
  function findByCode(q: string): PaintDTO | undefined {
    const qq = q.toLowerCase();
    return allPaints.find(p => p.code.trim().toLowerCase() === qq);
  }
  function findByNameExact(q: string): PaintDTO | undefined {
    const qq = q.toLowerCase();
    return allPaints.find(p => p.name.trim().toLowerCase() === qq);
  }
  function findByNamePartial(q: string): PaintDTO | undefined {
    const qq = q.toLowerCase();
    return allPaints.find(p => p.name.toLowerCase().includes(qq) || p.code.toLowerCase().includes(qq));
  }

  function setTargetFromPaint(p: PaintDTO, raw: string) {
    const hex = hexOfRgb(p.r, p.g, p.b);
    target = { paintId: p.id, name: p.name, r: p.r, g: p.g, b: p.b, hex };
    askedLabel = raw;
    wantMixAnyway = false;
    inputFocused = false;
    pushHistory({ label: p.name, hex, r: p.r, g: p.g, b: p.b, paintId: p.id });
    computeResult();
  }

  function setTargetFromHex(rgb: RGB, hexU: string) {
    target = { name: '', r: rgb.r, g: rgb.g, b: rgb.b, hex: hexU };
    askedLabel = hexU;
    wantMixAnyway = false;
    inputFocused = false;
    pushHistory({ label: hexU, hex: hexU, r: rgb.r, g: rgb.g, b: rgb.b });
    computeResult();
  }

  function runSearch(raw: string) {
    const q = raw.trim();
    if (!q) return;
    const paint = findByCode(q) || findByNameExact(q) || findByNamePartial(q);
    if (paint) { setTargetFromPaint(paint, q); return; }
    const rgb = rgbOfHex(q);
    if (rgb) { setTargetFromHex(rgb, hexOfRgb(rgb.r, rgb.g, rgb.b)); return; }
    target = null;
    resultKind = 'no-result';
    askedLabel = q.toUpperCase();
    inputFocused = false;
  }

  function onSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') { e.preventDefault(); runSearch(query); }
    else if (e.key === 'Escape') { inputFocused = false; (e.currentTarget as HTMLInputElement).blur(); }
  }

  function selectSwatch(p: PaintDTO) {
    query = p.code || p.name;
    setTargetFromPaint(p, query);
  }

  function selectHistoryEntry(h: HistoryEntry) {
    query = h.label;
    if (h.paintId !== undefined) {
      const p = paintById.get(h.paintId);
      if (p) { setTargetFromPaint(p, h.label); return; }
    }
    setTargetFromHex({ r: h.r, g: h.g, b: h.b }, h.hex);
  }

  // ── Filtros (US-09/US-14) ──
  function toggleOnlyStock() {
    onlyStock = !onlyStock;
    if (target) computeResult();
  }
  function setBrand(b: number | 'all') {
    brandFilter = b;
    if (target) computeResult();
  }
  function useAllBrandsFallback() {
    onlyStock = false;
    if (target) computeResult();
  }
  function toggleWantMix() {
    wantMixAnyway = !wantMixAnyway;
    if (target) computeResult();
  }

  async function suggestForBrand(mfrId: number): Promise<EquivalentRecipeDTO> {
    if (target?.paintId) return PaintService.SuggestEquivalentRecipe(target.paintId, mfrId);
    return PaintService.SuggestRecipeForColor(target!.r, target!.g, target!.b, mfrId);
  }

  // ── RG-07: pool por precedência — marca forçada > só o que eu tenho > cross-brand ──
  async function computeResult() {
    if (!target) return;
    loadingResult = true;
    errorMsg = '';
    brandCompare = [];
    formula = null;
    potePaint = null;
    fromStock = false;
    try {
      const found = (await PaintService.FindSimilar(target.r, target.g, target.b, 100, 40)) || [];
      let candidates = found
        .map(f => ({ ...f, code: paintById.get(f.paintId)?.code ?? '' }))
        .sort((a, b) => a.deltaE - b.deltaE);

      if (brandFilter !== 'all') {
        const name = mfrName(brandFilter);
        candidates = candidates.filter(c => c.manufacturer === name);
      } else if (onlyStock) {
        candidates = candidates.filter(c => stockKeys.has(`${c.manufacturer.toLowerCase()}|${c.code.trim().toLowerCase()}`));
      }

      if (onlyStock && brandFilter === 'all' && userPaints.length === 0) {
        resultKind = 'empty-stock';
        return;
      }

      const best = candidates[0];
      if (best && best.deltaE < 2.5 && !wantMixAnyway) {
        resultKind = 'pote';
        potePaint = best;
        return;
      }

      if (brandFilter !== 'all') {
        formula = await suggestForBrand(brandFilter);
      } else if (onlyStock) {
        if (target.paintId) {
          formula = await PaintService.SuggestEquivalentFromStock(target.paintId);
          fromStock = true;
        } else {
          formula = null;
        }
      } else if (manufacturers.length > 0) {
        const results = await Promise.all(manufacturers.map(async m => {
          try {
            const r = await suggestForBrand(m.id);
            return { mfrId: m.id, mfrName: m.name, r };
          } catch {
            return null;
          }
        }));
        const ok = results.filter((x): x is { mfrId: number; mfrName: string; r: EquivalentRecipeDTO } => !!x);
        ok.sort((a, b) => a.r.deltaE - b.r.deltaE);
        brandCompare = ok.map(x => ({
          manufacturerId: x.mfrId,
          manufacturerName: x.mfrName,
          deltaE: x.r.deltaE,
          kind: (x.r.ingredients ?? []).filter(i => i.percentage > 0.5).length <= 1 ? 'pote' : 'mix',
        }));
        formula = ok[0]?.r ?? null;
      }

      const ingCount = (formula?.ingredients ?? []).filter(i => i.percentage > 0.5).length;
      if (!formula || ingCount === 0) {
        resultKind = onlyStock ? 'empty-stock' : 'empty-pool';
        return;
      }

      const filtered = formula.ingredients!.filter(i => i.percentage > 0.5);
      manualDrops = computeSuggestedDrops(filtered.map(i => i.percentage));
      resultKind = 'formula';
    } catch (e) {
      console.error('Erro calculando resposta:', e);
      errorMsg = 'O cálculo falhou.';
      toast('O cálculo falhou. Tente de novo.', 'error');
      resultKind = 'error';
    } finally {
      loadingResult = false;
    }
  }

  let ingredients = $derived((formula?.ingredients ?? []).filter(i => i.percentage > 0.5));

  let liveMix: RGB | null = $derived.by(() => {
    if (!ingredients.length) return null;
    return mixDropsLinear(ingredients.map((i, idx) => ({ r: i.r, g: i.g, b: i.b, drops: manualDrops[idx] ?? 1 })));
  });

  let liveDeltaE = $derived.by(() => {
    if (!target || !liveMix) return formula?.deltaE ?? 0;
    return deltaE2000(rgbToLab(target.r, target.g, target.b), rgbToLab(liveMix.r, liveMix.g, liveMix.b));
  });

  let totalDrops = $derived(manualDrops.reduce((a, b) => a + b, 0));

  function unitValue(drops: number): string {
    return unit === 'drops' ? `${drops} ${drops === 1 ? t('drop') : t('drops')}` : `${decimal(dropsToMl(drops), 2)} ${t('unitMl')}`;
  }

  function incDrop(i: number) { manualDrops[i] = Math.min(40, (manualDrops[i] ?? 1) + 1); }
  function decDrop(i: number) { manualDrops[i] = Math.max(1, (manualDrops[i] ?? 1) - 1); }
  function setDrop(i: number, v: number) { manualDrops[i] = Math.max(1, Math.min(40, v)); }
  function resetDrops() { manualDrops = computeSuggestedDrops(ingredients.map(i => i.percentage)); }

  function ingredientOwned(ing: { code: string }): boolean {
    if (fromStock) return true;
    const mfr = formula?.targetManufacturer ?? '';
    return stockKeys.has(`${mfr.toLowerCase()}|${(ing.code || '').trim().toLowerCase()}`);
  }

  async function toggleIngredientOwn(ing: { name: string; code: string; r: number; g: number; b: number }) {
    const mfr = formula?.targetManufacturer ?? '';
    const mfrId = mfrIdByName(mfr);
    const existing = userPaints.find(p => p.manufacturer.toLowerCase() === mfr.toLowerCase() && p.code.trim().toLowerCase() === (ing.code || '').trim().toLowerCase());
    try {
      if (existing) {
        await PaintService.DeleteUserPaint(existing.id);
      } else if (mfrId) {
        await PaintService.AddUserPaint({ id: 0, manufacturerId: mfrId, manufacturer: '', name: ing.name, code: ing.code || '', r: ing.r, g: ing.g, b: ing.b, volume: '', notes: '' });
      }
      userPaints = (await PaintService.GetUserPaints()) || [];
    } catch (e) {
      console.error('Erro marcando posse:', e);
      toast(String(e), 'error');
    }
  }

  function pickBrandCompare(row: BrandCompareRow) {
    brandFilter = row.manufacturerId;
    wantMixAnyway = false;
    computeResult();
  }

  function saveFormulaAsRecipe() {
    if (!target || !formula || !ingredients.length) return;
    saveRecipe({
      sourcePaintId: target.paintId ?? -Date.now(),
      sourceName: target.name || askedLabel,
      sourceHex: target.hex,
      targetManufacturerId: fromStock ? 0 : (brandFilter !== 'all' ? brandFilter : mfrIdByName(formula.targetManufacturer)),
      targetManufacturer: fromStock ? 'meu estoque' : formula.targetManufacturer,
      ingredients: ingredients.map(i => ({ name: i.name, code: i.code || '', hex: hexOfRgb(i.r, i.g, i.b), percentage: i.percentage })),
      deltaE: formula.deltaE,
    });
    toast(t('recipeSaved'));
  }

  function goStock() { onNavigate?.('stock'); }

  function poolLabel(): string {
    if (fromStock) return t('poolCross');
    const brand = brandFilter !== 'all' ? mfrName(brandFilter) : (formula?.targetManufacturer ?? '');
    return t('poolBrand', { brand });
  }

  let answerHeadline = $derived.by(() => {
    if (!target) return '';
    if (resultKind === 'no-result') return t('noResultH');
    if (resultKind === 'empty-stock') return onlyStock ? t('noneTitleStock') : t('hNoRegistered');
    if (resultKind === 'pote' && potePaint) return t('hPote');
    if (resultKind === 'formula' && formula) {
      const n = ingredients.length;
      if (fromStock) {
        const brands = new Set(ingredients.map(i => stockById.get(i.paintId)?.manufacturer).filter((x): x is string => !!x));
        if (brands.size > 1) return t('hMixBrands', { n, b: brands.size });
      }
      return t('hMix', { n });
    }
    if (resultKind === 'empty-pool') {
      if (brandFilter !== 'all') return t('hNoneBrand', { brand: mfrName(brandFilter) });
      return t('hNoneCross');
    }
    return '';
  });

  let answerDeltaE = $derived(resultKind === 'pote' && potePaint ? potePaint.deltaE : (resultKind === 'formula' ? liveDeltaE : 0));
  let answerConsequence = $derived(resultKind === 'pote' || resultKind === 'formula' ? t(consequenceKey(answerDeltaE)) : '');
  let mixSwatch: RGB | null = $derived(resultKind === 'pote' && potePaint ? { r: potePaint.r, g: potePaint.g, b: potePaint.b } : (resultKind === 'formula' ? liveMix : null));
</script>

<div class="view-shell">
  <div class="t1-head view-fixed">
    <div class="search-wrap">
      <input
        type="search"
        class="t1-search font-mono"
        bind:value={query}
        placeholder={t('phSearch')}
        maxlength="120"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        onfocus={() => (inputFocused = true)}
        onblur={() => setTimeout(() => (inputFocused = false), 140)}
        onkeydown={onSearchKeydown}
        aria-label={t('phSearch')}
      />
      {#if suggestOpen && suggestions.length > 0}
        <div class="suggest-dropdown">
          <p class="label-mono suggest-title">{t('suggestTitle')}</p>
          {#each suggestions as p (p.id)}
            <button class="suggest-row" onmousedown={() => selectSwatch(p)}>
              <span class="swatch-flat" style="width: 30px; height: 30px; background: rgb({p.r}, {p.g}, {p.b});"></span>
              <span class="suggest-text">
                <span class="suggest-name">{p.name}</span>
                <span class="suggest-meta font-mono">{p.code ? `${p.code} · ` : ''}{p.manufacturer}</span>
              </span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <div class="filters-row">
      <button class="filter-pill" class:active={onlyStock} onclick={toggleOnlyStock} aria-pressed={onlyStock}>
        {t('onlyStock')} · {userPaints.length}
      </button>
      <div class="brand-pills">
        <button class="filter-pill" class:active={brandFilter === 'all'} onclick={() => setBrand('all')}>{t('allBrands')}</button>
        {#each manufacturers as m (m.id)}
          <button class="filter-pill" class:active={brandFilter === m.id} onclick={() => setBrand(m.id)}>{m.name}</button>
        {/each}
      </div>
      <div class="unit-seg" role="group" aria-label={t('labelUnit')}>
        <button class:active={unit === 'drops'} onclick={() => (unit = 'drops')} aria-pressed={unit === 'drops'}>{t('unitDrops')}</button>
        <button class:active={unit === 'ml'} onclick={() => (unit = 'ml')} aria-pressed={unit === 'ml'}>{t('unitMl')}</button>
      </div>
    </div>
  </div>

  <div class="view-scroll t1-scroll">
    {#if !target}
      <div class="empty-hero animate-rise">
        <h1 class="empty-h font-display">{t('emptyH')}</h1>
        <p class="empty-p">{t('emptyP')}</p>
      </div>
      <p class="label-mono swatch-title">{t('swatchGridTitle')}</p>
      <div class="swatch-grid">
        {#each swatchGrid as p (p.id)}
          <button class="swatch-cell" style="background: rgb({p.r}, {p.g}, {p.b});" title="{p.name} · {p.code} · {p.manufacturer} — {t('tapToSearch')}" onclick={() => selectSwatch(p)} aria-label={p.name}></button>
        {/each}
      </div>
    {:else if resultKind === 'no-result'}
      <div class="notice-card animate-rise">
        <div class="notice-title">{t('noResultH')}</div>
        <p class="notice-text">{t('noResultBody')}</p>
      </div>
    {:else}
      <div class="answer-grid animate-rise">
        <div class="answer-left">
          <p class="label-mono">{t('answerKicker')}</p>
          <h1 class="answer-headline font-display">{answerHeadline}</h1>
          {#if answerConsequence}<p class="answer-consequence">{answerConsequence}</p>{/if}

          {#if resultKind === 'pote' || resultKind === 'formula'}
            <div class="pair-row">
              <div class="pair-item">
                <span class="swatch-flat pair-swatch" style="background: rgb({target.r}, {target.g}, {target.b});"></span>
                <span class="label-mono">{t('target')}</span>
              </div>
              <div class="pair-item">
                {#if mixSwatch}
                  <span class="swatch-flat pair-swatch" style="background: rgb({mixSwatch.r}, {mixSwatch.g}, {mixSwatch.b});"></span>
                {/if}
                <span class="label-mono">{t('mixture')}</span>
              </div>
              <div class="pair-delta">
                <span class="delta-reading pair-delta-n" class:good={answerDeltaE < 2}>{decimal(answerDeltaE, 1)}</span>
                <span class="delta-verdict" class:good={answerDeltaE < 2}>{t(verdictKey(answerDeltaE))}</span>
              </div>
            </div>
          {/if}

          <p class="asked-line font-mono">{t('youAsked')}: {askedLabel}</p>
        </div>

        <div class="answer-right">
          {#if loadingResult}
            <p class="font-mono calc-line">{t('loadingNote', { brand: brandFilter !== 'all' ? mfrName(brandFilter) : t('allBrands') })}</p>
          {:else if resultKind === 'empty-stock'}
            <div class="notice-card">
              <div class="notice-title">{onlyStock ? t('noneTitleStock') : t('hNoRegistered')}</div>
              <p class="notice-text">{onlyStock ? t('noneBodyStock') : t('noneRegHere')}</p>
              <div class="notice-actions">
                <button class="pill-dark" onclick={goStock}>{t('addPaint')}</button>
                {#if onlyStock}<button class="pill-light" onclick={useAllBrandsFallback}>{t('useAllBrands')}</button>{/if}
              </div>
            </div>
          {:else if resultKind === 'empty-pool'}
            <div class="notice-card">
              <div class="notice-title">{answerHeadline}</div>
              <p class="notice-text">{t('chooseOther')}</p>
              <div class="notice-actions">
                <button class="pill-dark" onclick={goStock}>{t('addPaint')}</button>
              </div>
            </div>
          {:else if resultKind === 'pote' && potePaint}
            <div class="pote-card">
              <PaintBottle r={potePaint.r} g={potePaint.g} b={potePaint.b} size={96} />
              <div class="pote-text">
                <h2 class="pote-name font-display">{potePaint.name}</h2>
                <p class="pote-meta font-mono">{potePaint.code ? `${potePaint.code} · ` : ''}{potePaint.manufacturer}</p>
                <p class="pote-note">{t('mixWell')}</p>
              </div>
              <div class="pote-actions">
                <button class="pill-light" onclick={toggleWantMix}>{t('wantMix')}</button>
              </div>
            </div>
          {:else if resultKind === 'formula' && formula}
            {#if wantMixAnyway}
              <button class="back-to-pot font-mono" onclick={toggleWantMix}>‹ {t('backToPot')}</button>
            {/if}

            {#if isUnreachable(liveDeltaE)}
              <div class="notice-card mb-6">
                <div class="notice-title">{t('unreachT')}</div>
                <p class="notice-text">{t('unreachB', { pool: poolLabel(), d: decimal(liveDeltaE, 1) })}</p>
              </div>
            {/if}

            {#if onlyStock && brandFilter === 'all' && !target.paintId}
              <div class="notice-card mb-6">
                <p class="notice-text">{t('stockOnlyNoFormula')}</p>
              </div>
            {/if}

            <p class="formula-title font-mono">
              {ingredients.length} {t('colors')}{fromStock ? '' : ` · ${formula.targetManufacturer}`}
            </p>

            <FormulaRibbon segments={ingredients.map((i, idx) => ({ r: i.r, g: i.g, b: i.b, code: i.code || i.name, percentage: (manualDrops[idx] / (totalDrops || 1)) * 100 }))} />

            <div class="ingredient-list">
              {#each ingredients as ing, idx (ing.paintId + '-' + idx)}
                <div class="ing-row">
                  <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={44} />
                  <div class="ing-text">
                    <span class="ing-name">{ing.name}</span>
                    <span class="ing-meta font-mono">{ing.code ? `${ing.code} · ` : ''}{fromStock ? (stockById.get(ing.paintId)?.manufacturer ?? '') : formula.targetManufacturer}</span>
                  </div>
                  <label class="ing-own">
                    <input type="checkbox" checked={ingredientOwned(ing)} onchange={() => toggleIngredientOwn(ing)} aria-label={t('ariaHave')} />
                    {t('haveOnShelf')}
                  </label>
                  <div class="ing-stepper">
                    <button class="stepper-btn" onclick={() => decDrop(idx)} aria-label={t('ariaMinus')}>−</button>
                    <input
                      type="range"
                      min="1"
                      max="40"
                      value={manualDrops[idx] ?? 1}
                      oninput={(e) => setDrop(idx, Number((e.currentTarget as HTMLInputElement).value))}
                      aria-label={ing.name}
                    />
                    <button class="stepper-btn" onclick={() => incDrop(idx)} aria-label={t('ariaPlus')}>+</button>
                  </div>
                  <span class="ing-unit font-mono">{unitValue(manualDrops[idx] ?? 1)}</span>
                </div>
              {/each}
            </div>

            <div class="formula-foot">
              <span class="total-line font-mono">{t('totalLabel', { v: unitValue(totalDrops) })}</span>
              <button class="reset-link" onclick={resetDrops}>{t('resetProp')}</button>
            </div>

            <div class="formula-actions">
              <button class="pill-dark" onclick={saveFormulaAsRecipe}>{t('saveRecipeBtn')}</button>
            </div>

            {#if brandCompare.length > 0}
              <div class="compare-block">
                <p class="label-mono compare-title">{t('sameOtherBrand')}</p>
                {#each brandCompare as row (row.manufacturerId)}
                  <button class="compare-row" onclick={() => pickBrandCompare(row)}>
                    <span class="compare-name">{row.manufacturerName}</span>
                    <span class="compare-kind font-mono">{row.kind === 'pote' ? t('kindPote') : t('kindMix')}</span>
                    <span class="delta-reading compare-delta" class:good={row.deltaE < 2}>{decimal(row.deltaE, 1)}</span>
                  </button>
                {/each}
              </div>
            {:else if !loadingResult}
              <p class="compare-empty">{t('compareBrandsEmpty')}</p>
            {/if}
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <div class="t1-foot view-fixed">
    <span class="label-mono foot-label">{t('onTable')}</span>
    {#if sessionHistory.length === 0}
      <span class="foot-empty">{t('sessionEmpty')}</span>
    {:else}
      <div class="foot-scroll">
        {#each sessionHistory as h (h.id)}
          <button class="foot-chip" onclick={() => selectHistoryEntry(h)} title={h.label}>
            <span class="foot-swatch" style="background: rgb({h.r}, {h.g}, {h.b});"></span>
            <span class="foot-chip-label">{h.label}</span>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .t1-head {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-4) var(--space-6);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-divider);
  }

  .search-wrap {
    position: relative;
  }

  .t1-search {
    width: 100%;
    height: 48px;
    padding: 0 var(--space-4);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-md);
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 15px;
  }

  .suggest-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 6;
    max-height: 320px;
    overflow-y: auto;
    padding: var(--space-2);
    background: var(--color-surface);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
  }

  .suggest-title {
    padding: var(--space-2) var(--space-3);
  }

  .suggest-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    min-height: 44px;
    padding: var(--space-2) var(--space-3);
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    text-align: left;
    font: inherit;
    color: var(--color-text);
    cursor: pointer;
  }

  .suggest-row:hover {
    background: color-mix(in srgb, var(--color-accent) 14%, transparent);
  }

  .suggest-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .suggest-name {
    font-size: 13.5px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .suggest-meta {
    font-size: 10.5px;
    color: var(--text-2);
  }

  .filters-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .brand-pills {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
    flex: 1;
    min-width: 0;
  }

  .unit-seg {
    display: flex;
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-pill);
    overflow: hidden;
    flex-shrink: 0;
  }

  .unit-seg button {
    min-height: 44px;
    padding: 0 var(--space-4);
    border: none;
    background: var(--color-bg);
    color: var(--text-2);
    font: inherit;
    font-size: 12.5px;
    font-weight: 600;
    cursor: pointer;
  }

  .unit-seg button.active {
    background: var(--color-accent-600);
    color: var(--color-neutral-100);
  }

  .t1-scroll {
    padding: var(--space-8) var(--space-6);
  }

  .empty-hero {
    max-width: 560px;
    margin: 0 auto var(--space-8);
    text-align: center;
  }

  .empty-h {
    font-size: clamp(1.8rem, 3.4vw, 2.4rem);
    font-weight: 720;
    color: var(--color-text);
    margin-bottom: var(--space-3);
  }

  .empty-p {
    font-size: 14px;
    color: var(--text-2);
    line-height: 1.6;
  }

  .swatch-title {
    text-align: center;
    margin-bottom: var(--space-4);
  }

  .swatch-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(52px, 1fr));
    gap: var(--space-2);
    max-width: 900px;
    margin: 0 auto;
  }

  .swatch-cell {
    aspect-ratio: 1;
    min-width: 44px;
    min-height: 44px;
    border: none;
    border-radius: var(--radius-sm);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
    cursor: pointer;
    transition: transform 0.12s ease;
  }

  .swatch-cell:hover {
    transform: scale(1.08);
  }

  .answer-grid {
    display: grid;
    grid-template-columns: minmax(280px, 38%) minmax(0, 1fr);
    gap: var(--space-8);
    max-width: 1180px;
    margin: 0 auto;
  }

  @media (max-width: 900px) {
    .answer-grid { grid-template-columns: 1fr; }
  }

  .answer-headline {
    font-size: clamp(1.6rem, 3vw, 2.2rem);
    font-weight: 720;
    color: var(--color-text);
    line-height: 1.12;
    margin: var(--space-2) 0 var(--space-3);
  }

  .answer-consequence {
    font-size: 14px;
    color: var(--text-2);
    line-height: 1.55;
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
    width: 84px;
    height: 64px;
  }

  .pair-delta {
    margin-left: var(--space-4);
    display: flex;
    flex-direction: column;
  }

  .pair-delta-n {
    font-size: 40px;
  }

  .asked-line {
    font-size: 11.5px;
    color: var(--text-3);
  }

  .calc-line {
    font-size: 12.5px;
    color: var(--text-2);
  }

  .notice-actions {
    display: flex;
    gap: var(--space-3);
    margin-top: var(--space-3);
  }

  .pote-card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-4);
    padding: var(--space-6);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
  }

  .pote-name {
    font-size: 22px;
    font-weight: 700;
    color: var(--color-text);
  }

  .pote-meta {
    font-size: 12px;
    color: var(--text-2);
    margin-top: var(--space-1);
  }

  .pote-note {
    font-size: 13px;
    color: var(--text-2);
    margin-top: var(--space-3);
    max-width: 340px;
  }

  .back-to-pot {
    border: none;
    background: none;
    color: var(--color-accent-2);
    font-size: 12px;
    cursor: pointer;
    padding: 0;
    margin-bottom: var(--space-4);
  }

  .back-to-pot:hover { text-decoration: underline; }

  .formula-title {
    font-size: 12.5px;
    color: var(--text-2);
    margin-bottom: var(--space-3);
  }

  .ingredient-list {
    margin-top: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .ing-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) 0;
    border-bottom: 1px solid var(--color-divider);
    flex-wrap: wrap;
  }

  .ing-text {
    display: flex;
    flex-direction: column;
    min-width: 120px;
    flex: 1;
  }

  .ing-name {
    font-size: 14px;
    font-weight: 640;
    color: var(--color-text);
  }

  .ing-meta {
    font-size: 11px;
    color: var(--text-2);
  }

  .ing-own {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 11.5px;
    color: var(--text-2);
    min-height: 44px;
    cursor: pointer;
  }

  .ing-own input {
    width: 18px;
    height: 18px;
    accent-color: var(--color-accent);
  }

  .ing-stepper {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 180px;
  }

  .stepper-btn {
    width: 44px;
    height: 44px;
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    color: var(--color-text);
    font-size: 18px;
    cursor: pointer;
  }

  .stepper-btn:hover { border-color: var(--color-accent); }

  .ing-unit {
    min-width: 70px;
    text-align: right;
    font-size: 12.5px;
    color: var(--text-2);
  }

  .formula-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: var(--space-4);
  }

  .total-line {
    font-size: 12.5px;
    color: var(--text-2);
  }

  .reset-link {
    border: none;
    background: none;
    color: var(--color-accent-2);
    font-size: 12px;
    cursor: pointer;
    min-height: 44px;
  }

  .reset-link:hover { text-decoration: underline; }

  .formula-actions {
    margin-top: var(--space-6);
    display: flex;
    gap: var(--space-3);
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
    padding: var(--space-2) var(--space-2);
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

  .compare-kind {
    font-size: 11px;
    color: var(--text-2);
  }

  .compare-delta {
    font-size: 16px;
    min-width: 44px;
    text-align: right;
  }

  .compare-empty {
    margin-top: var(--space-6);
    font-size: 12.5px;
    color: var(--text-2);
  }

  .t1-foot {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-6);
    background: var(--color-surface);
    border-top: 1px solid var(--color-divider);
    min-height: 60px;
  }

  .foot-label {
    flex-shrink: 0;
  }

  .foot-empty {
    font-size: 12.5px;
    color: var(--text-3);
  }

  .foot-scroll {
    display: flex;
    gap: var(--space-2);
    overflow-x: auto;
    flex: 1;
    min-width: 0;
  }

  .foot-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 44px;
    padding: 0 var(--space-3);
    border: 1px solid var(--color-divider);
    border-radius: var(--radius-pill);
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 12px;
    white-space: nowrap;
    cursor: pointer;
    flex-shrink: 0;
  }

  .foot-chip:hover {
    border-color: var(--color-accent);
  }

  .foot-swatch {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.2);
    flex-shrink: 0;
  }

  .foot-chip-label {
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
