<script lang="ts">
  // Bottom sheet mobile: sobe do rodapé, fecha por swipe-down, tap no scrim
  // ou back do Android (via nav.pushLayer — cada sheet aberto vira uma entrada
  // de histórico que o back consome).
  import type { Snippet } from 'svelte';
  import { pushLayer } from '../nav.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    /** 'auto' cresce com o conteúdo (máx 85%); 'tall' ocupa 85% fixo (listas). */
    height?: 'auto' | 'tall';
    title?: string;
    children: Snippet;
  }

  let { open, onClose, height = 'auto', title = '', children }: Props = $props();

  let panel: HTMLDivElement | undefined = $state();
  let dragY = $state(0);
  let dragging = $state(false);
  let startY = 0;
  let closeLayer: (() => void) | null = null;

  // Integração com o back: abrir empilha a camada; o popstate chama onClose.
  $effect(() => {
    if (open && !closeLayer) {
      closeLayer = pushLayer(() => {
        closeLayer = null;
        onClose();
      });
    } else if (!open && closeLayer) {
      // Fechamento programático (scrim/swipe/botão): consome a entrada.
      const c = closeLayer;
      closeLayer = null;
      c();
    }
  });

  function pointerDown(e: PointerEvent) {
    // Só inicia o arrasto a partir da alça/cabeçalho, ou quando o conteúdo
    // está no topo do scroll — senão rouba o scroll interno da lista.
    dragging = true;
    startY = e.clientY;
    dragY = 0;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function pointerMove(e: PointerEvent) {
    if (!dragging) return;
    dragY = Math.max(0, e.clientY - startY);
  }

  function pointerUp() {
    if (!dragging) return;
    dragging = false;
    if (dragY > 90) {
      onClose();
    }
    dragY = 0;
  }
</script>

{#if open}
  <div class="sheet-scrim" onclick={onClose} aria-hidden="true"></div>
  <div
    class="sheet-panel"
    class:tall={height === 'tall'}
    bind:this={panel}
    style={dragY ? `transform: translateY(${dragY}px); transition: none;` : ''}
    role="dialog"
    aria-modal="true"
    aria-label={title || 'Painel'}
  >
    <div
      class="sheet-grab"
      role="presentation"
      onpointerdown={pointerDown}
      onpointermove={pointerMove}
      onpointerup={pointerUp}
      onpointercancel={pointerUp}
    >
      <span class="sheet-handle"></span>
      {#if title}
        <h2 class="sheet-title font-display">{title}</h2>
      {/if}
    </div>
    <div class="sheet-body">
      {@render children()}
    </div>
  </div>
{/if}

<style>
  .sheet-scrim {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: rgba(26, 23, 18, 0.42);
    animation: fade-in 0.2s ease both;
  }

  .sheet-panel {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 61;
    display: flex;
    flex-direction: column;
    max-height: 85dvh;
    background: var(--ink-950);
    border-radius: 18px 18px 0 0;
    box-shadow: 0 -8px 40px rgba(0, 0, 0, 0.25);
    animation: sheet-up 0.28s cubic-bezier(0.32, 0.72, 0, 1) both;
    transition: transform 0.2s ease;
    padding-bottom: var(--safe-bottom);
  }

  .sheet-panel.tall {
    height: 85dvh;
  }

  .sheet-grab {
    flex-shrink: 0;
    padding: 10px 20px 6px;
    touch-action: none;
    cursor: grab;
  }

  .sheet-handle {
    display: block;
    width: 40px;
    height: 4px;
    border-radius: 999px;
    background: var(--ink-600);
    margin: 0 auto 10px;
  }

  .sheet-title {
    font-size: 19px;
    font-weight: 600;
    color: var(--ink-100);
    padding-bottom: 6px;
  }

  .sheet-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 0 20px 20px;
  }

  @keyframes sheet-up {
    from { transform: translateY(100%); }
    to { transform: translateY(0); }
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
