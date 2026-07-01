<script lang="ts">
  import Card, { Content, PrimaryAction } from '@smui/card';

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

  function getLuminance(r: number, g: number, b: number): number {
    return (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  }

  let isDark = $derived(getLuminance(paint.r, paint.g, paint.b) < 0.5);
</script>

<Card variant="outlined" class="paint-card" style="overflow: hidden;">
  <PrimaryAction {onclick}>
    <!-- Swatch area -->
    <div class="swatch-area swatch-shimmer" style="background: rgb({paint.r}, {paint.g}, {paint.b});">
      {#if paint.swatchPath}
        <img
          src="file://{paint.swatchPath}"
          alt={paint.name}
          class="swatch-img"
          loading="lazy"
        />
      {/if}
      <div class="swatch-gradient"></div>
      <div class="code-badge">
        <span>{paint.code}</span>
      </div>
    </div>

    <!-- Info -->
    <Content>
      <div class="paint-name">{paint.name}</div>
      <div class="paint-mfr">{paint.manufacturer}</div>
      {#if paint.finishType}
        <div class="finish-tag">{paint.finishType}</div>
      {/if}
    </Content>
  </PrimaryAction>
</Card>

<style>
  :global(.smui-card--outlined.paint-card) {
    background: linear-gradient(145deg, rgba(255,255,255,0.035), rgba(255,255,255,0.01));
    border-color: rgba(255, 255, 255, 0.06);
    border-radius: 14px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    overflow: hidden;
  }

  :global(.smui-card--outlined.paint-card:hover) {
    background: linear-gradient(145deg, rgba(255,255,255,0.06), rgba(255,255,255,0.025));
    border-color: rgba(212, 160, 83, 0.15);
    box-shadow: 0 8px 32px rgba(0,0,0,0.4), 0 0 0 1px rgba(212,160,83,0.08);
    transform: translateY(-1px);
  }

  .swatch-area {
    position: relative;
    width: 100%;
    aspect-ratio: 4 / 3;
    overflow: hidden;
    border-radius: 14px 14px 0 0;
  }

  .swatch-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .swatch-gradient {
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(0,0,0,0.4), transparent, transparent);
    opacity: 0;
    transition: opacity 0.3s;
  }

  :global(.paint-card:hover) .swatch-gradient {
    opacity: 1;
  }

  .code-badge {
    position: absolute;
    top: 10px;
    left: 10px;
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 500;
    padding: 3px 8px;
    border-radius: 6px;
    backdrop-filter: blur(12px);
    background: rgba(0, 0, 0, 0.5);
    color: rgba(255, 255, 255, 0.9);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .paint-name {
    font-weight: 600;
    font-size: 13px;
    color: white;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.3;
    margin-bottom: 4px;
    transition: color 0.2s;
  }

  :global(.paint-card:hover) .paint-name {
    color: var(--color-amber-hot);
  }

  .paint-mfr {
    font-size: 11px;
    color: var(--color-obsidian-400);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin-bottom: 6px;
  }

  .finish-tag {
    display: inline-block;
    font-size: 9px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--color-obsidian-800);
    color: var(--color-obsidian-500);
  }
</style>
