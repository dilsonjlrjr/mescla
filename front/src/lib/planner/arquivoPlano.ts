// rf-22 — plano em arquivo JSON (RN19–RN21). Formato `mescla-plano` v3 para
// baixar e abrir na web, e leitura tolerante do formato antigo do desktop
// (`.mesclaplan`, JSON v1/v2, `regions` e `imageData` na raiz). Puro: sem tela
// e sem rede. O arquivo é entrada não confiável — cada campo passa pela
// normalização do rascunho e nada dele vira HTML, URL ou chave dinâmica.
import {
  normalizarRascunhoBruto, LIMITE_ABAS, LIMITE_REGIOES, type RegiaoRascunho,
} from './rascunho';
import { validarImagemDaAba, type PlanoDTO } from '../services/plans';

export const FORMATO_ARQUIVO = 'mescla-plano';
export const VERSAO_ARQUIVO = 3;
/** 10 abas × 2 MB de foto + folga. */
export const TETO_ARQUIVO_PLANO = 25 * 1024 * 1024;

const MAX_NOME_PLANO = 200;
const MAX_NOME_ABA = 80;

export interface RegiaoImportada extends RegiaoRascunho {
  /** Nome do fabricante do ajuste da região — casa em outra instalação (RN23). */
  regionManufacturerName: string;
}

export interface AbaImportada {
  name: string;
  imageData: string;
  regions: RegiaoImportada[];
}

export interface PlanoImportado {
  name: string;
  selectedManufacturerId: number | null;
  selectedManufacturerName: string;
  useStockOnly: boolean;
  tabs: AbaImportada[];
}

/** `chave` é uma chave i18n; `abaNome` acompanha os erros que são de uma aba. */
export type LeituraPlano =
  | { ok: true; plano: PlanoImportado }
  | { ok: false; chave: string; abaNome?: string };

const NAO_RECONHECIDO: LeituraPlano = { ok: false, chave: 'fileNotRecognized' };

