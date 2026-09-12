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
  /** rf-13: fabricante DESTE ingrediente — cross-brand mistura potes de marcas diferentes. */
  manufacturerId: number;
  manufacturer: string;
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
  /** rf-11: faixa de qualidade do acerto — decide o selo e se o diálogo de
   *  fallback abre. */
  faixa?: 'otimo' | 'aproximada' | 'nao-encontrei';
  /** rf-11: a região foi resolvida fora do universo pedido, com autorização
   *  do usuário no diálogo. */
  foraDoUniverso?: boolean;
  /** D-010: pode vir `null` de servidor antigo (fatia nil do Go virava
   *  `null` no JSON). Quem lê guarda antes de indexar. */
  tips: string[] | null;
  /** rf-13 RN6: true quando os ingredientes têm mais de um manufacturerId distinto.
   *  Decisão do servidor — o cliente não infere. */
  crossBrand: boolean;
  /** rf-13 RN7: fabricantes presentes nos ingredientes, ordenados, sem repetição. */
  manufacturers: string[];
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
  /** Id da tinta do catálogo de onde esta nasceu (só no front; a API ignora). */
  catalogId?: number;
  /** Tipo de tinta (rf-15). Estoque anterior ao campo recebe o tipo na abertura de T4. */
  paintTypeId?: number;
  /** Quantos potes iguais o pintor tem. Estoque anterior ao campo vira 1 na abertura de T4. */
  quantity?: number;
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

/** rf-13: targetManufacturerId agora é opcional — ausente ou 0 = catálogo inteiro
 *  (cross-brand real, RG-13). maxIngredients é o teto de ingredientes da fórmula
 *  (ausente ou 0 = sem teto, comportamento anterior de T2/T3). */
export async function suggestEquivalentRecipe(
  paintId: number,
  targetManufacturerId?: number,
  maxIngredients?: number,
): Promise<EquivalentRecipe> {
  const params = new URLSearchParams({ sourcePaintId: String(paintId) });
  if (targetManufacturerId && targetManufacturerId > 0) {
    params.set('targetManufacturerId', String(targetManufacturerId));
  }
  if (maxIngredients !== undefined) {
    params.set('maxIngredients', String(maxIngredients));
  }
  return apiGet<EquivalentRecipe>(`/recipes/by-paint?${params}`);
}

/** rf-11: o universo de busca. Os dois interruptores são independentes e
 *  combináveis; ligados juntos significam "só o que eu tenho daquela marca". */
export interface UniversoBusca {
  /** 0 ou ausente = sem fabricante base. */
  targetManufacturerId?: number;
  useStockOnly?: boolean;
  /** Só depois de o usuário autorizar a saída no diálogo de fallback. */
  foraDoUniverso?: boolean;
  /** rf-13 RN5: teto de ingredientes da receita. Ausente ou 0 = sem teto. */
  maxIngredients?: number;
  /** rf-16 RN14: o estoque vive no aparelho (`stock.svelte.ts`), não no
   *  servidor. Com `useStockOnly` e esta lista, o cálculo vai por POST com o
   *  estoque no corpo; sem ela, o GET antigo (que lê `user_paints`). */
  stock?: StockPaint[];
}

/** Resposta possível quando o universo escolhido não tem tinta nenhuma. Não é
 *  erro: a tela mostra o motivo e oferece as saídas (rf-11 RN9). */
export interface UniversoVazio {
  universoVazio: true;
  motivo: string;
}

export function ehUniversoVazio(v: EquivalentRecipe | UniversoVazio): v is UniversoVazio {
  return (v as UniversoVazio).universoVazio === true;
}

function paramsDoUniverso(r: number, g: number, b: number, u: UniversoBusca): URLSearchParams {
  const params = new URLSearchParams({ r: String(r), g: String(g), b: String(b) });
  if (u.targetManufacturerId && u.targetManufacturerId > 0) {
    params.set('targetManufacturerId', String(u.targetManufacturerId));
  }
  if (u.useStockOnly) params.set('useStockOnly', '1');
  if (u.foraDoUniverso) params.set('foraDoUniverso', '1');
  if (u.maxIngredients && u.maxIngredients > 0) params.set('maxIngredients', String(u.maxIngredients));
  return params;
}

export async function suggestRecipeForColor(
  r: number,
  g: number,
  b: number,
  universo: UniversoBusca,
): Promise<EquivalentRecipe | UniversoVazio> {
  if (universo.useStockOnly && universo.stock) {
    // Campo a campo, nunca spread: quantidade, notas e ids de catálogo do
    // estoque local não interessam ao motor.
    return apiPost<EquivalentRecipe | UniversoVazio>('/recipes/by-color', {
      r,
      g,
      b,
      targetManufacturerId: universo.targetManufacturerId ?? 0,
      foraDoUniverso: universo.foraDoUniverso ?? false,
      maxIngredients: universo.maxIngredients ?? 0,
      stock: universo.stock.map(p => ({
        id: p.id,
        manufacturerId: p.manufacturerId,
        manufacturer: p.manufacturer,
        name: p.name,
        code: p.code,
        r: p.r,
        g: p.g,
        b: p.b,
      })),
    });
  }
  return apiGet<EquivalentRecipe | UniversoVazio>(`/recipes/by-color?${paramsDoUniverso(r, g, b, universo)}`);
}

/** Atalho para quem só quer o fabricante: T1/T3 usam o universo simples de uma
 *  marca, sem os interruptores do plano de peça. Nunca devolve universo vazio —
 *  uma marca do catálogo sempre tem tintas com cor. */
export async function recipeForColorInBrand(
  r: number,
  g: number,
  b: number,
  targetManufacturerId: number,
  maxIngredients?: number,
): Promise<EquivalentRecipe> {
  const resp = await suggestRecipeForColor(r, g, b, { targetManufacturerId, maxIngredients });
  if (ehUniversoVazio(resp)) {
    throw new Error(resp.motivo);
  }
  return resp;
}

/** rf-11 RN6: o melhor ΔE00 alcançável numa combinação, sem montar a receita —
 *  é o número que o diálogo de fallback mostra em cada saída. */
export async function melhorDeltaE(
  r: number,
  g: number,
  b: number,
  universo: UniversoBusca,
): Promise<number | null> {
  const resp = await apiGet<{ deltaE?: number; universoVazio?: boolean }>(
    `/recipes/best-delta-e?${paramsDoUniverso(r, g, b, universo)}`,
  );
  return typeof resp.deltaE === 'number' ? resp.deltaE : null;
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
