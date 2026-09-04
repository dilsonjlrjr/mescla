<script lang="ts">
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import * as DialogService from '../../../bindings/paint-match-ai/dialogservice';
  import type { PickColorResponse, ColorMatchDTO, ManufacturerDTO } from '../../../bindings/paint-match-ai/models';
  import Icon from './Icon.svelte';
  import FloatingToolbar from './FloatingToolbar.svelte';
  import PrecisionLoupe from './PrecisionLoupe.svelte';
  import RegionCard from './RegionCard.svelte';
  import PlanListGrid from './PlanListGrid.svelte';
  import { toast } from '../toast.svelte';

  // ── Fabricantes ──
  let manufacturers: ManufacturerDTO[] = $state([]);
  PaintService.GetManufacturers().then(m => { manufacturers = m ?? []; });

  // ── Estado da imagem ──
  let canvasEl: HTMLCanvasElement | undefined = $state();
  let fileInputEl: HTMLInputElement | undefined = $state();
  let planFileInputEl: HTMLInputElement | undefined = $state();
  let image: HTMLImageElement | null = $state(null);
  let zoom = $state(1);
  let panX = $state(0);
  let panY = $state(0);
  let dragging = $state(false);
  let dragStartX = $state(0);
  let dragStartY = $state(0);
  let draggingPickId: number | null = $state(null);
  let cursorImgX = $state(-1);
  let cursorImgY = $state(-1);
  let canvasW = $state(800);
  let canvasH = $state(600);

  // ── Estado do plano ──
  let regions: PaintingRegion[] = $state([]);
  let nextId = $state(1);
  let planName = $state('');
  let imageDataUrl = $state('');
  let currentPlanId = $state(0);

  // ── UI state ──
  let hasImage = $state(false);
  let pickedInfo: { hex: string; brand: string; name: string } | null = $state(null);
  let selectedRegionId: number | null = $state(null);
  let mode: 'view' | 'pick' | 'edit' = $state('pick');
  let picking = $state(false);
  let savedIndicator = $state(false);
  let showPlanGrid = $state(false);
  let planGridLoading = $state(false);
  let planGridData: any[] = $state([]);

  // ── Undo/Redo ──
  interface UndoAction {
    type: 'add' | 'remove' | 'move' | 'changeMfr';
    data: any;
  }
  let undoStack: UndoAction[] = $state([]);
  let redoStack: UndoAction[] = $state([]);
  const MAX_UNDO = 50;

  // ── Auto-save ──
  let autoSaveTimer: ReturnType<typeof setTimeout> | null = null;
  const AUTO_SAVE_DELAY = 2000;

  // ── Touch state ──
  let touchStartDist = $state(0);
  let touchStartZoom = $state(1);
  let touchStartPanX = $state(0);
  let touchStartPanY = $state(0);
  let touchMidX = $state(0);
  let touchMidY = $state(0);
  let longPressTimer: ReturnType<typeof setTimeout> | null = $state(null);
  let isLongPress = $state(false);

  // ── Loupe state ──
  let loupeVisible = $state(false);
  let loupeX = $state(0);
  let loupeY = $state(0);
  let loupeScreenX = $state(0);
  let loupeScreenY = $state(0);
  let loupeHex = $state('');

  interface PaintingRegion {
    id: number;
    x: number; y: number;
    r: number; g: number; b: number;
    hex: string;
    regionName: string;
    note: string;
    targetMfrId: number;
    match: ColorMatchDTO | null;
    allMatches: ColorMatchDTO[];
    recipe: any;
    picking?: boolean;
  }

  // ── Zoom ──
  function zoomIn() { zoom = Math.min(5, zoom * 1.25); draw(); }
  function zoomOut() { zoom = Math.max(0.25, zoom / 1.25); draw(); }
  function zoomReset() { zoom = 1; panX = 0; panY = 0; draw(); }
  function zoomFit() {
    if (!canvasEl || !image) return;
    const wrap = canvasEl.parentElement;
    if (!wrap) return;
    const rect = wrap.getBoundingClientRect();
    const cw = rect.width;
    const ch = rect.height;
    if (cw <= 0 || ch <= 0) return;
    zoom = Math.min(cw / image.width, ch / image.height, 1);
    panX = (cw - image.width * zoom) / 2;
    panY = (ch - image.height * zoom) / 2;
    draw();
  }

  // ── Redesenha ──
  $effect(() => {
    if (image && canvasEl) {
      requestAnimationFrame(() => { zoomFit(); });
    }
  });

  // ── Desenho no canvas ──
  function draw() {
    const c = canvasEl;
    if (!c || !image) return;
    const ctx = c.getContext('2d');
    if (!ctx) return;

    const wrap = c.parentElement;
    if (!wrap) return;
    const rect = wrap.getBoundingClientRect();
    c.width = Math.floor(rect.width);
    c.height = Math.floor(rect.height);

    ctx.clearRect(0, 0, c.width, c.height);
    ctx.save();
    ctx.translate(panX, panY);
    ctx.scale(zoom, zoom);
    ctx.drawImage(image, 0, 0);
    ctx.restore();

    for (const r of regions) {
      const sx = r.x * zoom + panX;
      const sy = r.y * zoom + panY;
      const R = draggingPickId === r.id ? 18 : (selectedRegionId === r.id ? 16 : 14);

      ctx.save();
      ctx.translate(sx, sy);

      // Outer ring (shadow)
      ctx.beginPath();
      ctx.arc(0, 0, R + 2, 0, Math.PI * 2);
      ctx.fillStyle = 'rgba(0,0,0,0.2)';
      ctx.fill();

      // Main circle
      ctx.beginPath();
      ctx.arc(0, 0, R, 0, Math.PI * 2);
      ctx.fillStyle = r.hex;
      ctx.fill();

      // Border
      ctx.strokeStyle = selectedRegionId === r.id ? '#e8542c' : '#fff';
      ctx.lineWidth = selectedRegionId === r.id ? 3 : 2;
      ctx.stroke();

      // Number badge
      if (zoom >= 0.4) {
        ctx.fillStyle = 'rgba(0,0,0,0.6)';
        ctx.beginPath();
        ctx.arc(0, 0, R * 0.55, 0, Math.PI * 2);
        ctx.fill();

        ctx.fillStyle = '#fff';
        ctx.font = `bold ${Math.max(10, R * 0.4)}px system-ui`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(String(r.id), 0, 0);
      }

      // Loading spinner
      if (r.picking) {
        ctx.beginPath();
        ctx.arc(0, 0, R + 4, 0, Math.PI * 2);
        ctx.strokeStyle = '#e8542c';
        ctx.lineWidth = 2;
        ctx.setLineDash([4, 4]);
        ctx.stroke();
        ctx.setLineDash([]);
      }

      ctx.restore();
    }
  }

  // ── Hit test ──
  function hitTest(cx: number, cy: number): PaintingRegion | null {
    let best: PaintingRegion | null = null;
    let bestDist = 20;
    for (const r of regions) {
      const sx = r.x * zoom + panX;
      const sy = r.y * zoom + panY;
      const d = Math.sqrt((cx - sx) ** 2 + (cy - sy) ** 2);
      if (d < bestDist) { bestDist = d; best = r; }
    }
    return best;
  }

  // ── Canvas coords helper ──
  function canvasMousePos(clientX: number, clientY: number): { mx: number; my: number } {
    if (!canvasEl) return { mx: 0, my: 0 };
    const rect = canvasEl.getBoundingClientRect();
    const scaleX = canvasEl.width / rect.width;
    const scaleY = canvasEl.height / rect.height;
    return {
      mx: (clientX - rect.left) * scaleX,
      my: (clientY - rect.top) * scaleY,
    };
  }

  // ── Mouse events ──
  function onCanvasMouseDown(e: MouseEvent) {
    if (!canvasEl) return;
    const { mx, my } = canvasMousePos(e.clientX, e.clientY);

    if (mode === 'pick' || mode === 'edit') {
      const hit = hitTest(mx, my);
      if (hit) {
        selectedRegionId = hit.id;
        draggingPickId = hit.id;
        draw();
        return;
      }
    }

    if (mode === 'view' || (mode === 'pick' && !hitTest(mx, my))) {
      dragging = true;
      dragStartX = e.clientX - panX;
      dragStartY = e.clientY - panY;
    }
  }

  function onCanvasMouseMove(e: MouseEvent) {
    if (!canvasEl || !image) return;
    const { mx, my } = canvasMousePos(e.clientX, e.clientY);
    cursorImgX = Math.round((mx - panX) / zoom);
    cursorImgY = Math.round((my - panY) / zoom);

    if (draggingPickId !== null) {
      const r = regions.find(r => r.id === draggingPickId);
      if (r) {
        r.x = Math.max(0, Math.min(image.width - 1, cursorImgX));
        r.y = Math.max(0, Math.min(image.height - 1, cursorImgY));
        draw();
      }
      return;
    }

    if (dragging) {
      panX = e.clientX - dragStartX;
      panY = e.clientY - dragStartY;
      draw();
    }
  }

  function onCanvasMouseUp(e: MouseEvent) {
    if (draggingPickId !== null) {
      const pickId = draggingPickId;
      draggingPickId = null;
      draw();
      repickColor(pickId);
      return;
    }

    if (!dragging) return;
    const dx = Math.abs(e.clientX - dragStartX - panX);
    const dy = Math.abs(e.clientY - dragStartY - panY);
    dragging = false;

    if (dx < 3 && dy < 3 && (mode === 'pick')) {
      pickColor(cursorImgX, cursorImgY);
    }
  }

  function onCanvasWheel(e: WheelEvent) {
    e.preventDefault();
    if (!canvasEl) return;
    const { mx, my } = canvasMousePos(e.clientX, e.clientY);
    const wx = (mx - panX) / zoom;
    const wy = (my - panY) / zoom;
    zoom = Math.max(0.25, Math.min(5, zoom * (e.deltaY > 0 ? 0.9 : 1.1)));
    panX = mx - wx * zoom;
    panY = my - wy * zoom;
    draw();
  }

  // ── Touch events ──
  function getTouchDist(t: TouchList): number {
    if (t.length < 2) return 0;
    const dx = t[0].clientX - t[1].clientX;
    const dy = t[0].clientY - t[1].clientY;
    return Math.sqrt(dx * dx + dy * dy);
  }

  function getTouchMid(t: TouchList): { x: number; y: number } {
    if (t.length < 2) return { x: t[0].clientX, y: t[0].clientY };
    return {
      x: (t[0].clientX + t[1].clientX) / 2,
      y: (t[0].clientY + t[1].clientY) / 2,
    };
  }

  function onTouchStart(e: TouchEvent) {
    if (!canvasEl || !image) return;

    if (e.touches.length === 2) {
      e.preventDefault();
      touchStartDist = getTouchDist(e.touches);
      touchStartZoom = zoom;
      const mid = getTouchMid(e.touches);
      touchMidX = mid.x;
      touchMidY = mid.y;
      touchStartPanX = panX;
      touchStartPanY = panY;
      cancelLongPress();
      return;
    }

    if (e.touches.length === 1) {
      const touch = e.touches[0];
      const { mx, my } = canvasMousePos(touch.clientX, touch.clientY);
      cursorImgX = Math.round((mx - panX) / zoom);
      cursorImgY = Math.round((my - panY) / zoom);

      const hit = hitTest(mx, my);
      if (hit && mode !== 'view') {
        selectedRegionId = hit.id;
        draggingPickId = hit.id;
        draw();
        return;
      }

      if (mode === 'pick') {
        isLongPress = false;
        longPressTimer = setTimeout(() => {
          isLongPress = true;
          showLoupe(touch.clientX, touch.clientY, cursorImgX, cursorImgY);
        }, 500);
      }

      dragging = true;
      dragStartX = touch.clientX - panX;
      dragStartY = touch.clientY - panY;
    }
  }

  function onTouchMove(e: TouchEvent) {
    if (!canvasEl || !image) return;
    e.preventDefault();

    if (e.touches.length === 2) {
      const dist = getTouchDist(e.touches);
      const scale = dist / touchStartDist;
      zoom = Math.max(0.25, Math.min(5, touchStartZoom * scale));

      const mid = getTouchMid(e.touches);
      const { mx, my } = canvasMousePos(mid.x, mid.y);
      const { mx: omx, my: omy } = canvasMousePos(touchMidX, touchMidY);
      panX = touchStartPanX + (mx - omx);
      panY = touchStartPanY + (my - omy);
      draw();
      return;
    }

    if (e.touches.length === 1) {
      const touch = e.touches[0];

      if (longPressTimer) {
        const { mx, my } = canvasMousePos(touch.clientX, touch.clientY);
        const dx = Math.abs(mx - (cursorImgX * zoom + panX));
        const dy = Math.abs(my - (cursorImgY * zoom + panY));
        if (dx > 10 || dy > 10) cancelLongPress();
      }

      if (isLongPress) {
        const { mx, my } = canvasMousePos(touch.clientX, touch.clientY);
        cursorImgX = Math.round((mx - panX) / zoom);
        cursorImgY = Math.round((my - panY) / zoom);
        showLoupe(touch.clientX, touch.clientY, cursorImgX, cursorImgY);
        return;
      }

      if (draggingPickId !== null) {
        const r = regions.find(r => r.id === draggingPickId);
        if (r) {
          const { mx, my } = canvasMousePos(touch.clientX, touch.clientY);
          r.x = Math.max(0, Math.min(image.width - 1, Math.round((mx - panX) / zoom)));
          r.y = Math.max(0, Math.min(image.height - 1, Math.round((my - panY) / zoom)));
          draw();
        }
        return;
      }

      if (dragging) {
        panX = touch.clientX - dragStartX;
        panY = touch.clientY - dragStartY;
        draw();
      }
    }
  }

  function onTouchEnd(e: TouchEvent) {
    cancelLongPress();

    if (isLongPress) {
      hideLoupe();
      pickColor(cursorImgX, cursorImgY);
      isLongPress = false;
      dragging = false;
      return;
    }

    if (draggingPickId !== null) {
      const pickId = draggingPickId;
      draggingPickId = null;
      draw();
      repickColor(pickId);
      return;
    }

    if (dragging && e.changedTouches.length === 1) {
      const touch = e.changedTouches[0];
      const dx = Math.abs(touch.clientX - dragStartX - panX);
      const dy = Math.abs(touch.clientY - dragStartY - panY);
      dragging = false;

      if (dx < 10 && dy < 10 && mode === 'pick') {
        pickColor(cursorImgX, cursorImgY);
      }
    }
  }

  function cancelLongPress() {
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
  }

  // ── Loupe ──
  function showLoupe(screenX: number, screenY: number, imgX: number, imgY: number) {
    if (!image) return;
    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(imgX, imgY, 1, 1).data);
    loupeHex = `#${r.toString(16).padStart(2,'0')}${g.toString(16).padStart(2,'0')}${b.toString(16).padStart(2,'0')}`.toUpperCase();
    loupeX = imgX;
    loupeY = imgY;
    loupeScreenX = screenX;
    loupeScreenY = screenY;
    loupeVisible = true;
  }

  function hideLoupe() {
    loupeVisible = false;
  }

  // ── Keyboard shortcuts ──
  function onKeyDown(e: KeyboardEvent) {
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return;

    if (e.key === 'v' || e.key === 'V') { mode = 'view'; }
    if (e.key === 'i' || e.key === 'I') { mode = 'pick'; }
    if (e.key === 'e' || e.key === 'E') { mode = 'edit'; }

    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
      e.preventDefault();
      undo();
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && e.shiftKey) {
      e.preventDefault();
      redo();
    }
  }

  // ── Undo/Redo ──
  function pushUndo(action: UndoAction) {
    undoStack = [...undoStack.slice(-MAX_UNDO + 1), action];
    redoStack = [];
  }

  function undo() {
    if (undoStack.length === 0) return;
    const action = undoStack[undoStack.length - 1];
    undoStack = undoStack.slice(0, -1);

    switch (action.type) {
      case 'add':
        regions = regions.filter(r => r.id !== action.data.id);
        redoStack = [...redoStack, { type: 'add', data: action.data }];
        break;
      case 'remove':
        regions = [...regions, action.data];
        redoStack = [...redoStack, { type: 'remove', data: action.data }];
        break;
      case 'move': {
        const r = regions.find(r => r.id === action.data.id);
        if (r) {
          const oldX = r.x, oldY = r.y;
          r.x = action.data.x;
          r.y = action.data.y;
          redoStack = [...redoStack, { type: 'move', data: { id: r.id, x: oldX, y: oldY } }];
        }
        break;
      }
    }
    draw();
    toast('Ação desfeita');
  }

  function redo() {
    if (redoStack.length === 0) return;
    const action = redoStack[redoStack.length - 1];
    redoStack = redoStack.slice(0, -1);

    switch (action.type) {
      case 'add':
        regions = [...regions, action.data];
        undoStack = [...undoStack, { type: 'add', data: action.data }];
        break;
      case 'remove':
        regions = regions.filter(r => r.id !== action.data.id);
        undoStack = [...undoStack, { type: 'remove', data: action.data }];
        break;
      case 'move': {
        const r = regions.find(r => r.id === action.data.id);
        if (r) {
          const oldX = r.x, oldY = r.y;
          r.x = action.data.x;
          r.y = action.data.y;
          undoStack = [...undoStack, { type: 'move', data: { id: r.id, x: oldX, y: oldY } }];
        }
        break;
      }
    }
    draw();
    toast('Ação refeita');
  }

  // ── Color pick ──
  async function pickColor(ix: number, iy: number) {
    if (!image || ix < 0 || iy < 0 || ix >= image.width || iy >= image.height) return;

    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(ix, iy, 1, 1).data);

    const region: PaintingRegion = {
      id: nextId++,
      x: ix, y: iy,
      r, g, b,
      hex: `#${r.toString(16).padStart(2,'0')}${g.toString(16).padStart(2,'0')}${b.toString(16).padStart(2,'0')}`.toUpperCase(),
      regionName: '',
      note: '',
      targetMfrId: 0,
      match: null,
      allMatches: [],
      recipe: null,
      picking: true,
    };

    regions = [...regions, region];
    selectedRegionId = region.id;
    pushUndo({ type: 'add', data: region });
    draw();

    try {
      const resp = await PaintService.PickColor(r, g, b, 0);
      const best = resp.matches?.[0];
      region.hex = resp.hex;
      region.match = best ? { ...best } : null;
      region.allMatches = resp.matches ?? [];
      region.recipe = resp.recipe ?? null;
      region.picking = false;
      pickedInfo = best ? { hex: resp.hex, brand: best.manufacturer, name: best.name } : null;
      draw();
      triggerAutoSave();
    } catch (e) {
      console.error('PickColor:', e);
      toast('Erro ao buscar cor', 'error');
      region.picking = false;
      draw();
    }
  }

  async function repickColor(pickId: number) {
    const region = regions.find(r => r.id === pickId);
    if (!region || !image) return;

    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(region.x, region.y, 1, 1).data);

    region.picking = true;
    draw();

    try {
      const resp = await PaintService.PickColor(r, g, b, region.targetMfrId);
      const best = resp.matches?.[0];
      region.r = r; region.g = g; region.b = b;
      region.hex = resp.hex;
      region.match = best ? { ...best } : null;
      region.allMatches = resp.matches ?? [];
      region.recipe = resp.recipe ?? null;
      region.picking = false;
      pickedInfo = best ? { hex: resp.hex, brand: best.manufacturer, name: best.name } : null;
      draw();
      triggerAutoSave();
    } catch (e) {
      console.error('RepickColor:', e);
      toast('Erro ao atualizar cor', 'error');
      region.picking = false;
      draw();
    }
  }

  // ── Sidebar ──
  function updateRegionName(id: number, name: string) {
    const r = regions.find(r => r.id === id);
    if (r) { r.regionName = name; triggerAutoSave(); }
  }

  function updateRegionNote(id: number, note: string) {
    const r = regions.find(r => r.id === id);
    if (r) { r.note = note; triggerAutoSave(); }
  }

  function removeRegion(id: number) {
    const region = regions.find(r => r.id === id);
    if (region) {
      pushUndo({ type: 'remove', data: { ...region } });
    }
    regions = regions.filter(r => r.id !== id);
    if (selectedRegionId === id) selectedRegionId = null;
    draw();
    triggerAutoSave();
  }

  function selectRegion(id: number) {
    selectedRegionId = id;
    draw();
  }

  async function changeRegionMfr(region: PaintingRegion, mfrId: number) {
    region.targetMfrId = mfrId;
    if (!image) return;
    const tmp = document.createElement('canvas');
    tmp.width = image.width;
    tmp.height = image.height;
    const tctx = tmp.getContext('2d')!;
    tctx.drawImage(image, 0, 0);
    const [r, g, b] = Array.from(tctx.getImageData(region.x, region.y, 1, 1).data);
    try {
      const resp = await PaintService.PickColor(r, g, b, mfrId);
      const best = resp.matches?.[0];
      region.r = r; region.g = g; region.b = b;
      region.hex = resp.hex;
      region.match = best ? { ...best } : null;
      region.allMatches = resp.matches ?? [];
      region.recipe = resp.recipe ?? null;
      draw();
      triggerAutoSave();
    } catch (e) {
      console.error('ChangeMfr:', e);
      toast('Erro ao trocar fabricante', 'error');
    }
  }

  // ── Stock check ──
  function isStockMatch(region: PaintingRegion): boolean {
    // Placeholder: requires user stock data to compare
    return false;
  }

  // ── Auto-save ──
  function triggerAutoSave() {
    if (autoSaveTimer) clearTimeout(autoSaveTimer);
    autoSaveTimer = setTimeout(autoSave, AUTO_SAVE_DELAY);
  }

  async function autoSave() {
    if (!hasImage || regions.length === 0) return;
    try {
      const dto = buildPlanDTO();
      const result = await PaintService.SavePlan(dto);
      currentPlanId = Number(result.id);
      savedIndicator = true;
      setTimeout(() => { savedIndicator = false; }, 3000);
    } catch (e) {
      console.error('AutoSave:', e);
    }
  }

  function buildPlanDTO() {
    const now = new Date().toISOString();
    return {
      id: currentPlanId,
      name: planName || `Plano ${new Date().toLocaleDateString('pt-BR')}`,
      imageData: imageDataUrl,
      regions: regions.map((r, i) => ({
        id: 0,
        planId: 0,
        x: r.x,
        y: r.y,
        r: r.r,
        g: r.g,
        b: r.b,
        hex: r.hex,
        regionName: r.regionName,
        note: r.note,
        paintId: r.match?.paintId ?? 0,
        paintBrand: r.match?.manufacturer ?? '',
        paintName: r.match?.name ?? '',
        paintCode: r.match?.code ?? '',
        deltaE: r.match?.deltaE ?? 0,
        sortOrder: i,
      })),
      createdAt: now,
      updatedAt: now,
    };
  }

  // ── Plan management ──
  async function openPlanGrid() {
    showPlanGrid = true;
    planGridLoading = true;
    try {
      planGridData = (await PaintService.ListPlans()) ?? [];
    } catch (e) {
      console.error('ListPlans:', e);
      toast('Erro ao carregar planos', 'error');
    }
    planGridLoading = false;
  }

  async function loadPlanFromDB(id: number) {
    try {
      const plan = await PaintService.LoadPlan(id);
      if (!plan) { toast('Plano não encontrado', 'error'); return; }

      const img = new Image();
      img.onload = () => {
        image = img;
        hasImage = true;
        zoom = 1; panX = 0; panY = 0;
        planName = plan.name || '';
        imageDataUrl = plan.imageData || '';
        currentPlanId = Number(plan.id);
        nextId = 1;
        regions = (plan.regions || []).map((r: any) => ({
          id: nextId++,
          x: r.x, y: r.y,
          r: r.r, g: r.g, b: r.b,
          hex: r.hex,
          regionName: r.regionName || '',
          note: r.note || '',
          targetMfrId: 0,
          match: r.paintId ? {
            paintId: r.paintId,
            name: r.paintName,
            manufacturer: r.paintBrand,
            code: r.paintCode,
            r: r.r, g: r.g, b: r.b,
            hex: r.hex,
            deltaE: r.deltaE,
          } : null,
          allMatches: [],
          recipe: null,
        }));
        showPlanGrid = false;
        toast(`Plano carregado: ${regions.length} regiões`);
        requestAnimationFrame(() => zoomFit());
      };
      img.src = plan.imageData || '';
    } catch (e) {
      console.error('LoadPlan:', e);
      toast('Erro ao carregar plano', 'error');
    }
  }

  async function deletePlanFromDB(id: number) {
    try {
      await PaintService.DeletePlan(id);
      planGridData = planGridData.filter((p: any) => p.id !== id);
      toast('Plano excluído');
    } catch (e) {
      console.error('DeletePlan:', e);
      toast('Erro ao excluir plano', 'error');
    }
  }

  // ── Util ──
  function escHtml(s: string) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  // ── Upload de imagem ──
  function loadImage(src: string) {
    const img = new Image();
    img.onload = () => {
      image = img;
      hasImage = true;
      zoom = 1;
      panX = 0;
      panY = 0;
      regions = [];
      nextId = 1;
      planName = '';
      imageDataUrl = src;
      currentPlanId = 0;
      undoStack = [];
      redoStack = [];
    };
    img.src = src;
  }

  function onFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      toast('Imagem excede 2MB', 'error');
      return;
    }
    const reader = new FileReader();
    reader.onload = () => loadImage(reader.result as string);
    reader.readAsDataURL(file);
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer?.files?.[0];
    if (!file || !file.type.startsWith('image/')) return;
    if (file.size > 2 * 1024 * 1024) {
      toast('Imagem excede 2MB', 'error');
      return;
    }
    const reader = new FileReader();
    reader.onload = () => loadImage(reader.result as string);
    reader.readAsDataURL(file);
  }

  function onPaste(e: ClipboardEvent) {
    const items = e.clipboardData?.items;
    if (!items) return;
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        const blob = item.getAsFile();
        if (!blob) continue;
        const reader = new FileReader();
        reader.onload = () => loadImage(reader.result as string);
        reader.readAsDataURL(blob);
        break;
      }
    }
  }

  // ── Export PNG ──
  async function exportPNG() {
    if (!hasImage || regions.length === 0 || !image) return;

    const HEADER_H = 60;
    const FOOTER_H = 48 + Math.ceil(regions.length / 3) * 40;
    const GAP = 16;
    
    const exportCanvas = document.createElement('canvas');
    const maxW = Math.max(image.width, 800);
    const scale = maxW / image.width;
    const scaledW = Math.round(image.width * scale);
    const scaledH = Math.round(image.height * scale);
    
    exportCanvas.width = scaledW;
    exportCanvas.height = HEADER_H + scaledH + GAP + FOOTER_H;
    const ectx = exportCanvas.getContext('2d')!;

    ectx.fillStyle = '#ffffff';
    ectx.fillRect(0, 0, exportCanvas.width, exportCanvas.height);

    ectx.fillStyle = '#1a1712';
    ectx.fillRect(0, 0, exportCanvas.width, HEADER_H);
    
    ectx.fillStyle = '#ffffff';
    ectx.font = 'bold 24px system-ui, sans-serif';
    ectx.textAlign = 'left';
    ectx.textBaseline = 'middle';
    ectx.fillText(planName || 'Plano de Pintura', 20, HEADER_H / 2);
    
    ectx.fillStyle = '#e8542c';
    ectx.font = 'bold 14px system-ui, sans-serif';
    ectx.textAlign = 'right';
    ectx.fillText('MESCLA AI', exportCanvas.width - 20, HEADER_H / 2);

    ectx.drawImage(image, 0, HEADER_H, scaledW, scaledH);

    for (const r of regions) {
      const sx = r.x * scale;
      const sy = HEADER_H + r.y * scale;
      const R = 16;

      ectx.save();
      ectx.translate(sx, sy);

      ectx.fillStyle = 'rgba(0,0,0,0.3)';
      ectx.beginPath();
      ectx.arc(0, 0, R + 3, 0, Math.PI * 2);
      ectx.fill();

      ectx.beginPath();
      ectx.arc(0, 0, R, 0, Math.PI * 2);
      ectx.fillStyle = r.hex;
      ectx.fill();
      ectx.strokeStyle = '#ffffff';
      ectx.lineWidth = 2.5;
      ectx.stroke();

      ectx.fillStyle = '#000000';
      ectx.beginPath();
      ectx.arc(0, 0, R * 0.6, 0, Math.PI * 2);
      ectx.fill();

      ectx.fillStyle = '#ffffff';
      ectx.font = `bold ${Math.max(10, R * 0.55)}px system-ui`;
      ectx.textAlign = 'center';
      ectx.textBaseline = 'middle';
      ectx.fillText(String(r.id), 0, 0);
      
      ectx.restore();
    }

    const footerY = HEADER_H + scaledH + GAP;
    
    ectx.fillStyle = '#f6f4ef';
    ectx.fillRect(0, footerY, exportCanvas.width, FOOTER_H);
    
    ectx.strokeStyle = '#dcd6c9';
    ectx.lineWidth = 1;
    ectx.beginPath();
    ectx.moveTo(0, footerY);
    ectx.lineTo(exportCanvas.width, footerY);
    ectx.stroke();

    ectx.fillStyle = '#1a1712';
    ectx.font = 'bold 12px system-ui';
    ectx.textAlign = 'left';
    ectx.fillText(`${regions.length} regiões`, 16, footerY + 20);

    const cols = 3;
    const colW = (exportCanvas.width - 32) / cols;
    const itemH = 36;
    const startY = footerY + 40;

    regions.forEach((r, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      const x = 16 + col * colW;
      const y = startY + row * itemH;

      const R = 12;
      ectx.beginPath();
      ectx.arc(x + R, y + R, R, 0, Math.PI * 2);
      ectx.fillStyle = r.hex;
      ectx.fill();
      ectx.strokeStyle = '#dcd6c9';
      ectx.lineWidth = 1;
      ectx.stroke();

      ectx.fillStyle = '#000000';
      ectx.beginPath();
      ectx.arc(x + R, y + R, R * 0.55, 0, Math.PI * 2);
      ectx.fill();

      ectx.fillStyle = '#ffffff';
      ectx.font = `bold ${Math.max(8, R * 0.5)}px system-ui`;
      ectx.textAlign = 'center';
      ectx.textBaseline = 'middle';
      ectx.fillText(String(r.id), x + R, y + R);

      ectx.fillStyle = '#1a1712';
      ectx.font = '12px system-ui';
      ectx.textAlign = 'left';
      ectx.textBaseline = 'middle';
      
      const label = r.regionName || `#${r.id}`;
      let text = label;
      if (r.match) {
        text += ` · ${r.match.name}`;
      }
      
      const maxTextW = colW - 40;
      while (ectx.measureText(text).width > maxTextW && text.length > 3) {
        text = text.slice(0, -1);
      }
      if (text !== label && r.match) text += '…';
      
      ectx.fillText(text, x + 32, y + R);

      if (r.match) {
        ectx.fillStyle = '#e8542c';
        ectx.font = '10px "IBM Plex Mono", monospace';
        const deltaE = `ΔE ${r.match.deltaE.toFixed(1)}`;
        const deltaW = ectx.measureText(deltaE).width;
        ectx.fillText(deltaE, x + colW - deltaW - 8, y + R);
      }
    });

    exportCanvas.toBlob(async (blob) => {
      if (!blob) return;
      
      try {
        let filepath = await DialogService.SavePNG();
        if (!filepath) return;
        if (!filepath.endsWith('.png')) filepath += '.png';
        
        const reader = new FileReader();
        reader.onload = async () => {
          const dataUrl = reader.result as string;
          const base64 = dataUrl.split(',')[1];
          
          await DialogService.SaveFileWithData(filepath, base64, '.png');
          toast('PNG exportado!');
        };
        reader.readAsDataURL(blob);
      } catch (e) {
        console.error('Erro ao salvar PNG:', e);
        toast('Erro ao salvar arquivo', 'error');
      }
    }, 'image/png');
  }

  // ── Export PDF ──
  async function exportPDF() {
    if (!hasImage || regions.length === 0 || !image) return;

    const printCanvas = document.createElement('canvas');
    printCanvas.width = image.width;
    printCanvas.height = image.height;
    const pctx = printCanvas.getContext('2d')!;
    pctx.drawImage(image, 0, 0);

    for (const r of regions) {
      const R = 20;
      pctx.save();
      pctx.translate(r.x, r.y);

      pctx.fillStyle = 'rgba(0,0,0,0.25)';
      pctx.beginPath();
      pctx.arc(0, 0, R + 4, 0, Math.PI * 2);
      pctx.fill();

      pctx.beginPath();
      pctx.arc(0, 0, R, 0, Math.PI * 2);
      pctx.fillStyle = r.hex;
      pctx.fill();
      pctx.strokeStyle = '#fff';
      pctx.lineWidth = 3;
      pctx.stroke();

      pctx.fillStyle = '#000';
      pctx.beginPath();
      pctx.arc(0, 0, R * 0.6, 0, Math.PI * 2);
      pctx.fill();

      pctx.fillStyle = '#fff';
      pctx.font = `bold ${Math.max(12, R * 0.55)}px system-ui`;
      pctx.textAlign = 'center';
      pctx.textBaseline = 'middle';
      pctx.fillText(String(r.id), 0, 0);
      pctx.restore();
    }

    const imgSrc = printCanvas.toDataURL('image/png');

    let regionRows = '';
    const uniquePaints = new Map<string, { brand: string; name: string; code: string; hex: string; deltaE: number }>();
    
    for (const r of regions) {
      const m = r.match;
      regionRows += `<tr>
        <td class="col-id"><span class="marker">${r.id}</span></td>
        <td class="col-color">
          <div class="color-cell">
            <span class="swatch" style="background:${r.hex}"></span>
            <code>${r.hex}</code>
          </div>
        </td>
        <td class="col-region">${r.regionName ? escHtml(r.regionName) : '<span class="muted">—</span>'}</td>
        <td class="col-paint">
          ${m ? `
            <div class="paint-cell">
              <strong>${escHtml(m.manufacturer)}</strong><br>
              <span class="paint-name">${escHtml(m.name)}</span>
              <span class="paint-code">${escHtml(m.code)}</span>
            </div>
          ` : '<span class="muted">—</span>'}
        </td>
        <td class="col-delta">${m ? `<span class="delta-badge">${m.deltaE.toFixed(1)}</span>` : '—'}</td>
        <td class="col-note">${r.note ? escHtml(r.note) : '<span class="muted">—</span>'}</td>
      </tr>`;
      
      if (m) {
        const key = `${m.manufacturer}|${m.code}`;
        if (!uniquePaints.has(key)) {
          uniquePaints.set(key, { 
            brand: m.manufacturer, 
            name: m.name, 
            code: m.code,
            hex: m.hex,
            deltaE: m.deltaE
          });
        }
      }
    }

    const mfrGroups = new Map<string, typeof uniquePaints extends Map<string, infer V> ? V[] : never>();
    for (const [, p] of uniquePaints) {
      if (!mfrGroups.has(p.brand)) mfrGroups.set(p.brand, []);
      mfrGroups.get(p.brand)!.push(p);
    }

    let shopList = '';
    if (mfrGroups.size > 0) {
      let mfrIndex = 0;
      for (const [mfr, paints] of mfrGroups) {
        shopList += `
        <div class="mfr-group ${mfrIndex % 2 === 0 ? 'mfr-even' : 'mfr-odd'}">
          <div class="mfr-header">${escHtml(mfr)}</div>
          <div class="mfr-paints">
            ${paints.map(p => `
              <div class="paint-item">
                <span class="paint-swatch" style="background:${p.hex}"></span>
                <div class="paint-info">
                  <strong>${escHtml(p.name)}</strong>
                  <span class="paint-code">${escHtml(p.code)}</span>
                </div>
              </div>
            `).join('')}
          </div>
        </div>`;
        mfrIndex++;
      }
    }

    const html = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${escHtml(planName || 'Plano de Pintura')} - Mescla AI</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;600&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    
    :root {
      --bancada: #f6f4ef;
      --papel: #ffffff;
      --grafite: #1a1712;
      --hairline: #dcd6c9;
      --laca: #e8542c;
      --muted: #999;
    }
    
    body {
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      color: var(--grafite);
      background: var(--papel);
      line-height: 1.5;
      padding: 40px;
      max-width: 210mm;
      margin: 0 auto;
    }
    
    .doc-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding-bottom: 24px;
      border-bottom: 2px solid var(--grafite);
      margin-bottom: 32px;
    }
    
    .doc-title {
      flex: 1;
    }
    
    .doc-title h1 {
      font-size: 28px;
      font-weight: 700;
      letter-spacing: -0.5px;
      margin-bottom: 4px;
    }
    
    .doc-title .subtitle {
      font-size: 13px;
      color: var(--muted);
      font-weight: 500;
    }
    
    .doc-brand {
      text-align: right;
    }
    
    .doc-brand .logo {
      font-size: 20px;
      font-weight: 700;
      color: var(--laca);
      letter-spacing: -0.5px;
    }
    
    .doc-brand .tagline {
      font-size: 11px;
      color: var(--muted);
      margin-top: 2px;
    }
    
    .doc-meta {
      display: flex;
      gap: 24px;
      margin-bottom: 32px;
      font-size: 12px;
      color: var(--muted);
    }
    
    .doc-meta span {
      display: flex;
      align-items: center;
      gap: 4px;
    }
    
    .doc-meta strong {
      color: var(--grafite);
      font-weight: 600;
    }
    
    .image-section {
      background: var(--bancada);
      border: 1px solid var(--hairline);
      border-radius: 8px;
      padding: 24px;
      margin-bottom: 32px;
      text-align: center;
    }
    
    .image-section img {
      max-width: 100%;
      height: auto;
      border-radius: 4px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.08);
    }
    
    .image-caption {
      margin-top: 12px;
      font-size: 12px;
      color: var(--muted);
      font-style: italic;
    }
    
    h2 {
      font-size: 18px;
      font-weight: 700;
      margin-top: 40px;
      margin-bottom: 16px;
      padding-bottom: 8px;
      border-bottom: 1px solid var(--hairline);
    }
    
    .regions-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      margin-bottom: 32px;
    }
    
    .regions-table th {
      text-align: left;
      padding: 10px 12px;
      background: var(--bancada);
      font-weight: 600;
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      color: var(--grafite);
      border-bottom: 2px solid var(--hairline);
    }
    
    .regions-table td {
      padding: 12px;
      border-bottom: 1px solid var(--hairline);
      vertical-align: middle;
    }
    
    .regions-table tr:hover {
      background: var(--bancada);
    }
    
    .col-id { width: 50px; text-align: center; }
    .col-color { width: 100px; }
    .col-region { width: 120px; }
    .col-paint { width: 200px; }
    .col-delta { width: 60px; text-align: center; }
    .col-note { width: auto; }
    
    .marker {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 28px;
      height: 28px;
      background: var(--grafite);
      color: #fff;
      border-radius: 50%;
      font-weight: 700;
      font-size: 13px;
    }
    
    .swatch {
      display: inline-block;
      width: 24px;
      height: 24px;
      border-radius: 4px;
      border: 1px solid var(--hairline);
      flex-shrink: 0;
    }
    
    .color-cell {
      display: flex;
      align-items: center;
      gap: 8px;
    }
    
    .color-cell code {
      font-family: 'IBM Plex Mono', monospace;
      font-size: 12px;
      font-weight: 600;
    }
    
    .paint-cell strong {
      font-size: 13px;
      display: block;
    }
    
    .paint-name {
      font-size: 12px;
      color: var(--grafite);
    }
    
    .paint-code {
      font-family: 'IBM Plex Mono', monospace;
      font-size: 11px;
      color: var(--muted);
      margin-left: 6px;
    }
    
    .delta-badge {
      display: inline-block;
      padding: 3px 8px;
      background: var(--grafite);
      color: #fff;
      border-radius: 4px;
      font-family: 'IBM Plex Mono', monospace;
      font-size: 11px;
      font-weight: 600;
    }
    
    .muted {
      color: var(--muted);
      font-style: italic;
    }
    
    .shopping-section {
      margin-top: 40px;
    }
    
    .mfr-group {
      border: 1px solid var(--hairline);
      border-radius: 8px;
      margin-bottom: 16px;
      overflow: hidden;
    }
    
    .mfr-even { background: var(--papel); }
    .mfr-odd { background: var(--bancada); }
    
    .mfr-header {
      padding: 12px 16px;
      font-weight: 700;
      font-size: 15px;
      background: var(--grafite);
      color: #fff;
    }
    
    .mfr-paints {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 12px;
      padding: 16px;
    }
    
    .paint-item {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 8px;
      background: var(--papel);
      border-radius: 6px;
      border: 1px solid var(--hairline);
    }
    
    .paint-swatch {
      width: 32px;
      height: 32px;
      border-radius: 4px;
      border: 1px solid var(--hairline);
      flex-shrink: 0;
    }
    
    .paint-info {
      flex: 1;
      min-width: 0;
    }
    
    .paint-info strong {
      display: block;
      font-size: 12px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    
    .paint-info .paint-code {
      display: block;
      font-size: 11px;
    }
    
    .doc-footer {
      margin-top: 48px;
      padding-top: 24px;
      border-top: 1px solid var(--hairline);
      font-size: 11px;
      color: var(--muted);
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    
    .doc-footer .note {
      font-style: italic;
    }
    
    @media print {
      body {
        padding: 20mm;
        max-width: none;
      }
      
      .regions-table th {
        background: #f6f4ef !important;
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
      }
      
      .marker, .delta-badge, .mfr-header {
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
      }
      
      .mfr-paints {
        grid-template-columns: repeat(3, 1fr);
      }
    }
  </style>
</head>
<body>
  <header class="doc-header">
    <div class="doc-title">
      <h1>${escHtml(planName || 'Plano de Pintura')}</h1>
      <div class="subtitle">Guia de pintura para action figures</div>
    </div>
    <div class="doc-brand">
      <div class="logo">Mescla</div>
      <div class="tagline">Paint Match AI</div>
    </div>
  </header>
  
  <div class="doc-meta">
    <span><strong>${regions.length}</strong> regiões identificadas</span>
    <span><strong>${uniquePaints.size}</strong> tintas únicas</span>
    <span><strong>${mfrGroups.size}</strong> fabricantes</span>
    <span>Criado em <strong>${new Date().toLocaleDateString('pt-BR')}</strong></span>
  </div>
  
  <div class="image-section">
    <img src="${imgSrc}" alt="Action figure com marcações numeradas" />
    <div class="image-caption">Clique nas marcações numeradas para identificar cada região</div>
  </div>
  
  <h2>Detalhamento por Região</h2>
  <table class="regions-table">
    <thead>
      <tr>
        <th class="col-id">Nº</th>
        <th class="col-color">Cor</th>
        <th class="col-region">Região</th>
        <th class="col-paint">Tinta Equivalente</th>
        <th class="col-delta">ΔE</th>
        <th class="col-note">Observações</th>
      </tr>
    </thead>
    <tbody>
      ${regionRows}
    </tbody>
  </table>
  
  ${uniquePaints.size > 0 ? `
  <div class="shopping-section">
    <h2>Lista de Compras por Fabricante</h2>
    ${shopList}
  </div>
  ` : ''}
  
  <footer class="doc-footer">
    <div class="note">Documento gerado pelo Mescla AI - Paint Match</div>
    <div>${new Date().toLocaleString('pt-BR')}</div>
  </footer>
  
  <script>window.onload=()=>setTimeout(()=>window.print(),500);<\/script>
</body>
</html>`;

    const blob = new Blob([html], { type: 'text/html' });
    
    try {
      let filepath = await DialogService.SaveHTML();
      if (!filepath) return;
      if (!filepath.endsWith('.html')) filepath += '.html';
      
      const reader = new FileReader();
      reader.onload = async () => {
        const dataUrl = reader.result as string;
        const base64 = dataUrl.split(',')[1];
        
        await DialogService.SaveFileWithData(filepath, base64, '.html');
        toast('Documento exportado — abra no navegador para imprimir.');
      };
      reader.readAsDataURL(blob);
    } catch (e) {
      console.error('Erro ao salvar HTML:', e);
      toast('Erro ao salvar arquivo', 'error');
    }
  }

  // ── Export JSON ──
  async function exportJSON() {
    if (!hasImage || regions.length === 0) return;
    const data = {
      version: 2,
      name: planName || 'Plano de Pintura',
      imageData: imageDataUrl,
      regions: regions.map(r => ({
        x: r.x, y: r.y,
        r: r.r, g: r.g, b: r.b,
        hex: r.hex,
        regionName: r.regionName,
        note: r.note,
        match: r.match ? {
          paintId: r.match.paintId,
          name: r.match.name,
          manufacturer: r.match.manufacturer,
          code: r.match.code,
          hex: r.match.hex,
          deltaE: r.match.deltaE,
        } : null,
        recipe: r.recipe || null,
      })),
      exportedAt: new Date().toISOString(),
    };
    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    
    try {
      let filepath = await DialogService.SaveJSON();
      if (!filepath) return;
      if (!filepath.endsWith('.json')) filepath += '.json';
      
      const reader = new FileReader();
      reader.onload = async () => {
        const dataUrl = reader.result as string;
        const base64 = dataUrl.split(',')[1];
        
        await DialogService.SaveFileWithData(filepath, base64, '.json');
        toast('JSON exportado!');
      };
      reader.readAsDataURL(blob);
    } catch (e) {
      console.error('Erro ao salvar JSON:', e);
      toast('Erro ao salvar arquivo', 'error');
    }
  }

  // ── Arquivo .mesclaplan (legado) ──
  async function saveToFile() {
    if (!hasImage || regions.length === 0) return;
    await autoSave();
    toast('Plano salvo no banco de dados!');
  }

  async function openFile() {
    try {
      const [handle] = await (window as any).showOpenFilePicker({
        types: [
          { description: 'Mescla Plan', accept: { 'application/json': ['.mesclaplan', '.json'] } },
        ],
        multiple: false,
      });
      const file = await handle.getFile();
      const text = await file.text();
      loadPlanFromJSON(text);
    } catch (e: any) {
      if (e?.name === 'AbortError') return;
      planFileInputEl?.click();
    }
  }

  function loadPlanFromJSON(raw: string) {
    try {
      const data = JSON.parse(raw);
      if (!data.regions) {
        toast('Arquivo inválido.', 'error');
        return;
      }

      const doLoad = (imgSrc: string) => {
        const img = new Image();
        img.onload = () => {
          image = img;
          hasImage = true;
          zoom = 1; panX = 0; panY = 0;
          planName = data.name || '';
          imageDataUrl = imgSrc;
          currentPlanId = 0;
          nextId = 1;
          regions = (data.regions as any[]).map(r => ({
            id: nextId++,
            x: r.x, y: r.y,
            r: r.r, g: r.g, b: r.b,
            hex: r.hex,
            regionName: r.regionName || '',
            note: r.note || '',
            targetMfrId: r.targetMfrId || 0,
            match: r.match || null,
            allMatches: r.match ? [r.match] : [],
            recipe: r.recipe || null,
          }));
          undoStack = [];
          redoStack = [];
          toast(`Plano carregado: ${regions.length} regiões`);
          requestAnimationFrame(() => zoomFit());
        };
        img.src = imgSrc;
      };

      if (data.imageData) {
        doLoad(data.imageData);
      } else {
        toast('Arquivo sem imagem.', 'error');
      }
    } catch {
      toast('Erro ao ler arquivo.', 'error');
    }
  }

  function loadFromFile(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => loadPlanFromJSON(reader.result as string);
    reader.readAsText(file);
    input.value = '';
  }

  // ── Export menu handler ──
  // ── Sidebar collapse ──
  let sidebarCollapsed = $state(false);
  let viewportWidth = $state(typeof window !== 'undefined' ? window.innerWidth : 1024);

  function updateViewportWidth() {
    viewportWidth = window.innerWidth;
    if (viewportWidth < 768) sidebarCollapsed = true;
  }

  $effect(() => {
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', updateViewportWidth);
      updateViewportWidth();
      return () => window.removeEventListener('resize', updateViewportWidth);
    }
  });

  let isCompact = $derived(viewportWidth < 768);
  let isRegular = $derived(viewportWidth >= 768 && viewportWidth < 1024);

</script>

<svelte:window onpaste={onPaste} onkeydown={onKeyDown} />

<div class="planner-layout" class:compact={isCompact} class:regular={isRegular}>
  <!-- Coluna principal: canvas -->
  <div class="planner-main">
    {#if !hasImage}
      <div
        class="dropzone"
        role="button"
        tabindex={0}
        ondragover={(e: DragEvent) => e.preventDefault()}
        ondrop={onDrop}
      >
        <div class="dropzone-inner">
          <Icon name="pipette" size={48} />
          <p class="dropzone-title">Arraste uma imagem do action figure aqui</p>
          <p class="dropzone-sub">ou <button class="link-btn" onclick={() => fileInputEl?.click()}>selecione um arquivo</button> ou <kbd>Ctrl+V</kbd></p>
          <input type="file" accept="image/*" class="hidden" bind:this={fileInputEl} onchange={onFileInput} />
        </div>
        <button class="open-plan-btn" onclick={openPlanGrid}>
          <Icon name="box" size={16} /> Abrir plano salvo
        </button>
      </div>
    {:else}
      <div class="canvas-wrap">
        <canvas
          bind:this={canvasEl}
          onmousedown={onCanvasMouseDown}
          onmousemove={onCanvasMouseMove}
          onmouseup={onCanvasMouseUp}
          onmouseleave={() => { dragging = false; draggingPickId = null; hideLoupe(); }}
          onwheel={onCanvasWheel}
          ontouchstart={onTouchStart}
          ontouchmove={onTouchMove}
          ontouchend={onTouchEnd}
          style:cursor={mode === 'pick' ? 'crosshair' : mode === 'view' ? 'grab' : 'default'}
        ></canvas>
      </div>

      <FloatingToolbar
        {mode}
        canUndo={undoStack.length > 0}
        canRedo={redoStack.length > 0}
        {hasImage}
        hasRegions={regions.length > 0}
        onModeChange={(m) => { mode = m; }}
        onUndo={undo}
        onRedo={redo}
        onZoomIn={zoomIn}
        onZoomOut={zoomOut}
        onZoomFit={zoomFit}
        onExportPNG={exportPNG}
        onExportPDF={exportPDF}
        onExportJSON={exportJSON}
        onSave={saveToFile}
        onOpen={openPlanGrid}
        {savedIndicator}
      />

      {#if pickedInfo}
        <div class="pick-banner">
          <span class="swatch-dot" style="background:{pickedInfo.hex}"></span>
          <strong>{pickedInfo.hex}</strong>
          <span>→ {pickedInfo.brand} {pickedInfo.name}</span>
        </div>
      {/if}
    {/if}
  </div>

  <!-- Sidebar / Bottom sheet -->
  {#if !isCompact}
    <div class="planner-sidebar" class:collapsed={sidebarCollapsed}>
      {#if !sidebarCollapsed}
        <div class="sidebar-header">
          <div class="sidebar-title">
            <h3>Regiões ({regions.length})</h3>
          </div>
          <div class="sidebar-actions">
            <button class="icon-btn sm" onclick={() => { sidebarCollapsed = true; }} title="Recolher sidebar">
              <Icon name="sidebar" size={16} />
            </button>
          </div>
        </div>

        {#if regions.length === 0}
          <div class="empty-state">
            <Icon name="droplet" size={32} />
            <p>Clique na imagem para identificar cores.</p>
            <p class="meta">Use o eyedropper (I) ou toque longo no tablet.</p>
          </div>
        {:else}
          <div class="region-list">
            {#each [...regions].reverse() as r (r.id)}
              <RegionCard
                region={r}
                {manufacturers}
                isStockMatch={isStockMatch(r)}
                selected={selectedRegionId === r.id}
                onSelect={selectRegion}
                onRemove={removeRegion}
                onUpdateName={updateRegionName}
                onUpdateNote={updateRegionNote}
                onChangeMfr={changeRegionMfr}
              />
            {/each}
          </div>
        {/if}
      {:else}
        <button class="sidebar-expand" onclick={() => { sidebarCollapsed = false; }} title="Expandir sidebar">
          <Icon name="sidebar" size={20} />
        </button>
      {/if}
    </div>
  {/if}

  <!-- Compact: bottom bar -->
  {#if isCompact && hasImage}
    <div class="compact-bar">
      <button class="compact-btn" onclick={undo} disabled={undoStack.length === 0}>
        <Icon name="undo" size={18} />
      </button>
      <button class="compact-btn" class:active={mode === 'pick'} onclick={() => mode = 'pick'}>
        <Icon name="pipette" size={18} />
      </button>
      <button class="compact-btn" onclick={saveToFile} disabled={regions.length === 0}>
        <Icon name="save" size={18} />
      </button>
      <button class="compact-btn" onclick={openPlanGrid}>
        <Icon name="box" size={18} />
      </button>
      <span class="compact-count">{regions.length}</span>
    </div>
  {/if}
</div>

{#if showPlanGrid}
  <PlanListGrid
    plans={planGridData}
    loading={planGridLoading}
    onSelect={loadPlanFromDB}
    onDelete={deletePlanFromDB}
    onClose={() => { showPlanGrid = false; }}
  />
{/if}

<PrecisionLoupe
  visible={loupeVisible}
  x={loupeX}
  y={loupeY}
  screenX={loupeScreenX}
  screenY={loupeScreenY}
  hex={loupeHex}
  {zoom}
/>

<input type="file" accept=".mesclaplan,.json,application/json" class="hidden" bind:this={planFileInputEl} onchange={loadFromFile} />

<style>
  .planner-layout {
    display: flex;
    gap: 0;
    height: calc(100vh - 64px);
    background: var(--bancada);
  }

  .planner-layout.compact {
    flex-direction: column;
  }

  .planner-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
  }

  .planner-sidebar {
    width: 340px;
    border-left: 1px solid var(--hairline);
    background: var(--papel);
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    transition: width 0.2s;
  }

  .planner-sidebar.collapsed {
    width: 48px;
    align-items: center;
    padding-top: 8px;
  }

  .regular .planner-sidebar { width: 280px; }

  /* ── Dropzone ── */
  .dropzone {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    margin: 24px;
    border: 2px dashed var(--hairline);
    border-radius: var(--radius-surface);
    cursor: pointer;
    position: relative;
  }
  .dropzone-inner {
    text-align: center;
    padding: 48px;
  }
  .dropzone-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--grafite);
    margin: 12px 0 4px;
  }
  .dropzone-sub {
    font-size: 14px;
    color: var(--grafite);
    opacity: 0.6;
  }
  .dropzone-sub kbd {
    background: var(--bancada);
    border: 1px solid var(--hairline);
    border-radius: 4px;
    padding: 1px 6px;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 12px;
  }
  .open-plan-btn {
    position: absolute;
    bottom: 16px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    background: var(--papel);
    cursor: pointer;
    font-size: 13px;
    color: var(--grafite);
  }
  .open-plan-btn:hover { background: var(--bancada); }

  /* ── Canvas ── */
  .canvas-wrap {
    flex: 1;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bancada);
    touch-action: none;
    position: relative;
  }
  canvas {
    touch-action: none;
    display: block;
  }
  /* ── Pick banner ── */
  .pick-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: var(--papel);
    border-top: 1px solid var(--hairline);
    font-size: 14px;
    font-family: 'IBM Plex Mono', monospace;
  }

  /* ── Export menu wrapper ── */
  /* ── Sidebar ── */
  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid var(--hairline);
  }
  .sidebar-title h3 {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    white-space: nowrap;
  }
  .sidebar-actions {
    display: flex;
    gap: 4px;
  }
  .sidebar-expand {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--grafite);
    border-radius: var(--radius-surface);
  }
  .sidebar-expand:hover { background: var(--bancada); }

  .region-list {
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-y: auto;
    flex: 1;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 32px 16px;
    gap: 8px;
    text-align: center;
    color: var(--grafite);
    opacity: 0.5;
  }
  .empty-state p { margin: 0; font-size: 13px; }
  .empty-state .meta { font-size: 11px; }

  /* ── Shared ── */
  .swatch-dot {
    display: inline-block;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    border: 1px solid var(--hairline);
    flex-shrink: 0;
  }
  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    background: var(--papel);
    cursor: pointer;
    font-size: 14px;
    color: var(--grafite);
    line-height: 1;
    flex-shrink: 0;
  }
  .icon-btn.sm { width: 28px; height: 28px; border: none; }
  .icon-btn:hover { background: var(--bancada); }
  .link-btn {
    background: none;
    border: none;
    color: var(--laca);
    cursor: pointer;
    font-size: inherit;
    text-decoration: underline;
  }
  .hidden { display: none; }

  /* ── Compact bottom bar ── */
  .compact-bar {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 8px;
    background: var(--papel);
    border-top: 1px solid var(--hairline);
  }
  .compact-btn {
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
  .compact-btn:hover { background: var(--bancada); }
  .compact-btn.active { background: var(--laca); color: #fff; }
  .compact-btn:disabled { opacity: 0.3; }
  .compact-count {
    margin-left: auto;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 13px;
    color: var(--grafite);
  }
</style>
