// Catálogo em memória — carrega de /api/manufacturers e /api/paints (banco
// vivo do servidor, via api/httpapi) e responde listagem/busca/filtro em TS.
// A matemática de cor e as receitas também vêm da API agora; ver engine.ts.

import { apiGet } from './api';

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
}

export interface Manufacturer {
  id: number;
  name: string;
  paintCount: number;
}

interface ManufacturerResponse {
  id: number;
  name: string;
  paintCount: number;
}

interface PaintResponse {
  id: number;
  manufacturerId: number;
  manufacturer: string;
  name: string;
  code: string;
  productLine: string;
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

export function loadCatalog(): Promise<void> {
  if (!loadPromise) {
    loadPromise = (async () => {
      const [mfrs, paintRows] = await Promise.all([
        apiGet<ManufacturerResponse[]>('/manufacturers'),
        apiGet<PaintResponse[]>('/paints'),
      ]);
      manufacturers = mfrs;
      paints = paintRows.map(p => ({
        id: p.id,
        manufacturerId: p.manufacturerId,
        manufacturer: p.manufacturer,
        name: p.name,
        code: p.code,
        line: p.productLine,
        r: p.r,
        g: p.g,
        b: p.b,
      }));
      byId = new Map(paints.map(p => [p.id, p]));
      codeIndex = paints.map(p => compactCode(p.code));
      searchIndex = paints.map((p, i) => normalize(`${p.name} ${p.code} ${codeIndex[i]} ${p.manufacturer}`));
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
