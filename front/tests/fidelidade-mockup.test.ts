// D-001 (Epic D) — teste de reprodução do defeito "interface diverge do mockup".
//
// Oráculo: o protótipo aprovado `docs/oficial/Mescla AI.html` (Nocturne), cujas 4 telas foram
// extraídas para `tests/fixtures/mockup/*.html`. `docs/oficial/ESPECIFICACAO.md` declara o
// protótipo fonte de verdade de comportamento e layout — então a métrica que está lá é contrato.
//
// Cada marco é verificado DUAS vezes:
//   1. guarda — a declaração existe mesmo no protótipo (se o protótipo mudar, o teste acusa);
//   2. asserção — a tela renderizada carrega a mesma declaração em algum elemento.
//
// Este teste nasceu VERMELHO (as telas usavam o sistema visual anterior) e fica na suíte como
// regressão.

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const here = dirname(fileURLToPath(import.meta.url));
const fixture = (name: string) => readFileSync(join(here, 'fixtures', 'mockup', name), 'utf8');

// ── mocks: TDD unit-only, nenhuma chamada HTTP real ──────────────────────────
const PAINTS = [
  { id: 1, manufacturerId: 1, manufacturer: 'Vallejo', name: 'Sombra', code: '70.951', line: 'Model Color', r: 30, g: 40, b: 60 },
  { id: 2, manufacturerId: 1, manufacturer: 'Vallejo', name: 'Ocre', code: '70.914', line: 'Model Color', r: 190, g: 150, b: 70 },
  { id: 3, manufacturerId: 2, manufacturer: 'Citadel', name: 'Mephiston', code: 'C1', line: 'Base', r: 150, g: 30, b: 30 },
];
const MFRS = [
  { id: 1, name: 'Vallejo', paintCount: 2, userPaintCount: 0 },
  { id: 2, name: 'Citadel', paintCount: 1, userPaintCount: 0 },
];

vi.mock('../src/lib/services/catalog', () => ({
  loadCatalog: () => Promise.resolve(),
  allPaints: () => PAINTS,
  allManufacturers: () => MFRS,
  paintById: (id: number) => PAINTS.find(p => p.id === id),
  searchPaints: () => PAINTS,
  hexOf: (p: { r: number; g: number; b: number }) =>
    `#${[p.r, p.g, p.b].map(n => n.toString(16).padStart(2, '0')).join('').toUpperCase()}`,
  reloadManufacturers: () => Promise.resolve(MFRS),
  migrateLegacyManufacturers: () => Promise.resolve(0),
  createManufacturer: () => Promise.reject(new Error('não usado')),
  updateManufacturer: () => Promise.reject(new Error('não usado')),
  deleteManufacturer: () => Promise.reject(new Error('não usado')),
  MAX_MANUFACTURER_NAME: 80,
}));

const RECIPE = {
  sourcePaintId: 1,
  sourceName: 'Sombra',
  sourceManufacturer: 'Vallejo',
  sourceR: 30, sourceG: 40, sourceB: 60,
  targetManufacturer: 'Citadel',
  ingredients: [
    { paintId: 3, name: 'Mephiston', code: 'C1', percentage: 0.7, r: 150, g: 30, b: 30 },
    { paintId: 2, name: 'Ocre', code: '70.914', percentage: 0.3, r: 190, g: 150, b: 70 },
  ],
  resultR: 32, resultG: 41, resultB: 58,
  deltaE: 1.4,
  method: 'mix',
  reproducible: true,
  tips: [],
};

vi.mock('../src/lib/services/engine', () => ({
  engineReady: () => Promise.resolve(),
  findSimilar: () => Promise.resolve([]),
  suggestEquivalentRecipe: () => Promise.resolve(RECIPE),
  suggestRecipeForColor: () => Promise.resolve(RECIPE),
  ehUniversoVazio: (v: unknown) => (v as { universoVazio?: boolean })?.universoVazio === true,
  melhorDeltaE: () => Promise.resolve(null),
  suggestFromStock: () => Promise.resolve(RECIPE),
  bestBrandsFor: () => Promise.resolve([]),
  compareToAnchor: () => Promise.resolve([]),
  parseStockCSV: () => Promise.resolve({ paints: [], errors: [] }),
  stockCSVTemplate: () => Promise.resolve(''),
  stockToCSV: () => Promise.resolve(''),
}));

