<script lang="ts">
  // Mesclar — a jornada nº 1: "como reproduzo esta cor com o que tenho?"
  // Form de 2 campos + CTA; resultado renderiza NA MESMA tela (o back do
  // Android re-expande o form via pilha de histórico). A receita é calculada
  // por marca da estante — a melhor aparece primeiro, as outras viram pills.
  import Icon from '../components/Icon.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import BrandMark from '../components/BrandMark.svelte';
  import FormulaRibbon from '../components/FormulaRibbon.svelte';
  import FullScreenSearch from '../components/FullScreenSearch.svelte';
  import BrandSheet from '../components/BrandSheet.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import BenchMode from '../components/BenchMode.svelte';
  import { pushLayer, switchTab } from '../nav.svelte';
  import { allManufacturers, allPaints, paintById, hexOf, type Paint } from '../services/catalog';
  import { suggestEquivalentRecipe, suggestFromStock, bestBrandsFor, type EquivalentRecipe, type BrandBest } from '../services/engine';
  import { shelf, toggleShelf } from '../services/shelf.svelte';
  import { stock } from '../services/stock.svelte';
  import { appState } from '../appState.svelte';
  import { recents, rememberMescla, rememberPaint } from '../recents.svelte';
  import { toast } from '../toast.svelte';
  import { contrastOn, deltaVerdict, deltaIsGood } from '../ui';

  let sourcePaint: Paint | null = $state(null);
  let searchOpen = $state(false);
  let brandSheetOpen = $state(false);
  let deltaSheetOpen = $state(false);
  let benchOpen = $state(false);
  let computing = $state(false);
  let recipes: EquivalentRecipe[] = $state([]);
  let activeIdx = $state(0);
  let showResult = $state(false);
  let bestBrands: BrandBest[] = $state([]);
  // Priorizar o estoque próprio: quando ligado, a receita feita só com as tintas
  // que o pintor tem entra na disputa (e, se alcança a cor, aparece primeiro).
  let stockPriority = $state(stock.paints.length > 0);

  let closeResultLayer: (() => void) | null = null;

  // Deep-link interno: Catálogo → "Gerar fórmula equivalente".
  $effect(() => {
    if (appState.pendingMesclarPaint) {
      sourcePaint = appState.pendingMesclarPaint;
      appState.pendingMesclarPaint = null;
      collapseResult();
    }
  });

  let active = $derived(recipes[activeIdx] ?? null);
  let shelfNames = $derived(
    allManufacturers().filter(m => shelf.manufacturerIds.includes(m.id)).map(m => m.name)
  );
  let usingStock = $derived(stockPriority && stock.paints.length > 0);
  let canMesclar = $derived(sourcePaint !== null && (shelf.manufacturerIds.length > 0 || usingStock));
  let sameBrand = $derived(active !== null && active.targetManufacturer === active.sourceManufacturer);
  let brandLabel = $derived(
    shelfNames.length === 0
      ? 'Marcas'
      : shelfNames.length === 1
        ? shelfNames[0]
        : `${shelfNames[0]} +${shelfNames.length - 1}`
  );

  function openResult() {
    if (showResult) return;
    showResult = true;
    closeResultLayer = pushLayer(() => {
      closeResultLayer = null;
      showResult = false;
    });
  }

  function collapseResult() {
    closeResultLayer?.();
  }

  async function mesclar() {
    if (!sourcePaint || !canMesclar || computing) return;
    computing = true;
    bestBrands = [];
    try {
      const found: EquivalentRecipe[] = [];
      // O "pulo do gato": a receita feita só com o estoque entra primeiro na
      // disputa. Se alcança a cor (menor ΔE), o sort a deixa no topo.
      if (usingStock) {
        try {
          found.push(await suggestFromStock(sourcePaint.id, stock.paints));
        } catch {
          /* estoque não montou receita — segue com as marcas da estante */
        }
      }
      for (const brandId of shelf.manufacturerIds) {
        try {
          found.push(await suggestEquivalentRecipe(sourcePaint.id, brandId));
        } catch {
          /* marca sem candidatos (ex.: só a própria tinta) — pula */
        }
      }
      if (found.length === 0) {
        toast('Nada pra mesclar: escolha marcas ou cadastre tintas no estoque.', 'error');
        return;
      }
      found.sort((a, b) => a.deltaE - b.deltaE);
      recipes = found;
      activeIdx = 0;
      rememberPaint(sourcePaint.id);
      rememberMescla({
        sourceId: sourcePaint.id,
        brandIds: [...shelf.manufacturerIds],
        sourceName: found[0].sourceName,
        targetManufacturer: found[0].targetManufacturer,
        deltaE: found[0].deltaE,
        sourceRGB: [found[0].sourceR, found[0].sourceG, found[0].sourceB],
        resultRGB: [found[0].resultR, found[0].resultG, found[0].resultB],
      });
      openResult();
      if (navigator.vibrate) navigator.vibrate(10);
      if (!found[0].reproducible) {
        bestBrands = (await bestBrandsFor(sourcePaint.id))
          .filter(b => !shelfNames.includes(b.manufacturer))
          .slice(0, 3);
      }
    } catch (e) {
      console.error('Erro na mescla:', e);
      toast('O cálculo falhou. Tente de novo.', 'error');
    } finally {
      computing = false;
    }
  }

  // ── Gotas: menor proporção inteira (gcd + maior resto) — portado do desktop ──
  function gcd(a: number, b: number): number {
    return b === 0 ? a : gcd(b, a % b);
  }

  function computeDrops(ingredients: { percentage: number }[]): number[] {
    const raw = ingredients.map(i => i.percentage);
    const ints = raw.map(Math.floor);
    let left = 100 - ints.reduce((a, b) => a + b, 0);
    const byFrac = raw
      .map((v, idx) => ({ idx, frac: v - Math.floor(v) }))
      .sort((a, b) => b.frac - a.frac);
    for (let k = 0; left > 0 && byFrac.length; k++, left--) {
      ints[byFrac[k % byFrac.length].idx]++;
    }
    const g = ints.filter(v => v > 0).reduce((acc, v) => gcd(acc, v), 0) || 1;
    return ints.map(v => Math.round(v / g));
  }

  let drops = $derived(active ? computeDrops(active.ingredients) : []);
  let totalDrops = $derived(drops.reduce((a, b) => a + b, 0));

  function recipeText(): string {
    if (!active) return '';
    return [
      `${active.sourceName} (${active.sourceManufacturer}) → ${active.targetManufacturer}`,
      `Mistura de ${totalDrops} gotas`,
      ...active.ingredients.map(
        (ing, i) =>
          `${drops[i]} ${drops[i] === 1 ? 'gota' : 'gotas'} (${ing.percentage.toFixed(1)}%)  ${ing.name}${ing.code ? ` (${ing.code})` : ''}`
      ),
      `ΔE2000 ${active.deltaE.toFixed(2)}`,
    ].join('\n');
  }

  function copyRecipe() {
    navigator.clipboard.writeText(recipeText());
    toast('Receita copiada');
  }

  async function shareRecipe() {
    const text = recipeText();
    if (navigator.share) {
      try {
        await navigator.share({ title: 'Receita Mescla', text });
      } catch {
        /* usuário cancelou */
      }
    } else {
      navigator.clipboard.writeText(text);
      toast('Receita copiada');
    }
  }

  // Exemplo executável de 1 tap — isso É o onboarding.
  async function runExample() {
    const paints = allPaints();
    const example =
      paints.find(p => p.name.toLowerCase() === 'mephiston red') ??
      paints.find(p => p.name.toLowerCase().includes('red')) ??
      paints[0];
    if (!example) return;
    sourcePaint = example;
    if (shelf.manufacturerIds.length === 0) {
      const vallejo = allManufacturers().find(m => m.name === 'Vallejo') ?? allManufacturers()[0];
      if (vallejo) toggleShelf(vallejo.id);
    }
    await mesclar();
  }

  async function rerun(m: (typeof recents.mesclas)[number]) {
    const p = paintById(m.sourceId);
    if (!p) return;
    sourcePaint = p;
    await mesclar();
  }

  function pickBrandSuggestion(b: BrandBest) {
    toggleShelf(b.manufacturerId);
    void mesclar();
  }

  function seeReadyPaints() {
    if (!active) return;
    appState.corPreset = { r: active.sourceR, g: active.sourceG, b: active.sourceB };
    switchTab('cor');
  }

  function pickSource(p: Paint) {
    sourcePaint = p;
    rememberPaint(p.id);
    // Com resultado aberto, trocar a tinta re-roda a fórmula na hora.
    if (showResult) void mesclar();
  }
