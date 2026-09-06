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
  import { t, decimal, type DictKey } from '../i18n.svelte';
  import { toast } from '../toast.svelte';
  import { nav } from '../nav.svelte';
  import {
    lerRascunho, agendarGravacao, apagarRascunho,
    lerPlanoAtivo, gravarPlanoAtivo, limparPlanoAtivo, estadoAutoSave,
    type Rascunho, type RegiaoRascunho, type EstadoAutoSave,
  } from '../planner/rascunho';
  import {
    validarPlano, salvarPlano, carregarPlano, baixarRelatorio, PlanoError,
    type PlanoDTO, type RegiaoDTO,
  } from '../services/plans';

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
    /** Tinta que veio do plano salvo/rascunho. `result` nasce nulo ao
     *  hidratar e só é preenchido quando o recálculo termina — sem esta
     *  cópia, salvar antes disso gravaria tinta vazia por cima da boa. */
    salvo: RegiaoSalva | null;
  }

  interface RegiaoSalva {
    paintId: number | null;
    paintBrand: string;
    paintName: string;
    paintCode: string;
    deltaE: number;
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

  // ── rf-07: plano da peça (nome, salvar, auto save local) ──
  let planId: number | null = $state(null);
  let planName = $state('');
  /** data URL da foto — guardada à parte de `image` (HTMLImageElement) porque
   *  o rascunho e o PlanoDTO precisam da string crua, não do bitmap. */
  let imageDataUrl = $state('');

  type SaveState = 'idle' | 'saving' | 'saved' | 'error';
  let saveState: SaveState = $state('idle');
  let savedAtLabel = $state('');
  /** Chave i18n do erro de Salvar — nunca texto cru do servidor (RN11). */
  let saveErrorKey: DictKey | null = $state(null);
  /** Chave i18n do erro de validação do formulário (CA11/CA13/CA17-CA20). */
  let validationErrorKey: DictKey | null = $state(null);

  let draftBannerVisible = $state(false);
  /** Reflete `estadoAutoSave()` (RN8) — só o módulo de rascunho decide a
   *  escada; a tela apenas relê depois que a gravação (debounce) já rodou. */
  let autoSaveStatus: EstadoAutoSave = $state('ligado');

  /** RN5: T2 fica sempre montada — hidrata uma única vez na TRANSIÇÃO para
   *  'plano', nunca de novo. `hidratando` impede reentrância do próprio
   *  `$effect`; `hidratado` é o que trava `marcarAlteracao` até a hidratação
   *  terminar (maior risco da fatia: auto save sobrescrevendo rascunho bom
   *  com estado vazio no boot). */
  let hidratando = false;
  let hidratado = $state(false);
  /** Marca que o usuário mexeu na tela enquanto o `GET` da hidratação estava
   *  em voo. Aplicar o plano do servidor por cima apagaria esse trabalho, e
   *  não há rascunho de resgate (`marcarAlteracao` ainda está travado). */
  let alteradoDuranteHidratacao = false;
  /** Sobe a cada Salvar concluído e a cada alteração de conteúdo — deixa o
   *  código distinguir "nada mudou desde o POST" de "mudou durante o POST". */
  let salvamentoSeq = 0;
  let alteracaoSeq = 0;
  /** rf-08/RN11: valor de `alteracaoSeq` no último Salvar bem-sucedido sem
   *  edição concorrente (o ramo que apaga o rascunho). Reaproveita os
   *  contadores do rf-07 em vez de inventar flag nova — "rascunho pendente"
   *  é `alteracaoSeq` ter avançado depois desse ponto, ou o banner de
   *  rascunho restaurado ainda estar visível. */
  let alteracaoSeqSalva = $state(0);

  // ── rf-08: exportar relatório (PDF/PNG) ──
  let reportFormat: 'pdf' | 'png' = $state('pdf');
  let reportGenerating = $state(false);
  /** Chave i18n do estado/erro da exportação — nunca texto cru do servidor (CA18). */
  let reportErrorKey: DictKey | null = $state(null);

  let planNamePlaceholder = $derived.by(() => {
    const d = new Date();
    const dd = String(d.getDate()).padStart(2, '0');
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    return t('planNamePh', { data: `${dd}/${mm}` });
  });

  $effect(() => {
    if (nav.tab === 'plano' && !hidratando) {
      hidratando = true;
      void hidratarT2();
    }
  });

  $effect(() => {
    if (manufacturerId === null && manufacturers.length > 0) manufacturerId = manufacturers[0].id;
  });

  let selectedRegion = $derived(regions.find(r => r.id === selectedId) ?? null);
  let paintedCount = $derived(regions.filter(r => r.painted).length);
  let progressPct = $derived(regions.length ? Math.round((paintedCount / regions.length) * 100) : 0);

  let plannerTitle = $derived(
    hasImage ? t('progress', { a: paintedCount, b: regions.length }) : t('t2EmptyTitle')
  );

  // rf-08/RN11: plano nunca salvo ou com rascunho local pendente bloqueia a exportação.
  let rascunhoPendente = $derived(draftBannerVisible || alteracaoSeq !== alteracaoSeqSalva);
  let exportDisabled = $derived(planId == null || rascunhoPendente || reportGenerating);

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
      imageDataUrl = src;
      marcarAlteracao();
    };
    img.src = src;
  }

  /** Restaura a foto de um plano/rascunho carregado — sem zerar regiões
   *  (quem chama já as restaura à parte) e sem contar como alteração de
   *  conteúdo (hidratação, não input do usuário). */
  function loadImageForHydration(src: string): Promise<void> {
    return new Promise(resolve => {
      const img = new Image();
      img.onload = () => {
        image = img;
        resolve();
      };
      img.onerror = () => resolve();
      img.src = src;
    });
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
      salvo: null,
    };
    regions = [...regions, region];
    selectedId = region.id;
    draw();
    await computeRegion(region.id);
    marcarAlteracao();
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
    const epoca = salvamentoSeq;
    // marcarAlteracao só depois que o laço de recálculo termina — nunca
    // antes, e nunca por região (um `$effect` sobre `regions` dispararia a
    // cada escrita assíncrona de computeRegion, o que a spec proíbe).
    await Promise.all(regions.map(r => computeRegion(r.id)));
    // Um Salvar concluiu durante o recálculo: marcar agora ressuscitaria um
    // rascunho para um plano que já está salvo.
    if (salvamentoSeq === epoca) marcarAlteracao();
  }

  function selectRegion(id: number) {
    selectedId = id;
    draw();
  }

  function togglePainted(r: PlannerRegion) {
    r.painted = !r.painted;
    marcarAlteracao();
  }

  function removeSelected() {
    if (selectedId == null) return;
    regions = regions.filter(r => r.id !== selectedId);
    selectedId = null;
    draw();
    marcarAlteracao();
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

  // ── rf-07: mapeamento tela → rascunho/DTO ──
  // PlannerRegion não tem `regionName`/`note` — deriva-se do rótulo que a
  // tela já usa (regionName(i)) e envia-se `note: ''` (não inventa campo).
  function mapRegionCommon(r: PlannerRegion, i: number) {
    const base = { x: r.x, y: r.y, r: r.r, g: r.g, b: r.b, hex: r.hex, regionName: regionName(i), note: '' };
    if (!r.result) {
      // Recálculo ainda não terminou (ou a API está fora): preserva a tinta
      // que veio do plano salvo em vez de gravar vazio por cima.
      const s = r.salvo;
      return {
        ...base,
        paintId: s?.paintId ?? null,
        paintBrand: s?.paintBrand ?? '',
        paintName: s?.paintName ?? '',
        paintCode: s?.paintCode ?? '',
        deltaE: s?.deltaE ?? 0,
      };
    }
    const ing = r.result.ingredients ?? [];
    return {
      ...base,
      paintId: ing.length === 1 ? ing[0].paintId : null,
      paintBrand: r.result.targetManufacturer ?? '',
      paintName: ing.length === 1 ? ing[0].name : ing.map(x => x.name).join(' + '),
      paintCode: ing.length === 1 ? ing[0].code : '',
      deltaE: r.result.deltaE ?? 0,
    };
  }

  function regiaoParaRascunho(r: PlannerRegion, i: number): RegiaoRascunho {
    return { ...mapRegionCommon(r, i), painted: r.painted };
  }

  function regiaoParaDTO(r: PlannerRegion, i: number): RegiaoDTO {
    return { ...mapRegionCommon(r, i), painted: r.painted ? 1 : 0 };
  }

  function buildPlanoDTO(): PlanoDTO {
    return {
      id: planId ?? undefined,
      name: planName,
      imageData: imageDataUrl,
      selectedManufacturerId: manufacturerId,
      regions: regions.map((r, i) => regiaoParaDTO(r, i)),
    };
  }

  // ── rf-07: auto save local (RN3/RN4) ──
  /** Chamada explícita nos mutadores de CONTEÚDO — nunca em zoom/pan/view
   *  (RN4/CA7) e nunca antes da hidratação terminar (RN5). */
  function marcarAlteracao() {
    if (!hidratado) {
      alteradoDuranteHidratacao = true;
      return;
    }
    const payload: Rascunho = {
      planId,
      name: planName,
      imageData: imageDataUrl,
      regions: regions.map((r, i) => regiaoParaRascunho(r, i)),
      selectedManufacturerId: manufacturerId,
      salvoEm: new Date().toISOString(),
    };
    alteracaoSeq++;
    agendarGravacao(payload);
    // A escada de cota (RN8) só se resolve dentro do módulo depois do
    // debounce de 1500 ms — relê um pouco depois para refletir na tela.
    setTimeout(() => { autoSaveStatus = estadoAutoSave(); }, 1600);
  }

  // ── rf-07: hidratar / aplicar plano carregado ──
  function resetToEmpty() {
    planId = null;
    planName = '';
    imageDataUrl = '';
    image = null;
    hasImage = false;
    regions = [];
    nextId = 1;
    selectedId = null;
    ownedPaints = new Set();
    manufacturerId = null;
  }

  /** Aplica um `PlanoDTO` (do servidor ou remontado do rascunho) ao estado
   *  da tela. Nunca chama `marcarAlteracao` — é hidratação, não input. */
  function applyLoadedPlan(plano: PlanoDTO, recalcular = true) {
    planId = plano.id ?? null;
    planName = plano.name;
    imageDataUrl = plano.imageData || '';
    manufacturerId = plano.selectedManufacturerId ?? null;
    selectedId = null;
    ownedPaints = new Set();
    nextId = 1;
    regions = plano.regions.map((rd): PlannerRegion => ({
      id: nextId++,
      x: rd.x, y: rd.y, r: rd.r, g: rd.g, b: rd.b, hex: rd.hex,
      painted: rd.painted === 1,
      result: null,
      computing: false,
      salvo: {
        paintId: rd.paintId ?? null,
        paintBrand: rd.paintBrand,
        paintName: rd.paintName,
        paintCode: rd.paintCode,
        deltaE: rd.deltaE,
      },
    }));
    if (imageDataUrl) {
      hasImage = true;
      void loadImageForHydration(imageDataUrl);
    } else {
      hasImage = false;
      image = null;
    }
    // CA8: restaurar rascunho não consulta o servidor — a tinta já veio no
    // próprio rascunho. Só o plano vindo do banco recalcula.
    if (recalcular && manufacturerId != null) {
      for (const r of regions) void computeRegion(r.id);
    }
  }

  /** RN5 — entrada em T2: rascunho vence; senão plano ativo; senão vazia. */
  async function hidratarT2() {
    try {
      const { rascunho, corrompido, truncado } = lerRascunho();
      if (corrompido) {
        toast(t('draftCorrupted'), 'error');
      }
      if (rascunho) {
        // RN8 cortou a foto para caber na cota: a imagem continua no banco,
        // então busca de lá em vez de reabrir o plano sem foto — e sem ela o
        // Salvar seguinte apagaria a foto do plano no servidor.
        let fotoDoServidor = '';
        if (!rascunho.imageData && rascunho.planId != null) {
          try {
            const salvo = await carregarPlano(rascunho.planId);
            fotoDoServidor = salvo.imageData;
          } catch {
            toast(t('draftNoPhoto'), 'error');
          }
        }
        applyLoadedPlan({
          id: rascunho.planId ?? undefined,
          name: rascunho.name,
          imageData: rascunho.imageData || fotoDoServidor,
          selectedManufacturerId: rascunho.selectedManufacturerId,
          regions: rascunho.regions.map((rr): RegiaoDTO => ({
            x: rr.x, y: rr.y, r: rr.r, g: rr.g, b: rr.b, hex: rr.hex,
            regionName: rr.regionName, note: rr.note,
            paintId: rr.paintId, paintBrand: rr.paintBrand,
            paintName: rr.paintName, paintCode: rr.paintCode,
            deltaE: rr.deltaE, painted: rr.painted ? 1 : 0,
          })),
        }, false);
        draftBannerVisible = true;
        // CAN8: reaproveita a mensagem de teto de regiões — mesmo limite,
        // mesmo aviso.
        if (truncado) toast(t('errRegionsMax'), 'error');
        return;
      }
      const idAtivo = lerPlanoAtivo();
      if (idAtivo != null) {
        try {
          const plano = await carregarPlano(idAtivo);
          // O usuário mexeu na tela enquanto o GET vinha: o trabalho dele
          // vence o plano do servidor, e vira rascunho na próxima marcação.
          if (!alteradoDuranteHidratacao) applyLoadedPlan(plano);
        } catch (e) {
          limparPlanoAtivo();
          if (!alteradoDuranteHidratacao) resetToEmpty();
          const key = e instanceof PlanoError && e.code === 'nao-encontrado' ? 'planNotFound' : 'planSaveError';
          toast(t(key), 'error');
        }
      }
    } finally {
      hidratado = true;
    }
  }

  // ── rf-07: descartar rascunho (RN6) ──
  function onDiscardDraftClick() {
    if (!window.confirm(t('discardDraftConfirm'))) return;
    void discardDraftAndReload();
  }

  async function discardDraftAndReload() {
    if (planId != null) {
      // Nunca apaga antes de ter o plano em mãos — falhando, o rascunho
      // permanece intacto.
      try {
        const plano = await carregarPlano(planId);
        applyLoadedPlan(plano);
        apagarRascunho();
      } catch (e) {
        if (e instanceof PlanoError && e.code === 'nao-encontrado') {
          // Plano apagado no servidor: manter o rascunho deixaria o botão
          // Descartar travado para sempre. Volta à tela vazia.
          apagarRascunho();
          limparPlanoAtivo();
          resetToEmpty();
          toast(t('planNotFound'), 'error');
        } else {
          saveState = 'error';
          saveErrorKey = 'planSaveError';
          return;
        }
      }
    } else {
      apagarRascunho();
      resetToEmpty();
    }
    draftBannerVisible = false;
    // Sem isto, `rascunhoPendente` fica verdadeiro para sempre depois de um
    // descarte e a ação de exportar trava até o próximo Salvar.
    alteracaoSeqSalva = alteracaoSeq;
  }

  // ── rf-07: Salvar (CA1-CA23, RN10, CAN1-3) ──
  async function handleSave() {
    if (saveState === 'saving') return; // RN10/CAN3: um Salvar por vez
    const dto = buildPlanoDTO();
    const erro = validarPlano(dto);
    if (erro) {
      // validarPlano devolve uma chave i18n como string solta (plans.ts não
      // é arquivo desta fatia) — as chaves usadas são sempre as da spec.
      validationErrorKey = erro as DictKey;
      return;
    }
    validationErrorKey = null;
    saveErrorKey = null;
    saveState = 'saving';
    const seqNoEnvio = alteracaoSeq;
    try {
      const { plano, suspeitaD003 } = await salvarPlano(dto);
      // O id vem para o estado mesmo na suspeita de D-003: o servidor já
      // commitou, e sem adotá-lo cada nova tentativa criaria outro plano.
      if (plano.id != null) {
        planId = plano.id;
        gravarPlanoAtivo(plano.id);
      }
      if (suspeitaD003) {
        // RN1/RN7/CA16: não apaga o rascunho.
        saveState = 'error';
        saveErrorKey = 'planSaveError';
        return;
      }
      salvamentoSeq++;
      if (alteracaoSeq === seqNoEnvio) {
        apagarRascunho();
        alteracaoSeqSalva = seqNoEnvio;
      } else {
        // O usuário editou durante o POST: apagar o rascunho perderia essa
        // edição, que não entrou no plano enviado. Regrava com o estado atual.
        marcarAlteracao();
      }
      draftBannerVisible = false;
      const agora = new Date();
      savedAtLabel = `${String(agora.getHours()).padStart(2, '0')}:${String(agora.getMinutes()).padStart(2, '0')}`;
      saveState = 'saved';
    } catch (e) {
      saveState = 'error';
      saveErrorKey = e instanceof PlanoError && e.code === 'nao-encontrado' ? 'planNotFound' : 'planSaveError';
    }
  }

  // ── rf-08: exportar relatório (CA16-CA18, CAN1, CAN8) ──
  async function handleExport() {
    if (reportGenerating) return; // CAN8: um download por vez
    if (planId == null || rascunhoPendente) {
      reportErrorKey = 'exportSaveFirst';
      return;
    }
    // CAN1: o id só pode vir do estado interno da tela, nunca de entrada do
    // usuário — checagem defensiva antes de montar a URL em plans.ts.
    if (!Number.isInteger(planId) || planId <= 0) {
      reportErrorKey = 'exportInvalid';
      return;
    }
    reportErrorKey = null;
    reportGenerating = true;
    try {
      const { blob, filename } = await baixarRelatorio(planId, reportFormat);
      // RN12: fetch + blob + âncora temporária — nunca navegação direta.
      const url = URL.createObjectURL(blob);
      try {
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
      } finally {
        URL.revokeObjectURL(url);
      }
    } catch (e) {
      if (e instanceof PlanoError && e.code === 'nao-encontrado') {
        reportErrorKey = 'planNotFound';
      } else if (e instanceof PlanoError && e.code === 'invalido') {
        reportErrorKey = 'exportInvalid';
      } else {
        reportErrorKey = 'exportFailed';
      }
    } finally {
      reportGenerating = false;
    }
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

      <div style="flex-shrink: 0; padding: 16px 20px; border-top: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 10px;">
        {#if draftBannerVisible}
          <div role="status" aria-live="polite" style="display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 10px 12px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: var(--color-accent-panel); font-size: 13.5px; color: var(--color-text);">
            <span>{t('draftRestored')}</span>
            <button
              class="t2-discard-btn"
              style="min-width: 44px; min-height: 44px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); font-family: inherit; font-size: 13px; cursor: pointer;"
              onclick={onDiscardDraftClick}
            >{t('discardDraftBtn')}</button>
          </div>
        {/if}

        {#if autoSaveStatus === 'sem-foto'}
          <div aria-live="polite" style="padding: 8px 12px; border-radius: 8px; background: var(--color-raised); font-size: 13px; color: var(--color-neutral-400);">
            <i class="ph ph-warning" style="margin-right: 6px;"></i>{t('draftNoPhoto')}
          </div>
        {:else if autoSaveStatus === 'desligado'}
          <div role="alert" style="padding: 8px 12px; border-radius: 8px; border: 1px solid var(--color-neutral-800); background: var(--color-raised); font-size: 13px; color: var(--color-neutral-300);">
            <i class="ph ph-warning-circle" style="margin-right: 6px;"></i>{t('autosaveOff')}
          </div>
        {/if}

        <label for="plan-name-input" class="section-label" style="margin: 0;">{t('planNameLabel')}</label>
        <input
          id="plan-name-input"
          type="text"
          bind:value={planName}
          oninput={marcarAlteracao}
          placeholder={planNamePlaceholder}
          style="height: 44px; padding: 0 12px; border: 1px solid var(--color-field-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-family: inherit; font-size: 14px;"
        />

        {#if validationErrorKey}
          <div role="alert" style="font-size: 13px; color: var(--color-neutral-300);">
            <i class="ph ph-warning-circle" style="margin-right: 6px;"></i>{t(validationErrorKey)}
          </div>
        {/if}

        <button
          class="t2-save-plan-btn"
          style="display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;"
          disabled={saveState === 'saving'}
          onclick={handleSave}
        >
          {#if saveState === 'saving'}
            <Spinner size={18} label={t('planSaving')} />
          {:else}
            <i class="ph ph-floppy-disk" style="font-size: 19px;"></i>
          {/if}
          {t('savePlanBtn')}
        </button>

        <div aria-live="polite" style="min-height: 16px; font-size: 12.5px; color: var(--color-neutral-500);">
          {#if saveState === 'saving'}{t('planSaving')}
          {:else if saveState === 'saved'}{t('planSavedAt', { hora: savedAtLabel })}
          {:else if saveState === 'error' && saveErrorKey}{t(saveErrorKey)}
          {/if}
        </div>
      </div>

      <div style="flex-shrink: 0; padding: 16px 20px; border-top: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 10px;">
        <p class="section-label" style="margin: 0;">{t('exportReportBtn')}</p>
        <fieldset style="display: flex; align-items: center; gap: 18px; margin: 0; padding: 0; border: none;">
          <legend style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{t('exportFormatLabel')}</legend>
          <label style="display: inline-flex; align-items: center; gap: 8px; min-width: 44px; min-height: 44px; font-size: 14px; color: var(--color-text); cursor: pointer;">
            <input type="radio" name="report-format" value="pdf" bind:group={reportFormat} style="width: 20px; height: 20px; accent-color: var(--color-accent);" />
            {t('exportPdf')}
          </label>
          <label style="display: inline-flex; align-items: center; gap: 8px; min-width: 44px; min-height: 44px; font-size: 14px; color: var(--color-text); cursor: pointer;">
            <input type="radio" name="report-format" value="png" bind:group={reportFormat} style="width: 20px; height: 20px; accent-color: var(--color-accent);" />
            {t('exportPng')}
          </label>
        </fieldset>

        <button
          class="t2-export-btn"
          style="display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; min-height: 44px; height: 52px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
          disabled={exportDisabled}
          onclick={handleExport}
        >
          {#if reportGenerating}
            <Spinner size={18} label={t('exportGenerating')} />
          {:else}
            <i class="ph ph-file-arrow-down" style="font-size: 18px;"></i>
          {/if}
          {t('exportReportBtn')}
        </button>

        <div aria-live="polite" style="min-height: 16px; font-size: 12.5px; color: var(--color-neutral-500);">
          {#if reportGenerating}{t('exportGenerating')}
          {:else if reportErrorKey}{t(reportErrorKey)}
          {:else if planId == null || rascunhoPendente}{t('exportSaveFirst')}
          {/if}
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

  .t2-save-plan-btn:hover {
    background: var(--color-accent-hover);
  }

  .t2-save-plan-btn:disabled {
    opacity: 0.45;
    pointer-events: none;
  }

  .t2-discard-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-export-btn:hover {
    background: var(--color-accent-hover);
  }

  .t2-export-btn:disabled {
    opacity: 0.45;
    pointer-events: none;
  }
</style>
