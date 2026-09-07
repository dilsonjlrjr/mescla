import { describe, it, expect, vi, beforeEach } from 'vitest';
import { suggestRecipeForColor, melhorDeltaE, recipeForColorInBrand, ehUniversoVazio, type EquivalentRecipe, type UniversoVazio } from '../src/lib/services/engine';

vi.mock('../src/lib/services/api', () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
}));

import { apiGet } from '../src/lib/services/api';

function recipe(over: Partial<EquivalentRecipe> = {}): EquivalentRecipe {
  return {
    sourcePaintId: 0, sourceName: 'cor alvo', sourceManufacturer: '',
    sourceR: 10, sourceG: 20, sourceB: 30,
    targetManufacturer: 'Acrilex',
    ingredients: [], resultR: 10, resultG: 20, resultB: 30,
    deltaE: 2.4, method: 'kubelka-munk', reproducible: true,
    tips: [], faixa: 'otimo', foraDoUniverso: false,
    ...over,
  };
}

describe('rf-11 — engine.ts sobre o universo', () => {
  beforeEach(() => vi.mocked(apiGet).mockReset());

  it('suggestRecipeForColor monta a URL só com os parâmetros presentes', async () => {
    vi.mocked(apiGet).mockResolvedValue(recipe());

    await suggestRecipeForColor(10, 20, 30, { targetManufacturerId: 5, useStockOnly: true });

    const url = vi.mocked(apiGet).mock.calls[0][0] as string;
    expect(url).toContain('/recipes/by-color?');
    expect(url).toContain('targetManufacturerId=5');
    expect(url).toContain('useStockOnly=1');
  });

  it('suggestRecipeForColor omite targetManufacturerId quando ausente ou 0', async () => {
    vi.mocked(apiGet).mockResolvedValue(recipe());

    await suggestRecipeForColor(10, 20, 30, {});

    const url = vi.mocked(apiGet).mock.calls[0][0] as string;
    expect(url).not.toContain('targetManufacturerId');
  });

  it('inclui foraDoUniverso na URL quando true', async () => {
    vi.mocked(apiGet).mockResolvedValue(recipe());

    await suggestRecipeForColor(10, 20, 30, { foraDoUniverso: true });

    const url = vi.mocked(apiGet).mock.calls[0][0] as string;
    expect(url).toContain('foraDoUniverso=1');
  });

  it('ehUniversoVazio distingue receita de universo vazio', () => {
    const vazio: UniversoVazio = { universoVazio: true, motivo: 'Seu estoque está vazio' };
    expect(ehUniversoVazio(vazio)).toBe(true);
    expect(ehUniversoVazio(recipe())).toBe(false);
  });

  it('melhorDeltaE devolve o número quando a rota responde deltaE', async () => {
    vi.mocked(apiGet).mockResolvedValue({ deltaE: 3.7 });

    const d = await melhorDeltaE(10, 20, 30, { targetManufacturerId: 2 });

    expect(d).toBe(3.7);
    const url = vi.mocked(apiGet).mock.calls[0][0] as string;
    expect(url).toContain('/recipes/best-delta-e?');
  });

  it('melhorDeltaE devolve null quando o universo está vazio', async () => {
    vi.mocked(apiGet).mockResolvedValue({ universoVazio: true, motivo: 'x' });

    expect(await melhorDeltaE(10, 20, 30, {})).toBeNull();
  });

  it('recipeForColorInBrand devolve a receita quando o universo não está vazio', async () => {
    vi.mocked(apiGet).mockResolvedValue(recipe({ targetManufacturer: 'Citadel' }));

    const r = await recipeForColorInBrand(10, 20, 30, 7);

    expect(r.targetManufacturer).toBe('Citadel');
  });

  it('recipeForColorInBrand lança com o motivo quando o universo está vazio', async () => {
    vi.mocked(apiGet).mockResolvedValue({ universoVazio: true, motivo: 'Citadel não tem tintas com cor cadastrada' });

    await expect(recipeForColorInBrand(10, 20, 30, 7)).rejects.toThrow('Citadel não tem tintas com cor cadastrada');
  });
});
