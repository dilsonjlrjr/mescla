// Persistência do plano da peça (T2) — POST /plans (criar/atualizar) e
// GET /plans/{id} (retomar). Contrato espelha service.PaintingPlanDTO/
// PaintingTabDTO/PaintingRegionDTO do Go (rf-09): o plano agora carrega
// `tabs[]`, cada aba com sua foto, seu fabricante e suas regiões próprias.
// RN11 do rf-07 continua valendo: a mensagem crua do servidor (achado
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
  painted: 0 | 1;
  /** rf-11 RN7: veio de fora do universo pedido (com autorização do usuário). */
  foraDoUniverso?: 0 | 1;
  /** rf-16 RN19: 1 = a região usa o próprio fornecedor/estoque; 0 (ou
   *  ausente, cliente antigo) = segue o projeto. */
  regionOverride?: 0 | 1;
  regionManufacturerId?: number | null;
  regionUseStockOnly?: 0 | 1;
  /** rf-16 RN24: a mistura calculada, gravada junto do plano. */
  resultR?: number | null;
  resultG?: number | null;
  resultB?: number | null;
  /** `''` = sem mistura salva (RN27). */
  faixa?: string;
  method?: string;
  ingredients?: IngredienteDTO[];
}

/** rf-16: um pote da mistura salva. `paintId` negativo é tinta do estoque
 *  do aparelho; o servidor grava nulo e devolve 0. */
export interface IngredienteDTO {
  paintId: number;
  manufacturerId: number;
  manufacturer: string;
  name: string;
  code: string;
  r: number;
  g: number;
  b: number;
  percentage: number;
}

/** rf-16 RN2: um card da lista de projetos (`GET /plans`). */
export interface ResumoPlanoDTO {
  id: number;
  name: string;
  createdAt: string;
  updatedAt: string;
  regionCount: number;
  tabCount: number;
  paintedCount: number;
}

export interface AbaDTO {
  id?: number;
  name: string;
  imageData: string;
  regions: RegiaoDTO[];
}

// selectedManufacturerId/useStockOnly são do plano inteiro desde a mudança
// macro de 2026-09-07 — um controle só em T2, valendo para todas as abas.
// Antes disso eram campos de AbaDTO.
export interface PlanoDTO {
  id?: number;
  name: string;
  selectedManufacturerId?: number | null;
  useStockOnly: 0 | 1;
  tabs: AbaDTO[];
}

/** Código estável que a tela traduz — nunca o texto original do servidor. */
export class PlanoError extends Error {
  constructor(public readonly code: 'nao-encontrado' | 'invalido' | 'falha') {
    super(code);
  }
}

/** Erro de validação de `validarPlano`: a `chave` é traduzida pela tela, que
 *  monta a mensagem final — para os erros por aba, `abaNome` acompanha (RN15,
 *  ex.: "Figura 2: o plano aceita no máximo 50 regiões"). Erros de escopo do
 *  plano inteiro (quantidade de abas, nome do plano) não trazem `abaNome`. */
export interface ErroValidacaoPlano {
  chave: string;
  abaNome?: string;
}

const IMAGE_DATA_URL_PREFIXES = ['data:image/png;base64,', 'data:image/jpeg;base64,'];
const MAX_IMAGE_BYTES = 2 * 1024 * 1024;
const MAX_PLAN_NAME = 200;
const MAX_TAB_NAME = 80;
const MAX_ABAS = 10;
const MAX_REGIONS = 50;
const MAX_REGION_NAME = 100;
const MAX_NOTE = 2000;

/** rf-16 RN11: maior arquivo de foto que ainda cabe no teto de 2 MB depois de
 *  virar data URL — base64 aumenta em um terço, e o prefixo mais longo da
 *  lista branca (`data:image/jpeg;base64,`) tem 23 bytes. 1.572.846 bytes. */
export function tetoArquivoImagem(): number {
  const prefixoMaisLongo = Math.max(...IMAGE_DATA_URL_PREFIXES.map(p => p.length));
  return Math.floor((MAX_IMAGE_BYTES - prefixoMaisLongo) / 4) * 3;
}

export const TIPOS_IMAGEM_ACEITOS = ['image/png', 'image/jpeg'];

function logOriginalErrorOnlyInDev(context: string, original: unknown): void {
  if (import.meta.env.DEV) {
    console.debug(context, original);
  }
}

/** Nome efetivo da aba pra citar num erro — antes do salvar aplicar a
 *  RN3 (vazio vira `Figura N`), a mensagem já precisa de algo pra mostrar. */
