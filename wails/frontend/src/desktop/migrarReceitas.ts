// Receitas salvas pela interface antiga do desktop. Ela usava a mesma chave
// `mescla.recipes.v1` da web, com outra forma: origem do catálogo, hex e
// fabricante alvo. A interface da web lê alvo em RGB e fabricante escolhido
// (lib/services/recipes.svelte.ts). Roda uma vez no boot, antes de montar a
// tela: converte o que estiver na forma antiga e deixa o resto como está.

const KEY = 'mescla.recipes.v1';
const SEQ_KEY = 'mescla.recipes.seq.v1';

interface ReceitaAntiga {
  id: number;
  sourcePaintId: number;
  sourceName?: string;
  sourceHex: string;
  targetManufacturerId: number;
  savedAt?: string;
}

function antiga(x: unknown): x is ReceitaAntiga {
  const r = x as Partial<ReceitaAntiga> | null;
  return !!r && typeof r.sourceHex === 'string' && !('targetR' in (r as object));
}

function hexParaRgb(hex: string): [number, number, number] | null {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const n = Number.parseInt(m[1], 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

export function migrarReceitasAntigas(): void {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return;
    const lista = JSON.parse(raw);
    if (!Array.isArray(lista) || !lista.some(antiga)) return;

    const convertidas = lista.flatMap((r: unknown) => {
      if (!antiga(r)) return [r];
      const rgb = hexParaRgb(r.sourceHex);
      if (!rgb) return [];
      return [{
        id: r.id,
        name: r.sourceName ?? r.sourceHex.toUpperCase(),
        targetPaintId: r.sourcePaintId > 0 ? r.sourcePaintId : null,
        targetR: rgb[0],
        targetG: rgb[1],
        targetB: rgb[2],
        manufacturerId: r.targetManufacturerId,
        createdAt: r.savedAt ?? new Date().toISOString(),
      }];
    });
    localStorage.setItem(KEY, JSON.stringify(convertidas));

    const maiorId = convertidas.reduce<number>((m, r) => Math.max(m, Number((r as { id?: unknown }).id) || 0), 0);
    if (Number(localStorage.getItem(SEQ_KEY)) < maiorId) localStorage.setItem(SEQ_KEY, String(maiorId));
  } catch {
    // storage indisponível ou lixo: a tela de Receitas já tolera lista vazia
  }
}
