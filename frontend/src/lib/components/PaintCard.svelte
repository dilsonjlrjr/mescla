<script lang="ts">
  import PaintBottle from './PaintBottle.svelte';

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    productLine: string;
    r: number;
    g: number;
    b: number;
    swatchPath: string;
    thumbnail: string;
    imageUrl: string;
    finishType: string;
    paintType: string;
  }

  interface Props {
    paint: Paint;
    onclick: () => void;
  }

  let { paint, onclick }: Props = $props();
</script>

<button class="chip" {onclick}>
  <div class="chip-swatch" style="background: color-mix(in srgb, rgb({paint.r}, {paint.g}, {paint.b}) 22%, var(--ink-800));">
    <div class="chip-bottle">
      <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={96} />
    </div>
    <span class="chip-punch"></span>
    <span class="chip-code">{paint.code}</span>
  </div>

  <div class="chip-label">
    <div class="chip-name">{paint.name}</div>
    <div class="chip-mfr">{paint.manufacturer}</div>
    {#if paint.finishType}
      <span class="chip-finish">{paint.finishType}</span>
    {/if}
  </div>
</button>
