// Utilitários de UI do tema Tintômetro — contraste sobre cor de tinta,
// verdicto textual da escala ΔE00 e matiz (pra ordenação por cor).

/** Branco ou grafite, conforme a luminância da cor de fundo. */
export function contrastOn(r: number, g: number, b: number): string {
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.58 ? '#1a1712' : '#ffffff';
}

/** Verdicto da escala ΔE00 (leitura de instrumento). */
export function deltaVerdict(deltaE: number): string {
  if (deltaE < 1) return 'Indistinguível a olho nu';
  if (deltaE < 2) return 'Equivalência excelente';
  if (deltaE < 4) return 'Boa sob luz de bancada';
  if (deltaE < 8) return 'Perceptível lado a lado';
  return 'Outra cor, na prática';
}

/** ΔE bom (verdicto em laca) quando fica abaixo de 2. */
export function deltaIsGood(deltaE: number): boolean {
  return deltaE < 2;
}

/** Matiz 0..360 — usado pra ordenar amostras por cor (arquivo de museu). */
export function hueOf(r: number, g: number, b: number): number {
  const rn = r / 255, gn = g / 255, bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const d = max - min;
  if (d === 0) return 361; // neutros vão pro fim da fila
  let h = 0;
  if (max === rn) h = ((gn - bn) / d) % 6;
  else if (max === gn) h = (bn - rn) / d + 2;
  else h = (rn - gn) / d + 4;
  h *= 60;
  return h < 0 ? h + 360 : h;
}
