// Ponte do desktop: a interface do Mescla desktop é a mesma da web (front/),
// copiada sem alteração. A web chama `fetch('/api/...')`; aqui não há
// servidor HTTP, então este módulo troca o `fetch` global por um que atende
// o mesmo contrato de api/httpapi chamando o PaintService pelos bindings
// Wails, dentro do processo e offline. Nenhum pedido sai para a rede.
//
// Espelha de api/httpapi/*.go: caminho, método, forma do corpo, código de
// status e mensagem fixa de erro. Mudou uma rota lá, muda aqui também.

import * as Svc from '../../bindings/paint-match-ai/api/service/paintservice';

const PREFIXO = '/api';

/** Erro que vira resposta HTTP com status e `{ error }` no corpo. */
class ErroRota extends Error {
  status: number;
  extra?: Record<string, unknown>;
  constructor(status: number, message: string, extra?: Record<string, unknown>) {
    super(message);
    this.status = status;
    this.extra = extra;
  }
}

/** Mensagem do erro Go que o runtime do Wails entrega (texto puro ou JSON). */
function mensagemGo(err: unknown): string {
  const bruta = err instanceof Error ? err.message : String(err ?? '');
  try {
    const obj = JSON.parse(bruta) as { message?: unknown };
    if (obj && typeof obj.message === 'string') return obj.message;
  } catch {
    // não é JSON — a mensagem é o próprio texto
  }
  return bruta;
}

// Textos de service.ErrUniversoVazio (api/service/universo.go). O HTTP
// responde 200 com `{ universoVazio, motivo }`; o binding só tem o texto.
const UNIVERSO_VAZIO = [
  /não tem tintas com cor cadastrada$/,
  /^Seu estoque (neste aparelho )?está vazio$/,
  /^Você não tem tintas .+ (no estoque|neste aparelho)$/,
];

function universoVazio(msg: string): { universoVazio: true; motivo: string } | null {
  return UNIVERSO_VAZIO.some(re => re.test(msg)) ? { universoVazio: true, motivo: msg } : null;
}

// writeManufacturerError (api/httpapi/manufacturers.go).
function erroFabricante(err: unknown): never {
  const msg = mensagemGo(err);
  if (msg === 'fabricante não encontrado') throw new ErroRota(404, 'Fabricante não encontrado.');
  if (msg === 'nome de fabricante repetido') throw new ErroRota(409, 'Já existe um fabricante com esse nome.');
  if (msg === 'fabricante em uso por receita ou recurso') {
    throw new ErroRota(409, 'O fabricante está em uso e não pode ser excluído.');
  }
  if (/não pode ser excluído\.$/.test(msg)) throw new ErroRota(409, msg);
  if (/^(Não foi possível|erro|sql)/i.test(msg)) throw new ErroRota(500, 'Não foi possível gravar o fabricante.');
  throw new ErroRota(400, msg);
}

// writePaintTypeError (api/httpapi/paint_types.go).
function erroTipo(err: unknown): never {
  const msg = mensagemGo(err);
  if (msg === 'tipo de tinta não encontrado') throw new ErroRota(404, 'Tipo de tinta não encontrado.');
  if (msg === 'nome de tipo de tinta repetido') throw new ErroRota(409, 'Já existe um tipo de tinta com esse nome.');
  const emUso = /usado por (\d+) tintas do catálogo e (\d+) no estoque/.exec(msg);
  if (emUso) throw new ErroRota(409, msg, { paints: Number(emUso[1]), stock: Number(emUso[2]) });
  if (/^(Não foi possível|erro|sql)/i.test(msg)) throw new ErroRota(500, 'Não foi possível gravar o tipo de tinta.');
  throw new ErroRota(400, msg);
}

// writeUserPaintError (api/httpapi/stock.go).
function erroEstoque(err: unknown): never {
  const msg = mensagemGo(err);
  if (msg === 'tinta de estoque não encontrada') throw new ErroRota(404, msg);
  if (/^(erro|sql|database|begin|commit)/i.test(msg)) throw new ErroRota(500, 'não foi possível gravar a tinta');
  throw new ErroRota(400, msg);
}

