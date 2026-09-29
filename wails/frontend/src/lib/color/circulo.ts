// Círculo cromático do pintor — modelo puro, sem Svelte e sem DOM.
//
// Réplica digital do disco giratório de papel (referência: temp/IMG_0005–0007):
//   • anel externo FIXO com 12 matizes de pigmento (roda RYB do pintor, não a
//     roda RGB da tela — o complementar do amarelo é o azul-violetado, não o
//     azul);
//   • disco interno GIRATÓRIO com janelas que revelam a escala de cada matiz
//     (+claro, +suave, +profundo; o +brilhante é o próprio anel), duas escalas
//     de cinza (10–50 % e 60–100 %) e o diagrama das harmonias no miolo.
//
// Toda a geometria das harmonias é contada em "graus de roda" (0° = topo,
// sentido horário, 30° por matiz). Uma cor qualquer entra na roda pelo mapa
// `hslHueToWheel`, que interpola entre os 12 matizes do anel — é isso que faz
// a cor do usuário cair no ponto certo entre, por exemplo, Laranja e Amarelo.

import { clamp, hexToRgb, hslToRgb, rgbToHex, rgbToHsl, type HSL, type RGB } from './theory';

// ---------------------------------------------------------------------------
// Anel de matizes
// ---------------------------------------------------------------------------

export type HueId =
  | 'magenta'
  | 'violeta'
  | 'azulVioletado'
  | 'azul'
  | 'azulCiano'
  | 'turquesa'
  | 'verde'
  | 'verdeLima'
  | 'amarelo'
  | 'laranja'
  | 'vermelho'
  | 'rosa';

export interface WheelHue {
  id: HueId;
  /** posição no anel: índice × 30° a partir do topo, sentido horário */
  index: number;
  /** cor impressa do anel (amostrada das fotos do disco, corrigida da luz) */
  hex: string;
  rgb: RGB;
  warm: boolean;
  /** primária/secundária/terciária na roda do pintor (RYB estendida) */
  order: 'primaria' | 'secundaria' | 'terciaria';
}

// Ordem do anel, do topo em sentido horário — a mesma das fotos.
// Quentes: Amarelo → Laranja → Vermelho → Rosa → Magenta → Violeta.
// Frias: Azul-violetado → Azul → Azul-ciano → Turquesa → Verde → Verde-lima.
const RING: Array<[HueId, string, boolean, WheelHue['order']]> = [
  ['magenta', '#CC026A', true, 'secundaria'],
  ['violeta', '#76176C', true, 'terciaria'],
  ['azulVioletado', '#2F2B7E', false, 'secundaria'],
  ['azul', '#0157BC', false, 'primaria'],
  ['azulCiano', '#019DF0', false, 'terciaria'],
  ['turquesa', '#00979A', false, 'secundaria'],
  ['verde', '#03913F', false, 'terciaria'],
  ['verdeLima', '#72B01E', false, 'secundaria'],
  ['amarelo', '#F7E014', true, 'primaria'],
  ['laranja', '#E8731C', true, 'secundaria'],
  ['vermelho', '#D0142C', true, 'primaria'],
  ['rosa', '#D0024C', true, 'terciaria'],
];

export const WHEEL: WheelHue[] = RING.map(([id, hex, warm, order], index) => ({
  id,
  index,
  hex,
  rgb: hexToRgb(hex)!,
  warm,
  order,
}));

export const STEP = 30;

export function norm360(a: number): number {
  return ((a % 360) + 360) % 360;
}

/** Índice do matiz do anel mais perto de um ângulo de roda. */
export function nearestIndex(wheelDeg: number): number {
  return Math.round(norm360(wheelDeg) / STEP) % 12;
}

/** Fronteiras quente/frio no anel, em graus de roda: entre Violeta e
 *  Azul-violetado (45°) e entre Verde-lima e Amarelo (225°). */
export const WARM_COOL_BOUNDARIES = [45, 225] as const;

export function isWarm(wheelDeg: number): boolean {
  const a = norm360(wheelDeg);
  return !(a > 45 && a < 225);
}

// ---------------------------------------------------------------------------
// Mapa matiz HSL ↔ grau de roda
// ---------------------------------------------------------------------------
// O matiz HSL de cada cor do anel DESCE à medida que o grau de roda sobe
// (Magenta 329° → Violeta 306° → … → Rosa 338°−360°). Desenrolado, vira uma
// função monótona que a interpolação linear inverte sem ambiguidade.

