// Estado compartilhado entre abas — os "corredores" do app:
// Catálogo → Mesclar (mesclar esta cor), Mesclar/Catálogo → Cor (RGB preset),
// Catálogo → Comparar (seleção persiste entre abas).

import type { Paint } from './services/catalog';

export const appState = $state({
  /** Tinta pré-selecionada ao entrar na aba Mesclar (deep-link interno). */
  pendingMesclarPaint: null as Paint | null,
  /** Cor pré-carregada ao entrar na aba Cor (ex.: "ver tintas prontas próximas"). */
  corPreset: null as { r: number; g: number; b: number } | null,
  /** Filtro de marca pré-aplicado ao entrar no Catálogo (vindo de Marcas). */
  pendingCatalogMfrId: null as number | null,
  /** Tintas escolhidas pra comparação (máx 6) — badge no menu Mais. */
  compareIds: [] as number[],
  /** Pré-preenche o form do estoque ("Tenho outra parecida" no detalhe). */
  pendingStockPrefill: null as { manufacturerId: number; hex: string } | null,
});

export function addToCompare(id: number): boolean {
  if (appState.compareIds.includes(id)) return false;
  if (appState.compareIds.length >= 6) return false;
  appState.compareIds.push(id);
  return true;
}

export function removeFromCompare(id: number) {
  const i = appState.compareIds.indexOf(id);
  if (i !== -1) appState.compareIds.splice(i, 1);
}
