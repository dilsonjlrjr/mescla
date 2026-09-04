// Catálogo em memória — carrega o JSON gerado por cmd/export e responde
// listagem/busca/filtro em TS (a matemática de cor fica no WASM; ver engine.ts).

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

let paints: Paint[] = [];
let manufacturers: Manufacturer[] = [];
let byId = new Map<number, Paint>();
// Índice de busca pré-normalizado: "nome código marca" sem acento, minúsculo.
let searchIndex: string[] = [];
let rawJSON = '';

let loadPromise: Promise<void> | null = null;

function normalize(s: string): string {
  return s
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '');
}

export function loadCatalog(): Promise<void> {
  if (!loadPromise) {
    loadPromise = (async () => {
      const res = await fetch('/data/catalog.json');
      if (!res.ok) throw new Error(`catálogo: HTTP ${res.status}`);
      rawJSON = await res.text();
      const data = JSON.parse(rawJSON) as {
        manufacturers: Manufacturer[];
        paints: [number, number, string, string, string, number, number, number][];
      };
      manufacturers = data.manufacturers;
      const mfrName = new Map(manufacturers.map(m => [m.id, m.name]));
      paints = data.paints.map(([id, mfrId, name, code, line, r, g, b]) => ({
        id,
        manufacturerId: mfrId,
        manufacturer: mfrName.get(mfrId) ?? '',
        name,
        code,
        line,
        r,
        g,
        b,
      }));
      byId = new Map(paints.map(p => [p.id, p]));
      searchIndex = paints.map(p => normalize(`${p.name} ${p.code} ${p.manufacturer}`));
    })();
  }
  return loadPromise;
}

/** JSON bruto do catálogo — repassado ao init do WASM sem re-serializar. */
export function catalogJSON(): string {
  return rawJSON;
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

/** Busca por nome/código/marca. Ranking: prefixo de código > prefixo de nome
 *  > substring — o pintor digita "70.9" ou "meph", não frases. */
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

  const codeHits: Paint[] = [];
  const nameHits: Paint[] = [];
  const subHits: Paint[] = [];
  const scan = (i: number) => {
    const p = paints[i];
    if (normalize(p.code).startsWith(q)) {
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

  return [...codeHits, ...nameHits, ...subHits].slice(0, limit);
}

export function hexOf(p: { r: number; g: number; b: number }): string {
  const h = (n: number) => n.toString(16).padStart(2, '0').toUpperCase();
  return `#${h(p.r)}${h(p.g)}${h(p.b)}`;
}