interface Anchor {
  wheel: number;
  hue: number;
}

const ANCHORS: Anchor[] = (() => {
  const out: Anchor[] = [];
  let prev = Infinity;
  for (const h of WHEEL) {
    let hue = rgbToHsl(h.rgb).h;
    // desenrola: cada âncora tem matiz menor que a anterior
    while (hue > prev) hue -= 360;
    out.push({ wheel: h.index * STEP, hue });
    prev = hue;
  }
  // fecha o laço: Magenta de novo em 360° de roda, 360° de matiz abaixo
  out.push({ wheel: 360, hue: out[0].hue - 360 });
  return out;
})();

/** Matiz HSL (0–360) → grau de roda (0–360). */
export function hslHueToWheel(hue: number): number {
  // leva o matiz para a faixa (âncora final, âncora inicial]
  const top = ANCHORS[0].hue;
  let h = hue;
  while (h > top) h -= 360;
  while (h <= top - 360) h += 360;
  for (let i = 0; i < ANCHORS.length - 1; i++) {
    const a = ANCHORS[i];
    const b = ANCHORS[i + 1];
    if (h <= a.hue && h >= b.hue) {
      const t = a.hue === b.hue ? 0 : (a.hue - h) / (a.hue - b.hue);
      return norm360(a.wheel + t * (b.wheel - a.wheel));
    }
  }
  return 0;
}

/** Grau de roda → matiz HSL (0–360). */
export function wheelToHslHue(wheelDeg: number): number {
  const w = norm360(wheelDeg);
  for (let i = 0; i < ANCHORS.length - 1; i++) {
    const a = ANCHORS[i];
    const b = ANCHORS[i + 1];
    if (w >= a.wheel && w <= b.wheel) {
      const t = (w - a.wheel) / (b.wheel - a.wheel);
      return norm360(a.hue + t * (b.hue - a.hue));
    }
  }
  return norm360(ANCHORS[0].hue);
}

/** Cor pura do anel num grau qualquer — mistura as duas vizinhas impressas.
 *  Nos múltiplos de 30° devolve exatamente a cor do anel. */
export function ringColorAt(wheelDeg: number): RGB {
  const w = norm360(wheelDeg);
  const i = Math.floor(w / STEP) % 12;
  const t = (w - i * STEP) / STEP;
  if (t < 1e-6) return { ...WHEEL[i].rgb };
  return mixPaint(WHEEL[i].rgb, WHEEL[(i + 1) % 12].rgb, t);
}

/** Onde uma cor qualquer cai na roda. Neutro (sem croma) não tem matiz: null. */
export function wheelDegOf(rgb: RGB): number | null {
  const hsl = rgbToHsl(rgb);
  if (hsl.s < 0.08 || hsl.l < 0.04 || hsl.l > 0.97) return null;
  return hslHueToWheel(hsl.h);
}

/** A mesma cor (mesma saturação e luminosidade HSL) girada para outro ponto
 *  da roda. É o que faz a paleta de harmonia RESPEITAR a saturação e o valor
 *  da cor de partida: um verde-musgo apagado gera um vermelho igualmente
 *  apagado, não um vermelho de anel. */
export function rotateColor(rgb: RGB, fromWheel: number, toWheel: number): RGB {
  const hsl = rgbToHsl(rgb);
  const delta = hslHueToWheel(hsl.h) - fromWheel;
  const target = wheelToHslHue(toWheel + delta);
  return hslToRgb({ h: target, s: hsl.s, l: hsl.l });
}

// ---------------------------------------------------------------------------
// Mistura de pigmento (aproximação subtrativa leve)
// ---------------------------------------------------------------------------
// A tela mistura luz; a paleta mistura pigmento. A média ponderada GEOMÉTRICA
// da refletância linear (em vez da média aritmética em sRGB) imita o que o
// pigmento faz: azul + amarelo dá verde, e o complementar escurece e
// neutraliza em vez de dar um cinza-claro leitoso.