/** Chama o binding; erro Go vira ErroRota com o status padrão da rota. */
async function chamar<T>(p: PromiseLike<T>, status: number, fixa?: string): Promise<T> {
  try {
    return await p;
  } catch (err) {
    if (err instanceof ErroRota) throw err;
    throw new ErroRota(status, fixa ?? mensagemGo(err));
  }
}

/** Receita por cor: universo vazio responde 200 com o motivo (RN9). */
async function receitaOuVazio<T>(p: PromiseLike<T>): Promise<T | { universoVazio: true; motivo: string }> {
  try {
    return await p;
  } catch (err) {
    const msg = mensagemGo(err);
    const vazio = universoVazio(msg);
    if (vazio) return vazio;
    throw new ErroRota(400, msg);
  }
}

// ── leitura de parâmetros, como api/httpapi/json.go ──

function qInt(q: URLSearchParams, nome: string, padrao = 0): number {
  const raw = q.get(nome);
  if (!raw) return padrao;
  const v = Number.parseInt(raw, 10);
  return Number.isFinite(v) ? v : padrao;
}

function qFloat(q: URLSearchParams, nome: string, padrao: number): number {
  const raw = q.get(nome);
  if (!raw) return padrao;
  const v = Number.parseFloat(raw);
  return Number.isFinite(v) ? v : padrao;
}

function qUint8(q: URLSearchParams, nome: string): number {
  return Math.min(255, Math.max(0, qInt(q, nome)));
}

function qBool(q: URLSearchParams, nome: string): boolean {
  const v = (q.get(nome) ?? '').trim().toLowerCase();
  return v === '1' || v === 'true';
}

function qLista(q: URLSearchParams, nome: string): number[] {
  const raw = q.get(nome);
  if (!raw) return [];
  return raw.split(',').map(s => s.trim()).filter(Boolean).map(Number).filter(Number.isInteger);
}

/** targetManufacturerId: ausente = 0; presente e inválido = 400. */
function qFabricante(q: URLSearchParams): number {
  const raw = (q.get('targetManufacturerId') ?? '').trim();
  if (raw === '') return 0;
  const v = Number(raw);
  if (!Number.isInteger(v) || v < 0) throw new ErroRota(400, 'fabricante não encontrado');
  return v;
}

/** maxIngredients: ausente = 0 (teto padrão do motor); fora de 0-8 = 400. */
function qMaxIngredientes(q: URLSearchParams): number {
  const raw = (q.get('maxIngredients') ?? '').trim();
  if (raw === '') return 0;
  const v = Number(raw);
  if (!Number.isInteger(v) || v < 0 || v > 8) throw new ErroRota(400, 'maxIngredients fora da faixa');
  return v;
}

function idCaminho(bruto: string): number {
  const id = Number(bruto);
  if (!Number.isInteger(id)) throw new ErroRota(400, `id inválido: ${bruto}`);
  return id;
}

function idPositivo(bruto: string): number {
  const id = idCaminho(bruto);
  if (id <= 0) throw new ErroRota(400, `id inválido: ${id}`);
  return id;
}

function corValida(v: unknown): v is number {
  return Number.isInteger(v) && (v as number) >= 0 && (v as number) <= 255;
}

// ── respostas ──

function json(corpo: unknown, status = 200): Response {
  return new Response(JSON.stringify(corpo), {
    status,
    headers: { 'Content-Type': 'application/json; charset=utf-8' },
  });
}

function semCorpo(): Response {
  return new Response(null, { status: 204 });
}

