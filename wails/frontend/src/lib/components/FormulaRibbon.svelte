<script lang="ts">
  // Fita de fórmula proporcional (máquina tintométrica): segmentos na cor de
  // cada ingrediente, largura = % da receita, label mono dentro (código + %),
  // régua com ticks 0 / 50 / 100 embaixo.
  import { contrastOn } from '../ui';

  export interface RibbonSegment {
    r: number;
    g: number;
    b: number;
    code: string;
    percentage: number;
  }

  interface Props {
    segments: RibbonSegment[];
    height?: number; // px
    ruler?: boolean;
    labels?: boolean; // mini fita (Home) desliga os labels
  }

  let { segments, height = 60, ruler = true, labels = true }: Props = $props();

  const ticks = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
</script>

<div class="ribbon-wrap">
  <div class="ribbon" style="height: {height}px;" role="img" aria-label="Fita de fórmula proporcional">
    {#each segments as seg}
      <div
        class="ribbon-seg"
        style="width: {seg.percentage}%; background: rgb({seg.r}, {seg.g}, {seg.b}); color: {contrastOn(seg.r, seg.g, seg.b)};"
      >
        {#if labels && seg.percentage >= 16}
          <span class="ribbon-label font-mono">{seg.code}&nbsp;&nbsp;{Math.round(seg.percentage)}%</span>
        {:else if labels && seg.percentage >= 9}
          <span class="ribbon-label font-mono">{seg.code}</span>
        {/if}
      </div>
    {/each}
  </div>
  {#if ruler}
    <div class="ruler" aria-hidden="true">
      {#each ticks as t}
        <span class="tick" class:major={t % 50 === 0} style="left: {t}%;"></span>
      {/each}
      <span class="tick-label font-mono" style="left: 0%;">0</span>
      <span class="tick-label font-mono" style="left: 50%;">50</span>
      <span class="tick-label font-mono last" style="left: 100%;">100</span>
    </div>
  {/if}
</div>

<style>
  .ribbon-wrap {
    width: 100%;
  }

  .ribbon {
    display: flex;
    width: 100%;
    overflow: hidden;
    /* cor chapada = raio 0 */
    border-radius: 0;
  }

  .ribbon-seg {
    display: flex;
    align-items: flex-start;
    min-width: 0;
    overflow: hidden;
    transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .ribbon-label {
    padding: 10px 12px;
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
  }

  .ruler {
    position: relative;
    height: 22px;
    margin-top: 2px;
    border-top: 1px solid var(--hairline);
  }

  .tick {
    position: absolute;
    top: 0;
    width: 1px;
    height: 6px;
    background: var(--hairline);
    transform: translateX(-0.5px);
  }

  .tick.major {
    height: 9px;
    background: var(--text-3);
  }

  .tick-label {
    position: absolute;
    top: 10px;
    font-size: 10px;
    color: var(--text-3);
    transform: translateX(-50%);
  }

  .tick-label:first-of-type {
    transform: none;
  }

  .tick-label.last {
    transform: translateX(-100%);
  }
</style>
