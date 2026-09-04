<script lang="ts">
  interface Props {
    visible: boolean;
    x: number;
    y: number;
    screenX: number;
    screenY: number;
    hex: string;
    zoom: number;
  }

  let { visible, x, y, screenX, screenY, hex, zoom }: Props = $props();

  const LOUPE_RADIUS = 48;
  const MAGNIFICATION = 8;
</script>

{#if visible}
  <div
    class="loupe-container"
    style:left="{screenX}px"
    style:top="{screenY}px"
  >
    <div class="loupe">
      <div class="loupe-inner">
        <canvas
          class="loupe-canvas"
          width={LOUPE_RADIUS * 2}
          height={LOUPE_RADIUS * 2}
        ></canvas>
        <div class="crosshair-h"></div>
        <div class="crosshair-v"></div>
        <div class="center-dot"></div>
      </div>
      <div class="loupe-info">
        <span class="loupe-hex">{hex}</span>
        <span class="loupe-coord">({x}, {y})</span>
      </div>
    </div>
  </div>
{/if}

<style>
  .loupe-container {
    position: fixed;
    pointer-events: none;
    z-index: 100;
    transform: translate(-50%, -120%);
  }

  .loupe {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .loupe-inner {
    position: relative;
    width: 96px;
    height: 96px;
    border-radius: 50%;
    border: 2px solid var(--grafite);
    overflow: hidden;
    background: var(--papel);
  }

  .loupe-canvas {
    width: 100%;
    height: 100%;
  }

  .crosshair-h, .crosshair-v {
    position: absolute;
    background: var(--grafite);
    opacity: 0.4;
  }

  .crosshair-h {
    top: 50%;
    left: 0;
    right: 0;
    height: 1px;
  }

  .crosshair-v {
    left: 50%;
    top: 0;
    bottom: 0;
    width: 1px;
  }

  .center-dot {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--laca);
    transform: translate(-50%, -50%);
  }

  .loupe-info {
    display: flex;
    gap: 6px;
    padding: 2px 8px;
    background: var(--grafite);
    color: var(--papel);
    border-radius: 6px;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 11px;
  }

  .loupe-coord {
    opacity: 0.6;
  }
</style>
