// Cliente HTTP pro mescla-api (api/httpapi, fasthttp). Caminho relativo
// "/api/..." sempre — em dev o proxy do Vite (vite.config.ts) encaminha pra
// http://localhost:8080; em produção o nginx faz o mesmo (front/deploy/nginx.conf).
// Front não tem fallback offline: sem API alcançável, a chamada rejeita e a
// UI mostra erro (decisão do dono de 04/09/2026 — ver nota do reorg no vault).

const BASE = '/api';

async function parseErrorBody(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    if (body?.error) return body.error;
  } catch {
    // corpo não é JSON — usa o texto de status mesmo
  }
  return `HTTP ${res.status}`;
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(await parseErrorBody(res));
  return res.json() as Promise<T>;
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await parseErrorBody(res));
  return res.json() as Promise<T>;
}
