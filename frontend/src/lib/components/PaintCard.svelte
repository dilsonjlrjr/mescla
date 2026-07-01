<script lang="ts">
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
  <div class="chip-swatch" style="background: rgb({paint.r}, {paint.g}, {paint.b});">
    {#if paint.swatchPath}
      <img
        src="file://{paint.swatchPath}"
        alt={paint.name}
        loading="lazy"
        onerror={(e) => (e.currentTarget as HTMLImageElement).style.display = 'none'}
      />
    {/if}
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
