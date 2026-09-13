// "Minhas tintas" — o estoque do pintor. Desde o rf-17 vive no servidor
// (`/user-paints`) e vale em qualquer navegador; o que ainda estiver no
// localStorage antigo sobe por `estoque-migracao.ts`, que nunca altera a chave.
//
// IDs no cliente continuam NEGATIVOS (`-id do servidor`): o catálogo usa IDs
// positivos, então um id negativo nunca colide com uma tinta do catálogo
// dentro do motor de mistura nem na mistura salva do rf-16.
//
// Nenhuma gravação falha em silêncio (D-018): toda escrita lança `ApiError` e
// a tela decide a mensagem; a lista em memória só muda com sucesso.

import { apiDelete, apiGet, apiPost, apiPut, ApiError } from './api';
import type { StockPaint } from './engine';

export type EstadoEstoque = 'carregando' | 'pronto' | 'erro';

/** Linha do localStorage que o servidor recusou (dado local, RN7). */
export interface LinhaNaoSubiu {
  localRef: string;
  name: string;
  code: string;
  manufacturer: string;
  motivo: string;
}

/** Linha local ainda não migrada nem dispensada — prende exclusão (RN15). */
export interface LinhaPendente {
  manufacturerId: number;
  manufacturer: string;
  paintTypeId: number | null;
}

export const stock = $state({
  paints: [] as StockPaint[],
  estado: 'carregando' as EstadoEstoque,
  /** Falha na primeira carga deixa `paints` vazio: sem esta marca, T4 mostraria
   *  o catálogo inteiro como "não tenho" e um "tenho" duplicaria no servidor. */
  carregadoAlgumaVez: false,
  naoSubiram: [] as LinhaNaoSubiu[],
  falhaEnvio: false,
  pendentes: [] as LinhaPendente[],
});

interface UserPaintDTO {
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
  quantity: number;
  paintTypeId: number | null;
  catalogId: number | null;
}

function doServidor(d: UserPaintDTO): StockPaint {
  return {
    id: -d.id,
    manufacturerId: d.manufacturerId,
    manufacturer: d.manufacturer ?? '',
    name: d.name ?? '',
    code: d.code ?? '',
    r: d.r, g: d.g, b: d.b,
    volume: d.volume ?? '',
    notes: d.notes ?? '',
    catalogId: d.catalogId ?? undefined,
    paintTypeId: d.paintTypeId ?? undefined,
    quantity: normalizeQuantity(d.quantity),
  };
}

// 0 em tipo/catálogo limpa no servidor: o formulário descreve a tinta inteira.
function paraServidor(p: Omit<StockPaint, 'id'>) {
  return {
    manufacturerId: p.manufacturerId,
    name: p.name,
    code: p.code,
    r: p.r, g: p.g, b: p.b,
    volume: p.volume,
    notes: p.notes,
    quantity: normalizeQuantity(p.quantity),
    paintTypeId: p.paintTypeId ?? 0,
    catalogId: p.catalogId ?? 0,
  };
}

// Quantidade sempre inteira e >= 1: ausente, não finita ou < 1 vira 1.
function normalizeQuantity(n: unknown): number {
  const q = Math.floor(Number(n));
  return Number.isFinite(q) ? Math.max(1, q) : 1;
}

function idServidor(idLocal: number): number {
  if (!Number.isInteger(idLocal) || idLocal >= 0) throw new ApiError('id inválido', 400);
  return -idLocal;
}

// ── Carga (RN11) ──
// Toda carga e toda escrita sobem `rev`: resposta de uma carga iniciada antes
// da última escrita (ou de uma carga mais nova) é descartada — senão um GET
// lento tiraria da tela a tinta que acabou de ser gravada.
let rev = 0;
let esperando: Array<() => void> = [];
let migracoesEmVoo = 0;

function liberarQuemEspera(): void {
  if (stock.estado === 'carregando' || migracoesEmVoo > 0) return;
  const fila = esperando;
  esperando = [];
  fila.forEach(f => f());
}