/** Todas as declarações inline (`prop: valor`) presentes num HTML. */
function declarations(html: string): Set<string> {
  const out = new Set<string>();
  for (const m of html.matchAll(/style="([^"]*)"/g)) {
    for (const d of m[1].split(';')) {
      const t = d.trim();
      if (!t) continue;
      const i = t.indexOf(':');
      if (i === -1) continue;
      out.add(`${t.slice(0, i).trim()}: ${t.slice(i + 1).trim()}`);
    }
  }
  return out;
}

/** As declarações que a tela renderizada carrega (inline, como no protótipo). */
function renderedDeclarations(container: HTMLElement): Set<string> {
  return declarations(container.innerHTML);
}

interface Marco {
  tela: string;
  arquivo: string;
  componente: () => Promise<{ default: unknown }>;
  fonte: string;
  /** marcos que a tela mostra sem interação nenhuma — verificados no DOM */
  marcos: string[];
  /** marcos presos atrás de estado (ex.: a fórmula só existe depois de uma
   *  mistura calculada) — verificados no código-fonte da tela */
  marcosEmEstado: string[];
}

const TELAS: Marco[] = [
  {
    tela: 'T1 — Pergunta e resposta',
    arquivo: 't1-pergunta.html',
    componente: () => import('../src/lib/views/PerguntaView.svelte'),
    fonte: 'src/lib/views/PerguntaView.svelte',
    marcos: [
      'min-height: 76px',                                        // cabeçalho fixo
      'height: 54px',                                            // campo de busca
      'height: 60px',                                            // barra de filtros
      'grid-template-columns: minmax(0, 38%) minmax(0, 1fr)',    // miolo em 2 colunas
      'height: 108px',                                           // rodapé "Na mesa hoje"
    ],
    marcosEmEstado: [
      'width: 96px',                                             // campo de gotas do stepper
      'height: 56px',                                            // botões −/+ do stepper
      'height: 12px',                                            // fita de proporção
    ],
  },
  {
    tela: 'T2 — Plano da peça',
    arquivo: 't2-plano.html',
    componente: () => import('../src/lib/views/PlannerView.svelte'),
    fonte: 'src/lib/views/PlannerView.svelte',
    marcos: [
      'min-height: 76px',
      'height: 52px',
      'height: 6px',                                             // barra de progresso pintada
    ],
    marcosEmEstado: [
      'grid-template-columns: minmax(0, 1fr) minmax(0, 34%)',    // foto | painel
    ],
  },
  {
    tela: 'T3 — Receitas',
    arquivo: 't3-receitas.html',
    componente: () => import('../src/lib/views/ReceitasView.svelte'),
    fonte: 'src/lib/views/ReceitasView.svelte',
    marcos: [
      'min-height: 76px',
      'grid-template-columns: minmax(0, 34%) minmax(0, 1fr)',
      'height: 72px',                                            // par alvo/resultado
    ],
    marcosEmEstado: [
      'height: 48px',                                            // pílula de fabricante
    ],
  },
  {
    tela: 'T4 — Minhas tintas',
    arquivo: 't4-tintas.html',
    componente: () => import('../src/lib/views/CatalogoView.svelte'),
    fonte: 'src/lib/views/CatalogoView.svelte',
    marcos: [
      'min-height: 76px',
      'height: 50px',                                            // abas Tintas/Fabricantes
      'height: 58px',                                            // ação do formulário
    ],
    marcosEmEstado: [
      'padding: 24px',                                           // card de modal
    ],
  },
];

describe('D-001 — fidelidade ao protótipo Nocturne', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  for (const tela of TELAS) {
    describe(tela.tela, () => {
      const doProtótipo = declarations(fixture(tela.arquivo));

      it('o protótipo declara os marcos verificados (guarda do oráculo)', () => {
        for (const marco of [...tela.marcos, ...tela.marcosEmEstado]) {
          expect(doProtótipo, `protótipo ${tela.arquivo} sem "${marco}"`).toContain(marco);
        }
      });

      it('a tela renderizada carrega os marcos do protótipo', async () => {
        const mod = (await tela.componente()) as { default: never };
        const { container } = render(mod.default);
        const daTela = renderedDeclarations(container);
        const faltando = tela.marcos.filter(m => !daTela.has(m));
        expect(faltando, `${tela.tela}: marcos ausentes na tela`).toEqual([]);
      });

      it('a tela declara os marcos presos atrás de estado', () => {
        const src = readFileSync(join(here, '..', tela.fonte), 'utf8');
        const faltando = tela.marcosEmEstado.filter(m => !src.includes(m));
        expect(faltando, `${tela.tela}: marcos ausentes no código da tela`).toEqual([]);
      });
    });
  }
});
