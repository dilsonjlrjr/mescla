<script lang="ts">
  // rf-24 — guia de primeiro uso. Montado uma vez em App.svelte: abre sozinho
  // na primeira vez (marca `mescla.guia.v1` ausente) e volta pelo "?" do
  // cabeçalho ou pela paleta (`appState.guiaAberta`). Fechar grava a marca.
  //
  // RN6: não abre por cima da faixa de migração do estoque (rf-17) nem de uma
  // camada aberta; fica para o próximo boot. Sem `localStorage`, a marca vive
  // em memória e o guia abre uma vez por sessão.
  //
  // Entra como camada de navegação (`pushLayer`), como a paleta: o "voltar" do
  // Android o fecha, e uma ação que troca de tela espera a camada sair.
  import { onDestroy, onMount, tick, untrack } from 'svelte';
  import BrandMark from './BrandMark.svelte';
  import { switchTab, pushLayer, COMPARAR_VISIVEL, type Tab } from '../nav.svelte';
  import { appState } from '../appState.svelte';
  import { stock, estoqueAssentado } from '../services/stock.svelte';
  import { t, type DictKey } from '../i18n.svelte';

  const CHAVE = 'mescla.guia.v1';
  let vistoNaSessao = false;

  function jaViu(): boolean {
    if (vistoNaSessao) return true;
    try {
      return localStorage.getItem(CHAVE) !== null;
    } catch {
      return false;
    }
  }

  function marcar() {
    vistoNaSessao = true;
    try {
      localStorage.setItem(CHAVE, '1');
    } catch {
      // sem armazenamento: a marca em memória basta pela sessão
    }
  }

  /** RN6: faixa de migração do estoque, camada aberta ou diálogo modal na tela. */
  function ocupado(): boolean {
    if (stock.falhaEnvio || stock.naoSubiram.length > 0) return true;
    if (appState.paletaAberta) return true;
    if (history.state?.mescla === 'layer') return true;
    // Só os visíveis: T4 mantém alertdialogs estáticos no DOM, escondidos.
    return [...document.querySelectorAll('[aria-modal="true"]')].some(el => el.getClientRects().length > 0);
  }

  let montado = true;
  onMount(() => {
    void estoqueAssentado().then(() => {
      if (!montado || jaViu() || ocupado()) return;
      appState.guiaAberta = true;
    });
  });

  let layerAtiva = false;
  let fecharLayer: (() => void) | null = null;
  let origem: HTMLElement | null = null;
  let depois: (() => void) | null = null;
  let restaurarFoco = true;
  let estavaAberta = false;
  let dialogEl: HTMLDivElement | undefined = $state();

  function abrir() {
    if (layerAtiva) return;
    estavaAberta = true;
    restaurarFoco = true;
    origem = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    fecharLayer = pushLayer(() => {
      layerAtiva = false;
      marcar();
      appState.guiaAberta = false;
      const f = depois;
      depois = null;
      f?.();
    });
    layerAtiva = true;
    void tick().then(() => dialogEl?.focus());
  }

  function aoFechar() {
    if (!estavaAberta) return;
    estavaAberta = false;
    if (layerAtiva) {
      layerAtiva = false;
      fecharLayer?.();
    }
    const o = origem;
    origem = null;
    if (restaurarFoco && o?.isConnected) o.focus();
  }

  $effect(() => {
    if (appState.guiaAberta) untrack(abrir);
    else untrack(aoFechar);
  });

  onDestroy(() => {
    montado = false;
  });

  /** Fecha; `seguinte` roda depois da camada sair. */
  function fechar(seguinte?: () => void) {
    if (seguinte) restaurarFoco = false;
    depois = seguinte ?? null;
    if (layerAtiva && fecharLayer) {
      fecharLayer();
    } else {
      marcar();
      appState.guiaAberta = false;
      const f = depois;
      depois = null;
      f?.();
    }
  }

  function aoTeclar(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      fechar();
    } else if (e.key === 'Tab' && dialogEl) {
      const focaveis = [...dialogEl.querySelectorAll<HTMLElement>('button')].filter(el => !el.hasAttribute('disabled'));
      if (focaveis.length === 0) return;
      const primeiro = focaveis[0];
      const ultimo = focaveis[focaveis.length - 1];
      if (e.shiftKey && (document.activeElement === primeiro || document.activeElement === dialogEl)) {
        e.preventDefault();
        ultimo.focus();
      } else if (!e.shiftKey && document.activeElement === ultimo) {
        e.preventDefault();
        primeiro.focus();
      }
    }
  }

  const PASSOS: DictKey[] = ['guiaS1', 'guiaS2', 'guiaS3'];

  const TODAS_TELAS: Array<{ tab: Tab; n: string; chave: DictKey; icone: string }> = [
    { tab: 'pergunta', n: 'T1', chave: 'question', icone: 'ph-chat-circle-text' },
    { tab: 'plano', n: 'T2', chave: 'navPlano', icone: 'ph-crosshair' },
    { tab: 'receitas', n: 'T3', chave: 'navReceitas', icone: 'ph-flask' },
    { tab: 'estante', n: 'T4', chave: 'navTintas', icone: 'ph-paint-bucket' },
    { tab: 'circulo', n: 'T5', chave: 'navCirculo', icone: 'ph-circle-half-tilt' },
    { tab: 'comparar', n: 'T6', chave: 'navComparar', icone: 'ph-columns' },
  ];
  const TELAS = TODAS_TELAS.filter(tela => COMPARAR_VISIVEL || tela.tab !== 'comparar');

  // Faixas do ΔE00 = as de `verdictKeys` (ui.ts): <1, <2, <4, <8, o resto.
  const FAIXAS: Array<{ faixa: string; chave: DictKey; cor: string }> = [
    { faixa: '< 1', chave: 'v1', cor: 'var(--color-accent-400)' },
    { faixa: '1 – 2', chave: 'v2', cor: 'var(--color-accent-400)' },
    { faixa: '2 – 4', chave: 'v3', cor: 'var(--color-neutral-300)' },
    { faixa: '4 – 8', chave: 'v4', cor: 'var(--color-neutral-400)' },
    { faixa: '≥ 8', chave: 'v5', cor: 'var(--color-neutral-500)' },
  ];
