<script lang="ts">
  // T2 — Plano da peça (rf-04). Reescrita para fidelidade ao protótipo
  // (D-001/Epic D) — oráculo tests/fixtures/mockup/t2-plano.html.
  //
  // Foto com pins numerados por região (ainda desenhados em <canvas>, com
  // zoom por pinça e pan — não dá pra trocar para pins DOM absolutos do
  // protótipo sem reescrever a matemática de hit-test/zoom, que não está na
  // lista de "preservar toda a lógica" do rito; ver relato de entrega).
  // CARD INTEIRO da região é selecionável (CA13). Painel direito: seletor de
  // fabricante ("Pintar com o que eu tenho") que recalcula TODAS as regiões
  // (CA14), painel de equivalência da região selecionada, lista de regiões
  // com estado pintada/a pintar (toggle no próprio card, como no protótipo)
  // e "Tintas da peça" — checklist local de compra dos ingredientes usados
  // pelas regiões (US-15/T3 herda a receita salva no rodapé).
  import Header from '../components/Header.svelte';
  import Spinner from '../components/Spinner.svelte';
  import {
    fitView, toImage, toScreen, zoomAround, zoomPercent, type View,
  } from '../planner/viewport';
  import { suggestRecipeForColor, type EquivalentRecipe } from '../services/engine';
  import { allManufacturers } from '../services/catalog';
  import { saveRecipe } from '../services/recipes.svelte';
  import { verdictKeys } from '../ui';
  import { t, decimal } from '../i18n.svelte';
  import { toast } from '../toast.svelte';

  interface PlannerRegion {
    id: number;
    x: number;
    y: number;
    r: number;
    g: number;
    b: number;
    hex: string;
    painted: boolean;
    result: EquivalentRecipe | null;
    computing: boolean;
  }

  interface ShoppingItem {
    paintId: number;
    name: string;
    code: string;
    manufacturer: string;
  }

  let canvasEl: HTMLCanvasElement | undefined = $state();
  /** a caixa da foto — é ELA que dá o tamanho do bitmap e o enquadramento
   *  (D-002a: medir o próprio canvas é circular, porque o bitmap dependia da
   *  resolução da imagem). */
  let boxEl: HTMLDivElement | undefined = $state();
  let fileInputEl: HTMLInputElement | undefined = $state();
  let image: HTMLImageElement | null = $state(null);
  let hasImage = $state(false);

  let regions: PlannerRegion[] = $state([]);
  let nextId = $state(1);
  let selectedId: number | null = $state(null);

  let view: View = $state({ zoom: 1, panX: 0, panY: 0 });
  let touchStartDist = 0;
  let touchStartZoom = 1;

  let manufacturers = $derived(allManufacturers());
  let manufacturerId: number | null = $state(null);

  // Checklist local de "tenho"/"comprar" das tintas da peça — não há conceito
  // de posse por tinta no app hoje (só por marca, na estante); este estado
  // vive só nesta tela, como um checklist de compra da montagem atual.
  let ownedPaints: Set<number> = $state(new Set());

  $effect(() => {
    if (manufacturerId === null && manufacturers.length > 0) manufacturerId = manufacturers[0].id;
  });

  let selectedRegion = $derived(regions.find(r => r.id === selectedId) ?? null);
  let paintedCount = $derived(regions.filter(r => r.painted).length);
  let progressPct = $derived(regions.length ? Math.round((paintedCount / regions.length) * 100) : 0);

  let plannerTitle = $derived(
    hasImage ? t('progress', { a: paintedCount, b: regions.length }) : t('t2EmptyTitle')
  );

  let shoppingItems = $derived.by((): ShoppingItem[] => {
    const map = new Map<number, ShoppingItem>();
    for (const r of regions) {
      if (!r.result) continue;
      for (const ing of r.result.ingredients) {
        if (!map.has(ing.paintId)) {
          map.set(ing.paintId, {
            paintId: ing.paintId,
            name: ing.name,
            code: ing.code,
            manufacturer: r.result.targetManufacturer,
          });
        }
      }
    }
    return [...map.values()];
  });

  function regionName(i: number): string {
    return t('regionLabel', { n: i + 1 });
  }

  function regionMatchText(r: PlannerRegion): string {
    if (r.computing) return t('calculating');
    if (!r.result) return t('noSimilarPaint');
    return `ΔE ${decimal(r.result.deltaE, 1)} · ${r.result.ingredients.map(ing => ing.name).join(' + ')}`;
  }

  function equivHeadline(res: EquivalentRecipe): string {
    return res.ingredients.length <= 1 ? t('hPote') : t('hMix', { n: res.ingredients.length });
  }

  // Cor do número de ΔE00 no painel de equivalência — some conforme o
  // veredito piora (RG-05), sem inventar tom fora dos tokens do tema.
  function verdictColor(deltaE: number): string {
    const { v } = verdictKeys(deltaE);
    if (v === 'v1' || v === 'v2') return 'var(--color-accent-400)';
    if (v === 'v3') return 'var(--color-neutral-300)';
    return 'var(--color-neutral-500)';
  }

  function toggleOwned(paintId: number) {
    const next = new Set(ownedPaints);
    if (next.has(paintId)) next.delete(paintId);
    else next.add(paintId);
    ownedPaints = next;
  }

  /** Cor de token para o canvas — o `<canvas>` não entende `var(--x)`, então o
   *  valor é lido do documento em vez de escrito em hex aqui (NFR-10). */
  function token(name: string, fallback: string): string {
    const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  /** Tamanho da caixa da foto em CSS px. Todo o enquadramento é calculado
   *  aqui — nunca a partir do canvas, que é justamente o que se ajusta. */
  function boxSize(): { w: number; h: number } {
    return { w: boxEl?.clientWidth ?? 0, h: boxEl?.clientHeight ?? 0 };
  }

  function draw() {
    const c = canvasEl;
    if (!c || !image) return;
    const ctx = c.getContext('2d');
    if (!ctx) return;

    const { w, h } = boxSize();
    if (w <= 0 || h <= 0) return;
    const dpr = window.devicePixelRatio || 1;

    // O bitmap acompanha a CAIXA (em pixels do dispositivo); o desenho fala em
    // CSS px, com a escala do dispositivo aplicada de uma vez na transformação.
    c.width = Math.max(1, Math.round(w * dpr));
    c.height = Math.max(1, Math.round(h * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);

    ctx.save();
    ctx.translate(view.panX, view.panY);
    ctx.scale(view.zoom, view.zoom);
    ctx.drawImage(image, 0, 0);
    ctx.restore();

    regions.forEach((r, i) => {
      const { x: sx, y: sy } = toScreen(view, r.x, r.y);
      const R = selectedId === r.id ? 16 : 12;
      ctx.save();
      ctx.translate(sx, sy);
      ctx.beginPath();
      ctx.arc(0, 0, R, 0, Math.PI * 2);
      ctx.fillStyle = r.hex;
      ctx.fill();
      ctx.strokeStyle = token('--color-neutral-100', '#f3f5fe');
      ctx.lineWidth = 2;
      ctx.stroke();
      ctx.fillStyle = token('--color-neutral-100', '#f3f5fe');
      ctx.font = 'bold 11px sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(String(i + 1), 0, 1);
      if (selectedId === r.id) {
        ctx.beginPath();
        ctx.arc(0, 0, R + 4, 0, Math.PI * 2);
        ctx.strokeStyle = token('--color-accent', '#9184d9');
        ctx.lineWidth = 2;
        ctx.setLineDash([3, 3]);
        ctx.stroke();
        ctx.setLineDash([]);
      }
      ctx.restore();
    });
  }

  /** Enquadra a foto inteira na caixa (também é o botão "ajustar à tela"). */
  function fitToBox() {
    if (!image) return;
    const { w, h } = boxSize();
    if (w <= 0 || h <= 0) return;
    view = fitView(image.width, image.height, w, h);
    draw();
  }

  $effect(() => {
    if (!image || !canvasEl) return;
    // Uma volta de layout antes de medir: no primeiro quadro depois do
    // {#if hasImage} a caixa ainda não tem tamanho.
    requestAnimationFrame(fitToBox);
  });

  // Redimensionar a janela mantém o enquadramento válido; sem isso a foto
  // fica deslocada depois de girar o tablet.
  $effect(() => {
    if (!boxEl) return;
    const ro = new ResizeObserver(() => {
      if (image) draw();
    });
    ro.observe(boxEl);
    return () => ro.disconnect();
  });

  function loadImage(src: string) {
    const img = new Image();
    img.onload = () => {
      image = img;
      hasImage = true;
      regions = [];
      nextId = 1;
      selectedId = null;
      ownedPaints = new Set();
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

  /** Ponto do evento em CSS px relativos à caixa — mesmo espaço do desenho. */
  function canvasPos(clientX: number, clientY: number) {
    const el = canvasEl ?? boxEl;
    if (!el) return { mx: 0, my: 0 };
    const rect = el.getBoundingClientRect();
    return { mx: clientX - rect.left, my: clientY - rect.top };
  }

  function hitTest(cx: number, cy: number): PlannerRegion | null {
    let best: PlannerRegion | null = null;
    let bestDist = 22;
    for (const r of regions) {
      const { x: sx, y: sy } = toScreen(view, r.x, r.y);
      const d = Math.hypot(cx - sx, cy - sy);
      if (d < bestDist) {
        bestDist = d;
        best = r;
      }
    }
    return best;
  }

  // ── rf-06: zoom in/out ──
  function zoomBy(factor: number, anchor?: { x: number; y: number }) {
    if (!image) return;
    const { w, h } = boxSize();
    view = zoomAround(view, factor, anchor ?? { x: w / 2, y: h / 2 });
    draw();
  }

  function onWheel(e: WheelEvent) {
    if (!image) return;
    e.preventDefault();
    const { mx, my } = canvasPos(e.clientX, e.clientY);
    zoomBy(e.deltaY < 0 ? 1.12 : 1 / 1.12, { x: mx, y: my });
  }

  function getTouchDist(t: TouchList): number {
    if (t.length < 2) return 0;
    return Math.hypot(t[0].clientX - t[1].clientX, t[0].clientY - t[1].clientY);
  }

  function onTouchStart(e: TouchEvent) {
    if (!canvasEl || !image) return;
    e.preventDefault();
    if (e.touches.length === 2) {
      touchStartDist = getTouchDist(e.touches);
      touchStartZoom = view.zoom;
      return;
    }
    if (e.touches.length === 1) {
      const tt = e.touches[0];
      handlePoint(tt.clientX, tt.clientY);
    }
  }

  function onTouchMove(e: TouchEvent) {
    if (!canvasEl || !image) return;
    e.preventDefault();
    if (e.touches.length === 2 && touchStartDist > 0) {
      const dist = getTouchDist(e.touches);
      const alvo = touchStartZoom * (dist / touchStartDist);
      const meio = canvasPos(
        (e.touches[0].clientX + e.touches[1].clientX) / 2,
        (e.touches[0].clientY + e.touches[1].clientY) / 2,
      );
      // A pinça também ancora no ponto entre os dedos.
      view = zoomAround(view, alvo / view.zoom, { x: meio.mx, y: meio.my });
      draw();
    }
  }

  function onCanvasClick(e: MouseEvent) {
    handlePoint(e.clientX, e.clientY);
  }

  function handlePoint(clientX: number, clientY: number) {
    if (!image) return;
    const { mx, my } = canvasPos(clientX, clientY);
    const hit = hitTest(mx, my);
    if (hit) {
      selectedId = hit.id;
      draw();
      return;
    }
    const p = toImage(view, mx, my);
    const ix = Math.round(p.x);
    const iy = Math.round(p.y);
    if (ix >= 0 && iy >= 0 && ix < image.width && iy < image.height) {
      void addRegion(ix, iy);
    }
  }

  async function addRegion(ix: number, iy: number) {
    if (!image) return;
    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(ix, iy, 1, 1).data);
    const hex = `#${[r, g, b].map(n => n.toString(16).padStart(2, '0')).join('')}`;

    const region: PlannerRegion = {
      id: nextId++,
      x: ix,
      y: iy,
      r, g, b, hex,
      painted: false,
      result: null,
      computing: true,
    };
    regions = [...regions, region];
    selectedId = region.id;
    draw();
    await computeRegion(region.id);
  }

  /** D-002b: o cálculo é endereçado por ID, nunca por referência.
   *  `regions` é `$state`, então o que está no array é o PROXY da região —
   *  escrever no objeto cru que `addRegion` criou não dispara reatividade
   *  nenhuma, e a tela fica presa em "calculando" com a resposta já em mãos. */
  async function computeRegion(id: number) {
    const alvo = () => regions.find(r => r.id === id) ?? null;
    const inicio = alvo();
    if (!inicio) return;
    inicio.computing = true;
    try {
      const res =
        manufacturerId != null
          ? await suggestRecipeForColor(inicio.r, inicio.g, inicio.b, manufacturerId)
          : null;
      const agora = alvo();
      if (agora) agora.result = res;
    } catch (e) {
      console.error('Erro ao calcular região:', e);
      const agora = alvo();
      if (agora) agora.result = null;
    } finally {
      const agora = alvo();
      if (agora) agora.computing = false;
      draw();
    }
  }

  // CA14: trocar de fabricante recalcula TODAS as regiões visíveis.
  async function onManufacturerChange(id: number) {
    manufacturerId = id;
    for (const r of regions) {
      void computeRegion(r.id);
    }
  }

  function selectRegion(id: number) {
    selectedId = id;
    draw();
  }

  function togglePainted(r: PlannerRegion) {
    r.painted = !r.painted;
  }

  function removeSelected() {
    if (selectedId == null) return;
    regions = regions.filter(r => r.id !== selectedId);
    selectedId = null;
    draw();
  }

  function saveRegionAsRecipe() {
    const r = selectedRegion;
    if (!r || manufacturerId == null) return;
    const idx = regions.findIndex(x => x.id === r.id);
    saveRecipe({
      name: regionName(idx),
      targetPaintId: null,
      targetR: r.r,
      targetG: r.g,
      targetB: r.b,
      manufacturerId,
    });
    toast(t('recipeSavedToast'));
  }
</script>

<div style="display: flex; flex-direction: column; height: 100%; min-height: 0;">
  <Header kicker={t('navPlano')} title={plannerTitle}>
    {#snippet actions()}
      <span style="display: inline-flex; align-items: center; gap: 10px; height: 48px; padding: 0 16px; border: 1px solid var(--color-neutral-800); border-radius: 8px; font-size: 15px; color: var(--color-neutral-400);">
        <i class="ph ph-pencil-simple" style="font-size: 18px; color: var(--color-accent-400);"></i>{t('pencilTool')}
      </span>
      <button
        class="t2-newphoto-btn"
        style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 18px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
        onclick={() => fileInputEl?.click()}
      >
        <i class="ph ph-camera" style="font-size: 19px;"></i>{t('newPhoto')}
      </button>
      <input type="file" accept="image/*" bind:this={fileInputEl} onchange={onFileInput} style="display: none;" />
    {/snippet}
  </Header>

  <div style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 34%);">
    <div style="position: relative; border-right: 1px solid var(--color-line); overflow: hidden;">
      <div
        bind:this={boxEl}
        style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: repeating-conic-gradient(var(--color-panel) 0% 25%, var(--color-bg) 0% 50%) 0 0 / 40px 40px; cursor: crosshair;"
      >
        {#if hasImage}
          <!-- O canvas ocupa a caixa inteira; o enquadramento da foto vive na
               transformação, não no tamanho do elemento (D-002a). -->
          <canvas
            bind:this={canvasEl}
            style="width: 100%; height: 100%; display: block; touch-action: none; cursor: crosshair;"
            ontouchstart={onTouchStart}
            ontouchmove={onTouchMove}
            onclick={onCanvasClick}
            onwheel={onWheel}
          ></canvas>
        {:else}
          <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; text-align: center; padding: 20px; pointer-events: none;">
            <i class="ph ph-image" style="font-size: 44px; color: var(--color-neutral-800);"></i>
            <span style="font-size: 15px; color: var(--color-neutral-500);">{t('t2EmptyTitle')}</span>
            <span style="font-size: 13.5px; color: var(--color-neutral-600); max-width: 40ch;">{t('t2EmptyHint')}</span>
          </div>
        {/if}
      </div>

      {#if hasImage}
        <!-- rf-06: zoom in/out. A pinça já existia; no desktop não havia como
             aproximar. Roda do mouse faz o mesmo, ancorada no cursor. -->
        <div
          style="position: absolute; right: 20px; top: 20px; display: flex; align-items: center; gap: 6px; padding: 6px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: rgba(22,24,38,0.92); backdrop-filter: blur(8px);"
        >
          <button
            class="t2-zoom-btn"
            aria-label={t('zoomOut')}
            title={t('zoomOut')}
            onclick={() => zoomBy(1 / 1.25)}
            style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); cursor: pointer;"
          >
            <i class="ph ph-minus" style="font-size: 18px;"></i>
          </button>
          <span
            aria-live="polite"
            style="min-width: 58px; text-align: center; font-size: 13px; color: var(--color-neutral-400); font-variant-numeric: tabular-nums;"
          >{zoomPercent(view)}%</span>
          <button
            class="t2-zoom-btn"
            aria-label={t('zoomIn')}
            title={t('zoomIn')}
            onclick={() => zoomBy(1.25)}
            style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); cursor: pointer;"
          >
            <i class="ph ph-plus" style="font-size: 18px;"></i>
          </button>
          <button
            class="t2-zoom-btn"
            aria-label={t('zoomFit')}
            title={t('zoomFit')}
            onclick={fitToBox}
            style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); cursor: pointer;"
          >
            <i class="ph ph-corners-out" style="font-size: 18px;"></i>
          </button>
        </div>
      {/if}

      <div style="position: absolute; left: 20px; bottom: 20px; display: flex; align-items: center; gap: 14px; padding: 12px 18px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: rgba(22,24,38,0.92); backdrop-filter: blur(8px);">
        <span style="font-size: 15px; color: var(--color-neutral-400);">{t('progress', { a: paintedCount, b: regions.length })}</span>
        <span style="width: 160px; height: 6px; border-radius: 999px; background: var(--color-rule); overflow: hidden;">
          <span style="display: block; height: 6px; width: {progressPct}%; background: var(--color-accent);"></span>
        </span>
      </div>
    </div>

    <div style="min-height: 0; display: flex; flex-direction: column;">
      <div style="flex: 1; min-height: 0; overflow-y: auto; padding: 20px;">
        <p class="section-label" style="margin: 0 0 10px;">{t('paintWithMine')}</p>
        <div style="display: flex; flex-wrap: wrap; gap: 8px;">
          {#each manufacturers as m (m.id)}
            <button
              class="t2-mfr-pill"
              style="height: 46px; padding: 0 14px; border: 1px solid {manufacturerId === m.id ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {manufacturerId === m.id ? 'var(--color-accent)' : 'var(--color-bg)'}; color: {manufacturerId === m.id ? 'var(--color-accent-100)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer;"
              onclick={() => onManufacturerChange(m.id)}
            >{m.name}</button>
          {/each}
        </div>

        {#if selectedRegion}
          {@const region = selectedRegion}
          {@const idx = regions.findIndex(x => x.id === region.id)}
          <div style="position: relative; margin: 14px 0 22px; padding: 16px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel);">
            <button
              class="t2-remove-btn"
              aria-label={t('removeRegionBtn')}
              style="position: absolute; top: 10px; right: 10px; width: 32px; height: 32px; display: inline-flex; align-items: center; justify-content: center; border-radius: 8px; background: transparent; color: var(--color-neutral-500); cursor: pointer;"
              onclick={removeSelected}
            >
              <i class="ph ph-trash-simple" style="font-size: 16px;"></i>
            </button>

            {#if region.computing}
              <div
                style="display: flex; align-items: center; justify-content: center; gap: 12px; height: 80px; border-radius: 8px; background: var(--color-raised); font-size: 14px; color: var(--color-neutral-500);"
              >
                <Spinner size={22} label={t('calculating')} />{t('calculating')}
              </div>
            {:else if region.result}
              {@const res = region.result}
              {@const vc = verdictKeys(res.deltaE)}
              <div style="display: flex; align-items: center; gap: 14px;">
                <span style="display: flex; border-radius: 4px; overflow: hidden; border: 1px solid var(--color-neutral-800); flex-shrink: 0;">
                  <span style="width: 30px; height: 44px; background: {region.hex};"></span>
                  <span style="width: 30px; height: 44px; background: rgb({res.resultR}, {res.resultG}, {res.resultB});"></span>
                </span>
                <span style="display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0;">
                  <span style="font-size: 12px; letter-spacing: 0.1em; text-transform: uppercase; color: var(--color-accent-400);">{regionName(idx)}</span>
                  <span style="font-size: clamp(14px, 1.45cqi, 17px); font-weight: 500; color: var(--color-text); text-wrap: pretty;">{equivHeadline(res)}</span>
                </span>
                <span style="display: flex; flex-direction: column; align-items: flex-end; flex-shrink: 0;">
                  <span class="font-mono" style="font-size: 26px; font-weight: 500; line-height: 1; color: {verdictColor(res.deltaE)};">{decimal(res.deltaE, 1)}</span>
                  <span style="font-size: 11px; color: var(--color-neutral-500);">ΔE00</span>
                </span>
              </div>
              <div style="display: flex; flex-direction: column; gap: 6px; margin-top: 14px;">
                {#each res.ingredients as ing (ing.paintId)}
                  <div style="display: flex; align-items: center; gap: 12px;">
                    <span style="width: 22px; height: 22px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({ing.r}, {ing.g}, {ing.b}); flex-shrink: 0;"></span>
                    <span style="flex: 1; min-width: 0; font-size: 14px; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.name}</span>
                    <span style="font-size: 13px; color: var(--color-neutral-500); flex-shrink: 0;">{ing.code}</span>
                    <span class="font-mono" style="width: 66px; text-align: right; font-size: 14px; font-weight: 500; color: var(--color-neutral-400); flex-shrink: 0;">{Math.round(ing.percentage)}%</span>
                  </div>
                {/each}
              </div>
              <p style="margin: 12px 0 0; font-size: 13.5px; color: var(--color-neutral-400); text-wrap: pretty;">{t(vc.c)}</p>
            {:else}
              <p style="font-size: 13.5px; color: var(--color-neutral-400);">{t('noSimilarPaint')}</p>
            {/if}
          </div>
        {/if}

        <p class="section-label" style="margin: 0 0 12px;">{t('regions')}</p>
        <div style="display: flex; flex-direction: column; gap: 10px;">
          {#each regions as r, i (r.id)}
            <div
              class="t2-region-card"
              role="button"
              tabindex="0"
              style="padding: 14px; border: 1px solid {selectedId === r.id ? 'var(--color-accent-700)' : 'var(--color-neutral-800)'}; border-radius: 14px; background: {selectedId === r.id ? 'var(--color-accent-panel)' : 'var(--color-panel)'}; cursor: pointer;"
              onclick={() => selectRegion(r.id)}
              onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectRegion(r.id); } }}
            >
              <div style="display: flex; align-items: center; gap: 12px;">
                <span style="width: 34px; height: 34px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: {r.hex}; flex-shrink: 0;"></span>
                <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                  <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{regionName(i)}</span>
                  <span class="font-mono" style="font-size: 12.5px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{regionMatchText(r)}</span>
                </span>
                <button
                  class="t2-done-btn"
                  aria-pressed={r.painted}
                  style="display: inline-flex; align-items: center; gap: 8px; height: 44px; padding: 0 12px; border: 1px solid {r.painted ? 'var(--color-accent-700)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: transparent; color: {r.painted ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 13px; font-weight: 500; cursor: pointer; flex-shrink: 0;"
                  onclick={(e) => { e.stopPropagation(); togglePainted(r); }}
                >
                  <i class="ph-bold ph-check" style="font-size: 14px;"></i>{r.painted ? t('painted') : t('toPaint')}
                </button>
              </div>
            </div>
          {/each}
        </div>

        <p class="section-label" style="margin: 26px 0 12px;">{t('piecePaints')}</p>
        <div style="display: flex; flex-direction: column;">
          {#each shoppingItems as s (s.paintId)}
            {@const owned = ownedPaints.has(s.paintId)}
            <div style="display: flex; align-items: center; gap: 12px; min-height: 56px; padding: 8px 2px; border-bottom: 1px solid var(--color-line);">
              <button
                aria-pressed={owned}
                style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: none; background: transparent; cursor: pointer; flex-shrink: 0;"
                onclick={() => toggleOwned(s.paintId)}
              >
                <span style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {owned ? 'var(--color-accent)' : 'var(--color-field-border)'}; border-radius: 4px; background: {owned ? 'var(--color-accent)' : 'transparent'};">
                  {#if owned}<i class="ph-bold ph-check" style="font-size: 15px; color: var(--color-accent-100);"></i>{/if}
                </span>
              </button>
              <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                <span style="font-size: 15px; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{s.name}</span>
                <span class="font-mono" style="font-size: 12.5px; color: var(--color-neutral-500);">{s.code} · {s.manufacturer}</span>
              </span>
              <span style="font-size: 12px; letter-spacing: 0.06em; text-transform: uppercase; color: {owned ? 'var(--color-accent-2)' : 'var(--color-neutral-500)'}; flex-shrink: 0;">{owned ? t('tagHave') : t('tagBuy')}</span>
            </div>
          {/each}
        </div>
      </div>

      <div style="flex-shrink: 0; padding: 16px 20px; border-top: 1px solid var(--color-line);">
        <button
          class="t2-save-btn"
          style="display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; height: 60px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 17px; font-weight: 500; cursor: pointer;"
          disabled={!selectedRegion}
          onclick={saveRegionAsRecipe}
        >
          <i class="ph ph-bookmark-simple" style="font-size: 20px;"></i>{t('saveRegionBtn')}
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .t2-newphoto-btn:hover {
    background: var(--color-accent-hover);
  }

  .t2-mfr-pill:hover {
    border-color: var(--color-accent-700);
  }

  .t2-region-card:hover {
    border-color: var(--color-accent-700);
  }

  .t2-done-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-zoom-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-remove-btn:hover {
    color: var(--color-neutral-300);
  }

  .t2-save-btn:hover {
    background: var(--color-accent-hover);
  }

  .t2-save-btn:disabled {
    opacity: 0.45;
    pointer-events: none;
  }
</style>
