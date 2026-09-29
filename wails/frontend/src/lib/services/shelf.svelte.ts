// "Minha estante" — as marcas que o pintor tem em casa, persistidas em
// localStorage. É o que transforma a jornada da 2ª visita em 1 busca + 1 tap:
// o campo "Tenho tintas de" já abre preenchido.

const KEY = 'mescla.shelf.v1';

function load(): number[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const ids = JSON.parse(raw);
    return Array.isArray(ids) ? ids.filter(n => typeof n === 'number') : [];
  } catch {
    return [];
  }
}

export const shelf = $state({ manufacturerIds: load() });

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify(shelf.manufacturerIds));
  } catch {
    /* storage cheio/indisponível: a estante só não persiste */
  }
}

export function toggleShelf(manufacturerId: number) {
  const i = shelf.manufacturerIds.indexOf(manufacturerId);
  if (i === -1) {
    shelf.manufacturerIds.push(manufacturerId);
  } else {
    shelf.manufacturerIds.splice(i, 1);
  }
  persist();
}

export function inShelf(manufacturerId: number): boolean {
  return shelf.manufacturerIds.includes(manufacturerId);
}
