// Teoria de cor para a Roda Cromática — matemática pura, sem dependências.
//
// O ponto pedagógico: clarear/escurecer "do jeito comum" (misturar branco ou
// preto puro) desbota e suja a cor. O jeito certo do pintor é DESLOCAR O MATIZ:
//   • luz  → puxa o matiz pro amarelo (polo claro) e sobe o valor;
//   • sombra → puxa o matiz pro azul-violeta (polo frio) e desce o valor,
//              mantendo o croma vivo.
// As funções abaixo geram as duas rampas (certa e comum) lado a lado, além das
// harmonias clássicas da roda.
//
// IMPORTANTE: este arquivo é copiado idêntico em frontend-mobile/src/lib/color/
// theory.ts — os dois frontends são projetos Vite separados. Manter em sincronia.

export interface RGB {
  r: number;
  g: number;
  b: number;
}

export interface HSL {
  h: number; // 0..360
  s: number; // 0..1
  l: number; // 0..1
}

/** Polo claro: pra onde o matiz caminha ao clarear (amarelo quente). */
const LIGHT_POLE = 52;
/** Polo de sombra: pra onde o matiz caminha ao escurecer (azul-violeta frio). */
const SHADOW_POLE = 250;

// ---------------------------------------------------------------------------
// Conversões
// ---------------------------------------------------------------------------

export function clamp(v: number, min: number, max: number): number {
  return v < min ? min : v > max ? max : v;
}

export function rgbToHsl({ r, g, b }: RGB): HSL {
  const rn = r / 255,
    gn = g / 255,
    bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const d = max - min;
  let h = 0;
  if (d !== 0) {
    if (max === rn) h = ((gn - bn) / d) % 6;
    else if (max === gn) h = (bn - rn) / d + 2;
    else h = (rn - gn) / d + 4;
    h *= 60;
    if (h < 0) h += 360;
  }
  const l = (max + min) / 2;
  const s = d === 0 ? 0 : d / (1 - Math.abs(2 * l - 1));
  return { h, s, l };
}

export function hslToRgb({ h, s, l }: HSL): RGB {
  const c = (1 - Math.abs(2 * l - 1)) * s;
  const hp = ((h % 360) + 360) % 360 / 60;
  const x = c * (1 - Math.abs((hp % 2) - 1));
  let r1 = 0,
    g1 = 0,
    b1 = 0;
  if (hp < 1) [r1, g1, b1] = [c, x, 0];
  else if (hp < 2) [r1, g1, b1] = [x, c, 0];
  else if (hp < 3) [r1, g1, b1] = [0, c, x];
  else if (hp < 4) [r1, g1, b1] = [0, x, c];
  else if (hp < 5) [r1, g1, b1] = [x, 0, c];
  else [r1, g1, b1] = [c, 0, x];
  const m = l - c / 2;
  return {
    r: Math.round((r1 + m) * 255),
    g: Math.round((g1 + m) * 255),
    b: Math.round((b1 + m) * 255),
  };
}

export function rgbToHex({ r, g, b }: RGB): string {
  return (
    '#' +
    [r, g, b].map(n => clamp(Math.round(n), 0, 255).toString(16).padStart(2, '0')).join('').toUpperCase()
  );
}

