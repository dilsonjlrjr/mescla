<script lang="ts">
  // Fita de fórmula proporcional (máquina tintométrica de loja): segmentos na
  // cor de cada ingrediente, largura = % da receita; embaixo, régua com ticks
  // e marcas mono 0 / 50 / 100.
  import { contrastOn } from '../ui';

  interface Segment {
    r: number;
    g: number;
    b: number;
    percentage: number;
    code?: string;
  }

  interface Props {
    segments: Segment[];
    /** altura da fita em px */
    height?: number;
    /** mostra o rótulo mono dentro dos segmentos largos */
    labels?: boolean;
  }

  let { segments, height = 48, labels = true }: Props = $props();

  let total = $derived(segments.reduce((a, s) => a + s.percentage, 0) || 1);
</script>

<div class="ribbon-wrap">
  <div class="ribbon" style="height: {height}px;" role="img" aria-label="Fita de fórmula proporcional">
    {#each segments as s, i (i)}
      {@const pct = (s.percentage / total) * 100}
      <div
        class="ribbon-seg"
        style="width: {pct}%; background: rgb({s.r}, {s.g}, {s.b}); color: {contrastOn(s.r, s.g, s.b)};"
      >
        {#if labels && pct >= 18}
          <span class="ribbon-label font-mono">
            {#if s.code && pct >= 34}{s.code}&nbsp;&nbsp;{/if}{Math.round(s.percentage)}%
          </span>
        {/if}
      </div>
    {/each}
  </div>
  <div class="ruler" aria-hidden="true">
    {#each Array(11) as _, i}
      <span class="tick" class:major={i % 5 === 0}></span>
    {/each}
  </div>
  <div class="ruler-marks font-mono" aria-hidden="true">
    <span>0</span>
    <span>50</span>
    <span>100</span>
  </div>
</div>

<style>
  .ribbon-wrap {
    width: 100%;
  }

  /* Cor chapada: raio 0 — a fita é instrumento, não card. */
  .ribbon {
    display: flex;
    width: 100%;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
  }

  .ribbon-seg {
    position: relative;
    display: flex;
    align-items: center;
    min-width: 2px;
    overflow: hidden;
  }

  .ribbon-seg + .ribbon-seg {
    box-shadow: inset 1px 0 0 rgba(255, 255, 255, 0.35);
  }

  .ribbon-label {
    padding-left: 10px;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.02em;
    white-space: nowrap;
  }

  .ruler {
    display: flex;
    justify-content: space-between;
    margin-top: 5px;
    padding: 0 0.5px;
  }

  .tick {
    width: 1px;
    height: 5px;
    background: var(--ink-500);
    opacity: 0.55;
  }

  .tick.major {
    height: 8px;
    opacity: 1;
  }

  .ruler-marks {
    display: flex;
    justify-content: space-between;
    margin-top: 2px;
    font-size: 10px;
    color: var(--ink-500);
  }
</style>
