<script lang="ts">
  // Roda — ensina a usar a roda cromática e a clarear/escurecer do jeito certo
  // (deslocando o matiz, não só branco e preto). Sugere a tinta real mais
  // próxima de cada passo da rampa.
  import Icon from '../components/Icon.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import DeltaBadge from '../components/DeltaBadge.svelte';
  import BottomSheet from '../components/BottomSheet.svelte';
  import WheelGuide from '../components/WheelGuide.svelte';
  import {
    findSimilar,
    suggestRecipeForColor,
    type SearchResult,
    type EquivalentRecipe,
  } from '../services/engine';
  import { allManufacturers } from '../services/catalog';
  import { shelf } from '../services/shelf.svelte';
  import {
    rgbToHsl,
    hslToRgb,
    rgbToHex,
    hexToRgb,
    buildRamp,
    buildNaiveRamp,
    harmony,
    colorName,
    HARMONY_LABEL,
    HARMONY_HINT,
    type HarmonyKind,
    type RGB,
  } from '../color/theory';

  let hue = $state(14);
  let sat = $state(0.82);
  let light = $state(0.5);

  let baseRgb = $derived(hslToRgb({ h: hue, s: sat, l: light }));
  let baseHex = $derived(rgbToHex(baseRgb));
  let ramp = $derived(buildRamp(baseRgb, { highlights: 3, shadows: 2 }));
  let naive = $derived(buildNaiveRamp(baseRgb, { highlights: 3, shadows: 2 }));

  let harmonyKind: HarmonyKind = $state('complementary');
  let harmonySwatches = $derived(harmony({ h: hue, s: sat, l: light }, harmonyKind));

  let showNaive = $state(false);
  let onlyShelf = $state(false);
  let hexInput = $state('');
  // Marca-alvo POR PASSO ('' = qualquer marca) — dá pra montar a rampa
  // misturando marcas. Tem prioridade sobre o filtro "minhas marcas".
  let brandByLevel: Record<number, string> = $state({});

  // ---- Roda ----------------------------------------------------------------
  const SIZE = 280;
  const R = SIZE / 2;
  let wheelEl: HTMLDivElement;
  let dragging = false;

  const wheelBg = (() => {
    const stops: string[] = [];
    for (let h = 0; h <= 360; h += 20) stops.push(`hsl(${h} 92% 55%) ${(h / 360) * 100}%`);
    return `radial-gradient(circle at center, #fff 0%, rgba(255,255,255,0) 72%), conic-gradient(from 0deg, ${stops.join(', ')})`;
  })();

  function polar(h: number, s: number) {
    const rad = (h * Math.PI) / 180;
    return { x: R + R * s * Math.sin(rad), y: R - R * s * Math.cos(rad) };
  }
  let marker = $derived(polar(hue, sat));

  function pick(e: PointerEvent) {
    const rect = wheelEl.getBoundingClientRect();
    const dx = e.clientX - rect.left - R;
    const dy = e.clientY - rect.top - R;
    sat = Math.min(1, Math.sqrt(dx * dx + dy * dy) / R);
    hue = (((Math.atan2(dx, -dy) * 180) / Math.PI) % 360 + 360) % 360;
  }
  function onDown(e: PointerEvent) {
    dragging = true;
    wheelEl.setPointerCapture(e.pointerId);
    if (navigator.vibrate) navigator.vibrate(6);
    pick(e);
  }
  function onMove(e: PointerEvent) {
    if (dragging) pick(e);
  }
  function onUp(e: PointerEvent) {
    dragging = false;
    wheelEl.releasePointerCapture(e.pointerId);
  }

  function applyHex() {
    const rgb = hexToRgb(hexInput);
    if (!rgb) return;
    const h = rgbToHsl(rgb);
    hue = h.h;
    sat = h.s;
    light = h.l;
    hexInput = '';
  }

  function rgbCss(c: RGB) {
    return `rgb(${c.r}, ${c.g}, ${c.b})`;
  }

  let shelfNames = $derived(
    allManufacturers()
      .filter(m => shelf.manufacturerIds.includes(m.id))
      .map(m => m.name),
  );

  let brands = $derived(
    allManufacturers()
      .map(m => ({ id: m.id, name: m.name }))
      .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR')),
  );

  function brandId(name: string): number | null {
    return brands.find(x => x.name === name)?.id ?? null;
  }

  // ---- Modal de mistura (marca sem a cor pronta) ---------------------------
  let recipe: EquivalentRecipe | null = $state(null);
  let recipeLoading = $state(false);
  let recipeError = $state('');
  let recipeTarget: { hex: string; brand: string } | null = $state(null);

  async function openRecipe(hex: string, rgb: RGB, brand: string) {
    const id = brandId(brand);
    if (id == null) return;
    recipeTarget = { hex, brand };
    recipe = null;
    recipeError = '';
    recipeLoading = true;
    try {
      recipe = await suggestRecipeForColor(rgb.r, rgb.g, rgb.b, id);
    } catch (e) {
      recipeError = e instanceof Error ? e.message : 'Não foi possível montar a mistura.';
    } finally {
      recipeLoading = false;
    }
  }

  function closeRecipe() {
    recipeTarget = null;
    recipe = null;
    recipeError = '';
  }

  // ---- Tinta real por passo ------------------------------------------------
  // Busca ampla cacheada por passo; a escolha por marca é derivada sem refazer.
  let rawByLevel: Record<number, SearchResult[]> = $state({});
  let timer: ReturnType<typeof setTimeout>;

  $effect(() => {
    const steps = ramp;
    clearTimeout(timer);
    timer = setTimeout(async () => {
      const entries = await Promise.all(
        steps.map(async s => {
          try {
            const res = await findSimilar(s.rgb.r, s.rgb.g, s.rgb.b, 100, 40);
            return [s.level, res] as const;
          } catch {
            return [s.level, [] as SearchResult[]] as const;
          }
        }),
      );
      rawByLevel = Object.fromEntries(entries);
    }, 200);
  });

  let paintByLevel = $derived.by(() => {
    const out: Record<number, SearchResult | null> = {};
    for (const s of ramp) {
      const res = rawByLevel[s.level] || [];
      const brand = brandByLevel[s.level] || '';
      let pick: SearchResult | undefined;
      if (brand) pick = res.find(x => x.manufacturer === brand);
      else if (onlyShelf && shelfNames.length > 0) pick = res.find(x => shelfNames.includes(x.manufacturer));
      else pick = res[0];
      out[s.level] = pick ?? null;
    }
    return out;
  });

  function setBrand(level: number, value: string) {
    brandByLevel = { ...brandByLevel, [level]: value };
  }

  const harmonyKinds: HarmonyKind[] = ['complementary', 'analogous', 'triad', 'split'];
