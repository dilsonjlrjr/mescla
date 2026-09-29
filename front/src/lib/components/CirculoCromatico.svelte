<script lang="ts">
  // rf-21 — o disco do círculo cromático, fiel às fotos de referência
  // (temp/IMG_0005–0007): anel externo fixo com os 12 matizes e o disco de
  // papel por cima, que gira. As janelas do disco são FUROS de verdade (máscara
  // SVG) sobre uma base fixa impressa com a escala de cada matiz — girar o
  // disco faz a janela passar de um matiz para o outro, como no papel.
  //
  // Em relação à seta (a base, offset 0), o disco segue a disposição da foto:
  // janelas em −30, 0, +30, +90, +150, +180, +210 e +270; escala de cinza clara
  // em +120 e escura em +300; legenda (+claro/+suave/+profundo/+brilhante) em
  // +60; marca em +240. Entalhes coloridos na borda marcam cada harmonia.
  import '@fontsource/ibm-plex-mono/500.css';
  import '@fontsource-variable/archivo';
  import { onMount } from 'svelte';
  import {
    WHEEL,
    SCHEMES,
    STEP,
    GRAY_DARK,
    GRAY_LIGHT,
    grayAt,
    monoBands,
    nearestIndex,
    norm360,
    rgbToHex,
    schemesAtOffset,
    type SchemeId,
  } from '../color/circulo';
  import { t, i18n } from '../i18n.svelte';
  import { hueKey, schemeKey } from '../circuloTexto';

  interface Props {
    /** grau de roda para onde a seta aponta (a cor-base) — assentado */
    angle: number;
    /** grau "ao vivo" durante o arrasto e a animação */
    live?: number;
    /** harmonia em destaque; 'todas' mostra o diagrama inteiro como impresso */
    scheme: SchemeId | 'todas';
    /** gira em passos de 30° (um matiz do anel) ao soltar */
    snap?: boolean;
    /** a cor do usuário, marcada por fora do anel */
    marker?: { deg: number; hex: string } | null;
    /** vista em valores (escala de cinza) */
    grayscale?: boolean;
    /** degrau de cinza (10–100) que corresponde ao valor da base */
    valueStep?: number | null;
    onpick?: (deg: number) => void;
  }

  let {
    angle = $bindable(0),
    live = $bindable(0),
    scheme,
    snap = true,
    marker = null,
    grayscale = false,
    valueStep = null,
    onpick,
  }: Props = $props();

  // ── geometria (viewBox centrado em 0,0) ──────────────────────────────────
  const R_OUT = 490;
  const R_RING_IN = 386;
  const R_DISK = 404;
  const W_IN = 150;
  const W_OUT = 335;
  const BAND = (W_OUT - W_IN) / 5;
  const W_HALF = 11.5; // meia-abertura angular da janela
  const R_SHAPE = 124;

  const COLOR_WINDOWS = [-30, 0, 30, 90, 150, 180, 210, 270];
  const GRAY_WINDOWS: Array<{ off: number; steps: number[] }> = [
    { off: 120, steps: GRAY_LIGHT },
    { off: 300, steps: GRAY_DARK },
  ];
  const OFFSETS = Array.from({ length: 12 }, (_, i) => i * STEP);
  const LOGO = ['M', 'E', 'S', 'C', 'L', 'A'];
  const LOGO_HUES = [10, 9, 8, 7, 4, 1];

  function polar(r: number, deg: number): [number, number] {
    const a = (deg * Math.PI) / 180;
    return [r * Math.sin(a), -r * Math.cos(a)];
  }

  function f(n: number): string {
    return n.toFixed(2);
  }

  /** Setor anular de r1 a r2, de a1 a a2 graus (sentido horário). */
  function sector(r1: number, r2: number, a1: number, a2: number): string {
    const [x1, y1] = polar(r2, a1);
    const [x2, y2] = polar(r2, a2);
    const [x3, y3] = polar(r1, a2);
    const [x4, y4] = polar(r1, a1);
    const large = a2 - a1 > 180 ? 1 : 0;
    return `M${f(x1)} ${f(y1)} A${r2} ${r2} 0 ${large} 1 ${f(x2)} ${f(y2)} L${f(x3)} ${f(y3)} A${r1} ${r1} 0 ${large} 0 ${f(x4)} ${f(y4)}Z`;
  }

  /** Arco centrado no topo, para textPath (o grupo gira até a posição). */
  function arcTop(r: number, span = 40): string {
    const [x1, y1] = polar(r, -span);
    const [x2, y2] = polar(r, span);
    return `M${f(x1)} ${f(y1)} A${r} ${r} 0 0 1 ${f(x2)} ${f(y2)}`;
  }

  const windowPath = sector(W_IN, W_OUT, -W_HALF, W_HALF);

  // Base fixa impressa: 12 matizes × 5 faixas (sob o disco).
  const under = WHEEL.map(h => ({ index: h.index, bands: monoBands(h.rgb) }));

  // ── rotação: mola + arrasto com inércia ─────────────────────────────────
  let shown = $state(angle - 210);
  let vel = 0;
  let target = angle;
  let dragging = $state(false);
  let raf = 0;
  let last = 0;
  let reduced = false;
  let internalSet = false;
  let svgEl: SVGSVGElement | undefined = $state();
  let entered = $state(false);

  function unwrapNear(a: number, ref: number): number {
    return ref + ((((a - ref) % 360) + 540) % 360) - 180;
  }

  function tick(now: number) {
    const dt = Math.min(0.034, (now - last) / 1000 || 0.016);
    last = now;
    if (!dragging) {
      const k = 150;
      const c = 2 * Math.sqrt(k) * 0.72; // levemente subamortecida: o papel "assenta"
      const acc = k * (target - shown) - c * vel;
      vel += acc * dt;
      shown += vel * dt;
      if (Math.abs(target - shown) < 0.02 && Math.abs(vel) < 0.05) {
        shown = target;
        vel = 0;
        live = norm360(shown);
        raf = 0;
        return;
      }
    }
    live = norm360(shown);
    raf = requestAnimationFrame(tick);
  }

  function run() {
    if (reduced) {
      shown = target;
      live = norm360(shown);
      return;
    }
    if (!raf) {
      last = performance.now();
      raf = requestAnimationFrame(tick);
    }
  }

  $effect(() => {
    const a = angle;
    if (internalSet) {
      internalSet = false;
      return;
    }
    target = unwrapNear(a, shown);
    run();
  });

  onMount(() => {
    reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
    target = angle;
    run();
    requestAnimationFrame(() => (entered = true));
    return () => cancelAnimationFrame(raf);
  });

  // ── arrasto ──────────────────────────────────────────────────────────────
  let pointerAngle = 0;
  let moveTime = 0;
  let moved = 0;

  function angleAt(e: PointerEvent): number {
    const r = svgEl!.getBoundingClientRect();
    const cx = r.left + r.width / 2;
    const cy = r.top + r.height / 2;
    return (Math.atan2(e.clientX - cx, -(e.clientY - cy)) * 180) / Math.PI;
  }

  function down(e: PointerEvent) {
    if (!svgEl) return;
    (e.currentTarget as Element).setPointerCapture(e.pointerId);
    dragging = true;
    pointerAngle = angleAt(e);
    moveTime = performance.now();
    moved = 0;
    vel = 0;
    run();
  }

  function move(e: PointerEvent) {
    if (!dragging) return;
    const a = angleAt(e);
    const d = ((a - pointerAngle + 540) % 360) - 180;
    pointerAngle = a;
    const now = performance.now();
    const dt = Math.max(1, now - moveTime) / 1000;
    moveTime = now;
    shown += d;
    moved += Math.abs(d);
    vel = vel * 0.6 + (d / dt) * 0.4;
    live = norm360(shown);
  }

  function up() {
    if (!dragging) return;
    dragging = false;
    const fling = performance.now() - moveTime > 90 ? 0 : vel * 0.22;
    let dest = shown + Math.max(-240, Math.min(240, fling));
    if (snap) dest = Math.round(dest / STEP) * STEP;
    target = dest;
    internalSet = true;
    angle = norm360(dest);
    run();
  }

  function key(e: KeyboardEvent) {
    const d = e.key === 'ArrowRight' || e.key === 'ArrowUp' ? STEP : e.key === 'ArrowLeft' || e.key === 'ArrowDown' ? -STEP : 0;
    if (!d) return;
    e.preventDefault();
    const base = Math.round(target / STEP) * STEP;
    target = base + d;
    internalSet = true;
    angle = norm360(target);
    run();
  }

  // ── destaque no anel ──────────────────────────────────────────────────────
  let liveIndex = $derived(nearestIndex(live));
  let active = $derived(scheme === 'todas' ? null : SCHEMES.find(s => s.id === scheme)!);
  let members = $derived.by(() => {
    const set = new Set<number>();
    if (!active) return set;
    for (const o of active.offsets) set.add((liveIndex + Math.round(o / STEP) + 12) % 12);
    return set;
  });

  // ── diagrama do miolo ────────────────────────────────────────────────────
  const DASH: Record<string, string> = {
    solid: 'none',
    dash: '16 10',
    longdash: '26 12',
    dot: '2 9',
  };

  function shapePoints(offsets: number[]): string {
    return [...offsets]
      .map(o => norm360(o))
      .sort((a, b) => a - b)
      .map(o => polar(R_SHAPE, o).map(f).join(','))
      .join(' ');
  }

  function chevron(o: number): string {
    const [x, y] = polar(R_SHAPE + 4, o);
    const [lx, ly] = polar(R_SHAPE - 9, o - 5.5);
    const [rx, ry] = polar(R_SHAPE - 9, o + 5.5);
    return `M${f(lx)} ${f(ly)} L${f(x)} ${f(y)} L${f(rx)} ${f(ry)}`;
  }

  const diagram = SCHEMES.filter(s => s.id !== 'monocromatico');

  // ── legenda da escala (em +60, ao lado da janela de +30) ───────────────
  const LEGEND_A = 30 + W_HALF + 3.2;
  const legend = [
    { key: 'bandClaro', sub: 'bandClaroSub', r1: W_IN, r2: W_IN + 2 * BAND },
    { key: 'bandSuave', sub: 'bandSuaveSub', r1: W_IN + 2 * BAND, r2: W_IN + 3 * BAND },
    { key: 'bandProfundo', sub: 'bandProfundoSub', r1: W_IN + 3 * BAND, r2: W_OUT },
    { key: 'bandBrilhante', sub: 'bandBrilhanteSub', r1: W_OUT + 4, r2: W_OUT + 26 },
  ] as const;

  function legendLine(r1: number, r2: number): string {
    const [x1, y1] = polar(r1 + 3, LEGEND_A);
    const [x2, y2] = polar(r2 - 3, LEGEND_A);
    const [t1x, t1y] = polar(r1 + 3, LEGEND_A - 1.6);
    const [t2x, t2y] = polar(r2 - 3, LEGEND_A - 1.6);
    return `M${f(t1x)} ${f(t1y)} L${f(x1)} ${f(y1)} L${f(x2)} ${f(y2)} L${f(t2x)} ${f(t2y)}`;
  }

  /** Texto na tangente (perpendicular ao raio), começando em `deg`. */
  function tangentText(r: number, deg: number): string {
    const [x, y] = polar(r, deg);
    return `translate(${f(x)} ${f(y)}) rotate(${f(deg)})`;
  }

  // O rótulo de cada matiz lê melhor em caixa alta curta; os nomes compostos
  // quebram em duas linhas só se não couberem (nenhum passa de 15 letras).
  let ringLabels = $derived.by(() => {
    void i18n.lang;
    return WHEEL.map(h => t(hueKey(h.id)));
  });

  // O gomo tem ~233px de arco no raio do nome. Em 27px, "AZUL VIOLETADO"
  // (14 letras) passava disso e o gomo vizinho, pintado depois, cortava a
  // sobra. Nome longo encolhe até caber em ~205px, com folga para a borda.
  const NAME_MAX = 27;
  const NAME_FIT = 205;
  const CHAR_EM = 0.72; // largura média de uma letra em caixa alta no Archivo 800, com o espaçamento

  function nameSize(label: string): number {
    return Math.min(NAME_MAX, NAME_FIT / (label.length * CHAR_EM));
  }

  function schemeLabel(id: SchemeId): string {
    void i18n.lang;
    return t(schemeKey(id));
  }
