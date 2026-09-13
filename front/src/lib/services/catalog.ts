// Catálogo em memória — carrega de /api/manufacturers e /api/paints (banco
// vivo do servidor, via api/httpapi) e responde listagem/busca/filtro em TS.
// A matemática de cor e as receitas também vêm da API agora; ver engine.ts.

import { apiDelete, apiGet, apiPost, apiPut } from './api';
import { catalogRev } from './catalogRev.svelte';

export interface Paint {
  id: number;
  manufacturerId: number;
  manufacturer: string;
  name: string;
  code: string;
  line: string;
  r: number;
  g: number;
  b: number;
  /** Tipo de tinta (rf-15); 0 = sem tipo. */
  paintTypeId: number;
}

export interface Manufacturer {
  id: number;
  name: string;
  paintCount: number;
  /** Tintas do estoque do servidor — junto com paintCount, prende a exclusão. */
  userPaintCount: number;
}

interface ManufacturerResponse {
  id: number;
  name: string;
  paintCount: number;
  userPaintCount?: number;
}

export const MAX_MANUFACTURER_NAME = 80;

function toManufacturer(m: ManufacturerResponse): Manufacturer {
  return { id: m.id, name: m.name, paintCount: m.paintCount, userPaintCount: m.userPaintCount ?? 0 };
}

interface PaintResponse {
  id: number;
  manufacturerId: number;
  manufacturer: string;
  name: string;
  code: string;
  productLine: string;
  paintTypeId?: number;
  r: number;
  g: number;
  b: number;
}

let paints: Paint[] = [];
let manufacturers: Manufacturer[] = [];
let byId = new Map<number, Paint>();
// Índice de busca pré-normalizado: "nome código marca" sem acento, minúsculo.
let searchIndex: string[] = [];
// Código do pote sem pontuação ("70.951" → "70951", "XF-2" → "xf2"): o pintor
// digita o código como lembra, não com o ponto ou o hífen do rótulo.
let codeIndex: string[] = [];

let loadPromise: Promise<void> | null = null;

function normalize(s: string): string {
  return s
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '');
}

function compactCode(s: string): string {
  return normalize(s).replace(/[^a-z0-9]/g, '');
}

function indexEntry(p: Paint, compact: string): string {
  return normalize(`${p.name} ${p.code} ${compact} ${p.manufacturer}`);
}

export function loadCatalog(): Promise<void> {
  if (!loadPromise) {
    loadPromise = (async () => {
      const [mfrs, paintRows, types] = await Promise.all([
        apiGet<ManufacturerResponse[]>('/manufacturers'),
        apiGet<PaintResponse[]>('/paints'),
        apiGet<PaintType[]>('/paint-types').catch(() => [] as PaintType[]),
      ]);
      manufacturers = mfrs.map(toManufacturer);
      paintTypes = types;
      paints = paintRows.map(p => ({
        id: p.id,
        manufacturerId: p.manufacturerId,
        manufacturer: p.manufacturer,
        name: p.name,
        code: p.code,
        line: p.productLine,
        paintTypeId: p.paintTypeId ?? 0,
        r: p.r,
        g: p.g,
        b: p.b,
      }));
      byId = new Map(paints.map(p => [p.id, p]));
      codeIndex = paints.map(p => compactCode(p.code));
      searchIndex = paints.map((p, i) => indexEntry(p, codeIndex[i]));
    })();
  }
  return loadPromise;
}

export function allPaints(): Paint[] {
  return paints;
}

export function allManufacturers(): Manufacturer[] {
  return manufacturers;
}

export function paintById(id: number): Paint | undefined {
  return byId.get(id);
}

// ── Fabricantes no servidor (rf-14) ──

let reloadSeq = 0;

/** Recarrega os fabricantes (RN7) e acerta o nome do fabricante nas tintas em
 *  memória: o rename chega à lista de tintas, aos filtros e à busca. */
export async function reloadManufacturers(): Promise<Manufacturer[]> {
  // Só a recarga mais recente vale: uma resposta atrasada (o GET da abertura
  // de T4 chegando depois do GET pós-PUT) desfaria o rename.
  const seq = ++reloadSeq;
  const rows = await apiGet<ManufacturerResponse[]>('/manufacturers');
  if (seq !== reloadSeq) return manufacturers;
  manufacturers = rows.map(toManufacturer);
  const nameById = new Map(manufacturers.map(m => [m.id, m.name]));
  for (let i = 0; i < paints.length; i++) {
    const name = nameById.get(paints[i].manufacturerId);
    if (name !== undefined && name !== paints[i].manufacturer) {
      paints[i].manufacturer = name;
      searchIndex[i] = indexEntry(paints[i], codeIndex[i]);
    }
  }
  catalogRev.n++;
  return manufacturers;
}

export async function createManufacturer(name: string): Promise<Manufacturer> {
  return toManufacturer(await apiPost<ManufacturerResponse>('/manufacturers', { name }));
}

export async function updateManufacturer(id: number, name: string): Promise<Manufacturer> {
  return toManufacturer(await apiPut<ManufacturerResponse>(`/manufacturers/${id}`, { name }));
}

export async function deleteManufacturer(id: number): Promise<void> {
  await apiDelete<{ id: number }>(`/manufacturers/${id}`);
}

