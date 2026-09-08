// rf-09 — abas de figura no plano: contrato por aba em plans.ts e migração
// v1 → v2 do rascunho local em rascunho.ts. TDD unit-only: fetch e
// localStorage mockados, nenhuma rede nem servidor de verdade.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { validarPlano, salvarPlano, type PlanoDTO, type AbaDTO, type RegiaoDTO } from '../src/lib/services/plans';
import { lerRascunho, type Rascunho } from '../src/lib/planner/rascunho';

function regiao(overrides: Partial<RegiaoDTO> = {}): RegiaoDTO {
  return {
    x: 1, y: 1, r: 0, g: 0, b: 0, hex: '#000000',
    regionName: 'r', note: '',
    paintId: null, paintBrand: '', paintName: '', paintCode: '',
    deltaE: 0, painted: 0,
    ...overrides,
  };
}

function aba(overrides: Partial<AbaDTO> = {}): AbaDTO {
  return {
    name: 'Figura 1', imageData: '', regions: [],
    ...overrides,
  };
}

function plano(overrides: Partial<PlanoDTO> = {}): PlanoDTO {
  return { name: 'Plano', useStockOnly: 0, tabs: [aba()], ...overrides };
}

describe('validarPlano — por aba (plans.ts)', () => {
  it('CA8 — 11ª aba é negada com errTabsMax, sem citar aba', () => {
    const dto = plano({ tabs: Array.from({ length: 11 }, (_, i) => aba({ name: `F${i}` })) });
    expect(validarPlano(dto)).toEqual({ chave: 'errTabsMax' });
  });

  it('CA9/RN15 — 51ª região numa aba cita o nome dela em errRegionsMax', () => {
    const dto = plano({
      tabs: [aba({ name: 'Costas', regions: Array.from({ length: 51 }, () => regiao()) })],
    });
    expect(validarPlano(dto)).toEqual({ chave: 'errRegionsMax', abaNome: 'Costas' });
  });

  it('CA24 — nome de aba com 81 caracteres devolve errTabNameMax', () => {
    const dto = plano({ tabs: [aba({ name: 'x'.repeat(81) })] });
    const erro = validarPlano(dto);
    expect(erro?.chave).toBe('errTabNameMax');
  });

  it('CA25 — foto de 2,5 MB numa aba cita o nome dela em errImageMax', () => {
    const dataUrl = 'data:image/png;base64,' + 'A'.repeat(3 * 1024 * 1024);
    const dto = plano({ tabs: [aba({ name: 'Frente', imageData: dataUrl })] });
    expect(validarPlano(dto)).toEqual({ chave: 'errImageMax', abaNome: 'Frente' });
  });

  it('CAN5-equivalente de validação — regionName > 100 cita a aba', () => {
    const dto = plano({
      tabs: [aba({ name: 'Base', regions: [regiao({ regionName: 'n'.repeat(101) })] })],
    });
    expect(validarPlano(dto)).toEqual({ chave: 'errRegionNameMax', abaNome: 'Base' });
  });

  it('note > 2000 cita a aba', () => {
    const dto = plano({
      tabs: [aba({ name: 'Base', regions: [regiao({ note: 'n'.repeat(2001) })] })],
    });
    expect(validarPlano(dto)).toEqual({ chave: 'errNoteMax', abaNome: 'Base' });
  });

  it('nome do plano vazio continua sem citar aba (errPlanNameEmpty)', () => {
    const dto = plano({ name: '' });
    expect(validarPlano(dto)).toEqual({ chave: 'errPlanNameEmpty' });
  });

  it('plano válido com várias abas não devolve erro', () => {
    const dto = plano({ tabs: [aba({ name: 'Frente' }), aba({ name: 'Costas' })] });
    expect(validarPlano(dto)).toBeNull();
  });
});

describe('salvarPlano — conta regiões através das abas (D-003)', () => {
  const originalFetch = globalThis.fetch;
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it('soma regiões de todas as abas para a suspeita de D-003', async () => {
    const dto = plano({
      tabs: [aba({ regions: [regiao(), regiao()] }), aba({ regions: [regiao()] })],
    });
    const respostaComMenosRegioes: PlanoDTO = {
      ...dto,
      tabs: [aba({ regions: [regiao()] }), aba({ regions: [] })],
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => respostaComMenosRegioes,
    }) as unknown as typeof fetch;

    const { suspeitaD003 } = await salvarPlano(dto);
    expect(suspeitaD003).toBe(true);
  });
});

