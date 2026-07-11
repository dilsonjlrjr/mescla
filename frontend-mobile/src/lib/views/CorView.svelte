<script lang="ts">
  // Cor — "tenho uma cor de referência, que tinta chega perto?"
  // Roda donut de 12 segmentos (o matiz é o volante), centro com hex + matiz,
  // harmonias (Análogas / Complementar / Tríade) e as tintas reais mais
  // próximas do matiz escolhido, ao vivo via findSimilar (WASM).
  import BrandMark from '../components/BrandMark.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import PaintDetailSheet from '../components/PaintDetailSheet.svelte';
  import { switchTab } from '../nav.svelte';
  import { paintById, type Paint } from '../services/catalog';
  import { findSimilar, type SearchResult } from '../services/engine';
  import { appState } from '../appState.svelte';
  import { hslToRgb, rgbToHsl, rgbToHex, hexToRgb, harmony, HARMONY_LABEL, type HarmonyKind } from '../color/theory';

  let hue = $state(165);
  let sat = $state(0.62);
  let light = $state(0.42);
  let hexInput = $state('');
  let results: SearchResult[] = $state([]);
  let searching = $state(false);
  let deltaSheetOpen = $state(false);
  let detailPaint: Paint | null = $state(null);

  // Preset vindo de outra aba ("ver tintas prontas mais próximas")
  $effect(() => {
    if (appState.corPreset) {
      const { r, g, b } = appState.corPreset;
      const h = rgbToHsl({ r, g, b });
      hue = h.h;
      sat = h.s;
      light = h.l;
      appState.corPreset = null;
    }
  });

  let rgb = $derived(hslToRgb({ h: hue, s: sat, l: light }));
  let hex = $derived(rgbToHex(rgb));

  function applyHex() {
    const c = hexToRgb(hexInput);
    if (!c) return;
    const h = rgbToHsl(c);
    hue = h.h;
    sat = h.s;
    light = h.l;
    hexInput = '';
  }

  // Busca ao vivo (debounce 150ms) — o resultado acompanha o giro da roda.
  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    const { r, g, b } = rgb;
    clearTimeout(timer);
    timer = setTimeout(async () => {
      searching = true;
      try {
        results = await findSimilar(r, g, b, 100, 12);
      } catch (e) {
        console.error('findSimilar:', e);
        results = [];
      } finally {
        searching = false;
      }
    }, 150);
  });

  let visible = $derived(results.slice(0, 6));

  function openDetail(res: SearchResult) {
    const p = paintById(res.paintId);
    if (p) detailPaint = p;
  }

  function mesclarFrom(paint: Paint) {
    detailPaint = null;
    appState.pendingMesclarPaint = paint;
    switchTab('mesclar');
  }

  // ── Roda donut de 12 segmentos (SVG) ──
  const SEGS = 12;
  const CX = 130, CY = 130, R_OUT = 122, R_IN = 72;

  function arcPath(startDeg: number, endDeg: number): string {
    const rad = (d: number) => ((d - 90) * Math.PI) / 180;
    const p = (r: number, d: number) => `${CX + r * Math.cos(rad(d))} ${CY + r * Math.sin(rad(d))}`;
    const large = endDeg - startDeg > 180 ? 1 : 0;
    return `M ${p(R_OUT, startDeg)} A ${R_OUT} ${R_OUT} 0 ${large} 1 ${p(R_OUT, endDeg)} L ${p(R_IN, endDeg)} A ${R_IN} ${R_IN} 0 ${large} 0 ${p(R_IN, startDeg)} Z`;
  }

  const GAP = 2.5; // graus de respiro entre segmentos
  let segments = $derived(
    Array.from({ length: SEGS }, (_, i) => {
      const center = i * (360 / SEGS);
      const { r, g, b } = hslToRgb({ h: center, s: 0.82, l: 0.5 });
      return { center, path: arcPath(center - 15 + GAP / 2, center + 15 - GAP / 2), fill: `rgb(${r}, ${g}, ${b})` };
    })
  );

  let activeSeg = $derived(Math.round(hue / (360 / SEGS)) % SEGS);
  let markerPos = $derived.by(() => {
    const d = ((activeSeg * (360 / SEGS)) - 90) * Math.PI / 180;
    const r = (R_OUT + R_IN) / 2;
    return { x: CX + r * Math.cos(d), y: CY + r * Math.sin(d) };
  });

  function pickSeg(i: number) {
    if (navigator.vibrate) navigator.vibrate(6);
    hue = i * (360 / SEGS);
    if (sat < 0.25) sat = 0.62; // sair do neutro ao girar a roda
  }

  // ── Harmonia ──
  const kinds: HarmonyKind[] = ['analogous', 'complementary', 'triad'];
  let harmonyKind: HarmonyKind = $state('analogous');
  let harmonySwatches = $derived(harmony({ h: hue, s: sat, l: light }, harmonyKind));
