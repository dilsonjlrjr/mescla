// rf-17 — recuperação do estoque que ficou no localStorage do navegador
// (`mescla.stock.v1`). Sobe para o servidor por `POST /user-paints/migrate`,
// de forma idempotente, e **nunca altera nem apaga a chave local** (M4): ela
// fica como cópia de segurança. Grava só a marca da migração e o id do
// navegador.
//
// O app é servido por HTTP sem TLS: `crypto.randomUUID` e `crypto.subtle` não
// existem ali. O id do navegador sai de `crypto.getRandomValues` e o hash é
// uma função pura (M9).

import { apiPost } from './api';
import { allManufacturers, allPaints, allPaintTypes, paintById, reloadManufacturers, reloadPaintTypes } from './catalog';
import { stock, definirMigrando, type LinhaNaoSubiu, type LinhaPendente } from './stock.svelte';
import { toast } from '../toast.svelte';
import { t } from '../i18n.svelte';

const CHAVE_ESTOQUE = 'mescla.stock.v1';
const CHAVE_MARCA = 'mescla.stock.migracao.v1';
const CHAVE_NAVEGADOR = 'mescla.device.v1';
const TAMANHO_LOTE = 500;
export const MAX_ARQUIVO_BYTES = 5 * 1024 * 1024;

interface LinhaEnvio {
  localRef: string;
  /** Impressão do conteúdo da linha local (ref + fabricante + nome + código +
   *  cor). A mesma lista vinda por outro caminho (arquivo, outro navegador)
   *  reconhece a origem já migrada e não duplica nem ressuscita tinta apagada. */
  fingerprint: string;
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

interface ResultadoLinha {
  localRef: string;
  status: 'criada' | 'mesclada' | 'ja-migrada' | 'recusada';
  motivo: string;
}

interface RespostaMigrate {
  results: ResultadoLinha[];
  criadas: number;
  mescladas: number;
  jaMigradas: number;
  recusadas: number;
}

export interface ResumoEnvio {
  criadas: number;
  mescladas: number;
  jaMigradas: number;
  recusadas: LinhaNaoSubiu[];
  /** localRef das linhas que estão no servidor (criada, mesclada, já migrada). */
  noServidor: Set<string>;
}

// ── Hash (cyrb53, 53 bits) — identidade idempotente, não segurança ──
function cyrb53(str: string, seed: number): string {
  let h1 = 0xdeadbeef ^ seed;
  let h2 = 0x41c6ce57 ^ seed;
  for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i);
    h1 = Math.imul(h1 ^ ch, 2654435761);
    h2 = Math.imul(h2 ^ ch, 1597334677);
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507) ^ Math.imul(h2 ^ (h2 >>> 13), 3266489909);
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507) ^ Math.imul(h1 ^ (h1 >>> 13), 3266489909);
  const n = 4294967296 * (2097151 & h2) + (h1 >>> 0);
  return n.toString(16).padStart(14, '0').slice(-13);
}

export function hashTexto(texto: string): string {
  return cyrb53(texto, 1) + cyrb53(texto, 2);
}

// ── localStorage, sempre tolerante: falha de leitura nunca vira escrita ──
function lerChave(chave: string): string | null {
  try {
    return localStorage.getItem(chave);
  } catch {
    return null;
  }
}

function gravarChave(chave: string, valor: string): boolean {
  try {
    localStorage.setItem(chave, valor);
    return true;
  } catch {
    return false;
  }
}

interface Marca {
  hash?: string;
  dispensadas: string[];
}

function lerMarca(): Marca {
  const cru = lerChave(CHAVE_MARCA);
  if (!cru) return { dispensadas: [] };
  try {
    const o = JSON.parse(cru) as Record<string, unknown>;
    return {
      hash: typeof o.hash === 'string' ? o.hash : undefined,
      dispensadas: Array.isArray(o.dispensadas) ? o.dispensadas.filter((x): x is string => typeof x === 'string') : [],
    };
  } catch {
    return { dispensadas: [] };
  }
}

function gravarMarca(marca: Marca): void {
  gravarChave(CHAVE_MARCA, JSON.stringify({ ...marca, em: new Date().toISOString() }));
}

