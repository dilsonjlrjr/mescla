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

<button
  {onclick}
  class="glass glass-hover rounded-xl p-4 text-left transition-all duration-200 group w-full"
>
  <!-- Swatch -->
  <div class="w-full aspect-square rounded-lg mb-3 relative overflow-hidden shadow-lg"
    style="background: rgb({paint.r}, {paint.g}, {paint.b}); border: 1px solid rgba(255,255,255,0.08);">
    <div class="absolute inset-0 bg-gradient-to-br from-white/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
    {#if paint.swatchPath}
      <img
        src="file://{paint.swatchPath}"
        alt={paint.name}
        class="w-full h-full object-cover"
        loading="lazy"
      />
    {/if}
  </div>

  <!-- Info -->
  <div class="space-y-1">
    <div class="flex items-center gap-2">
      <span class="text-xs font-mono px-1.5 py-0.5 rounded" style="background: var(--color-surface-800); color: var(--color-accent-400);">
        {paint.code}
      </span>
    </div>
    <div class="font-semibold text-sm text-white group-hover:text-accent-300 transition-colors truncate">{paint.name}</div>
    <div class="text-xs truncate" style="color: var(--color-surface-500);">{paint.manufacturer}</div>
    {#if paint.finishType}
      <div class="text-[10px] uppercase tracking-wider font-medium" style="color: var(--color-surface-600);">{paint.finishType}</div>
    {/if}
  </div>
</button>
