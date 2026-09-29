<script lang="ts">
  // rf-24 — folha modal do círculo (guia e receita de mistura). Entra como
  // camada de navegação (`pushLayer`): o "voltar" do Android a fecha. Quem a
  // monta troca `{#if}`; se sair sem o voltar, a camada é devolvida ao sair.
  // A entrada é CSS puro, então `prefers-reduced-motion` (app.css) a desliga.
  import { onDestroy, onMount, type Snippet } from 'svelte';
  import { pushLayer } from '../nav.svelte';
  import { t } from '../i18n.svelte';

  interface Props {
    titulo: string;
    /** roda quando a folha fecha (botão, scrim, Esc ou voltar) */
    onfechar: () => void;
    children: Snippet<[() => void]>;
  }

  let { titulo, onfechar, children }: Props = $props();

  let emPilha = false;
  let fecharLayer: (() => void) | null = null;
  let origem: HTMLElement | null = null;
  let folhaEl: HTMLDivElement | undefined = $state();

  function fechar() {
    if (emPilha && fecharLayer) fecharLayer();
    else onfechar();
  }

  onMount(() => {
    origem = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    fecharLayer = pushLayer(() => {
      emPilha = false;
      onfechar();
    });
    emPilha = true;
    folhaEl?.focus();
  });

  onDestroy(() => {
    if (emPilha && fecharLayer) {
      emPilha = false;
      fecharLayer();
    }
    if (origem?.isConnected) origem.focus();
  });

  function aoTeclar(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      fechar();
    } else if (e.key === 'Tab' && folhaEl) {
      const focaveis = [...folhaEl.querySelectorAll<HTMLElement>('button, select, input')].filter(el => !el.hasAttribute('disabled'));
      if (focaveis.length === 0) return;
      const primeiro = focaveis[0];
      const ultimo = focaveis[focaveis.length - 1];
      if (e.shiftKey && (document.activeElement === primeiro || document.activeElement === folhaEl)) {
        e.preventDefault();
        ultimo.focus();
      } else if (!e.shiftKey && document.activeElement === ultimo) {
        e.preventDefault();
        primeiro.focus();
      }
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
<div class="fc-scrim" onclick={(e) => { if (e.target === e.currentTarget) fechar(); }}>
  <div
    bind:this={folhaEl}
    class="fc-folha"
    role="dialog"
    aria-modal="true"
    aria-label={titulo}
    tabindex="-1"
    onkeydown={aoTeclar}
  >
    <div class="fc-cab">
      <h2>{titulo}</h2>
      <button class="pressable fc-x" onclick={fechar} aria-label={t('ariaClose')}>
        <i class="ph ph-x" style="font-size: 18px;"></i>
      </button>
    </div>
    <div class="fc-corpo">{@render children(fechar)}</div>
  </div>
</div>

<style>
  .fc-scrim {
    position: fixed;
    inset: 0;
    z-index: 150;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    background: rgba(9, 10, 16, 0.7);
  }

  .fc-folha {
    width: min(640px, 100%);
    max-height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--color-neutral-800);
    border-radius: 14px;
    background: var(--color-modal);
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);
    color: var(--color-text);
    font-family: var(--font-body);
    user-select: none;
    animation: fc-entra 240ms cubic-bezier(0.22, 1, 0.36, 1);
  }

  .fc-folha:focus {
    outline: none;
  }

  .fc-cab {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 14px 14px 20px;
    border-bottom: 1px solid var(--color-line);
  }

  .fc-cab h2 {
    margin: 0;
    flex: 1;
    min-width: 0;
    font-size: 19px;
    font-weight: 500;
    letter-spacing: -0.01em;
  }

  .fc-x {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    flex-shrink: 0;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-400);
    cursor: pointer;
  }

  .fc-corpo {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 18px 20px 22px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  @keyframes fc-entra {
    from {
      opacity: 0;
      transform: translateY(14px);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }
</style>
