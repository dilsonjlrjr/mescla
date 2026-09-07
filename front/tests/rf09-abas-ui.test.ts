// rf-09 — tira de abas de figura em T2 (PlannerView.svelte). A tira nasceu
// sem nenhum teste (apontado pelo guardrail); este arquivo cobre os CA da
// spec `brain-mescla-ai/requirements/rf-09-abas-figura.md` que dizem respeito
// à UI (os de contrato/rascunho já estão em rf09-plans-rascunho.test.ts).
//
// TDD unit-only: `fetch`, `localStorage`, catálogo e motor de sugestão são
// mockados — nenhum servidor. `<canvas>`/`Image` são dublados como em
// t2-plano-viewport.test.ts (D-002), único jeito de criar região em jsdom.

import { render, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import PlannerView from '../src/lib/views/PlannerView.svelte';
import { t, dict } from '../src/lib/i18n.svelte';
import { toasts } from '../src/lib/toast.svelte';
import { nav } from '../src/lib/nav.svelte';

const MFRS = [
  { id: 1, name: 'Vallejo', paintCount: 2 },
  { id: 2, name: 'Citadel', paintCount: 1 },
];

// Fila de respostas do "motor" — cada clique na foto consome uma entrada (ou
// `null`, região sem tinta parecida). `vi.hoisted` porque `vi.mock` é
// hoisted para o topo do arquivo.
const { engineQueue } = vi.hoisted(() => ({ engineQueue: [] as unknown[] }));

vi.mock('../src/lib/services/catalog', () => ({
  loadCatalog: () => Promise.resolve(),
  allPaints: () => [],
  allManufacturers: () => MFRS,
  paintById: () => undefined,
  searchPaints: () => [],
  hexOf: () => '#000000',
}));

vi.mock('../src/lib/services/engine', () => ({
  engineReady: () => Promise.resolve(),
  suggestRecipeForColor: () => Promise.resolve(engineQueue.length ? engineQueue.shift() : null),
}));

function recipe(over: Record<string, unknown> = {}) {
  return {
    sourcePaintId: 0, sourceName: '', sourceManufacturer: '',
    sourceR: 10, sourceG: 20, sourceB: 30,
    targetManufacturer: 'Citadel',
    ingredients: [{ paintId: 101, name: 'Khorne Red', code: '22-14', percentage: 1, r: 150, g: 30, b: 30 }],
    resultR: 150, resultG: 30, resultB: 30,
    deltaE: 1.2, method: 'mix', reproducible: true, tips: [],
    ...over,
  };
}

/** Dubla `<canvas>` e `Image` (D-002): jsdom não tem contexto 2d nem carrega
 *  imagem de verdade. Mesma dublagem de t2-plano-viewport.test.ts. */
function stubCanvasAndImage(imgW = 1200, imgH = 800) {
  const ctx = {
    clearRect: () => {}, save: () => {}, restore: () => {}, setTransform: () => {},
    translate: () => {}, scale: () => {}, drawImage: () => {},
    beginPath: () => {}, arc: () => {}, fill: () => {}, stroke: () => {},
    fillText: () => {}, setLineDash: () => {},
    getImageData: () => ({ data: new Uint8ClampedArray([10, 20, 30, 255]) }),
    fillStyle: '', strokeStyle: '', lineWidth: 0, font: '', textAlign: '', textBaseline: '',
  };
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx as never);

  class FakeImage {
    onload: (() => void) | null = null;
    width = imgW;
    height = imgH;
    set src(_v: string) {
      queueMicrotask(() => this.onload?.());
    }
  }
  vi.stubGlobal('Image', FakeImage);

  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, right: 900, bottom: 600, width: 900, height: 600,
    toJSON: () => ({}),
  } as DOMRect);
  for (const prop of ['clientWidth', 'clientHeight'] as const) {
    Object.defineProperty(HTMLElement.prototype, prop, {
      configurable: true,
      get() {
        return prop === 'clientWidth' ? 900 : 600;
      },
    });
  }
}

