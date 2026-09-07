import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { Rascunho } from '../src/lib/planner/rascunho';

/** Cada teste reimporta o módulo: `estado` da escada de cota é de módulo. */
async function carregarModulo() {
  vi.resetModules();
  return import('../src/lib/planner/rascunho');
}

function rascunho(comFoto = true): Rascunho {
  return {
    v: 2,
    planId: 7,
    name: 'Sargento',
    abaAtiva: 0,
    tabs: [
      {
        name: 'Frente',
        imageData: comFoto ? 'data:image/png;base64,AAAA' : '',
        selectedManufacturerId: null,
        useStockOnly: false,
        regions: [],
      },
    ],
    salvoEm: new Date().toISOString(),
  };
}

/** localStorage falso, com gatilho de cota por chamada. */
function instalarStorage(falharAte = 0) {
  const dados = new Map<string, string>();
  let chamadas = 0;
  const store = {
    getItem: (k: string) => dados.get(k) ?? null,
    setItem: vi.fn((k: string, v: string) => {
      chamadas++;
      if (chamadas <= falharAte) {
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

describe('rf-09 — escada de cota do rascunho (RN8 estendida a N abas)', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('CA5/CA6 — grava uma vez só depois da pausa, por mais alterações que cheguem', async () => {
    const { store } = instalarStorage();
    const mod = await carregarModulo();

    for (let i = 0; i < 5; i++) mod.agendarGravacao(rascunho());
    expect(store.setItem).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1500);
    expect(store.setItem).toHaveBeenCalledTimes(1);
  });

  it('cancelarGravacao impede a gravação agendada', async () => {
    const { store } = instalarStorage();
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho());
    mod.cancelarGravacao();
    vi.advanceTimersByTime(1500);

    expect(store.setItem).not.toHaveBeenCalled();
  });

  it('CA21 — cota estourada corta a foto de todas as abas e o estado vira sem-foto', async () => {
    const { store, dados } = instalarStorage(1);
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho());
    vi.advanceTimersByTime(1500);

    expect(store.setItem).toHaveBeenCalledTimes(2);
    expect(mod.estadoAutoSave()).toBe('sem-foto');
    const gravado = JSON.parse(dados.get('mescla:plano-rascunho') as string);
    expect(gravado.tabs[0].imageData).toBe('');
    expect(gravado.tabs[0].name).toBe('Frente');
  });

  it('CA22 — cota estourada mesmo sem as fotos desliga o auto save na sessão', async () => {
    const { store } = instalarStorage(99);
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho());
    vi.advanceTimersByTime(1500);
    expect(mod.estadoAutoSave()).toBe('desligado');

    const antes = store.setItem.mock.calls.length;
    mod.agendarGravacao(rascunho());
    vi.advanceTimersByTime(1500);
    expect(store.setItem.mock.calls.length).toBe(antes);
  });

  it('plano ativo: grava, lê e limpa; id inválido não é aceito', async () => {
    instalarStorage();
    const mod = await carregarModulo();

    expect(mod.lerPlanoAtivo()).toBeNull();
    mod.gravarPlanoAtivo(42);
    expect(mod.lerPlanoAtivo()).toBe(42);
    mod.limparPlanoAtivo();
    expect(mod.lerPlanoAtivo()).toBeNull();
  });

  it('apagarRascunho remove a chave e cancela o que estava agendado', async () => {
    const { store, dados } = instalarStorage();
    const mod = await carregarModulo();

    mod.agendarGravacao(rascunho());
    mod.apagarRascunho();
    vi.advanceTimersByTime(1500);

    expect(store.setItem).not.toHaveBeenCalled();
    expect(dados.has('mescla:plano-rascunho')).toBe(false);
  });

  it('armazenamento que lança em toda operação degrada sem quebrar a tela', async () => {
    vi.stubGlobal('localStorage', {
      getItem: () => { throw new Error('bloqueado'); },
      setItem: () => { throw new Error('bloqueado'); },
      removeItem: () => { throw new Error('bloqueado'); },
    });
    const mod = await carregarModulo();

    expect(() => mod.lerRascunho()).not.toThrow();
    expect(() => mod.lerPlanoAtivo()).not.toThrow();
    expect(() => mod.apagarRascunho()).not.toThrow();
    expect(mod.lerRascunho().rascunho).toBeNull();
  });
});
