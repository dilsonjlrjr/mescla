// rf-13 — T1 cross-brand e fluxo em dois passos: testes da lógica pura
// extraída em PerguntaView.svelte (bloco `<script module>`), sem montar o
// componente. Mocka fetch/localStorage nunca é necessário aqui — são funções
// puras, sem I/O.

import { describe, it, expect } from 'vitest';
import {
  pctToMl, countManufacturers, crossBrandTitle, isCurrentRequest, computeDropsFromIngredients, repartir,
} from '../src/lib/views/PerguntaView.svelte';
import { dict, type Lang } from '../src/lib/i18n.svelte';

describe('rf-13 — pctToMl (RN10/CA10: percentual × volume alvo, arredondado a 0,05 ml)', () => {
  it('60% de 10 ml dá 6,00 ml', () => {
    expect(pctToMl(60, 10)).toBeCloseTo(6.0, 5);
  });

  it('40% de 10 ml dá 4,00 ml', () => {
    expect(pctToMl(40, 10)).toBeCloseTo(4.0, 5);
  });

  it('arredonda pra 0,05 ml mais próximo (uma gota)', () => {
    // 33% de 5 ml = 1.65 ml exato — já múltiplo de 0.05, fica como está.
    expect(pctToMl(33, 5)).toBeCloseTo(1.65, 5);
    // 17% de 5 ml = 0.85 ml — múltiplo de 0.05, mantém.
    expect(pctToMl(17, 5)).toBeCloseTo(0.85, 5);
  });

  it('escala com o volume alvo (5/10/20 ml, RN10)', () => {
    expect(pctToMl(50, 5)).toBeCloseTo(2.5, 5);
    expect(pctToMl(50, 10)).toBeCloseTo(5.0, 5);
    expect(pctToMl(50, 20)).toBeCloseTo(10.0, 5);
  });
});

describe('rf-13 — countManufacturers (CA19: correção do bug que contava nome de tinta)', () => {
  it('conta fabricantes distintos por manufacturerId, mesmo com tintas de nomes diferentes', () => {
    const ingredients = [
      { manufacturerId: 1 },
      { manufacturerId: 2 },
    ];
    expect(countManufacturers(ingredients)).toBe(2);
  });

  it('NÃO conta por nome de tinta — dois ingredientes do MESMO fabricante valem 1', () => {
    // Regressão do bug em PerguntaView.svelte:376-379: contar Set(nomes) daria
    // 2 aqui (nomes diferentes), mas o fabricante é o mesmo — correto é 1.
    const ingredients = [
      { manufacturerId: 5 },
      { manufacturerId: 5 },
    ];
    expect(countManufacturers(ingredients)).toBe(1);
  });

  it('fórmula de um ingrediente só vale 1 fabricante', () => {
    expect(countManufacturers([{ manufacturerId: 3 }])).toBe(1);
  });
});

describe('rf-13 — crossBrandTitle (CA2/RG-13: fabricantes unidos por " + ")', () => {
  it('une os fabricantes de `manufacturers` com " + "', () => {
    expect(crossBrandTitle(['Citadel', 'Mr. Color'])).toBe('Citadel + Mr. Color');
  });

  it('um fabricante só não tem separador', () => {
    expect(crossBrandTitle(['Vallejo'])).toBe('Vallejo');
  });

  it('lista vazia dá string vazia (defensivo)', () => {
    expect(crossBrandTitle([])).toBe('');
  });
});

describe('rf-13 — isCurrentRequest (CAN7: resposta obsoleta é descartada)', () => {
  it('token igual ao mais recente é aplicado', () => {
    expect(isCurrentRequest(2, 2)).toBe(true);
  });

  it('token antigo é descartado quando um mais novo já chegou', () => {
    expect(isCurrentRequest(1, 2)).toBe(false);
  });

  it('simula duas requisições em voo: a que chega depois só é descartada se não for a mais recente', async () => {
    let latest = 0;
    const applied: string[] = [];

    function lancar(label: string, atraso: number) {
      const token = ++latest;
      return new Promise<void>(resolve => {
        setTimeout(() => {
          if (isCurrentRequest(token, latest)) applied.push(label);
          resolve();
        }, atraso);
      });
    }

    // "mix" dispara primeiro mas demora mais; "marca" dispara depois e chega antes.
    const p1 = lancar('mix', 20);
    const p2 = lancar('marca', 5);
    await Promise.all([p1, p2]);

    expect(applied).toEqual(['marca']);
  });
});

