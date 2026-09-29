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
  /** rf-23: id de catálogo que T6 Comparar assume como âncora ao abrir (vindo
   *  de T4 ou da paleta de comandos). */
  pendingCompareAnchor: null as number | null,
  /** rf-23: tinta do catálogo que T4 usa para preencher o formulário de
   *  cadastro (vindo da paleta, "Adicionar ao meu estoque"). */
  pendingStockPrefill: null as Paint | null,
  /** rf-23: paleta de comandos (Ctrl+K) aberta — o botão do cabeçalho liga. */
  /** rf-23: aba de T4 pedida pela paleta (Fabricantes ou Tipos). */
  pendingCatalogTab: null as 'fabricantes' | 'tipos' | null,
  paletaAberta: false,
  /** rf-24: guia de primeiro uso aberto — o "?" do cabeçalho e a paleta ligam. */
  guiaAberta: false,
});
