<script lang="ts">
  // Mesclar — a jornada nº 1: "como reproduzo esta cor com o que tenho?"
  // Form de 2 campos + CTA acima da dobra; resultado renderiza NA MESMA tela
  // (o form recolhe pra uma barra compacta; o back do Android re-expande).
  // A receita é calculada por marca da estante — a melhor aparece primeiro,
  // as outras viram chips.
  import Icon from '../components/Icon.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import DeltaBadge from '../components/DeltaBadge.svelte';
  import FullScreenSearch from '../components/FullScreenSearch.svelte';
  import BrandSheet from '../components/BrandSheet.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import BenchMode from '../components/BenchMode.svelte';
  import { pushLayer, switchTab } from '../nav.svelte';
  import { allManufacturers, allPaints, paintById, type Paint } from '../services/catalog';
  import { suggestEquivalentRecipe, bestBrandsFor, type EquivalentRecipe, type BrandBest } from '../services/engine';
  import { shelf, toggleShelf } from '../services/shelf.svelte';
  import { appState } from '../appState.svelte';
  import { recents, rememberMescla, rememberPaint } from '../recents.svelte';
  import { toast } from '../toast.svelte';

  let sourcePaint: Paint | null = $state(null);
  let searchOpen = $state(false);
  let brandSheetOpen = $state(false);
  let deltaSheetOpen = $state(false);
  let benchOpen = $state(false);
  let computing = $state(false);
  let recipes: EquivalentRecipe[] = $state([]);
  let activeIdx = $state(0);
  let showResult = $state(false);
  let unit: 'drops' | 'percent' = $state('drops'); // bancada mede em gotas
  let bestBrands: BrandBest[] = $state([]);
  let firstResultHintSeen = $state(localStorage.getItem('mescla.hint.deltae') === '1');

  let closeResultLayer: (() => void) | null = null;

  // Deep-link interno: Catálogo → "Mesclar esta cor".
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
  let canMesclar = $derived(sourcePaint !== null && shelf.manufacturerIds.length > 0);
  let sameBrand = $derived(active !== null && active.targetManufacturer === active.sourceManufacturer);

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
    if (!sourcePaint || shelf.manufacturerIds.length === 0 || computing) return;
    computing = true;
    bestBrands = [];
    try {
      const found: EquivalentRecipe[] = [];
      for (const brandId of shelf.manufacturerIds) {
        try {
          found.push(await suggestEquivalentRecipe(sourcePaint.id, brandId));
        } catch {
          /* marca sem candidatos (ex.: só a própria tinta) — pula */
        }
      }
      if (found.length === 0) {
        toast('Nenhuma marca da estante tem tintas pra essa mistura.', 'error');
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

  function dismissHint() {
    firstResultHintSeen = true;
    localStorage.setItem('mescla.hint.deltae', '1');
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
    const measure = (i: number) =>
      unit === 'drops'
        ? `${drops[i]} ${drops[i] === 1 ? 'gota' : 'gotas'}`
        : `${active!.ingredients[i].percentage.toFixed(1)}%`;
    return [
      `${active.sourceName} (${active.sourceManufacturer}) → ${active.targetManufacturer}`,
      ...(unit === 'drops' ? [`Mistura de ${totalDrops} gotas`] : []),
      ...active.ingredients.map((ing, i) => `${measure(i)}  ${ing.name}${ing.code ? ` (${ing.code})` : ''}`),
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
</script>

<div class="mesclar">
  {#if showResult && active}
    <!-- Barra compacta: tap re-expande o form (mesmo efeito do back) -->
    <button class="compact-bar pressable" onclick={collapseResult}>
      <span class="color-pair" style="width: 26px; height: 17px;">
        <span style="background: rgb({active.sourceR}, {active.sourceG}, {active.sourceB});"></span>
        <span style="background: rgb({active.resultR}, {active.resultG}, {active.resultB});"></span>
      </span>
      <span class="compact-text">{active.sourceName} → {active.targetManufacturer}</span>
      <Icon name="chevron-down" size={16} />
    </button>

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
      <div class="warn-banner animate-rise">
        <span class="warn-icon"><Icon name="info" size={18} /></span>
        <div>
          <p class="warn-title">Nenhuma mistura de {active.targetManufacturer} chega perto desta cor</p>
          <p class="warn-text">A receita abaixo é a melhor aproximação. Compare o par antes de usar.</p>
        </div>
      </div>
    {/if}

    {#if sameBrand}
      <div class="own-note animate-rise">
        {#if active.ingredients.length === 1}
          A {active.targetManufacturer} já tem um tom equivalente — use a tinta abaixo direto, sem misturar.
        {:else}
          Esta cor é da própria {active.targetManufacturer} — a mistura abaixo reproduz o tom com <strong>outras</strong> tintas dela.
        {/if}
      </div>
    {/if}

    <!-- O par: alvo × mistura, colados — o momento da verdade -->
    <div class="verdict animate-rise">
      <div class="verdict-pair">
        <div class="verdict-half" style="background: rgb({active.sourceR}, {active.sourceG}, {active.sourceB});">
          <span class="verdict-tag font-mono">alvo</span>
        </div>
        <div class="verdict-half" style="background: rgb({active.resultR}, {active.resultG}, {active.resultB});">
          <span class="verdict-tag font-mono">mistura</span>
        </div>
      </div>
      <div class="verdict-badge">
        <DeltaBadge deltaE={active.deltaE} size="lg" onclick={() => { deltaSheetOpen = true; dismissHint(); }} />
        {#if !firstResultHintSeen}
          <button class="hint-bubble animate-rise" onclick={() => { deltaSheetOpen = true; dismissHint(); }}>
            Toque para entender a escala ↑
          </button>
        {/if}
      </div>
    </div>

    <!-- Receita -->
    <div class="recipe panel animate-rise">
      <div class="recipe-head">
        <h3 class="font-display">{active.reproducible ? 'Receita' : 'Melhor aproximação'}</h3>
        <div class="unit-toggle">
          <button class:active={unit === 'drops'} onclick={() => (unit = 'drops')}>gotas</button>
          <button class:active={unit === 'percent'} onclick={() => (unit = 'percent')}>%</button>
        </div>
      </div>

      {#if unit === 'drops'}
        <p class="recipe-sub">mistura de {totalDrops} {totalDrops === 1 ? 'gota' : 'gotas'} — multiplique (2×, 3×…) pra fazer mais</p>
      {/if}

      {#each active.ingredients as ing, i (ing.paintId)}
        <div class="ing-row">
          <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={44} />
          <div class="ing-text">
            <span class="ing-name">{ing.name}</span>
            {#if ing.code}<span class="ing-code font-mono">{ing.code}</span>{/if}
          </div>
          <div class="ing-measure font-mono">
            {#if unit === 'drops'}
              <strong>{drops[i]}</strong> <span>{drops[i] === 1 ? 'gota' : 'gotas'}</span>
            {:else}
              <strong>{ing.percentage.toFixed(1)}%</strong>
            {/if}
          </div>
        </div>
      {/each}

      <div class="recipe-actions">
        <button class="btn-primary" onclick={() => (benchOpen = true)}>
          <Icon name="flask" size={18} />
          Modo bancada
        </button>
        <div class="recipe-actions-row">
          <button class="btn-ghost" style="flex: 1;" onclick={copyRecipe}>
            <Icon name="copy" size={16} />
            Copiar
          </button>
          <button class="btn-ghost" style="flex: 1;" onclick={shareRecipe}>
            <Icon name="share" size={16} />
            Compartilhar
          </button>
        </div>
      </div>
    </div>

    {#if !active.reproducible && bestBrands.length > 0}
      <div class="panel escape animate-rise">
        <h3 class="font-display">Sai melhor nestas marcas</h3>
        {#each bestBrands as b (b.manufacturerId)}
          <button class="escape-row pressable" onclick={() => pickBrandSuggestion(b)}>
            <span class="swatch-flat" style="width: 34px; height: 34px; background: rgb({b.r}, {b.g}, {b.b});"></span>
            <span class="escape-text">
              <span class="escape-brand">{b.manufacturer}</span>
              <span class="escape-paint">{b.name}{b.code ? ` · ${b.code}` : ''}</span>
            </span>
            <span class="font-mono escape-delta">ΔE {b.deltaE.toFixed(1)}</span>
          </button>
        {/each}
        <button class="btn-ghost" style="width: 100%; margin-top: 10px;" onclick={seeReadyPaints}>
          <Icon name="pipette" size={16} />
          Ver tintas prontas mais próximas
        </button>
      </div>
    {/if}

    {#if active.tips.length > 0}
      <div class="panel tips animate-rise">
        <h3 class="font-display"><Icon name="info" size={15} /> Dicas de ajuste</h3>
        <ul>
          {#each active.tips as tip}
            <li>{tip}</li>
          {/each}
        </ul>
      </div>
    {/if}

    <p class="screen-note">Cores de tela são aproximadas — confie no ΔE.</p>
  {:else}
    <!-- ── Formulário: 2 campos + CTA, tudo acima da dobra ── -->
    <header class="mesclar-head">
      <h1 class="screen-title">Mescla</h1>
      <p class="mesclar-tagline">cor certa, qualquer marca</p>
    </header>

    <p class="field-label">Quero esta cor</p>
    {#if sourcePaint}
      <button class="field-filled pressable" onclick={() => (searchOpen = true)}>
        <PaintBottle r={sourcePaint.r} g={sourcePaint.g} b={sourcePaint.b} size={44} label={sourcePaint.code} />
        <span class="field-filled-text">
          <span class="field-filled-name">{sourcePaint.name}</span>
          <span class="field-filled-meta"><span class="font-mono">{sourcePaint.code}</span> · {sourcePaint.manufacturer}</span>
        </span>
        <span class="field-swap">Trocar</span>
      </button>
    {:else}
      <button class="field-empty pressable" onclick={() => (searchOpen = true)}>
        <Icon name="search" size={19} />
        <span>Nome ou código da tinta…</span>
      </button>
    {/if}

    <p class="field-label" style="margin-top: 18px;">Tenho tintas de</p>
    <button class="field-empty pressable" class:has-brands={shelfNames.length > 0} onclick={() => (brandSheetOpen = true)}>
      {#if shelfNames.length === 0}
        <Icon name="building" size={19} />
        <span>Escolher marcas…</span>
      {:else}
        <span class="brand-chips">
          {#each shelfNames.slice(0, 3) as name}
            <span class="mini-chip">{name}</span>
          {/each}
          {#if shelfNames.length > 3}
            <span class="mini-chip more">+{shelfNames.length - 3}</span>
          {/if}
        </span>
      {/if}
      <Icon name="chevron-down" size={16} />
    </button>

    <button class="btn-primary" style="margin-top: 22px;" disabled={!canMesclar || computing} onclick={mesclar}>
      {#if computing}
        Calculando mistura…
      {:else}
        <Icon name="droplet" size={19} />
        Mesclar
      {/if}
    </button>

    {#if !sourcePaint && recents.mesclas.length === 0}
      <button class="example pressable" onclick={runExample}>
        <span class="example-eyebrow">Experimente</span>
        <span class="example-text">Mephiston Red → com tintas Vallejo</span>
      </button>
    {/if}

    {#if recents.mesclas.length > 0}
      <p class="field-label" style="margin-top: 26px;">Últimas mesclas</p>
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

<FullScreenSearch
  open={searchOpen}
  onClose={() => (searchOpen = false)}
  onSelect={p => { sourcePaint = p; rememberPaint(p.id); }}
  recentIds={recents.paintIds}
/>
<BrandSheet open={brandSheetOpen} onClose={() => (brandSheetOpen = false)} mode="shelf" />
<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} deltaE={active?.deltaE ?? null} />
{#if benchOpen && active}
  <BenchMode recipe={active} {drops} {totalDrops} onClose={() => (benchOpen = false)} />
{/if}

<style>
  .mesclar {
    padding: 16px 16px 24px;
  }

  .mesclar-head {
    padding: 8px 0 18px;
  }

  .mesclar-tagline {
    font-family: var(--font-mono);
    font-size: 11px;
    letter-spacing: 0.14em;
    color: var(--ink-500);
    margin-top: 2px;
  }

  .field-label {
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--ink-500);
    margin-bottom: 8px;
  }

  .field-empty,
  .field-filled {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 64px;
    padding: 10px 16px;
    border: 1px solid var(--ink-700);
    border-radius: 12px;
    background: var(--ink-900);
    text-align: left;
    color: var(--ink-500);
    font-size: 16px;
  }

  .field-empty.has-brands {
    justify-content: space-between;
  }

  .field-filled-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .field-filled-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .field-filled-meta {
    font-size: 13px;
    color: var(--ink-500);
  }

  .field-swap {
    flex-shrink: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--lacquer-deep);
  }

  .brand-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    flex: 1;
    min-width: 0;
  }

  .mini-chip {
    display: inline-flex;
    align-items: center;
    padding: 5px 11px;
    border-radius: 999px;
    background: var(--ink-800);
    color: var(--ink-300);
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
  }

  .mini-chip.more {
    background: color-mix(in srgb, var(--lacquer) 12%, transparent);
    color: var(--lacquer-deep);
  }

  .example {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 3px;
    width: 100%;
    margin-top: 22px;
    padding: 14px 16px;
    border: 1px dashed var(--ink-600);
    border-radius: 12px;
    background: var(--ink-850);
    text-align: left;
  }

  .example-eyebrow {
    font-family: var(--font-mono);
    font-size: 10.5px;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--lacquer-deep);
  }

  .example-text {
    font-size: 15px;
    font-weight: 500;
    color: var(--ink-300);
  }

  .history {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .history-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 52px;
    padding: 8px 14px;
    border: 1px solid var(--ink-700);
    border-radius: 10px;
    background: var(--ink-900);
    text-align: left;
  }

  .history-text {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    font-weight: 500;
    color: var(--ink-300);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .history-delta {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  /* ── Resultado ── */
  .compact-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 52px;
    padding: 8px 14px;
    border: 1px solid var(--ink-700);
    border-radius: 12px;
    background: var(--ink-900);
    margin-bottom: 12px;
    text-align: left;
    color: var(--ink-500);
  }

  .compact-text {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .brand-tabs {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 4px;
    margin-bottom: 12px;
    scrollbar-width: none;
  }

  .brand-tab {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    padding: 0 14px;
    border-radius: 999px;
    border: 1px solid var(--ink-700);
    background: var(--ink-900);
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-300);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .brand-tab span {
    font-size: 11px;
    color: var(--ink-500);
  }

  .brand-tab.active {
    background: var(--paper);
    border-color: var(--paper);
    color: var(--ink-950);
  }

  .brand-tab.active span {
    color: var(--ink-600);
  }

  .warn-banner {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 14px 16px;
    border-radius: 12px;
    border: 1px solid color-mix(in srgb, var(--delta-poor) 45%, transparent);
    background: color-mix(in srgb, var(--delta-poor) 10%, transparent);
    margin-bottom: 12px;
  }

  .warn-icon {
    color: var(--delta-poor);
    flex-shrink: 0;
    margin-top: 2px;
  }

  .warn-title {
    font-size: 14.5px;
    font-weight: 600;
    color: var(--delta-poor);
  }

  .warn-text {
    font-size: 13px;
    color: var(--ink-300);
    margin-top: 2px;
  }

  .own-note {
    padding: 12px 16px;
    border-radius: 12px;
    background: var(--ink-800);
    font-size: 13.5px;
    color: var(--ink-300);
    margin-bottom: 12px;
    line-height: 1.5;
  }

  .verdict {
    margin-bottom: 14px;
  }

  .verdict-pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    height: 140px;
    border-radius: 14px;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.18);
  }

  .verdict-half {
    position: relative;
  }

  .verdict-tag {
    position: absolute;
    bottom: 8px;
    left: 8px;
    font-size: 10px;
    padding: 3px 8px;
    border-radius: 4px;
    background: rgba(26, 23, 18, 0.55);
    color: rgba(255, 255, 255, 0.92);
  }

  /* Abaixo do par, nunca por cima: o badge não pode cobrir a junção das
     cores — é ali que o olho compara alvo × mistura. */
  .verdict-badge {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    position: relative;
  }

  .hint-bubble {
    font-size: 12.5px;
    font-weight: 500;
    color: var(--lacquer-deep);
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    border-radius: 999px;
    padding: 6px 14px;
    min-height: 36px;
  }

  .recipe {
    padding: 16px;
    margin-bottom: 12px;
  }

  .recipe-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .recipe-head h3,
  .escape h3,
  .tips h3 {
    font-size: 16px;
    font-weight: 600;
    color: var(--ink-100);
  }

  .unit-toggle {
    display: inline-flex;
    padding: 2px;
    border-radius: 9px;
    background: var(--ink-800);
  }

  .unit-toggle button {
    min-height: 38px;
    padding: 0 16px;
    border-radius: 7px;
    font-size: 13.5px;
    font-weight: 600;
    color: var(--ink-500);
  }

  .unit-toggle button.active {
    background: var(--lacquer);
    color: white;
  }

  .recipe-sub {
    font-size: 12.5px;
    color: var(--ink-500);
    margin-bottom: 10px;
  }

  .ing-row {
    display: flex;
    align-items: center;
    gap: 14px;
    min-height: 60px;
    padding: 8px 0;
    border-bottom: 1px solid var(--ink-800);
  }

  .ing-row:last-of-type {
    border-bottom: none;
  }

  .ing-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .ing-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ing-code {
    font-size: 12px;
    color: var(--lacquer-tint);
  }

  .ing-measure {
    flex-shrink: 0;
    text-align: right;
    color: var(--ink-100);
  }

  .ing-measure strong {
    font-size: 24px;
    font-weight: 600;
  }

  .ing-measure span {
    font-size: 12px;
    color: var(--ink-500);
  }

  .recipe-actions {
    margin-top: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .recipe-actions-row {
    display: flex;
    gap: 10px;
  }

  .escape {
    padding: 16px;
    margin-bottom: 12px;
  }

  .escape-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 56px;
    padding: 8px 4px;
    text-align: left;
    border-bottom: 1px solid var(--ink-800);
  }

  .escape-row:last-of-type {
    border-bottom: none;
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
    font-weight: 600;
    color: var(--ink-100);
  }

  .escape-paint {
    font-size: 12.5px;
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

  .tips {
    padding: 16px;
    margin-bottom: 12px;
  }

  .tips h3 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }

  .tips ul {
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .tips li {
    font-size: 14px;
    color: var(--ink-300);
    line-height: 1.5;
  }

  .screen-note {
    text-align: center;
    font-size: 12px;
    color: var(--ink-500);
    padding: 8px 0 4px;
  }
</style>