</script>

{#if appState.guiaAberta}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    style="position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; padding: 16px; background: rgba(9, 10, 16, 0.7);"
    onclick={(e) => { if (e.target === e.currentTarget) fechar(); }}
  >
    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-label={t('guiaTitle')}
      tabindex="-1"
      onkeydown={aoTeclar}
      style="width: min(620px, 100%); max-height: 100%; display: flex; flex-direction: column; gap: 18px; overflow-y: auto; padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55); color: var(--color-text); font-family: var(--font-body); user-select: none;"
    >
      <div style="display: flex; align-items: center; gap: 14px;">
        <BrandMark size={40} />
        <h2 style="margin: 0; flex: 1; min-width: 0; font-size: 22px; font-weight: 500; letter-spacing: -0.02em;">{t('guiaTitle')}</h2>
        <button
          class="pressable"
          onclick={() => fechar()}
          aria-label={t('ariaClose')}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; flex-shrink: 0; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;"
        ><i class="ph ph-x" style="font-size: 18px;"></i></button>
      </div>

      <p style="margin: 0; font-size: 16px; color: var(--color-neutral-300); text-wrap: pretty;">{t('guiaLead')}</p>

      <ol style="margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: 10px;">
        {#each PASSOS as chave, i (chave)}
          <li style="display: flex; align-items: flex-start; gap: 12px; font-size: 15px; text-wrap: pretty;">
            <span class="font-mono" style="display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; flex-shrink: 0; border-radius: 999px; background: var(--color-accent-900); color: var(--color-accent-400); font-size: 13px;">{i + 1}</span>
            <span style="padding-top: 2px;">{t(chave)}</span>
          </li>
        {/each}
      </ol>

      <div style="display: flex; flex-direction: column; gap: 8px;">
        <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{t('guiaDe')}</span>
        <ul style="margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: 4px;">
          {#each FAIXAS as f (f.chave)}
            <li style="display: flex; align-items: baseline; gap: 12px; font-size: 14px;">
              <span class="font-mono" style="width: 56px; flex-shrink: 0; color: {f.cor};">{f.faixa}</span>
              <span style="color: var(--color-neutral-300);">{t(f.chave)}</span>
            </li>
          {/each}
        </ul>
      </div>

      <div style="display: flex; flex-direction: column; gap: 8px;">
        <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{t('guiaMapa')}</span>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 8px;">
          {#each TELAS as tela (tela.tab)}
            <button
              class="pressable"
              onclick={() => fechar(() => switchTab(tela.tab))}
              style="display: flex; align-items: center; gap: 10px; min-height: 48px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-text); font-family: inherit; font-size: 14px; text-align: left; cursor: pointer;"
            >
              <i class="ph {tela.icone}" style="font-size: 19px; color: var(--color-accent-400); flex-shrink: 0;"></i>
              <span style="flex: 1; min-width: 0;">{t(tela.chave)}</span>
              <span class="font-mono" style="font-size: 12px; color: var(--color-neutral-500);">{tela.n}</span>
            </button>
          {/each}
        </div>
      </div>

      <button
        class="pressable"
        onclick={() => fechar()}
        style="align-self: flex-end; min-height: 48px; padding: 0 24px; border: 1px solid var(--color-accent); border-radius: 8px; background: var(--color-accent); color: var(--color-accent-100); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
      >{t('guiaOk')}</button>
    </div>
  </div>
{/if}
