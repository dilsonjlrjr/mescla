// D-005 — a foto da aba não sobrevivia ao auto save: entrava no estado sem
// passar pela guarda de 2 MB (só o Salvar a aplicava), a RN8 a cortava na
// primeira gravação e o degrau 'sem-foto' ficava colado pelo resto da sessão.
// Estes testes são a regressão dos dois lados do fix.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { validarImagemDaAba } from '../src/lib/services/plans';
import type { Rascunho } from '../src/lib/planner/rascunho';

const PNG = 'data:image/png;base64,';

async function carregarModulo() {
  vi.resetModules();
  return import('../src/lib/planner/rascunho');
}

function rascunho(imageData: string): Rascunho {
  return {
    v: 2,
    planId: null,
    name: 'Spawn',
    selectedManufacturerId: null,
    useStockOnly: false,
    abaAtiva: 0,
    tabs: [{ id: null, name: 'Frente', imageData, regions: [] }],
    salvoEm: new Date().toISOString(),
  };
}

/** localStorage falso que rejeita qualquer valor acima de `tetoBytes` — o
 *  mesmo formato de falha que o navegador dá quando a cota estoura. */
function instalarStorage(tetoBytes = Infinity) {
  const dados = new Map<string, string>();
  const store = {
    getItem: (k: string) => dados.get(k) ?? null,
    setItem: vi.fn((k: string, v: string) => {
      if (v.length > tetoBytes) {
        const e = new Error('QuotaExceededError');
        e.name = 'QuotaExceededError';
        throw e;
      }
      dados.set(k, v);
    }),
    removeItem: (k: string) => void dados.delete(k),
    clear: () => dados.clear(),
    key: () => null,
    length: 0,
  };
  vi.stubGlobal('localStorage', store);
  return { store, dados };
}

describe('D-005 — guarda da foto (CA17) fora do Salvar', () => {
  it('foto dentro do teto passa', () => {
    expect(validarImagemDaAba(`${PNG}${'A'.repeat(1024)}`)).toBeNull();
  });

  it('aba sem foto passa', () => {
    expect(validarImagemDaAba('')).toBeNull();
  });

  it('foto acima de 2 MB é recusada com a chave da CA17', () => {
    expect(validarImagemDaAba(`${PNG}${'A'.repeat(2 * 1024 * 1024)}`)).toBe('errImageMax');
  });

  it('formato fora da lista branca é recusado', () => {
    expect(validarImagemDaAba('data:image/gif;base64,AAAA')).toBe('errImageType');
  });
});

describe('D-005 — o degrau sem-foto da RN8 é da gravação, não da sessão', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('gravação cheia bem-sucedida devolve o estado a ligado, e a foto seguinte persiste', async () => {
    const grande = `${PNG}${'A'.repeat(5000)}`;
    const pequena = `${PNG}${'A'.repeat(10)}`;
    const { dados } = instalarStorage(2000);
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho(grande));
    vi.advanceTimersByTime(1500);
    expect(mod.estadoAutoSave()).toBe('sem-foto');
    expect(JSON.parse(dados.get('mescla:plano-rascunho') as string).tabs[0].imageData).toBe('');

    mod.agendarGravacao(rascunho(pequena));
    vi.advanceTimersByTime(1500);
    expect(mod.estadoAutoSave()).toBe('ligado');
    expect(JSON.parse(dados.get('mescla:plano-rascunho') as string).tabs[0].imageData).toBe(pequena);
  });

  it('desligado continua colado na sessão (RN8 literal)', async () => {
    instalarStorage(0);
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho(`${PNG}AAAA`));
    vi.advanceTimersByTime(1500);
    expect(mod.estadoAutoSave()).toBe('desligado');

    mod.agendarGravacao(rascunho(''));
    vi.advanceTimersByTime(1500);
    expect(mod.estadoAutoSave()).toBe('desligado');
  });
});