/** RN2: id do navegador (32 hex). `null` quando não dá para guardar. */
function idDoNavegador(): string | null {
  const salvo = lerChave(CHAVE_NAVEGADOR);
  if (salvo && /^[0-9a-f]{32}$/.test(salvo)) return salvo;
  try {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    const novo = [...bytes].map(b => b.toString(16).padStart(2, '0')).join('');
    return gravarChave(CHAVE_NAVEGADOR, novo) ? novo : null;
  } catch {
    return null;
  }
}

// ── Preparação (RN3, RN4) ──
/** RN3: `localRef` estável por linha, mesmo com id repetido. */
export function localRefs(lista: unknown[]): string[] {
  const vistos = new Map<number, number>();
  return lista.map((item, i) => {
    const id = (item as { id?: unknown } | null)?.id;
    if (typeof id !== 'number' || !Number.isInteger(id) || Math.abs(id) > 999_999_999_999_999) return `i${i}`;
    const n = (vistos.get(id) ?? 0) + 1;
    vistos.set(id, n);
    return n === 1 ? String(id) : `${id}#${n}`;
  });
}

function texto(v: unknown): string {
  if (typeof v === 'string') return v;
  if (v == null) return '';
  return String(v);
}

function inteiro(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? Math.round(n) : -1;
}

function codigoSemSentido(code: string): boolean {
  return /^(null(-\d+)?)?$/i.test(code.trim());
}

interface Contexto {
  mfrPorId: Map<number, string>;
  mfrPorNome: Map<string, number>;
  tiposValidos: Set<number>;
  acrilica: number | null;
  catalogoPorChave: Map<string, number>;
}

function montarContexto(): Contexto {
  const mfrs = allManufacturers();
  const tipos = allPaintTypes();
  // Chave igual à de `catalogOriginOf` em T4; `-1` marca chave ambígua.
  const catalogoPorChave = new Map<string, number>();
  for (const p of allPaints()) {
    if (codigoSemSentido(p.code)) continue;
    const chave = `${p.manufacturer}|${p.code}`;
    catalogoPorChave.set(chave, catalogoPorChave.has(chave) ? -1 : p.id);
  }
  return {
    mfrPorId: new Map(mfrs.map(m => [m.id, m.name])),
    mfrPorNome: new Map(mfrs.map(m => [m.name.trim().toLowerCase(), m.id])),
    tiposValidos: new Set(tipos.map(pt => pt.id)),
    acrilica: tipos.find(pt => pt.name.trim().toLowerCase() === 'acrílica')?.id ?? null,
    catalogoPorChave,
  };
}

function preparar(item: unknown, localRef: string, ctx: Contexto): LinhaEnvio {
  const o = (typeof item === 'object' && item !== null ? item : {}) as Record<string, unknown>;
  const manufacturer = texto(o.manufacturer);
  const code = texto(o.code);

  let manufacturerId = inteiro(o.manufacturerId);
  if (!ctx.mfrPorId.has(manufacturerId)) {
    manufacturerId = ctx.mfrPorNome.get(manufacturer.trim().toLowerCase()) ?? manufacturerId;
  }

  let catalogId: number | null = null;
  const catalogoInformado = inteiro(o.catalogId);
  if (catalogoInformado > 0 && paintById(catalogoInformado)) {
    catalogId = catalogoInformado;
  } else if (!codigoSemSentido(code)) {
    const achado = ctx.catalogoPorChave.get(`${manufacturer}|${code}`);
    if (achado !== undefined && achado > 0) catalogId = achado;
  }

  let paintTypeId: number | null = null;
  const tipoInformado = inteiro(o.paintTypeId);
  if (ctx.tiposValidos.has(tipoInformado)) {
    paintTypeId = tipoInformado;
  } else {
    const doCatalogo = catalogId != null ? paintById(catalogId)?.paintTypeId : undefined;
    paintTypeId = doCatalogo && ctx.tiposValidos.has(doCatalogo) ? doCatalogo : ctx.acrilica;
  }

  const quantidade = inteiro(o.quantity);
  const name = texto(o.name);
  const r = inteiro(o.r);
  const g = inteiro(o.g);
  const b = inteiro(o.b);
  return {
    localRef,
    fingerprint: hashTexto(JSON.stringify([localRef, manufacturer.trim().toLowerCase(), name.trim(), code.trim(), r, g, b])),
    manufacturerId,
    manufacturer,
    name,
    code,
    r,
    g,
    b,
    volume: texto(o.volume),
    notes: texto(o.notes),
    quantity: quantidade >= 1 ? quantidade : 1,
    paintTypeId,
    catalogId,
  };
}

