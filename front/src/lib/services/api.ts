// Cliente HTTP pro mescla-api (api/httpapi, fasthttp). Caminho relativo
// "/api/..." sempre — em dev o proxy do Vite (vite.config.ts) encaminha pra
// http://localhost:8080; em produção o nginx faz o mesmo (front/deploy/nginx.conf).
// Front não tem fallback offline: sem API alcançável, a chamada rejeita e a
// UI mostra erro (decisão do dono de 04/09/2026 — ver nota do reorg no vault).

const BASE = '/api';

/** Erro de chamada à API com o status HTTP (0 = sem resposta, rede). A tela
 *  escolhe a mensagem pelo status e pela operação, não pelo texto do servidor. */
export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function parseErrorBody(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    if (body?.error) return body.error;
  } catch {
    // corpo não é JSON — usa o texto de status mesmo
  }
  return `HTTP ${res.status}`;
}

async function send<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${BASE}${path}`, body === undefined
      ? { method }
      : { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  } catch (e) {
    throw new ApiError(e instanceof Error ? e.message : String(e), 0);
  }
  if (!res.ok) throw new ApiError(await parseErrorBody(res), res.status);
  // 204 não tem corpo: `res.json()` lançaria e uma exclusão bem-sucedida
  // pareceria falha (rf-17, DELETE /user-paints).
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export function apiGet<T>(path: string): Promise<T> {
  return send<T>('GET', path);
}

export function apiPost<T>(path: string, body: unknown): Promise<T> {
  return send<T>('POST', path, body);
}

export function apiPut<T>(path: string, body: unknown): Promise<T> {
  return send<T>('PUT', path, body);
}

export function apiDelete<T>(path: string): Promise<T> {
  return send<T>('DELETE', path);
}
