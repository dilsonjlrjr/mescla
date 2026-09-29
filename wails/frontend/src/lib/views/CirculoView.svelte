<script lang="ts">
  // rf-21 — T5 Círculo cromático. À esquerda, o disco giratório fiel às fotos
  // de referência (components/CirculoCromatico.svelte); à direita, o que o
  // disco ensina, calculado para a cor-base: a harmonia escolhida (com a tinta
  // mais próxima na estante), a escala monocromática, os caminhos para
  // escurecer, clarear e baixar a saturação, e o valor na escala de cinza.
  //
  // A cor-base é um matiz do anel (giro em passos de 30°) ou a cor do usuário
  // (hex ou uma das "Minhas tintas"), que cai no ponto exato da roda e faz a
  // harmonia herdar a saturação e o valor dela.
  import { untrack } from 'svelte';
  import { fly, scale } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { cubicOut, backOut } from 'svelte/easing';
  import Header from '../components/Header.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import CirculoCromatico from '../components/CirculoCromatico.svelte';
  import GuiaCirculo, { type VerNoDisco } from '../components/GuiaCirculo.svelte';
  import MisturaHarmonia from '../components/MisturaHarmonia.svelte';
  import {
    WHEEL,
    SCHEMES,
    STEP,
    chromaOf,
    darkenPaths,
    deltaE00,
    desaturatePaths,
    grayAt,
    grayOfSameValue,
    grayStepOf,
    harmonyOf,
    hexToRgb,
    inkOn,
    isWarm,
    lightenPaths,
    monoBands,
    nearestIndex,
    norm360,
    ringColorAt,
    rgbToHex,
    rgbToHsl,
    valueOf,
    wheelDegOf,
    withSaturation,
    type HarmonySwatch,
    type Path,
    type RGB,
    type SchemeId,
  } from '../color/circulo';
  import { hintKey, hueKey, schemeKey } from '../circuloTexto';
  import { deltaIsGood } from '../ui';
  import { stock } from '../services/stock.svelte';
  import { allPaints } from '../services/catalog';
  import { catalogRev } from '../services/catalogRev.svelte';
  import { t, decimal, type DictKey } from '../i18n.svelte';

  // ── estado ────────────────────────────────────────────────────────────────
  const STORE = 'mescla-circulo';

  function restore(): { angle: number; scheme: SchemeId | 'todas' } {
    try {
      const raw = JSON.parse(localStorage.getItem(STORE) ?? 'null');
      if (raw && typeof raw.angle === 'number' && typeof raw.scheme === 'string') return raw;
    } catch {
      // armazenamento indisponível: começa na posição das fotos
    }
    return { angle: 30, scheme: 'complementar' };
  }

  const saved = restore();
  let angle = $state(saved.angle);
  let live = $state(saved.angle);
  let scheme: SchemeId | 'todas' = $state(saved.scheme);
  let freeSpin = $state(false);
  let grayscale = $state(false);
  let satPct = $state(100);
  let custom = $state<{ rgb: RGB; deg: number; label: string } | null>(null);
  let tab: 'escurecer' | 'clarear' | 'saturacao' | 'valor' = $state('escurecer');
  let level = $state(2);
  let hexDraft = $state('');
  let hexError = $state('');
  let pickerOpen = $state(false);
  let pickerQuery = $state('');

  $effect(() => {
    const snapshot = { angle, scheme };
    try {
      localStorage.setItem(STORE, JSON.stringify(snapshot));
    } catch {
      // conveniência apenas
    }
  });

  // Girar o disco para longe da cor do usuário devolve a base ao anel.
  // Fora do giro livre, o disco volta a encaixar num matiz do anel.
  $effect(() => {
    const a = angle;
    if (custom && Math.abs(((a - custom.deg + 540) % 360) - 180) > 0.5) {
      custom = null;
      if (!freeSpin) angle = Math.round(a / STEP) * STEP;
    }
  });

  // ── cor-base ──────────────────────────────────────────────────────────────
  let baseWheel = $derived(custom ? custom.deg : freeSpin ? norm360(angle) : norm360(Math.round(angle / STEP) * STEP));
  let rawBase = $derived(custom ? custom.rgb : ringColorAt(baseWheel));
  let base = $derived(satPct === 100 ? rawBase : withSaturation(rawBase, rgbToHsl(rawBase).s * (satPct / 100)));
  let fromRing = $derived(!custom && satPct === 100 && baseWheel % STEP === 0);
  let baseHex = $derived(rgbToHex(base));
  let baseHue = $derived(WHEEL[nearestIndex(baseWheel)]);
  let liveHue = $derived(WHEEL[nearestIndex(live)]);
  let baseValue = $derived(valueOf(base));
  let baseStep = $derived(grayStepOf(base));
  let baseChroma = $derived(chromaOf(base));

  function orderLabel(o: string): string {
    return o === 'primaria' ? t('cwPrimary') : o === 'secundaria' ? t('cwSecondary') : t('cwTertiary');
  }

  // ── harmonia ──────────────────────────────────────────────────────────────
  let activeScheme = $derived(scheme === 'todas' ? null : SCHEMES.find(s => s.id === scheme)!);
  let swatches = $derived(activeScheme ? harmonyOf(activeScheme, baseWheel, base, fromRing) : []);
  let allRows = $derived(
    SCHEMES.filter(s => s.id !== 'monocromatico').map(s => ({ s, sw: harmonyOf(s, baseWheel, base, fromRing) }))
  );
  let bands = $derived(monoBands(base));

  function offsetLabel(o: number): string {
    if (o === 0) return t('cwOffsetBase');
    return `${o > 0 ? '+' : '−'}${Math.abs(o)}°`;
  }

  // Tinta mais próxima: estante se houver, senão o catálogo.
  let pool = $derived.by(() => {
    void catalogRev.n;
    if (stock.paints.length > 0) return { kind: 'stock' as const, paints: stock.paints };
    return { kind: 'cat' as const, paints: allPaints() };
  });

  function nearest(rgb: RGB): { name: string; d: number; r: number; g: number; b: number } | null {
    let best: { name: string; d: number; r: number; g: number; b: number } | null = null;
    for (const p of pool.paints) {
      if (p.ignoreInMix) continue;
      const d = deltaE00(rgb, p);
      if (!best || d < best.d) best = { name: `${p.name}${p.code ? ` · ${p.code}` : ''}`, d, r: p.r, g: p.g, b: p.b };
    }
    return best;
  }

  let nearestBySwatch = $derived(swatches.map(sw => nearest(sw.rgb)));

  // rf-24: guia do círculo e receita de mistura (folhas). A mistura fecha ao
  // girar o disco ou trocar a harmonia (RN5); o pedido velho morre junto,
  // porque a folha desmonta.
  let guiaAberto = $state(false);
  let mistura = $state<{ r: number; g: number; b: number; nome: string } | null>(null);

  $effect(() => {
    void baseHex;
    void scheme;
    untrack(() => (mistura = null));
  });

  function verNoDisco(v: VerNoDisco) {
    if (v.scheme) scheme = v.scheme;
    if (v.tab) tab = v.tab;
  }

  /** RN1: sem tinta pronta = não há tinta mais próxima, ou ΔE00 >= 2. */
  function semTintaPronta(near: { d: number } | null): boolean {
    return !near || !deltaIsGood(near.d);
  }

  function useAsBase(sw: HarmonySwatch) {
    if (fromRing) {
      angle = sw.wheel;
      return;
    }
    custom = { rgb: sw.rgb, deg: sw.wheel, label: t(hueKey(sw.hue.id)) };
    angle = sw.wheel;
  }

  // ── cor do usuário ────────────────────────────────────────────────────────
  function applyColor(rgb: RGB, label: string): boolean {
    const deg = wheelDegOf(rgb);
    if (deg == null) {
      hexError = t('cwNeutral');
      return false;
    }
    hexError = '';
    satPct = 100;
    custom = { rgb, deg, label };
    angle = deg;
    return true;
  }

  function applyHex() {
    const v = hexDraft.trim();
    const rgb = hexToRgb(v.startsWith('#') ? v : `#${v}`);
    if (!rgb) {
      hexError = t('cwInvalidHex');
      return;
    }
    if (applyColor(rgb, rgbToHex(rgb))) hexDraft = '';
  }

  function backToRing() {
    custom = null;
    satPct = 100;
    angle = Math.round(angle / STEP) * STEP;
  }

  let pickerList = $derived.by(() => {
    const q = pickerQuery.trim().toLowerCase();
    const list = stock.paints.filter(
      p => !q || p.name.toLowerCase().includes(q) || p.code.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q)
    );
    return list.slice(0, 60);
  });

  function pickPaint(p: { r: number; g: number; b: number; name: string }) {
    if (applyColor({ r: p.r, g: p.g, b: p.b }, p.name)) pickerOpen = false;
  }

  // ── caminhos ──────────────────────────────────────────────────────────────
  interface PathView {
    path: Path;
    title: DictKey;
    why: DictKey;
    good: boolean;
  }

  let darkRows: PathView[] = $derived.by(() => {
    const [preto, comp, matiz] = darkenPaths(base, baseWheel);
    return [
      { path: preto, title: 'pathPreto', why: 'pathPretoWhy', good: false },
      { path: comp, title: 'pathComp', why: 'pathCompWhy', good: true },
      { path: matiz, title: 'pathMatizDark', why: 'pathMatizDarkWhy', good: true },
    ];
  });

  let lightRows: PathView[] = $derived.by(() => {
    const [branco, vizinha, matiz] = lightenPaths(base, baseWheel);
    return [
      { path: branco, title: 'pathBranco', why: 'pathBrancoWhy', good: false },
      { path: vizinha, title: 'pathVizinha', why: 'pathVizinhaWhy', good: true },
      { path: matiz, title: 'pathMatizLight', why: 'pathMatizLightWhy', good: true },
    ];
  });

  let satRows: PathView[] = $derived.by(() => {
    const [cinza, comp] = desaturatePaths(base, baseWheel);
    return [
      { path: cinza, title: 'pathCinza', why: 'pathCinzaWhy', good: true },
      { path: comp, title: 'pathCompSat', why: 'pathCompSatWhy', good: true },
    ];
  });

  let rows = $derived(tab === 'escurecer' ? darkRows : tab === 'clarear' ? lightRows : satRows);

  function chromaKept(rgb: RGB): number {
    return baseChroma < 1 ? 0 : Math.round((chromaOf(rgb) / baseChroma) * 100);
  }

  const TABS = [
    { id: 'escurecer', key: 'cwTabDarken', icon: 'ph-moon' },
    { id: 'clarear', key: 'cwTabLighten', icon: 'ph-sun' },
    { id: 'saturacao', key: 'cwTabSat', icon: 'ph-drop-half' },
    { id: 'valor', key: 'cwTabValue', icon: 'ph-circle-half' },
  ] as const;
  let tabIndex = $derived(TABS.findIndex(x => x.id === tab));

  const GRAY_STEPS = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];

  function css(rgb: RGB): string {
    return `rgb(${rgb.r}, ${rgb.g}, ${rgb.b})`;
  }

  function ink(rgb: RGB): string {
    return inkOn(rgb) === 'dark' ? 'var(--color-bg)' : 'var(--color-text)';
  }

  const DASH_SAMPLE: Record<string, string> = { solid: '', dash: '7 5', longdash: '11 5', dot: '1.5 5' };
