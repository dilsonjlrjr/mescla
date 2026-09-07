// Rascunho local de T2 (Plano da peça): auto save sem rede, com debounce e
// escada de cota. Formato v2 (rf-09, várias abas) — migra o v1 (rf-07, uma
// foto/regiões na raiz) na leitura. O timer do debounce mora aqui, não na tela.

const RASCUNHO_KEY = 'mescla:plano-rascunho';
const PLANO_ATIVO_KEY = 'mescla:plano-ativo';

const DEBOUNCE_MS = 1500;
const MAX_REGIOES = 50;
const MAX_ABAS = 10;

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

export interface AbaRascunho {
  /** Identidade estável da aba no servidor (achado 2 do guardrail rf-09):
   *  ausente numa aba nunca salva. Usada para casar a aba do rascunho com a
   *  do servidor na hidratação — nunca a posição na tira, que pode ter sido
   *  reordenada depois do último Salvar. */
  id?: number | null;
  name: string;
  imageData: string;
  selectedManufacturerId: number | null;
  useStockOnly: boolean;
  regions: RegiaoRascunho[];
}

export interface Rascunho {
  v: 2;
  planId: number | null;
  name: string;
  abaAtiva: number;
  tabs: AbaRascunho[];
  salvoEm: string;
}

export type EstadoAutoSave = 'ligado' | 'sem-foto' | 'desligado';

let timer: ReturnType<typeof setTimeout> | null = null;
let estado: EstadoAutoSave = 'ligado';

function normalizarBooleano(valor: unknown): boolean {
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
    painted: normalizarBooleano(o.painted),
  };
}

/** Identificador vindo do rascunho: só inteiro positivo passa. Rascunho
 *  adulterado com `-7` ou `1.5` viraria `id` no POST (CAN1). */
function idValido(v: unknown): number | null {
  return typeof v === 'number' && Number.isInteger(v) && v > 0 ? v : null;
}

/** Uma aba do rascunho, com o mesmo corte de 50 regiões (CAN5 estende a
 *  aba). `truncado` sinaliza que a aba tinha mais regiões do que o teto. */
function normalizarAba(bruto: unknown): { aba: AbaRascunho; truncado: boolean } | null {
  if (typeof bruto !== 'object' || bruto === null) return null;
  const o = bruto as Record<string, unknown>;

  const regioesBrutas = Array.isArray(o.regions) ? o.regions : [];
  const regioesValidas = regioesBrutas
    .map(normalizarRegiao)
    .filter((r): r is RegiaoRascunho => r !== null);
  const truncado = regioesValidas.length > MAX_REGIOES;
  const regions = truncado ? regioesValidas.slice(0, MAX_REGIOES) : regioesValidas;

  const aba: AbaRascunho = {
    id: idValido(o.id),
    name: typeof o.name === 'string' ? o.name : '',
    imageData: normalizarImagem(o.imageData),
    selectedManufacturerId: idValido(o.selectedManufacturerId),
    useStockOnly: normalizarBooleano(o.useStockOnly),
    regions,
  };
  return { aba, truncado };
}

/** Normalização defensiva de um rascunho lido do disco: JSON inválido ou
 *  formato inesperado nunca quebra a tela (CAN6). RN11: payload sem `v` e
 *  com `regions` na raiz é o formato v1 (rf-07) — migra para uma aba única
 *  `Figura 1`, sem descartar nada. */
function normalizar(bruto: unknown): { rascunho: Rascunho; truncado: boolean } | null {
  if (typeof bruto !== 'object' || bruto === null) return null;
  const o = bruto as Record<string, unknown>;

  const ehV1 = o.v !== 2 && Array.isArray(o.regions);

  const tabsBrutas: unknown[] = ehV1
    ? [
        {
          name: 'Figura 1',
          imageData: o.imageData,
          selectedManufacturerId: o.selectedManufacturerId,
          useStockOnly: false,
          regions: o.regions,
        },
      ]
    : Array.isArray(o.tabs)
      ? o.tabs
      : [];

  const abasNormalizadas = tabsBrutas
    .map(normalizarAba)
    .filter((a): a is { aba: AbaRascunho; truncado: boolean } => a !== null);

  const truncadoAbas = abasNormalizadas.length > MAX_ABAS;
  const abasFinal = truncadoAbas ? abasNormalizadas.slice(0, MAX_ABAS) : abasNormalizadas;
  const truncadoRegioes = abasFinal.some(a => a.truncado);

  // Plano/rascunho sem aba é impossível (mesma invariante do servidor,
  // RN1) — um payload vazio ou irreconhecível ainda vira uma aba em branco.
  const tabs = abasFinal.length > 0 ? abasFinal.map(a => a.aba) : [
    { name: '', imageData: '', selectedManufacturerId: null, useStockOnly: false, regions: [] },
  ];

  const abaAtivaBruta = Number.isInteger(o.abaAtiva) ? (o.abaAtiva as number) : 0;
  const abaAtiva = Math.min(Math.max(abaAtivaBruta, 0), tabs.length - 1);

  const rascunho: Rascunho = {
    v: 2,
    planId: idValido(o.planId),
    name: typeof o.name === 'string' ? o.name : '',
    abaAtiva,
    tabs,
    salvoEm: typeof o.salvoEm === 'string' ? o.salvoEm : new Date().toISOString(),
  };
  return { rascunho, truncado: truncadoAbas || truncadoRegioes };
}

/** Lê o rascunho gravado. JSON inválido apaga a chave e devolve `corrompido:
 *  true` (CAN6); mais de 10 abas ou mais de 50 regiões numa aba corta e
 *  devolve `truncado: true` (CAN5). Rascunho v1 nunca é descartado em
 *  silêncio — é migrado (RN11, CA17). */
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
    // Degrau único da RN8 (estendida a N abas): cortar a foto de *todas* as
    // abas. O rascunho sem imagem ainda salva o trabalho (nomes, regiões,
    // cores), e só desliga quando nem ele cabe.
    try {
      const semFotos: Rascunho = {
        ...payload,
        tabs: payload.tabs.map(t => ({ ...t, imageData: '' })),
      };
      localStorage.setItem(RASCUNHO_KEY, JSON.stringify(semFotos));
      estado = 'sem-foto';
    } catch {
      estado = 'desligado';
    }
  }
}

/** Agenda a gravação do rascunho após 1500 ms de silêncio (RN3 do rf-07);
 *  chamadas seguidas dentro da janela resultam em uma única gravação. */
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
