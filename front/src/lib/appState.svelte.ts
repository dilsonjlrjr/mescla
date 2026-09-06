// Estado compartilhado entre telas (rf-04: T1 Pergunta/T2 Plano/T3 Receitas/
// T4 Minhas tintas) — os "corredores" do app: T4 → T1 (gerar fórmula
// equivalente a partir de uma tinta do catálogo), T4 (Fabricantes → Tintas
// filtradas por marca).

import type { Paint } from './services/catalog';

export const appState = $state({
  /** Tinta pré-selecionada ao entrar em T1 (deep-link interno, ex.: "gerar
   *  fórmula equivalente" a partir do detalhe de uma tinta em T4). */
  pendingTargetPaint: null as Paint | null,
  /** Filtro de fabricante pré-aplicado ao entrar na aba Tintas de T4 (vindo
   *  de "Ver tintas" na aba Fabricantes). */
  pendingCatalogMfrId: null as number | null,
});
