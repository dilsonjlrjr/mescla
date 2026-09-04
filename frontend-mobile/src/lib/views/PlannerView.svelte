<script lang="ts">
  import Icon from '../components/Icon.svelte';
  import BottomSheet from '../components/BottomSheet.svelte';
  import { findSimilar, type SearchResult } from '../services/engine';
  import { toast } from '../toast.svelte';

  let canvasEl: HTMLCanvasElement | undefined = $state();
  let fileInputEl: HTMLInputElement | undefined = $state();
  let image: HTMLImageElement | null = $state(null);
  let hasImage = $state(false);
  let imageDataUrl = $state('');

  let regions: PlannerRegion[] = $state([]);
  let nextId = $state(1);
  let selectedRegion: PlannerRegion | null = $state(null);
  let sheetOpen = $state(false);
  let searching = $state(false);

  let zoom = $state(1);
  let panX = $state(0);
  let panY = $state(0);
  let touchStartDist = $state(0);
  let touchStartZoom = $state(1);
  let touchStartPanX = $state(0);
  let touchStartPanY = $state(0);

  interface PlannerRegion {
    id: number;
    x: number; y: number;
    r: number; g: number; b: number;
    hex: string;
    regionName: string;
    matches: SearchResult[];
    selectedMatch: SearchResult | null;
  }

  function draw() {
    const c = canvasEl;
    if (!c || !image) return;
    const ctx = c.getContext('2d');
    if (!ctx) return;

    c.width = image.width;
    c.height = image.height;
    ctx.clearRect(0, 0, c.width, c.height);
    ctx.save();
    ctx.translate(panX, panY);
    ctx.scale(zoom, zoom);
    ctx.drawImage(image, 0, 0);
    ctx.restore();

    for (const r of regions) {
      const sx = r.x * zoom + panX;
      const sy = r.y * zoom + panY;
      const R = selectedRegion?.id === r.id ? 16 : 12;

      ctx.save();
      ctx.translate(sx, sy);
      ctx.beginPath();
      ctx.arc(0, 0, R, 0, Math.PI * 2);
      ctx.fillStyle = r.hex;
      ctx.fill();
      ctx.strokeStyle = '#fff';
      ctx.lineWidth = 2;
      ctx.stroke();

      if (selectedRegion?.id === r.id) {
        ctx.beginPath();
        ctx.arc(0, 0, R + 3, 0, Math.PI * 2);
        ctx.strokeStyle = 'var(--laca)';
        ctx.lineWidth = 2;
        ctx.setLineDash([3, 3]);
        ctx.stroke();
        ctx.setLineDash([]);
      }

      ctx.restore();
    }
  }

  $effect(() => {
    if (image && canvasEl) {
      requestAnimationFrame(() => {
        if (!canvasEl || !image) return;
        const cw = canvasEl.clientWidth;
        const ch = canvasEl.clientHeight;
        zoom = Math.min(cw / image.width, ch / image.height, 1);
        panX = (cw - image.width * zoom) / 2;
        panY = (ch - image.height * zoom) / 2;
        draw();
      });
    }
  });

  function loadImage(src: string) {
    const img = new Image();
    img.onload = () => {
      image = img;
      hasImage = true;
      imageDataUrl = src;
      regions = [];
      nextId = 1;
      selectedRegion = null;
    };
    img.src = src;
  }

  function onFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => loadImage(reader.result as string);
    reader.readAsDataURL(file);
  }

  function openCamera() {
    fileInputEl?.click();
  }

  function getTouchDist(t: TouchList): number {
    if (t.length < 2) return 0;
    return Math.sqrt((t[0].clientX - t[1].clientX) ** 2 + (t[0].clientY - t[1].clientY) ** 2);
  }

  function canvasPos(clientX: number, clientY: number) {
    if (!canvasEl) return { mx: 0, my: 0 };
    const rect = canvasEl.getBoundingClientRect();
    const sx = canvasEl.width / rect.width;
    const sy = canvasEl.height / rect.height;
    return { mx: (clientX - rect.left) * sx, my: (clientY - rect.top) * sy };
  }

  function hitTest(cx: number, cy: number): PlannerRegion | null {
    let best: PlannerRegion | null = null;
    let bestDist = 20;
    for (const r of regions) {
      const sx = r.x * zoom + panX;
      const sy = r.y * zoom + panY;
      const d = Math.sqrt((cx - sx) ** 2 + (cy - sy) ** 2);
      if (d < bestDist) { bestDist = d; best = r; }
    }
    return best;
  }

  function onTouchStart(e: TouchEvent) {
    if (!canvasEl || !image) return;
    e.preventDefault();

    if (e.touches.length === 2) {
      touchStartDist = getTouchDist(e.touches);
      touchStartZoom = zoom;
      touchStartPanX = panX;
      touchStartPanY = panY;
      return;
    }

    if (e.touches.length === 1) {
      const t = e.touches[0];
      const { mx, my } = canvasPos(t.clientX, t.clientY);

      const hit = hitTest(mx, my);
      if (hit) {
        selectedRegion = hit;
        sheetOpen = true;
        draw();
        return;
      }

      const ix = Math.round((mx - panX) / zoom);
      const iy = Math.round((my - panY) / zoom);
      if (ix >= 0 && iy >= 0 && ix < image.width && iy < image.height) {
        pickColor(ix, iy);
      }
    }
  }

  function onTouchMove(e: TouchEvent) {
    if (!canvasEl || !image) return;
    e.preventDefault();

    if (e.touches.length === 2) {
      const dist = getTouchDist(e.touches);
      zoom = Math.max(0.25, Math.min(5, touchStartZoom * (dist / touchStartDist)));
      draw();
    }
  }

  async function pickColor(ix: number, iy: number) {
    if (!image) return;

    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(ix, iy, 1, 1).data);
    const hex = `#${r.toString(16).padStart(2,'0')}${g.toString(16).padStart(2,'0')}${b.toString(16).padStart(2,'0')}`.toUpperCase();

    const region: PlannerRegion = {
      id: nextId++,
      x: ix, y: iy,
      r, g, b,
      hex,
      regionName: '',
      matches: [],
      selectedMatch: null,
    };

    regions = [...regions, region];
    selectedRegion = region;
    draw();

    searching = true;
    try {
      const matches = await findSimilar(r, g, b, 50, 5);
      region.matches = matches;
      region.selectedMatch = matches[0] || null;
      sheetOpen = true;
      draw();
    } catch (e) {
      console.error('findSimilar:', e);
      toast('Erro ao buscar cor');
    }
    searching = false;
  }

  function removeRegion(id: number) {
    regions = regions.filter(r => r.id !== id);
    if (selectedRegion?.id === id) {
      selectedRegion = null;
      sheetOpen = false;
    }
    draw();
  }

  function exportJSON() {
    if (regions.length === 0) return;
    const data = {
      version: 2,
      name: 'Plano Mobile',
      imageData: imageDataUrl,
      regions: regions.map(r => ({
        x: r.x, y: r.y,
        r: r.r, g: r.g, b: r.b,
        hex: r.hex,
        regionName: r.regionName,
        match: r.selectedMatch ? {
          paintId: r.selectedMatch.paintId,
          name: r.selectedMatch.name,
          manufacturer: r.selectedMatch.manufacturer,
          code: r.selectedMatch.code,
          hex: r.hex,
          deltaE: r.selectedMatch.deltaE,
        } : null,
      })),
      exportedAt: new Date().toISOString(),
    };
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `mescla-plano-${new Date().toISOString().slice(0,10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
    toast('Exportado!');
  }
</script>

<div class="planner-view">
  {#if !hasImage}
    <div class="empty-state">
      <Icon name="pipette" size={48} />
      <h3>Planejador de Pintura</h3>
      <p>Carregue uma foto da miniatura para identificar cores.</p>
      <button class="cta-btn" onclick={openCamera}>
        <Icon name="upload" size={18} /> Carregar imagem
      </button>
      <input type="file" accept="image/*" capture="environment" class="hidden" bind:this={fileInputEl} onchange={onFileInput} />
    </div>
  {:else}
    <div class="canvas-area">
      <canvas
        bind:this={canvasEl}
        ontouchstart={onTouchStart}
        ontouchmove={onTouchMove}
      ></canvas>

      {#if searching}
        <div class="searching-indicator">
          <span class="spinner"></span>
          Buscando cor...
        </div>
      {/if}
    </div>

    <div class="bottom-bar">
      <button class="bar-btn" onclick={() => { regions = []; draw(); }}>
        <Icon name="trash" size={18} />
      </button>
      <span class="bar-count">{regions.length} regiões</span>
      <button class="bar-btn" onclick={exportJSON} disabled={regions.length === 0}>
        <Icon name="download" size={18} />
      </button>
    </div>
  {/if}
</div>

{#if selectedRegion}
  <BottomSheet open={sheetOpen} onClose={() => { sheetOpen = false; }}>
    <div class="region-sheet">
      <div class="sheet-header">
        <span class="sheet-swatch" style:background={selectedRegion.hex}></span>
        <div>
          <strong>#{selectedRegion.id}</strong>
          <span class="sheet-hex">{selectedRegion.hex}</span>
        </div>
        <button class="sheet-close" onclick={() => { sheetOpen = false; }}>
          <Icon name="close" size={18} />
        </button>
      </div>

      <input
        type="text"
        class="field"
        placeholder="Nome da região"
        bind:value={selectedRegion.regionName}
      />

      {#if selectedRegion.matches.length > 0}
        <div class="matches-list">
          {#each selectedRegion.matches as m}
            <button
              class="match-item"
              class:selected={selectedRegion.selectedMatch?.paintId === m.paintId}
              onclick={() => { selectedRegion!.selectedMatch = m; draw(); }}
            >
              <span class="match-swatch" style:background={`rgb(${m.r},${m.g},${m.b})`}></span>
              <div class="match-info">
                <span class="match-name">{m.manufacturer} {m.name}</span>
                <span class="match-code">{m.code}</span>
              </div>
              <span class="match-delta">ΔE {m.deltaE.toFixed(1)}</span>
            </button>
          {/each}
        </div>
      {:else}
        <p class="no-matches">Nenhuma tinta similar encontrada.</p>
      {/if}

      <button class="danger-btn" onclick={() => removeRegion(selectedRegion!.id)}>
        <Icon name="trash" size={14} /> Remover região
      </button>
    </div>
  </BottomSheet>
{/if}

<style>
  .planner-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bancada);
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 32px;
    text-align: center;
    color: var(--grafite);
  }

  .empty-state h3 { margin: 0; font-size: 18px; }
  .empty-state p { margin: 0; opacity: 0.6; font-size: 14px; }

  .cta-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 12px 24px;
    background: var(--laca);
    color: #fff;
    border: none;
    border-radius: var(--radius-surface);
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
  }

  .canvas-area {
    flex: 1;
    position: relative;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    touch-action: none;
  }

  canvas {
    max-width: 100%;
    max-height: 100%;
    touch-action: none;
  }

  .searching-indicator {
    position: absolute;
    top: 12px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 14px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: 20px;
    font-size: 13px;
    color: var(--grafite);
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--hairline);
    border-top-color: var(--laca);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .bottom-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--papel);
    border-top: 1px solid var(--hairline);
  }

  .bar-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    color: var(--grafite);
  }

  .bar-btn:hover { background: var(--bancada); }
  .bar-btn:disabled { opacity: 0.3; }

  .bar-count {
    flex: 1;
    text-align: center;
    font-size: 13px;
    color: var(--grafite);
    opacity: 0.6;
  }

  .hidden { display: none; }

  /* Sheet */
  .region-sheet {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
  }

  .sheet-header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .sheet-swatch {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: 2px solid var(--hairline);
  }

  .sheet-hex {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 13px;
    color: var(--grafite);
    margin-left: 6px;
  }

  .sheet-close {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--grafite);
  }

  .field {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    font-size: 14px;
    font-family: inherit;
    box-sizing: border-box;
  }

  .field:focus { outline: none; border-color: var(--laca); }

  .matches-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .match-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    background: var(--papel);
    cursor: pointer;
    text-align: left;
    width: 100%;
  }

  .match-item:hover { background: var(--bancada); }
  .match-item.selected { border-color: var(--laca); border-width: 2px; }

  .match-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 1px solid var(--hairline);
    flex-shrink: 0;
  }

  .match-info {
    display: flex;
    flex-direction: column;
    flex: 1;
  }

  .match-name { font-size: 13px; font-weight: 600; }
  .match-code { font-size: 11px; color: var(--grafite); opacity: 0.6; }

  .match-delta {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 12px;
    color: var(--grafite);
    background: var(--bancada);
    padding: 2px 8px;
    border-radius: 4px;
  }

  .no-matches {
    text-align: center;
    font-size: 13px;
    color: var(--grafite);
    opacity: 0.5;
    padding: 16px;
  }

  .danger-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 10px;
    border: 1px solid var(--laca);
    border-radius: var(--radius-surface);
    background: transparent;
    color: var(--laca);
    font-size: 13px;
    cursor: pointer;
    width: 100%;
  }
</style>