/** Carrega uma foto na aba ATIVA pelo `<input type="file">` do cabeçalho e
 *  espera o `<canvas>` ficar enquadrado (fora do tamanho padrão 300×150).
 *
 *  Espera primeiro a aba ativa estar SEM foto (nenhum `<canvas>` no DOM):
 *  depois de trocar/criar aba o clique é síncrono, mas o re-render do
 *  Svelte não é — sem essa espera, este helper podia pegar o `<canvas>`
 *  ANTIGO (ainda montado) e devolver na hora, sem nunca carregar a foto
 *  nova na aba certa. */
async function loadPhoto(container: HTMLElement): Promise<HTMLCanvasElement> {
  await waitFor(() => {
    if (container.querySelector('canvas')) throw new Error('ainda há canvas de outra aba montado');
  });
  const input = container.querySelector('input[type="file"]') as HTMLInputElement;
  const file = new File(['x'], 'peca.png', { type: 'image/png' });
  Object.defineProperty(input, 'files', { value: [file], configurable: true });
  input.dispatchEvent(new Event('change', { bubbles: true }));
  return waitFor(() => {
    const c = container.querySelector('canvas');
    if (!c) throw new Error('canvas não montou depois de carregar a foto');
    if (c.width === 300) throw new Error('canvas ainda no tamanho padrão');
    return c;
  }, { timeout: 2000 });
}

function clickCanvas(canvas: HTMLCanvasElement, x: number, y: number) {
  canvas.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: x, clientY: y }));
}

/** A seção "Tintas da peça" — o `<div>` logo depois do rótulo `t('piecePaints')`. */
function shoppingSection(container: HTMLElement): HTMLElement {
  const labels = [...container.querySelectorAll('p.section-label')];
  const label = labels.find(el => el.textContent === t('piecePaints'));
  if (!label) throw new Error('seção "Tintas da peça" não encontrada');
  return label.nextElementSibling as HTMLElement;
}

