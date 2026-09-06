// Persistência do plano da peça (T2) — POST /plans (criar/atualizar) e
// GET /plans/{id} (retomar). Contrato espelha service.PaintingPlanDTO/
// PaintingRegionDTO do Go (rf-07). RN11: a mensagem crua do servidor (achado
// S-001, texto do driver SQLite) nunca sai deste módulo — só PlanoError.code.

import { apiPost } from './api';

// Mesmo caminho relativo de api.ts (proxy do Vite em dev, nginx em prod).
// Refeito aqui (em vez de apiGet) porque carregarPlano precisa do status
// HTTP para distinguir 404 — apiGet descarta o status ao lançar o erro.
const BASE = '/api';

export interface RegiaoDTO {
  x: number;
  y: number;
  r: number;
  g: number;
  b: number;
  hex: string;
  regionName: string;
  note: string;
  paintId?: number | null;
  paintBrand: string;
  paintName: string;
  paintCode: string;
  deltaE: number;
  painted: number;
}

export interface PlanoDTO {
  id?: number;
  name: string;
  imageData: string;
  selectedManufacturerId?: number | null;
  regions: RegiaoDTO[];
}

/** Código estável que a tela traduz — nunca o texto original do servidor. */
export class PlanoError extends Error {
  constructor(public readonly code: 'nao-encontrado' | 'invalido' | 'falha') {
    super(code);
  }
}

const IMAGE_DATA_URL_PREFIXES = ['data:image/png;base64,', 'data:image/jpeg;base64,'];
const MAX_IMAGE_BYTES = 2 * 1024 * 1024;
const MAX_PLAN_NAME = 200;
const MAX_REGIONS = 50;
const MAX_REGION_NAME = 100;
const MAX_NOTE = 2000;

function logOriginalErrorOnlyInDev(context: string, original: unknown): void {
  if (import.meta.env.DEV) {
    console.debug(context, original);
  }
}

/** Ordem fixa das guardas (contrato da spec) — devolve a chave i18n do primeiro erro. */
export function validarPlano(dto: PlanoDTO): string | null {
  if (dto.name.trim().length === 0) return 'errPlanNameEmpty';
  if (dto.name.length > MAX_PLAN_NAME) return 'errPlanNameMax';
  if (dto.regions.length > MAX_REGIONS) return 'errRegionsMax';

  if (dto.imageData.length > 0) {
    const temPrefixoValido = IMAGE_DATA_URL_PREFIXES.some(prefixo =>
      dto.imageData.startsWith(prefixo)
    );
    if (!temPrefixoValido) return 'errImageType';
    if (new TextEncoder().encode(dto.imageData).length > MAX_IMAGE_BYTES) return 'errImageMax';
  }

  if (dto.regions.some(r => r.regionName.length > MAX_REGION_NAME)) return 'errRegionNameMax';
  if (dto.regions.some(r => r.note.length > MAX_NOTE)) return 'errNoteMax';

  return null;
}

export async function salvarPlano(
  dto: PlanoDTO
): Promise<{ plano: PlanoDTO; suspeitaD003: boolean }> {
  const regioesEnviadas = dto.regions.length;
  try {
    const plano = await apiPost<PlanoDTO>('/plans', dto);
    const suspeitaD003 = regioesEnviadas > 0 && (plano.regions?.length ?? 0) < regioesEnviadas;
    return { plano, suspeitaD003 };
  } catch (err) {
    logOriginalErrorOnlyInDev('salvarPlano: POST /plans falhou', err);
    throw new PlanoError('falha');
  }
}

export async function carregarPlano(id: number): Promise<PlanoDTO> {
  if (!Number.isInteger(id) || id <= 0) {
    throw new PlanoError('falha');
  }

  let res: Response;
  try {
    res = await fetch(`${BASE}/plans/${id}`);
  } catch (err) {
    logOriginalErrorOnlyInDev('carregarPlano: GET /plans falhou (rede)', err);
    throw new PlanoError('falha');
  }

  if (!res.ok) {
    let corpo: unknown = `HTTP ${res.status}`;
    try {
      corpo = await res.json();
    } catch {
      // corpo não é JSON — mantém o texto de status
    }
    logOriginalErrorOnlyInDev('carregarPlano: GET /plans respondeu erro', corpo);
    throw new PlanoError(res.status === 404 ? 'nao-encontrado' : 'falha');
  }

  return res.json() as Promise<PlanoDTO>;
}

/** RN12: extrai o nome do arquivo de `Content-Disposition` — RFC 5987
 *  (`filename*=UTF-8''...`) primeiro, depois `filename="..."` simples. */
function parseFilename(disposition: string): string | null {
  const utf8 = /filename\*=UTF-8''([^;]+)/i.exec(disposition);
  if (utf8) {
    try {
      return decodeURIComponent(utf8[1]);
    } catch {
      // sequência não decodifica — cai para o filename simples abaixo
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(disposition);
  return plain ? plain[1] : null;
}

/** rf-08 — GET /plans/{id}/report?format=pdf|png (RN12: fetch + blob, nunca
 *  navegação direta). `id` validado como inteiro positivo antes de entrar na
 *  URL (CAN1). Mesma disciplina de erro de `carregarPlano`. */
export async function baixarRelatorio(
  id: number,
  formato: 'pdf' | 'png'
): Promise<{ blob: Blob; filename: string }> {
  if (!Number.isInteger(id) || id <= 0) {
    throw new PlanoError('falha');
  }

  let res: Response;
  try {
    res = await fetch(`${BASE}/plans/${id}/report?format=${formato}`);
  } catch (err) {
    logOriginalErrorOnlyInDev('baixarRelatorio: GET /plans/report falhou (rede)', err);
    throw new PlanoError('falha');
  }

  if (!res.ok) {
    let corpo: unknown = `HTTP ${res.status}`;
    try {
      corpo = await res.json();
    } catch {
      // corpo não é JSON — mantém o texto de status
    }
    logOriginalErrorOnlyInDev('baixarRelatorio: GET /plans/report respondeu erro', corpo);
    if (res.status === 404) throw new PlanoError('nao-encontrado');
    if (res.status === 400) throw new PlanoError('invalido');
    throw new PlanoError('falha');
  }

  const blob = await res.blob();
  // 200 com corpo vazio baixaria um arquivo de 0 byte sem nenhum aviso.
  if (blob.size === 0) throw new PlanoError('falha');
  const filename = parseFilename(res.headers.get('Content-Disposition') ?? '') ?? `relatorio.${formato}`;
  return { blob, filename };
}
