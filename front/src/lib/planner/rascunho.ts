// Rascunho local de T2 (Plano da peça): auto save sem rede, com debounce e
// escada de cota. Formato e chaves congelados na nota macro de 2026-09-06
// (M5) e detalhados em rf-07. O timer do debounce mora aqui, não na tela.

const RASCUNHO_KEY = 'mescla:plano-rascunho';
const PLANO_ATIVO_KEY = 'mescla:plano-ativo';

const DEBOUNCE_MS = 1500;
const MAX_REGIOES = 50;

const PREFIXOS_IMAGEM_VALIDOS = ['data:image/png;base64,', 'data:image/jpeg;base64,'];

export interface RegiaoRascunho {
  x: number;
  y: number;
  r: number;
  g: number;
  b: number;
  hex: string;
  regionName: string;
  note: string;
  paintId: number | null;
  paintBrand: string;
  paintName: string;
  paintCode: string;
  deltaE: number;
  painted: boolean;
}

export interface Rascunho {
  planId: number | null;
  name: string;
  imageData: string;
  regions: RegiaoRascunho[];
  selectedManufacturerId: number | null;
  salvoEm: string;
}

export type EstadoAutoSave = 'ligado' | 'sem-foto' | 'desligado';

let timer: ReturnType<typeof setTimeout> | null = null;
let estado: EstadoAutoSave = 'ligado';

function normalizarPainted(valor: unknown): boolean {
  return valor === true || valor === 1;
}

function normalizarImagem(imageData: unknown): string {
  if (typeof imageData !== 'string') return '';
  return PREFIXOS_IMAGEM_VALIDOS.some(p => imageData.startsWith(p)) ? imageData : '';
}

function normalizarRegiao(r: unknown): RegiaoRascunho | null {
  if (typeof r !== 'object' || r === null) return null;
  const o = r as Record<string, unknown>;
  return {
    x: Number(o.x) || 0,
    y: Number(o.y) || 0,
    r: Number(o.r) || 0,
    g: Number(o.g) || 0,
    b: Number(o.b) || 0,
    hex: typeof o.hex === 'string' ? o.hex : '',
    regionName: typeof o.regionName === 'string' ? o.regionName : '',
    note: typeof o.note === 'string' ? o.note : '',
    paintId: typeof o.paintId === 'number' ? o.paintId : null,
    paintBrand: typeof o.paintBrand === 'string' ? o.paintBrand : '',
    paintName: typeof o.paintName === 'string' ? o.paintName : '',
    paintCode: typeof o.paintCode === 'string' ? o.paintCode : '',
    deltaE: Number(o.deltaE) || 0,
    painted: normalizarPainted(o.painted),
  };
}

/** Normalização defensiva de um rascunho lido do disco: JSON inválido ou
 *  formato inesperado nunca quebra a tela (CAN6, CAN7, CAN8). */
/** Identificador vindo do rascunho: só inteiro positivo passa. Rascunho
 *  adulterado com `-7` ou `1.5` viraria `id` no POST (CAN1). */
function idValido(v: unknown): number | null {
  return typeof v === 'number' && Number.isInteger(v) && v > 0 ? v : null;
}

function normalizar(bruto: unknown): { rascunho: Rascunho; truncado: boolean } | null {
  if (typeof bruto !== 'object' || bruto === null) return null;
  const o = bruto as Record<string, unknown>;

  const regioesBrutas = Array.isArray(o.regions) ? o.regions : [];
  const regioesValidas = regioesBrutas
    .map(normalizarRegiao)
    .filter((r): r is RegiaoRascunho => r !== null);
  const truncado = regioesValidas.length > MAX_REGIOES;
  const regions = truncado ? regioesValidas.slice(0, MAX_REGIOES) : regioesValidas;

  const rascunho: Rascunho = {
    planId: idValido(o.planId),
    name: typeof o.name === 'string' ? o.name : '',
    imageData: normalizarImagem(o.imageData),
    regions,
    selectedManufacturerId: idValido(o.selectedManufacturerId),
    salvoEm: typeof o.salvoEm === 'string' ? o.salvoEm : new Date().toISOString(),
  };
  return { rascunho, truncado };
}

/** Lê o rascunho gravado. JSON inválido apaga a chave e devolve `corrompido:
 *  true` (CAN7); mais de 50 regiões corta e devolve `truncado: true` (CAN8). */
export function lerRascunho(): { rascunho: Rascunho | null; corrompido: boolean; truncado: boolean } {
  let raw: string | null;
  try {
    raw = localStorage.getItem(RASCUNHO_KEY);
  } catch {
    return { rascunho: null, corrompido: false, truncado: false };
  }
  if (!raw) return { rascunho: null, corrompido: false, truncado: false };

  let bruto: unknown;
  try {
    bruto = JSON.parse(raw);
  } catch {
    apagarRascunho();
    return { rascunho: null, corrompido: true, truncado: false };
  }

  const normalizado = normalizar(bruto);
  if (normalizado === null) {
    apagarRascunho();
    return { rascunho: null, corrompido: true, truncado: false };
  }
  return { rascunho: normalizado.rascunho, corrompido: false, truncado: normalizado.truncado };
}

function gravar(payload: Rascunho): void {
  try {
    localStorage.setItem(RASCUNHO_KEY, JSON.stringify(payload));
  } catch {
    if (estado === 'desligado') return;
    // Degrau único da RN8: cortar a foto. Vale tanto na primeira falha
    // quanto já em `sem-foto` — o rascunho sem imagem ainda salva o
    // trabalho, e só desliga quando nem ele cabe.
    try {
      localStorage.setItem(RASCUNHO_KEY, JSON.stringify({ ...payload, imageData: '' }));
      estado = 'sem-foto';
    } catch {
      estado = 'desligado';
    }
  }
}

/** Agenda a gravação do rascunho após 1500 ms de silêncio (RN3); chamadas
 *  seguidas dentro da janela resultam em uma única gravação (CA5, CA6). */
export function agendarGravacao(payload: Rascunho): void {
  if (estado === 'desligado') return;
  if (timer !== null) clearTimeout(timer);
  timer = setTimeout(() => {
    timer = null;
    gravar(payload);
  }, DEBOUNCE_MS);
}

export function cancelarGravacao(): void {
  if (timer !== null) {
    clearTimeout(timer);
    timer = null;
  }
}

export function apagarRascunho(): void {
  cancelarGravacao();
  try {
    localStorage.removeItem(RASCUNHO_KEY);
  } catch {
    /* sem armazenamento, sem rascunho pra apagar mesmo */
  }
}

export function lerPlanoAtivo(): number | null {
  try {
    const raw = localStorage.getItem(PLANO_ATIVO_KEY);
    if (!raw) return null;
    const id = Number(raw);
    return Number.isFinite(id) ? id : null;
  } catch {
    return null;
  }
}

export function gravarPlanoAtivo(id: number): void {
  try {
    localStorage.setItem(PLANO_ATIVO_KEY, String(id));
  } catch {
    /* sem persistência, sem drama */
  }
}

export function limparPlanoAtivo(): void {
  try {
    localStorage.removeItem(PLANO_ATIVO_KEY);
  } catch {
    /* sem persistência, sem drama */
  }
}

/** Reflete a escada de cota da RN8: 'ligado' → 'sem-foto' → 'desligado'. */
export function estadoAutoSave(): EstadoAutoSave {
  return estado;
}
