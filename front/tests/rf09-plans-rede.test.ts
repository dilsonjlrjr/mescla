import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { salvarPlano, carregarPlano, baixarRelatorio, PlanoError } from '../src/lib/services/plans';
import type { PlanoDTO } from '../src/lib/services/plans';

/** Plano mínimo válido com uma aba e N regiões. */
function plano(regioes = 1): PlanoDTO {
  return {
    name: 'Sargento',
    tabs: [
      {
        name: 'Frente',
        imageData: '',
        useStockOnly: 0,
        regions: Array.from({ length: regioes }, () => ({
          x: 1, y: 1, r: 10, g: 20, b: 30, hex: '#0A141E',
          regionName: 'Manto', note: '', paintBrand: '', paintName: '', paintCode: '',
          deltaE: 0, painted: 0 as 0 | 1,
        })),
      },
    ],
  };
}

function resposta(body: unknown, init: { status?: number; headers?: Record<string, string> } = {}) {
  const { status = 200, headers = {} } = init;
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: (k: string) => headers[k] ?? null },
    json: async () => body,
    blob: async () => new Blob([JSON.stringify(body)]),
  } as unknown as Response;
}

describe('rf-09/rf-08 — plans.ts sobre a rede', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('salvarPlano devolve o plano salvo quando o servidor responde 200', async () => {
    const salvo = { ...plano(1), id: 12 };
    vi.mocked(fetch).mockResolvedValue(resposta(salvo));

    const { plano: devolvido, suspeitaD003 } = await salvarPlano(plano(1));

    expect(devolvido.id).toBe(12);
    expect(suspeitaD003).toBe(false);
  });

  it('CA16 — salvarPlano acusa suspeita de D-003 quando voltam menos regiões do que foram enviadas', async () => {
    vi.mocked(fetch).mockResolvedValue(resposta({ ...plano(0), id: 12 }));

    const { suspeitaD003 } = await salvarPlano(plano(3));

    expect(suspeitaD003).toBe(true);
  });

  it('salvarPlano converte falha de rede em PlanoError, sem vazar o texto do servidor', async () => {
    vi.mocked(fetch).mockRejectedValue(new Error('detalhe interno do sqlite'));

    const erro = await salvarPlano(plano(1)).catch((e) => e);
    expect(erro).toBeInstanceOf(PlanoError);
    expect(erro.code).toBe('falha');
    // RN11: o texto do servidor nunca chega à tela.
    expect(String(erro.message)).not.toContain('sqlite');
  });

  it('carregarPlano recusa id inválido antes de tocar a rede (CAN1)', async () => {
    for (const id of [0, -7, 1.5, NaN]) {
      await expect(carregarPlano(id)).rejects.toBeInstanceOf(PlanoError);
    }
    expect(fetch).not.toHaveBeenCalled();
  });

  it('carregarPlano traduz 404 em nao-encontrado e 500 em falha', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(resposta({ error: 'plano não encontrado' }, { status: 404 }));
    await expect(carregarPlano(9)).rejects.toMatchObject({ code: 'nao-encontrado' });

    vi.mocked(fetch).mockResolvedValueOnce(resposta({ error: 'boom' }, { status: 500 }));
    await expect(carregarPlano(9)).rejects.toMatchObject({ code: 'falha' });
  });

  it('carregarPlano devolve o plano com as abas quando responde 200', async () => {
    vi.mocked(fetch).mockResolvedValue(resposta({ ...plano(2), id: 3 }));

    const p = await carregarPlano(3);

    expect(p.id).toBe(3);
    expect(p.tabs[0].regions).toHaveLength(2);
  });

  it('baixarRelatorio lê o nome do arquivo de filename* (RFC 5987)', async () => {
    vi.mocked(fetch).mockResolvedValue(
      resposta('conteudo', {
        headers: { 'Content-Disposition': "attachment; filename=\"x.pdf\"; filename*=UTF-8''mescla-a%C3%A7%C3%A3o-2026-09-07.pdf" },
      })
    );

    const { filename } = await baixarRelatorio(5, 'pdf');

    expect(filename).toBe('mescla-ação-2026-09-07.pdf');
  });

  it('baixarRelatorio cai no filename simples quando não há filename*', async () => {
    vi.mocked(fetch).mockResolvedValue(
      resposta('conteudo', { headers: { 'Content-Disposition': 'attachment; filename="relatorio.png"' } })
    );

    const { filename } = await baixarRelatorio(5, 'png');

    expect(filename).toBe('relatorio.png');
  });

  it('baixarRelatorio recusa id inválido sem tocar a rede', async () => {
    await expect(baixarRelatorio(0, 'pdf')).rejects.toBeInstanceOf(PlanoError);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('baixarRelatorio traduz 404, 400 e 500 em códigos distintos', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(resposta({ error: 'x' }, { status: 404 }));
    await expect(baixarRelatorio(5, 'pdf')).rejects.toMatchObject({ code: 'nao-encontrado' });

    vi.mocked(fetch).mockResolvedValueOnce(resposta({ error: 'formato inválido' }, { status: 400 }));
    await expect(baixarRelatorio(5, 'pdf')).rejects.toMatchObject({ code: 'invalido' });

    vi.mocked(fetch).mockResolvedValueOnce(resposta({ error: 'x' }, { status: 500 }));
    await expect(baixarRelatorio(5, 'pdf')).rejects.toMatchObject({ code: 'falha' });
  });
});