// ── Regressões do guardrail do rf-13 (2026-09-08) ──
// Cada teste abaixo nasceu de um achado com artefato das lentes cegas.


describe('rf-13 — regressões do guardrail', () => {
  it('proporção coprima acima de 40 gotas é reescalada, não truncada', () => {
    // Achado: 57/43 (g=1) virava 40/40 — 50%/50%, uma mistura que não é a do
    // ΔE00 exibido. A razão precisa sobreviver ao teto de 40 gotas.
    const drops = computeDropsFromIngredients([{ percentage: 57 }, { percentage: 43 }]);
    expect(Math.max(...drops)).toBeLessThanOrEqual(40);
    const total = drops[0] + drops[1];
    expect((drops[0] / total) * 100).toBeGreaterThan(54);
    expect((drops[0] / total) * 100).toBeLessThan(60);
  });

  it('proporção que já cabe em 40 gotas não é mexida', () => {
    expect(computeDropsFromIngredients([{ percentage: 60 }, { percentage: 40 }])).toEqual([3, 2]);
  });

  it('as chaves novas do rf-13 existem nos quatro idiomas de LANGS', () => {
    // Achado: entraram só em pt/en; es e fr caíam no fallback português.
    const novas = [
      'universeLabel', 'universoMix', 'universoBrand', 'chooseBrandAria', 'chooseSupplierPlaceholder',
      'searchEquivalenceBtn', 'targetVolumeLabel', 'voltarBtn', 'sheetTitle',
      'tapToCalc', 'saveDisabledCrossBrand',
    ];
    for (const lang of Object.keys(dict) as Lang[]) {
      for (const k of novas) {
        expect(`${lang}.${k}=${(dict[lang] as Record<string, string>)[k] ?? ''}`).not.toBe(`${lang}.${k}=`);
      }
    }
  });

  it('idiomas distintos não compartilham a mesma string de universo', () => {
    const vals = (Object.keys(dict) as Lang[]).map(l => (dict[l] as Record<string, string>).universoMix);
    expect(new Set(vals).size).toBe(vals.length);
  });
});

describe('rf-13 — regressões do guardrail, rodada 2', () => {
  it('percentuais exibidos somam exatamente 100', () => {
    // Achado: 3 componentes iguais viravam "33% 33% 33%", soma 99.
    for (const pcts of [[1, 1, 1], [40, 40, 20], [57, 43], [100], [33.4, 33.3, 33.3]]) {
      expect(repartir(pcts, 100, 1).reduce((a, b) => a + b, 0)).toBe(100);
    }
  });

  it('as parcelas em ml somam exatamente o volume alvo', () => {
    // Achado: 3 linhas de 3,35 ml somavam 10,05 contra "Total 10,00 ml".
    for (const alvo of [5, 10, 20]) {
      for (const pcts of [[1, 1, 1], [60, 40], [50, 30, 20]]) {
        const soma = repartir(pcts, alvo, 0.05).reduce((a, b) => a + b, 0);
        expect(Math.round(soma * 100) / 100).toBe(alvo);
      }
    }
  });

  it('cada parcela em ml é múltipla de 0,05 (uma gota)', () => {
    for (const v of repartir([1, 1, 1], 10, 0.05)) {
      expect(Math.round((v / 0.05) * 1e6) % 1e6).toBe(0);
    }
  });

  it('repartir preserva a ordem de grandeza dos componentes', () => {
    const [a, b] = repartir([60, 40], 100, 1);
    expect(a).toBeGreaterThan(b);
  });
});
