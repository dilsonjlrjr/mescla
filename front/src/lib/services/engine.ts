// Motor de cor via mescla-api (api/httpapi) — mesma regra de negócio do
// desktop (api/service + api/domain), agora por HTTP em vez de bind Wails ou
// WASM local. Front não roda mais nada disso no navegador: sem API
// alcançável, estas chamadas rejeitam (decisão do dono de 04/09/2026, sem
// fallback offline — ver nota do reorg no vault).

import { apiGet, apiPost } from './api';
import { loadCatalog } from './catalog';

export interface SearchResult {
  paintId: number;
  name: string;
  code: string;
  manufacturer: string;
  r: number;
  g: number;
  b: number;
  deltaE: number;
  similarity: number;
}

export interface RecipeIngredient {
  paintId: number;
  name: string;
  code: string;
  percentage: number;
  r: number;
  g: number;
  b: number;
}

export interface EquivalentRecipe {
  sourcePaintId: number;
  sourceName: string;
  sourceManufacturer: string;
  sourceR: number;
  sourceG: number;
  sourceB: number;
  targetManufacturer: string;
  ingredients: RecipeIngredient[];
  resultR: number;
  resultG: number;
  resultB: number;
  deltaE: number;
  method: string;
  reproducible: boolean;
  tips: string[];
}

export interface BrandBest {
  manufacturerId: number;
  manufacturer: string;
  paintId: number;
  name: string;
  code: string;
  r: number;
  g: number;
  b: number;
  deltaE: number;
}

/** Uma tinta do estoque do usuário, no formato que a API entende (espelha
 *  api/domain/stock.Paint). O `id` local é atribuído pelo serviço de estoque. */
export interface StockPaint {
  id: number;
  manufacturerId: number;
  manufacturer: string;
  name: string;
  code: string;
  r: number;
  g: number;
  b: number;
  volume: string;
  notes: string;
}

export interface StockRowError {
  line: number;
  message: string;
  raw: string;
}

export interface StockCSVResult {
  paints: StockPaint[];
  errors: StockRowError[];
}

let readyPromise: Promise<void> | null = null;

/** Pré-carrega o catálogo em paralelo com o first paint — a UI não espera por
 *  ele; quem precisar do motor faz `await engineReady()`. Mantido pelo mesmo
 *  motivo de antes (boot mais rápido), só que agora não há WASM pra inicializar. */
export function engineReady(): Promise<void> {
  if (!readyPromise) {
    readyPromise = loadCatalog();
  }
  return readyPromise;
}

export async function findSimilar(
  r: number,
  g: number,
  b: number,
  maxDeltaE = 25,
  maxResults = 10,
): Promise<SearchResult[]> {
  const params = new URLSearchParams({
    r: String(r),
    g: String(g),
    b: String(b),
    maxDeltaE: String(maxDeltaE),
    maxResults: String(maxResults),
  });
  return (await apiGet<SearchResult[]>(`/similar?${params}`)) ?? [];
}

export async function suggestEquivalentRecipe(
  paintId: number,
  targetManufacturerId: number,
): Promise<EquivalentRecipe> {
  const params = new URLSearchParams({
    sourcePaintId: String(paintId),
    targetManufacturerId: String(targetManufacturerId),
  });
  return apiGet<EquivalentRecipe>(`/recipes/by-paint?${params}`);
}

export async function suggestRecipeForColor(
  r: number,
  g: number,
  b: number,
  targetManufacturerId: number,
): Promise<EquivalentRecipe> {
  const params = new URLSearchParams({
    r: String(r),
    g: String(g),
    b: String(b),
    targetManufacturerId: String(targetManufacturerId),
  });
  return apiGet<EquivalentRecipe>(`/recipes/by-color?${params}`);
}

/** Receita da cor de origem usando SÓ o estoque do pintor (localStorage,
 *  nunca o banco do servidor). O pool viaja no corpo da requisição. */
export async function suggestFromStock(
  sourcePaintId: number,
  stock: StockPaint[],
): Promise<EquivalentRecipe> {
  return apiPost<EquivalentRecipe>('/stock/suggest-recipe', { sourcePaintId, stock });
}

/** Valida um CSV de importação contra os fabricantes do catálogo (mesma
 *  crítica do desktop, agora do lado do servidor). Devolve as tintas boas e
 *  os erros por linha. */
export async function parseStockCSV(csvText: string): Promise<StockCSVResult> {
  const res = await apiPost<StockCSVResult>('/stock/parse-csv', { csv: csvText });
  return { paints: res.paints ?? [], errors: res.errors ?? [] };
}

/** Conteúdo do CSV-modelo para download. */
export async function stockCSVTemplate(): Promise<string> {
  return (await apiGet<{ csv: string }>('/stock/csv-template')).csv;
}

/** Serializa o estoque no formato de importação (backup/exportação). */
export async function stockToCSV(stock: StockPaint[]): Promise<string> {
  return (await apiPost<{ csv: string }>('/stock/to-csv', { stock })).csv;
}

export async function compareToAnchor(anchorId: number, ids: number[]): Promise<SearchResult[]> {
  const params = new URLSearchParams({ anchorId: String(anchorId), ids: ids.join(',') });
  return (await apiGet<SearchResult[]>(`/compare-to-anchor?${params}`)) ?? [];
}

export async function bestBrandsFor(paintId: number): Promise<BrandBest[]> {
  const params = new URLSearchParams({ paintId: String(paintId) });
  return (await apiGet<BrandBest[]>(`/best-brands?${params}`)) ?? [];
}