function toLinear(c: number): number {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function fromLinear(v: number): number {
  const c = v <= 0.0031308 ? v * 12.92 : 1.055 * v ** (1 / 2.4) - 0.055;
  return clamp(Math.round(c * 255), 0, 255);
}

/** Mistura `t` (0–1) de `b` em `a`, à moda do pigmento. */
export function mixPaint(a: RGB, b: RGB, t: number): RGB {
  const k = clamp(t, 0, 1);
  const f = (x: number, y: number) => {
    const lx = Math.max(toLinear(x), 1e-4);
    const ly = Math.max(toLinear(y), 1e-4);
    return fromLinear(Math.exp((1 - k) * Math.log(lx) + k * Math.log(ly)));
  };
  return { r: f(a.r, b.r), g: f(a.g, b.g), b: f(a.b, b.b) };
}

/** Mistura linear (tinta branca, cinza e preta se comportam quase assim). */
export function mixLinear(a: RGB, b: RGB, t: number): RGB {
  const k = clamp(t, 0, 1);
  const f = (x: number, y: number) => fromLinear(toLinear(x) * (1 - k) + toLinear(y) * k);
  return { r: f(a.r, b.r), g: f(a.g, b.g), b: f(a.b, b.b) };
}

export const WHITE: RGB = { r: 246, g: 246, b: 242 };
export const BLACK: RGB = { r: 24, g: 25, b: 31 };

/** Sombra com tinta preta: o pigmento preto domina rápido (por isso a mistura
 *  é a geométrica) e, sendo levemente frio, esverdeia os amarelos. */
export function withBlack(base: RGB, t: number): RGB {
  return mixPaint(base, BLACK, clamp(t, 0, 1) * 0.5);
}

/** Neutralizar com o complementar. Mistura de pigmento em 3 canais (RGB)
 *  erra o caminho (azul + laranja sai esverdeado), então o caminho é traçado
 *  no CIELAB: o croma cai para um "cinza colorido" (12 % do croma, ainda do
 *  lado da cor-base) e o valor desce até onde a mistura de pigmento desce. */
export function withComplement(base: RGB, comp: RGB, t: number): RGB {
  const k = clamp(t, 0, 1);
  const lab = rgbToLab(base);
  const floorL = Math.min(lab.L, valueOf(mixPaint(base, comp, 0.45)));
  const L = lab.L + (floorL - lab.L) * k;
  const keep = 1 - 0.88 * k;
  return labToRgb({ L, a: lab.a * keep, b: lab.b * keep });
}

/** Tom (cinza do mesmo valor): o croma cai, o valor fica. */
export function withGray(base: RGB, t: number): RGB {
  const lab = rgbToLab(base);
  const keep = 1 - clamp(t, 0, 1);
  return labToRgb({ L: lab.L, a: lab.a * keep, b: lab.b * keep });
}

// ---------------------------------------------------------------------------
// Medidas: valor (L*), croma e ΔE00
// ---------------------------------------------------------------------------

export interface Lab {
  L: number;
  a: number;
  b: number;
}

export function rgbToLab({ r, g, b }: RGB): Lab {
  const R = toLinear(r);
  const G = toLinear(g);
  const B = toLinear(b);
  const X = (R * 0.4124564 + G * 0.3575761 + B * 0.1804375) / 0.95047;
  const Y = R * 0.2126729 + G * 0.7151522 + B * 0.072175;
  const Z = (R * 0.0193339 + G * 0.119192 + B * 0.9503041) / 1.08883;
  const f = (t: number) => (t > 216 / 24389 ? Math.cbrt(t) : (24389 / 27 * t + 16) / 116);
  const fx = f(X);
  const fy = f(Y);
  const fz = f(Z);
  return { L: 116 * fy - 16, a: 500 * (fx - fy), b: 200 * (fy - fz) };
}

export function labToRgb({ L, a, b }: Lab): RGB {
  const fy = (L + 16) / 116;
  const fx = fy + a / 500;
  const fz = fy - b / 200;
  const inv = (f: number) => (f ** 3 > 216 / 24389 ? f ** 3 : (116 * f - 16) / (24389 / 27));
  const X = inv(fx) * 0.95047;
  const Y = inv(fy);
  const Z = inv(fz) * 1.08883;
  const R = X * 3.2404542 - Y * 1.5371385 - Z * 0.4985314;
  const G = -X * 0.969266 + Y * 1.8760108 + Z * 0.041556;
  const B = X * 0.0556434 - Y * 0.2040259 + Z * 1.0572252;
  return { r: fromLinear(clamp(R, 0, 1)), g: fromLinear(clamp(G, 0, 1)), b: fromLinear(clamp(B, 0, 1)) };
}

/** Valor (luminosidade percebida) de 0 (preto) a 100 (branco). */
export function valueOf(rgb: RGB): number {
  return rgbToLab(rgb).L;
}

/** Croma CIELAB — "quanto de cor" a mistura ainda carrega. */
export function chromaOf(rgb: RGB): number {
  const { a, b } = rgbToLab(rgb);
  return Math.hypot(a, b);
}

/** Cinza neutro com o mesmo valor (L*) da cor. */
export function grayOfSameValue(rgb: RGB): RGB {
  const L = valueOf(rgb);
  const fy = (L + 16) / 116;
  const Y = fy ** 3 > 216 / 24389 ? fy ** 3 : (116 * fy - 16) / (24389 / 27);
  const c = fromLinear(Y);
  return { r: c, g: c, b: c };
}

/** Degrau da escala de cinza do disco (10 %–100 % de escuro) que mais se
 *  parece com o valor da cor. */
export function grayStepOf(rgb: RGB): number {
  const dark = 100 - valueOf(rgb);
  return clamp(Math.round(dark / 10) * 10, 10, 100);
}

/** ΔE00 (CIEDE2000) — a mesma régua que o resto da Mescla usa. */
export function deltaE00(c1: RGB, c2: RGB): number {
  const l1 = rgbToLab(c1);
  const l2 = rgbToLab(c2);
  const rad = Math.PI / 180;
  const C1 = Math.hypot(l1.a, l1.b);
  const C2 = Math.hypot(l2.a, l2.b);
  const Cm = (C1 + C2) / 2;
  const G = 0.5 * (1 - Math.sqrt(Cm ** 7 / (Cm ** 7 + 25 ** 7)));
  const a1 = (1 + G) * l1.a;
  const a2 = (1 + G) * l2.a;
  const C1p = Math.hypot(a1, l1.b);
  const C2p = Math.hypot(a2, l2.b);
  const h1 = C1p === 0 ? 0 : norm360(Math.atan2(l1.b, a1) / rad);
  const h2 = C2p === 0 ? 0 : norm360(Math.atan2(l2.b, a2) / rad);
  const dL = l2.L - l1.L;
  const dC = C2p - C1p;
  let dh = 0;
  if (C1p * C2p !== 0) {
    dh = h2 - h1;
    if (dh > 180) dh -= 360;
    else if (dh < -180) dh += 360;
  }
  const dH = 2 * Math.sqrt(C1p * C2p) * Math.sin((dh * rad) / 2);
  const Lm = (l1.L + l2.L) / 2;
  const Cpm = (C1p + C2p) / 2;
  let hm = h1 + h2;
  if (C1p * C2p !== 0) {
    if (Math.abs(h1 - h2) > 180) hm = h1 + h2 < 360 ? (h1 + h2 + 360) / 2 : (h1 + h2 - 360) / 2;
    else hm = (h1 + h2) / 2;
  }
  const T =
    1 -
    0.17 * Math.cos((hm - 30) * rad) +
    0.24 * Math.cos(2 * hm * rad) +
    0.32 * Math.cos((3 * hm + 6) * rad) -
    0.2 * Math.cos((4 * hm - 63) * rad);
  const dTheta = 30 * Math.exp(-(((hm - 275) / 25) ** 2));
  const Rc = 2 * Math.sqrt(Cpm ** 7 / (Cpm ** 7 + 25 ** 7));
  const Sl = 1 + (0.015 * (Lm - 50) ** 2) / Math.sqrt(20 + (Lm - 50) ** 2);
  const Sc = 1 + 0.045 * Cpm;
  const Sh = 1 + 0.015 * Cpm * T;
  const Rt = -Math.sin(2 * dTheta * rad) * Rc;
  return Math.sqrt(
    (dL / Sl) ** 2 + (dC / Sc) ** 2 + (dH / Sh) ** 2 + Rt * (dC / Sc) * (dH / Sh)
  );
}

// ---------------------------------------------------------------------------
// Janelas do disco: escala monocromática
// ---------------------------------------------------------------------------
// Da borda de dentro para fora, como na legenda impressa:
//   +CLARO    (branco adicionado) — 2 faixas
//   +SUAVE    (tom misturado: cinza do mesmo valor) — 1 faixa
//   +PROFUNDO (preto adicionado) — 2 faixas
//   +BRILHANTE (tom saturado) — é o anel, fora da janela.

export type BandKind = 'claro' | 'suave' | 'profundo' | 'brilhante';

export interface Band {
  kind: BandKind;
  rgb: RGB;
  hex: string;
}

export function monoBands(base: RGB): Band[] {
  const bands: Array<[BandKind, RGB]> = [
    ['claro', mixLinear(base, WHITE, 0.62)],
    ['claro', mixLinear(base, WHITE, 0.32)],
    ['suave', withGray(base, 0.55)],
    ['profundo', withBlack(base, 0.32)],
    ['profundo', withBlack(base, 0.62)],
  ];
  return bands.map(([kind, rgb]) => ({ kind, rgb, hex: rgbToHex(rgb) }));
}

/** Escalas de cinza do disco, em % de escuro. A janela clara vai de 50 % (borda
 *  de dentro) a 10 % (borda de fora); a escura de 60 % a 100 %. */
export const GRAY_LIGHT = [50, 40, 30, 20, 10];
export const GRAY_DARK = [60, 70, 80, 90, 100];

export function grayAt(darkPct: number): RGB {
  const L = 100 - darkPct;
  const fy = (L + 16) / 116;
  const Y = fy ** 3 > 216 / 24389 ? fy ** 3 : (116 * fy - 16) / (24389 / 27);
  const c = fromLinear(Math.max(Y, 0.0045));
  return { r: c, g: c, b: c };
}

// ---------------------------------------------------------------------------
// Harmonias
// ---------------------------------------------------------------------------

export type SchemeId =
  | 'complementar'
  | 'analoga'
  | 'triade'
  | 'complementarDividido'
  | 'tetrade'
  | 'complementarDupla'
  | 'monocromatico';

export interface Scheme {
  id: SchemeId;
  /** posições em graus de roda relativas à cor-base (0 = a base) */
  offsets: number[];
  /** traço do diagrama no miolo, como impresso: contínuo, tracejado, pontilhado */
  stroke: 'solid' | 'dash' | 'longdash' | 'dot';
  /** token CSS da cor do traço e do rótulo (app.css) */
  token: string;
}

export const SCHEMES: Scheme[] = [
  { id: 'complementar', offsets: [0, 180], stroke: 'solid', token: '--wheel-complementar' },
  { id: 'analoga', offsets: [-30, 0, 30], stroke: 'dot', token: '--wheel-analoga' },
  { id: 'triade', offsets: [0, 120, 240], stroke: 'longdash', token: '--wheel-triade' },
  { id: 'complementarDividido', offsets: [0, 150, 210], stroke: 'dot', token: '--wheel-dividido' },
  { id: 'tetrade', offsets: [0, 90, 180, 270], stroke: 'dot', token: '--wheel-tetrade' },
  { id: 'complementarDupla', offsets: [0, 60, 180, 240], stroke: 'dash', token: '--wheel-dupla' },
  { id: 'monocromatico', offsets: [0], stroke: 'solid', token: '--wheel-mono' },
];

export function schemeById(id: SchemeId): Scheme {
  return SCHEMES.find(s => s.id === id)!;
}

/** Esquemas (exceto o monocromático) que marcam cada posição relativa do disco,
 *  na ordem da lista — é o que vai impresso junto de cada entalhe. */
export function schemesAtOffset(offset: number): Scheme[] {
  const o = norm360(offset);
  return SCHEMES.filter(s => s.id !== 'monocromatico' && s.offsets.some(x => norm360(x) === o));
}

export interface HarmonySwatch {
  /** grau de roda absoluto */
  wheel: number;
  /** posição relativa à base (0, 180, −30…) */
  offset: number;
  rgb: RGB;
  hex: string;
  /** matiz do anel mais perto */
  hue: WheelHue;
}

/** Cores de uma harmonia partindo de uma base. Com `base` do anel, as cores são
 *  as do anel; com a cor do usuário, cada uma herda a saturação e a
 *  luminosidade dela (rotateColor). */
export function harmonyOf(scheme: Scheme, baseWheel: number, base: RGB, fromRing: boolean): HarmonySwatch[] {
  if (scheme.id === 'monocromatico') {
    return [0].map(offset => ({
      wheel: norm360(baseWheel),
      offset,
      rgb: base,
      hex: rgbToHex(base),
      hue: WHEEL[nearestIndex(baseWheel)],
    }));
  }
  return scheme.offsets.map(offset => {
    const wheel = norm360(baseWheel + offset);
    const rgb = offset === 0 ? base : fromRing ? ringColorAt(wheel) : rotateColor(base, baseWheel, wheel);
    return { wheel, offset, rgb, hex: rgbToHex(rgb), hue: WHEEL[nearestIndex(wheel)] };
  });
}

// ---------------------------------------------------------------------------
// Escurecer, clarear e dessaturar — os caminhos do pintor lado a lado
// ---------------------------------------------------------------------------

export interface Path {
  id: 'preto' | 'complementar' | 'matiz' | 'branco' | 'vizinha' | 'cinza';
  steps: RGB[];
}

const LEVELS = [0, 0.2, 0.4, 0.6, 0.8];

/** Três caminhos para escurecer, do 0 (a cor) ao mais escuro:
 *   • preto — sombra "de manual": suja e acinzenta;
 *   • complementar — neutraliza e escurece mantendo a cor viva por dentro;
 *   • matiz — desloca para o polo frio (azul-violeta) e desce o valor. */
export function darkenPaths(base: RGB, baseWheel: number): Path[] {
  const comp = ringColorAt(baseWheel + 180);
  const hsl = rgbToHsl(base);
  // polo de sombra da roda: Azul-violetado (60°)
  const towardCool = (t: number): RGB => {
    const w = hslHueToWheel(hsl.h);
    let d = ((60 - w + 540) % 360) - 180;
    // Amarelo e vizinhos: o caminho curto para o frio passa pelo verde e
    // "adoece" a sombra — o pintor escurece o amarelo pelo ocre/laranja.
    if (Math.abs(d) >= 150) d = Math.abs(d);
    const moved = wheelToHslHue(w + Math.sign(d) * Math.min(Math.abs(d), 40 * t));
    return hslToRgb({ h: moved, s: clamp(hsl.s * (1 + 0.05 * t), 0, 1), l: hsl.l * (1 - 0.72 * t) });
  };
  return [
    { id: 'preto', steps: LEVELS.map(t => withBlack(base, t * 1.25)) },
    { id: 'complementar', steps: LEVELS.map(t => withComplement(base, comp, t * 1.25)) },
    { id: 'matiz', steps: LEVELS.map(t => towardCool(t)) },
  ];
}

/** Três caminhos para clarear:
 *   • branco — tinta pastel, esfria e desbota (efeito Abney);
 *   • vizinha — branco + uma pitada da vizinha quente, que devolve o matiz;
 *   • matiz — desloca para o polo de luz (amarelo) e sobe o valor. */
export function lightenPaths(base: RGB, baseWheel: number): Path[] {
  const hsl = rgbToHsl(base);
  const w = hslHueToWheel(hsl.h);
  // a vizinha na direção do amarelo (240°)
  const dirWarm = Math.sign(((240 - w + 540) % 360) - 180) || 1;
  const neighbor = ringColorAt(baseWheel + dirWarm * 30);
  const towardLight = (t: number): RGB => {
    const d = ((240 - w + 540) % 360) - 180;
    const moved = wheelToHslHue(w + Math.sign(d) * Math.min(Math.abs(d), 32 * t));
    return hslToRgb({ h: moved, s: clamp(hsl.s * (1 - 0.15 * t * t), 0, 1), l: hsl.l + (0.9 - hsl.l) * t });
  };
  return [
    { id: 'branco', steps: LEVELS.map(t => mixLinear(base, WHITE, t * 0.85)) },
    {
      id: 'vizinha',
      steps: LEVELS.map(t => mixPaint(mixLinear(base, WHITE, t * 0.85), neighbor, t * 0.12)),
    },
    { id: 'matiz', steps: LEVELS.map(t => towardLight(t)) },
  ];
}

/** Dois caminhos para baixar a saturação (croma):
 *   • cinza do mesmo valor — o valor fica, só a cor sai;
 *   • complementar — a cor sai e o valor desce junto. */
export function desaturatePaths(base: RGB, baseWheel: number): Path[] {
  const comp = ringColorAt(baseWheel + 180);
  return [
    { id: 'cinza', steps: LEVELS.map(t => withGray(base, t * 1.25)) },
    { id: 'complementar', steps: LEVELS.map(t => withComplement(base, comp, t * 1.25)) },
  ];
}

/** Cor com a saturação HSL trocada — para o controle "saturação" do painel. */
export function withSaturation(rgb: RGB, s: number): RGB {
  const hsl: HSL = rgbToHsl(rgb);
  return hslToRgb({ ...hsl, s: clamp(s, 0, 1) });
}

/** Texto legível sobre um fundo (tinta clara → texto escuro). */
export function inkOn(rgb: RGB): 'dark' | 'light' {
  return valueOf(rgb) > 62 ? 'dark' : 'light';
}

export { rgbToHex, hexToRgb, rgbToHsl, hslToRgb };
export type { RGB, HSL };
