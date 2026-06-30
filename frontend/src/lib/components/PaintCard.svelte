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

  $effect(() => {
    // Compute luminance for contrast
  });

  function getLuminance(r: number, g: number, b: number): number {
    return (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  }

  let isDark = $derived(getLuminance(paint.r, paint.g, paint.b) < 0.5);
</script>

<button
  {onclick}
  class="artisan-card text-left transition-all duration-300 group w-full overflow-hidden"
>
  <!-- Swatch area -->
  <div class="relative overflow-hidden" style="border-radius: 14px 14px 0 0;">
    <div class="w-full aspect-[4/3] swatch-shimmer"
      style="background: rgb({paint.r}, {paint.g}, {paint.b});">
      {#if paint.swatchPath}
        <img
          src="file://{paint.swatchPath}"
          alt={paint.name}
          class="w-full h-full object-cover"
          loading="lazy"
        />
      {/if}
    </div>

    <!-- Subtle gradient overlay -->
    <div class="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>

    <!-- Code badge -->
    <div class="absolute top-2.5 left-2.5">
      <span class="font-mono text-[10px] font-medium px-2 py-1 rounded-md backdrop-blur-md"
        style="background: rgba(0,0,0,0.5); color: rgba(255,255,255,0.9); border: 1px solid rgba(255,255,255,0.1);">
        {paint.code}
      </span>
    </div>
  </div>

  <!-- Info -->
  <div class="p-3.5">
    <div class="font-semibold text-[13px] text-white group-hover:text-[var(--color-amber-hot)] transition-colors duration-200 truncate leading-tight mb-1">
      {paint.name}
    </div>
    <div class="text-[11px] truncate mb-1.5" style="color: var(--color-obsidian-400);">{paint.manufacturer}</div>
    {#if paint.finishType}
      <div class="inline-flex items-center gap-1 text-[9px] uppercase tracking-[0.12em] font-semibold px-1.5 py-0.5 rounded"
        style="background: var(--color-obsidian-800); color: var(--color-obsidian-500);">
        {paint.finishType}
      </div>
    {/if}
  </div>
</button>
