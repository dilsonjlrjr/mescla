// Navegação mobile com integração ao botão back do Android (History API).
//
// Modelo: a aba inicial (Mesclar) é a raiz. Camadas de UI (sheets, busca em
// tela cheia, o passo "resultado" da Mesclar) empilham entradas de histórico;
// o back fecha a camada do topo. Trocar de aba não empilha — exceto que sair
// da aba inicial cria UMA entrada, então o back de qualquer aba volta pra
// Mesclar, e um segundo back sai do app (convenção Android).

export type Tab = 'mesclar' | 'catalogo' | 'cor' | 'mais';

const HOME: Tab = 'mesclar';

export const nav = $state({ tab: HOME as Tab });

// Callbacks das camadas abertas, do fundo pro topo. O back (popstate) fecha o topo.
let layerStack: Array<() => void> = [];
// true quando NÓS chamamos history.back() pra consumir a entrada de aba —
// o popstate resultante não deve reagir de novo.
let suppressNextPop = false;
let tabEntryPushed = false;

export function switchTab(tab: Tab) {
  if (nav.tab === tab) return;
  // Trocar de aba com camada aberta não acontece pela UI (a tab bar some sob
  // sheets), mas por segurança fecha tudo primeiro.
  closeAllLayers();

  if (tab === HOME) {
    nav.tab = HOME;
    if (tabEntryPushed) {
      suppressNextPop = true;
      history.back();
    }
  } else {
    if (!tabEntryPushed) {
      history.pushState({ mescla: 'tab' }, '');
      tabEntryPushed = true;
    }
    nav.tab = tab;
  }
}

/** Empilha uma camada (sheet, busca, passo). onPop roda quando o back a fechar.
 *  Retorna a função de fechamento programático (usar no botão ✕/scrim). */
export function pushLayer(onPop: () => void): () => void {
  layerStack.push(onPop);
  history.pushState({ mescla: 'layer', depth: layerStack.length }, '');
  let closed = false;
  return () => {
    if (closed) return;
    closed = true;
    // Fechamento programático consome a entrada de histórico via back();
    // o popstate chama o onPop.
    history.back();
  };
}

function closeAllLayers() {
  while (layerStack.length > 0) {
    const cb = layerStack.pop()!;
    cb();
  }
}

export function initNav() {
  history.replaceState({ mescla: 'root' }, '');
  window.addEventListener('popstate', () => {
    if (suppressNextPop) {
      suppressNextPop = false;
      tabEntryPushed = false;
      return;
    }
    if (layerStack.length > 0) {
      const cb = layerStack.pop()!;
      cb();
      return;
    }
    if (tabEntryPushed) {
      tabEntryPushed = false;
      nav.tab = HOME;
      return;
    }
    // Raiz: o navegador cuida (sai do app) — nada a fazer.
  });
}
