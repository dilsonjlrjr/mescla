// D-002 (Epic D) — teste de reprodução dos dois defeitos de T2 (Plano da peça):
//
//   (a) a foto carregada aparece minúscula: o ajuste inicial é calculado no espaço de CSS px do
//       próprio <canvas>, cujo tamanho intrínseco só é definido depois, pela resolução da imagem.
//       A escala sai circular. Reproduzido aqui na matemática do viewport, fora do DOM.
//   (b) clicar na foto deixa a região presa em "calculando": `addRegion` passa a referência CRUA
//       da região para `computeRegion`, enquanto o array reativo guarda o PROXY de `$state` —
//       escrever no objeto cru não dispara sinal, e a tela nunca atualiza. Reproduzido no DOM.
//
// TDD unit-only: solver mockado, <canvas>, Image e geometria dublados. Nenhum servidor.

import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fitView, toImage, toScreen, zoomAround, ZOOM_MAX, ZOOM_MIN } from '../src/lib/planner/viewport';

const MFRS = [{ id: 1, name: 'Vallejo', paintCount: 2 }];

vi.mock('../src/lib/services/catalog', () => ({
  loadCatalog: () => Promise.resolve(),
  allPaints: () => [],
  allManufacturers: () => MFRS,
  paintById: () => undefined,
  searchPaints: () => [],
  hexOf: () => '#000000',
}));

const RECIPE = {
  sourcePaintId: 0,
  sourceName: '',
  sourceManufacturer: '',
  sourceR: 10, sourceG: 20, sourceB: 30,
  targetManufacturer: 'Vallejo',
  ingredients: [{ paintId: 1, name: 'Sombra', code: '70.951', percentage: 1, r: 10, g: 20, b: 30 }],
  resultR: 11, resultG: 21, resultB: 31,
  deltaE: 1.4,
  method: 'mix',
  reproducible: true,
  tips: [],
};

vi.mock('../src/lib/services/engine', () => ({
  engineReady: () => Promise.resolve(),
  suggestRecipeForColor: () => Promise.resolve(RECIPE),
  ehUniversoVazio: (v: unknown) => (v as { universoVazio?: boolean })?.universoVazio === true,
  melhorDeltaE: () => Promise.resolve(null),
}));

// ── (a) matemática do viewport ───────────────────────────────────────────────

describe('D-002a — ajuste inicial da foto', () => {
  it('preenche uma das dimensões da caixa (a foto não pode sair minúscula)', () => {
    const box = { w: 900, h: 600 };
    for (const img of [
      { w: 3000, h: 2000 },   // foto de celular, muito maior que a caixa
      { w: 4000, h: 1200 },   // panorâmica
      { w: 800, h: 3000 },    // retrato alto
    ]) {
      const v = fitView(img.w, img.h, box.w, box.h);
      const usoX = (img.w * v.zoom) / box.w;
      const usoY = (img.h * v.zoom) / box.h;
      // cabe inteira…
      expect(usoX).toBeLessThanOrEqual(1.0001);
      expect(usoY).toBeLessThanOrEqual(1.0001);
      // …e encosta em pelo menos uma borda: é isso que o defeito quebrava.
      expect(Math.max(usoX, usoY)).toBeCloseTo(1, 3);
    }
  });

  it('centraliza a foto na caixa', () => {
    const v = fitView(3000, 2000, 900, 600);
    expect(v.panX).toBeCloseTo((900 - 3000 * v.zoom) / 2, 6);
    expect(v.panY).toBeCloseTo((600 - 2000 * v.zoom) / 2, 6);
  });

  it('não amplia uma foto menor que a caixa além do tamanho real', () => {
    const v = fitView(200, 100, 900, 600);
    expect(v.zoom).toBe(1);
  });

  it('converte tela ↔ imagem de ida e volta', () => {
    const v = fitView(3000, 2000, 900, 600);
    const p = toScreen(v, 1234, 567);
    const back = toImage(v, p.x, p.y);
    expect(back.x).toBeCloseTo(1234, 6);
    expect(back.y).toBeCloseTo(567, 6);
  });
});

describe('rf-06 — zoom com âncora', () => {
  it('mantém sob o cursor o mesmo ponto da imagem', () => {
    const v0 = fitView(3000, 2000, 900, 600);
    const ancora = { x: 300, y: 220 };
    const antes = toImage(v0, ancora.x, ancora.y);
    const v1 = zoomAround(v0, 2, ancora);
    const depois = toImage(v1, ancora.x, ancora.y);
    expect(depois.x).toBeCloseTo(antes.x, 6);
    expect(depois.y).toBeCloseTo(antes.y, 6);
    expect(v1.zoom).toBeCloseTo(v0.zoom * 2, 6);
  });

  it('prende o zoom nos limites', () => {
    const v = fitView(1000, 1000, 900, 600);
    expect(zoomAround(v, 1000, { x: 0, y: 0 }).zoom).toBe(ZOOM_MAX);
    expect(zoomAround(v, 0.0001, { x: 0, y: 0 }).zoom).toBe(ZOOM_MIN);
  });
});

// ── (b) região presa em "calculando" ─────────────────────────────────────────

/** jsdom não tem contexto 2d nem carrega imagem: os dois são dublados aqui. */
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

  // A caixa da foto: sem isso toda a geometria vira 0 em jsdom.
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

describe('D-002b — clique na foto', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it('dimensiona o canvas pela caixa, não pela resolução da foto', async () => {
    stubCanvasAndImage(1200, 800);
    const { default: PlannerView } = await import('../src/lib/views/PlannerView.svelte');
    const { container } = render(PlannerView);

    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['x'], 'peca.png', { type: 'image/png' });
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    input.dispatchEvent(new Event('change', { bubbles: true }));

    const canvas = await waitFor(() => {
      const c = container.querySelector('canvas');
      if (!c) throw new Error('canvas não montou depois de carregar a foto');
      return c;
    });
    await waitFor(() => {
      if (canvas.width === 300) throw new Error('canvas ainda no tamanho padrao: ' + canvas.width);
    }, { timeout: 2000 });
    // O bitmap segue a caixa (900×600 dublada); seguir a foto é o defeito.
    expect(canvas.width).not.toBe(1200);
    expect(canvas.width / (window.devicePixelRatio || 1)).toBeCloseTo(900, 0);
  });

  it('sai de "calculando" quando o solver responde', async () => {
    stubCanvasAndImage();
    const { default: PlannerView } = await import('../src/lib/views/PlannerView.svelte');
    const { container, findAllByText, queryByText } = render(PlannerView);

    // Carrega a foto pelo input de arquivo da tela.
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['x'], 'peca.png', { type: 'image/png' });
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    input.dispatchEvent(new Event('change', { bubbles: true }));

    const canvas = await waitFor(() => {
      const c = container.querySelector('canvas');
      if (!c) throw new Error('canvas não montou depois de carregar a foto');
      return c;
    });

    // Espera o enquadramento inicial: antes dele a caixa ainda não foi medida.
    await waitFor(() => {
      if (canvas.width === 300) throw new Error('canvas ainda no tamanho padrao');
    }, { timeout: 2000 });

    // Toque no meio da foto: cria a região e dispara o cálculo.
    canvas.dispatchEvent(
      new MouseEvent('click', { bubbles: true, clientX: 450, clientY: 300 }),
    );

    // O solver mockado responde na hora — a região TEM de sair de "calculando".
    await findAllByText(/ΔE/, undefined, { timeout: 2000 });
    await waitFor(() => expect(queryByText('calculando…')).toBeNull());
  });
});
