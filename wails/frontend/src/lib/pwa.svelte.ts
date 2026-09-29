// Instalação da PWA — captura o beforeinstallprompt e segura: o prompt só
// aparece no momento de valor (após a primeira receita) ou pela entrada
// permanente em Mais. Nunca no load.

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

const DISMISS_KEY = 'mescla.install.dismissed';
const DISMISS_DAYS = 7;

let deferred: BeforeInstallPromptEvent | null = null;

export const pwa = $state({ canInstall: false, installed: false });

export function initPwa() {
  window.addEventListener('beforeinstallprompt', e => {
    e.preventDefault();
    deferred = e as BeforeInstallPromptEvent;
    pwa.canInstall = true;
  });
  window.addEventListener('appinstalled', () => {
    pwa.installed = true;
    pwa.canInstall = false;
    deferred = null;
  });
}

export async function promptInstall() {
  if (!deferred) return;
  await deferred.prompt();
  await deferred.userChoice;
  deferred = null;
  pwa.canInstall = false;
}

/** Banner dispensado → silêncio por 7 dias (a entrada em Mais permanece). */
export function bannerDismissed(): boolean {
  const at = Number(localStorage.getItem(DISMISS_KEY) ?? 0);
  return Date.now() - at < DISMISS_DAYS * 24 * 60 * 60 * 1000;
}

export function dismissBanner() {
  localStorage.setItem(DISMISS_KEY, String(Date.now()));
}