beforeEach(() => {
  window.localStorage.clear();
  engineQueue.length = 0;
  toasts.length = 0;
  nav.tab = 'pergunta';
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('rf-09 — tira de abas (T2/PlannerView)', () => {
  it('CA1 — plano novo abre com uma aba "Figura 1" ativa', async () => {
    const { getAllByRole } = render(PlannerView);
    await waitFor(() => {
      const tabs = getAllByRole('tab');
      expect(tabs).toHaveLength(1);
      expect(tabs[0]).toHaveAttribute('aria-selected', 'true');
      expect(tabs[0].textContent).toContain('Figura 1');
    });
  });

  it('CA2 — criar aba entra ao lado, vazia, e vira a ativa', async () => {
    const { getAllByRole, getByLabelText } = render(PlannerView);
    getByLabelText(t('addTabBtn')).click();
    await waitFor(() => {
      const tabs = getAllByRole('tab');
      expect(tabs).toHaveLength(2);
      expect(tabs[0]).toHaveAttribute('aria-selected', 'false');
      expect(tabs[1]).toHaveAttribute('aria-selected', 'true');
      expect(tabs[1].textContent).toContain('Figura 2');
      expect(tabs[1].textContent).toContain('0/0');
    });
  });

  it('CA3 — alternar A→B→A preserva as regiões de cada aba e a seleção não vaza', async () => {
    stubCanvasAndImage();
    const { container, getAllByRole, getByLabelText } = render(PlannerView);

    const canvasA = await loadPhoto(container);
    clickCanvas(canvasA, 150, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(1));
    clickCanvas(canvasA, 350, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(2));
    clickCanvas(canvasA, 150, 350);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(3));

    // Seleciona explicitamente a 2ª região de A.
    (container.querySelectorAll('.t2-region-card')[1] as HTMLElement).click();
    await waitFor(() => {
      expect(container.querySelectorAll('.t2-region-card')[1].getAttribute('style'))
        .toContain('var(--color-accent-700)');
    });

    // Aba B: uma foto nova, uma região só.
    getByLabelText(t('addTabBtn')).click();
    const canvasB = await loadPhoto(container);
    clickCanvas(canvasB, 150, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(1));

    // Volta pra A: as 3 regiões originais, e a seleção da 2ª voltou — sem vazar de B.
    getAllByRole('tab')[0].click();
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(3));
    const cardsA = container.querySelectorAll('.t2-region-card');
    expect(cardsA[1].getAttribute('style')).toContain('var(--color-accent-700)');
    expect(cardsA[0].getAttribute('style')).not.toContain('var(--color-accent-700)');
    expect(cardsA[2].getAttribute('style')).not.toContain('var(--color-accent-700)');

    // B continua com a região dela, intacta.
    getAllByRole('tab')[1].click();
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(1));
  });

  it('CA4 — renomear a aba persiste no estado (sobrevive à troca de aba)', async () => {
    const { container, getAllByLabelText, getAllByRole, getByLabelText } = render(PlannerView);

    getAllByLabelText(t('renameTabLabel'))[0].click();
    const input = await waitFor(() => {
      const el = container.querySelector('input[maxlength="80"]') as HTMLInputElement | null;
      if (!el) throw new Error('campo de renomear não apareceu');
      return el;
    });
    input.value = 'Costas';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('blur', { bubbles: true }));

    await waitFor(() => expect(getAllByRole('tab')[0].textContent).toContain('Costas'));

    // Cria outra aba, troca pra ela e volta — "Costas" não se perde.
    getByLabelText(t('addTabBtn')).click();
    await waitFor(() => expect(getAllByRole('tab')).toHaveLength(2));
    getAllByRole('tab')[0].click();
    await waitFor(() => {
      expect(getAllByRole('tab')[0]).toHaveAttribute('aria-selected', 'true');
      expect(getAllByRole('tab')[0].textContent).toContain('Costas');
    });
  });

  it('CA7 — excluir a única aba é negado com "O plano precisa de pelo menos uma figura"', async () => {
    const { getAllByLabelText, getAllByRole } = render(PlannerView);
    getAllByLabelText(t('deleteTabBtn'))[0].click();
    await waitFor(() => {
      expect(toasts.some(x => x.message === t('errTabDeleteLast') && x.kind === 'error')).toBe(true);
    });
    expect(getAllByRole('tab')).toHaveLength(1);
  });

  it('CA10/CA11/CA12 — tinta repetida some, misturas diferentes ficam separadas, região sem tinta some', async () => {
    stubCanvasAndImage();
    const { container, getAllByRole, getByLabelText } = render(PlannerView);

    // Aba 1 (Figura 1): Khorne Red (Citadel) numa região, e uma região sem tinta parecida.
    const canvas1 = await loadPhoto(container);
    engineQueue.push(recipe());
    clickCanvas(canvas1, 150, 150);
    await waitFor(() => expect(container.textContent).toContain('Khorne Red'));
    engineQueue.push(null);
    clickCanvas(canvas1, 400, 150);
    await waitFor(() => expect(container.textContent).not.toContain(t('calculating')));

    // Aba 2 (Figura 2): a MESMA Khorne Red de novo (CA10), e duas misturas
    // diferentes da mesma marca, ambas com paintId nulo (CA11).
    getByLabelText(t('addTabBtn')).click();
    const canvas2 = await loadPhoto(container);
    engineQueue.push(recipe());
    clickCanvas(canvas2, 150, 150);
    await waitFor(() => expect(container.textContent).not.toContain(t('calculating')));
    engineQueue.push(recipe({
      targetManufacturer: 'Vallejo',
      ingredients: [
        { paintId: 1, name: 'A', code: '', percentage: 0.5, r: 1, g: 1, b: 1 },
        { paintId: 2, name: 'B', code: '', percentage: 0.5, r: 2, g: 2, b: 2 },
      ],
    }));
    clickCanvas(canvas2, 400, 150);
    await waitFor(() => expect(container.textContent).toContain('A + B'));
    engineQueue.push(recipe({
      targetManufacturer: 'Vallejo',
      ingredients: [
        { paintId: 3, name: 'C', code: '', percentage: 0.5, r: 3, g: 3, b: 3 },
        { paintId: 4, name: 'D', code: '', percentage: 0.5, r: 4, g: 4, b: 4 },
      ],
    }));
    clickCanvas(canvas2, 150, 400);
    await waitFor(() => expect(container.textContent).toContain('C + D'));

    // A lista deduplicada: Khorne Red (uma vez, citando as duas abas), A + B, C + D — 3 itens.
    const section = await waitFor(() => {
      const s = shoppingSection(container);
      if (s.children.length < 3) throw new Error(`só ${s.children.length} itens até agora`);
      return s;
    });
    expect(section.children).toHaveLength(3);
    const texto = section.textContent ?? '';
    expect(texto).toContain('Khorne Red');
    expect(texto).toContain('A + B');
    expect(texto).toContain('C + D');
    // CA10: a Khorne Red cita as duas abas.
    expect(texto).toContain(t('usedInTabsLabel', { tabs: 'Figura 1, Figura 2' }));

    // Confirma que as duas abas ainda existem, como esperado pelo cenário.
    expect(getAllByRole('tab')).toHaveLength(2);
  });

  it('CA13 — o progresso do cabeçalho soma todas as abas; cada aba mostra o seu', async () => {
    stubCanvasAndImage();
    const { container, getAllByRole, getByLabelText } = render(PlannerView);

    const canvas1 = await loadPhoto(container);
    clickCanvas(canvas1, 150, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(1));
    (container.querySelector('.t2-done-btn') as HTMLElement).click();
    await waitFor(() => {
      expect(container.querySelector('.t2-done-btn')).toHaveAttribute('aria-pressed', 'true');
    });
    clickCanvas(canvas1, 400, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(2));

    getByLabelText(t('addTabBtn')).click();
    const canvas2 = await loadPhoto(container);
    clickCanvas(canvas2, 150, 150);
    await waitFor(() => expect(container.querySelectorAll('.t2-region-card')).toHaveLength(1));

    await waitFor(() => {
      expect(container.textContent).toContain(t('progress', { a: 1, b: 3 }));
      const tabs = getAllByRole('tab');
      expect(tabs[0].textContent).toContain('1/2');
      expect(tabs[1].textContent).toContain('0/1');
    });
  });

  it('CA14 — o checklist "tenho" continua marcado depois de trocar de aba', async () => {
    stubCanvasAndImage();
    const { container, getByLabelText } = render(PlannerView);

    const canvas1 = await loadPhoto(container);
    engineQueue.push(recipe());
    clickCanvas(canvas1, 150, 150);
    await waitFor(() => expect(shoppingSection(container).children.length).toBeGreaterThan(0));

    const toggleBtn = shoppingSection(container).querySelector('button[aria-pressed]') as HTMLElement;
    expect(toggleBtn).toHaveAttribute('aria-pressed', 'false');
    toggleBtn.click();
    await waitFor(() => {
      expect(shoppingSection(container).querySelector('button[aria-pressed]'))
        .toHaveAttribute('aria-pressed', 'true');
    });

    // Troca de aba (cria uma nova, vazia, e vira a ativa) — a marcação é do plano, não da aba.
    getByLabelText(t('addTabBtn')).click();
    await waitFor(() => {
      expect(shoppingSection(container).querySelector('button[aria-pressed]'))
        .toHaveAttribute('aria-pressed', 'true');
    });
  });

  it('CA21 — trocar de aba não grava rascunho', async () => {
    nav.tab = 'plano';
    const { getAllByRole, getByLabelText } = render(PlannerView);
    // Deixa a hidratação (sem rascunho/plano ativo salvo) terminar antes de mexer.
    await new Promise(resolve => setTimeout(resolve, 0));

    vi.useFakeTimers();
    const setItemSpy = vi.spyOn(window.localStorage, 'setItem');

    // Cria uma 2ª aba — ISSO é alteração de conteúdo, grava depois do debounce.
    getByLabelText(t('addTabBtn')).click();
    await vi.advanceTimersByTimeAsync(1600);
    expect(window.localStorage.getItem('mescla:plano-rascunho')).not.toBeNull();

    setItemSpy.mockClear();

    // Só troca de aba — nenhuma gravação nova.
    getAllByRole('tab')[0].click();
    await vi.advanceTimersByTimeAsync(2000);
    expect(setItemSpy).not.toHaveBeenCalled();
  });

  it('CA22 — criar e excluir aba gravam rascunho depois da pausa de 1500 ms', async () => {
    nav.tab = 'plano';
    const { getAllByLabelText, getByLabelText } = render(PlannerView);
    await new Promise(resolve => setTimeout(resolve, 0));

    vi.useFakeTimers();
    vi.spyOn(window, 'confirm').mockReturnValue(true);

    // Criar aba grava depois do debounce.
    getByLabelText(t('addTabBtn')).click();
    await vi.advanceTimersByTimeAsync(1499);
    expect(window.localStorage.getItem('mescla:plano-rascunho')).toBeNull();
    await vi.advanceTimersByTimeAsync(1);
    const apósCriar = window.localStorage.getItem('mescla:plano-rascunho');
    expect(apósCriar).not.toBeNull();
    expect(JSON.parse(apósCriar as string).tabs).toHaveLength(2);

    // Excluir a aba criada também grava (2 abas → pode excluir uma).
    getAllByLabelText(t('deleteTabBtn'))[1].click();
    await vi.advanceTimersByTimeAsync(1600);
    const apósExcluir = window.localStorage.getItem('mescla:plano-rascunho');
    expect(apósExcluir).not.toBeNull();
    expect(JSON.parse(apósExcluir as string).tabs).toHaveLength(1);
  });

  it('CA28 — role="tablist"/"tab" por aba, aria-selected na ativa, tabpanel com aria-labelledby válido', async () => {
    const { container, getAllByRole, getByLabelText } = render(PlannerView);
    getByLabelText(t('addTabBtn')).click();

    await waitFor(() => {
      const tablist = container.querySelector('[role="tablist"]');
      expect(tablist).toHaveAttribute('aria-label', t('tabsListLabel'));

      const tabs = getAllByRole('tab');
      expect(tabs).toHaveLength(2);
      expect(tabs[0]).toHaveAttribute('aria-selected', 'false');
      expect(tabs[1]).toHaveAttribute('aria-selected', 'true');

      const panel = container.querySelector('[role="tabpanel"]');
      expect(panel).not.toBeNull();
      const labelledBy = panel!.getAttribute('aria-labelledby');
      expect(labelledBy).toBe(tabs[1].id);
      expect(document.getElementById(labelledBy as string)).toBe(tabs[1]);
    });
  });

  it('CA29 — seta direita move o foco para a próxima aba', async () => {
    const { getAllByRole, getByLabelText } = render(PlannerView);
    getByLabelText(t('addTabBtn')).click();
    await waitFor(() => expect(getAllByRole('tab')).toHaveLength(2));

    const tab0 = getAllByRole('tab')[0] as HTMLElement;
    tab0.focus();
    expect(document.activeElement).toBe(tab0);

    tab0.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));

    await waitFor(() => {
      const tabs = getAllByRole('tab');
      expect(document.activeElement).toBe(tabs[1]);
      expect(tabs[1]).toHaveAttribute('aria-selected', 'true');
      expect(tabs[0]).toHaveAttribute('aria-selected', 'false');
    });
  });
});

describe('CA30 — textos novos da tira de abas existem nos 4 idiomas', () => {
  const chaves = [
    'tabsListLabel', 'renameTabLabel', 'addTabBtn', 'moveTabLeft', 'moveTabRight',
    'deleteTabBtn', 'deleteTabConfirm', 'errTabsMax', 'errTabDeleteLast',
    'tabProgressShort', 'usedInTabsLabel',
  ] as const;

  it('cada chave existe (chave própria, não cai no fallback pt) nos 4 idiomas', () => {
    for (const idioma of ['pt', 'en', 'es', 'fr'] as const) {
      for (const chave of chaves) {
        expect(chave in dict[idioma], `dict.${idioma} sem chave própria "${chave}"`).toBe(true);
        const valor = (dict[idioma] as Record<string, string>)[chave];
        expect(typeof valor, `dict.${idioma}.${chave} não é texto`).toBe('string');
        expect(valor.length, `dict.${idioma}.${chave} vazio`).toBeGreaterThan(0);
      }
    }
  });
});