export function hexToRgb(hex: string): RGB | null {
  const m = hex.trim().replace(/^#/, '');
  if (!/^[0-9a-fA-F]{6}$/.test(m)) return null;
  return {
    r: parseInt(m.slice(0, 2), 16),
    g: parseInt(m.slice(2, 4), 16),
    b: parseInt(m.slice(4, 6), 16),
  };
}

// ---------------------------------------------------------------------------
// Matiz
// ---------------------------------------------------------------------------

/** Menor diferença angular assinada de `from` para `to`, em graus (-180..180). */
export function hueDelta(from: number, to: number): number {
  return (((to - from + 540) % 360) - 180);
}

/** Move `h` em direção a `pole` pelo caminho mais curto, no máximo `deg` graus. */
export function shiftHueToward(h: number, pole: number, deg: number): number {
  const d = hueDelta(h, pole);
  const move = Math.sign(d) * Math.min(deg, Math.abs(d));
  return (((h + move) % 360) + 360) % 360;
}

// ---------------------------------------------------------------------------
// Rampas de luz e sombra
// ---------------------------------------------------------------------------

export type StepKind = 'shadow' | 'base' | 'highlight';

export interface RampStep {
  /** Negativo = sombra, 0 = base, positivo = highlight. */
  level: number;
  kind: StepKind;
  hsl: HSL;
  rgb: RGB;
  hex: string;
  /** Explicação curta do que mudou nesse passo (jeito certo). */
  why: string;
}

export interface RampOptions {
  highlights?: number;
  shadows?: number;
}

const TOP_L = 0.9; // teto de luminosidade dos highlights
const BOTTOM_L = 0.13; // piso das sombras
const HL_HUE_MAX = 26; // deslocamento máximo de matiz nos highlights (graus)
const SH_HUE_MAX = 22; // deslocamento máximo de matiz nas sombras (graus)

function poleName(pole: number): string {
  return pole === LIGHT_POLE ? 'amarelo' : 'azul';
}

/** Rampa CORRETA: highlight desloca o matiz pro amarelo + sobe luz; sombra
 *  desloca pro azul-violeta + desce luz, preservando o croma. Ordenada do fundo
 *  (sombra mais escura) ao topo (highlight mais claro). */
export function buildRamp(base: RGB, opts: RampOptions = {}): RampStep[] {
  const highlights = opts.highlights ?? 3;
  const shadows = opts.shadows ?? 2;
  const hsl = rgbToHsl(base);
  const steps: RampStep[] = [];

  for (let k = shadows; k >= 1; k--) {
    const t = k / shadows;
    const l = hsl.l - (hsl.l - BOTTOM_L) * t;
    const deg = SH_HUE_MAX * t;
    const h = shiftHueToward(hsl.h, SHADOW_POLE, deg);
    const s = clamp(hsl.s * (1 + 0.06 * t), 0, 1);
    const stepHsl = { h, s, l };
    const rgb = hslToRgb(stepHsl);
    steps.push({
      level: -k,
      kind: 'shadow',
      hsl: stepHsl,
      rgb,
      hex: rgbToHex(rgb),
      why: `Matiz +${Math.round(deg)}° pro ${poleName(SHADOW_POLE)} · luz ↓`,
    });
  }

  steps.push({
    level: 0,
    kind: 'base',
    hsl,
    rgb: base,
    hex: rgbToHex(base),
    why: 'Cor base',
  });

  for (let k = 1; k <= highlights; k++) {
    const t = k / highlights;
    const l = hsl.l + (TOP_L - hsl.l) * t;
    const deg = HL_HUE_MAX * t;
    const h = shiftHueToward(hsl.h, LIGHT_POLE, deg);
    const s = clamp(hsl.s * (1 - 0.18 * t * t), 0, 1);
    const stepHsl = { h, s, l };
    const rgb = hslToRgb(stepHsl);
    steps.push({
      level: k,
      kind: 'highlight',
      hsl: stepHsl,
      rgb,
      hex: rgbToHex(rgb),
      why: `Matiz +${Math.round(deg)}° pro ${poleName(LIGHT_POLE)} · luz ↑`,
    });
  }

  return steps;
}

/** Rampa COMUM (o jeito errado): só mistura branco (clarear) ou preto (escurecer)
 *  em RGB, sem mexer no matiz. Serve de contraste — fica leitosa/suja. */
export function buildNaiveRamp(base: RGB, opts: RampOptions = {}): RGB[] {
  const highlights = opts.highlights ?? 3;
  const shadows = opts.shadows ?? 2;
  const out: RGB[] = [];
  const lerp = (a: number, b: number, t: number) => Math.round(a + (b - a) * t);

  for (let k = shadows; k >= 1; k--) {
    const t = (k / shadows) * 0.72;
    out.push({ r: lerp(base.r, 0, t), g: lerp(base.g, 0, t), b: lerp(base.b, 0, t) });
  }
  out.push({ ...base });
  for (let k = 1; k <= highlights; k++) {
    const t = (k / highlights) * 0.72;
    out.push({ r: lerp(base.r, 255, t), g: lerp(base.g, 255, t), b: lerp(base.b, 255, t) });
  }
  return out;
}

// ---------------------------------------------------------------------------
// Nome da cor (aproximado, em PT) — pra identificar uma cor arbitrária
// ---------------------------------------------------------------------------

/** Nome aproximado da cor em português (VERMELHO, AZUL, MARROM…). Serve pra
 *  identificar uma cor calculada que não é uma tinta de catálogo. */
export function colorName(rgb: RGB): string {
  const { h, s, l } = rgbToHsl(rgb);

  if (l >= 0.93) return 'BRANCO';
  if (l <= 0.07) return 'PRETO';
  if (s <= 0.12) return l < 0.35 ? 'CINZA-ESCURO' : l > 0.72 ? 'CINZA-CLARO' : 'CINZA';

  // Casos especiais: marrom (laranja escuro) e rosa (vermelho/magenta claro).
  if (h < 42 && l < 0.4 && s > 0.2) return 'MARROM';
  if ((h < 15 || h >= 328) && l > 0.72) return 'ROSA';

  const base =
    h < 15 ? 'VERMELHO' :
    h < 42 ? 'LARANJA' :
    h < 68 ? 'AMARELO' :
    h < 90 ? 'AMARELO-ESVERDEADO' :
    h < 160 ? 'VERDE' :
    h < 195 ? 'CIANO' :
    h < 255 ? 'AZUL' :
    h < 290 ? 'ROXO' :
    h < 328 ? 'MAGENTA' :
    'VERMELHO';

  if (l < 0.3) return `${base} ESCURO`;
  if (l > 0.76) return `${base} CLARO`;
  return base;
}

// ---------------------------------------------------------------------------
// Harmonias da roda cromática
// ---------------------------------------------------------------------------

export type HarmonyKind = 'complementary' | 'analogous' | 'triad' | 'split';

export interface HarmonySwatch {
  h: number;
  rgb: RGB;
  hex: string;
  role: string;
}

const HARMONY_OFFSETS: Record<HarmonyKind, number[]> = {
  complementary: [0, 180],
  analogous: [-30, 0, 30],
  triad: [0, 120, 240],
  split: [0, 150, 210],
};

export const HARMONY_LABEL: Record<HarmonyKind, string> = {
  complementary: 'Complementar',
  analogous: 'Análogas',
  triad: 'Tríade',
  split: 'Split',
};

/** Descrição pedagógica de cada harmonia. */
export const HARMONY_HINT: Record<HarmonyKind, string> = {
  complementary:
    'Opostos na roda. Contraste máximo — e a chave pra escurecer: um toque do complementar tira o brilho sem sujar como o preto.',
  analogous: 'Vizinhas na roda. Combinam de forma calma e natural, ótimas pra transições suaves.',
  triad: 'Três cores igualmente espaçadas. Paleta vibrante e equilibrada.',
  split: 'A cor e as duas vizinhas do seu complementar. Contraste forte, porém mais harmônico.',
};

export function harmony(baseHsl: HSL, kind: HarmonyKind): HarmonySwatch[] {
  const roles: Record<HarmonyKind, string[]> = {
    complementary: ['Base', 'Complementar'],
    analogous: ['−30°', 'Base', '+30°'],
    triad: ['Base', '+120°', '+240°'],
    split: ['Base', '+150°', '+210°'],
  };
  return HARMONY_OFFSETS[kind].map((off, i) => {
    const h = (((baseHsl.h + off) % 360) + 360) % 360;
    const hsl = { h, s: Math.max(baseHsl.s, 0.55), l: clamp(baseHsl.l, 0.4, 0.6) };
    const rgb = hslToRgb(hsl);
    return { h, rgb, hex: rgbToHex(rgb), role: roles[kind][i] };
  });
}
