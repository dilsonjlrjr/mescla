// rf-04 — matemática de cor do lado do cliente. NÃO é o motor de mistura
// (api/domain/mix/*, api/domain/color/deltae.go) — não decide pool, não faz
// pré-seleção nem testa pares/trios (RG-07 a RG-10 continuam só no servidor).
// Existe só para o que a spec pede explicitamente ao cliente: RG-12 (ajuste
// manual de gotas recalcula "sem re-executar o solver") e RG-23 (ΔE00/veredicto
// ao vivo conforme o idioma). As fórmulas de RG-01/RG-02/RG-04 são as mesmas
// declaradas na spec, para o cliente e o servidor concordarem no que mostram.

export interface RGB {
  r: number;
  g: number;
  b: number;
}

/** RG-01 — sRGB para luz linear. */
function srgbToLinear(v: number): number {
  const s = v / 255;
  return s <= 0.04045 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
}

/** RG-01 — inverso: luz linear para sRGB (0..255). */
function linearToSrgb(v: number): number {
  const s = v <= 0.0031308 ? v * 12.92 : 1.055 * Math.pow(v, 1 / 2.4) - 0.055;
  return Math.max(0, Math.min(255, Math.round(s * 255)));
}

/** RG-02 — Lab (D65) a partir de sRGB 0..255. */
export function rgbToLab(r: number, g: number, b: number): [number, number, number] {
  const rl = srgbToLinear(r);
  const gl = srgbToLinear(g);
  const bl = srgbToLinear(b);

  // Matriz sRGB → XYZ padrão (D65).
  const x = rl * 0.4124564 + gl * 0.3575761 + bl * 0.1804375;
  const y = rl * 0.2126729 + gl * 0.7151522 + bl * 0.072175;
  const z = rl * 0.0193339 + gl * 0.119192 + bl * 0.9503041;

  const Xn = 0.95047, Yn = 1.0, Zn = 1.08883;
  const xn = x / Xn, yn = y / Yn, zn = z / Zn;

  const f = (t: number) => (t > 0.008856 ? Math.cbrt(t) : 7.787 * t + 16 / 116);
  const fx = f(xn), fy = f(yn), fz = f(zn);

  const L = 116 * fy - 16;
  const A = 500 * (fx - fy);
  const B = 200 * (fy - fz);
  return [L, A, B];
}

/** RG-04 — CIEDE2000 completo (kL = kC = kH = 1), termos T e RT inclusos. */
export function deltaE2000(lab1: [number, number, number], lab2: [number, number, number]): number {
  const [L1, a1, b1] = lab1;
  const [L2, a2, b2] = lab2;

  const avgLp = (L1 + L2) / 2;
  const C1 = Math.sqrt(a1 * a1 + b1 * b1);
  const C2 = Math.sqrt(a2 * a2 + b2 * b2);
  const avgC = (C1 + C2) / 2;
  const G = 0.5 * (1 - Math.sqrt(Math.pow(avgC, 7) / (Math.pow(avgC, 7) + Math.pow(25, 7))));

  const a1p = a1 * (1 + G);
  const a2p = a2 * (1 + G);
  const C1p = Math.sqrt(a1p * a1p + b1 * b1);
  const C2p = Math.sqrt(a2p * a2p + b2 * b2);
  const avgCp = (C1p + C2p) / 2;

  const hp = (a: number, bb: number) => {
    if (a === 0 && bb === 0) return 0;
    const h = (Math.atan2(bb, a) * 180) / Math.PI;
    return h < 0 ? h + 360 : h;
  };
  const h1p = hp(a1p, b1);
  const h2p = hp(a2p, b2);

  let deltahp: number;
  if (C1p * C2p === 0) deltahp = 0;
  else if (Math.abs(h1p - h2p) <= 180) deltahp = h2p - h1p;
  else if (h2p <= h1p) deltahp = h2p - h1p + 360;
  else deltahp = h2p - h1p - 360;

  const deltaLp = L2 - L1;
  const deltaCp = C2p - C1p;
  const deltaHp = 2 * Math.sqrt(C1p * C2p) * Math.sin((deltahp * Math.PI) / 360);

  let avgHp: number;
  if (C1p * C2p === 0) avgHp = h1p + h2p;
  else if (Math.abs(h1p - h2p) <= 180) avgHp = (h1p + h2p) / 2;
  else if (h1p + h2p < 360) avgHp = (h1p + h2p + 360) / 2;
  else avgHp = (h1p + h2p - 360) / 2;

  const T =
    1 -
    0.17 * Math.cos(((avgHp - 30) * Math.PI) / 180) +
    0.24 * Math.cos((2 * avgHp * Math.PI) / 180) +
    0.32 * Math.cos(((3 * avgHp + 6) * Math.PI) / 180) -
    0.2 * Math.cos(((4 * avgHp - 63) * Math.PI) / 180);

  const deltaTheta = 30 * Math.exp(-Math.pow((avgHp - 275) / 25, 2));
  const RC = 2 * Math.sqrt(Math.pow(avgCp, 7) / (Math.pow(avgCp, 7) + Math.pow(25, 7)));
  const SL = 1 + (0.015 * Math.pow(avgLp - 50, 2)) / Math.sqrt(20 + Math.pow(avgLp - 50, 2));
  const SC = 1 + 0.045 * avgCp;
  const SH = 1 + 0.015 * avgCp * T;
  const RT = -Math.sin((2 * deltaTheta * Math.PI) / 180) * RC;

  const kL = 1, kC = 1, kH = 1;
  const term1 = deltaLp / (kL * SL);
  const term2 = deltaCp / (kC * SC);
  const term3 = deltaHp / (kH * SH);

  return Math.sqrt(term1 * term1 + term2 * term2 + term3 * term3 + RT * term2 * term3);
}