</script>

<svg
  bind:this={svgEl}
  class="wheel"
  class:entered
  class:gray={grayscale}
  class:dragging
  viewBox="-520 -520 1040 1040"
  role="img"
  aria-label={t('wheelAria')}
>
  <defs>
    <radialGradient id="cw-paper" cx="50%" cy="42%" r="62%">
      <stop offset="0%" stop-color="var(--wheel-paper)" />
      <stop offset="100%" stop-color="var(--wheel-paper-shade)" />
    </radialGradient>
    <radialGradient id="cw-rivet" cx="38%" cy="34%" r="70%">
      <stop offset="0%" stop-color="var(--wheel-rivet-hi)" />
      <stop offset="55%" stop-color="var(--wheel-rivet-mid)" />
      <stop offset="100%" stop-color="var(--wheel-rivet-lo)" />
    </radialGradient>
    <radialGradient id="cw-sheen" cx="30%" cy="22%" r="80%">
      <stop offset="0%" stop-color="var(--wheel-ring-text)" stop-opacity="0.22" />
      <stop offset="55%" stop-color="var(--wheel-ring-text)" stop-opacity="0" />
    </radialGradient>
    <filter id="cw-inset" x="-30%" y="-30%" width="160%" height="160%">
      <feGaussianBlur stdDeviation="5" />
    </filter>
    <filter id="cw-glow" x="-30%" y="-30%" width="160%" height="160%">
      <feGaussianBlur stdDeviation="7" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
    <path id="cw-arc-name" d={arcTop(446)} />
    <path id="cw-arc-warm" d={arcTop(410)} />
    <path id="cw-arc-l0" d={arcTop(390)} />
    <path id="cw-arc-l1" d={arcTop(375)} />
    <path id="cw-arc-l2" d={arcTop(360)} />
    <path id="cw-arc-gray" d={arcTop(345)} />
    <!-- Furos do disco: branco = papel, preto = janela. -->
    <mask id="cw-holes" maskUnits="userSpaceOnUse" x="-520" y="-520" width="1040" height="1040">
      <circle r={R_DISK} fill="#fff" />
      {#each COLOR_WINDOWS as o (o)}
        <path d={windowPath} transform="rotate({o})" fill="#000" />
      {/each}
    </mask>
    <clipPath id="cw-win-clip">
      {#each COLOR_WINDOWS as o (o)}
        <path d={windowPath} transform="rotate({o})" />
      {/each}
    </clipPath>
    <clipPath id="cw-one-win">
      <path d={windowPath} />
    </clipPath>
  </defs>

  <!-- ── anel externo (fixo) ── -->
  <g class="ring">
    <circle r={R_OUT + 3} cy="5" fill="var(--wheel-cardboard)" opacity="0.55" />
    {#each WHEEL as h, i (h.id)}
      {@const lifted = members.has(i)}
      {@const dim = active != null && !lifted}
      {@const [lx, ly] = polar(lifted ? 12 : 0, i * STEP)}
      <g
        class="seg"
        class:lifted
        class:dim
        class:base={i === liveIndex}
        style="--d: {i * 38}ms; transform: translate({f(lx)}px, {f(ly)}px);"
      >
        <path
          d={sector(R_RING_IN, R_OUT, i * STEP - 15, i * STEP + 15)}
          fill={h.hex}
          stroke="var(--wheel-paper)"
          stroke-width="2.5"
          role="button"
          tabindex="-1"
          aria-label={ringLabels[i]}
          onclick={() => onpick?.(i * STEP)}
          onkeydown={() => {}}
        />
        <g transform="rotate({i * STEP})" pointer-events="none">
          <text class="ring-name" style="font-size: {nameSize(ringLabels[i])}px;">
            <textPath href="#cw-arc-name" xlink:href="#cw-arc-name" startOffset="50%" text-anchor="middle">{ringLabels[i]}</textPath>
          </text>
        </g>
      </g>
    {/each}
    <circle r={R_OUT} fill="url(#cw-sheen)" pointer-events="none" />
    <!-- Quentes | frias: fronteiras em 45° e 225°, setas para longe da fronteira -->
    <g pointer-events="none" class="warmcool">
      <g transform="rotate(26)"><text><textPath href="#cw-arc-warm" xlink:href="#cw-arc-warm" startOffset="50%" text-anchor="middle">← {t('warmColors')}</textPath></text></g>
      <g transform="rotate(64)"><text><textPath href="#cw-arc-warm" xlink:href="#cw-arc-warm" startOffset="50%" text-anchor="middle">{t('coolColors')} →</textPath></text></g>
      <g transform="rotate(206)"><text><textPath href="#cw-arc-warm" xlink:href="#cw-arc-warm" startOffset="50%" text-anchor="middle">← {t('coolColors')}</textPath></text></g>
      <g transform="rotate(244)"><text><textPath href="#cw-arc-warm" xlink:href="#cw-arc-warm" startOffset="50%" text-anchor="middle">{t('warmColors')} →</textPath></text></g>
    </g>
    {#if marker}
      {@const [mx, my] = polar(R_OUT + 14, marker.deg)}
      <g class="marker" transform="translate({f(mx)} {f(my)}) rotate({f(marker.deg)})">
        <circle class="marker-pulse" cy="-6" r="14" fill="none" stroke={marker.hex} stroke-width="3" />
        <path d="M0 14 C-13 0 -13 -18 0 -18 C13 -18 13 0 0 14Z" fill={marker.hex} stroke="var(--wheel-paper)" stroke-width="3" />
      </g>
    {/if}
  </g>

  <!-- sombra do disco sobre o anel (fixa: um disco redondo gira sem mudar a sombra) -->
  <circle r={R_DISK - 8} cy="7" fill="none" stroke="#000" stroke-opacity="0.34" stroke-width="18" filter="url(#cw-inset)" pointer-events="none" />

  <!-- ── base fixa impressa, vista pelas janelas ── -->
  <g class="under" clip-path="url(#cw-win-clip)" transform="rotate({f(shown)})">
    <!-- o clip gira junto do disco; a base desfaz a rotação e fica parada -->
    <g transform="rotate({f(-shown)})">
      {#each under as u (u.index)}
        {#each u.bands as b, k (k)}
          <path
            d={sector(W_IN + k * BAND, W_IN + (k + 1) * BAND, u.index * STEP - 15.2, u.index * STEP + 15.2)}
            fill={b.hex}
          />
        {/each}
      {/each}
      {#each [1, 2, 3, 4] as k (k)}
        <circle r={W_IN + k * BAND} fill="none" stroke="var(--wheel-paper)" stroke-width="2.4" />
      {/each}
    </g>
  </g>

  <!-- ── disco de papel (gira) ── -->
  <g
    class="disk"
    transform="rotate({f(shown)})"
    role="slider"
    tabindex="0"
    aria-label={t('wheelDiskAria')}
    aria-valuemin={0}
    aria-valuemax={359}
    aria-valuenow={Math.round(norm360(shown))}
    aria-valuetext={ringLabels[liveIndex]}
    onpointerdown={down}
    onpointermove={move}
    onpointerup={up}
    onpointercancel={up}
    onkeydown={key}
  >
    <g class="disk-body">
      <circle r={R_DISK} fill="url(#cw-paper)" mask="url(#cw-holes)" />
      <circle class="disk-edge" r={R_DISK - 1} fill="none" stroke="var(--wheel-paper-edge)" stroke-width="2" />

      <!-- borda e sombra interna das janelas (o corte do papelão) -->
      {#each COLOR_WINDOWS as o (o)}
        <g transform="rotate({o})">
          <!-- sombra interna sem filtro (degradê de traços): barata de girar -->
          <g clip-path="url(#cw-one-win)" fill="none" stroke="#000">
            <path d={windowPath} stroke-opacity="0.1" stroke-width="18" />
            <path d={windowPath} stroke-opacity="0.12" stroke-width="10" />
            <path d={windowPath} stroke-opacity="0.16" stroke-width="4" />
          </g>
          <path d={windowPath} fill="none" stroke="var(--wheel-cardboard)" stroke-width="2.2" />
        </g>
      {/each}

      <!-- escalas de cinza impressas -->
      {#each GRAY_WINDOWS as g (g.off)}
        <g transform="rotate({g.off})">
          {#each g.steps as pct, k (pct)}
            <path
              class="gray-band"
              class:hit={valueStep === pct}
              d={sector(W_IN + k * BAND, W_IN + (k + 1) * BAND, -W_HALF, W_HALF)}
              fill={rgbToHex(grayAt(pct))}
              stroke="var(--wheel-paper)"
              stroke-width="2.4"
            />
          {/each}
          <path d={windowPath} fill="none" stroke="var(--wheel-cardboard)" stroke-width="2.2" />
          {#each g.steps as pct, k (pct)}
            {@const r = W_IN + (k + 0.5) * BAND}
            {@const [tx, ty] = polar(r, -W_HALF - 4.5)}
            {@const [ax, ay] = polar(W_IN + k * BAND + 2, -W_HALF - 1.2)}
            {@const [bx, by] = polar(W_IN + (k + 1) * BAND - 2, -W_HALF - 1.2)}
            <path d="M{f(ax)} {f(ay)} L{f(bx)} {f(by)}" stroke="var(--wheel-ink)" stroke-width="1.3" fill="none" />
            <text class="pct" class:hit={valueStep === pct} transform="translate({f(tx)} {f(ty)}) rotate(-90)" text-anchor="middle" dominant-baseline="middle">{pct}%</text>
          {/each}
          <text class="rim gray-title"><textPath href="#cw-arc-gray" xlink:href="#cw-arc-gray" startOffset="50%" text-anchor="middle">{t('grayScale')}</textPath></text>
        </g>
      {/each}

      <!-- legenda da escala monocromática -->
      <g class="legend">
        {#each legend as l (l.key)}
          {@const mid = (l.r1 + l.r2) / 2}
          <path d={legendLine(l.r1, l.r2)} fill="none" stroke="var(--wheel-ink)" stroke-width="1.4" />
          <text class="legend-main" transform={tangentText(mid + 5, LEGEND_A + 1.6)}>+{t(l.key)}</text>
          <text class="legend-sub" transform={tangentText(mid - 7, LEGEND_A + 1.6)}>{t(l.sub)}</text>
        {/each}
      </g>

      <!-- marca (no lugar da logo impressa) -->
      <g transform="rotate(240) translate(0 -236) rotate(90)" class="logo">
        <text class="logo-kicker" y="-26" text-anchor="middle">{t('wheelKicker')}</text>
        <text class="logo-word" y="14" text-anchor="middle">
          {#each LOGO as ch, i (i)}<tspan fill={WHEEL[LOGO_HUES[i]].hex}>{ch}</tspan>{/each}
        </text>
      </g>

      <!-- entalhes e rótulos das harmonias, na borda -->
      {#each OFFSETS as o (o)}
        {@const list = o === 0 ? diagram : schemesAtOffset(o)}
        <g transform="rotate({o})">
          {#each list as s, k (s.id)}
            {@const spread = (k - (list.length - 1) / 2) * 3.1}
            {@const on = active?.id === s.id}
            <g transform="rotate({f(spread)})">
              <path
                class="notch"
                class:on
                d="M-7 {-R_DISK + 3} L0 {-R_DISK - 13} L7 {-R_DISK + 3}Z"
                fill="var({s.token})"
                style="--d: {400 + o * 2 + k * 40}ms"
              />
            </g>
          {/each}
          {#if o === 0}
            <text class="rim mono-title" class:on={active?.id === 'monocromatico'}>
              <textPath href="#cw-arc-l0" xlink:href="#cw-arc-l0" startOffset="50%" text-anchor="middle">{schemeLabel('monocromatico')}</textPath>
            </text>
          {:else}
            {#each list as s, k (s.id)}
              <text class="rim" class:on={active?.id === s.id} class:off={active != null && active.id !== s.id} fill="var({s.token})">
                <textPath href="#cw-arc-l{k}" xlink:href="#cw-arc-l{k}" startOffset="50%" text-anchor="middle">{schemeLabel(s.id)}</textPath>
              </text>
            {/each}
          {/if}
        </g>
      {/each}

      <!-- diagrama das harmonias no miolo -->
      <g class="diagram">
        {#each diagram as s (s.id)}
          {@const on = active?.id === s.id}
          {@const off = active != null && !on}
          {#key on ? `${s.id}-on` : s.id}
            <g class="shape" class:on class:off style="--c: var({s.token});">
              {#if s.id === 'complementar'}
                {@const [ax, ay] = polar(R_SHAPE, 0)}
                {@const [bx, by] = polar(R_SHAPE, 180)}
                <path d="M{f(ax)} {f(ay)} L{f(bx)} {f(by)}" fill="none" stroke-dasharray={DASH[s.stroke]} />
              {:else}
                <polygon points={shapePoints(s.offsets)} fill="none" stroke-dasharray={DASH[s.stroke]} />
              {/if}
              {#each s.offsets as o (o)}
                <path class="chev" d={chevron(o)} fill="none" />
              {/each}
            </g>
          {/key}
        {/each}
      </g>

      <!-- rebite central -->
      <circle r="22" fill="#000" opacity="0.14" cy="3" />
      <circle r="19.5" fill="#000" opacity="0.14" cy="2" />
      <circle r="17" fill="url(#cw-rivet)" stroke="var(--wheel-rivet-lo)" stroke-width="1.5" />
      <ellipse rx="6" ry="3.5" cx="-5" cy="-7" fill="var(--wheel-rivet-hi)" opacity="0.85" />
    </g>
  </g>
</svg>

<style>
  .wheel {
    display: block;
    width: 100%;
    height: 100%;
    touch-action: none;
    overflow: visible;
    transition: filter 700ms cubic-bezier(0.4, 0, 0.2, 1);
    -webkit-user-select: none;
    user-select: none;
  }

  .wheel.gray {
    filter: grayscale(1) contrast(1.05);
  }

  /* ── anel ── */
  .seg {
    opacity: 0;
    transition:
      transform 520ms cubic-bezier(0.34, 1.56, 0.64, 1),
      opacity 380ms ease,
      filter 380ms ease;
  }

  .entered .seg {
    opacity: 1;
    animation: seg-in 640ms cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
    animation-delay: var(--d);
  }

  .seg path {
    cursor: pointer;
  }

  .seg.dim {
    opacity: 0.42;
    filter: saturate(0.55);
  }

  .seg.lifted {
    filter: url(#cw-glow) brightness(1.06);
  }

  .seg:hover:not(.lifted) {
    filter: brightness(1.12);
  }

  @keyframes seg-in {
    from {
      opacity: 0;
      scale: 0.86;
    }
  }

  .ring-name {
    font-family: 'Archivo Variable', 'Inter', sans-serif;
    font-weight: 800;
    letter-spacing: 0.035em;
    fill: var(--wheel-ring-text);
    paint-order: stroke;
    stroke: rgba(0, 0, 0, 0.12);
    stroke-width: 1px;
  }

  .warmcool text {
    font-family: 'IBM Plex Mono', monospace;
    font-weight: 500;
    font-size: 13px;
    letter-spacing: 0.1em;
    fill: var(--wheel-ring-text);
    opacity: 0.92;
  }

  .marker {
    animation: marker-drop 700ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }

  .marker-pulse {
    animation: pulse 1.8s ease-out infinite;
    transform-box: fill-box;
    transform-origin: center;
  }

  @keyframes marker-drop {
    from {
      opacity: 0;
      translate: 0 -40px;
    }
  }

  @keyframes pulse {
    0% {
      scale: 0.6;
      opacity: 0.9;
    }
    100% {
      scale: 2.2;
      opacity: 0;
    }
  }

  /* ── disco ── */
  .disk {
    cursor: grab;
    outline: none;
  }

  .dragging .disk {
    cursor: grabbing;
  }

  .disk:focus-visible .disk-edge {
    stroke: var(--color-accent);
    stroke-width: 5;
  }

  .under {
    opacity: 0;
    transition: opacity 600ms ease 250ms;
  }

  .entered .under {
    opacity: 1;
  }

  .rim {
    font-family: 'IBM Plex Mono', monospace;
    font-weight: 500;
    font-size: 12.5px;
    letter-spacing: 0.09em;
    text-transform: uppercase;
    transition: opacity 300ms ease;
  }

  .rim.off {
    opacity: 0.4;
  }

  .rim.on {
    font-weight: 700;
    animation: label-pop 520ms cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .mono-title,
  .gray-title {
    fill: var(--wheel-ink);
  }

  .mono-title.on {
    fill: var(--color-accent-600);
  }

  @keyframes label-pop {
    0% {
      opacity: 0.2;
    }
    100% {
      opacity: 1;
    }
  }

  .pct {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 12px;
    font-weight: 500;
    fill: var(--wheel-ink);
    transition: fill 300ms ease;
  }

  .pct.hit {
    fill: var(--color-accent-600);
    font-weight: 700;
  }

  .gray-band {
    transition: stroke 300ms ease, stroke-width 300ms ease;
  }

  .gray-band.hit {
    stroke: var(--color-accent);
    stroke-width: 5;
    animation: band-hit 1.4s ease-in-out infinite;
  }

  @keyframes band-hit {
    50% {
      stroke-opacity: 0.35;
    }
  }

  .legend-main {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 13px;
    font-weight: 500;
    letter-spacing: 0.12em;
    fill: var(--wheel-ink);
  }

  .legend-sub {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 7.5px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    fill: var(--wheel-ink-soft);
  }

  .logo-kicker {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 10px;
    letter-spacing: 0.3em;
    fill: var(--wheel-ink-soft);
  }

  .logo-word {
    font-family: 'Archivo Variable', 'Inter', sans-serif;
    font-weight: 800;
    font-size: 40px;
    letter-spacing: 0.04em;
  }

  .notch {
    opacity: 0;
    transform-box: fill-box;
    transform-origin: 50% 100%;
    transition: scale 360ms cubic-bezier(0.34, 1.56, 0.64, 1), filter 300ms ease;
  }

  .entered .notch {
    opacity: 1;
    animation: notch-in 520ms cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
    animation-delay: var(--d);
  }

  .notch.on {
    scale: 1.45;
    filter: drop-shadow(0 0 4px var(--wheel-paper));
  }

  @keyframes notch-in {
    from {
      opacity: 0;
      scale: 0;
    }
  }

  /* ── diagrama ── */
  .shape {
    fill: none;
    stroke: var(--c);
    stroke-width: 3;
    stroke-linecap: round;
    stroke-linejoin: round;
    opacity: 0.92;
    transform-origin: 0 0;
    transition: opacity 360ms ease, stroke-width 360ms ease;
  }

  .shape polygon,
  .shape > path:first-child {
    fill: none;
  }

  .shape .chev {
    fill: none;
    stroke-width: 3.2;
    stroke-dasharray: none;
  }

  .shape.off {
    opacity: 0.2;
  }

  .shape.on {
    stroke-width: 4.6;
    opacity: 1;
    animation: shape-pop 720ms cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  }

  .shape.on polygon,
  .shape.on > path:first-child:not(.chev) {
    animation: march 1.1s linear infinite;
  }

  .shape.on .chev {
    animation: chev 1.2s ease-in-out infinite;
  }

  .entered .diagram {
    animation: diagram-in 900ms cubic-bezier(0.34, 1.56, 0.64, 1) 350ms backwards;
  }

  @keyframes diagram-in {
    from {
      opacity: 0;
      scale: 0.3;
      rotate: -40deg;
    }
  }

  @keyframes shape-pop {
    from {
      scale: 0.55;
      opacity: 0;
    }
  }

  @keyframes march {
    to {
      stroke-dashoffset: -52;
    }
  }

  @keyframes chev {
    50% {
      opacity: 0.45;
    }
  }
</style>
