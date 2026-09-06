<script lang="ts">
  // T1 — Pergunta e resposta (rf-04). Unifica o que antes eram CorView +
  // MesclarView num único fluxo "uma pergunta, uma resposta": campo de busca
  // único (nome/código/hex), sugestões por toque, grade de amostras no vazio,
  // resposta em 2 colunas, barra de filtros fixa, rodapé com histórico.
  //
  // D-001 (Epic D): reescrita de porte pra ficar fiel ao protótipo aprovado
  // (tests/fixtures/mockup/t1-pergunta.html). T1 tem cabeçalho PRÓPRIO — não usa
  // o <Header> compartilhado (esse é só das telas secundárias T2-T4; ver
  // comentário em components/Header.svelte) — porque o campo de busca, os dois
  // atalhos de câmera e a navegação inteira vivem nesta barra.
  //
  // Motor de cor: esta tela NÃO reimplementa nem ajusta RG-01/RG-15 (fora de
  // escopo do rf-04, "Não faz" — ver observação de construção). Ela consome
  // findSimilar/suggestEquivalentRecipe/suggestRecipeForColor/suggestFromStock/
  // bestBrandsFor como o backend devolve hoje. O stepper de gotas (RG-12/CA6)
  // ajusta a PROPORÇÃO local (gotas/percentuais e a prévia de cor, via mistura
  // em luz linear — RG-03) sem re-chamar o solver; o ΔE00 mostrado continua
  // sendo o último valor calculado pelo servidor para a fórmula base — recalcular
  // o ΔE00 exato ao vivo exigiria portar CIEDE2000 pro cliente, o que é ajuste
  // de motor (rf-05), fora desta spec.
  import PaintBottle from '../components/PaintBottle.svelte';
  import BrandMark from '../components/BrandMark.svelte';
  import LangSwitch from '../components/LangSwitch.svelte';
  import { allManufacturers, allPaints, paintById, searchPaints, hexOf, type Paint } from '../services/catalog';
  import {
    findSimilar, suggestEquivalentRecipe, suggestRecipeForColor, suggestFromStock, bestBrandsFor,
    type EquivalentRecipe, type BrandBest, type SearchResult,
  } from '../services/engine';
  import { shelf, inShelf, toggleShelf } from '../services/shelf.svelte';
  import { stock } from '../services/stock.svelte';
  import { saveRecipe } from '../services/recipes.svelte';
  import { switchTab } from '../nav.svelte';
  import { appState } from '../appState.svelte';
  import { recents, rememberPaint, rememberMescla } from '../recents.svelte';
  import { toast } from '../toast.svelte';
  import { hexToRgb } from '../color/theory';
  import { verdictKeys, deltaIsGood } from '../ui';
  import { t, decimal } from '../i18n.svelte';

  // ── Busca ──
  let query = $state('');
  let suggestOpen = $state(false);
  let inputEl: HTMLInputElement | undefined = $state();

  type Target = Paint | { r: number; g: number; b: number; hex: string };

  let sourcePaint: Paint | null = $state(null);
  let freeTarget: { r: number; g: number; b: number; hex: string } | null = $state(null);
  let searchedButNothing = $state(false);

  let target: Target | null = $derived(sourcePaint ?? freeTarget);

  // Indicador "teclado de código" (protótipo): a caixa que o pintor está
  // digitando parece um código de pote (dígito + letras/pontuação, sem
  // espaço), não um nome — 70.951, XF-2, C1.
  let codeMode = $derived.by(() => {
    const q = query.trim();
    if (!q) return false;
    return /^[a-zA-Z0-9.-]+$/.test(q) && /\d/.test(q);
  });

  // ── Filtros fixos (US-09, RG-07) ──
  let onlyHave = $state(false);
  let brandFilter: number | 'all' = $state('all');
  let unit: 'drops' | 'ml' = $state('drops');
  let forceMix = $state(false);

  let haveCount = $derived(
    stock.paints.length + allPaints().filter(p => shelf.manufacturerIds.includes(p.manufacturerId)).length
  );

  // ── Resultado ──
  let computing = $state(false);
  let readyPot: SearchResult | null = $state(null);
  let formula: EquivalentRecipe | null = $state(null);
  let otherBrands: BrandBest[] = $state([]);
  let usedManufacturerId: number | null = $state(null);
  let manualDrops: number[] | null = $state(null);
  let poolEmpty = $state(false);

  // Deep-link interno: T4 → "gerar fórmula equivalente" com esta tinta como alvo.
  $effect(() => {
    if (appState.pendingTargetPaint) {
      const p = appState.pendingTargetPaint;
      appState.pendingTargetPaint = null;
      pickPaint(p);
    }
  });

  let suggestions = $derived.by(() => {
    const q = query.trim();
    if (!q || sourcePaint || freeTarget) return [];
    return searchPaints(q, { limit: 8 });
  });

  // Amostras da grade vazia (US-03): até 24 tintas espalhadas do catálogo.
  let sampleGrid = $derived.by(() => {
    const all = allPaints();
    if (all.length <= 24) return all;
    const step = Math.max(1, Math.floor(all.length / 24));
    const out: Paint[] = [];
    for (let i = 0; i < all.length && out.length < 24; i += step) out.push(all[i]);
    return out;
  });

  function looksLikeHex(q: string): string | null {
    const m = /^#?([0-9a-fA-F]{6})$/.exec(q.trim());
    return m ? `#${m[1].toUpperCase()}` : null;
  }

  function onQueryInput() {
    suggestOpen = query.trim().length > 0;
  }

  function closeSuggest() {
    suggestOpen = false;
  }

  function submitQuery() {
    const q = query.trim();
    if (!q) return;
    const exact = searchPaints(q, { limit: 1 })[0];
    const hex = looksLikeHex(q);
    if (exact && (!hex || exact.code.toLowerCase() === q.toLowerCase())) {
      pickPaint(exact);
      return;
    }
    if (hex) {
      const rgb = hexToRgb(hex)!;
      pickHex(rgb.r, rgb.g, rgb.b, hex);
      return;
    }
    if (exact) {
      pickPaint(exact);
      return;
    }
    searchedButNothing = true;
    sourcePaint = null;
    freeTarget = null;
    readyPot = null;
    formula = null;
    otherBrands = [];
  }

  function pickPaint(p: Paint) {
    sourcePaint = p;
    freeTarget = null;
    searchedButNothing = false;
    suggestOpen = false;
    query = '';
    forceMix = false;
    manualDrops = null;
    rememberPaint(p.id);
    void compute();
  }

  function pickHex(r: number, g: number, b: number, hex: string) {
    sourcePaint = null;
    freeTarget = { r, g, b, hex };
    searchedButNothing = false;
    suggestOpen = false;
    query = '';
    forceMix = false;
    manualDrops = null;
    void compute();
  }

  function pickSample(p: Paint) {
    pickPaint(p);
  }

  // Atalhos de câmera do cabeçalho: "fotografar a peça" já tem casa (T2, Plano
  // da peça — RF de mapear a peça por foto). "Fotografar o pote" (ler nome,
  // código e cor de uma etiqueta por foto) não tem serviço de OCR/câmera no
  // código-base ainda (fora de escopo do rf-04, como o motor de cor no topo do
  // arquivo) — o botão existe pra fidelidade visual e leva o foco pro campo de
  // busca como retorno inofensivo até essa capability existir.
  function shootPiece() {
    switchTab('plano');
  }

  function shootPot() {
    inputEl?.focus();
  }

  // ── Cálculo (RG-07/RG-10, consumindo o backend como devolve hoje) ──
  async function bestOverManufacturers<T extends { deltaE: number }>(
    ids: number[],
    call: (mfrId: number) => Promise<T>,
  ): Promise<T | null> {
    const settled = await Promise.allSettled(ids.map(call));
    const ok: T[] = [];
    for (const s of settled) {
      if (s.status === 'fulfilled') ok.push(s.value);
    }
    if (ok.length === 0) return null;
    ok.sort((a, b) => a.deltaE - b.deltaE);
    return ok[0];
  }

  async function compute() {
    const tg = target;
    if (!tg) return;
    computing = true;
    readyPot = null;
    formula = null;
    otherBrands = [];
    poolEmpty = false;
    usedManufacturerId = null;
    manualDrops = null;
    try {
      const similar = await findSimilar(tg.r, tg.g, tg.b, 30, 1);
      const best = similar[0] ?? null;
      if (best && best.deltaE < 2.5 && !forceMix) {
        readyPot = best;
        computing = false;
        return;
      }

      if (allManufacturers().length === 0) {
        poolEmpty = true;
        computing = false;
        return;
      }

      let result: EquivalentRecipe | null = null;

      if (typeof brandFilter === 'number') {
        result = sourcePaint
          ? await suggestEquivalentRecipe(sourcePaint.id, brandFilter)
          : await suggestRecipeForColor(tg.r, tg.g, tg.b, brandFilter);
        usedManufacturerId = brandFilter;
      } else if (onlyHave) {
        if (shelf.manufacturerIds.length === 0 && stock.paints.length === 0) {
          poolEmpty = true;
          computing = false;
          return;
        }
        if (sourcePaint && stock.paints.length > 0) {
          try {
            result = await suggestFromStock(sourcePaint.id, stock.paints);
          } catch {
            /* estoque não monta receita — segue pras marcas da estante */
          }
        }
        if (!result && shelf.manufacturerIds.length > 0) {
          const found = await bestOverManufacturers(shelf.manufacturerIds, id =>
            sourcePaint
              ? suggestEquivalentRecipe(sourcePaint.id, id)
              : suggestRecipeForColor(tg.r, tg.g, tg.b, id),
          );
          result = found;
          if (found) usedManufacturerId = allManufacturers().find(m => m.name === found.targetManufacturer)?.id ?? null;
        }
      } else {
        // "Todas as marcas": aproxima o cross-brand (RG-13) escolhendo, entre
        // todos os fabricantes, o que dá o menor ΔE00 — o motor de mistura
        // real que combina potes de MARCAS diferentes numa única fórmula não
        // está exposto por nenhuma rota hoje (ver observação de construção).
        if (sourcePaint) {
          const brands = await bestBrandsFor(sourcePaint.id);
          otherBrands = [...brands].sort((a, b) => a.deltaE - b.deltaE);
          const top = otherBrands[0];
          if (top) {
            result = await suggestEquivalentRecipe(sourcePaint.id, top.manufacturerId);
            usedManufacturerId = top.manufacturerId;
          }
        } else {
          const ids = allManufacturers().map(m => m.id);
          const found = await bestOverManufacturers(ids, id => suggestRecipeForColor(tg.r, tg.g, tg.b, id));
          result = found;
          if (found) usedManufacturerId = allManufacturers().find(m => m.name === found.targetManufacturer)?.id ?? null;
        }
      }

      if (!result) {
        poolEmpty = true;
        computing = false;
        return;
      }
      formula = result;

      if (sourcePaint && otherBrands.length === 0 && !result.reproducible) {
        otherBrands = (await bestBrandsFor(sourcePaint.id)).sort((a, b) => a.deltaE - b.deltaE);
      }

      rememberMescla({
        sourceId: sourcePaint?.id ?? -1,
        brandIds: usedManufacturerId != null ? [usedManufacturerId] : [],
        sourceName: result.sourceName || freeTarget?.hex || '',
        targetManufacturer: result.targetManufacturer,
        deltaE: result.deltaE,
        sourceRGB: [result.sourceR, result.sourceG, result.sourceB],
        resultRGB: [result.resultR, result.resultG, result.resultB],
      });
    } catch (e) {
      console.error('Erro ao calcular:', e);
      toast(t('noneRegHere'), 'error');
    } finally {
      computing = false;
    }
  }

  function useMixAnyway() {
    forceMix = true;
    void compute();
  }

  function pickBrandPill(id: number | 'all') {
    brandFilter = id;
    if (target) void compute();
  }

  function toggleOnlyHave() {
    onlyHave = !onlyHave;
    if (target) void compute();
  }

  // ── Gotas: menor proporção inteira (portado do app anterior) ──
  function gcd(a: number, b: number): number {
    return b === 0 ? a : gcd(b, a % b);
  }
  function computeDrops(ingredients: { percentage: number }[]): number[] {
    const raw = ingredients.map(i => i.percentage);
    const ints = raw.map(Math.floor);
    let left = 100 - ints.reduce((a, b) => a + b, 0);
    const byFrac = raw.map((v, idx) => ({ idx, frac: v - Math.floor(v) })).sort((a, b) => b.frac - a.frac);
    for (let k = 0; left > 0 && byFrac.length; k++, left--) ints[byFrac[k % byFrac.length].idx]++;
    const g = ints.filter(v => v > 0).reduce((acc, v) => gcd(acc, v), 0) || 1;
    return ints.map(v => Math.max(1, Math.min(40, Math.round(v / g))));
  }

  let baseDrops = $derived.by(() => {
    const f = formula;
    return f ? computeDrops(f.ingredients) : [];
  });
  let drops = $derived(manualDrops ?? baseDrops);
  let totalDrops = $derived(drops.reduce((a, b) => a + b, 0));
  let totalMl = $derived(decimal(totalDrops * 0.05, 2));

  function adjustDrop(i: number, delta: number) {
    if (!formula) return;
    const cur = manualDrops ? [...manualDrops] : [...baseDrops];
    cur[i] = Math.max(1, Math.min(40, cur[i] + delta));
    manualDrops = cur;
  }

  function resetDrops() {
    manualDrops = null;
  }

  // Prévia de cor da mistura sob as gotas atuais — média em luz linear
  // (RG-03), só para a amostra "mistura" reagir ao stepper; o ΔE00 exibido
  // continua sendo o do servidor (ver nota no topo do arquivo).
  function srgbToLinear(v: number): number {
    const c = v / 255;
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  }
  function linearToSrgb(c: number): number {
    const v = c <= 0.0031308 ? c * 12.92 : 1.055 * c ** (1 / 2.4) - 0.055;
    return Math.max(0, Math.min(255, Math.round(v * 255)));
  }
  let mixPreview = $derived.by(() => {
    if (!formula) return null;
    const total = drops.reduce((a, b) => a + b, 0) || 1;
    let rl = 0, gl = 0, bl = 0;
    formula.ingredients.forEach((ing, i) => {
      const w = drops[i] / total;
      rl += srgbToLinear(ing.r) * w;
      gl += srgbToLinear(ing.g) * w;
      bl += srgbToLinear(ing.b) * w;
    });
    return { r: linearToSrgb(rl), g: linearToSrgb(gl), b: linearToSrgb(bl) };
  });

  // ── Rótulo/manchete (RG-05/RG-13) ──
  let manufacturerCount = $derived.by(() => {
    const f = formula;
    return f ? new Set(f.ingredients.map((i: { name: string }) => i.name)).size : 0;
  });
  let headline = $derived.by(() => {
    const f = formula;
    if (readyPot) return t('hPote');
    if (!f) return '';
    if (f.ingredients.length <= 1) return t('hPote');
    return t('hMix', { n: f.ingredients.length });
  });
  let vc = $derived.by(() => {
    const f = formula;
    const rp = readyPot;
    if (f) return verdictKeys(f.deltaE);
    if (rp) return verdictKeys(rp.deltaE);
    return null;
  });
  let deltaValue = $derived.by(() => {
    const rp = readyPot;
    const f = formula;
    return rp ? rp.deltaE : (f?.deltaE ?? 0);
  });
  let deltaColor = $derived(deltaIsGood(deltaValue) ? 'var(--color-accent-2)' : 'var(--color-text)');

  let mixSwatch = $derived.by(() => {
    const rp = readyPot;
    const f = formula;
    const mp = mixPreview;
    if (rp) return { r: rp.r, g: rp.g, b: rp.b };
    if (mp) return mp;
    if (f) return { r: f.resultR, g: f.resultG, b: f.resultB };
    return null;
  });

  let askedName = $derived.by(() => {
    const sp = sourcePaint;
    const ft = freeTarget;
    return sp ? sp.name : (ft?.hex ?? '');
  });
  let askedMeta = $derived.by(() => {
    const sp = sourcePaint;
    return sp ? `${sp.code} · ${sp.manufacturer}` : '';
  });

  // "Tenho na estante" por ingrediente (protótipo): o dado mais fino que o
  // app guarda é por FABRICANTE (services/shelf), não por tinta — então o
  // check reflete/alterna se o fabricante da fórmula está na estante, e vale
  // igual pra todos os ingredientes dela (são todos do mesmo fabricante-alvo).
  let formulaHave = $derived(usedManufacturerId != null && inShelf(usedManufacturerId));
  function toggleFormulaHave() {
    if (usedManufacturerId != null) toggleShelf(usedManufacturerId);
  }

  let computingBrandLabel = $derived(
    typeof brandFilter === 'number' ? (allManufacturers().find(m => m.id === brandFilter)?.name ?? '') : t('allBrands')
  );

  function rerun(m: (typeof recents.mesclas)[number]) {
    if (m.sourceId >= 0) {
      const p = paintById(m.sourceId);
      if (p) {
        pickPaint(p);
        return;
      }
    }
    pickHex(m.sourceRGB[0], m.sourceRGB[1], m.sourceRGB[2], hexOf({ r: m.sourceRGB[0], g: m.sourceRGB[1], b: m.sourceRGB[2] }));
  }

  function doSaveRecipe() {
    if (!target || (!formula && !readyPot)) return;
    const mfrId = usedManufacturerId ?? (readyPot ? paintById(readyPot.paintId)?.manufacturerId ?? null : null);
    if (mfrId == null) return;
    saveRecipe({
      name: sourcePaint?.name ?? freeTarget?.hex ?? '',
      targetPaintId: sourcePaint?.id ?? null,
      targetR: target.r,
      targetG: target.g,
      targetB: target.b,
      manufacturerId: mfrId,
    });
    toast(t('recipeSavedToast'));
    switchTab('receitas');
  }