</script>

<div class="cor">
  <header class="head">
    <BrandMark size={26} />
    <h1 class="head-title font-display">Cor</h1>
  </header>

  <div class="wheel-wrap">
    <svg viewBox="0 0 260 260" class="wheel" role="group" aria-label="Roda de matiz">
      {#each segments as s, i (i)}
        <path
          d={s.path}
          fill={s.fill}
          role="button"
          tabindex="0"
          aria-label="Matiz {s.center} graus"
          onclick={() => pickSeg(i)}
          onkeydown={e => (e.key === 'Enter' || e.key === ' ') && pickSeg(i)}
        />
      {/each}
      <!-- anel de destaque no segmento ativo -->
      <circle cx={markerPos.x} cy={markerPos.y} r="15" fill="rgb({rgb.r}, {rgb.g}, {rgb.b})" stroke="#ffffff" stroke-width="4" pointer-events="none" />
      <text x={CX} y={CY - 4} text-anchor="middle" class="wheel-hex">{hex}</text>
      <text x={CX} y={CY + 18} text-anchor="middle" class="wheel-hue">matiz {Math.round(hue)}°</text>
    </svg>
  </div>

  <div class="fine">
    <label class="fine-row">
      <span class="section-label">Luz</span>
      <input
        type="range"
        min="4"
        max="92"
        value={Math.round(light * 100)}
        aria-label="Luminosidade"
        style="--track: linear-gradient(to right, {rgbToHex(hslToRgb({ h: hue, s: sat, l: 0.08 }))}, {rgbToHex(hslToRgb({ h: hue, s: sat, l: 0.5 }))}, {rgbToHex(hslToRgb({ h: hue, s: sat, l: 0.92 }))});"
        oninput={e => (light = +(e.currentTarget as HTMLInputElement).value / 100)}
      />
    </label>
    <div class="fine-hex">
      <input
        bind:value={hexInput}
        type="text"
        class="font-mono"
        placeholder="#2E7D6B"
        aria-label="Cor em hexadecimal"
        maxlength="7"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        onkeydown={e => e.key === 'Enter' && applyHex()}
      />
      <button class="btn-ghost" onclick={applyHex}>Aplicar</button>
    </div>
  </div>

  <section class="block">
    <p class="section-label">Harmonia</p>
    <div class="pills">
      {#each kinds as k (k)}
        <button class="pill pressable" class:active={harmonyKind === k} onclick={() => (harmonyKind = k)}>
          {HARMONY_LABEL[k]}
        </button>
      {/each}
    </div>
    <div class="harmony-row">
      {#each harmonySwatches as sw (sw.role)}
        <button
          class="harmony-sw pressable"
          style="background: {sw.hex};"
          aria-label="{sw.role} {sw.hex}"
          onclick={() => (hue = sw.h)}
        ></button>
      {/each}
    </div>
  </section>

  <section class="block">
    <div class="near-head">
      <p class="section-label">Tintas perto desse matiz</p>
      <button class="near-what font-mono pressable" onclick={() => (deltaSheetOpen = true)}>ΔE?</button>
    </div>
    {#if visible.length === 0}
      {#if searching}
        <div class="near-skel">
          {#each [0, 1, 2] as k (k)}
            <div class="near-skel-row">
              <div class="skeleton" style="width: 44px; height: 40px;"></div>
              <div class="skeleton" style="flex: 1; height: 14px;"></div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="empty-state">
          <p class="empty-title">Nenhuma tinta próxima</p>
          <p class="empty-hint">Gire a roda ou ajuste a luz.</p>
        </div>
      {/if}
    {:else}
      {#each visible as res, i (res.paintId)}
        <button class="near-row pressable" onclick={() => openDetail(res)}>
          <span class="swatch-flat" style="width: 46px; height: 40px; background: rgb({res.r}, {res.g}, {res.b});"></span>
          <PaintBottle r={res.r} g={res.g} b={res.b} size={44} />
          <span class="near-text">
            <span class="near-name">{res.name}</span>
            <span class="near-meta font-mono">{res.code} · {res.manufacturer}</span>
          </span>
          <span class="near-delta font-mono" class:best={i === 0}>{res.deltaE.toFixed(1)}</span>
        </button>
      {/each}
    {/if}
  </section>
</div>

<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} />
<PaintDetailSheet paint={detailPaint} onClose={() => (detailPaint = null)} onMesclar={mesclarFrom} />

<style>
  .cor {
    padding: 0 16px 16px;
  }

  .head {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 56px;
    margin: 0 -16px;
    padding: 8px 16px;
    background: var(--papel);
    border-bottom: 1px solid var(--hairline);
  }

  .head-title {
    font-size: 20px;
    font-weight: 750;
    color: var(--grafite);
  }

  .wheel-wrap {
    display: flex;
    justify-content: center;
    padding: 18px 0 6px;
  }

  .wheel {
    width: min(280px, 78vw);
    height: auto;
  }

  .wheel path {
    cursor: pointer;
  }

  .wheel-hex {
    font-family: var(--font-mono);
    font-size: 17px;
    font-weight: 600;
    fill: var(--grafite);
  }

  .wheel-hue {
    font-family: var(--font-mono);
    font-size: 11px;
    fill: var(--ink-500);
  }

  .fine {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding-bottom: 6px;
  }

  .fine-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .fine-row .section-label {
    width: 34px;
    flex-shrink: 0;
  }

  .fine-row input[type='range'] {
    flex: 1;
    appearance: none;
    -webkit-appearance: none;
    height: 10px;
    border-radius: var(--radius-pill);
    background: var(--track);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.12);
    outline: none;
  }

  .fine-row input[type='range']::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--grafite);
    border: 3px solid #fff;
    box-shadow: 0 1px 4px rgba(26, 23, 18, 0.3);
    cursor: pointer;
  }

  .fine-row input[type='range']::-moz-range-thumb {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--grafite);
    border: 3px solid #fff;
    cursor: pointer;
  }

  .fine-hex {
    display: flex;
    gap: 10px;
  }

  .fine-hex input {
    flex: 1;
    min-width: 0;
    min-height: 46px;
    padding: 0 14px;
    font-size: 16px;
    text-transform: uppercase;
  }

  .block {
    padding-top: 16px;
  }

  .pills {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }

  .pill {
    min-height: 42px;
    padding: 0 16px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-pill);
    background: var(--papel);
    font-size: 13.5px;
    font-weight: 600;
    color: var(--grafite);
  }

  .pill.active {
    background: var(--grafite);
    border-color: var(--grafite);
    color: var(--papel);
  }

  .harmony-row {
    display: flex;
    gap: 8px;
    margin-top: 12px;
  }

  .harmony-sw {
    flex: 1;
    height: 56px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
  }

  .near-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .near-what {
    font-size: 11px;
    font-weight: 600;
    color: var(--ink-500);
    min-height: 32px;
    padding: 0 6px;
  }

  .near-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 62px;
    padding: 9px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .near-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .near-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .near-meta {
    font-size: 12px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .near-delta {
    flex-shrink: 0;
    font-size: 22px;
    font-weight: 600;
    color: var(--grafite);
  }

  .near-delta.best {
    color: var(--laca);
  }

  .near-skel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 10px 0;
  }

  .near-skel-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
</style>