/** RN13: T1/T2 esperam a carga E a migração do navegador — no boot a carga
 *  paralela volta "pronto" com 0 tintas antes de a migração enviar as dele. */
export function estoqueAssentado(): Promise<void> {
  if (stock.estado !== 'carregando' && migracoesEmVoo === 0) return Promise.resolve();
  return new Promise(resolve => esperando.push(resolve));
}

/** Marca o começo/fim de uma migração (boot no App, envio em estoque-migracao). */
export function definirMigrando(ativo: boolean): void {
  migracoesEmVoo = Math.max(0, migracoesEmVoo + (ativo ? 1 : -1));
  liberarQuemEspera();
}

export async function carregarEstoque(opts: { mostrarCarregando?: boolean } = {}): Promise<void> {
  const minha = ++rev;
  if (opts.mostrarCarregando) stock.estado = 'carregando';
  try {
    const lista = await apiGet<UserPaintDTO[] | null>('/user-paints');
    if (minha !== rev) return;
    stock.paints = (lista ?? []).map(doServidor);
    stock.estado = 'pronto';
    stock.carregadoAlgumaVez = true;
  } catch {
    if (minha !== rev) return;
    stock.estado = 'erro';
  } finally {
    liberarQuemEspera();
  }
}

/** Uma escrita invalida cargas em voo; se a primeira carga foi descartada por
 *  ela, dispara outra para a tela não ficar presa em "carregando". */
async function escrever<T>(fn: () => Promise<T>): Promise<T> {
  ++rev;
  try {
    return await fn();
  } finally {
    ++rev;
    if (stock.estado === 'carregando') void carregarEstoque();
  }
}

// ── Escrita (RN12) ──
export function addStockPaint(p: Omit<StockPaint, 'id'>): Promise<StockPaint> {
  return escrever(async () => {
    const salvo = doServidor(await apiPost<UserPaintDTO>('/user-paints', paraServidor(p)));
    stock.paints = [...stock.paints.filter(x => x.id !== salvo.id), salvo];
    return salvo;
  });
}

export function updateStockPaint(p: StockPaint): Promise<StockPaint> {
  return escrever(async () => {
    const salvo = doServidor(await apiPut<UserPaintDTO>(`/user-paints/${idServidor(p.id)}`, paraServidor(p)));
    stock.paints = stock.paints.map(x => (x.id === p.id ? salvo : x));
    return salvo;
  });
}

/** 404 conta como sucesso: a tinta já não existe no servidor e sai da lista. */
export function removeStockPaint(id: number): Promise<void> {
  return escrever(async () => {
    try {
      await apiDelete<void>(`/user-paints/${idServidor(id)}`);
    } catch (e) {
      if (!(e instanceof ApiError && e.status === 404)) throw e;
    }
    stock.paints = stock.paints.filter(x => x.id !== id);
  });
}

// ── Leitura ──
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

/** Tintas do estoque (servidor) daquele fabricante. */
export function stockCountFor(mfr: { id: number }): number {
  return stock.paints.filter(p => p.manufacturerId === mfr.id).length;
}

/** Tintas do estoque (servidor) daquele tipo. */
export function stockCountForType(typeId: number): number {
  return stock.paints.filter(p => p.paintTypeId === typeId).length;
}

/** RN15: linhas locais pendentes daquele fabricante (por id ou nome). */
export function pendentesDoFabricante(mfr: { id: number; name: string }): number {
  const nome = mfr.name.trim().toLowerCase();
  return stock.pendentes.filter(p => p.manufacturerId === mfr.id || p.manufacturer.trim().toLowerCase() === nome).length;
}

/** RN15: linhas locais pendentes daquele tipo. */
export function pendentesDoTipo(typeId: number): number {
  return stock.pendentes.filter(p => p.paintTypeId === typeId).length;
}

/** Soma dos potes do estoque (conta a quantidade, não a linha). */
export function totalPots(): number {
  return stock.paints.reduce((sum, p) => sum + normalizeQuantity(p.quantity), 0);
}
