<script lang="ts" generics="T">
  // Lista virtualizada de altura fixa por linha — obrigatória pro catálogo
  // (11.932 tintas): renderiza só o viewport + overscan. O consumidor agrupa
  // itens em "linhas" (ex.: pares pro grid de 2 colunas) e desenha via snippet.
  import type { Snippet } from 'svelte';

  interface Props {
    rows: T[];
    rowHeight: number;
    overscan?: number;
    row: Snippet<[T, number]>;
  }

  let { rows, rowHeight, overscan = 6, row }: Props = $props();

  let container: HTMLDivElement | undefined = $state();
  let scrollTop = $state(0);
  let viewportH = $state(0);

  let first = $derived(Math.max(0, Math.floor(scrollTop / rowHeight) - overscan));
  let last = $derived(
    Math.min(rows.length, Math.ceil((scrollTop + viewportH) / rowHeight) + overscan)
  );
  let visible = $derived(rows.slice(first, last));

  function onScroll() {
    if (container) scrollTop = container.scrollTop;
  }

  $effect(() => {
    if (!container) return;
    const ro = new ResizeObserver(() => {
      viewportH = container!.clientHeight;
    });
    ro.observe(container);
    viewportH = container.clientHeight;
    return () => ro.disconnect();
  });

  // Filtro/busca mudou o conjunto: volta pro topo.
  $effect(() => {
    void rows;
    if (container && container.scrollTop > 0) {
      container.scrollTop = 0;
      scrollTop = 0;
    }
  });
</script>

<div class="vlist" bind:this={container} onscroll={onScroll}>
  <div class="vlist-spacer" style="height: {rows.length * rowHeight}px;">
    <div class="vlist-window" style="transform: translateY({first * rowHeight}px);">
      {#each visible as r, i (first + i)}
        <div style="height: {rowHeight}px;">
          {@render row(r, first + i)}
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  .vlist {
    height: 100%;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .vlist-spacer {
    position: relative;
  }

  .vlist-window {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }
</style>