function base64ParaBytes(b64: string): Uint8Array {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

/** handleRecipeByColorFromDeviceStock: itens inválidos são descartados. */
function itensEstoqueDoCorpo(lista: unknown): Record<string, unknown>[] {
  if (!Array.isArray(lista)) return [];
  if (lista.length > 2000) throw new ErroRota(400, 'estoque grande demais para o cálculo');
  const out: Record<string, unknown>[] = [];
  for (const it of lista as Record<string, unknown>[]) {
    if (!it || !corValida(it.r) || !corValida(it.g) || !corValida(it.b)) continue;
    const nome = String(it.name ?? '');
    const codigo = String(it.code ?? '');
    const fabricante = String(it.manufacturer ?? '');
    if ([...nome].length > 200 || [...codigo].length > 60 || [...fabricante].length > 120) continue;
    out.push({
      id: Number(it.id) || 0,
      manufacturerId: Number(it.manufacturerId) || 0,
      manufacturer: fabricante,
      name: nome,
      code: codigo,
      r: it.r,
      g: it.g,
      b: it.b,
      ignoreInMix: it.ignoreInMix === true,
    });
  }
  return out;
}

type Corpo = Record<string, any>;

async function atender(metodo: string, caminho: string, q: URLSearchParams, corpo: Corpo | null): Promise<Response> {
  const partes = caminho.split('/').filter(Boolean);
  const r1 = partes[1];
  const rota = `${metodo} /${partes.map((p, i) => (i > 0 && /^-?\d+$/.test(p) ? '{id}' : p)).join('/')}`;
  const b = corpo ?? {};

  switch (rota) {
    case 'GET /healthz':
      return json({ ok: true });

    // ── catálogo ──
    case 'GET /stats':
      return json(await chamar(Svc.GetStats(), 500));
    case 'GET /manufacturers':
      return json(await chamar(Svc.GetManufacturers(), 500));
    case 'POST /manufacturers':
      return json(await Svc.AddManufacturer(String(b.name ?? '')).catch(erroFabricante));
    case 'PUT /manufacturers/{id}':
      return json(await Svc.UpdateManufacturer(idPositivo(r1), String(b.name ?? '')).catch(erroFabricante));
    case 'DELETE /manufacturers/{id}': {
      const id = idPositivo(r1);
      await Svc.DeleteManufacturer(id).catch(erroFabricante);
      return json({ id });
    }
    case 'GET /paint-types':
      return json(await chamar(Svc.GetPaintTypes(), 500));
    case 'POST /paint-types':
      return json(await Svc.AddPaintType(String(b.name ?? '')).catch(erroTipo));
    case 'PUT /paint-types/{id}':
      return json(await Svc.UpdatePaintType(idPositivo(r1), String(b.name ?? '')).catch(erroTipo));
    case 'DELETE /paint-types/{id}': {
      const id = idPositivo(r1);
      await Svc.DeletePaintType(id).catch(erroTipo);
      return json({ id });
    }
    case 'GET /paints': {
      const busca = q.get('q') ?? '';
      return json(await chamar(busca ? Svc.SearchPaints(busca) : Svc.GetAllPaints(), 500));
    }
    case 'GET /paints/{id}':
      return json(await chamar(Svc.GetPaintByID(idCaminho(r1)), 404));
    case 'GET /paints/{id}/equivalences':
      return json(await chamar(Svc.FindEquivalences(idCaminho(r1)), 500));
    case 'PUT /paints/{id}/ignore-in-mix': {
      const id = Number(r1);
      if (!Number.isInteger(id) || id <= 0) throw new ErroRota(400, 'id inválido');
      if (typeof b.ignoreInMix !== 'boolean') throw new ErroRota(400, 'pedido inválido');
      try {
        await Svc.SetPaintIgnoreInMix(id, b.ignoreInMix);
      } catch (err) {
        if (mensagemGo(err) === 'tinta não encontrada') throw new ErroRota(404, 'tinta não encontrada');
        throw new ErroRota(500, 'não foi possível gravar a marca');
      }
      return json({ id, ignoreInMix: b.ignoreInMix });
    }
    case 'POST /paints/compare':
      return json(await chamar(Svc.CompareColors(b.paintIds ?? null), 400));
    case 'POST /query':
      return json(await chamar(Svc.ProcessQuery(String(b.text ?? '')), 500));

    // ── motor de cor ──
    case 'GET /similar':
      return json(await chamar(Svc.FindSimilar(
        qUint8(q, 'r'), qUint8(q, 'g'), qUint8(q, 'b'),
        qFloat(q, 'maxDeltaE', 10), qInt(q, 'maxResults', 10),
      ), 500));
    case 'GET /recipes/by-paint': {
      const fab = qFabricante(q);
      const max = qMaxIngredientes(q);
      return json(await chamar(Svc.SuggestEquivalentRecipe(qInt(q, 'sourcePaintId'), fab, max), 400));
    }
    case 'GET /recipes/by-color': {
      const fab = qFabricante(q);
      const max = qMaxIngredientes(q);
      return json(await receitaOuVazio(Svc.ResolverCorNoUniverso(
        qUint8(q, 'r'), qUint8(q, 'g'), qUint8(q, 'b'), fab,
        qBool(q, 'useStockOnly'), qBool(q, 'foraDoUniverso'), max,
      )));
    }
    case 'POST /recipes/by-color': {
      const max = b.maxIngredients ?? 0;
      const fab = b.targetManufacturerId ?? 0;
      const itens = itensEstoqueDoCorpo(b.stock);
      if (!corValida(b.r) || !corValida(b.g) || !corValida(b.b)) throw new ErroRota(400, 'cor inválida');
      if (!Number.isInteger(max) || max < 0 || max > 8) throw new ErroRota(400, 'maxIngredients fora da faixa');
      if (!Number.isInteger(fab) || fab < 0) throw new ErroRota(400, 'fabricante não encontrado');
      return json(await receitaOuVazio(Svc.ResolverCorComEstoqueDoAparelho(
        b.r, b.g, b.b, fab, b.foraDoUniverso === true, max, itens as any,
        b.respeitarIgnorados !== false,
      )));
    }
    case 'GET /recipes/best-delta-e': {
      const fab = qFabricante(q);
      const r = await receitaOuVazio(Svc.MelhorDeltaENoUniverso(
        qUint8(q, 'r'), qUint8(q, 'g'), qUint8(q, 'b'), fab, qBool(q, 'useStockOnly'),
      ));
      return json(typeof r === 'number' ? { deltaE: r } : r);
    }
    case 'GET /recipes':
      return json(await chamar(Svc.ListRecipes(), 500));
    case 'POST /recipes':
      return json(await chamar(Svc.SaveRecipe(String(b.name ?? ''), String(b.targetHex ?? '')), 400));
    case 'DELETE /recipes/{id}':
      await chamar(Svc.DeleteRecipe(idCaminho(r1)), 500);
      return json({ ok: true });
    case 'GET /recipes/{id}/resolve':
      if (qInt(q, 'targetManufacturerId') === 0) throw new ErroRota(400, 'targetManufacturerId é obrigatório');
      return json(await chamar(Svc.ResolveSavedRecipe(idCaminho(r1), qInt(q, 'targetManufacturerId')), 400));
    case 'GET /color-pick':
      return json(await chamar(Svc.PickColor(
        qUint8(q, 'r'), qUint8(q, 'g'), qUint8(q, 'b'), qInt(q, 'targetManufacturerId'),
      ), 400));
    case 'GET /compare-to-anchor': {
      const ancora = qInt(q, 'anchorId');
      const ids = qLista(q, 'ids');
      if (ancora === 0) throw new ErroRota(400, 'anchorId é obrigatório');
      if (ids.length > 20) throw new ErroRota(400, 'ids demais');
      return json(await chamar(Svc.CompareToAnchor(ancora, ids), 400));
    }
    case 'GET /best-brands': {
      const id = qInt(q, 'paintId');
      if (id === 0) throw new ErroRota(400, 'paintId é obrigatório');
      return json(await chamar(Svc.BestBrandsFor(id), 400));
    }

    // ── estoque ad-hoc ──
    case 'GET /stock/csv-template':
      return json({ csv: await chamar(Svc.UserPaintCSVTemplate(), 500) });
    case 'POST /stock/parse-csv': {
      const [paints, errors] = await chamar(Svc.ValidateStockCSV(String(b.csv ?? '')), 500);
      return json({ paints, errors });
    }
    case 'POST /stock/suggest-recipe':
      return json(await chamar(Svc.SuggestEquivalentFromPool(Number(b.sourcePaintId) || 0, b.stock ?? null), 400));

    // ── estoque persistido (rf-17) ──
    case 'GET /user-paints':
      return json(await chamar(Svc.GetUserPaints(), 500));
    case 'POST /user-paints':
      return json(await Svc.AddUserPaint({ ...b, id: 0 } as any).catch(erroEstoque), 201);
    case 'POST /user-paints/migrate':
      try {
        return json(await Svc.MigrateUserPaints(String(b.deviceId ?? ''), b.paints ?? null));
      } catch (err) {
        const msg = mensagemGo(err);
        if (/^(erro|sql|database|begin|commit)/i.test(msg)) throw new ErroRota(500, 'não foi possível migrar o estoque');
        throw new ErroRota(400, msg);
      }
    case 'PUT /user-paints/{id}':
      return json(await Svc.UpdateUserPaint({ ...b, id: idCaminho(r1) } as any).catch(erroEstoque));
    case 'DELETE /user-paints/{id}':
      await Svc.DeleteUserPaint(idCaminho(r1)).catch(erroEstoque);
      return semCorpo();
    case 'GET /user-paints/export-csv':
      return json({ csv: await chamar(Svc.ExportUserPaintsCSV(), 500) });
    case 'POST /user-paints/import-csv':
      return json(await chamar(Svc.ImportUserPaintsCSV(String(b.csv ?? '')), 500, 'não foi possível importar o estoque'));
    case 'GET /user-paints/suggest-recipe': {
      const id = qInt(q, 'sourcePaintId');
      if (id === 0) throw new ErroRota(400, 'sourcePaintId é obrigatório');
      return json(await chamar(Svc.SuggestEquivalentFromStock(id), 400));
    }

    // ── projetos de pintura ──
    case 'GET /plans':
      return json(await chamar(Svc.ListPlanSummaries(), 500));
    case 'POST /plans':
      return json(await chamar(Svc.SavePlan(b as any), 400));
    case 'GET /plans/{id}':
      return json(await chamar(Svc.LoadPlan(idCaminho(r1)), 404));
    case 'DELETE /plans/{id}':
      await chamar(Svc.DeletePlan(idCaminho(r1)), 404);
      return semCorpo();
    case 'GET /plans/{id}/report': {
      const id = Number(r1);
      if (!Number.isInteger(id) || id <= 0) throw new ErroRota(400, 'id inválido');
      const formato = q.get('format') || 'pdf';
      if (formato !== 'pdf' && formato !== 'png') throw new ErroRota(400, 'formato inválido');
      let corpoB64: string | null;
      let nome: string;
      try {
        [corpoB64, nome] = await Svc.BuildPlanReport(id, formato);
      } catch (err) {
        if (mensagemGo(err) === 'plano não encontrado') throw new ErroRota(404, 'plano não encontrado');
        throw new ErroRota(500, 'não foi possível gerar o relatório');
      }
      if (!nome || /[\r\n"]/.test(nome)) {
        nome = `mescla-plano-${new Date().toISOString().slice(0, 10)}.${formato}`;
      }
      const bytes = base64ParaBytes(corpoB64 ?? '');
      return new Response(bytes, {
        status: 200,
        headers: {
          'Content-Type': formato === 'png' ? 'image/png' : 'application/pdf',
          'Content-Disposition': `attachment; filename="${nome}"; filename*=UTF-8''${encodeURIComponent(nome)}`,
        },
      });
    }
  }

  throw new ErroRota(404, 'rota não encontrada');
}

/** Troca `window.fetch`: `/api/...` vai para os bindings; o resto segue igual. */
export function instalarPonteApi(): void {
  const fetchOriginal = window.fetch.bind(window);

  window.fetch = async (entrada: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const bruto = typeof entrada === 'string' ? entrada : entrada instanceof URL ? entrada.href : entrada.url;
    const url = new URL(bruto, window.location.origin);
    if (url.origin !== window.location.origin || !url.pathname.startsWith(`${PREFIXO}/`)) {
      return fetchOriginal(entrada, init);
    }

    const metodo = (init?.method ?? (entrada instanceof Request ? entrada.method : 'GET')).toUpperCase();
    let corpo: Corpo | null = null;
    if (typeof init?.body === 'string' && init.body !== '') {
      try {
        corpo = JSON.parse(init.body);
      } catch {
        return json({ error: 'pedido inválido' }, 400);
      }
    }

    try {
      return await atender(metodo, url.pathname.slice(PREFIXO.length), url.searchParams, corpo);
    } catch (err) {
      if (err instanceof ErroRota) return json({ error: err.message, ...err.extra }, err.status);
      return json({ error: mensagemGo(err) }, 500);
    }
  };
}