</script>

<div class="mesclar">
  <!-- Header papel: marca + seletor de marcas da estante -->
  <header class="head">
    <span class="head-brand">
      <BrandMark size={26} />
      <span class="head-name font-display">Mescla</span>
    </span>
    <button class="head-shelf pressable" onclick={() => (brandSheetOpen = true)}>
      {brandLabel}
      <Icon name="chevron-down" size={14} />
    </button>
  </header>

  {#if showResult && active}
    <!-- Bloco de origem: cor CHAPADA, raio 0, full-bleed. Tap troca a tinta. -->
    <button
      class="origin pressable"
      style="background: rgb({active.sourceR}, {active.sourceG}, {active.sourceB}); color: {contrastOn(active.sourceR, active.sourceG, active.sourceB)};"
      onclick={() => (searchOpen = true)}
    >
      <span class="origin-tag font-mono">Tinta de origem</span>
      <span class="origin-name">{active.sourceName}</span>
      <span class="origin-meta font-mono">
        {active.sourceManufacturer}{sourcePaint?.line ? ` · ${sourcePaint.line}` : ''}&nbsp;&nbsp;&nbsp;#{[active.sourceR, active.sourceG, active.sourceB].map(n => n.toString(16).padStart(2, '0').toUpperCase()).join('')}
      </span>
    </button>

    <div class="result-body">
      {#if recipes.length > 1}
        <div class="brand-tabs">
          {#each recipes as r, i (r.targetManufacturer)}
            <button class="brand-tab pressable" class:active={i === activeIdx} onclick={() => (activeIdx = i)}>
              {r.targetManufacturer}
              <span class="font-mono">ΔE {r.deltaE.toFixed(1)}</span>
            </button>
          {/each}
        </div>
      {/if}

      {#if !active.reproducible}
        <div class="notice-card animate-rise" style="margin-bottom: 16px;">
          <p class="notice-title">Essa marca não alcança a cor</p>
          <p class="notice-text">
            Melhor resultado ficou em ΔE {active.deltaE.toFixed(1)}. A fórmula abaixo é a
            aproximação mais próxima com {active.targetManufacturer}.
          </p>
          <button class="notice-link pressable" onclick={() => (brandSheetOpen = true)}>Tentar outra marca</button>
        </div>
      {/if}

      {#if sameBrand}
        <p class="own-note animate-rise">
          {#if active.ingredients.length === 1}
            A {active.targetManufacturer} já tem um tom equivalente: use a tinta abaixo direto, sem misturar.
          {:else}
            Esta cor é da própria {active.targetManufacturer}. A fórmula abaixo reproduz o tom com outras tintas dela.
          {/if}
        </p>
      {/if}

      <h2 class="h-formula font-display">Fórmula</h2>
      <p class="formula-sub font-mono">{active.targetManufacturer} · {totalDrops} {totalDrops === 1 ? 'gota' : 'gotas'}</p>

      <FormulaRibbon
        segments={active.ingredients.map(ing => ({ r: ing.r, g: ing.g, b: ing.b, percentage: ing.percentage, code: ing.code }))}
      />

      <div class="ings">
        {#each active.ingredients as ing, i (ing.paintId)}
          <div class="ing-row">
            <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={46} />
            <div class="ing-text">
              <span class="ing-name">{ing.name}</span>
              <span class="ing-meta font-mono">{ing.code ? `${ing.code} · ` : ''}{drops[i]} {drops[i] === 1 ? 'gota' : 'gotas'}</span>
            </div>
            <span class="ing-pct font-display">{Math.round(ing.percentage)}%</span>
          </div>
        {/each}
      </div>

      {#if active.tips.length > 0}
        <p class="tip">{active.tips[0]}</p>
      {/if}

      <!-- ΔE00: leitura de instrumento -->
      <div class="delta-block">
        <button class="delta-main pressable" onclick={() => (deltaSheetOpen = true)}>
          <span class="section-label">ΔE00</span>
          <span class="delta-read">
            <span class="delta-num font-mono">{active.deltaE.toFixed(1)}</span>
            <span class="delta-verdict" class:good={deltaIsGood(active.deltaE)}>{deltaVerdict(active.deltaE)}</span>
          </span>
        </button>
        <div class="delta-pair" aria-hidden="true">
          <span class="delta-sw">
            <span style="background: rgb({active.sourceR}, {active.sourceG}, {active.sourceB});"></span>
            <span class="font-mono">alvo</span>
          </span>
          <span class="delta-sw">
            <span style="background: rgb({active.resultR}, {active.resultG}, {active.resultB});"></span>
            <span class="font-mono">mistura</span>
          </span>
        </div>
      </div>

      {#if !active.reproducible && bestBrands.length > 0}
        <div class="escape animate-rise">
          <p class="section-label" style="margin-bottom: 4px;">Sai melhor nestas marcas</p>
          {#each bestBrands as b (b.manufacturerId)}
            <button class="escape-row pressable" onclick={() => pickBrandSuggestion(b)}>
              <span class="swatch-flat" style="width: 36px; height: 36px; background: rgb({b.r}, {b.g}, {b.b});"></span>
              <span class="escape-text">
                <span class="escape-brand">{b.manufacturer}</span>
                <span class="escape-paint font-mono">{b.name}{b.code ? ` · ${b.code}` : ''}</span>
              </span>
              <span class="font-mono escape-delta">ΔE {b.deltaE.toFixed(1)}</span>
            </button>
          {/each}
          <button class="btn-ghost" style="width: 100%; margin-top: 10px;" onclick={seeReadyPaints}>
            Ver tintas prontas mais próximas
          </button>
        </div>
      {/if}

      <button class="btn-primary" style="margin-top: 20px;" onclick={() => (benchOpen = true)}>
        Modo bancada
      </button>
      <div class="share-row">
        <button class="share-btn pressable" onclick={copyRecipe}><Icon name="copy" size={15} /> Copiar receita</button>
        <button class="share-btn pressable" onclick={shareRecipe}><Icon name="share" size={15} /> Compartilhar</button>
      </div>

      <p class="screen-note">Cores de tela são aproximadas. Confie no ΔE.</p>
    </div>
  {:else}
    <!-- ── Formulário: origem + estante + CTA ── -->
    {#if sourcePaint}
      <button
        class="origin pressable"
        style="background: rgb({sourcePaint.r}, {sourcePaint.g}, {sourcePaint.b}); color: {contrastOn(sourcePaint.r, sourcePaint.g, sourcePaint.b)};"
        onclick={() => (searchOpen = true)}
      >
        <span class="origin-tag font-mono">Tinta de origem</span>
        <span class="origin-name">{sourcePaint.name}</span>
        <span class="origin-meta font-mono">
          {sourcePaint.manufacturer}{sourcePaint.line ? ` · ${sourcePaint.line}` : ''}&nbsp;&nbsp;&nbsp;{hexOf(sourcePaint)}
        </span>
      </button>
    {:else}
      <button class="origin empty pressable" onclick={() => (searchOpen = true)}>
        <span class="origin-tag font-mono">Tinta de origem</span>
        <span class="origin-prompt"><Icon name="search" size={18} /> Nome ou código da tinta…</span>
      </button>
    {/if}

    <div class="form-body">
      {#if computing}
        <!-- Board Estados: skeleton + linha mono -->
        <div class="calc" aria-live="polite">
          <div class="skeleton" style="height: 48px; margin-bottom: 14px;"></div>
          {#each [0, 1, 2] as k (k)}
            <div class="calc-row">
              <div class="skeleton" style="width: 42px; height: 42px;"></div>
              <div style="flex: 1; display: flex; flex-direction: column; gap: 6px;">
                <div class="skeleton" style="height: 12px; width: 70%;"></div>
                <div class="skeleton" style="height: 12px; width: 45%;"></div>
              </div>
              <div class="skeleton" style="width: 40px; height: 18px;"></div>
            </div>
          {/each}
          <p class="calc-note font-mono">Testando combinações no catálogo {shelfNames[0] ?? ''}…</p>
        </div>
      {:else}
        {#if stock.paints.length > 0}
          <button
            class="stock-toggle pressable"
            class:on={stockPriority}
            onclick={() => (stockPriority = !stockPriority)}
            aria-pressed={stockPriority}
          >
            <span class="stock-toggle-text">
              <span class="stock-toggle-title">Priorizar meu estoque</span>
              <span class="stock-toggle-sub font-mono">{stockPriority ? `usando ${stock.paints.length} tintas suas` : `${stock.paints.length} tintas cadastradas`}</span>
            </span>
            <span class="switch" class:on={stockPriority}><span class="knob"></span></span>
          </button>
        {/if}

        <button class="btn-primary" style="margin-top: 18px;" disabled={!canMesclar} onclick={mesclar}>
          Gerar fórmula
        </button>

        {#if !sourcePaint && recents.mesclas.length === 0}
          <button class="example pressable" onclick={runExample}>
            <span class="example-eyebrow font-mono">Experimente</span>
            <span class="example-text">Mephiston Red com tintas Vallejo</span>
          </button>
        {/if}

        {#if recents.mesclas.length > 0}
          <p class="section-label" style="margin: 26px 0 4px;">Últimas mesclas</p>
          <div class="history">
            {#each recents.mesclas as m (m.sourceId + '|' + m.targetManufacturer)}
              <button class="history-row pressable" onclick={() => rerun(m)}>
                <span class="color-pair" style="width: 30px; height: 19px;">
                  <span style="background: rgb({m.sourceRGB[0]}, {m.sourceRGB[1]}, {m.sourceRGB[2]});"></span>
                  <span style="background: rgb({m.resultRGB[0]}, {m.resultRGB[1]}, {m.resultRGB[2]});"></span>
                </span>
                <span class="history-text">{m.sourceName} → {m.targetManufacturer}</span>
                <span class="font-mono history-delta">ΔE {m.deltaE.toFixed(1)}</span>
              </button>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<FullScreenSearch
  open={searchOpen}
  onClose={() => (searchOpen = false)}
  onSelect={pickSource}
  recentIds={recents.paintIds}
/>
<BrandSheet open={brandSheetOpen} onClose={() => (brandSheetOpen = false)} mode="shelf" />
<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} deltaE={active?.deltaE ?? null} />
{#if benchOpen && active}
  <BenchMode recipe={active} {drops} {totalDrops} onClose={() => (benchOpen = false)} />
{/if}

<style>
  .mesclar {
    padding: 0 0 24px;
  }

  /* ── Header papel com hairline ── */
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    min-height: 56px;
    padding: 8px 16px;
    background: var(--papel);
    border-bottom: 1px solid var(--hairline);
  }

  .head-brand {
    display: inline-flex;
    align-items: center;
    gap: 9px;
  }

  .head-name {
    font-size: 20px;
    font-weight: 750;
    letter-spacing: -0.015em;
    color: var(--grafite);
  }

  .head-shelf {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 40px;
    padding: 0 14px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-pill);
    background: var(--papel);
    font-size: 14px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
  }

  /* ── Bloco de origem: cor chapada, raio 0, full-bleed ── */
  .origin {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: flex-end;
    gap: 4px;
    width: 100%;
    min-height: 172px;
    padding: 16px;
    text-align: left;
  }

  .origin-tag {
    position: absolute;
    top: 14px;
    left: 16px;
    font-size: 10.5px;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    opacity: 0.8;
  }

  .origin-name {
    font-family: var(--font-display);
    font-optical-sizing: auto;
    font-size: 38px;
    font-weight: 750;
    letter-spacing: -0.02em;
    line-height: 1.05;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .origin-meta {
    font-size: 12.5px;
    letter-spacing: 0.03em;
    opacity: 0.85;
  }

  .origin.empty {
    background: var(--ink-800);
    color: var(--ink-500);
    border-bottom: 1px solid var(--hairline);
  }

  .origin-prompt {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    font-size: 16px;
    color: var(--ink-500);
  }

  .result-body,
  .form-body {
    padding: 18px 16px 0;
  }

  /* ── Pills de marca (várias receitas) ── */
  .brand-tabs {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 4px;
    margin-bottom: 14px;
    scrollbar-width: none;
  }

  .brand-tabs::-webkit-scrollbar {
    display: none;
  }

  .brand-tab {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 42px;
    padding: 0 14px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--hairline);
    background: var(--papel);
    font-size: 14px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .brand-tab span {
    font-size: 11px;
    color: var(--ink-500);
  }

  .brand-tab.active {
    background: var(--grafite);
    border-color: var(--grafite);
    color: var(--papel);
  }

  .brand-tab.active span {
    color: color-mix(in srgb, var(--papel) 70%, transparent);
  }

  .own-note {
    font-size: 13.5px;
    color: var(--ink-500);
    line-height: 1.5;
    margin-bottom: 14px;
  }

  .h-formula {
    font-size: 26px;
    font-weight: 750;
    color: var(--grafite);
  }

  .formula-sub {
    font-size: 11px;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--ink-500);
    margin: 2px 0 12px;
  }

  /* ── Ingredientes: linhas hairline ── */
  .ings {
    margin-top: 14px;
  }

  .ing-row {
    display: flex;
    align-items: center;
    gap: 14px;
    min-height: 64px;
    padding: 9px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .ing-row:first-child {
    border-top: 1px solid var(--hairline);
  }

  .ing-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .ing-name {
    font-size: 16px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ing-meta {
    font-size: 12px;
    color: var(--ink-500);
  }

  .ing-pct {
    flex-shrink: 0;
    font-size: 27px;
    font-weight: 750;
    color: var(--grafite);
    letter-spacing: -0.01em;
  }

  .tip {
    font-size: 13.5px;
    color: var(--ink-500);
    line-height: 1.5;
    margin-top: 12px;
  }

  /* ── ΔE00: leitura de instrumento ── */
  .delta-block {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    margin-top: 18px;
    padding: 14px 0;
    border-top: 1px solid var(--hairline);
    border-bottom: 1px solid var(--hairline);
  }

  .delta-main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    text-align: left;
    min-width: 0;
  }

  .delta-read {
    display: flex;
    align-items: baseline;
    gap: 12px;
    flex-wrap: wrap;
  }

  .delta-num {
    font-size: 54px;
    font-weight: 600;
    line-height: 1;
    color: var(--grafite);
    letter-spacing: -0.02em;
  }

  .delta-verdict {
    font-size: 13.5px;
    font-weight: 700;
    color: var(--grafite);
    max-width: 150px;
    line-height: 1.25;
  }

  .delta-verdict.good {
    color: var(--laca);
  }

  .delta-pair {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .delta-sw {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
  }

  .delta-sw > span:first-child {
    width: 44px;
    height: 36px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.12);
  }

  .delta-sw .font-mono {
    font-size: 9.5px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--ink-500);
  }

  /* ── Escape (marcas melhores) ── */
  .escape {
    margin-top: 18px;
  }

  .escape-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 56px;
    padding: 8px 0;
    text-align: left;
    border-bottom: 1px solid var(--hairline);
  }

  .escape-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .escape-brand {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
  }

  .escape-paint {
    font-size: 12px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .escape-delta {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  .share-row {
    display: flex;
    justify-content: center;
    gap: 22px;
    margin-top: 12px;
  }

  .share-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 44px;
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-500);
  }

  .screen-note {
    text-align: center;
    font-size: 12px;
    color: var(--ink-500);
    padding: 14px 0 4px;
  }

  /* ── Form ── */
  .stock-toggle {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 56px;
    padding: 8px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .stock-toggle-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .stock-toggle-title {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
  }

  .stock-toggle-sub {
    font-size: 11px;
    color: var(--ink-500);
  }

  /* Toggle: trilho pílula, laca quando on, knob branco */
  .switch {
    flex-shrink: 0;
    width: 44px;
    height: 26px;
    border-radius: var(--radius-pill);
    background: var(--hairline);
    position: relative;
    transition: background 0.18s ease;
  }

  .switch.on {
    background: var(--laca);
  }

  .knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.18s ease;
  }

  .switch.on .knob {
    transform: translateX(18px);
  }

  .example {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 3px;
    width: 100%;
    margin-top: 22px;
    padding: 14px 16px;
    border: 1px dashed var(--hairline);
    border-radius: var(--radius-surface);
    background: var(--papel);
    text-align: left;
  }

  .example-eyebrow {
    font-size: 10.5px;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--lacquer-deep);
  }

  .example-text {
    font-size: 15px;
    font-weight: 500;
    color: var(--grafite);
  }

  .history {
    display: flex;
    flex-direction: column;
  }

  .history-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 54px;
    padding: 8px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .history-text {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    font-weight: 500;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .history-delta {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  /* ── Calculando (board Estados) ── */
  .calc-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 0;
  }

  .calc-note {
    font-size: 12px;
    color: var(--ink-500);
    margin-top: 10px;
  }
</style>
