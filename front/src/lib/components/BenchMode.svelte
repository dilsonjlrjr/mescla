<script lang="ts">
  // Modo Bancada — consulta com as mãos sujas de tinta, a 40cm da tela:
  // fundo PAPEL, números gigantes (Bricolage ~110px), multiplicador ×1 ×2 ×3,
  // Wake Lock (a tela não apaga) e nenhum outro controle além de sair.
  import PaintBottle from './PaintBottle.svelte';
  import { pushLayer } from '../nav.svelte';
  import type { EquivalentRecipe } from '../services/engine';

  interface Props {
    recipe: EquivalentRecipe;
    drops: number[];
    totalDrops: number;
    onClose: () => void;
  }

  let { recipe, drops, totalDrops, onClose }: Props = $props();

  let mult = $state(1);
  let wakeLock: { release: () => Promise<void> } | null = null;
  let closeLayer: (() => void) | null = null;

  $effect(() => {
    // camada de histórico: back sai do modo bancada
    closeLayer = pushLayer(() => {
      closeLayer = null;
      onClose();
    });
    // tela não apaga enquanto o pintor mistura
    (async () => {
      try {
        wakeLock = (await navigator.wakeLock?.request('screen')) ?? null;
      } catch {
        /* sem suporte/permissão: só perde o "tela sempre acesa" */
      }
    })();

    return () => {
      void wakeLock?.release().catch(() => {});
      wakeLock = null;
    };
  });

  function exit() {
    closeLayer?.();
  }
</script>

<div class="bench" role="dialog" aria-modal="true" aria-label="Modo bancada">
  <div class="bench-head">
    <h1 class="bench-title font-display">Bancada</h1>
    <button class="bench-exit pressable" onclick={exit} aria-label="Sair do modo bancada">
      sair <span aria-hidden="true">×</span>
    </button>
  </div>

  <div class="bench-mult">
    {#each [1, 2, 3] as m (m)}
      <button class="bench-mult-btn pressable" class:active={mult === m} onclick={() => (mult = m)}>
        ×{m}
      </button>
    {/each}
    {#if mult > 1}
      <span class="bench-mult-note font-mono">{mult === 2 ? 'lote dobrado' : 'lote triplicado'}</span>
    {/if}
  </div>

  <div class="bench-rows">
    {#each recipe.ingredients as ing, i (ing.paintId)}
      <div class="bench-row">
        <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={92} />
        <div class="bench-main">
          <div class="bench-count">
            <span class="bench-num font-display">{drops[i] * mult}</span>
            <span class="bench-unit">{drops[i] * mult === 1 ? 'gota' : 'gotas'}</span>
          </div>
          <span class="bench-meta font-mono">{ing.name}{ing.code ? ` · ${ing.code}` : ''}</span>
        </div>
      </div>
    {/each}
  </div>

  <p class="bench-foot">
    <span class="font-mono">total {totalDrops * mult} gotas</span>
    A tela fica acesa enquanto você mistura.
  </p>
</div>

<style>
  /* Fundo papel: o modo bancada é uma folha de instrução, não um app. */
  .bench {
    position: fixed;
    inset: 0;
    z-index: 80;
    display: flex;
    flex-direction: column;
    background: var(--papel);
    padding: calc(var(--safe-top) + 16px) 20px calc(var(--safe-bottom) + 16px);
    animation: fade-in 0.15s ease both;
  }

  .bench-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 10px;
  }

  .bench-title {
    font-size: 30px;
    font-weight: 750;
    letter-spacing: -0.015em;
    color: var(--grafite);
  }

  .bench-exit {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 44px;
    padding: 0 8px;
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-500);
  }

  .bench-exit span {
    font-size: 19px;
    line-height: 1;
  }

  .bench-mult {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 8px;
  }

  .bench-mult-btn {
    min-width: 56px;
    height: 48px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--hairline);
    background: var(--papel);
    font-family: var(--font-mono);
    font-size: 17px;
    font-weight: 600;
    color: var(--grafite);
  }

  .bench-mult-btn.active {
    background: var(--grafite);
    border-color: var(--grafite);
    color: var(--papel);
  }

  .bench-mult-note {
    font-size: 12px;
    color: var(--ink-500);
  }

  .bench-rows {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .bench-row {
    display: flex;
    align-items: center;
    gap: 22px;
    padding: 18px 0;
    border-top: 1px solid var(--hairline);
  }

  .bench-row:first-child {
    border-top: none;
  }

  .bench-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .bench-count {
    display: flex;
    align-items: baseline;
    gap: 16px;
  }

  .bench-num {
    font-size: 108px;
    font-weight: 780;
    line-height: 0.95;
    letter-spacing: -0.03em;
    color: var(--grafite);
  }

  .bench-unit {
    font-size: 19px;
    font-weight: 500;
    color: var(--ink-500);
  }

  .bench-meta {
    font-size: 13.5px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bench-foot {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-top: 10px;
    padding-top: 12px;
    border-top: 1px solid var(--hairline);
    font-size: 13px;
    color: var(--ink-500);
  }

  .bench-foot .font-mono {
    font-size: 12px;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