/** RN8: envia em lotes, em sequência. Falha de rede, 4xx ou 5xx de um lote
 *  interrompe e lança — quem chama não grava marca. */
async function enviar(deviceId: string, linhas: LinhaEnvio[], dispensadas: Set<string>): Promise<ResumoEnvio> {
  const resumo: ResumoEnvio = { criadas: 0, mescladas: 0, jaMigradas: 0, recusadas: [], noServidor: new Set() };
  for (let i = 0; i < linhas.length; i += TAMANHO_LOTE) {
    const lote = linhas.slice(i, i + TAMANHO_LOTE);
    const resp = await apiPost<RespostaMigrate>('/user-paints/migrate', { deviceId, paints: lote });
    lote.forEach((linha, k) => {
      const r = resp.results?.[k];
      if (r && r.status !== 'recusada') {
        resumo.noServidor.add(linha.localRef);
        if (r.status === 'criada') resumo.criadas++;
        else if (r.status === 'mesclada') resumo.mescladas++;
        else resumo.jaMigradas++;
        return;
      }
      if (dispensadas.has(linha.localRef)) return;
      resumo.recusadas.push({
        localRef: linha.localRef,
        name: linha.name,
        code: linha.code,
        manufacturer: linha.manufacturer,
        motivo: r?.motivo || 'linha-invalida',
      });
    });
  }
  return resumo;
}

function pendentesDe(linhas: LinhaEnvio[], noServidor: Set<string>, dispensadas: Set<string>): LinhaPendente[] {
  return linhas
    .filter(l => !noServidor.has(l.localRef) && !dispensadas.has(l.localRef))
    .map(l => ({ manufacturerId: l.manufacturerId, manufacturer: l.manufacturer, paintTypeId: l.paintTypeId }));
}

// ── Migração do navegador (RN1, RN7, RN9) ──
let emAndamento: Promise<void> | null = null;

/** Migra o estoque local deste navegador, se precisar. Uma execução por vez. */
export function migrarEstoqueLocal(): Promise<void> {
  if (!emAndamento) emAndamento = executar().finally(() => { emAndamento = null; });
  return emAndamento;
}

async function executar(): Promise<void> {
  const cru = lerChave(CHAVE_ESTOQUE);
  if (!cru) {
    stock.pendentes = [];
    stock.naoSubiram = [];
    stock.falhaEnvio = false;
    return;
  }
  let lista: unknown;
  try {
    lista = JSON.parse(cru);
  } catch {
    return;
  }
  if (!Array.isArray(lista) || lista.length === 0) return;

  const hash = hashTexto(cru);
  const marca = lerMarca();
  if (marca.hash === hash) {
    stock.pendentes = [];
    stock.naoSubiram = [];
    stock.falhaEnvio = false;
    return;
  }

  // Sem tipos ou fabricantes a preparação não acha Acrílica nem religa marca.
  // Tenta recarregar; se não vier, é falha visível — nunca pular calado, e as
  // linhas continuam prendendo exclusão (RN15).
  if (allPaintTypes().length === 0 || allManufacturers().length === 0) {
    await Promise.allSettled([reloadPaintTypes(), reloadManufacturers()]);
  }
  if (allPaintTypes().length === 0 || allManufacturers().length === 0) {
    stock.falhaEnvio = true;
    stock.pendentes = pendentesCrus(lista, new Set(marca.dispensadas));
    return;
  }

  const ctx = montarContexto();
  const refs = localRefs(lista);
  const linhas = lista.map((item, i) => preparar(item, refs[i], ctx));
  const deviceId = idDoNavegador() ?? `conteudo-${hash}`;

  let resumo: ResumoEnvio;
  definirMigrando(true);
  try {
    resumo = await enviar(deviceId, linhas, new Set(marca.dispensadas));
  } catch {
    stock.falhaEnvio = true;
    stock.pendentes = pendentesDe(linhas, new Set(), new Set(lerMarca().dispensadas));
    return;
  } finally {
    definirMigrando(false);
  }

  // Um "Dispensar" pode ter gravado durante o envio: a marca junta as duas.
  const dispensadas = new Set([...marca.dispensadas, ...lerMarca().dispensadas]);
  const recusadas = resumo.recusadas.filter(l => !dispensadas.has(l.localRef));
  stock.falhaEnvio = false;
  stock.naoSubiram = recusadas;
  stock.pendentes = pendentesDe(linhas, resumo.noServidor, dispensadas);
  gravarMarca(recusadas.length === 0 ? { hash, dispensadas: [...dispensadas] } : { dispensadas: [...dispensadas] });

  const novas = resumo.criadas + resumo.mescladas;
  if (novas > 0) toast(t('stockMigrated', { n: novas + resumo.jaMigradas }));
}

