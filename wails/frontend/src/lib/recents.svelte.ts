// Histórico local: tintas buscadas recentemente + últimas mesclas executadas.
// Na loja o pintor repete consultas — o histórico corta a digitação.

const RECENTS_KEY = 'mescla.recents.v1';
const MESCLAS_KEY = 'mescla.mesclas.v1';

export interface RecentMescla {
  sourceId: number;
  brandIds: number[];
  /** resumo pro card do histórico (sem recomputar): */
  sourceName: string;
  targetManufacturer: string;
  deltaE: number;
  sourceRGB: [number, number, number];
  resultRGB: [number, number, number];
}

function load<T>(key: string): T[] {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T[]) : [];
  } catch {
    return [];
  }
}

function save(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* sem persistência, sem drama */
  }
}

export const recents = $state({
  paintIds: load<number>(RECENTS_KEY),
  mesclas: load<RecentMescla>(MESCLAS_KEY),
});

export function rememberPaint(id: number) {
  const i = recents.paintIds.indexOf(id);
  if (i !== -1) recents.paintIds.splice(i, 1);
  recents.paintIds.unshift(id);
  recents.paintIds.length = Math.min(recents.paintIds.length, 8);
  save(RECENTS_KEY, recents.paintIds);
}

export function rememberMescla(m: RecentMescla) {
  const i = recents.mesclas.findIndex(
    x => x.sourceId === m.sourceId && x.targetManufacturer === m.targetManufacturer,
  );
  if (i !== -1) recents.mesclas.splice(i, 1);
  recents.mesclas.unshift(m);
  recents.mesclas.length = Math.min(recents.mesclas.length, 3);
  save(MESCLAS_KEY, recents.mesclas);
}
