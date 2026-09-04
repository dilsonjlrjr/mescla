// Receitas salvas — store em localStorage (chave mescla.recipes.v1).
// A Home lista; a Equivalência salva; "abrir" re-executa a receita
// (SuggestEquivalentRecipe com sourcePaintId + targetManufacturerId).

export interface SavedRecipeIngredient {
  name: string;
  code: string;
  hex: string;
  percentage: number;
}

export interface SavedRecipe {
  id: number;
  sourcePaintId: number;
  sourceName: string;
  sourceHex: string;
  targetManufacturerId: number;
  targetManufacturer: string;
  ingredients: SavedRecipeIngredient[];
  deltaE: number;
  savedAt: string;
}

const KEY = 'mescla.recipes.v1';

function load(): SavedRecipe[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export const recipes: SavedRecipe[] = $state(load());

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify(recipes));
  } catch {
    // storage cheio/indisponível: a lista continua em memória
  }
}

export function saveRecipe(r: Omit<SavedRecipe, 'id' | 'savedAt'>): SavedRecipe {
  const item: SavedRecipe = {
    ...r,
    id: Date.now(),
    savedAt: new Date().toISOString(),
  };
  // mesma origem + mesmo destino substitui (não acumula duplicata)
  const dup = recipes.findIndex(
    x => x.sourcePaintId === item.sourcePaintId && x.targetManufacturerId === item.targetManufacturerId
  );
  if (dup !== -1) recipes.splice(dup, 1);
  recipes.unshift(item);
  persist();
  return item;
}

export function removeRecipe(id: number) {
  const idx = recipes.findIndex(r => r.id === id);
  if (idx !== -1) {
    recipes.splice(idx, 1);
    persist();
  }
}

/** Em quantas receitas salvas um ingrediente aparece (match por nome, e por
 * código quando o ingrediente salvo tem código). Usado no painel do Estoque. */
export function recipesWithIngredient(manufacturer: string, code: string, name: string): number {
  const codeKey = code.trim().toLowerCase();
  const nameKey = name.trim().toLowerCase();
  const mfrKey = manufacturer.trim().toLowerCase();
  let n = 0;
  for (const r of recipes) {
    const sameMfr = r.targetManufacturer.trim().toLowerCase() === mfrKey;
    if (!sameMfr) continue;
    const hit = r.ingredients.some(i => {
      const iCode = (i.code || '').trim().toLowerCase();
      if (codeKey && iCode) return iCode === codeKey;
      return i.name.trim().toLowerCase() === nameKey;
    });
    if (hit) n++;
  }
  return n;
}