</script>

<div class="t5">
  <Header kicker={t('cwKicker')} title={t('cwTitle')}>
    {#snippet actions()}
      <button class="t5-toggle pressable" onclick={() => (guiaAberto = true)}>
        <i class="ph ph-graduation-cap" style="font-size: 18px;"></i>{t('cgOpen')}
      </button>
      <button class="t5-toggle pressable" class:on={freeSpin} onclick={() => (freeSpin = !freeSpin)} aria-pressed={freeSpin}>
        <i class="ph ph-arrows-clockwise" style="font-size: 18px;"></i>{t('cwFreeSpin')}
      </button>
      <button class="t5-toggle pressable" class:on={grayscale} onclick={() => (grayscale = !grayscale)} aria-pressed={grayscale}>
        <i class="ph {grayscale ? 'ph-palette' : 'ph-circle-half'}" style="font-size: 18px;"></i>{grayscale ? t('cwSeeColors') : t('cwSeeValues')}
      </button>
    {/snippet}
  </Header>

  <div class="t5-body">
    <!-- ── coluna do disco ── -->
    <section class="t5-wheelcol">
      <div class="t5-livebar">
        <span class="t5-livedot" style="background: {custom ? baseHex : liveHue.hex};"></span>
        {#key custom ? custom.label : liveHue.id}
          <span class="t5-livename" in:fly={{ y: 10, duration: 260, easing: cubicOut }}>
            {custom ? custom.label : t(hueKey(liveHue.id))}
          </span>
        {/key}
        <span class="t5-livemeta">
          {isWarm(custom ? custom.deg : live) ? t('cwWarm') : t('cwCool')} · {orderLabel(liveHue.order)}
        </span>
      </div>

      <div class="t5-wheelbox">
        <div class="t5-wheel">
          <CirculoCromatico
            bind:angle
            bind:live
            {scheme}
            snap={!freeSpin && !custom}
            marker={custom ? { deg: custom.deg, hex: rgbToHex(custom.rgb) } : null}
            {grayscale}
            valueStep={tab === 'valor' ? baseStep : null}
            onpick={deg => {
              custom = null;
              angle = deg;
            }}
          />
        </div>
      </div>

      <p class="t5-hint">{t('cwDragHint')}</p>

      <div class="t5-chips" role="radiogroup" aria-label={t('cwHarmony')}>
        {#each SCHEMES as s (s.id)}
          <button
            class="t5-chip pressable"
            class:on={scheme === s.id}
            role="radio"
            aria-checked={scheme === s.id}
            style="--c: var({s.token});"
            onclick={() => (scheme = s.id)}
          >
            <svg width="26" height="10" viewBox="0 0 26 10" aria-hidden="true">
              <line x1="2" y1="5" x2="24" y2="5" stroke="var(--c)" stroke-width="3" stroke-linecap="round" stroke-dasharray={DASH_SAMPLE[s.stroke]} />
            </svg>
            {t(schemeKey(s.id))}
          </button>
        {/each}
        <button class="t5-chip pressable" class:on={scheme === 'todas'} role="radio" aria-checked={scheme === 'todas'} onclick={() => (scheme = 'todas')}>
          <i class="ph ph-shapes" style="font-size: 16px;"></i>{t('schAll')}
        </button>
      </div>
    </section>

    <!-- ── painel ── -->
    <section class="t5-panel">
      <!-- cor-base -->
      <div class="t5-card t5-basecard">
        <div class="t5-baseswatch" style="background: {baseHex};">
          {#key baseHex}
            <span class="t5-ripple" style="background: {baseHex};" in:scale={{ start: 0.2, duration: 520, easing: cubicOut }}></span>
          {/key}
          <span class="t5-basehex font-mono" style="color: {ink(base)};">{baseHex}</span>
        </div>
        <div class="t5-baseinfo">
          <span class="section-label">{custom ? t('cwYourColor') : t('cwBase')}</span>
          {#key custom ? custom.label : baseHue.id}
            <span class="t5-basename" in:fly={{ x: -12, duration: 300, easing: cubicOut }}>
              {custom ? custom.label : t(hueKey(baseHue.id))}
            </span>
          {/key}
          <div class="t5-tags">
            <span class="t5-tag" class:warm={isWarm(baseWheel)}>{isWarm(baseWheel) ? t('cwWarm') : t('cwCool')}</span>
            {#if !custom}<span class="t5-tag">{orderLabel(baseHue.order)}</span>{/if}
            <span class="t5-tag">{t('cwValue')} {decimal(baseValue, 0)}</span>
            <span class="t5-tag">{t('cwValueStep', { p: baseStep })}</span>
          </div>
          <div class="t5-inputs">
            <input
              class="t5-hex font-mono"
              placeholder={t('cwHexPh')}
              bind:value={hexDraft}
              onkeydown={e => e.key === 'Enter' && applyHex()}
              aria-label={t('cwHexPh')}
            />
            <button class="t5-btn pressable" onclick={applyHex} disabled={!hexDraft.trim()}>{t('cwApply')}</button>
            <div class="t5-pickerwrap">
              <button class="t5-btn pressable" onclick={() => (pickerOpen = !pickerOpen)} aria-expanded={pickerOpen}>
                <i class="ph ph-paint-bucket" style="font-size: 17px;"></i>{t('cwFromStock')}
              </button>
              {#if pickerOpen}
                <div class="t5-picker" transition:fly={{ y: -8, duration: 180 }}>
                  <input class="t5-hex" placeholder={t('cwSearchStock')} bind:value={pickerQuery} />
                  <div class="t5-pickerlist">
                    {#each pickerList as p (p.id)}
                      <button class="t5-pickeritem" onclick={() => pickPaint(p)}>
                        <PaintBottle r={p.r} g={p.g} b={p.b} width={20} height={33} label={p.name} />
                        <span class="t5-pickername">{p.name}</span>
                        <span class="t5-pickermeta font-mono">{p.manufacturer}{p.code ? ` · ${p.code}` : ''}</span>
                      </button>
                    {:else}
                      <p class="t5-empty">{t('cwNoStock')}</p>
                    {/each}
                  </div>
                </div>
              {/if}
            </div>
            {#if custom || satPct !== 100}
              <button class="t5-link pressable" onclick={backToRing} transition:fly={{ x: 8, duration: 200 }}>
                <i class="ph ph-arrow-counter-clockwise"></i>{t('cwBackToRing')}
              </button>
            {/if}
          </div>
          {#if hexError}<p class="t5-error" transition:fly={{ y: -4, duration: 160 }}>{hexError}</p>{/if}
          <label class="t5-sat">
            <span>{t('cwSatSlider')} <b class="font-mono">{satPct}%</b></span>
            <input type="range" min="0" max="100" step="5" bind:value={satPct} style="--fill: {satPct}%; --c: {rgbToHex(rawBase)};" />
          </label>
        </div>
      </div>

      <!-- harmonia -->
      <div class="t5-card">
        {#if activeScheme}
          {#key activeScheme.id}
            <div class="t5-harmhead" in:fly={{ y: 8, duration: 260 }}>
              <span class="t5-harmline" style="--c: var({activeScheme.token});"></span>
              <div>
                <p class="t5-harmtitle">{t(schemeKey(activeScheme.id))}</p>
                <p class="t5-harmhint">{t(hintKey(activeScheme.id))}</p>
              </div>
            </div>
          {/key}
          {#if activeScheme.id === 'monocromatico'}
            <p class="section-label" style="margin-top: 14px;">{t('cwMono')}</p>
            <div class="t5-mono">
              {#each bands as b, i (i)}
                <div class="t5-monoband" style="background: {b.hex}; transition-delay: {i * 45}ms;">
                  <span style="color: {ink(b.rgb)};">+{t(b.kind === 'claro' ? 'bandClaro' : b.kind === 'suave' ? 'bandSuave' : 'bandProfundo')}</span>
                  <span class="font-mono" style="color: {ink(b.rgb)};">{b.hex}</span>
                </div>
              {/each}
              <div class="t5-monoband" style="background: {baseHex};">
                <span style="color: {ink(base)};">+{t('bandBrilhante')} · {t('cwPure')}</span>
                <span class="font-mono" style="color: {ink(base)};">{baseHex}</span>
              </div>
            </div>
          {:else}
            {#key activeScheme.id}
              <div class="t5-swatches">
                {#each swatches as sw, i (sw.offset)}
                  {@const near = nearestBySwatch[i]}
                  <div class="t5-sw" animate:flip={{ duration: 300 }} in:scale={{ start: 0.6, duration: 420, delay: i * 70, easing: backOut }}>
                    <button
                      class="t5-swcolor"
                      style="background: {sw.hex};"
                      onclick={() => useAsBase(sw)}
                      aria-label={t('cwUseAsBase')}
                      title={t('cwUseAsBase')}
                    >
                      <span class="t5-swoff font-mono" style="color: {ink(sw.rgb)};">{offsetLabel(sw.offset)}</span>
                    </button>
                    <span class="t5-swname">{t(hueKey(sw.hue.id))}</span>
                    <span class="t5-swhex font-mono">{sw.hex}</span>
                    {#if near}
                      <span class="t5-swnear">
                        <span class="t5-neardot" style="background: rgb({near.r}, {near.g}, {near.b});"></span>
                        {t(pool.kind === 'stock' ? 'cwNearest' : 'cwNearestCat', { name: near.name, d: decimal(near.d, 1) })}
                      </span>
                    {/if}
                    {#if semTintaPronta(near)}
                      <button class="t5-swmix pressable" onclick={() => (mistura = { r: sw.rgb.r, g: sw.rgb.g, b: sw.rgb.b, nome: t(hueKey(sw.hue.id)) })}>
                        <i class="ph ph-flask" style="font-size: 16px;"></i>{t('mxBtn')}
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            {/key}
          {/if}
        {:else}
          <div class="t5-allrows">
            {#each allRows as row, ri (row.s.id)}
              <button class="t5-allrow pressable" onclick={() => (scheme = row.s.id)} in:fly={{ x: -10, duration: 260, delay: ri * 50 }}>
                <span class="t5-harmline small" style="--c: var({row.s.token});"></span>
                <span class="t5-allname">{t(schemeKey(row.s.id))}</span>
                <span class="t5-allsw">
                  {#each row.sw as sw (sw.offset)}
                    <span style="background: {sw.hex};"></span>
                  {/each}
                </span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- escurecer / clarear / saturação / valor -->
      <div class="t5-card">
        <div class="t5-tabs" role="tablist" style="--i: {tabIndex};">
          <span class="t5-tabind" aria-hidden="true"></span>
          {#each TABS as tb (tb.id)}
            <button class="t5-tab" class:on={tab === tb.id} role="tab" aria-selected={tab === tb.id} onclick={() => (tab = tb.id)}>
              <i class="ph {tb.icon}" style="font-size: 17px;"></i>{t(tb.key)}
            </button>
          {/each}
        </div>

        {#key tab}
          <div class="t5-tabbody" in:fly={{ y: 10, duration: 280, easing: cubicOut }}>
            {#if tab === 'valor'}
              <p class="t5-harmtitle">{t('cwValueTitle')}</p>
              <p class="t5-harmhint">{t('cwValueNote')}</p>
              <div class="t5-valuescale">
                {#each GRAY_STEPS as p (p)}
                  <div class="t5-vstep" class:hit={p === baseStep} style="background: {rgbToHex(grayAt(p))};">
                    <span class="font-mono" style="color: {p >= 50 ? 'var(--color-text)' : 'var(--color-bg)'};">{p}%</span>
                  </div>
                {/each}
                <span class="t5-vmarker" style="left: {Math.min(98, Math.max(2, 100 - baseValue - 5))}%;">
                  <span style="background: {baseHex};"></span>
                </span>
              </div>
              <div class="t5-valuepair">
                <div class="t5-vbox" style="background: {baseHex};"><span style="color: {ink(base)};">{baseHex}</span></div>
                <i class="ph ph-equals" style="font-size: 20px; color: var(--color-neutral-500);"></i>
                <div class="t5-vbox" style="background: {rgbToHex(grayOfSameValue(base))};">
                  <span style="color: {ink(grayOfSameValue(base))};">{t('cwSameValueGray')}</span>
                </div>
                <button class="t5-toggle pressable" class:on={grayscale} onclick={() => (grayscale = !grayscale)}>
                  <i class="ph ph-eye" style="font-size: 18px;"></i>{grayscale ? t('cwSeeColors') : t('cwSeeValues')}
                </button>
              </div>
            {:else}
              {#if tab === 'saturacao'}
                <p class="t5-harmhint" style="margin-bottom: 12px;">{t('cwSatNote')}</p>
              {/if}
              <div class="t5-rows" style="--lv: {level};">
                {#each rows as row, ri (row.title + ri)}
                  <div class="t5-row">
                    <div class="t5-rowhead">
                      <span class="t5-rowtitle"><i class="ph {row.good ? 'ph-check-circle' : 'ph-warning-circle'}" class:good={row.good}></i>{t(row.title)}</span>
                      <span class="t5-rowwhy">{t(row.why)}</span>
                    </div>
                    <div class="t5-steps">
                      {#each row.path.steps as st, k (k)}
                        <button
                          class="t5-step"
                          class:sel={k === level}
                          style="background: {css(st)}; transition-delay: {k * 55 + ri * 30}ms;"
                          onclick={() => (level = k)}
                          aria-label="{t('cwHowMuch')} {k}"
                        ></button>
                      {/each}
                    </div>
                  </div>
                {/each}
              </div>
              <label class="t5-level">
                <span>{t('cwHowMuch')}</span>
                <input type="range" min="0" max="4" step="1" bind:value={level} style="--fill: {level * 25}%; --c: var(--color-accent);" />
              </label>
              <p class="section-label" style="margin: 14px 0 8px;">{t('cwResult')}</p>
              <div class="t5-compare">
                {#each rows as row, ri (row.title + ri)}
                  {@const st = row.path.steps[level]}
                  <div class="t5-cmp">
                    <div class="t5-cmpsw" style="background: {css(st)};"></div>
                    <span class="t5-cmptitle">{t(row.title)}</span>
                    <span class="t5-cmpmeta font-mono">{rgbToHex(st)}</span>
                    <span class="t5-meter" aria-hidden="true"><span style="width: {Math.min(100, chromaKept(st))}%;"></span></span>
                    <span class="t5-cmpmeta font-mono">
                      {t('cwChromaKept', { p: chromaKept(st) })} · {t('cwValueShort', { v: decimal(valueOf(st), 0) })}
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/key}
      </div>
    </section>
  </div>
</div>

{#if guiaAberto}
  <GuiaCirculo
    {base}
    {baseWheel}
    {fromRing}
    matiz={custom ? custom.label : t(hueKey(baseHue.id))}
    onfechar={() => (guiaAberto = false)}
    onver={verNoDisco}
  />
{/if}

{#if mistura}
  <MisturaHarmonia r={mistura.r} g={mistura.g} b={mistura.b} nome={mistura.nome} onfechar={() => (mistura = null)} />
{/if}

<style>
  .t5 {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .t5-body {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 52%) minmax(0, 1fr);
  }

  .t5-wheelcol {
    min-height: 0;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    padding: 14px 18px 16px;
    border-right: 1px solid var(--color-line);
    background:
      radial-gradient(ellipse at 50% 45%, var(--color-surface) 0%, transparent 70%),
      var(--color-bg);
  }

  .t5-livebar {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 30px;
  }

  .t5-livedot {
    width: 16px;
    height: 16px;
    border-radius: 999px;
    box-shadow: 0 0 0 3px var(--color-panel), 0 0 16px 2px currentColor;
    transition: background 300ms ease;
  }

  .t5-livename {
    font-size: 19px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--color-text);
  }

  .t5-livemeta {
    font-size: 13px;
    color: var(--color-neutral-500);
  }

  .t5-wheelbox {
    flex: 1;
    min-height: 0;
    container-type: size;
    display: grid;
    place-items: center;
  }

  .t5-wheel {
    width: min(100cqw, 100cqh);
    aspect-ratio: 1;
  }

  .t5-hint {
    text-align: center;
    font-size: 12.5px;
    color: var(--color-neutral-600);
  }

  .t5-chips {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px;
  }

  .t5-chip {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-400);
    font-family: inherit;
    font-size: 13.5px;
    font-weight: 500;
    cursor: pointer;
    transition: border-color 200ms ease, color 200ms ease, background 200ms ease, transform 260ms cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .t5-chip:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }

  .t5-chip.on {
    border-color: var(--color-accent);
    color: var(--color-text);
    background: var(--color-accent-hover);
    transform: translateY(-2px);
  }

  .t5-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 48px;
    padding: 0 14px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-400);
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    transition: all 200ms ease;
  }

  .t5-toggle.on {
    border-color: var(--color-accent);
    color: var(--color-accent-400);
    background: var(--color-accent-hover);
  }

  /* ── painel ── */
  .t5-panel {
    min-height: 0;
    overflow-y: auto;
    padding: 18px 20px 28px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .t5-card {
    padding: 16px;
    border: 1px solid var(--color-rule);
    border-radius: var(--radius-lg);
    background: var(--color-panel);
  }

  .t5-basecard {
    display: grid;
    grid-template-columns: 112px minmax(0, 1fr);
    gap: 16px;
  }

  .t5-baseswatch {
    position: relative;
    overflow: hidden;
    border-radius: 12px;
    min-height: 150px;
    display: flex;
    align-items: flex-end;
    padding: 10px;
    transition: background 480ms ease;
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
  }

  .t5-ripple {
    position: absolute;
    inset: -40%;
    border-radius: 999px;
    opacity: 0.9;
  }

  .t5-basehex {
    position: relative;
    font-size: 13px;
    font-weight: 600;
  }

  .t5-baseinfo {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  .t5-basename {
    font-size: clamp(19px, 2.2cqi, 26px);
    font-weight: 600;
    letter-spacing: 0.02em;
    line-height: 1.1;
  }

  .t5-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .t5-tag {
    padding: 3px 9px;
    border-radius: 999px;
    border: 1px solid var(--color-neutral-800);
    font-size: 12px;
    color: var(--color-neutral-400);
  }

  .t5-tag.warm {
    border-color: var(--color-accent-700);
    color: var(--color-accent-300);
  }

  .t5-inputs {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    margin-top: 4px;
  }

  .t5-hex {
    flex: 1;
    min-width: 150px;
    height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-field-border);
    border-radius: 8px;
    background: var(--color-field);
    color: var(--color-text);
    font-size: 14px;
  }

  .t5-hex:focus {
    outline: none;
    border-color: var(--color-accent);
  }

  .t5-btn {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-300);
    font-family: inherit;
    font-size: 13.5px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
  }

  .t5-btn:disabled {
    opacity: 0.45;
  }

  .t5-btn:hover:not(:disabled) {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t5-link {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 44px;
    padding: 0 6px;
    border: none;
    background: transparent;
    color: var(--color-accent-400);
    font-family: inherit;
    font-size: 13.5px;
    cursor: pointer;
  }

  .t5-error {
    font-size: 12.5px;
    color: var(--color-accent-300);
  }

  .t5-pickerwrap {
    position: relative;
  }

  .t5-picker {
    position: absolute;
    z-index: 30;
    top: 50px;
    right: 0;
    width: min(360px, 70cqi);
    padding: 10px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 12px;
    background: var(--color-raised);
    box-shadow: var(--shadow-pop);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .t5-pickerlist {
    max-height: 300px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .t5-pickeritem {
    display: grid;
    grid-template-columns: 24px minmax(0, 1fr);
    grid-template-rows: auto auto;
    column-gap: 10px;
    align-items: center;
    padding: 6px 8px;
    min-height: 48px;
    border: none;
    border-radius: 8px;
    background: transparent;
    text-align: left;
    font-family: inherit;
    cursor: pointer;
  }

  .t5-pickeritem :global(svg) {
    grid-row: span 2;
  }

  .t5-pickeritem:hover {
    background: var(--color-panel);
  }

  .t5-pickername {
    font-size: 14px;
    color: var(--color-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .t5-pickermeta {
    font-size: 12px;
    color: var(--color-neutral-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .t5-empty {
    padding: 12px;
    font-size: 13px;
    color: var(--color-neutral-500);
  }

  .t5-sat,
  .t5-level {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 6px;
    font-size: 13px;
    color: var(--color-neutral-400);
  }

  input[type='range'] {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    height: 32px;
    background: transparent;
    cursor: pointer;
  }

  input[type='range']::-webkit-slider-runnable-track {
    height: 6px;
    border-radius: 999px;
    background: linear-gradient(to right, var(--c) var(--fill), var(--color-neutral-800) var(--fill));
  }

  input[type='range']::-moz-range-track {
    height: 6px;
    border-radius: 999px;
    background: linear-gradient(to right, var(--c) var(--fill), var(--color-neutral-800) var(--fill));
  }

  input[type='range']::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 24px;
    height: 24px;
    margin-top: -9px;
    border-radius: 999px;
    background: var(--color-text);
    border: 3px solid var(--color-bg);
    box-shadow: 0 0 0 1px var(--color-neutral-600);
    transition: transform 160ms ease;
  }

  input[type='range']:active::-webkit-slider-thumb {
    transform: scale(1.2);
  }

  input[type='range']::-moz-range-thumb {
    width: 20px;
    height: 20px;
    border-radius: 999px;
    background: var(--color-text);
    border: 3px solid var(--color-bg);
  }

  /* ── harmonia ── */
  .t5-harmhead {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .t5-harmline {
    flex-shrink: 0;
    width: 6px;
    height: 44px;
    margin-top: 2px;
    border-radius: 999px;
    background: var(--c);
  }

  .t5-harmline.small {
    height: 22px;
    width: 5px;
    margin: 0;
  }

  .t5-harmtitle {
    font-size: 17px;
    font-weight: 600;
    color: var(--color-text);
  }

  .t5-harmhint {
    margin-top: 3px;
    font-size: 13.5px;
    line-height: 1.45;
    color: var(--color-neutral-400);
    text-wrap: pretty;
  }

  .t5-swatches {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 12px;
    margin-top: 14px;
  }

  .t5-sw {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .t5-swcolor {
    position: relative;
    height: 78px;
    border: none;
    border-radius: 10px;
    cursor: pointer;
    display: flex;
    align-items: flex-start;
    justify-content: flex-end;
    padding: 7px 9px;
    transition: background 480ms ease, transform 240ms cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 240ms ease;
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
  }

  .t5-swcolor:hover {
    transform: translateY(-3px) scale(1.03);
    box-shadow: 0 10px 22px rgba(0, 0, 0, 0.45);
  }

  .t5-swoff {
    font-size: 12px;
    font-weight: 600;
  }

  .t5-swname {
    font-size: 12.5px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--color-text);
  }

  .t5-swhex {
    font-size: 12px;
    color: var(--color-neutral-500);
  }

  .t5-swnear {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    font-size: 11.5px;
    line-height: 1.35;
    color: var(--color-neutral-400);
  }

  .t5-neardot {
    flex-shrink: 0;
    width: 10px;
    height: 10px;
    margin-top: 2px;
    border-radius: 999px;
    box-shadow: 0 0 0 1px var(--color-neutral-700);
  }

  .t5-swmix {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    min-height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-accent-700);
    border-radius: 8px;
    background: transparent;
    color: var(--color-accent-400);
    font-family: inherit;
    font-size: 13.5px;
    font-weight: 500;
    cursor: pointer;
  }

  .t5-mono {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 8px;
  }

  .t5-monoband {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 40px;
    padding: 0 12px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 600;
    transition: background 480ms ease;
    animation: band-in 420ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }

  .t5-monoband:nth-child(2) { animation-delay: 50ms; }
  .t5-monoband:nth-child(3) { animation-delay: 100ms; }
  .t5-monoband:nth-child(4) { animation-delay: 150ms; }
  .t5-monoband:nth-child(5) { animation-delay: 200ms; }
  .t5-monoband:nth-child(6) { animation-delay: 250ms; }

  @keyframes band-in {
    from {
      opacity: 0;
      transform: scaleX(0.4);
      transform-origin: left center;
    }
  }

  .t5-allrows {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .t5-allrow {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 48px;
    padding: 6px 10px;
    border: 1px solid transparent;
    border-radius: 10px;
    background: transparent;
    font-family: inherit;
    cursor: pointer;
    text-align: left;
  }

  .t5-allrow:hover {
    border-color: var(--color-neutral-800);
    background: var(--color-surface);
  }

  .t5-allname {
    flex: 1;
    font-size: 14px;
    color: var(--color-text);
  }

  .t5-allsw {
    display: flex;
    border-radius: 6px;
    overflow: hidden;
  }

  .t5-allsw span {
    width: 34px;
    height: 30px;
    transition: background 480ms ease;
  }

  /* ── abas ── */
  .t5-tabs {
    position: relative;
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border-bottom: 1px solid var(--color-rule);
    margin: -4px -4px 14px;
  }

  .t5-tabind {
    position: absolute;
    bottom: -1px;
    left: 0;
    width: 25%;
    height: 2px;
    background: var(--color-accent);
    border-radius: 2px;
    transform: translateX(calc(var(--i) * 100%));
    transition: transform 380ms cubic-bezier(0.34, 1.4, 0.64, 1);
  }

  .t5-tab {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    height: 48px;
    border: none;
    background: transparent;
    color: var(--color-neutral-500);
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: color 200ms ease;
  }

  .t5-tab.on {
    color: var(--color-text);
  }

  .t5-rows {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .t5-rowhead {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-bottom: 7px;
  }

  .t5-rowtitle {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text);
  }

  .t5-rowtitle i {
    font-size: 17px;
    color: var(--color-neutral-500);
  }

  .t5-rowtitle i.good {
    color: var(--color-accent-400);
  }

  .t5-rowwhy {
    font-size: 12.5px;
    color: var(--color-neutral-400);
    line-height: 1.4;
  }

  .t5-steps {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 4px;
  }

  .t5-step {
    height: 46px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition:
      background 520ms ease,
      transform 260ms cubic-bezier(0.34, 1.56, 0.64, 1),
      box-shadow 200ms ease;
  }

  .t5-step.sel {
    transform: translateY(-4px);
    box-shadow: 0 0 0 2px var(--color-bg), 0 0 0 4px var(--color-accent);
  }

  .t5-compare {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
    gap: 12px;
  }

  .t5-cmp {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .t5-cmpsw {
    height: 70px;
    border-radius: 10px;
    transition: background 520ms ease;
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.08);
  }

  .t5-cmptitle {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text);
  }

  .t5-cmpmeta {
    font-size: 11.5px;
    color: var(--color-neutral-500);
  }

  .t5-meter {
    height: 5px;
    border-radius: 999px;
    background: var(--color-neutral-800);
    overflow: hidden;
  }

  .t5-meter span {
    display: block;
    height: 100%;
    border-radius: 999px;
    background: var(--color-accent);
    transition: width 520ms cubic-bezier(0.34, 1.3, 0.64, 1);
  }

  /* ── valor ── */
  .t5-valuescale {
    position: relative;
    display: grid;
    grid-template-columns: repeat(10, 1fr);
    margin: 18px 0 30px;
    border-radius: 8px;
    overflow: visible;
  }

  .t5-vstep {
    height: 52px;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding-bottom: 5px;
    font-size: 11px;
    transition: transform 300ms cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .t5-vstep:first-child {
    border-radius: 8px 0 0 8px;
  }

  .t5-vstep:last-child {
    border-radius: 0 8px 8px 0;
  }

  .t5-vstep.hit {
    transform: scaleY(1.18);
    box-shadow: 0 0 0 2px var(--color-accent);
    z-index: 1;
  }

  .t5-vmarker {
    position: absolute;
    bottom: -24px;
    translate: -50% 0;
    transition: left 700ms cubic-bezier(0.34, 1.4, 0.64, 1);
  }

  .t5-vmarker span {
    display: block;
    width: 18px;
    height: 18px;
    border-radius: 999px 999px 999px 0;
    rotate: 135deg;
    box-shadow: 0 0 0 2px var(--color-text);
    transition: background 400ms ease;
  }

  .t5-valuepair {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
  }

  .t5-vbox {
    flex: 1;
    min-width: 110px;
    height: 64px;
    border-radius: 10px;
    display: flex;
    align-items: flex-end;
    padding: 8px 10px;
    font-size: 12px;
    font-weight: 600;
    transition: background 520ms ease;
  }

  @media (max-width: 900px) {
    .t5-body {
      grid-template-columns: minmax(0, 1fr);
      overflow-y: auto;
    }

    .t5-wheelcol {
      border-right: none;
      border-bottom: 1px solid var(--color-line);
    }

    .t5-wheelbox {
      flex: none;
      height: min(92vw, 70dvh);
    }

    .t5-panel {
      overflow: visible;
    }
  }
</style>
