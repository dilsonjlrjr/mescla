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
  <div class="chip-swatch">
    <span class="chip-color" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
    <span class="chip-bottle-side">
      <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={96} label={paint.code} />
    </span>
    <span class="chip-punch"></span>
  </div>

  <div class="chip-label">
    <div class="chip-name">{paint.name}</div>
    <div class="chip-code">{paint.code ? `${paint.code} · ${paint.manufacturer}` : paint.manufacturer}</div>
    {#if paint.finishType}
      <span class="chip-finish">{paint.finishType}</span>
    {/if}
  </div>
</button>