function pendentesCrus(lista: unknown[], dispensadas: Set<string>): LinhaPendente[] {
  const refs = localRefs(lista);
  return lista
    .map((item, i) => ({ item: (typeof item === 'object' && item !== null ? item : {}) as Record<string, unknown>, ref: refs[i] }))
    .filter(({ ref }) => !dispensadas.has(ref))
    .map(({ item }) => {
      const tipo = inteiro(item.paintTypeId);
      return { manufacturerId: inteiro(item.manufacturerId), manufacturer: texto(item.manufacturer), paintTypeId: tipo > 0 ? tipo : null };
    });
}

/** RN7: "Dispensar" as linhas recusadas listadas e refazer a migração — as que
 *  já estão no servidor voltam `ja-migrada`, e a marca fecha. */
export async function dispensarNaoSubiram(): Promise<void> {
  // Uma migração em voo regravaria a marca sem esta dispensa.
  if (emAndamento) await emAndamento.catch(() => undefined);
  const marca = lerMarca();
  const dispensadas = new Set([...marca.dispensadas, ...stock.naoSubiram.map(l => l.localRef)]);
  gravarMarca({ dispensadas: [...dispensadas] });
  stock.naoSubiram = [];
  await migrarEstoqueLocal();
}

// ── Importar arquivo (RN10) ──
export type ErroImportacao = 'grande' | 'nao-lista' | 'falha';

export async function importarArquivoDeEstoque(arquivo: File): Promise<ResumoEnvio> {
  if (arquivo.size > MAX_ARQUIVO_BYTES) throw new ErroDeImportacao('grande');
  const conteudo = await arquivo.text();
  let lista: unknown;
  try {
    lista = JSON.parse(conteudo);
  } catch {
    throw new ErroDeImportacao('nao-lista');
  }
  if (!Array.isArray(lista)) throw new ErroDeImportacao('nao-lista');
  if (lista.length === 0) return { criadas: 0, mescladas: 0, jaMigradas: 0, recusadas: [], noServidor: new Set() };
  const ctx = montarContexto();
  const refs = localRefs(lista);
  const linhas = lista.map((item, i) => preparar(item, refs[i], ctx));
  try {
    return await enviar(`arquivo-${hashTexto(conteudo)}`, linhas, new Set());
  } catch {
    throw new ErroDeImportacao('falha');
  }
}

export class ErroDeImportacao extends Error {
  constructor(public readonly codigo: ErroImportacao) {
    super(codigo);
  }
}

/** Chave i18n do motivo de recusa que o servidor devolve (código estável). */
export function chaveDoMotivo(motivo: string) {
  switch (motivo) {
    case 'linha-repetida': return 'motivoLinhaRepetida' as const;
    case 'fabricante-nao-encontrado': return 'motivoFabricante' as const;
    case 'nome-obrigatorio': return 'motivoNome' as const;
    case 'texto-longo': return 'motivoTexto' as const;
    case 'cor-invalida': return 'motivoCor' as const;
    case 'quantidade-invalida': return 'motivoQuantidade' as const;
    default: return 'motivoLinhaInvalida' as const;
  }
}