</script>

<div class="roda">
  <WheelGuide baseRgb={baseRgb} />

  <div class="wheel-wrap">
    <div
      class="wheel"
      bind:this={wheelEl}
      role="slider"
      tabindex="0"
      aria-label="Roda cromática"
      aria-valuenow={Math.round(hue)}
      aria-valuemin="0"
      aria-valuemax="360"
      style="width: {SIZE}px; height: {SIZE}px; background: {wheelBg};"
      onpointerdown={onDown}
      onpointermove={onMove}
      onpointerup={onUp}
    >
      {#each harmonySwatches as sw}
        {@const p = polar(sw.h, Math.max(sat, 0.5))}
        <span class="harmony-dot" style="left: {p.x}px; top: {p.y}px; background: {sw.hex};"></span>
      {/each}
      <span class="wheel-marker" style="left: {marker.x}px; top: {marker.y}px; background: {baseHex};"></span>
    </div>
  </div>

  <div class="controls">
    <div class="light-row">
      <span class="light-label">Luz</span>
      <input
        type="range"
        min="0"
        max="100"
        value={Math.round(light * 100)}
        style="--track: linear-gradient(to right, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.08 }))}, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.5 }))}, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.95 }))});"
        oninput={e => (light = +(e.currentTarget as HTMLInputElement).value / 100)}
      />
    </div>

    <!-- Cor escolhida chapada; a etiqueta hex fica ao lado, fora da cor. -->
    <div class="selected-row">
      <span class="selected-chip swatch-flat" style="background: {baseHex};"></span>
      <span class="selected-hex font-mono">{baseHex}</span>
    </div>

    <div class="hex-row">
      <input
        bind:value={hexInput}
        type="text"
        class="font-mono"
        placeholder="#RRGGBB"
        maxlength="7"
        spellcheck="false"
        autocomplete="off"
        autocapitalize="off"
        onkeydown={e => e.key === 'Enter' && applyHex()}
      />
      <button class="btn-ghost" onclick={applyHex}>HEX</button>
    </div>
  </div>

  <!-- Harmonias -->
  <section class="block">
    <h2 class="block-title font-display">Harmonias</h2>
    <div class="segmented">
      {#each harmonyKinds as k}
        <button class="seg" class:active={harmonyKind === k} onclick={() => (harmonyKind = k)}>
          {HARMONY_LABEL[k]}
        </button>
      {/each}
    </div>
    <p class="hint">{HARMONY_HINT[harmonyKind]}</p>
    <div class="harmony-swatches">
      {#each harmonySwatches as sw}
        <button class="h-swatch pressable" onclick={() => (hue = sw.h)}>
          <span class="hs-color" style="background: {sw.hex};"></span>
          <span class="hs-role">{sw.role}</span>
        </button>
      {/each}
    </div>
  </section>

  <!-- Rampa -->
  <section class="block">
    <h2 class="block-title font-display">Clarear e escurecer certo</h2>
    <p class="hint">
      O matiz caminha pro <strong>azul</strong> na sombra e pro <strong>amarelo</strong> na luz — é o
      que mantém a cor viva.
    </p>

    <div class="ramp-strip">
      {#each ramp as step (step.level)}
        <div class="ramp-cell" class:is-base={step.kind === 'base'} style="background: {step.hex};"></div>
      {/each}
    </div>

    <label class="toggle pressable">
      <input type="checkbox" bind:checked={showNaive} />
      <span>Comparar com o jeito comum (só branco/preto)</span>
    </label>
    {#if showNaive}
      <div class="ramp-strip naive">
        {#each naive as c}
          <div class="naive-cell">
            <div class="ramp-cell" style="background: {rgbCss(c)};"></div>
            <span class="naive-name">{colorName(c)}</span>
            <span class="naive-hex font-mono">{rgbToHex(c)}</span>
          </div>
        {/each}
      </div>
      <p class="hint muted-hint">Repare como desbota e "suja" — perde a vida.</p>
    {/if}
  </section>

  <!-- Tintas reais por passo -->
  <section class="block">
    <div class="block-head">
      <h2 class="block-title font-display" style="margin: 0;">Tintas pra cada passo</h2>
      {#if shelfNames.length > 0}
        <label class="mini-toggle pressable">
          <input type="checkbox" bind:checked={onlyShelf} />
          <span>Minhas marcas</span>
        </label>
      {/if}
    </div>

    {#each ramp as step (step.level)}
      {@const brand = brandByLevel[step.level] ?? ''}
      {@const paint = paintByLevel[step.level]}
      <div class="step-card">
        <div class="step-top">
          <span class="step-swatch" class:is-base={step.kind === 'base'} style="background: {step.hex};"></span>
          <span class="step-why">{step.why}</span>
        </div>
        <select
          class="col-brand"
          value={brand}
          onchange={e => setBrand(step.level, (e.currentTarget as HTMLSelectElement).value)}
          aria-label="Marca para este passo"
        >
          <option value="">Qualquer marca</option>
          {#each brands as b}
            <option value={b.name}>{b.name}</option>
          {/each}
        </select>
        {#if paint}
          <div class="step-result">
            <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={34} />
            <div class="step-result-text">
              <span class="step-paint">{paint.name}</span>
              {#if !brand}<span class="step-mfr">{paint.manufacturer}</span>{/if}
            </div>
            <DeltaBadge deltaE={paint.deltaE} size="sm" />
          </div>
        {:else}
          <div class="step-empty">
            <span class="step-paint muted">sem tinta {brand ? `da ${brand}` : ''} próxima</span>
            {#if brand}
              <button class="mix-btn pressable" onclick={() => openRecipe(step.hex, step.rgb, brand)}>
                <Icon name="flask" size={14} /> Ver mistura
              </button>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  </section>
</div>

<BottomSheet
  open={!!recipeTarget}
  onClose={closeRecipe}
  title={recipeTarget ? `Mistura na ${recipeTarget.brand}` : ''}
>
  {#if recipeTarget}
    <div class="mix-target-row">
      <span class="mix-target" style="background: {recipeTarget.hex};"></span>
      <span class="mix-target-text">Como chegar nesta cor combinando tintas da {recipeTarget.brand}</span>
    </div>

    {#if recipeLoading}
      <p class="mix-loading">Calculando a mistura…</p>
    {:else if recipeError}
      <p class="mix-error">{recipeError}</p>
    {:else if recipe}
      {#if !recipe.reproducible}
        <p class="mix-warn">
          <Icon name="info" size={14} /> A {recipeTarget.brand} não tem os pigmentos pra chegar exatamente nesta cor — abaixo está a aproximação mais próxima.
        </p>
      {/if}

      <div class="mix-ingredients">
        {#each recipe.ingredients ?? [] as ing}
          <div class="mix-ing">
            <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={34} />
            <div class="mix-ing-info">
              <span class="mix-ing-name">{ing.name}</span>
              {#if ing.code}<span class="mix-ing-code font-mono">{ing.code}</span>{/if}
            </div>
            <span class="mix-ing-pct font-mono">{Math.round(ing.percentage)}%</span>
          </div>
        {/each}
      </div>

      <div class="mix-result">
        <div class="mix-result-swatches">
          <div class="mix-swatch-col">
            <span class="mix-swatch" style="background: {recipeTarget.hex};"></span>
            <span class="mix-swatch-label">alvo</span>
          </div>
          <Icon name="swap" size={18} />
          <div class="mix-swatch-col">
            <span class="mix-swatch" style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB});"></span>
            <span class="mix-swatch-label">mistura</span>
          </div>
        </div>
        <DeltaBadge deltaE={recipe.deltaE} size="sm" />
      </div>

      {#if (recipe.tips ?? []).length > 0}
        <ul class="mix-tips">
          {#each recipe.tips as tip}
            <li>{tip}</li>
          {/each}
        </ul>
      {/if}
    {/if}
  {/if}
</BottomSheet>

<style>
  .roda {
    padding: 0 0 24px;
  }

  .wheel-wrap {
    display: flex;
    justify-content: center;
    padding: 18px 0 8px;
  }

  .wheel {
    position: relative;
    border-radius: 50%;
    touch-action: none;
    box-shadow:
      inset 0 0 0 1px rgba(0, 0, 0, 0.12),
      0 8px 28px rgba(0, 0, 0, 0.16);
  }

  .wheel-marker {
    position: absolute;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    border: 3px solid #fff;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35), 0 2px 6px rgba(0, 0, 0, 0.3);
    pointer-events: none;
  }

  .harmony-dot {
    position: absolute;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    border: 2px solid rgba(255, 255, 255, 0.9);
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.3);
    pointer-events: none;
  }

  .controls {
    padding: 8px 16px 4px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .light-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .light-label {
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--ink-500);
    width: 34px;
  }
  .light-row input[type='range'] {
    flex: 1;
    appearance: none;
    -webkit-appearance: none;
    height: 12px;
    border-radius: 999px;
    background: var(--track);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
    outline: none;
  }
  .light-row input[type='range']::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: var(--ink-900);
    border: 2px solid var(--paper);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
  }
  .light-row input[type='range']::-moz-range-thumb {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: var(--ink-900);
    border: 2px solid var(--paper);
  }

  .selected-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .selected-chip {
    flex: 1;
    height: 56px;
  }
  .selected-hex {
    font-size: 13px;
    letter-spacing: 0.04em;
    color: var(--ink-300);
    flex-shrink: 0;
  }

  .hex-row {
    display: flex;
    gap: 10px;
  }
  .hex-row input {
    flex: 1;
    min-width: 0;
    min-height: 46px;
    padding: 0 14px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    font-size: 16px;
    color: var(--ink-100);
    outline: none;
  }
  .hex-row input:focus {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .block {
    padding: 20px 16px 4px;
  }
  .block-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .block-title {
    font-size: 17px;
    font-weight: 640;
    color: var(--ink-100);
    margin-bottom: 10px;
  }

  .segmented {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 6px;
    margin-bottom: 12px;
  }
  .seg {
    padding: 10px 2px;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-pill);
    background: transparent;
    color: var(--ink-300);
    font-size: 12px;
    font-weight: 600;
  }
  .seg.active {
    background: var(--lacquer);
    border-color: var(--lacquer);
    color: #fff;
  }

  .hint {
    font-size: 13px;
    line-height: 1.5;
    color: var(--ink-300);
    margin-bottom: 14px;
  }
  .hint strong {
    color: var(--ink-100);
    font-weight: 600;
  }
  .muted-hint {
    font-size: 12px;
    margin-top: 8px;
  }

  .harmony-swatches {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }
  .h-swatch {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
  }
  .hs-color {
    width: 60px;
    height: 60px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .hs-role {
    font-size: 11px;
    font-weight: 600;
    color: var(--ink-300);
  }

  .ramp-strip {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 6px;
    margin-bottom: 14px;
  }
  .ramp-strip.naive {
    margin-top: 10px;
    margin-bottom: 0;
  }
  .ramp-cell {
    height: 64px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .ramp-cell.is-base {
    box-shadow: inset 0 0 0 2px var(--lacquer);
  }

  .toggle {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 44px;
    font-size: 13.5px;
    color: var(--ink-300);
  }
  .toggle input {
    width: 20px;
    height: 20px;
    accent-color: var(--lacquer);
  }

  .mini-toggle {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 13px;
    color: var(--ink-300);
  }
  .mini-toggle input {
    width: 18px;
    height: 18px;
    accent-color: var(--lacquer);
  }

  .col-brand {
    width: 100%;
    min-height: 44px;
    padding: 0 12px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    color: var(--ink-100);
    font-size: 16px;
    font-family: var(--font-body);
    outline: none;
  }
  .col-brand:focus {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .step-empty {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }
  .mix-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 9px 14px;
    border: 1px solid var(--lacquer);
    border-radius: var(--radius-pill);
    background: transparent;
    color: var(--lacquer);
    font-size: 14px;
    font-weight: 600;
  }
  .mix-btn:active {
    background: var(--lacquer);
    color: #fff;
  }

  /* Sheet de mistura */
  .mix-target-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }
  .mix-target {
    width: 46px;
    height: 46px;
    border-radius: var(--radius-control);
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
  }
  .mix-target-text {
    font-size: 13.5px;
    line-height: 1.4;
    color: var(--ink-300);
  }
  .mix-loading,
  .mix-error {
    text-align: center;
    padding: 24px 0;
    font-size: 14px;
    color: var(--ink-500);
  }
  .mix-error {
    color: var(--delta-poor);
  }
  .mix-warn {
    display: flex;
    align-items: flex-start;
    gap: 7px;
    font-size: 13px;
    line-height: 1.45;
    color: var(--delta-fair);
    background: color-mix(in srgb, var(--delta-fair) 12%, transparent);
    padding: 11px 13px;
    border-radius: var(--radius-surface);
    margin-bottom: 16px;
  }
  .mix-ingredients {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .mix-ing {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-surface);
    background: var(--ink-900);
  }
  .mix-ing-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .mix-ing-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
  }
  .mix-ing-code {
    font-size: 12px;
    color: var(--ink-500);
  }
  .mix-ing-pct {
    font-size: 17px;
    font-weight: 700;
    color: var(--lacquer);
    flex-shrink: 0;
  }
  .mix-result {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 18px;
    padding-top: 16px;
    border-top: 1px solid var(--ink-800);
  }
  .mix-result-swatches {
    display: flex;
    align-items: center;
    gap: 12px;
    color: var(--ink-500);
  }
  .mix-swatch-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }
  .mix-swatch {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
  }
  .mix-swatch-label {
    font-size: 11px;
    color: var(--ink-500);
  }
  .mix-tips {
    margin-top: 18px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    list-style: none;
  }
  .mix-tips li {
    position: relative;
    padding-left: 18px;
    font-size: 14px;
    line-height: 1.45;
    color: var(--ink-300);
  }
  .mix-tips li::before {
    content: '•';
    position: absolute;
    left: 3px;
    color: var(--lacquer);
  }

  .step-card {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px 0;
    border-bottom: 1px solid var(--ink-800);
  }
  .step-top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .step-swatch {
    width: 42px;
    height: 42px;
    border-radius: var(--radius-control);
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .step-swatch.is-base {
    box-shadow: inset 0 0 0 2px var(--lacquer);
  }
  .step-why {
    font-size: 13px;
    color: var(--ink-300);
  }
  .step-result {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .step-result-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .step-paint {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: normal;
    overflow-wrap: anywhere;
    line-height: 1.25;
  }
  .step-mfr {
    font-size: 12.5px;
    color: var(--ink-500);
  }
  .step-paint.muted {
    font-weight: 400;
    color: var(--ink-500);
    font-size: 13.5px;
  }

  .naive-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    min-width: 0;
  }
  .naive-cell .ramp-cell {
    width: 100%;
  }
  .naive-name {
    font-size: 8.5px;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--ink-300);
    text-align: center;
    line-height: 1.15;
  }
  .naive-hex {
    font-size: 8.5px;
    color: var(--ink-500);
  }
</style>