function ehObjeto(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function texto(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

/** RN20: monta o arquivo v3 a partir do `PlanoDTO` de trabalho. Sem `id` de
 *  plano nem de aba — um id de outro servidor não significa nada. */
export function montarArquivoPlano(
  plano: PlanoDTO,
  nomeFabricante: (id: number | null) => string,
  agora: Date = new Date(),
): string {
  const arquivo = {
    formato: FORMATO_ARQUIVO,
    versao: VERSAO_ARQUIVO,
    exportadoEm: agora.toISOString(),
    plano: {
      name: plano.name,
      selectedManufacturerId: plano.selectedManufacturerId ?? null,
      selectedManufacturerName: nomeFabricante(plano.selectedManufacturerId ?? null),
      useStockOnly: plano.useStockOnly,
      tabs: plano.tabs.map(aba => ({
        name: aba.name,
        imageData: aba.imageData,
        regions: aba.regions.map(r => ({
          ...r,
          regionManufacturerName: r.regionOverride === 1 ? nomeFabricante(r.regionManufacturerId ?? null) : '',
        })),
      })),
    },
  };
  return JSON.stringify(arquivo);
}

/** RN19: `mescla-{slug do nome ou "plano"}-AAAA-MM-DD.json`. */
export function nomeArquivoPlano(nome: string, agora: Date = new Date()): string {
  const slug = nome
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
    .replace(/-+$/, '');
  const p2 = (n: number) => String(n).padStart(2, '0');
  const data = `${agora.getFullYear()}-${p2(agora.getMonth() + 1)}-${p2(agora.getDate())}`;
  return `mescla-${slug || 'plano'}-${data}.json`;
}

function nomeDaAba(nome: unknown, i: number): string {
  const n = texto(nome).trim();
  return n.length > 0 ? n : `Figura ${i + 1}`;
}

function cor(v: number): number {
  return Math.min(255, Math.max(0, Math.round(v)));
}

/** RN21: reconhece o v3 e o formato antigo; normaliza tudo ou recusa tudo. */
export function lerArquivoPlano(conteudo: string): LeituraPlano {
  let bruto: unknown;
  try {
    bruto = JSON.parse(conteudo);
  } catch {
    return NAO_RECONHECIDO;
  }
  if (!ehObjeto(bruto)) return NAO_RECONHECIDO;

  let nome = '';
  let fabId: unknown = null;
  let fabNome = '';
  let estoque: unknown = false;
  let abasBrutas: unknown[];

  if (bruto.formato === FORMATO_ARQUIVO) {
    const versao = bruto.versao;
    if (typeof versao === 'number' && versao > VERSAO_ARQUIVO) return { ok: false, chave: 'fileNewerVersion' };
    if (versao !== VERSAO_ARQUIVO) return NAO_RECONHECIDO;
    const plano = bruto.plano;
    if (!ehObjeto(plano) || !Array.isArray(plano.tabs)) return NAO_RECONHECIDO;
    nome = texto(plano.name);
    fabId = plano.selectedManufacturerId;
    fabNome = texto(plano.selectedManufacturerName);
    estoque = plano.useStockOnly;
    abasBrutas = plano.tabs;
  } else if (Array.isArray(bruto.regions) && typeof bruto.imageData === 'string') {
    // Antigo do desktop: só x, y, r, g, b e o nome do plano são aproveitados.
    nome = texto(bruto.name) || texto(bruto.planName);
    abasBrutas = [{
      name: 'Figura 1',
      imageData: bruto.imageData,
      regions: bruto.regions.filter(ehObjeto).map(r => ({ x: r.x, y: r.y, r: r.r, g: r.g, b: r.b })),
    }];
  } else {
    return NAO_RECONHECIDO;
  }

  // Só objetos entram, e antes da normalização: os índices de aba e de região
  // continuam casando com os nomes de fabricante lidos daqui.
  const abas = abasBrutas.filter(ehObjeto);
  if (abas.length > LIMITE_ABAS) return { ok: false, chave: 'errTabsMax' };
  const limpas = abas.map((a): Record<string, unknown> & { regions: Record<string, unknown>[] } => ({
    ...a,
    regions: (Array.isArray(a.regions) ? a.regions : []).filter(ehObjeto),
  }));

  for (let i = 0; i < limpas.length; i++) {
    const aba = limpas[i];
    const abaNome = nomeDaAba(aba.name, i);
    if (aba.regions.length > LIMITE_REGIOES) return { ok: false, chave: 'errRegionsMax', abaNome };
    // Imagem barrada ANTES da normalização, que troca uma imagem inválida
    // por vazio: "javascript:" e SVG recusam o arquivo, não viram aba sem foto.
    const img = aba.imageData;
    if (img !== undefined && img !== null && img !== '') {
      if (typeof img !== 'string') return { ok: false, chave: 'errImageType', abaNome };
      const erro = validarImagemDaAba(img);
      if (erro) return { ok: false, chave: erro, abaNome };
    }
  }

  const normalizado = normalizarRascunhoBruto({
    v: 2,
    name: nome,
    selectedManufacturerId: fabId,
    useStockOnly: estoque,
    tabs: limpas,
  });
  if (!normalizado || normalizado.truncado) return NAO_RECONHECIDO;

  const { rascunho } = normalizado;
  if (rascunho.tabs.length !== limpas.length) return NAO_RECONHECIDO;

  const tabs: AbaImportada[] = [];
  for (let i = 0; i < rascunho.tabs.length; i++) {
    const aba = rascunho.tabs[i];
    const nomeAba = aba.name.slice(0, MAX_NOME_ABA);
    if (aba.regions.length > 0 && !aba.imageData) {
      return { ok: false, chave: 'fileTabNoPhoto', abaNome: nomeDaAba(nomeAba, i) };
    }
    tabs.push({
      name: nomeAba,
      imageData: aba.imageData,
      regions: aba.regions.map((r, j): RegiaoImportada => {
        const rr = cor(r.r);
        const gg = cor(r.g);
        const bb = cor(r.b);
        return {
          ...r,
          // x/y são presos às dimensões da imagem quando ela carrega (PlannerView).
          x: Math.round(r.x),
          y: Math.round(r.y),
          r: rr, g: gg, b: bb,
          hex: `#${[rr, gg, bb].map(n => n.toString(16).padStart(2, '0')).join('')}`,
          regionManufacturerName: texto(limpas[i].regions[j].regionManufacturerName).slice(0, 100),
        };
      }),
    });
  }

  return {
    ok: true,
    plano: {
      name: rascunho.name.slice(0, MAX_NOME_PLANO),
      selectedManufacturerId: rascunho.selectedManufacturerId,
      selectedManufacturerName: fabNome.slice(0, 100),
      useStockOnly: rascunho.useStockOnly,
      tabs,
    },
  };
}
