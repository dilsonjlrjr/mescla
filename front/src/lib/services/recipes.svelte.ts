// T3 "Receitas" — persistência local (mesmo padrão de stock.svelte.ts):
// o cabeçalho da spec rf-04 lista as rotas HTTP já em uso (GET
// /recipes/by-paint, /recipes/by-color…) mas NÃO localizou uma rota de
// ESCRITA para receita contra api/httpapi (pendência 3, R1 herdado) — então
// esta tela nasce 100% local (localStorage), sem inventar rota.
//
// Decisão de semântica (RG-16, "Não faz" da spec — permitido resolver aqui
// se sair natural): guarda o ALVO, não a fórmula. Reabrir uma receita chama
// de novo suggestEquivalentRecipe/suggestRecipeForColor no fabricante
// escolhido — é o comportamento "correto" descrito no produto (US-15
// "a fórmula pode mudar, porque o alvo é resolvido novamente").

export interface Recipe {
  id: number;
  name: string;
  /** id do pote casado no catálogo — null quando o alvo é hex livre (sem pote). */
  targetPaintId: number | null;
  targetR: number;
  targetG: number;
  targetB: number;
  /** último fabricante usado para reproduzir (reaberto com este selecionado). */
  manufacturerId: number;
  createdAt: string;
}

const KEY = 'mescla.recipes.v1';
const SEQ_KEY = 'mescla.recipes.seq.v1';

function load(): Recipe[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const list = JSON.parse(raw);
    return Array.isArray(list) ? list : [];
  } catch {
    return [];
  }
}

let seq = (() => {
  const raw = Number(localStorage.getItem(SEQ_KEY));
  return Number.isFinite(raw) && raw > 0 ? raw : 0;
})();

function nextId(): number {
  seq += 1;
  try {
    localStorage.setItem(SEQ_KEY, String(seq));
  } catch {
    /* storage indisponível */
  }
  return seq;
}

export const recipes = $state({
  items: load() as Recipe[],
  /** id da receita recém-salva — T3 lê e limpa ao selecionar (US-15 CA). */
  lastSavedId: null as number | null,
});

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify(recipes.items));
  } catch {
    /* estoque cheio/indisponível: a receita só não persiste */
  }
}

export function saveRecipe(data: Omit<Recipe, 'id' | 'createdAt'>): Recipe {
  const r: Recipe = { ...data, id: nextId(), createdAt: new Date().toISOString() };
  recipes.items.unshift(r);
  recipes.lastSavedId = r.id;
  persist();
  return r;
}

export function removeRecipe(id: number) {
  const i = recipes.items.findIndex(r => r.id === id);
  if (i !== -1) {
    recipes.items.splice(i, 1);
    persist();
  }
}

export function updateRecipeManufacturer(id: number, manufacturerId: number) {
  const r = recipes.items.find(x => x.id === id);
  if (r) {
    r.manufacturerId = manufacturerId;
    persist();
  }
}
