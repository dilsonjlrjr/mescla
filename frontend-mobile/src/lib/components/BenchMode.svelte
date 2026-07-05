<script lang="ts">
  // Modo Bancada — consulta com as mãos sujas de tinta, a 40cm da tela:
  // gotas gigantes (Plex Mono 48-56px), multiplicador ×1 ×2 ×3 com alvos de
  // 64px, Wake Lock (a tela não apaga) e nenhum outro controle além de sair.
  import Icon from './Icon.svelte';
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
    <div class="bench-title">
      <span class="color-pair" style="width: 34px; height: 22px;">
        <span style="background: rgb({recipe.sourceR}, {recipe.sourceG}, {recipe.sourceB});"></span>
        <span style="background: rgb({recipe.resultR}, {recipe.resultG}, {recipe.resultB});"></span>
      </span>
      <span>{recipe.sourceName}</span>
    </div>
    <button class="bench-exit pressable" onclick={exit} aria-label="Sair do modo bancada">
      <Icon name="close" size={22} />
    </button>
  </div>

  <div class="bench-rows">
    {#each recipe.ingredients as ing, i (ing.paintId)}
      <div class="bench-row">
        <span class="bench-swatch" style="background: rgb({ing.r}, {ing.g}, {ing.b});"></span>
        <div class="bench-text">
          <span class="bench-name">{ing.name}</span>
          {#if ing.code}<span class="bench-code font-mono">{ing.code}</span>{/if}
        </div>
        <div class="bench-drops font-mono">
          {drops[i] * mult}
          <span>{drops[i] * mult === 1 ? 'gota' : 'gotas'}</span>
        </div>
      </div>
    {/each}
  </div>

  <div class="bench-foot">
    <p class="bench-total font-mono">total {totalDrops * mult} gotas</p>
    <div class="bench-mult">
      {#each [1, 2, 3] as m}
        <button class="bench-mult-btn pressable" class:active={mult === m} onclick={() => (mult = m)}>
          ×{m}
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .bench {
    position: fixed;
    inset: 0;
    z-index: 80;
    display: flex;
    flex-direction: column;
    background: var(--ink-950);
    padding: calc(var(--safe-top) + 16px) 20px calc(var(--safe-bottom) + 20px);
    animation: fade-in 0.15s ease both;
  }

  .bench-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 18px;
  }

  .bench-title {
    display: flex;
    align-items: center;
    gap: 12px;
    font-family: var(--font-display);
    font-size: 20px;
    font-weight: 600;
    color: var(--ink-100);
    min-width: 0;
  }

  .bench-title span:last-child {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bench-exit {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    border-radius: 14px;
    border: 1px solid var(--ink-700);
    background: var(--ink-900);
    color: var(--ink-300);
    flex-shrink: 0;
  }

  .bench-rows {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .bench-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 18px;
    border-radius: 16px;
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
  }

  .bench-swatch {
    width: 56px;
    height: 56px;
    border-radius: 12px;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink-100) 15%, transparent);
    flex-shrink: 0;
  }

  .bench-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .bench-name {
    font-size: 18px;
    font-weight: 600;
    color: var(--ink-100);
    line-height: 1.25;
  }

  .bench-code {
    font-size: 14px;
    color: var(--lacquer-tint);
  }

  .bench-drops {
    font-size: 52px;
    font-weight: 600;
    color: var(--ink-100);
    line-height: 1;
    text-align: right;
    flex-shrink: 0;
  }

  .bench-drops span {
    display: block;
    font-size: 13px;
    font-weight: 500;
    color: var(--ink-500);
    margin-top: 2px;
  }

  .bench-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    margin-top: 18px;
  }

  .bench-total {
    font-size: 15px;
    color: var(--ink-500);
  }

  .bench-mult {
    display: flex;
    gap: 10px;
  }

  .bench-mult-btn {
    width: 64px;
    height: 64px;
    border-radius: 16px;
    border: 1px solid var(--ink-700);
    background: var(--ink-900);
    font-family: var(--font-mono);
    font-size: 20px;
    font-weight: 600;
    color: var(--ink-300);
  }

  .bench-mult-btn.active {
    background: var(--lacquer);
    border-color: var(--lacquer);
    color: white;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
