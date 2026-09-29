// Download no desktop: a interface da web baixa arquivo criando um <a download>
// com URL de blob e clicando nele. A webview do Wails não tem gerenciador de
// download, então aqui o clique vira o diálogo nativo de salvar e o arquivo é
// gravado pelo DialogService (wails/dialogs.go).
//
// O planejador revoga a URL logo depois do clique, no mesmo tique. Por isso o
// Blob é guardado quando a URL nasce e lido de forma síncrona no clique.

import { SaveFileAs, SaveFileWithData } from '../../bindings/paint-match-ai/wails/dialogservice';
import { toast } from '../lib/toast.svelte';
import { i18n, type Lang } from '../lib/i18n.svelte';

const blobs = new Map<string, Blob>();

// Textos próprios do desktop: ficam aqui para o dicionário copiado da web
// seguir idêntico ao de front/.
const TEXTOS: Record<Lang, { salvo: string; falhou: string }> = {
  pt: { salvo: 'Arquivo salvo.', falhou: 'Não foi possível salvar o arquivo.' },
  en: { salvo: 'File saved.', falhou: 'Could not save the file.' },
  es: { salvo: 'Archivo guardado.', falhou: 'No se pudo guardar el archivo.' },
  fr: { salvo: 'Fichier enregistré.', falhou: "Impossible d'enregistrer le fichier." },
};

function texto(chave: 'salvo' | 'falhou'): string {
  return (TEXTOS[i18n.lang] ?? TEXTOS.pt)[chave];
}

const FILTROS: Record<string, [string, string]> = {
  csv: ['CSV', '*.csv'],
  json: ['JSON', '*.json'],
  pdf: ['PDF', '*.pdf'],
  png: ['PNG', '*.png'],
};

function blobParaBase64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const leitor = new FileReader();
    leitor.onload = () => {
      const dataUrl = String(leitor.result ?? '');
      resolve(dataUrl.slice(dataUrl.indexOf(',') + 1));
    };
    leitor.onerror = () => reject(leitor.error);
    leitor.readAsDataURL(blob);
  });
}

async function salvar(nome: string, blob: Blob): Promise<void> {
  const ext = /\.([a-z0-9]+)$/i.exec(nome)?.[1]?.toLowerCase() ?? '';
  const [filtroNome, filtro] = FILTROS[ext] ?? ['', ''];
  try {
    const caminho = await SaveFileAs(nome, nome, filtroNome, filtro);
    if (!caminho) return; // cancelado
    await SaveFileWithData(caminho, await blobParaBase64(blob), ext ? `.${ext}` : '');
    toast(texto('salvo'));
  } catch (err) {
    console.error('download nativo falhou', err);
    toast(texto('falhou'), 'error');
  }
}

export function instalarDownloadNativo(): void {
  const criar = URL.createObjectURL.bind(URL);
  const revogar = URL.revokeObjectURL.bind(URL);

  URL.createObjectURL = (obj: Blob | MediaSource): string => {
    const url = criar(obj);
    if (obj instanceof Blob) blobs.set(url, obj);
    return url;
  };
  URL.revokeObjectURL = (url: string): void => {
    blobs.delete(url);
    revogar(url);
  };

  const clicar = HTMLAnchorElement.prototype.click;
  HTMLAnchorElement.prototype.click = function (this: HTMLAnchorElement) {
    const blob = this.hasAttribute('download') ? blobs.get(this.href) : undefined;
    if (!blob) return clicar.call(this);
    void salvar(this.download || 'arquivo', blob);
  };
}