// ── Tipos de tinta (rf-15) ──

export interface PaintType {
  id: number;
  name: string;
  /** Tintas do catálogo com este tipo — prende a exclusão. */
  paintCount: number;
  /** rf-17: tintas do estoque (servidor) com este tipo — também prende. */
  userPaintCount?: number;
}

export const MAX_PAINT_TYPE_NAME = 60;

let paintTypes: PaintType[] = [];
let paintTypeSeq = 0;

export function allPaintTypes(): PaintType[] {
  return paintTypes;
}

/** Recarrega os tipos; resposta superada por outra mais nova é descartada. */
export async function reloadPaintTypes(): Promise<PaintType[]> {
  const seq = ++paintTypeSeq;
  const rows = await apiGet<PaintType[]>('/paint-types');
  if (seq !== paintTypeSeq) return paintTypes;
  paintTypes = rows;
  catalogRev.n++;
  return paintTypes;
}

export function createPaintType(name: string): Promise<PaintType> {
  return apiPost<PaintType>('/paint-types', { name });
}

export function updatePaintType(id: number, name: string): Promise<PaintType> {
  return apiPut<PaintType>(`/paint-types/${id}`, { name });
}

export async function deletePaintType(id: number): Promise<void> {
  await apiDelete<{ id: number }>(`/paint-types/${id}`);
}

const LEGACY_MFRS_KEY = 'mescla.customMfrs.v1';

/** Sobe a lista local de fabricantes, anterior ao servidor (RN6). Nome que o
 *  servidor já tem casa com ele; nome inválido é descartado; só falha de rede
 *  ou 5xx fica na chave para a próxima abertura. Devolve quantos ficaram. */
export async function migrateLegacyManufacturers(): Promise<number> {
  let raw: string | null;
  try {
    raw = localStorage.getItem(LEGACY_MFRS_KEY);
  } catch {
    return 0;
  }
  if (raw == null) return 0;

  let legacy: unknown = [];
  try {
    legacy = JSON.parse(raw);
  } catch {
    // chave corrompida: não há o que subir
  }
  const names = new Map<string, string>();
  for (const m of Array.isArray(legacy) ? legacy : []) {
    const name = typeof m?.name === 'string' ? m.name.trim() : '';
    if (name && [...name].length <= MAX_MANUFACTURER_NAME && !names.has(name.toLowerCase())) {
      names.set(name.toLowerCase(), name);
    }
  }

  let known: Set<string>;
  try {
    known = new Set((await reloadManufacturers()).map(m => m.name.toLowerCase()));
  } catch {
    return names.size;
  }
  const pending: { name: string }[] = [];
  for (const [key, name] of names) {
    if (known.has(key)) continue;
    try {
      await apiPost('/manufacturers', { name });
      known.add(key);
    } catch (e) {
      // 409: outro cadastro chegou antes e casa pelo nome; 400: descartado.
      const status = (e as { status?: number } | null)?.status ?? 0;
      if (status === 0 || status >= 500) pending.push({ name });
    }
  }
  try {
    if (pending.length > 0) localStorage.setItem(LEGACY_MFRS_KEY, JSON.stringify(pending));
    else localStorage.removeItem(LEGACY_MFRS_KEY);
  } catch {
    /* storage indisponível */
  }
  return pending.length;
}

export interface SearchOptions {
  manufacturerId?: number;
  limit?: number;
}

/** Busca por nome/código/marca. Ranking: código exato > prefixo de código > prefixo de nome
 *  > substring — o pintor digita "70.9", "70951" ou "meph", não frases. O
 *  código casa sem pontuação: "xf2" acha "XF-2". */
export function searchPaints(query: string, opts: SearchOptions = {}): Paint[] {
  const limit = opts.limit ?? 50;
  const q = normalize(query.trim());

  let pool: number[] | null = null;
  if (opts.manufacturerId != null) {
    pool = [];
    for (let i = 0; i < paints.length; i++) {
      if (paints[i].manufacturerId === opts.manufacturerId) pool.push(i);
    }
  }

  if (!q) {
    const base = pool ? pool.map(i => paints[i]) : paints;
    return base.slice(0, limit);
  }

  const exactHits: Paint[] = [];
  const codeHits: Paint[] = [];
  const nameHits: Paint[] = [];
  const subHits: Paint[] = [];
  const qc = compactCode(q);
  const scan = (i: number) => {
    const p = paints[i];
    if (qc && codeIndex[i] === qc) {
      exactHits.push(p);
    } else if (qc && codeIndex[i].startsWith(qc)) {
      codeHits.push(p);
    } else if (normalize(p.name).startsWith(q)) {
      nameHits.push(p);
    } else if (searchIndex[i].includes(q)) {
      subHits.push(p);
    }
  };
  if (pool) {
    for (const i of pool) scan(i);
  } else {
    for (let i = 0; i < paints.length; i++) scan(i);
  }

  return [...exactHits, ...codeHits, ...nameHits, ...subHits].slice(0, limit);
}

export function hexOf(p: { r: number; g: number; b: number }): string {
  const h = (n: number) => n.toString(16).padStart(2, '0').toUpperCase();
  return `#${h(p.r)}${h(p.g)}${h(p.b)}`;
}
