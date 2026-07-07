<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import DeltaBadge from './DeltaBadge.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import WheelGuide from './WheelGuide.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
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
    type RampStep,
    type RGB,
  } from '../color/theory';

  // Estado da cor: matiz + saturação vêm da roda, luz vem do slider.
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
  let showPaints = $state(true);
  let hexInput = $state('');

  // Marca-alvo POR PASSO da rampa ('' = tinta mais próxima de qualquer marca) —
  // dá pra montar a rampa misturando marcas (sombra de uma, base de outra…).
  let brands: { id: number; name: string }[] = $state([]);
  let brandByLevel: Record<number, string> = $state({});

  function brandId(name: string): number | null {
    return brands.find(x => x.name === name)?.id ?? null;
  }

  onMount(async () => {
    try {
      const mfrs = (await PaintService.GetManufacturers()) || [];
      brands = mfrs
        .filter(m => m.paintCount > 0)
        .map(m => ({ id: m.id, name: m.name }))
        .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'));
    } catch (e) {
      console.error('GetManufacturers:', e);
    }
  });

  // ---- Modal de mistura (quando a marca não tem a cor pronta) ---------------
  let recipe: any = $state(null);
  let recipeLoading = $state(false);
  let recipeError = $state('');
  let recipeTarget: { hex: string; brand: string } | null = $state(null);

  async function openRecipe(step: RampStep, brand: string) {
    const id = brandId(brand);
    if (id == null) return;
    recipeTarget = { hex: step.hex, brand };
    recipe = null;
    recipeError = '';
    recipeLoading = true;
    try {
      recipe = await PaintService.SuggestRecipeForColor(step.rgb.r, step.rgb.g, step.rgb.b, id);
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

  // ---- Roda: geometria -----------------------------------------------------
  const SIZE = 300;
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

  function pickFromEvent(e: PointerEvent) {
    const rect = wheelEl.getBoundingClientRect();
    const dx = e.clientX - rect.left - R;
    const dy = e.clientY - rect.top - R;
    const dist = Math.sqrt(dx * dx + dy * dy);
    sat = Math.min(1, dist / R);
    hue = (((Math.atan2(dx, -dy) * 180) / Math.PI) % 360 + 360) % 360;
  }

  function onDown(e: PointerEvent) {
    dragging = true;
    wheelEl.setPointerCapture(e.pointerId);
    pickFromEvent(e);
  }
  function onMove(e: PointerEvent) {
    if (dragging) pickFromEvent(e);
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

  function pickHarmony(h: number) {
    hue = h;
  }

  // ---- Tintas reais por passo da rampa -------------------------------------
  // Busca ampla por passo (cacheada); a escolha por marca é derivada sem refazer
  // a busca. Debounce pra não disparar a cada arrasto na roda.
  let rawByLevel: Record<number, any[]> = $state({});
  let timer: ReturnType<typeof setTimeout>;

  $effect(() => {
    const steps = ramp;
    const enabled = showPaints;
    clearTimeout(timer);
    if (!enabled) {
      rawByLevel = {};
      return;
    }
    timer = setTimeout(async () => {
      const entries = await Promise.all(
        steps.map(async s => {
          try {
            const res = (await PaintService.FindSimilar(s.rgb.r, s.rgb.g, s.rgb.b, 100, 40)) || [];
            return [s.level, res] as const;
          } catch {
            return [s.level, []] as const;
          }
        }),
      );
      rawByLevel = Object.fromEntries(entries);
    }, 220);
  });

  // Tinta escolhida por passo = marca do passo (se houver) ou a mais próxima.
  let paintByLevel = $derived.by(() => {
    const out: Record<number, any> = {};
    for (const s of ramp) {
      const res = rawByLevel[s.level] || [];
      const brand = brandByLevel[s.level] || '';
      out[s.level] = (brand ? res.find(x => x.manufacturer === brand) : res[0]) ?? null;
    }
    return out;
  });

  function setBrand(level: number, value: string) {
    brandByLevel = { ...brandByLevel, [level]: value };
  }

  function rgbCss(c: RGB) {
    return `rgb(${c.r}, ${c.g}, ${c.b})`;
  }

  const harmonyKinds: HarmonyKind[] = ['complementary', 'analogous', 'triad', 'split'];
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Roda cromática</h1>
    <p class="page-subtitle">
      Escolha uma cor e aprenda a clarear e escurecer do jeito certo — deslocando o matiz, não só
      jogando branco e preto
    </p>
    <div class="page-divider"></div>
  </div>

  <WheelGuide baseRgb={baseRgb} />

  <div class="wheel-layout">
    <!-- Coluna esquerda: a roda -->
    <div class="panel p-5 animate-rise" style="animation-delay: 60ms;">
      <div class="wheel-wrap">
        <div
          class="wheel"
          bind:this={wheelEl}
          role="slider"
          tabindex="0"
          aria-label="Roda cromática — matiz e saturação"
          aria-valuenow={Math.round(hue)}
          aria-valuemin="0"
          aria-valuemax="360"
          style="width: {SIZE}px; height: {SIZE}px; background: {wheelBg};"
          onpointerdown={onDown}
          onpointermove={onMove}
          onpointerup={onUp}
        >
          <!-- marcadores de harmonia -->
          {#each harmonySwatches as sw}
            {@const p = polar(sw.h, Math.max(sat, 0.5))}
            <span class="harmony-dot" style="left: {p.x}px; top: {p.y}px; background: {sw.hex};"></span>
          {/each}
          <!-- marcador principal -->
          <span class="wheel-marker" style="left: {marker.x}px; top: {marker.y}px; background: {baseHex};"></span>
        </div>
      </div>

      <!-- Slider de luz -->
      <div class="light-row">
        <span class="light-label">Luz</span>
        <input
          type="range"
          min="0"
          max="100"
          value={Math.round(light * 100)}
          class="light-slider"
          style="--track: linear-gradient(to right, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.08 }))}, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.5 }))}, {rgbCss(hslToRgb({ h: hue, s: sat, l: 0.95 }))});"
          oninput={e => (light = +(e.currentTarget as HTMLInputElement).value / 100)}
        />
      </div>

      <!-- Cor selecionada -->
      <div class="selected-chip" style="background: {baseHex};">
        <span class="selected-hex font-mono">{baseHex}</span>
      </div>
      <div class="hex-row">
        <input
          bind:value={hexInput}
          type="text"
          class="hex-input font-mono"
          placeholder="#RRGGBB"
          maxlength="7"
          spellcheck="false"
          autocomplete="off"
          onkeydown={e => e.key === 'Enter' && applyHex()}
        />
        <button class="hex-btn" onclick={applyHex}>Aplicar</button>
      </div>
    </div>

    <!-- Coluna direita: harmonias -->
    <div class="panel p-5 animate-rise" style="animation-delay: 120ms;">
      <h3 class="section-title">Harmonias</h3>
      <div class="segmented">
        {#each harmonyKinds as k}
          <button class="seg" class:active={harmonyKind === k} onclick={() => (harmonyKind = k)}>
            {HARMONY_LABEL[k]}
          </button>
        {/each}
      </div>

      <p class="harmony-hint">{HARMONY_HINT[harmonyKind]}</p>

      <div class="harmony-swatches">
        {#each harmonySwatches as sw}
          <button class="harmony-swatch" onclick={() => pickHarmony(sw.h)} title="Usar como base">
            <span class="hs-color" style="background: {sw.hex};"></span>
            <span class="hs-role">{sw.role}</span>
            <span class="hs-hex font-mono">{sw.hex}</span>
          </button>
        {/each}
      </div>
    </div>
  </div>

  <!-- Rampa: clarear e escurecer do jeito certo -->
  <div class="panel p-5 ramp-panel animate-rise" style="animation-delay: 180ms;">
    <div class="ramp-head">
      <div>
        <h3 class="section-title" style="margin-bottom: 4px;">Clarear e escurecer do jeito certo</h3>
        <p class="ramp-sub">
          Da sombra ao brilho. Repare: o matiz caminha pro <strong>azul</strong> na sombra e pro
          <strong>amarelo</strong> na luz — é isso que mantém a cor viva.
        </p>
      </div>
      <div class="ramp-toggles">
        <label class="toggle"><input type="checkbox" bind:checked={showPaints} /> Tintas reais</label>
        <label class="toggle"><input type="checkbox" bind:checked={showNaive} /> Comparar com o jeito comum</label>
      </div>
    </div>

    <div class="ramp-strip">
      {#each ramp as step (step.level)}
        <div class="ramp-step" class:is-base={step.kind === 'base'}>
          <div class="ramp-swatch" style="background: {step.hex};">
            {#if step.kind === 'base'}<span class="base-tag">base</span>{/if}
          </div>
          <span class="ramp-why">{step.why}</span>
          {#if showPaints}
            {@const brand = brandByLevel[step.level] ?? ''}
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
            {@const paint = paintByLevel[step.level]}
            {#if paint}
              <div class="ramp-paint">
                <div class="ramp-paint-top">
                  <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={24} />
                  <span class="ramp-paint-name" title={paint.name}>{paint.name}</span>
                </div>
                {#if !brand}<span class="ramp-paint-mfr">{paint.manufacturer}</span>{/if}
                <DeltaBadge deltaE={paint.deltaE} size="sm" />
              </div>
            {:else}
              <div class="ramp-paint muted">
                <span>Sem tinta {brand ? `da ${brand}` : ''} próxima</span>
                {#if brand}
                  <button class="mix-btn" onclick={() => openRecipe(step, brand)}>
                    <Icon name="flask" size={13} /> Ver mistura
                  </button>
                {/if}
              </div>
            {/if}
          {/if}
        </div>
      {/each}
    </div>

    {#if showNaive}
      <div class="naive-block">
        <p class="naive-label">
          <Icon name="info" size={14} /> O jeito comum (só branco e preto): mesma cor, mas repare como
          desbota e "suja" — perde a vida.
        </p>
        <div class="naive-strip">
          {#each naive as c, i}
            <div class="naive-cell">
              <div class="naive-swatch" style="background: {rgbCss(c)};"></div>
              <span class="naive-name">{colorName(c)}</span>
              <span class="naive-hex font-mono">{rgbToHex(c)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<svelte:window onkeydown={e => recipeTarget && e.key === 'Escape' && closeRecipe()} />

{#if recipeTarget}
  <div
    class="modal-scrim"
    onclick={closeRecipe}
    onkeydown={e => e.key === 'Escape' && closeRecipe()}
    role="presentation"
  >
    <div
      class="modal"
      onclick={e => e.stopPropagation()}
      onkeydown={e => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-label="Mistura equivalente"
      tabindex="-1"
    >
      <div class="modal-head">
        <div class="modal-title-wrap">
          <span class="modal-target" style="background: {recipeTarget.hex};"></span>
          <div>
            <h3 class="modal-title">Mistura na {recipeTarget.brand}</h3>
            <p class="modal-sub">Como chegar nesta cor combinando tintas da marca</p>
          </div>
        </div>
        <button class="modal-close" onclick={closeRecipe} aria-label="Fechar"><Icon name="close" size={18} /></button>
      </div>

      <div class="modal-body">
        {#if recipeLoading}
          <p class="modal-loading">Calculando a mistura…</p>
        {:else if recipeError}
          <p class="modal-error">{recipeError}</p>
        {:else if recipe}
          {#if !recipe.reproducible}
            <p class="modal-warn">
              <Icon name="info" size={14} /> A {recipeTarget.brand} não tem os pigmentos pra chegar exatamente nesta cor — abaixo está a aproximação mais próxima possível.
            </p>
          {/if}

          <div class="mix-ingredients">
            {#each recipe.ingredients ?? [] as ing}
              <div class="mix-ing">
                <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={30} />
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
              <Icon name="swap" size={16} />
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
      </div>
    </div>
  </div>
{/if}

<style>
  .wheel-layout {
    display: grid;
    grid-template-columns: 380px 1fr;
    gap: 24px;
    margin-bottom: 24px;
  }

  .wheel-wrap {
    display: flex;
    justify-content: center;
    margin-bottom: 20px;
  }

  .wheel {
    position: relative;
    border-radius: 50%;
    cursor: crosshair;
    touch-action: none;
    box-shadow:
      inset 0 0 0 1px rgba(0, 0, 0, 0.12),
      0 8px 28px rgba(0, 0, 0, 0.14);
  }

  .wheel-marker {
    position: absolute;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    border: 3px solid #fff;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35), 0 2px 6px rgba(0, 0, 0, 0.3);
    pointer-events: none;
  }

  .harmony-dot {
    position: absolute;
    width: 15px;
    height: 15px;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    border: 2px solid rgba(255, 255, 255, 0.9);
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.3);
    pointer-events: none;
  }

  .light-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 18px;
  }

  .light-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--ink-400);
    width: 34px;
  }

  .light-slider {
    flex: 1;
    appearance: none;
    -webkit-appearance: none;
    height: 12px;
    border-radius: 999px;
    background: var(--track);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
    outline: none;
  }

  .light-slider::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--ink-900);
    border: 2px solid var(--paper);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
    cursor: pointer;
  }

  .selected-chip {
    height: 64px;
    border-radius: 12px;
    display: flex;
    align-items: flex-end;
    padding: 8px 10px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
    margin-bottom: 10px;
  }

  .selected-hex {
    font-size: 12px;
    padding: 3px 8px;
    border-radius: 6px;
    background: rgba(26, 23, 18, 0.5);
    color: rgba(255, 255, 255, 0.95);
  }

  .hex-row {
    display: flex;
    gap: 8px;
  }

  .hex-input {
    flex: 1;
    min-width: 0;
    height: 40px;
    padding: 0 12px;
    border: 1px solid var(--ink-700);
    border-radius: 9px;
    background: var(--ink-900);
    color: var(--ink-100);
    font-size: 14px;
    outline: none;
  }
  .hex-input:focus {
    border-color: var(--lacquer);
  }

  .hex-btn {
    padding: 0 16px;
    border: none;
    border-radius: 9px;
    background: var(--lacquer);
    color: #fff;
    font-weight: 600;
    font-size: 13px;
    cursor: pointer;
  }
  .hex-btn:hover {
    background: var(--lacquer-deep);
  }

  .section-title {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    margin-bottom: 12px;
  }

  .segmented {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 6px;
    margin-bottom: 12px;
  }

  .seg {
    padding: 8px 4px;
    border: 1px solid var(--ink-700);
    border-radius: 8px;
    background: transparent;
    color: var(--ink-400);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .seg:hover {
    border-color: var(--ink-600);
    color: var(--ink-100);
  }
  .seg.active {
    background: var(--lacquer);
    border-color: var(--lacquer);
    color: #fff;
  }

  .harmony-hint {
    font-size: 13px;
    line-height: 1.5;
    color: var(--ink-400);
    margin-bottom: 16px;
  }

  .harmony-swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .harmony-swatch {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
    padding: 0;
    border: none;
    background: transparent;
    cursor: pointer;
  }

  .hs-color {
    width: 62px;
    height: 62px;
    border-radius: 10px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
    transition: transform 0.15s ease;
  }
  .harmony-swatch:hover .hs-color {
    transform: translateY(-2px);
  }

  .hs-role {
    font-size: 11px;
    font-weight: 600;
    color: var(--ink-300);
  }
  .hs-hex {
    font-size: 10px;
    color: var(--ink-500);
  }

  /* Rampa */
  .ramp-head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 18px;
    flex-wrap: wrap;
  }

  .ramp-sub {
    font-size: 13px;
    line-height: 1.5;
    color: var(--ink-400);
    max-width: 460px;
  }
  .ramp-sub strong {
    color: var(--ink-100);
    font-weight: 600;
  }

  .ramp-toggles {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .toggle {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--ink-300);
    cursor: pointer;
    white-space: nowrap;
  }
  .toggle input {
    width: 16px;
    height: 16px;
    accent-color: var(--lacquer);
  }

  .ramp-strip {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 12px;
  }

  .ramp-step {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0; /* deixa a coluna 1fr encolher; sem isso o nome longo da tinta estoura o grid */
  }

  .ramp-swatch {
    height: 92px;
    border-radius: 10px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
    position: relative;
  }
  .ramp-step.is-base .ramp-swatch {
    box-shadow: inset 0 0 0 2px var(--lacquer), 0 0 0 2px rgba(232, 84, 44, 0.25);
  }

  .base-tag {
    position: absolute;
    bottom: 6px;
    left: 50%;
    transform: translateX(-50%);
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    padding: 2px 8px;
    border-radius: 5px;
    background: var(--lacquer);
    color: #fff;
  }

  .ramp-why {
    font-size: 10.5px;
    line-height: 1.35;
    color: var(--ink-500);
    text-align: center;
    min-height: 28px;
  }

  .ramp-paint {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
    border-radius: 8px;
    background: var(--ink-850);
    border: 1px solid var(--ink-700);
    min-height: 40px;
  }
  .ramp-paint.muted {
    align-items: center;
    justify-content: center;
    text-align: center;
    color: var(--ink-500);
    font-size: 11px;
    line-height: 1.35;
  }

  .ramp-paint-top {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .ramp-paint-name {
    flex: 1;
    min-width: 0;
    font-size: 11.5px;
    font-weight: 600;
    color: var(--ink-100);
    line-height: 1.25;
    /* texto sempre dentro: quebra em vez de estourar/cortar */
    white-space: normal;
    overflow-wrap: anywhere;
  }
  .ramp-paint-mfr {
    font-size: 10.5px;
    font-weight: 500;
    color: var(--ink-400);
    white-space: normal;
    overflow-wrap: anywhere;
  }
  /* badge nunca estoura a coluna: quebra o texto se preciso */
  .ramp-paint :global(.delta-badge) {
    max-width: 100%;
    white-space: normal;
    text-align: center;
  }

  .col-brand {
    width: 100%;
    min-width: 0;
    padding: 6px 8px;
    border: 1px solid var(--ink-700);
    border-radius: 8px;
    background: var(--ink-900);
    color: var(--ink-100);
    font-size: 12px;
    font-family: var(--font-body);
    cursor: pointer;
    outline: none;
  }
  .col-brand:focus {
    border-color: var(--lacquer);
  }

  .naive-block {
    margin-top: 22px;
    padding-top: 18px;
    border-top: 1px dashed var(--ink-700);
  }

  .naive-label {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 12.5px;
    color: var(--ink-400);
    margin-bottom: 10px;
  }

  .naive-strip {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 12px;
  }

  .naive-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
    min-width: 0;
  }
  .naive-swatch {
    width: 100%;
    height: 48px;
    border-radius: 8px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .naive-name {
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--ink-300);
    text-align: center;
    line-height: 1.2;
  }
  .naive-hex {
    font-size: 10px;
    color: var(--ink-500);
  }

  .mix-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    margin-top: 8px;
    padding: 6px 12px;
    border: 1px solid var(--lacquer);
    border-radius: 8px;
    background: transparent;
    color: var(--lacquer-deep);
    font-size: 12px;
    font-weight: 600;
    font-family: var(--font-body);
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .mix-btn:hover {
    background: var(--lacquer);
    color: #fff;
  }

  /* Modal */
  .modal-scrim {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgba(26, 23, 18, 0.55);
    backdrop-filter: blur(2px);
  }
  .modal {
    width: 100%;
    max-width: 440px;
    max-height: 85vh;
    overflow-y: auto;
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    border-radius: 16px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
  }
  .modal-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    padding: 18px 18px 14px;
    border-bottom: 1px solid var(--ink-800);
  }
  .modal-title-wrap {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .modal-target {
    width: 42px;
    height: 42px;
    border-radius: 10px;
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
  }
  .modal-title {
    font-family: var(--font-display);
    font-size: 16px;
    font-weight: 600;
    color: var(--ink-100);
  }
  .modal-sub {
    font-size: 12px;
    color: var(--ink-500);
  }
  .modal-close {
    border: none;
    background: transparent;
    color: var(--ink-500);
    cursor: pointer;
    padding: 4px;
    flex-shrink: 0;
  }
  .modal-close:hover {
    color: var(--ink-100);
  }
  .modal-body {
    padding: 16px 18px 20px;
  }
  .modal-loading,
  .modal-error {
    text-align: center;
    padding: 20px 0;
    font-size: 14px;
    color: var(--ink-400);
  }
  .modal-error {
    color: var(--delta-poor);
  }
  .modal-warn {
    display: flex;
    align-items: flex-start;
    gap: 7px;
    font-size: 12.5px;
    line-height: 1.45;
    color: var(--delta-fair);
    background: color-mix(in srgb, var(--delta-fair) 10%, transparent);
    padding: 10px 12px;
    border-radius: 9px;
    margin-bottom: 14px;
  }
  .mix-ingredients {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .mix-ing {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border: 1px solid var(--ink-700);
    border-radius: 10px;
    background: var(--ink-850);
  }
  .mix-ing-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .mix-ing-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--ink-100);
  }
  .mix-ing-code {
    font-size: 11px;
    color: var(--ink-500);
  }
  .mix-ing-pct {
    font-size: 15px;
    font-weight: 700;
    color: var(--lacquer-deep);
    flex-shrink: 0;
  }
  .mix-result {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--ink-800);
  }
  .mix-result-swatches {
    display: flex;
    align-items: center;
    gap: 12px;
    color: var(--ink-400);
  }
  .mix-swatch-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }
  .mix-swatch {
    width: 40px;
    height: 40px;
    border-radius: 9px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
  }
  .mix-swatch-label {
    font-size: 10px;
    color: var(--ink-500);
  }
  .mix-tips {
    margin-top: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    list-style: none;
  }
  .mix-tips li {
    position: relative;
    padding-left: 16px;
    font-size: 13px;
    line-height: 1.45;
    color: var(--ink-300);
  }
  .mix-tips li::before {
    content: '•';
    position: absolute;
    left: 2px;
    color: var(--lacquer);
  }

  @media (max-width: 900px) {
    .wheel-layout {
      grid-template-columns: 1fr;
    }
  }
</style>