function nomeEfetivoDaAba(aba: AbaDTO, indice: number): string {
  const nome = aba.name.trim();
  return nome.length > 0 ? nome : `Figura ${indice + 1}`;
}

/** Guarda da foto de uma aba (M5/CA17): prefixo em lista branca e teto de
 *  2 MB. Devolve a chave i18n do erro, ou `null` se a foto passa. Exportada
 *  para T2 aplicar a mesma guarda no upload (D-005) — no Salvar apenas, a
 *  foto grande já teria sido cortada pela RN8 antes do usuário ver o aviso. */
export function validarImagemDaAba(imageData: string): string | null {
  if (imageData.length === 0) return null;
  if (!IMAGE_DATA_URL_PREFIXES.some(prefixo => imageData.startsWith(prefixo))) return 'errImageType';
  if (new TextEncoder().encode(imageData).length > MAX_IMAGE_BYTES) return 'errImageMax';
  return null;
}

/** Ordem fixa das guardas (contrato da spec): quantidade de abas → nome do
 *  plano → por aba, na ordem nome/regiões/imagem/regionName/note — devolve a
 *  chave i18n do primeiro erro, com o nome da aba quando o erro é dela. */
export function validarPlano(dto: PlanoDTO): ErroValidacaoPlano | null {
  if (dto.tabs.length > MAX_ABAS) return { chave: 'errTabsMax' };

  if (dto.name.trim().length === 0) return { chave: 'errPlanNameEmpty' };
  if (dto.name.length > MAX_PLAN_NAME) return { chave: 'errPlanNameMax' };

  for (let i = 0; i < dto.tabs.length; i++) {
    const aba = dto.tabs[i];
    const abaNome = nomeEfetivoDaAba(aba, i);

    if (aba.name.length > MAX_TAB_NAME) return { chave: 'errTabNameMax', abaNome };
    if (aba.regions.length > MAX_REGIONS) return { chave: 'errRegionsMax', abaNome };

    const erroImagem = validarImagemDaAba(aba.imageData);
    if (erroImagem) return { chave: erroImagem, abaNome };

    if (aba.regions.some(r => r.regionName.length > MAX_REGION_NAME)) {
      return { chave: 'errRegionNameMax', abaNome };
    }
    if (aba.regions.some(r => r.note.length > MAX_NOTE)) {
      return { chave: 'errNoteMax', abaNome };
    }
  }

  return null;
}

function contarRegioes(plano: Pick<PlanoDTO, 'tabs'>): number {
  return (plano.tabs ?? []).reduce((acc, aba) => acc + (aba.regions?.length ?? 0), 0);
}

export async function salvarPlano(
  dto: PlanoDTO
): Promise<{ plano: PlanoDTO; suspeitaD003: boolean }> {
  const regioesEnviadas = contarRegioes(dto);
  try {
    const plano = await apiPost<PlanoDTO>('/plans', dto);
    const suspeitaD003 = regioesEnviadas > 0 && contarRegioes(plano) < regioesEnviadas;
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

/** rf-16 RN1: lista de projetos. Nunca traz foto nem região. */
export async function listarPlanos(): Promise<ResumoPlanoDTO[]> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/plans`);
  } catch (err) {
    logOriginalErrorOnlyInDev('listarPlanos: GET /plans falhou (rede)', err);
    throw new PlanoError('falha');
  }
  if (!res.ok) {
    logOriginalErrorOnlyInDev('listarPlanos: GET /plans respondeu erro', `HTTP ${res.status}`);
    throw new PlanoError('falha');
  }
  const lista = (await res.json()) as ResumoPlanoDTO[] | null;
  return Array.isArray(lista) ? lista : [];
}

/** rf-16 RN6: excluir da lista. 404 conta como sucesso — o projeto já não
 *  existe, e o card deve sumir do mesmo jeito. */
export async function excluirPlano(id: number): Promise<void> {
  if (!Number.isInteger(id) || id <= 0) {
    throw new PlanoError('falha');
  }
  let res: Response;
  try {
    res = await fetch(`${BASE}/plans/${id}`, { method: 'DELETE' });
  } catch (err) {
    logOriginalErrorOnlyInDev('excluirPlano: DELETE /plans falhou (rede)', err);
    throw new PlanoError('falha');
  }
  if (!res.ok && res.status !== 404) {
    logOriginalErrorOnlyInDev('excluirPlano: DELETE /plans respondeu erro', `HTTP ${res.status}`);
    throw new PlanoError('falha');
  }
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