</script>

<div class="t1">
  <!-- Cabeçalho: marca, busca (com atalhos de câmera) e navegação — métrica do
       protótipo (min-height 76px, busca 54px). -->
  <div
    style="display: flex; align-items: center; flex-wrap: wrap; row-gap: 10px; column-gap: 18px; min-height: 76px; flex-shrink: 0; padding: 12px 20px; border-bottom: 1px solid var(--color-line);"
  >
    <div style="display: flex; align-items: center; gap: 10px; flex-shrink: 0;">
      <BrandMark size={30} />
      <!-- "Mescla" é o nome do produto (como no BrandMark, aria-label fixo), não
           copy de interface — não passa por t(). -->
      <span style="font-size: clamp(16px, 1.7cqi, 19px); font-weight: 500; letter-spacing: -0.02em;">Mescla</span>
    </div>

    <div style="position: relative; flex: 1 1 420px; min-width: 0; order: 3;">
      <div
        style="display: flex; align-items: center; gap: 12px; height: 54px; padding: 0 14px 0 16px; border: 1px solid var(--color-field-border); border-radius: 14px; background: var(--color-field); overflow: hidden;"
      >
        <i class="ph ph-magnifying-glass" style="font-size: 22px; color: var(--color-neutral-500); flex-shrink: 0;"></i>
        <input
          bind:this={inputEl}
          bind:value={query}
          oninput={onQueryInput}
          onkeydown={e => e.key === 'Enter' && submitQuery()}
          onfocus={() => (suggestOpen = query.trim().length > 0)}
          type="text"
          placeholder={t('phSearch')}
          aria-label={t('phSearch')}
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          enterkeyhint="search"
          style="flex: 1 1 auto; min-width: 60px; height: 50px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: clamp(16px, 1.7cqi, 19px); font-weight: 500; letter-spacing: -0.01em;"
        />
        {#if codeMode}
          <span
            style="display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0; height: 32px; padding: 0 10px; border-radius: 8px; background: var(--color-accent-900); color: var(--color-accent-400); font-size: 12px; font-weight: 500; letter-spacing: 0.04em; text-transform: uppercase; white-space: nowrap;"
          >
            <i class="ph-bold ph-keyboard" style="font-size: 14px;"></i>{t('kbdCode')}
          </span>
        {/if}
        <button
          class="pressable t1h-acc"
          onclick={shootPiece}
          aria-label={t('shootPiece')}
          title={t('shootPiece')}
          style="display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; width: 46px; height: 46px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; cursor: pointer;"
        >
          <i class="ph ph-camera" style="font-size: 19px;"></i>
        </button>
        <button
          class="pressable t1h-ghost"
          onclick={shootPot}
          aria-label={t('shootPot')}
          title={t('shootPot')}
          style="display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; width: 46px; height: 46px; border: 1px solid var(--color-field-border); border-radius: 8px; background: transparent; color: var(--color-neutral-300); font-family: inherit; cursor: pointer;"
        >
          <i class="ph ph-drop-half" style="font-size: 19px;"></i>
        </button>
      </div>

      {#if suggestOpen && suggestions.length > 0}
        <div
          role="listbox"
          aria-label={t('suggestTitle')}
          style="position: absolute; top: 62px; left: 0; right: 0; z-index: 40; padding: 16px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-raised); box-shadow: 0 18px 48px rgba(0, 0, 0, 0.55);"
        >
          <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
            <span class="section-label">{t('suggestTitle')}</span>
            <button
              class="pressable"
              onclick={closeSuggest}
              style="height: 44px; padding: 0 12px; border: none; background: transparent; color: var(--color-neutral-500); font-family: inherit; font-size: 14px; cursor: pointer;"
              >{t('close')}</button
            >
          </div>
          <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 12px;">
            {#each suggestions as p (p.id)}
              <button
                class="pressable t1h-border"
                onclick={() => pickPaint(p)}
                role="option"
                aria-selected="false"
                style="display: flex; align-items: center; gap: 14px; padding: 14px; min-height: 96px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-surface); text-align: left; cursor: pointer; font-family: inherit;"
              >
                <PaintBottle r={p.r} g={p.g} b={p.b} width={34} height={57} label={p.name} />
                <span style="display: flex; flex-direction: column; gap: 3px; min-width: 0;">
                  <span
                    style="font-size: 15px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
                    >{p.name}</span
                  >
                  <span class="font-mono" style="font-size: 12px; color: var(--color-neutral-500);"
                    >{p.code} · {p.manufacturer}</span
                  >
                </span>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <div style="display: flex; align-items: center; gap: 4px; flex-shrink: 0; white-space: nowrap;">
      <button
        class="pressable t1h-acc"
        onclick={() => switchTab('plano')}
        style="display: inline-flex; align-items: center; gap: 8px; height: 48px; padding: 0 16px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
      >
        <i class="ph ph-crosshair" style="font-size: 18px;"></i>{t('navPlano')}
      </button>
      <span style="width: 1px; height: 28px; margin: 0 12px; background: var(--color-neutral-800);"></span>
      <button
        class="pressable t1h-nav"
        onclick={() => switchTab('estante')}
        style="height: 48px; padding: 0 14px; border: none; border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('navCatalogo')}</button
      >
      <button
        class="pressable t1h-nav"
        onclick={() => switchTab('receitas')}
        style="height: 48px; padding: 0 14px; border: none; border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('navReceitas')}</button
      >
      <button
        class="pressable t1h-nav"
        onclick={() => switchTab('estante')}
        style="height: 48px; padding: 0 14px; border: none; border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('navTintas')}</button
      >
      <LangSwitch />
    </div>
  </div>

  <!-- Barra de filtros fixa (US-09/RG-07) — métrica do protótipo (60px). -->
  <div
    style="display: flex; align-items: center; gap: 12px; height: 60px; min-height: 60px; flex-shrink: 0; padding: 0 20px; overflow-x: auto; border-bottom: 1px solid var(--color-line); background: var(--color-bar);"
  >
    <button
      class="pressable t1h-border"
      onclick={toggleOnlyHave}
      aria-pressed={onlyHave}
      style="display: inline-flex; align-items: center; gap: 12px; height: 48px; padding: 0 16px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
    >
      <span
        style="position: relative; width: 46px; height: 26px; border-radius: 999px; background: {onlyHave
          ? 'var(--color-accent)'
          : 'var(--color-neutral-700)'};"
      >
        <span
          style="position: absolute; top: 3px; left: 3px; width: 20px; height: 20px; border-radius: 999px; background: var(--color-neutral-100); transform: translateX({onlyHave
            ? 20
            : 0}px); transition: transform .18s ease;"
        ></span>
      </span>
      {t('onlyStock')}
      <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">{haveCount}</span>
    </button>

    <span style="width: 1px; height: 26px; background: var(--color-rule);"></span>

    <div style="display: flex; gap: 6px; flex-shrink: 0;">
      <button
        class="pressable t1h-border"
        onclick={() => pickBrandPill('all')}
        style="height: 48px; padding: 0 12px; border: 1px solid {brandFilter === 'all'
          ? 'var(--color-accent)'
          : 'var(--color-neutral-800)'}; border-radius: 8px; background: {brandFilter === 'all'
          ? 'var(--color-accent)'
          : 'transparent'}; color: {brandFilter === 'all'
          ? 'var(--color-accent-100)'
          : 'var(--color-text)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
        >{t('allBrands')}</button
      >
      {#each allManufacturers() as m (m.id)}
        <button
          class="pressable t1h-border"
          onclick={() => pickBrandPill(m.id)}
          style="height: 48px; padding: 0 12px; border: 1px solid {brandFilter === m.id
            ? 'var(--color-accent)'
            : 'var(--color-neutral-800)'}; border-radius: 8px; background: {brandFilter === m.id
            ? 'var(--color-accent)'
            : 'transparent'}; color: {brandFilter === m.id
            ? 'var(--color-accent-100)'
            : 'var(--color-text)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
          >{m.name}</button
        >
      {/each}
    </div>

    <span style="width: 1px; height: 26px; background: var(--color-rule);"></span>

    <div style="display: flex; border: 1px solid var(--color-neutral-800); border-radius: 8px; overflow: hidden; flex-shrink: 0; white-space: nowrap;">
      <button
        class="pressable"
        onclick={() => (unit = 'drops')}
        style="height: 46px; padding: 0 14px; border: none; background: {unit === 'drops'
          ? 'var(--color-accent)'
          : 'transparent'}; color: {unit === 'drops' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
        >{t('unitDrops')}</button
      >
      <button
        class="pressable"
        onclick={() => (unit = 'ml')}
        style="height: 46px; padding: 0 14px; border: none; border-left: 1px solid var(--color-neutral-800); background: {unit ===
        'ml'
          ? 'var(--color-accent)'
          : 'transparent'}; color: {unit === 'ml' ? 'var(--color-accent-100)' : 'var(--color-text)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer;"
        >{t('unitMl')}</button
      >
    </div>

    <span style="flex: 1;"></span>
  </div>

  <!-- Miolo em 2 colunas (métrica do protótipo: 38% / 1fr) — grade sempre
       presente; cada coluna resolve seu próprio estado. -->
  <div style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 38%) minmax(0, 1fr);">
    <div
      style="padding: 22px 22px 20px 20px; border-right: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 20px; min-height: 0; overflow-y: auto;"
    >
      {#if computing}
        <div style="display: flex; flex-direction: column; gap: 16px;">
          <div style="height: 44px; width: 88%; border-radius: 8px; background: var(--color-surface);"></div>
          <div style="height: 44px; width: 62%; border-radius: 8px; background: var(--color-surface);"></div>
          <div style="height: 132px; width: 100%; border-radius: 14px; background: var(--color-raised); margin-top: 10px;"></div>
          <p style="margin: 0; font-size: 14px; color: var(--color-neutral-500);">
            {t('loadingNote', { brand: computingBrandLabel })}
          </p>
        </div>
      {:else if (readyPot || formula) && vc}
        <div style="display: flex; flex-direction: column; gap: 18px; min-height: 0;">
          <div>
            <p class="section-label" style="margin: 0 0 10px;">{t('answerKicker')}</p>
            <h1
              class="font-display"
              style="margin: 0; font-size: clamp(23px, 3.1cqi, 40px); font-weight: 500; letter-spacing: -0.025em; line-height: 1.12; color: var(--color-text); text-wrap: pretty;"
            >
              {headline}
            </h1>
            <p style="margin: 12px 0 0; font-size: clamp(14px, 1.5cqi, 18px); line-height: 1.5; color: var(--color-neutral-400); text-wrap: pretty;">
              {t(vc.c)}
            </p>
          </div>

          <!-- No protótipo este painel é estático: a "consequência" em palavras
               já explica o ΔE, então a folha de escala do tema anterior saiu. -->
          <div
            style="display: flex; flex-direction: column; gap: 12px; padding: 18px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);"
          >
            <span class="section-label">{t('howClose')}</span>
            <div style="display: flex; align-items: stretch; flex-wrap: wrap; gap: 14px;">
              <div style="display: flex; flex-direction: column; gap: 6px;">
                <div style="display: flex; border-radius: 8px; overflow: hidden; border: 1px solid var(--color-neutral-800);">
                  <span style="width: 74px; height: 86px; background: rgb({target?.r}, {target?.g}, {target?.b});"></span>
                  <span
                    style="width: 74px; height: 86px; background: {mixSwatch
                      ? `rgb(${mixSwatch.r}, ${mixSwatch.g}, ${mixSwatch.b})`
                      : 'transparent'};"
                  ></span>
                </div>
                <div style="display: flex; gap: 14px; font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">
                  <span style="width: 74px;">{t('target')}</span>
                  <span style="width: 74px;">{t('mixture')}</span>
                </div>
              </div>
              <div style="display: flex; flex-direction: column; justify-content: center; gap: 2px; min-width: 0;">
                <span style="display: flex; align-items: baseline; gap: 8px;">
                  <span class="font-mono" style="font-size: clamp(32px, 4.4cqi, 54px); font-weight: 500; line-height: 1; letter-spacing: -0.03em; color: {deltaColor};"
                    >{decimal(deltaValue, 1)}</span
                  >
                  <span style="font-size: 14px; color: var(--color-neutral-500);">ΔE00</span>
                </span>
                <span style="font-size: clamp(14px, 1.45cqi, 17px); font-weight: 500; color: var(--color-text); text-wrap: pretty;">{t(vc.v)}</span>
              </div>
            </div>
          </div>

          <div style="display: flex; align-items: center; gap: 14px; padding-top: 16px; border-top: 1px solid var(--color-line);">
            <PaintBottle r={target?.r ?? 0} g={target?.g ?? 0} b={target?.b ?? 0} width={30} height={50} label={askedName} />
            <span style="display: flex; flex-direction: column; min-width: 0;">
              <span style="font-size: 12px; letter-spacing: 0.1em; text-transform: uppercase; color: var(--color-neutral-500);">{t('youAsked')}</span>
              <span style="font-size: clamp(15px, 1.6cqi, 19px); font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{askedName}</span>
              {#if askedMeta}
                <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">{askedMeta}</span>
              {/if}
            </span>
          </div>
        </div>
      {:else}
        <div style="display: flex; flex-direction: column; gap: 14px;">
          <h1 class="font-display" style="margin: 0; font-size: clamp(22px, 2.9cqi, 36px); font-weight: 500; letter-spacing: -0.02em; line-height: 1.14; color: var(--color-text);">
            {t('emptyH')}
          </h1>
          <p style="margin: 0; font-size: 18px; line-height: 1.5; color: var(--color-neutral-400);">{t('emptyP')}</p>
          <button
            class="pressable t1h-acc"
            onclick={shootPiece}
            style="display: inline-flex; align-items: center; justify-content: center; gap: 10px; height: 72px; margin-top: 6px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 18px; font-weight: 500; cursor: pointer;"
          >
            <i class="ph ph-camera" style="font-size: 26px;"></i>{t('shootPiece')}
          </button>
        </div>
      {/if}
    </div>

    <div style="min-height: 0; overflow-y: auto; padding: 22px 20px 24px 22px;">
      {#if computing}
        <div style="display: flex; flex-direction: column; gap: 12px;">
          {#each Array(3) as _}
            <div style="display: flex; align-items: center; gap: 18px; height: 108px; padding: 0 18px; border: 1px solid var(--color-line); border-radius: 14px; background: var(--color-panel);">
              <div style="width: 44px; height: 74px; border-radius: 8px; background: var(--color-surface);"></div>
              <div style="flex: 1; display: flex; flex-direction: column; gap: 10px;">
                <div style="height: 16px; width: 44%; border-radius: 4px; background: var(--color-surface);"></div>
                <div style="height: 14px; width: 26%; border-radius: 4px; background: var(--color-surface);"></div>
              </div>
              <div style="width: 180px; height: 56px; border-radius: 8px; background: var(--color-surface);"></div>
            </div>
          {/each}
        </div>
      {:else if readyPot}
        <div>
          <div
            style="display: flex; align-items: center; gap: 22px; padding: 22px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel);"
          >
            <PaintBottle r={readyPot.r} g={readyPot.g} b={readyPot.b} width={72} height={120} label={readyPot.name} />
            <div style="display: flex; flex-direction: column; gap: 6px; min-width: 0;">
              <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-accent-400);">{t('useDirect')}</span>
              <span style="font-size: clamp(20px, 2.5cqi, 30px); font-weight: 500; letter-spacing: -0.02em; line-height: 1.12; color: var(--color-text);">{readyPot.name}</span>
              <span class="font-mono" style="font-size: 16px; color: var(--color-neutral-400);">{readyPot.code} · {readyPot.manufacturer}</span>
            </div>
          </div>
          <p style="margin: 18px 0 0; font-size: 16px; color: var(--color-neutral-400);">{t('mixWell')}</p>
          <button
            class="pressable t1h-ghost"
            onclick={useMixAnyway}
            style="margin-top: 20px; height: 52px; padding: 0 20px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
            >{t('wantMix')}</button
          >
        </div>
      {:else if formula}
        <div>
          {#if !formula.reproducible}
            <div style="margin-bottom: 20px; padding: 16px 18px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel);">
              <p style="margin: 0; font-size: 17px; font-weight: 500; color: var(--color-text);">{t('unreachT')}</p>
              <p style="margin: 6px 0 0; font-size: 15px; color: var(--color-neutral-400);">
                {t('unreachB', { pool: t('poolBrand', { brand: formula.targetManufacturer }), d: decimal(formula.deltaE, 1) })}
              </p>
            </div>
          {/if}

          <div style="display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; row-gap: 12px; gap: 16px; margin-bottom: 8px;">
            <div>
              <p class="section-label" style="margin: 0;">{t('formula')}</p>
              <h2 class="font-display" style="margin: 4px 0 0; font-size: clamp(17px, 2.1cqi, 26px); font-weight: 500; letter-spacing: -0.015em; color: var(--color-text);">
                {formula.targetManufacturer}
              </h2>
            </div>
            <button
              class="pressable t1h-acc"
              onclick={doSaveRecipe}
              style="display: inline-flex; align-items: center; gap: 10px; height: 56px; padding: 0 22px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 17px; font-weight: 500; cursor: pointer;"
            >
              <i class="ph ph-bookmark-simple" style="font-size: 21px;"></i>{t('saveRecipeBtn')}
            </button>
          </div>

          <div style="display: flex; height: 12px; margin: 16px 0 20px; border-radius: 4px; overflow: hidden; border: 1px solid var(--color-rule);">
            {#each formula.ingredients as ing, i (ing.paintId)}
              <span
                style="height: 12px; width: {(drops[i] / (totalDrops || 1)) * 100}%; background: rgb({ing.r}, {ing.g}, {ing.b});"
              ></span>
            {/each}
          </div>

          <div style="display: flex; flex-direction: column; gap: 12px;">
            {#each formula.ingredients as ing, i (ing.paintId)}
              <div
                style="display: flex; align-items: center; flex-wrap: wrap; row-gap: 12px; gap: 16px; min-height: 100px; padding: 12px 16px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);"
              >
                <PaintBottle r={ing.r} g={ing.g} b={ing.b} width={44} height={74} label={ing.name} />

                <div style="display: flex; flex-direction: column; gap: 6px; flex: 1 1 150px; min-width: 0;">
                  <span style="font-size: clamp(15px, 1.7cqi, 20px); font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.name}</span>
                  {#if ing.code}
                    <span class="font-mono" style="font-size: 14px; color: var(--color-neutral-500);">{ing.code}</span>
                  {/if}
                  <button
                    class="pressable"
                    onclick={toggleFormulaHave}
                    aria-label={t('ariaHave')}
                    style="display: inline-flex; align-items: center; gap: 8px; align-self: flex-start; height: 44px; padding: 0 12px 0 4px; border: none; background: transparent; color: {formulaHave
                      ? 'var(--color-accent-400)'
                      : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; white-space: nowrap;"
                  >
                    <span
                      style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {formulaHave
                        ? 'var(--color-accent)'
                        : 'var(--color-neutral-700)'}; border-radius: 4px; background: {formulaHave
                        ? 'var(--color-accent)'
                        : 'transparent'};"
                    >
                      <i class="ph-bold ph-check" style="font-size: 15px; color: {formulaHave ? 'var(--color-accent-100)' : 'transparent'};"></i>
                    </span>
                    {t('haveOnShelf')}
                  </button>
                </div>

                <div style="display: flex; align-items: center; gap: 10px; flex-shrink: 0;">
                  <button
                    class="pressable t1h-step"
                    onclick={() => adjustDrop(i, -1)}
                    aria-label={t('ariaMinus')}
                    style="width: 56px; height: 56px; border: 1px solid var(--color-field-border); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 26px; line-height: 1; cursor: pointer;"
                    >−</button
                  >
                  <div
                    style="display: flex; flex-direction: column; align-items: center; justify-content: center; width: 96px; height: 56px; border: 1px solid var(--color-rule); border-radius: 8px; background: var(--color-surface); cursor: ew-resize;"
                  >
                    {#if unit === 'drops'}
                      <span class="font-mono" style="font-size: clamp(19px, 2cqi, 25px); font-weight: 500; line-height: 1; color: var(--color-text);">{drops[i]}</span>
                      <span style="font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">{drops[i] === 1 ? t('drop') : t('drops')}</span>
                    {:else}
                      <span class="font-mono" style="font-size: clamp(19px, 2cqi, 25px); font-weight: 500; line-height: 1; color: var(--color-text);">{decimal(drops[i] * 0.05, 2)}</span>
                      <span style="font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500);">{t('unitMl')}</span>
                    {/if}
                  </div>
                  <button
                    class="pressable t1h-step"
                    onclick={() => adjustDrop(i, 1)}
                    aria-label={t('ariaPlus')}
                    style="width: 56px; height: 56px; border: 1px solid var(--color-field-border); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 26px; line-height: 1; cursor: pointer;"
                    >+</button
                  >
                </div>

                <span class="font-mono" style="min-width: 54px; text-align: right; flex-shrink: 0; font-size: clamp(17px, 1.9cqi, 24px); font-weight: 500; color: var(--color-neutral-400);"
                  >{Math.round((drops[i] / (totalDrops || 1)) * 100)}%</span
                >
              </div>
            {/each}
          </div>

          <div style="display: flex; align-items: center; gap: 18px; margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--color-line);">
            <span class="font-mono" style="font-size: 15px; color: var(--color-neutral-500);">
              {t('totalLabel', { v: unit === 'drops' ? `${totalDrops} ${t('drops')}` : `${totalMl} ml` })}
            </span>
            <span style="flex: 1;"></span>
            {#if manualDrops}
              <button
                class="pressable t1h-link"
                onclick={resetDrops}
                style="height: 44px; padding: 0 14px; border: none; background: transparent; color: var(--color-neutral-500); font-family: inherit; font-size: 14px; cursor: pointer;"
                >{t('resetProp')}</button
              >
            {/if}
          </div>

          <p style="margin: 18px 0 0; font-size: 16px; color: var(--color-neutral-400);">{formula.tips[0] || t('startBiggest')}</p>

          {#if otherBrands.length > 0}
            <p class="section-label" style="margin: 30px 0 12px;">{t('sameOtherBrand')}</p>
            <div style="display: flex; flex-direction: column;">
              {#each otherBrands as b (b.manufacturerId)}
                <button
                  class="pressable t1h-row"
                  onclick={() => pickBrandPill(b.manufacturerId)}
                  style="display: flex; align-items: center; gap: 16px; min-height: 64px; padding: 10px 4px; border: none; border-bottom: 1px solid var(--color-line); background: transparent; text-align: left; cursor: pointer; font-family: inherit;"
                >
                  <span style="width: 44px; height: 36px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({b.r}, {b.g}, {b.b}); flex-shrink: 0;"></span>
                  <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                    <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{b.manufacturer}</span>
                    <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
                      >{b.name}{b.code ? ` · ${b.code}` : ''}</span
                    >
                  </span>
                  <span class="font-mono" style="font-size: 14px; color: {deltaIsGood(b.deltaE) ? 'var(--color-accent-2)' : 'var(--color-neutral-400)'}; flex-shrink: 0;"
                    >ΔE {decimal(b.deltaE, 1)}</span
                  >
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if poolEmpty}
        <div style="display: flex; flex-direction: column; align-items: flex-start; gap: 14px; padding: 40px 0;">
          <i class="ph ph-drop-half" style="font-size: 40px; color: var(--color-neutral-700);"></i>
          <p style="margin: 0; font-size: 24px; font-weight: 500; color: var(--color-text);">{onlyHave ? t('noneTitleStock') : t('hNoneCross')}</p>
          <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400); max-width: 420px; text-wrap: pretty;">{onlyHave ? t('noneBodyStock') : t('noneRegHere')}</p>
          <div style="display: flex; gap: 10px; margin-top: 8px;">
            <button
              class="pressable t1h-acc"
              onclick={() => switchTab('estante')}
              style="display: inline-flex; align-items: center; gap: 10px; height: 56px; padding: 0 20px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
            >
              <i class="ph ph-plus" style="font-size: 18px;"></i>{t('addPaint')}
            </button>
            {#if onlyHave}
              <button
                class="pressable t1h-ghost"
                onclick={() => { onlyHave = false; void compute(); }}
                style="height: 56px; padding: 0 20px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
                >{t('useAllBrands')}</button
              >
            {/if}
          </div>
        </div>
      {:else if searchedButNothing}
        <div style="display: flex; flex-direction: column; align-items: flex-start; gap: 14px; padding: 40px 0;">
          <i class="ph ph-magnifying-glass" style="font-size: 40px; color: var(--color-neutral-700);"></i>
          <p style="margin: 0; font-size: 24px; font-weight: 500; color: var(--color-text);">{t('noResultH')}</p>
          <p style="margin: 0; font-size: 16px; color: var(--color-neutral-400); max-width: 420px;">{t('noResultBody')}</p>
          <button
            class="pressable t1h-acc"
            onclick={shootPot}
            style="display: inline-flex; align-items: center; gap: 10px; height: 56px; margin-top: 8px; padding: 0 20px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
          >
            <i class="ph ph-camera" style="font-size: 20px;"></i>{t('shootPot')}
          </button>
        </div>
      {:else}
        <div>
          <p class="section-label" style="margin: 0 0 14px;">{t('pickFinger')}</p>
          <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(64px, 1fr)); gap: 8px;">
            {#each sampleGrid as p (p.id)}
              <button
                class="pressable t1h-swatch"
                onclick={() => pickSample(p)}
                title="{p.name} · {p.manufacturer}"
                style="height: 60px; border: 1px solid var(--color-rule); border-radius: 8px; background: rgb({p.r}, {p.g}, {p.b}); cursor: pointer;"
              ></button>
            {/each}
          </div>
          <p style="margin: 24px 0 0; font-size: 15px; color: var(--color-neutral-500);">{t('pickFingerNote')}</p>
        </div>
      {/if}
    </div>
  </div>

  <!-- Rodapé "na mesa hoje" — métrica do protótipo (108px). -->
  <div
    style="height: 108px; min-height: 108px; flex-shrink: 0; padding: 0 20px; border-top: 1px solid var(--color-line); background: var(--color-bar); display: flex; align-items: center; gap: 18px;"
  >
    <span style="flex-shrink: 0; width: 84px; font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500); line-height: 1.4;">{t('onTable')}</span>
    <div style="flex: 1; min-width: 0; display: flex; gap: 10px; overflow-x: auto; padding: 6px 0;">
      {#each recents.mesclas as m (m.sourceId + '|' + m.targetManufacturer)}
        <button
          class="pressable t1h-hist"
          onclick={() => rerun(m)}
          style="display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; width: 80px; min-height: 86px; padding: 6px 4px; border: 1px solid transparent; border-radius: 8px; background: transparent; cursor: pointer; font-family: inherit;"
        >
          <PaintBottle r={m.sourceRGB[0]} g={m.sourceRGB[1]} b={m.sourceRGB[2]} width={28} height={47} label={m.sourceName} />
          <span style="font-size: 11.5px; color: var(--color-neutral-400); text-align: center; line-height: 1.25; max-width: 84px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"
            >{m.sourceName || t('youAsked')}</span
          >
        </button>
      {/each}
    </div>
  </div>
</div>


<style>
  .t1 {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  /* Hover — protótipo usa `style-hover`; reproduzido aqui por classe (regra 6). */
  .t1h-acc:hover {
    background: var(--color-accent-hover);
  }

  .t1h-ghost:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t1h-nav:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }

  .t1h-border:hover {
    border-color: var(--color-accent-700);
  }

  .t1h-swatch:hover {
    border-color: var(--color-accent-400);
  }

  .t1h-step:hover {
    border-color: var(--color-accent);
    color: var(--color-accent-400);
  }

  .t1h-link:hover {
    color: var(--color-accent-400);
  }

  .t1h-row:hover {
    background: var(--color-panel);
  }

  .t1h-hist:hover {
    border-color: var(--color-neutral-800);
    background: var(--color-panel);
  }
</style>
