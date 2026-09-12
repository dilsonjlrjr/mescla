// "Meu estoque" — as tintas que o pintor tem em casa, cadastradas uma a uma
// (nome, marca, cor). Diferente da estante (que guarda só IDs de marcas), aqui
// guardamos a tinta inteira, persistida em localStorage. É o que alimenta o
// "priorizar meu estoque" na aba Mesclar.
//
// IDs são locais e NEGATIVOS (contador decrescente): o catálogo usa IDs
// positivos, então um id negativo nunca colide com uma tinta do catálogo
// dentro do motor de mistura.

import type { StockPaint } from './engine';

const KEY = 'mescla.stock.v1';
const SEQ_KEY = 'mescla.stock.seq.v1';

function load(): StockPaint[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const list = JSON.parse(raw);
    return Array.isArray(list) ? list : [];
  } catch {
    return [];
  }
}

// Sequência de IDs locais: decrementa a cada tinta nova para nunca reusar um id.
let seq = (() => {
  const raw = Number(localStorage.getItem(SEQ_KEY));
  return Number.isFinite(raw) && raw < 0 ? raw : 0;
})();

function nextId(): number {
  seq -= 1;
  try {
    localStorage.setItem(SEQ_KEY, String(seq));
  } catch {
    /* storage indisponível */
  }
  return seq;
}

export const stock = $state({ paints: load() as StockPaint[] });

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify(stock.paints));
  } catch {
    /* storage cheio/indisponível: o estoque só não persiste */
  }
}

/** Tintas do estoque ordenadas por marca e nome (para listagem). */
export function sortedStock(): StockPaint[] {
  return [...stock.paints].sort(
    (a, b) => a.manufacturer.localeCompare(b.manufacturer) || a.name.localeCompare(b.name),
  );
}

/** Marcas distintas presentes no estoque. */
export function stockBrands(): string[] {
  return [...new Set(stock.paints.map(p => p.manufacturer))].sort();
}

// Quantidade sempre inteira e >= 1: ausente, não finita ou < 1 vira 1.
function normalizeQuantity(n: unknown): number {
  const q = Math.floor(Number(n));
  return Number.isFinite(q) ? Math.max(1, q) : 1;
}

export function addStockPaint(p: Omit<StockPaint, 'id'>): StockPaint {
  const paint: StockPaint = { ...p, id: nextId(), quantity: normalizeQuantity(p.quantity) };
  stock.paints.push(paint);
  persist();
  return paint;
}

export function updateStockPaint(p: StockPaint) {
  const i = stock.paints.findIndex(x => x.id === p.id);
  if (i !== -1) {
    stock.paints[i] = { ...p, quantity: normalizeQuantity(p.quantity) };
    persist();
  }
}

export function removeStockPaint(id: number) {
  const i = stock.paints.findIndex(x => x.id === id);
  if (i !== -1) {
    stock.paints.splice(i, 1);
    persist();
  }
}

/** Liga o estoque local aos fabricantes do servidor (rf-14, RN5): pelo id
 *  quando ele existe no servidor, senão pelo nome sem distinguir caixa. A
 *  tinta ligada recebe o id e o nome atuais, então o rename feito em qualquer
 *  aparelho chega aqui na próxima recarga. */
export function relinkStockManufacturers(mfrs: { id: number; name: string }[]) {
  const nameById = new Map(mfrs.map(m => [m.id, m.name]));
  const byName = new Map(mfrs.map(m => [m.name.toLowerCase(), m]));
  let changed = false;
  for (const p of stock.paints) {
    const current = nameById.get(p.manufacturerId);
    if (current !== undefined) {
      if (p.manufacturer !== current) {
        p.manufacturer = current;
        changed = true;
      }
      continue;
    }
    const match = byName.get(p.manufacturer.toLowerCase());
    if (match) {
      p.manufacturerId = match.id;
      p.manufacturer = match.name;
      changed = true;
    }
  }
  if (changed) persist();
}

/** Tintas do estoque local deste aparelho ligadas ao fabricante (RN3/RN5). */
export function localStockCountFor(mfr: { id: number; name: string }): number {
  const name = mfr.name.toLowerCase();
  return stock.paints.filter(p => p.manufacturerId === mfr.id || p.manufacturer.toLowerCase() === name).length;
}

/** Dá tipo de tinta a quem ainda não tem (rf-15) e persiste se algo mudou. */
export function fillStockPaintTypes(typeFor: (p: StockPaint) => number | undefined) {
  let changed = false;
  for (const p of stock.paints) {
    if (p.paintTypeId) continue;
    const id = typeFor(p);
    if (id) {
      p.paintTypeId = id;
      changed = true;
    }
  }
  if (changed) persist();
}

/** Tintas do estoque local deste aparelho com o tipo — prende a exclusão. */
export function localStockCountForType(typeId: number): number {
  return stock.paints.filter(p => p.paintTypeId === typeId).length;
}

/** Insere um lote (importação CSV já validada). Devolve quantas entraram. */
export function addStockPaints(paints: Omit<StockPaint, 'id'>[]): number {
  for (const p of paints) {
    stock.paints.push({ ...p, id: nextId(), quantity: normalizeQuantity(p.quantity) });
  }
  persist();
  return paints.length;
}

/** Dá quantidade 1 a quem foi cadastrado antes do campo e persiste se algo mudou. */
export function fillStockQuantities() {
  let changed = false;
  for (const p of stock.paints) {
    if (Number.isInteger(p.quantity) && (p.quantity as number) >= 1) continue;
    p.quantity = 1;
    changed = true;
  }
  if (changed) persist();
}

/** Soma dos potes do estoque local (conta a quantidade, não a linha). */
export function totalPots(): number {
  return stock.paints.reduce((sum, p) => sum + normalizeQuantity(p.quantity), 0);
}