export function deltaE00Rgb(c1: RGB, c2: RGB): number {
  return deltaE2000(rgbToLab(c1.r, c1.g, c1.b), rgbToLab(c2.r, c2.g, c2.b));
}

/** RG-03 — mistura como média ponderada em luz linear (não em sRGB). */
export function mixDropsLinear(components: (RGB & { drops: number })[]): RGB {
  const total = components.reduce((s, c) => s + c.drops, 0) || 1;
  let rl = 0, gl = 0, bl = 0;
  for (const c of components) {
    const w = c.drops / total;
    rl += srgbToLinear(c.r) * w;
    gl += srgbToLinear(c.g) * w;
    bl += srgbToLinear(c.b) * w;
  }
  return { r: linearToSrgb(rl), g: linearToSrgb(gl), b: linearToSrgb(bl) };
}

/** RG-05 — faixa de veredicto/consequência; chaves batem com dict v1..v5/c1..c5. */
export function verdictKey(deltaE: number): 'v1' | 'v2' | 'v3' | 'v4' | 'v5' {
  if (deltaE < 1) return 'v1';
  if (deltaE < 2) return 'v2';
  if (deltaE < 4) return 'v3';
  if (deltaE < 8) return 'v4';
  return 'v5';
}

export function consequenceKey(deltaE: number): 'c1' | 'c2' | 'c3' | 'c4' | 'c5' {
  if (deltaE < 1) return 'c1';
  if (deltaE < 2) return 'c2';
  if (deltaE < 4) return 'c3';
  if (deltaE < 8) return 'c4';
  return 'c5';
}

/** RG-06 — ΔE00 ≥ 8 é inalcançável (aproximação, aviso antes da fórmula). */
export function isUnreachable(deltaE: number): boolean {
  return deltaE >= 8;
}

/** RG-10.1 — pote pronto quando ΔE00 < 2,5. */
export function isReadyPot(deltaE: number): boolean {
  return deltaE < 2.5;
}

/** 1 gota = 0,05 ml (US-08). */
export function dropsToMl(drops: number): number {
  return drops * 0.05;
}

function gcd(a: number, b: number): number {
  return b === 0 ? a : gcd(b, a % b);
}

/** Menor proporção inteira 1-40 que aproxima os percentuais informados. */
export function computeSuggestedDrops(percentages: number[]): number[] {
  const ints = percentages.map(Math.floor);
  let left = 100 - ints.reduce((a, b) => a + b, 0);
  const byFrac = percentages
    .map((v, idx) => ({ idx, frac: v - Math.floor(v) }))
    .sort((a, b) => b.frac - a.frac);
  for (let k = 0; left > 0 && byFrac.length; k++, left--) {
    ints[byFrac[k % byFrac.length].idx]++;
  }
  const g = ints.filter(v => v > 0).reduce((acc, v) => gcd(acc, v), 0) || 1;
  const drops = ints.map(v => Math.max(1, Math.min(40, Math.round(v / g))));
  return drops;
}

export function hexOfRgb(r: number, g: number, b: number): string {
  const h = (n: number) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0');
  return `#${h(r)}${h(g)}${h(b)}`.toUpperCase();
}

export function rgbOfHex(hex: string): RGB | null {
  const m = /^#?([0-9a-fA-F]{6})$/.exec(hex.trim());
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
}
