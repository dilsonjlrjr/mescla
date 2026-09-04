// Ponte com o motor de cor Go/WASM (cmd/wasm → public/mescla.wasm).
// As formas de retorno espelham os DTOs do desktop (frontend/bindings/models.ts)
// — mesma matemática, mesmos campos. Todas as funções do WASM retornam string
// JSON; erros vêm como {"error": "..."}.

import { catalogJSON, loadCatalog } from './catalog';

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

/** Uma tinta do estoque do usuário, no formato que o WASM entende (espelha
 *  pkg/stock.Paint). O `id` local é atribuído pelo serviço de estoque. */
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

interface MesclaWasm {
  init(catalogJSON: string): string;
  findSimilar(r: number, g: number, b: number, maxDeltaE: number, maxResults: number): string;
  suggestEquivalentRecipe(paintId: number, targetManufacturerId: number): string;
  suggestRecipeForColor(r: number, g: number, b: number, targetManufacturerId: number): string;
  suggestFromStock(sourcePaintId: number, stockJSON: string): string;
  parseStockCSV(csvText: string): string;
  stockCSVTemplate(): string;
  stockToCSV(stockJSON: string): string;
  compareToAnchor(anchorId: number, ids: number[]): string;
  bestBrandsFor(paintId: number): string;
}

declare global {
  interface Window {
    __mescla?: MesclaWasm;
    __mesclaOnReady?: () => void;
    Go: new () => { importObject: WebAssembly.Imports; run(inst: WebAssembly.Instance): void };
  }
}

let readyPromise: Promise<void> | null = null;

/** Carrega wasm_exec.js + mescla.wasm e inicializa com o catálogo.
 *  Chamado no boot em paralelo com o first paint — a UI não espera por ele;
 *  quem precisar do motor faz `await engineReady()`. */
export function engineReady(): Promise<void> {
  if (!readyPromise) {
    readyPromise = (async () => {
      // wasm_exec.js define window.Go (runtime do Go pra js/wasm)
      await new Promise<void>((resolve, reject) => {
        const s = document.createElement('script');
        s.src = '/wasm_exec.js';
        s.onload = () => resolve();
        s.onerror = () => reject(new Error('falha carregando wasm_exec.js'));
        document.head.appendChild(s);
      });

      const go = new window.Go();
      const wasmReady = new Promise<void>(resolve => {
        window.__mesclaOnReady = resolve;
      });
      const { instance } = await WebAssembly.instantiateStreaming(
        fetch('/mescla.wasm'),
        go.importObject,
      );
      void go.run(instance); // roda pra sempre (select{} no main)
      await wasmReady;

      await loadCatalog();
      const res = JSON.parse(window.__mescla!.init(catalogJSON()));
      if (res.error) throw new Error(res.error);
    })();
  }
  return readyPromise;
}

function call<T>(fn: () => string): T {
  const parsed = JSON.parse(fn());
  if (parsed && typeof parsed === 'object' && 'error' in parsed) {
    throw new Error((parsed as { error: string }).error);
  }
  return parsed as T;
}

export async function findSimilar(
  r: number,
  g: number,
  b: number,
  maxDeltaE = 25,
  maxResults = 10,
): Promise<SearchResult[]> {
  await engineReady();
  return call<SearchResult[]>(() => window.__mescla!.findSimilar(r, g, b, maxDeltaE, maxResults)) ?? [];
}

export async function suggestEquivalentRecipe(
  paintId: number,
  targetManufacturerId: number,
): Promise<EquivalentRecipe> {
  await engineReady();
  return call<EquivalentRecipe>(() =>
    window.__mescla!.suggestEquivalentRecipe(paintId, targetManufacturerId),
  );
}

export async function suggestRecipeForColor(
  r: number,
  g: number,
  b: number,
  targetManufacturerId: number,
): Promise<EquivalentRecipe> {
  await engineReady();
  return call<EquivalentRecipe>(() =>
    window.__mescla!.suggestRecipeForColor(r, g, b, targetManufacturerId),
  );
}

/** Receita da cor de origem usando SÓ o estoque do pintor. O estoque vai como
 *  JSON (vive no localStorage, não no catálogo em memória). */
export async function suggestFromStock(
  sourcePaintId: number,
  stock: StockPaint[],
): Promise<EquivalentRecipe> {
  await engineReady();
  return call<EquivalentRecipe>(() =>
    window.__mescla!.suggestFromStock(sourcePaintId, JSON.stringify(stock)),
  );
}

/** Valida um CSV de importação contra os fabricantes do catálogo (mesma crítica
 *  do desktop). Devolve as tintas boas e os erros por linha. */
export async function parseStockCSV(csvText: string): Promise<StockCSVResult> {
  await engineReady();
  const res = call<StockCSVResult>(() => window.__mescla!.parseStockCSV(csvText));
  return { paints: res.paints ?? [], errors: res.errors ?? [] };
}

/** Conteúdo do CSV-modelo para download. */
export async function stockCSVTemplate(): Promise<string> {
  await engineReady();
  return call<{ csv: string }>(() => window.__mescla!.stockCSVTemplate()).csv;
}

/** Serializa o estoque no formato de importação (backup/exportação). */
export async function stockToCSV(stock: StockPaint[]): Promise<string> {
  await engineReady();
  return call<{ csv: string }>(() => window.__mescla!.stockToCSV(JSON.stringify(stock))).csv;
}

export async function compareToAnchor(anchorId: number, ids: number[]): Promise<SearchResult[]> {
  await engineReady();
  return call<SearchResult[]>(() => window.__mescla!.compareToAnchor(anchorId, ids)) ?? [];
}

export async function bestBrandsFor(paintId: number): Promise<BrandBest[]> {
  await engineReady();
  return call<BrandBest[]>(() => window.__mescla!.bestBrandsFor(paintId)) ?? [];
}
