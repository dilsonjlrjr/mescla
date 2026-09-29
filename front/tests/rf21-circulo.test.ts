// rf-21 — círculo cromático: regras do modelo (color/circulo.ts) e fumaça da
// tela T5. Unit-only com mocks, como o resto da suíte.
import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import {
  WHEEL,
  chromaOf,
  darkenPaths,
  harmonyOf,
  hslHueToWheel,
  isWarm,
  lightenPaths,
  monoBands,
  nearestIndex,
  rgbToHsl,
  schemeById,
  valueOf,
  wheelDegOf,
  wheelToHslHue,
  desaturatePaths,
} from '../src/lib/color/circulo';

vi.mock('../src/lib/services/catalog', () => ({
  allPaints: () => [
    { id: 1, manufacturerId: 1, manufacturer: 'Vallejo', name: 'Ocre', code: '70.914', line: '', r: 190, g: 150, b: 70, paintTypeId: 0 },
    { id: 2, manufacturerId: 1, manufacturer: 'Vallejo', name: 'Azul', code: '70.930', line: '', r: 30, g: 60, b: 160, paintTypeId: 0 },
  ],
}));

vi.mock('../src/lib/services/stock.svelte', () => ({
  stock: { paints: [], estado: 'pronto' },
}));

// jsdom não implementa Web Animations; as transições do Svelte usam.
if (!Element.prototype.animate) {
  Element.prototype.animate = function () {
    const anim = { onfinish: null as null | (() => void), cancel() {}, finish() {}, currentTime: 0, playState: 'finished' };
    queueMicrotask(() => anim.onfinish?.());
    return anim as unknown as Animation;
  };
}

const idx = (id: string) => WHEEL.findIndex(h => h.id === id) * 30;

describe('rf-21 — modelo do círculo', () => {
  it('é a roda do pintor: complementares de pigmento, não de tela', () => {
    const comp = (id: string) => WHEEL[nearestIndex(idx(id) + 180)].id;
    expect(comp('amarelo')).toBe('azulVioletado');
    expect(comp('vermelho')).toBe('azulCiano');
    expect(comp('laranja')).toBe('azul');
    expect(comp('magenta')).toBe('verde');
  });

  it('o mapa matiz ↔ roda cai exatamente nos 12 matizes do anel e volta', () => {
    for (const h of WHEEL) {
      const hue = rgbToHsl(h.rgb).h;
      expect(hslHueToWheel(hue)).toBeCloseTo(h.index * 30, 5);
      expect(wheelToHslHue(h.index * 30)).toBeCloseTo(hue, 5);
    }
  });

  it('uma cor qualquer cai entre os vizinhos certos; neutro não tem matiz', () => {
    const ocre = wheelDegOf({ r: 200, g: 150, b: 40 })!;
    expect(ocre).toBeGreaterThan(240);
    expect(ocre).toBeLessThan(270);
    expect(wheelDegOf({ r: 128, g: 128, b: 128 })).toBeNull();
  });

  it('quentes de Amarelo a Violeta; frias de Azul-violetado a Verde-lima', () => {
    expect(isWarm(idx('amarelo'))).toBe(true);
    expect(isWarm(idx('violeta'))).toBe(true);
    expect(isWarm(idx('azulVioletado'))).toBe(false);
    expect(isWarm(idx('verdeLima'))).toBe(false);
  });

  it('análogas são as vizinhas a 30°; tríade a 120°', () => {
    const base = WHEEL[8];
    const an = harmonyOf(schemeById('analoga'), 240, base.rgb, true).map(s => s.hue.id);
    expect(an).toEqual(['verdeLima', 'amarelo', 'laranja']);
    const tri = harmonyOf(schemeById('triade'), 240, base.rgb, true).map(s => s.hue.id);
    expect(tri).toEqual(['amarelo', 'magenta', 'azulCiano']);
  });

  it('a harmonia respeita a saturação e o valor de uma cor apagada', () => {
    const musgo = { r: 110, g: 120, b: 80 };
    const deg = wheelDegOf(musgo)!;
    const [, comp] = harmonyOf(schemeById('complementar'), deg, musgo, false);
    const a = rgbToHsl(musgo);
    const b = rgbToHsl(comp.rgb);
    expect(b.s).toBeCloseTo(a.s, 1);
    expect(b.l).toBeCloseTo(a.l, 1);
  });

  it('escala monocromática: claro > suave > profundo em valor', () => {
    const bands = monoBands(WHEEL[10].rgb).map(b => valueOf(b.rgb));
    expect(bands[0]).toBeGreaterThan(bands[1]);
    expect(bands[3]).toBeGreaterThan(bands[4]);
    expect(bands[1]).toBeGreaterThan(bands[4]);
  });

  it('escurecer: todos os caminhos descem o valor; o do matiz guarda mais croma que o preto', () => {
    const base = WHEEL[10].rgb;
    const [preto, comp, matiz] = darkenPaths(base, 300);
    for (const p of [preto, comp, matiz]) expect(valueOf(p.steps[4])).toBeLessThan(valueOf(base) - 15);
    expect(chromaOf(matiz.steps[3])).toBeGreaterThan(chromaOf(preto.steps[3]));
    // o complementar neutraliza: croma cai mais que no preto
    expect(chromaOf(comp.steps[4])).toBeLessThan(chromaOf(preto.steps[4]));
  });

  it('amarelo não esverdeia no caminho do matiz (vai pelo ocre)', () => {
    const [, , matiz] = darkenPaths(WHEEL[8].rgb, 240);
    const deg = wheelDegOf(matiz.steps[3])!;
    expect(deg).toBeGreaterThan(240);
  });

  it('clarear sobe o valor; cinza do mesmo valor mantém o valor', () => {
    const base = WHEEL[3].rgb;
    for (const p of lightenPaths(base, 90)) expect(valueOf(p.steps[4])).toBeGreaterThan(valueOf(base) + 15);
    const [cinza] = desaturatePaths(base, 90);
    expect(Math.abs(valueOf(cinza.steps[4]) - valueOf(base))).toBeLessThan(2);
    expect(chromaOf(cinza.steps[4])).toBeLessThan(2);
  });
});

describe('rf-21 — tela T5', () => {
  it('renderiza o disco, troca de harmonia e aceita um hex', async () => {
    const { default: CirculoView } = await import('../src/lib/views/CirculoView.svelte');
    const { container, getByText, getAllByRole, getByPlaceholderText } = render(CirculoView);
    expect(container.querySelector('svg.wheel')).not.toBeNull();
    expect(getAllByRole('radio').length).toBe(8);

    await fireEvent.click(getByText('Tríade', { selector: 'button' }));
    expect(container.querySelectorAll('.t5-sw').length).toBe(3);

    await fireEvent.click(getByText('Clarear'));
    await fireEvent.click(getByText('Valor', { selector: 'button' }));

    const hex = getByPlaceholderText('Cole um hex — #A83B2F') as HTMLInputElement;
    await fireEvent.input(hex, { target: { value: '#6E7850' } });
    await fireEvent.keyDown(hex, { key: 'Enter' });
    expect(container.textContent).toContain('Sua cor');
  });
});
