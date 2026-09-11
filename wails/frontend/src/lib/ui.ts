// Utilidades de UI do tema Tintômetro — contraste sobre cor de tinta,
// verdicto de ΔE00 como leitura de instrumento e ordenação por matiz.

/** Luminância relativa (WCAG) de uma cor 0..255. */
export function luminance(r: number, g: number, b: number): number {
  const lin = (c: number) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

/** Texto branco ou grafite conforme a luminância da cor de fundo. */
export function contrastOn(r: number, g: number, b: number): string {
  return luminance(r, g, b) > 0.35 ? '#1a1712' : '#ffffff';
}

/** true quando a cor pede texto claro (fundo escuro). */
export function isDark(r: number, g: number, b: number): boolean {
  return luminance(r, g, b) <= 0.35;
}

/** Verdicto da escala ΔE00 (board Direção). */
export function deltaVerdict(deltaE: number): string {
  if (deltaE < 1) return 'Indistinguível a olho nu';
  if (deltaE < 2) return 'Excelente: some na mini';
  if (deltaE < 4) return 'Boa: passa sob luz de bancada';
  if (deltaE < 8) return 'Perceptível lado a lado';
  return 'Outra cor, na prática';
}

/** Verdicto curto pra linhas de comparação. */
export function deltaVerdictShort(deltaE: number): string {
  if (deltaE < 2) return 'Equivalência excelente';
  if (deltaE < 4) return 'Boa sob luz de bancada';
  if (deltaE < 8) return 'Diferença perceptível';
  return 'Outra cor, na prática';
}

/** ΔE "bom" fica em laca. */
export function deltaIsGood(deltaE: number): boolean {
  return deltaE < 2;
}

export function hexOf(r: number, g: number, b: number): string {
  const h = (n: number) => n.toString(16).padStart(2, '0').toUpperCase();
  return `#${h(r)}${h(g)}${h(b)}`;
}

/** Código do pote comparável: sem caixa nem pontuação ("XF-2" → "xf2"). */
export function compactCode(code: string): string {
  return code.toLowerCase().replace(/[^a-z0-9]/g, '');
}

/** Busca de tinta por nome, marca ou código. O código casa sem pontuação:
 *  "70951" acha "70.951". */
export function paintMatches(p: { name: string; manufacturer: string; code?: string }, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  const code = p.code ?? '';
  const qc = compactCode(q);
  return (
    p.name.toLowerCase().includes(q) ||
    p.manufacturer.toLowerCase().includes(q) ||
    code.toLowerCase().includes(q) ||
    (qc !== '' && compactCode(code).includes(qc))
  );
}

/** Matiz 0..360 pra ordenação de catálogo; acromáticos vão pro fim. */
export function hueOf(r: number, g: number, b: number): number {
  const rn = r / 255, gn = g / 255, bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const d = max - min;
  if (d < 0.04) return 360 + (max + min); // quase cinza: agrupa no fim, do escuro ao claro
  let h = 0;
  if (max === rn) h = ((gn - bn) / d) % 6;
  else if (max === gn) h = (bn - rn) / d + 2;
  else h = (rn - gn) / d + 4;
  h *= 60;
  if (h < 0) h += 360;
  return h;
}