describe('lerRascunho — migração v1 → v2 (rascunho.ts)', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('CA17 — rascunho v1 vira aba Figura 1 (selectedManufacturerId migra para a raiz — mudança macro 07/09/2026)', () => {
    const v1 = {
      planId: 7,
      name: 'Miniatura',
      imageData: 'data:image/png;base64,AAA',
      selectedManufacturerId: 3,
      regions: [{ x: 1, y: 2, regionName: 'Manto', hex: '#7A1F2B', painted: 1 }],
      salvoEm: '2026-09-06T10:00:00.000Z',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(v1));

    const { rascunho, corrompido, truncado } = lerRascunho();
    expect(corrompido).toBe(false);
    expect(truncado).toBe(false);
    expect(rascunho).not.toBeNull();
    const r = rascunho as Rascunho;
    expect(r.v).toBe(2);
    expect(r.planId).toBe(7);
    expect(r.tabs).toHaveLength(1);
    expect(r.tabs[0].name).toBe('Figura 1');
    expect(r.tabs[0].imageData).toBe('data:image/png;base64,AAA');
    expect(r.selectedManufacturerId).toBe(3);
    expect(r.tabs[0].regions).toHaveLength(1);
    expect(r.tabs[0].regions[0].regionName).toBe('Manto');
    expect(r.tabs[0].regions[0].painted).toBe(true);
  });

  it('RN11 — rascunho v1 nunca é descartado em silêncio (não vira corrompido)', () => {
    localStorage.setItem(
      'mescla:plano-rascunho',
      JSON.stringify({ name: 'X', imageData: '', regions: [], salvoEm: 'x' })
    );
    const { rascunho, corrompido } = lerRascunho();
    expect(corrompido).toBe(false);
    expect(rascunho).not.toBeNull();
  });

  it('rascunho v2 já no formato novo é lido sem migração (useStockOnly na raiz — mudança macro 07/09/2026)', () => {
    const v2: Rascunho = {
      v: 2,
      planId: null,
      name: 'Plano',
      selectedManufacturerId: null,
      useStockOnly: true,
      abaAtiva: 1,
      tabs: [
        { name: 'Frente', imageData: '', regions: [] },
        { name: 'Costas', imageData: '', regions: [] },
      ],
      salvoEm: '2026-09-06T10:00:00.000Z',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(v2));

    const { rascunho, corrompido, truncado } = lerRascunho();
    expect(corrompido).toBe(false);
    expect(truncado).toBe(false);
    expect(rascunho?.tabs).toHaveLength(2);
    expect(rascunho?.abaAtiva).toBe(1);
    expect(rascunho?.useStockOnly).toBe(true);
  });

  it('CAN5 — rascunho adulterado com 40 abas é cortado em 10 com aviso', () => {
    const adulterado = {
      v: 2,
      planId: null,
      name: 'X',
      abaAtiva: 0,
      tabs: Array.from({ length: 40 }, (_, i) => ({
        name: `Aba ${i}`, imageData: '', selectedManufacturerId: null, useStockOnly: false, regions: [],
      })),
      salvoEm: 'x',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(adulterado));

    const { rascunho, corrompido, truncado } = lerRascunho();
    expect(corrompido).toBe(false);
    expect(truncado).toBe(true);
    expect(rascunho?.tabs).toHaveLength(10);
  });

  it('CAN5 — aba com mais de 50 regiões também é cortada com aviso', () => {
    const adulterado = {
      v: 2,
      planId: null,
      name: 'X',
      abaAtiva: 0,
      tabs: [
        {
          name: 'Cheia',
          imageData: '',
          selectedManufacturerId: null,
          useStockOnly: false,
          regions: Array.from({ length: 60 }, () => ({ x: 0, y: 0, regionName: 'r' })),
        },
      ],
      salvoEm: 'x',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(adulterado));

    const { rascunho, truncado } = lerRascunho();
    expect(truncado).toBe(true);
    expect(rascunho?.tabs[0].regions).toHaveLength(50);
  });

  it('CAN6 — rascunho v1 corrompido (JSON inválido) avisa e some, sem ficar em branco em silêncio', () => {
    localStorage.setItem('mescla:plano-rascunho', '{ isso não é json');
    const { rascunho, corrompido } = lerRascunho();
    expect(corrompido).toBe(true);
    expect(rascunho).toBeNull();
    // apagado — a tela cai para o plano do servidor, não reencontra o lixo.
    expect(localStorage.getItem('mescla:plano-rascunho')).toBeNull();
  });

  it('CAN10 — useStockOnly (raiz, mudança macro 07/09/2026) com "sim" ou 2 é normalizado para booleano, sem quebrar', () => {
    const base = { v: 2, planId: null, name: 'X', abaAtiva: 0, tabs: [{ name: 'A', imageData: '', regions: [] }], salvoEm: 'x' };

    localStorage.setItem('mescla:plano-rascunho', JSON.stringify({ ...base, useStockOnly: 'sim' }));
    expect(lerRascunho().rascunho?.useStockOnly).toBe(false);

    localStorage.setItem('mescla:plano-rascunho', JSON.stringify({ ...base, useStockOnly: 2 }));
    expect(lerRascunho().rascunho?.useStockOnly).toBe(false);

    localStorage.setItem('mescla:plano-rascunho', JSON.stringify({ ...base, useStockOnly: 1 }));
    expect(lerRascunho().rascunho?.useStockOnly).toBe(true);
  });

  it('planId e selectedManufacturerId (raiz) adulterados (negativo/fracionário) viram null', () => {
    const adulterado = {
      v: 2,
      planId: -7,
      name: 'X',
      selectedManufacturerId: 1.5,
      useStockOnly: false,
      abaAtiva: 0,
      tabs: [{ name: 'A', imageData: '', regions: [] }],
      salvoEm: 'x',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(adulterado));

    const { rascunho } = lerRascunho();
    expect(rascunho?.planId).toBeNull();
    expect(rascunho?.selectedManufacturerId).toBeNull();
  });

  it('imageData com prefixo fora da lista branca vira vazio', () => {
    const adulterado = {
      v: 2,
      planId: null,
      name: 'X',
      abaAtiva: 0,
      tabs: [{ name: 'A', imageData: 'data:text/html,<script>', regions: [] }],
      salvoEm: 'x',
    };
    localStorage.setItem('mescla:plano-rascunho', JSON.stringify(adulterado));

    const { rascunho } = lerRascunho();
    expect(rascunho?.tabs[0].imageData).toBe('');
  });
});
