<script lang="ts">
  // T2 — Plano da peça (rf-04). Reescrita para fidelidade ao protótipo
  // (D-001/Epic D) — oráculo tests/fixtures/mockup/t2-plano.html.
  //
  // rf-16: T2 entra por uma LISTA de projetos (ProjetosLista); abrir um
  // projeto leva ao editor. No editor: título no cabeçalho, Salvar/Exportar
  // numa barra de ícones junto do zoom, e cada região selecionada expande o
  // próprio card com a conta da cor e um ajuste de fornecedor só dela.
  //
  // Foto com pins numerados por região (desenhados em <canvas>, com zoom por
  // pinça e pan). Painel lateral: fabricante do projeto ("Pintar com o que eu
  // tenho", Combobox) que recalcula as regiões que seguem o projeto, lista de
  // regiões com estado pintada/a pintar e "Tintas da peça".
  import { tick, untrack } from 'svelte';
  import Header from '../components/Header.svelte';
  import Spinner from '../components/Spinner.svelte';
  import Combobox from '../components/Combobox.svelte';
  import ProjetosLista from '../components/ProjetosLista.svelte';
  import {
    fitView, toImage, toScreen, zoomAround, zoomPercent, type View,
  } from '../planner/viewport';
  import {
    suggestRecipeForColor, melhorDeltaE, ehUniversoVazio,
    type EquivalentRecipe, type UniversoBusca,
  } from '../services/engine';
  import { allManufacturers } from '../services/catalog';
  import { stock, estoqueAssentado } from '../services/stock.svelte';
  import { verdictKeys } from '../ui';
  import { t, decimal, type DictKey } from '../i18n.svelte';
  import { toast } from '../toast.svelte';
  import { nav } from '../nav.svelte';
  import {
    lerRascunho, agendarGravacao, apagarRascunho, gravarPendente, estadoAutoSave,
    type Rascunho, type RegiaoRascunho, type AbaRascunho, type EstadoAutoSave,
  } from '../planner/rascunho';
  import {
    validarPlano, validarImagemDaAba, salvarPlano, carregarPlano, baixarRelatorio, listarPlanos, excluirPlano,
    tetoArquivoImagem, TIPOS_IMAGEM_ACEITOS, PlanoError,
    type PlanoDTO, type RegiaoDTO, type AbaDTO, type ErroValidacaoPlano,
    type IngredienteDTO, type ResumoPlanoDTO,
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
    /** Última mistura boa (do plano salvo, do rascunho ou do último cálculo
     *  que deu certo). rf-16 RN26: sem `result` na hora de gravar, é isto
     *  que vai — nunca mistura vazia por cima da boa. */
    salvo: RegiaoSalva | null;
    /** rf-16 RN19: true = usa o próprio fornecedor/estoque, não o do projeto. */
    override: boolean;
    regionManufacturerId: number | null;
    regionUseStockOnly: boolean;
    /** rf-16 RN21: época de cálculo — resposta de um cálculo antigo é
     *  descartada. Não vai para DTO nem rascunho. */
    calcSeq: number;
    /** rf-16 RN20: motivo de universo vazio desta região. */
    vazioMotivo: string | null;
  }

  interface RegiaoSalva {
    paintId: number | null;
    paintBrand: string;
    paintName: string;
    paintCode: string;
    deltaE: number;
    foraDoUniverso: boolean;
    ingredients: IngredienteDTO[];
    resultR: number | null;
    resultG: number | null;
    resultB: number | null;
    /** `''` = sem mistura salva (RN27). */
    faixa: string;
    method: string;
  }

  /** rf-09/RN7: chave de dedup é marca+código+nome — `paintId` colapsaria
   *  misturas diferentes (CA11). `tabs` cita os nomes das abas onde a tinta
   *  aparece (CA10). */
  interface ShoppingItem {
    key: string;
    name: string;
    code: string;
    manufacturer: string;
    tabs: string[];
  }

  /** rf-09 — estado de uma aba (figura). `regions`, `image`, `hasImage`,
   *  `imageDataUrl`, `nextId`, `selectedId` e `view` só pertencem à aba ATIVA
   *  nas variáveis de topo (abaixo); o resto do tempo moram aqui. `uid` é a
   *  chave estável do `{#each}` — nunca o índice. Fabricante base, modo
   *  estoque e a resposta do diálogo de fallback (rf-11) são do PLANO. */
  interface AbaState {
    uid: number;
    serverId?: number;
    name: string;
    imageDataUrl: string;
    hasImage: boolean;
    image: HTMLImageElement | null;
    regions: PlannerRegion[];
    nextId: number;
    selectedId: number | null;
    view: View;
  }

  /** rf-16 D2: universo do plano passado explicitamente — a abertura calcula
   *  com o contexto do plano que está chegando, antes de gravá-lo no topo. */
  interface CtxUniverso {
    manufacturerId: number | null;
    useStockOnly: boolean;
    saida: 'marca' | 'todos' | null;
  }

  let canvasEl: HTMLCanvasElement | undefined = $state();
  /** a caixa da foto — é ELA que dá o tamanho do bitmap e o enquadramento
   *  (D-002a). */
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
  let useStockOnly = $state(false);
  let saidaAutorizada: 'marca' | 'todos' | null = $state(null);
  /** rf-11: região que estourou o limiar e está esperando a resposta do
   *  diálogo. null = diálogo fechado. */
  let regiaoNoDialogo: number | null = $state(null);
  let dialogoMelhorMarca: number | null = $state(null);
  let dialogoMelhorTodos: number | null = $state(null);
  let dialogoElemento: HTMLDivElement | undefined = $state();
  let dialogoDisparador: HTMLElement | null = null;

  // Checklist local de "tenho"/"comprar" das tintas do PLANO INTEIRO.
  let ownedPaints: Set<string> = $state(new Set());

  // ── rf-16: lista de projetos e abertura ──
  let modo: 'lista' | 'editor' = $state('lista');
  let planos: ResumoPlanoDTO[] = $state([]);
  let listaCarregando = $state(false);
  let listaErro = $state(false);
  let abrindo = $state(false);
  let rascunhoInfo: { planId: number | null; nome: string } | null = $state(null);
  let listaSeq = 0;
  /** RN6b: época de abertura. Toda abertura sobe; carga, cálculo e
   *  marcação de alteração de uma época antiga se descartam. */
  let aberturaSeq = 0;

  // ── rf-09: abas de figura ──
  let abaUidSeq = 1;
  function emptyAba(): AbaState {
    return {
      uid: abaUidSeq++,
      serverId: undefined,
      name: '',
      imageDataUrl: '',
      hasImage: false,
      image: null,
      regions: [],
      nextId: 1,
      selectedId: null,
      view: { zoom: 1, panX: 0, panY: 0 },
    };
  }
  let tabs: AbaState[] = $state([emptyAba()]);
  let activeTabIndex = $state(0);
  let renamingTabIndex: number | null = $state(null);
  const MAX_ABAS_UI = 10;

  function displayTabName(aba: AbaState, i: number): string {
    const nome = aba.name.trim();
    return nome.length > 0 ? nome : `Figura ${i + 1}`;
  }

  /** Regiões "ao vivo" de uma aba: a ativa lê do espelho de trabalho. */
  function tabRegions(i: number): PlannerRegion[] {
    return i === activeTabIndex ? regions : tabs[i].regions;
  }

  function tabProgressLabel(i: number): string {
    const rs = tabRegions(i);
    return t('tabProgressShort', { a: rs.filter(r => r.painted).length, b: rs.length });
  }

  function snapshotActiveIntoTabs(): void {
    const atual = tabs[activeTabIndex];
    if (!atual) return;
    tabs[activeTabIndex] = {
      ...atual,
      imageDataUrl, hasImage, image, regions, nextId, selectedId, view,
    };
  }

  /** Relê o espelho de trabalho a partir da aba `i` — nunca marca alteração. */
  function loadTabIntoWorkingState(i: number): void {
    const aba = tabs[i];
    imageDataUrl = aba.imageDataUrl;
    hasImage = aba.hasImage;
    image = aba.image;
    regions = aba.regions;
    nextId = aba.nextId;
    selectedId = aba.selectedId;
    view = aba.view;
    regiaoNoDialogo = null;
  }

  function switchTab(i: number): void {
    if (i === activeTabIndex || i < 0 || i >= tabs.length) return;
    snapshotActiveIntoTabs();
    activeTabIndex = i;
    loadTabIntoWorkingState(i);
  }

  function addTab(): void {
    if (tabs.length >= MAX_ABAS_UI) {
      toast(t('errTabsMax'), 'error');
      return;
    }
    snapshotActiveIntoTabs();
    tabs = [...tabs, emptyAba()];
    activeTabIndex = tabs.length - 1;
    loadTabIntoWorkingState(activeTabIndex);
    marcarAlteracao();
  }

  function moveTab(i: number, dir: -1 | 1): void {
    const j = i + dir;
    if (j < 0 || j >= tabs.length) return;
    snapshotActiveIntoTabs();
    const arr = [...tabs];
    [arr[i], arr[j]] = [arr[j], arr[i]];
    tabs = arr;
    if (activeTabIndex === i) activeTabIndex = j;
    else if (activeTabIndex === j) activeTabIndex = i;
    marcarAlteracao();
  }

  function onDeleteTabClick(i: number): void {
    if (tabs.length <= 1) {
      toast(t('errTabDeleteLast'), 'error');
      return;
    }
    if (!window.confirm(t('deleteTabConfirm', { name: displayTabName(tabs[i], i) }))) return;
    deleteTab(i);
  }

  function deleteTab(i: number): void {
    const arr = tabs.filter((_, idx) => idx !== i);
    tabs = arr;
    if (activeTabIndex === i) {
      activeTabIndex = Math.min(i, arr.length - 1);
      loadTabIntoWorkingState(activeTabIndex);
    } else if (activeTabIndex > i) {
      activeTabIndex -= 1;
    }
    marcarAlteracao();
  }

  function startRename(i: number): void {
    renamingTabIndex = i;
  }

  function commitRename(): void {
    if (renamingTabIndex === null) return;
    renamingTabIndex = null;
    marcarAlteracao();
  }

  function onRenameKeydown(e: KeyboardEvent): void {
    if (e.key === 'Enter') { e.preventDefault(); commitRename(); }
    else if (e.key === 'Escape') { e.preventDefault(); renamingTabIndex = null; }
  }

  function focusTabButton(i: number): void {
    document.getElementById(`t2-tab-btn-${i}`)?.focus();
  }

  function onTabKeydown(e: KeyboardEvent, i: number): void {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      const n = (i + 1) % tabs.length;
      switchTab(n);
      focusTabButton(n);
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      const n = (i - 1 + tabs.length) % tabs.length;
      switchTab(n);
      focusTabButton(n);
    }
  }

  // ── rf-07: plano da peça (nome, salvar, auto save local) ──
  let planId: number | null = $state(null);
  let planName = $state('');
  let imageDataUrl = $state('');

  type SaveState = 'idle' | 'saving' | 'saved' | 'error';
  let saveState: SaveState = $state('idle');
  let savedAtLabel = $state('');
  /** RN9: check no ícone por 3 s depois de salvar. */
  let saveFlash = $state(false);
  let saveFlashTimer: ReturnType<typeof setTimeout> | null = null;
  let saveErrorKey: DictKey | null = $state(null);
  let validationError: ErroValidacaoPlano | null = $state(null);

  /** RN11: teto do arquivo em MB, com o separador do idioma. */
  let tetoMb = $derived(decimal(tetoArquivoImagem() / (1024 * 1024), 1));

  let validationErrorText = $derived.by((): string | null => {
    const erro = validationError;
    if (!erro) return null;
    const msg = t(erro.chave as DictKey, { mb: tetoMb });
    return erro.abaNome ? `${erro.abaNome}: ${msg}` : msg;
  });

  let draftBannerVisible = $state(false);
  let autoSaveStatus: EstadoAutoSave = $state('ligado');

  /** Sobe a cada Salvar concluído e a cada alteração de conteúdo. */
  let salvamentoSeq = 0;
  let alteracaoSeq = 0;
  let alteracaoSeqSalva = $state(0);

  // ── rf-08: exportar relatório (PDF/PNG) ──
  let reportGenerating = $state(false);
  let reportErrorKey: DictKey | null = $state(null);
  let exportMenuOpen = $state(false);

  // ── rf-16 RN7: título editável ──
  let renomeandoTitulo = $state(false);
  let tituloRascunho = $state('');
  let tituloCancelado = false;
  let tituloInputEl: HTMLInputElement | undefined = $state();

  let planNamePlaceholder = $derived.by(() => {
    const d = new Date();
    const dd = String(d.getDate()).padStart(2, '0');
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    return t('projectNamePh', { data: `${dd}/${mm}` });
  });

  // RN1: a lista recarrega sempre que T2 fica visível no modo lista, e
  // sempre que o modo volta do editor para a lista.
  $effect(() => {
    if (nav.tab === 'plano' && modo === 'lista') {
      untrack(() => void carregarLista());
    }
  });

  $effect(() => {
    if (manufacturerId === null && manufacturers.length > 0) manufacturerId = manufacturers[0].id;
  });

  $effect(() => {
    if (regiaoNoDialogo !== null) dialogoElemento?.focus();
  });

  $effect(() => {
    if (renomeandoTitulo) tituloInputEl?.focus();
  });

  let selectedRegion = $derived(regions.find(r => r.id === selectedId) ?? null);
  let totalRegionsCount = $derived(tabs.reduce((acc, _a, i) => acc + tabRegions(i).length, 0));
  let paintedCount = $derived(
    tabs.reduce((acc, _a, i) => acc + tabRegions(i).filter(r => r.painted).length, 0)
  );
  let progressPct = $derived(totalRegionsCount ? Math.round((paintedCount / totalRegionsCount) * 100) : 0);

  /** RN25: Salvar espera os cálculos em voo. */
  let algumCalculando = $derived(tabs.some((_a, i) => tabRegions(i).some(r => r.computing)));

  /** RN20: o aviso do topo do painel é só das regiões que seguem o projeto. */
  let avisoUniversoProjeto = $derived.by((): string | null => {
    for (let i = 0; i < tabs.length; i++) {
      const achado = tabRegions(i).find(r => !r.override && r.vazioMotivo);
      if (achado) return achado.vazioMotivo;
    }
    return null;
  });

  let rascunhoPendente = $derived(draftBannerVisible || alteracaoSeq !== alteracaoSeqSalva);
  let exportDisabled = $derived(planId == null || rascunhoPendente || reportGenerating);

  function paintKey(brand: string, code: string, name: string): string {
    return `${brand} ${code} ${name}`;
  }

  let shoppingItems = $derived.by((): ShoppingItem[] => {
    const map = new Map<string, ShoppingItem>();
    tabs.forEach((aba, i) => {
      const abaNome = displayTabName(aba, i);
      tabRegions(i).forEach((r) => {
        const desc = descritorDaRegiao(r);
        if (!desc.paintBrand && !desc.paintCode && !desc.paintName) return;
        const key = paintKey(desc.paintBrand, desc.paintCode, desc.paintName);
        const existente = map.get(key);
        if (existente) {
          if (!existente.tabs.includes(abaNome)) existente.tabs.push(abaNome);
        } else {
          map.set(key, {
            key,
            name: desc.paintName,
            code: desc.paintCode,
            manufacturer: desc.paintBrand,
            tabs: [abaNome],
          });
        }
      });
    });
    return [...map.values()];
  });

  function regionName(i: number): string {
    return t('regionLabel', { n: i + 1 });
  }

  function regionMatchText(r: PlannerRegion): string {
    if (r.computing) return t('calculating');
    if (!r.result) return r.vazioMotivo ?? t('noSimilarPaint');
    return `ΔE ${decimal(r.result.deltaE, 1)} · ${r.result.ingredients.map(ing => ing.name).join(' + ')}`;
  }

  function seloDaFaixa(r: PlannerRegion): { texto: string; cor: string } | null {
    const faixa = r.result?.faixa;
    if (!faixa) return null;
    if (faixa === 'otimo') return { texto: t('faixaOtima'), cor: 'var(--color-success, #3FA34D)' };
    if (faixa === 'aproximada') return { texto: t('faixaAproximada'), cor: 'var(--color-warning, #E4A11B)' };
    return { texto: t('faixaNaoEncontrei'), cor: 'var(--color-danger, #D1495B)' };
  }

  function verdictColor(deltaE: number): string {
    const { v } = verdictKeys(deltaE);
    if (v === 'v1' || v === 'v2') return 'var(--color-accent-400)';
    if (v === 'v3') return 'var(--color-neutral-300)';
    return 'var(--color-neutral-500)';
  }

  function hexDe(r: number, g: number, b: number): string {
    return `#${[r, g, b].map(n => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0')).join('')}`.toUpperCase();
  }

  function nomeFabricante(id: number | null): string {
    if (id == null) return '';
    return manufacturers.find(m => m.id === id)?.name ?? '';
  }

  /** RN18.7: de onde a tinta da região pôde sair. */
  function origemDaRegiao(r: PlannerRegion): string {
    if (r.override) {
      const base = t('regionOriginOverride', { marca: nomeFabricante(r.regionManufacturerId) || t('allMakers') });
      return r.regionUseStockOnly ? `${base}, ${t('regionOriginStock')}` : base;
    }
    const marca = saidaAutorizada === 'todos' ? t('allMakers') : nomeFabricante(manufacturerId) || t('allMakers');
    const base = t('regionOriginProject', { marca });
    return useStockOnly && saidaAutorizada === null ? `${base}, ${t('regionOriginStock')}` : base;
  }

  function toggleOwned(key: string) {
    const next = new Set(ownedPaints);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    ownedPaints = next;
  }

  function token(name: string, fallback: string): string {
    const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

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

  function fitToBox() {
    if (!image) return;
    const { w, h } = boxSize();
    if (w <= 0 || h <= 0) return;
    view = fitView(image.width, image.height, w, h);
    draw();
  }

  $effect(() => {
    if (!image || !canvasEl) return;
    requestAnimationFrame(fitToBox);
  });

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
      imageDataUrl = src;
      marcarAlteracao();
    };
    img.src = src;
  }

  function loadImageBitmap(src: string): Promise<HTMLImageElement | null> {
    return new Promise(resolve => {
      const img = new Image();
      img.onload = () => resolve(img);
      img.onerror = () => resolve(null);
      img.src = src;
    });
  }

  function onFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    // rf-16 RN12: tipo e tamanho barrados ANTES de ler o arquivo — o usuário
    // já viu o limite na tela, e ler 20 MB só para recusar gasta memória.
    if (!TIPOS_IMAGEM_ACEITOS.includes(file.type)) {
      toast(t('errImageType'), 'error');
      input.value = '';
      return;
    }
    if (file.size > tetoArquivoImagem()) {
      toast(t('errImageMax', { mb: tetoMb }), 'error');
      input.value = '';
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const src = reader.result as string;
      // D-005: a mesma guarda da CA17 continua como rede de segurança.
      const erroImagem = validarImagemDaAba(src);
      if (erroImagem) {
        toast(t(erroImagem as DictKey, { mb: tetoMb }), 'error');
        input.value = '';
        return;
      }
      loadImage(src);
    };
    reader.readAsDataURL(file);
  }

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

  function getTouchDist(tl: TouchList): number {
    if (tl.length < 2) return 0;
    return Math.hypot(tl[0].clientX - tl[1].clientX, tl[0].clientY - tl[1].clientY);
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
      selectRegion(hit.id, true);
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
    if (regions.length >= 50) {
      toast(`${displayTabName(tabs[activeTabIndex], activeTabIndex)}: ${t('errRegionsMax')}`, 'error');
      return;
    }
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
      override: false,
      regionManufacturerId: null,
      regionUseStockOnly: false,
      calcSeq: 0,
      vazioMotivo: null,
    };
    regions = [...regions, region];
    selectRegion(region.id, true);
    const tabUid = tabs[activeTabIndex].uid;
    await recalcularComMarcacao(() => computeRegion(region.id, tabUid));
  }

  /** Acha a região `id` pela identidade da aba (`tabUid`) — nunca pelo
   *  binding vivo `regions` (achado 1 do guardrail rf-09). */
  function regiaoDaAba(tabUid: number, id: number): PlannerRegion | null {
    const arr = tabs[activeTabIndex]?.uid === tabUid ? regions : (tabs.find(a => a.uid === tabUid)?.regions ?? null);
    return arr?.find(r => r.id === id) ?? null;
  }

  function ctxAtual(): CtxUniverso {
    return { manufacturerId, useStockOnly, saida: saidaAutorizada };
  }

  /** rf-16 RN20: o universo sai da região — ajuste próprio, ou o do projeto
   *  (com a saída autorizada do rf-11). Com "só o que eu tenho", o estoque do
   *  aparelho viaja junto (RN14). */
  function universoDaRegiao(reg: PlannerRegion, ctx: CtxUniverso = ctxAtual()): UniversoBusca {
    if (reg.override) {
      const so = reg.regionUseStockOnly;
      return {
        targetManufacturerId: reg.regionManufacturerId ?? undefined,
        useStockOnly: so,
        foraDoUniverso: false,
        stock: so ? stock.paints : undefined,
      };
    }
    const so = ctx.saida === null && ctx.useStockOnly;
    return {
      targetManufacturerId: ctx.saida === 'todos' ? undefined : ctx.manufacturerId ?? undefined,
      useStockOnly: so,
      foraDoUniverso: ctx.saida !== null,
      stock: so ? stock.paints : undefined,
    };
  }

  /** rf-11 RN4/RN5/RN6: o diálogo de fallback. */
  async function abrirDialogoDeFallback(regiaoId: number, r: number, g: number, b: number): Promise<void> {
    if (regiaoNoDialogo !== null) return;
    dialogoDisparador = document.activeElement as HTMLElement | null;
    regiaoNoDialogo = regiaoId;
    dialogoMelhorMarca = null;
    dialogoMelhorTodos = null;

    if (manufacturerId != null) {
      dialogoMelhorMarca = await melhorDeltaE(r, g, b, { targetManufacturerId: manufacturerId });
    }
    dialogoMelhorTodos = await melhorDeltaE(r, g, b, {});
  }

  async function escolherSaida(saida: 'marca' | 'todos'): Promise<void> {
    saidaAutorizada = saida;
    regiaoNoDialogo = null;
    devolverFoco();
    await recalcularComMarcacao(recalcularRegioesDoProjeto);
  }

  /** Marca a alteração no gesto (antes do await) e de novo quando o cálculo
   *  termina. A marcação síncrona é o que faz um Salvar em voo perceber que
   *  a tela mudou e manter o rascunho (achado 2 do guardrail); a segunda
   *  grava o resultado. Só a época de abertura barra — um Salvar concluído
   *  no meio não barra mais, porque o Salvar fica desabilitado enquanto há
   *  cálculo (RN25) e a mudança só pode ter vindo depois dele. */
  async function recalcularComMarcacao(calcular: () => Promise<void>): Promise<void> {
    const ep = aberturaSeq;
    marcarAlteracao();
    await calcular();
    if (ep === aberturaSeq) marcarAlteracao();
  }

  /** RN22: mudança global recalcula só quem segue o projeto, em todas as
   *  figuras. Região ajustada mantém fornecedor, interruptor e mistura. */
  async function recalcularRegioesDoProjeto(): Promise<void> {
    await Promise.all(
      tabs.flatMap((aba, i) => tabRegions(i).filter(r => !r.override).map(r => computeRegion(r.id, aba.uid))),
    );
  }

  function devolverFoco(): void {
    dialogoDisparador?.focus();
    dialogoDisparador = null;
  }

  function onDialogoKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape') {
      e.preventDefault();
      fecharDialogo();
    }
  }

  function fecharDialogo(): void {
    regiaoNoDialogo = null;
    devolverFoco();
  }

  async function onUseStockOnlyChange(valor: boolean): Promise<void> {
    useStockOnly = valor;
    saidaAutorizada = null;
    await recalcularComMarcacao(recalcularRegioesDoProjeto);
  }

  /** Recibo de mistura no formato gravado (RN24/RN26). */
  function salvoDoResultado(res: EquivalentRecipe): RegiaoSalva {
    const ing = res.ingredients ?? [];
    return {
      paintId: ing.length === 1 ? ing[0].paintId : null,
      paintBrand: res.targetManufacturer ?? '',
      paintName: ing.length === 1 ? ing[0].name : ing.map(x => x.name).join(' + '),
      paintCode: ing.length === 1 ? ing[0].code : '',
      deltaE: res.deltaE ?? 0,
      foraDoUniverso: res.foraDoUniverso ?? false,
      ingredients: ing.map(x => ({
        paintId: x.paintId,
        manufacturerId: x.manufacturerId ?? 0,
        manufacturer: x.manufacturer ?? '',
        name: x.name,
        code: x.code,
        r: x.r,
        g: x.g,
        b: x.b,
        percentage: x.percentage,
      })),
      resultR: res.resultR,
      resultG: res.resultG,
      resultB: res.resultB,
      faixa: res.faixa ?? '',
      method: res.method ?? '',
    };
  }

  /** RN27: monta a receita a partir da mistura salva, sem chamar o motor.
   *  `null` quando não há mistura salva (`faixa` vazia). */
  function receitaDoSalvo(s: RegiaoSalva | null, r: number, g: number, b: number): EquivalentRecipe | null {
    if (!s || !s.faixa) return null;
    const ingredients = s.ingredients.map(i => ({ ...i }));
    return {
      sourcePaintId: 0,
      sourceName: '',
      sourceManufacturer: '',
      sourceR: r,
      sourceG: g,
      sourceB: b,
      targetManufacturer: s.paintBrand,
      ingredients,
      resultR: s.resultR ?? r,
      resultG: s.resultG ?? g,
      resultB: s.resultB ?? b,
      deltaE: s.deltaE,
      method: s.method,
      reproducible: s.faixa !== 'nao-encontrei',
      faixa: s.faixa as EquivalentRecipe['faixa'],
      foraDoUniverso: s.foraDoUniverso,
      tips: null,
      crossBrand: new Set(ingredients.map(i => i.manufacturerId)).size > 1,
      manufacturers: [...new Set(ingredients.map(i => i.manufacturer).filter(Boolean))].sort(),
    };
  }

  /** D-002b: cálculo endereçado por ID. rf-16: com época de abertura
   *  (RN6b) e época de cálculo da região (RN21) — resposta velha não escreve. */
  async function computeRegion(id: number, tabUidChamada: number = tabs[activeTabIndex].uid) {
    const alvo = () => regiaoDaAba(tabUidChamada, id);
    const inicio = alvo();
    if (!inicio) return;
    const ep = aberturaSeq;
    const seq = inicio.calcSeq + 1;
    inicio.calcSeq = seq;
    inicio.computing = true;
    const vigente = (): PlannerRegion | null => {
      const agora = alvo();
      return ep === aberturaSeq && agora && agora.calcSeq === seq ? agora : null;
    };
    try {
      const universo = universoDaRegiao(inicio);
      if (universo.useStockOnly) {
        // rf-17 RN13: o estoque vem do servidor — sem esperar a carga, iria vazio.
        await estoqueAssentado();
        universo.stock = stock.paints;
      }
      const resp = await suggestRecipeForColor(inicio.r, inicio.g, inicio.b, universo);
      const agora = vigente();
      if (!agora) return;
      if (!resp) {
        agora.result = null;
        return;
      }
      if (ehUniversoVazio(resp)) {
        agora.vazioMotivo = resp.motivo;
        agora.result = null;
        return;
      }
      agora.vazioMotivo = null;
      agora.result = resp;
      agora.salvo = salvoDoResultado(resp);
      // RN23: o diálogo de fallback é só de quem segue o projeto.
      if (resp.faixa === 'nao-encontrei' && !agora.override && saidaAutorizada === null) {
        void abrirDialogoDeFallback(id, inicio.r, inicio.g, inicio.b);
      }
    } catch (e) {
      console.error('Erro ao calcular região:', e);
      const agora = vigente();
      if (agora) agora.result = null;
    } finally {
      const agora = vigente();
      if (agora) agora.computing = false;
      draw();
    }
  }

  async function onManufacturerChange(id: number) {
    if (id === manufacturerId) return;
    manufacturerId = id;
    saidaAutorizada = null;
    await recalcularComMarcacao(recalcularRegioesDoProjeto);
  }

  /** RN35: seleciona, destaca, rola e dá foco ao card na lista. */
  function selectRegion(id: number, focar = false) {
    selectedId = id;
    draw();
    if (!focar) return;
    void tick().then(() => {
      const el = document.getElementById(`t2-region-card-${id}`);
      if (!el) return;
      el.scrollIntoView?.({ block: 'nearest' });
      el.focus({ preventScroll: true });
    });
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

  // ── rf-16 RN19–RN21: ajuste por região ──
  async function ajustarFornecedorDaRegiao(r: PlannerRegion, id: number | null): Promise<void> {
    if (id === null) {
      r.override = false;
      r.regionManufacturerId = null;
      r.regionUseStockOnly = false;
    } else {
      if (!r.override) r.regionUseStockOnly = useStockOnly;
      r.override = true;
      r.regionManufacturerId = id;
    }
    await recalcularUmaRegiao(r.id);
  }

  async function ajustarEstoqueDaRegiao(r: PlannerRegion, valor: boolean): Promise<void> {
    r.regionUseStockOnly = valor;
    await recalcularUmaRegiao(r.id);
  }

  async function recalcularUmaRegiao(id: number): Promise<void> {
    const tabUid = tabs[activeTabIndex].uid;
    await recalcularComMarcacao(() => computeRegion(id, tabUid));
  }

  // ── rf-07/rf-16: mapeamento tela → rascunho/DTO ──
  const SALVO_VAZIO: RegiaoSalva = {
    paintId: null, paintBrand: '', paintName: '', paintCode: '', deltaE: 0, foraDoUniverso: false,
    ingredients: [], resultR: null, resultG: null, resultB: null, faixa: '', method: '',
  };

  /** RN26: com `result`, a mistura atual; sem, a última mistura boa. */
  function descritorDaRegiao(r: PlannerRegion): RegiaoSalva {
    return r.result ? salvoDoResultado(r.result) : (r.salvo ?? SALVO_VAZIO);
  }

  function regiaoParaRascunho(r: PlannerRegion, i: number): RegiaoRascunho {
    const d = descritorDaRegiao(r);
    return {
      x: r.x, y: r.y, r: r.r, g: r.g, b: r.b, hex: r.hex, regionName: regionName(i), note: '',
      paintId: d.paintId, paintBrand: d.paintBrand, paintName: d.paintName, paintCode: d.paintCode,
      deltaE: d.deltaE, painted: r.painted, foraDoUniverso: d.foraDoUniverso,
      regionOverride: r.override,
      regionManufacturerId: r.override ? r.regionManufacturerId : null,
      regionUseStockOnly: r.override && r.regionUseStockOnly,
      resultR: d.resultR, resultG: d.resultG, resultB: d.resultB,
      faixa: d.faixa, method: d.method, ingredients: d.ingredients,
    };
  }

  function regiaoParaDTO(r: PlannerRegion, i: number): RegiaoDTO {
    const d = descritorDaRegiao(r);
    return {
      x: r.x, y: r.y, r: r.r, g: r.g, b: r.b, hex: r.hex, regionName: regionName(i), note: '',
      paintId: d.paintId, paintBrand: d.paintBrand, paintName: d.paintName, paintCode: d.paintCode,
      deltaE: d.deltaE, painted: r.painted ? 1 : 0, foraDoUniverso: d.foraDoUniverso ? 1 : 0,
      regionOverride: r.override ? 1 : 0,
      regionManufacturerId: r.override ? r.regionManufacturerId : null,
      regionUseStockOnly: r.override && r.regionUseStockOnly ? 1 : 0,
      resultR: d.resultR, resultG: d.resultG, resultB: d.resultB,
      faixa: d.faixa, method: d.method, ingredients: d.ingredients,
    };
  }

  function buildAbaDTO(aba: AbaState): AbaDTO {
    return {
      id: aba.serverId,
      name: aba.name,
      imageData: aba.imageDataUrl,
      regions: aba.regions.map((r, i) => regiaoParaDTO(r, i)),
    };
  }

  function buildAbaRascunho(aba: AbaState): AbaRascunho {
    return {
      id: aba.serverId ?? null,
      name: aba.name,
      imageData: aba.imageDataUrl,
      regions: aba.regions.map((r, i) => regiaoParaRascunho(r, i)),
    };
  }

  function buildPlanoDTO(): PlanoDTO {
    snapshotActiveIntoTabs();
    return {
      id: planId ?? undefined,
      name: planName,
      selectedManufacturerId: manufacturerId,
      useStockOnly: useStockOnly ? 1 : 0,
      tabs: tabs.map(buildAbaDTO),
    };
  }

  // ── rf-07/rf-09: auto save local ──
  /** Chamada explícita nos mutadores de CONTEÚDO — nunca em zoom/pan/view, e
   *  nunca com uma abertura em andamento (rf-16 RN6b; quem chama depois de um
   *  await confere a época). Vale também com a lista na tela: um cálculo
   *  pedido no editor pode terminar depois do "← Projetos", e o resultado
   *  dele é do projeto que ainda está no editor (achado 1 do guardrail). */
  function marcarAlteracao() {
    if (abrindo) return;
    snapshotActiveIntoTabs();
    const payload: Rascunho = {
      v: 2,
      planId,
      name: planName,
      selectedManufacturerId: manufacturerId,
      useStockOnly,
      abaAtiva: activeTabIndex,
      tabs: tabs.map(buildAbaRascunho),
      salvoEm: new Date().toISOString(),
    };
    alteracaoSeq++;
    agendarGravacao(payload);
    if (modo === 'lista') {
      // A lista já leu o rascunho: grava agora e atualiza o card.
      gravarPendente();
      lerRascunhoInfo();
    }
    setTimeout(() => { autoSaveStatus = estadoAutoSave(); }, 1600);
  }

  // ── rf-16: lista, abertura e volta ──
  function lerRascunhoInfo(): void {
    const { rascunho, corrompido } = lerRascunho();
    if (corrompido) toast(t('draftCorrupted'), 'error');
    rascunhoInfo = rascunho ? { planId: rascunho.planId, nome: rascunho.name } : null;
  }

  async function carregarLista(): Promise<void> {
    const ep = ++listaSeq;
    listaCarregando = true;
    listaErro = false;
    lerRascunhoInfo();
    try {
      const lista = await listarPlanos();
      if (ep !== listaSeq) return;
      planos = lista;
    } catch {
      if (ep === listaSeq) listaErro = true;
    } finally {
      if (ep === listaSeq) listaCarregando = false;
    }
  }

  function nomeDoRascunho(): string {
    return rascunhoInfo?.nome.trim() || t('projectNoName');
  }

  /** RN5: a vaga única do rascunho seria sobrescrita pelo outro projeto. */
  function confirmarDescarteDoRascunho(nomeAlvo: string): boolean {
    if (!rascunhoInfo) return true;
    if (!window.confirm(t('discardOtherDraftConfirm', { a: nomeDoRascunho(), b: nomeAlvo }))) return false;
    apagarRascunho();
    rascunhoInfo = null;
    return true;
  }

  function entrarNoEditor(comRascunho: boolean): void {
    saveState = 'idle';
    saveErrorKey = null;
    validationError = null;
    reportErrorKey = null;
    exportMenuOpen = false;
    renomeandoTitulo = false;
    draftBannerVisible = comRascunho;
    alteracaoSeqSalva = alteracaoSeq;
    autoSaveStatus = estadoAutoSave();
    modo = 'editor';
  }

  function voltarParaLista(): void {
    gravarPendente();
    exportMenuOpen = false;
    modo = 'lista';
  }

  function novoProjeto(): void {
    if (!confirmarDescarteDoRascunho(t('newProjectBtn'))) return;
    ++aberturaSeq;
    resetToEmpty();
    entrarNoEditor(false);
  }

  async function abrirProjeto(p: ResumoPlanoDTO): Promise<void> {
    if (rascunhoInfo && rascunhoInfo.planId === p.id) {
      await abrirRascunho();
      return;
    }
    if (!confirmarDescarteDoRascunho(p.name)) return;
    const ep = ++aberturaSeq;
    abrindo = true;
    try {
      const plano = await carregarPlano(p.id);
      if (ep !== aberturaSeq) return;
      await applyLoadedPlan(plano, true, 0, ep);
      if (ep !== aberturaSeq) return;
      entrarNoEditor(false);
    } catch (e) {
      if (ep !== aberturaSeq) return;
      const naoExiste = e instanceof PlanoError && e.code === 'nao-encontrado';
      toast(t(naoExiste ? 'planNotFound' : 'planSaveError'), 'error');
      if (naoExiste) void carregarLista();
    } finally {
      if (ep === aberturaSeq) abrindo = false;
    }
  }

  async function abrirRascunho(): Promise<void> {
    const { rascunho, corrompido, truncado } = lerRascunho();
    if (corrompido) toast(t('draftCorrupted'), 'error');
    if (!rascunho) {
      await carregarLista();
      return;
    }
    const ep = ++aberturaSeq;
    abrindo = true;
    try {
      // RN4: rascunho de projeto que não está mais na lista vira projeto novo.
      let planIdEfetivo = rascunho.planId;
      if (planIdEfetivo != null && !listaErro && !planos.some(pl => pl.id === planIdEfetivo)) {
        planIdEfetivo = null;
      }
      // RN8 do rf-07 cortou a foto para caber na cota: busca do servidor.
      const semFoto = rascunho.tabs.some(aba => !aba.imageData);
      let abasServidor: AbaDTO[] | null = null;
      if (semFoto && planIdEfetivo != null) {
        try {
          abasServidor = (await carregarPlano(planIdEfetivo)).tabs;
        } catch {
          /* sem o plano do servidor não há foto de resgate — avisa abaixo. */
        }
        if (ep !== aberturaSeq) return;
      }
      if (abasServidor === null && rascunho.tabs.some(aba => !aba.imageData && aba.regions.length > 0)) {
        toast(t('draftNoPhoto'), 'error');
      }
      const abaServidorPorIdentidade = (rt: AbaRascunho): AbaDTO | undefined => {
        if (!abasServidor) return undefined;
        if (rt.id != null) {
          const porId = abasServidor.find(a => a.id === rt.id);
          if (porId) return porId;
        }
        return abasServidor.find(a => a.name === rt.name);
      };
      // RN27: o rascunho traz ajuste e mistura — a remontagem copia tudo,
      // inclusive o `foraDoUniverso`, que antes se perdia.
      const tabsDTO: AbaDTO[] = rascunho.tabs.map((rt) => ({
        id: undefined,
        name: rt.name,
        imageData: rt.imageData || abaServidorPorIdentidade(rt)?.imageData || '',
        regions: rt.regions.map((rr): RegiaoDTO => ({
          x: rr.x, y: rr.y, r: rr.r, g: rr.g, b: rr.b, hex: rr.hex,
          regionName: rr.regionName, note: rr.note,
          paintId: rr.paintId, paintBrand: rr.paintBrand,
          paintName: rr.paintName, paintCode: rr.paintCode,
          deltaE: rr.deltaE, painted: rr.painted ? 1 : 0,
          foraDoUniverso: rr.foraDoUniverso ? 1 : 0,
          regionOverride: rr.regionOverride ? 1 : 0,
          regionManufacturerId: rr.regionManufacturerId ?? null,
          regionUseStockOnly: rr.regionUseStockOnly ? 1 : 0,
          resultR: rr.resultR ?? null, resultG: rr.resultG ?? null, resultB: rr.resultB ?? null,
          faixa: rr.faixa ?? '', method: rr.method ?? '', ingredients: rr.ingredients ?? [],
        })),
      }));
      await applyLoadedPlan(
        {
          id: planIdEfetivo ?? undefined,
          name: rascunho.name,
          selectedManufacturerId: rascunho.selectedManufacturerId,
          useStockOnly: rascunho.useStockOnly ? 1 : 0,
          tabs: tabsDTO,
        },
        false,
        rascunho.abaAtiva,
        ep,
      );
      if (ep !== aberturaSeq) return;
      entrarNoEditor(true);
      if (truncado) toast(t('errRegionsMax'), 'error');
    } finally {
      if (ep === aberturaSeq) abrindo = false;
    }
  }

  async function excluirProjeto(p: ResumoPlanoDTO): Promise<void> {
    if (!window.confirm(t('deleteProjectConfirm', { name: p.name }))) return;
    try {
      await excluirPlano(p.id);
    } catch {
      toast(t('deleteProjectError'), 'error');
      return;
    }
    if (rascunhoInfo?.planId === p.id) {
      apagarRascunho();
      rascunhoInfo = null;
    }
    if (planId === p.id) {
      ++aberturaSeq;
      resetToEmpty();
    }
    planos = planos.filter(x => x.id !== p.id);
  }

  // ── rf-07/rf-09: aplicar plano carregado ──
  function resetToEmpty() {
    planId = null;
    planName = '';
    manufacturerId = manufacturers[0]?.id ?? null;
    useStockOnly = false;
    saidaAutorizada = null;
    ownedPaints = new Set();
    tabs = [emptyAba()];
    activeTabIndex = 0;
    loadTabIntoWorkingState(0);
  }

  /** Cálculo de uma região de uma aba que ainda não está em `tabs`
   *  (abertura), com o contexto do plano que está chegando. */
  async function computeRegionInAba(aba: AbaState, id: number, ctx: CtxUniverso, ep: number): Promise<void> {
    const alvo = () => aba.regions.find(r => r.id === id) ?? null;
    const inicio = alvo();
    if (!inicio) return;
    inicio.computing = true;
    try {
      const universo = universoDaRegiao(inicio, ctx);
      if (universo.useStockOnly) {
        await estoqueAssentado();
        universo.stock = stock.paints;
      }
      const resp = await suggestRecipeForColor(inicio.r, inicio.g, inicio.b, universo);
      if (ep !== aberturaSeq) return;
      const agora = alvo();
      if (!agora) return;
      if (!resp) {
        agora.result = null;
      } else if (ehUniversoVazio(resp)) {
        agora.vazioMotivo = resp.motivo;
        agora.result = null;
      } else {
        agora.vazioMotivo = null;
        agora.result = resp;
        agora.salvo = salvoDoResultado(resp);
      }
    } catch (e) {
      console.error('Erro ao calcular região:', e);
    } finally {
      const agora = alvo();
      if (agora) agora.computing = false;
    }
  }

  /** Aplica um `PlanoDTO` (do servidor ou do rascunho). rf-16 RN6b: tudo o
   *  que é do topo (id, nome, universo, abas) só é gravado no FIM, e só se a
   *  época ainda for a desta abertura. */
  async function applyLoadedPlan(plano: PlanoDTO, recalcular: boolean, abaAtivaIndex: number, ep: number): Promise<void> {
    // RN13: fabricante efetivo fixado antes de qualquer recálculo.
    const salvoExiste = plano.selectedManufacturerId != null && manufacturers.some(m => m.id === plano.selectedManufacturerId);
    const ctx: CtxUniverso = {
      manufacturerId: salvoExiste ? (plano.selectedManufacturerId as number) : (manufacturers[0]?.id ?? null),
      useStockOnly: plano.useStockOnly === 1,
      saida: null,
    };

    const tabsDTO = plano.tabs.length > 0 ? plano.tabs : [{ name: '', imageData: '', regions: [] }];

    const novasAbas: AbaState[] = [];
    for (const abaDTO of tabsDTO) {
      let nextIdLocal = 1;
      const abaRegions: PlannerRegion[] = abaDTO.regions.map((rd): PlannerRegion => {
        const salvo: RegiaoSalva = {
          paintId: rd.paintId ?? null,
          paintBrand: rd.paintBrand,
          paintName: rd.paintName,
          paintCode: rd.paintCode,
          deltaE: rd.deltaE,
          foraDoUniverso: rd.foraDoUniverso === 1,
          ingredients: rd.ingredients ?? [],
          resultR: rd.resultR ?? null,
          resultG: rd.resultG ?? null,
          resultB: rd.resultB ?? null,
          faixa: rd.faixa ?? '',
          method: rd.method ?? '',
        };
        return {
          id: nextIdLocal++,
          x: rd.x, y: rd.y, r: rd.r, g: rd.g, b: rd.b, hex: rd.hex,
          painted: rd.painted === 1,
          result: receitaDoSalvo(salvo, rd.r, rd.g, rd.b),
          computing: false,
          salvo,
          override: rd.regionOverride === 1,
          regionManufacturerId: rd.regionManufacturerId ?? null,
          regionUseStockOnly: rd.regionUseStockOnly === 1,
          calcSeq: 0,
          vazioMotivo: null,
        };
      });
      const img = abaDTO.imageData ? await loadImageBitmap(abaDTO.imageData) : null;
      if (ep !== aberturaSeq) return;
      novasAbas.push({
        uid: abaUidSeq++,
        serverId: abaDTO.id,
        name: abaDTO.name,
        imageDataUrl: abaDTO.imageData || '',
        hasImage: !!abaDTO.imageData,
        image: img,
        regions: abaRegions,
        nextId: nextIdLocal,
        selectedId: null,
        view: { zoom: 1, panX: 0, panY: 0 },
      });
    }

    // RN27/RN28: só região sem mistura salva recalcula; rascunho nunca
    // consulta o servidor (CA8 do rf-07).
    if (recalcular) {
      await Promise.all(
        novasAbas.flatMap(aba => aba.regions.filter(r => !r.salvo?.faixa).map(r => computeRegionInAba(aba, r.id, ctx, ep))),
      );
    }
    if (ep !== aberturaSeq) return;

    planId = plano.id ?? null;
    planName = plano.name;
    manufacturerId = ctx.manufacturerId;
    useStockOnly = ctx.useStockOnly;
    saidaAutorizada = null;
    ownedPaints = new Set();
    tabs = novasAbas;
    activeTabIndex = Math.min(Math.max(abaAtivaIndex, 0), tabs.length - 1);
    loadTabIntoWorkingState(activeTabIndex);
  }

  // ── rf-07: descartar rascunho (RN6) ──
  function onDiscardDraftClick() {
    if (!window.confirm(t('discardDraftConfirm'))) return;
    void discardDraftAndReload();
  }

  async function discardDraftAndReload() {
    if (planId != null) {
      const ep = ++aberturaSeq;
      abrindo = true;
      try {
        const plano = await carregarPlano(planId);
        await applyLoadedPlan(plano, true, 0, ep);
        if (ep !== aberturaSeq) return;
        apagarRascunho();
      } catch (e) {
        if (ep !== aberturaSeq) return;
        if (e instanceof PlanoError && e.code === 'nao-encontrado') {
          apagarRascunho();
          resetToEmpty();
          toast(t('planNotFound'), 'error');
        } else {
          saveState = 'error';
          saveErrorKey = 'planSaveError';
          return;
        }
      } finally {
        if (ep === aberturaSeq) abrindo = false;
      }
    } else {
      apagarRascunho();
      ++aberturaSeq;
      resetToEmpty();
    }
    draftBannerVisible = false;
    alteracaoSeqSalva = alteracaoSeq;
  }

  // ── rf-07: Salvar ──
  async function handleSave() {
    if (saveState === 'saving' || algumCalculando) return;
    const dto = buildPlanoDTO();
    const erro = validarPlano(dto);
    if (erro) {
      validationError = erro;
      return;
    }
    validationError = null;
    saveErrorKey = null;
    saveState = 'saving';
    const seqNoEnvio = alteracaoSeq;
    const ep = aberturaSeq;
    try {
      const { plano, suspeitaD003 } = await salvarPlano(dto);
      if (ep !== aberturaSeq) return;
      if (plano.id != null) planId = plano.id;
      if (suspeitaD003) {
        saveState = 'error';
        saveErrorKey = 'planSaveError';
        return;
      }
      salvamentoSeq++;
      if (alteracaoSeq === seqNoEnvio) {
        apagarRascunho();
        alteracaoSeqSalva = seqNoEnvio;
      } else {
        marcarAlteracao();
      }
      draftBannerVisible = false;
      const agora = new Date();
      savedAtLabel = `${String(agora.getHours()).padStart(2, '0')}:${String(agora.getMinutes()).padStart(2, '0')}`;
      saveState = 'saved';
      saveFlash = true;
      if (saveFlashTimer) clearTimeout(saveFlashTimer);
      saveFlashTimer = setTimeout(() => { saveFlash = false; }, 3000);
    } catch (e) {
      if (ep !== aberturaSeq) return;
      saveState = 'error';
      saveErrorKey = e instanceof PlanoError && e.code === 'nao-encontrado' ? 'planNotFound' : 'planSaveError';
    }
  }

  // ── rf-08: exportar relatório ──
  async function handleExport(formato: 'pdf' | 'png') {
    exportMenuOpen = false;
    if (reportGenerating) return;
    if (planId == null || rascunhoPendente) {
      reportErrorKey = 'exportSaveFirst';
      return;
    }
    if (!Number.isInteger(planId) || planId <= 0) {
      reportErrorKey = 'exportInvalid';
      return;
    }
    reportErrorKey = null;
    reportGenerating = true;
    try {
      const { blob, filename } = await baixarRelatorio(planId, formato);
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

  // ── rf-16 RN7: renomear o projeto pelo título ──
  function iniciarRenomearTitulo(): void {
    tituloRascunho = planName;
    tituloCancelado = false;
    renomeandoTitulo = true;
  }

  function confirmarTitulo(): void {
    if (!renomeandoTitulo) return;
    renomeandoTitulo = false;
    if (tituloCancelado) return;
    if (tituloRascunho !== planName) {
      planName = tituloRascunho;
      marcarAlteracao();
    }
  }

  function onTituloKeydown(e: KeyboardEvent): void {
    if (e.key === 'Enter') {
      e.preventDefault();
      confirmarTitulo();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      tituloCancelado = true;
      renomeandoTitulo = false;
    }
  }
</script>

{#if modo === 'lista'}
  <ProjetosLista
    {planos}
    carregando={listaCarregando}
    erro={listaErro}
    {abrindo}
    rascunho={rascunhoInfo}
    onabrir={(p) => void abrirProjeto(p)}
    onabrirRascunho={() => void abrirRascunho()}
    onnovo={novoProjeto}
    onexcluir={(p) => void excluirProjeto(p)}
    onrecarregar={() => void carregarLista()}
  />
{:else}
<div style="display: flex; flex-direction: column; height: 100%; min-height: 0;">
  <Header kicker={t('projectKicker')} back={{ label: t('backToProjects'), onclick: voltarParaLista }}>
    {#snippet titleSlot()}
      {#if renomeandoTitulo}
        <input
          bind:this={tituloInputEl}
          bind:value={tituloRascunho}
          aria-label={t('projectNameLabel')}
          maxlength="200"
          placeholder={planNamePlaceholder}
          onkeydown={onTituloKeydown}
          onblur={confirmarTitulo}
          style="min-width: 220px; height: 36px; padding: 0 10px; border: 2px solid var(--color-accent); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-family: inherit; font-size: 17px;"
        />
      {:else}
        <span style="display: inline-flex; align-items: center; gap: 6px; min-width: 0;">
          <span
            style="font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: {planName.trim() ? 'var(--color-text)' : 'var(--color-neutral-500)'};"
          >{planName.trim() || planNamePlaceholder}</span>
          <button
            class="t2-icon-btn"
            aria-label={t('renameProjectBtn')}
            title={t('renameProjectBtn')}
            onclick={iniciarRenomearTitulo}
            style="width: 44px; height: 44px; flex-shrink: 0;"
          ><i class="ph ph-pencil-simple" style="font-size: 17px;"></i></button>
        </span>
      {/if}
    {/snippet}
    {#snippet actions()}
      <button
        class="t2-newphoto-btn"
        style="display: inline-flex; align-items: center; gap: 10px; min-height: 52px; padding: 4px 18px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; text-align: left;"
        onclick={() => fileInputEl?.click()}
      >
        <i class="ph ph-camera" style="font-size: 19px;"></i>
        <span style="display: flex; flex-direction: column; line-height: 1.2;">
          <span>{t('newPhoto')}</span>
          <span style="font-size: 11.5px; font-weight: 400; color: var(--color-neutral-400);">{t('uploadHint', { mb: tetoMb })}</span>
        </span>
      </button>
      <input type="file" accept={TIPOS_IMAGEM_ACEITOS.join(',')} bind:this={fileInputEl} onchange={onFileInput} style="display: none;" />
    {/snippet}
  </Header>

  <div
    style="flex-shrink: 0; display: flex; align-items: center; gap: 10px; padding: 8px 20px; border-bottom: 1px solid var(--color-line);"
  >
    <div
      role="tablist"
      aria-label={t('tabsListLabel')}
      style="flex: 1; min-width: 0; display: flex; align-items: center; gap: 6px; overflow-x: auto;"
    >
    {#each tabs as aba, i (aba.uid)}
      {#if renamingTabIndex === i}
        <span id={`t2-tab-btn-${i}`} style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{displayTabName(aba, i)}</span>
        <label for={`t2-tab-rename-${aba.uid}`} style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{t('renameTabLabel')}</label>
        <input
          id={`t2-tab-rename-${aba.uid}`}
          type="text"
          bind:value={tabs[i].name}
          maxlength="80"
          onblur={commitRename}
          onkeydown={onRenameKeydown}
          style="min-width: 120px; height: 44px; padding: 0 10px; border: 2px solid var(--color-accent); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-family: inherit; font-size: 13.5px; flex-shrink: 0;"
        />
      {:else}
        <button
          id={`t2-tab-btn-${i}`}
          role="tab"
          data-tab-index={i}
          aria-selected={i === activeTabIndex}
          aria-controls="t2-tab-panel"
          tabindex={i === activeTabIndex ? 0 : -1}
          class="t2-tab-btn"
          onclick={() => switchTab(i)}
          ondblclick={() => startRename(i)}
          onkeydown={(e) => onTabKeydown(e, i)}
          style="display: inline-flex; flex-direction: column; align-items: flex-start; justify-content: center; gap: 2px; min-width: 44px; min-height: 44px; padding: 4px 12px; border: 2px solid {i === activeTabIndex ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {i === activeTabIndex ? 'var(--color-accent-panel)' : 'var(--color-bg)'}; color: {i === activeTabIndex ? 'var(--color-accent-400)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: {i === activeTabIndex ? '700' : '500'}; cursor: pointer; flex-shrink: 0;"
        >
          <span>{displayTabName(aba, i)}</span>
          <span class="font-mono" style="font-size: 11px; opacity: 0.85;">{tabProgressLabel(i)}</span>
        </button>
      {/if}
    {/each}
      <button
        aria-label={t('addTabBtn')}
        title={t('addTabBtn')}
        onclick={addTab}
        style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); cursor: pointer; flex-shrink: 0;"
      ><i class="ph ph-plus" style="font-size: 18px;"></i></button>
    </div>

    <div aria-hidden="true" style="flex-shrink: 0; width: 1px; align-self: stretch; margin: 6px 0; background: var(--color-line);"></div>

    <div style="flex-shrink: 0; display: flex; align-items: center; gap: 6px;">
      <button
        aria-label={t('moveTabLeft')}
        title={t('moveTabLeft')}
        disabled={activeTabIndex === 0}
        onclick={() => moveTab(activeTabIndex, -1)}
        style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer; opacity: {activeTabIndex === 0 ? 0.4 : 1};"
      ><i class="ph ph-caret-left" style="font-size: 15px;"></i></button>
      <button
        aria-label={t('moveTabRight')}
        title={t('moveTabRight')}
        disabled={activeTabIndex === tabs.length - 1}
        onclick={() => moveTab(activeTabIndex, 1)}
        style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer; opacity: {activeTabIndex === tabs.length - 1 ? 0.4 : 1};"
      ><i class="ph ph-caret-right" style="font-size: 15px;"></i></button>
      <button
        aria-label={t('deleteTabBtn')}
        title={t('deleteTabBtn')}
        onclick={() => onDeleteTabClick(activeTabIndex)}
        style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-500); cursor: pointer;"
      ><i class="ph ph-trash-simple" style="font-size: 15px;"></i></button>
    </div>
  </div>

  <div
    role="tabpanel"
    id="t2-tab-panel"
    aria-labelledby={`t2-tab-btn-${activeTabIndex}`}
    style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 34%);"
  >
    <div style="position: relative; border-right: 1px solid var(--color-line); overflow: hidden;">
      <div
        bind:this={boxEl}
        style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: repeating-conic-gradient(var(--color-panel) 0% 25%, var(--color-bg) 0% 50%) 0 0 / 40px 40px; cursor: crosshair;"
      >
        {#if hasImage}
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
            <span style="font-size: 13px; color: var(--color-neutral-400);">{t('uploadHint', { mb: tetoMb })}</span>
          </div>
        {/if}
      </div>

      <!-- rf-16 RN8: barra única sobre a foto — zoom (rf-06) e, depois do
           separador, Salvar e Exportar. Aparece sempre; sem foto o zoom fica
           desabilitado. -->
      <div
        style="position: absolute; right: 20px; top: 20px; display: flex; flex-direction: column; align-items: flex-end; gap: 6px;"
      >
        <div
          style="display: flex; align-items: center; gap: 6px; padding: 6px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: rgba(22,24,38,0.92); backdrop-filter: blur(8px);"
        >
          <button class="t2-icon-btn t2-zoom-btn" aria-label={t('zoomOut')} title={t('zoomOut')} disabled={!hasImage} onclick={() => zoomBy(1 / 1.25)}>
            <i class="ph ph-minus" style="font-size: 18px;"></i>
          </button>
          <span
            aria-live="polite"
            style="min-width: 58px; text-align: center; font-size: 13px; color: var(--color-neutral-400); font-variant-numeric: tabular-nums;"
          >{hasImage ? `${zoomPercent(view)}%` : '—'}</span>
          <button class="t2-icon-btn t2-zoom-btn" aria-label={t('zoomIn')} title={t('zoomIn')} disabled={!hasImage} onclick={() => zoomBy(1.25)}>
            <i class="ph ph-plus" style="font-size: 18px;"></i>
          </button>
          <button class="t2-icon-btn t2-zoom-btn" aria-label={t('zoomFit')} title={t('zoomFit')} disabled={!hasImage} onclick={fitToBox}>
            <i class="ph ph-corners-out" style="font-size: 18px;"></i>
          </button>

          <div aria-hidden="true" style="width: 1px; align-self: stretch; margin: 4px 2px; background: var(--color-neutral-800);"></div>

          <button
            class="t2-icon-btn t2-save-plan-btn"
            aria-label={algumCalculando ? t('saveWaitCalc') : t('savePlanBtn')}
            title={algumCalculando ? t('saveWaitCalc') : t('savePlanBtn')}
            disabled={saveState === 'saving' || algumCalculando}
            onclick={handleSave}
          >
            {#if saveState === 'saving'}
              <Spinner size={18} label={t('planSaving')} />
            {:else if saveFlash}
              <i class="ph-bold ph-check" style="font-size: 18px; color: var(--color-accent-400);"></i>
            {:else}
              <i class="ph ph-floppy-disk" style="font-size: 18px;"></i>
            {/if}
          </button>

          <div style="position: relative;">
            <button
              class="t2-icon-btn t2-export-btn"
              aria-label={exportDisabled && !reportGenerating ? t('exportSaveFirst') : t('exportReportBtn')}
              title={exportDisabled && !reportGenerating ? t('exportSaveFirst') : t('exportReportBtn')}
              aria-haspopup="menu"
              aria-expanded={exportMenuOpen}
              disabled={exportDisabled}
              onclick={() => (exportMenuOpen = !exportMenuOpen)}
            >
              {#if reportGenerating}
                <Spinner size={18} label={t('exportGenerating')} />
              {:else}
                <i class="ph ph-file-arrow-down" style="font-size: 18px;"></i>
              {/if}
            </button>
            {#if exportMenuOpen}
              <div
                role="menu"
                style="position: absolute; right: 0; top: 52px; z-index: 5; display: flex; flex-direction: column; min-width: 140px; padding: 6px; border: 1px solid var(--color-neutral-800); border-radius: 10px; background: var(--color-panel);"
              >
                <button role="menuitem" class="t2-menu-item" onclick={() => void handleExport('pdf')}>{t('exportPdf')}</button>
                <button role="menuitem" class="t2-menu-item" onclick={() => void handleExport('png')}>{t('exportPng')}</button>
              </div>
            {/if}
          </div>
        </div>

        <div aria-live="polite" style="max-width: 320px; text-align: right; font-size: 12.5px; color: var(--color-neutral-300); text-shadow: 0 1px 2px rgba(0,0,0,0.6);">
          {#if validationErrorText}
            <span role="alert"><i class="ph ph-warning-circle" style="margin-right: 6px;"></i>{validationErrorText}</span>
          {:else if saveState === 'saving'}{t('planSaving')}
          {:else if saveState === 'saved'}{t('planSavedAt', { hora: savedAtLabel })}
          {:else if saveState === 'error' && saveErrorKey}{t(saveErrorKey)}
          {:else if reportGenerating}{t('exportGenerating')}
          {:else if reportErrorKey}{t(reportErrorKey)}
          {/if}
        </div>
      </div>

      <div style="position: absolute; left: 20px; bottom: 20px; display: flex; align-items: center; gap: 14px; padding: 12px 18px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: rgba(22,24,38,0.92); backdrop-filter: blur(8px);">
        <span style="font-size: 15px; color: var(--color-neutral-400);">{t('progress', { a: paintedCount, b: totalRegionsCount })}</span>
        <span style="width: 160px; height: 6px; border-radius: 999px; background: var(--color-rule); overflow: hidden;">
          <span style="display: block; height: 6px; width: {progressPct}%; background: var(--color-accent);"></span>
        </span>
      </div>
    </div>

    <div style="min-height: 0; display: flex; flex-direction: column;">
      <div style="flex: 1; min-height: 0; overflow-y: auto; padding: 20px;">
        {#if draftBannerVisible}
          <div role="status" aria-live="polite" style="display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px; padding: 10px 12px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: var(--color-accent-panel); font-size: 13.5px; color: var(--color-text);">
            <span>{t('draftRestored')}</span>
            <button
              class="t2-discard-btn"
              style="min-width: 44px; min-height: 44px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); font-family: inherit; font-size: 13px; cursor: pointer;"
              onclick={onDiscardDraftClick}
            >{t('discardDraftBtn')}</button>
          </div>
        {/if}

        {#if autoSaveStatus === 'sem-foto'}
          <div aria-live="polite" style="margin-bottom: 12px; padding: 8px 12px; border-radius: 8px; background: var(--color-raised); font-size: 13px; color: var(--color-neutral-400);">
            <i class="ph ph-warning" style="margin-right: 6px;"></i>{t('draftNoPhoto')}
          </div>
        {:else if autoSaveStatus === 'desligado'}
          <div role="alert" style="margin-bottom: 12px; padding: 8px 12px; border-radius: 8px; border: 1px solid var(--color-neutral-800); background: var(--color-raised); font-size: 13px; color: var(--color-neutral-300);">
            <i class="ph ph-warning-circle" style="margin-right: 6px;"></i>{t('autosaveOff')}
          </div>
        {/if}

        <p class="section-label" style="margin: 0 0 10px;">{t('paintWithMine')}</p>
        <Combobox
          options={manufacturers}
          value={manufacturerId}
          onchange={(id) => { if (id != null) void onManufacturerChange(id); }}
          placeholder={t('chooseSupplierPlaceholder')}
          searchPlaceholder={t('phFindMaker')}
          emptyText={t('noMakerFound')}
          ariaLabel={t('paintWithMine')}
          moreText={(n) => t('comboMore', { n })}
        />

        <label
          style="display: flex; align-items: center; gap: 10px; min-height: 44px; margin-top: 12px; font-size: 13.5px; color: var(--color-neutral-300); cursor: pointer;"
        >
          <input
            type="checkbox"
            role="switch"
            checked={useStockOnly}
            onchange={(e) => void onUseStockOnlyChange(e.currentTarget.checked)}
            style="width: 20px; height: 20px; accent-color: var(--color-accent);"
          />
          {t('onlyMyStock')}
        </label>

        {#if avisoUniversoProjeto}
          <p role="alert" style="margin: 8px 0 0; font-size: 12.5px; color: var(--color-warning, #E4A11B);">
            {avisoUniversoProjeto}
          </p>
        {/if}

        {#if regiaoNoDialogo !== null}
          <div
            bind:this={dialogoElemento}
            role="dialog"
            aria-modal="true"
            aria-labelledby="t2-fallback-titulo"
            tabindex="-1"
            onkeydown={onDialogoKeydown}
            style="margin: 14px 0; padding: 16px; border: 1px solid var(--color-accent-700); border-radius: 14px; background: var(--color-accent-panel); outline: none;"
          >
            <p id="t2-fallback-titulo" style="margin: 0 0 6px; font-size: 14px; font-weight: 600; color: var(--color-neutral-100);">
              {t('fallbackTitulo')}
            </p>
            <p style="margin: 0 0 12px; font-size: 12.5px; color: var(--color-neutral-400);">
              {t('fallbackExplicacao')}
            </p>

            {#if manufacturerId != null}
              <button
                style="display: block; width: 100%; min-height: 44px; margin-bottom: 8px; padding: 0 14px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 13.5px; cursor: pointer;"
                onclick={() => void escolherSaida('marca')}
              >
                {t('fallbackAbrirMarca')}
                {#if dialogoMelhorMarca !== null}· ΔE {decimal(dialogoMelhorMarca, 1)}{/if}
              </button>
            {/if}

            <button
              style="display: block; width: 100%; min-height: 44px; margin-bottom: 8px; padding: 0 14px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 13.5px; cursor: pointer;"
              onclick={() => void escolherSaida('todos')}
            >
              {t('fallbackAbrirTodos')}
              {#if dialogoMelhorTodos !== null}· ΔE {decimal(dialogoMelhorTodos, 1)}{/if}
            </button>

            <button
              style="display: block; width: 100%; min-height: 44px; border: none; background: transparent; color: var(--color-neutral-500); font-family: inherit; font-size: 13px; cursor: pointer;"
              onclick={fecharDialogo}
            >{t('fallbackManter')}</button>
          </div>
        {/if}

        <p class="section-label" style="margin: 22px 0 12px;">{t('regions')}</p>
        <div style="display: flex; flex-direction: column; gap: 10px;">
          {#each regions as r, i (r.id)}
            {@const selecionada = selectedId === r.id}
            <div
              class="t2-region-card"
              style="border: 1px solid {selecionada ? 'var(--color-accent-700)' : 'var(--color-neutral-800)'}; border-radius: 14px; background: {selecionada ? 'var(--color-accent-panel)' : 'var(--color-panel)'};"
            >
              <div
                id={`t2-region-card-${r.id}`}
                class="t2-region-head"
                role="button"
                tabindex="0"
                aria-expanded={selecionada}
                onclick={() => selectRegion(r.id)}
                onkeydown={(e) => { if (e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); selectRegion(r.id); } }}
                style="display: flex; align-items: center; gap: 12px; padding: 14px; border-radius: 14px; cursor: pointer;"
              >
                <span style="width: 34px; height: 34px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: {r.hex}; flex-shrink: 0;"></span>
                <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                  <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{regionName(i)}</span>
                  <span class="font-mono" style="font-size: 12.5px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{regionMatchText(r)}</span>
                  {#if seloDaFaixa(r)}
                    {@const selo = seloDaFaixa(r)}
                    <span style="font-size: 11.5px; font-weight: 600; color: {selo?.cor};">{selo?.texto}</span>
                  {/if}
                  {#if r.result?.foraDoUniverso}
                    <span style="font-size: 11.5px; color: var(--color-warning, #E4A11B);">{t('foraDoUniversoLabel')}</span>
                  {/if}
                </span>
                <button
                  class="t2-done-btn"
                  aria-pressed={r.painted}
                  style="display: inline-flex; align-items: center; gap: 8px; height: 44px; padding: 0 12px; border: 1px solid {r.painted ? 'var(--color-accent-700)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: transparent; color: {r.painted ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 13px; font-weight: 500; cursor: pointer; flex-shrink: 0;"
                  onclick={(e) => { e.stopPropagation(); togglePainted(r); }}
                  onkeydown={(e) => e.stopPropagation()}
                >
                  <i class="ph-bold ph-check" style="font-size: 14px;"></i>{r.painted ? t('painted') : t('toPaint')}
                </button>
              </div>

              {#if selecionada}
                <!-- rf-16 RN17/RN18: o detalhe mora no próprio card. -->
                <div style="padding: 0 14px 16px; display: flex; flex-direction: column; gap: 14px;">
                  {#if r.computing}
                    <div style="display: flex; align-items: center; justify-content: center; gap: 12px; height: 64px; border-radius: 8px; background: var(--color-raised); font-size: 14px; color: var(--color-neutral-500);">
                      <Spinner size={22} label={t('calculating')} />{t('calculating')}
                    </div>
                  {:else if r.result}
                    {@const res = r.result}
                    {@const vc = verdictKeys(res.deltaE)}
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
                      <div style="display: flex; align-items: center; gap: 10px;">
                        <span style="width: 34px; height: 34px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: {r.hex}; flex-shrink: 0;"></span>
                        <span style="display: flex; flex-direction: column; min-width: 0;">
                          <span style="font-size: 11.5px; color: var(--color-neutral-500);">{t('regionDetailPieceColor')}</span>
                          <span class="font-mono" style="font-size: 12.5px; color: var(--color-text);">{hexDe(r.r, r.g, r.b)} · {r.r} {r.g} {r.b}</span>
                        </span>
                      </div>
                      <div style="display: flex; align-items: center; gap: 10px;">
                        <span style="width: 34px; height: 34px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({res.resultR}, {res.resultG}, {res.resultB}); flex-shrink: 0;"></span>
                        <span style="display: flex; flex-direction: column; min-width: 0;">
                          <span style="font-size: 11.5px; color: var(--color-neutral-500);">{t('regionDetailMixColor')}</span>
                          <span class="font-mono" style="font-size: 12.5px; color: var(--color-text);">{hexDe(res.resultR, res.resultG, res.resultB)} · {res.resultR} {res.resultG} {res.resultB}</span>
                        </span>
                      </div>
                    </div>

                    <div style="display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap;">
                      <span class="font-mono" style="font-size: 26px; font-weight: 500; line-height: 1; color: {verdictColor(res.deltaE)};">{decimal(res.deltaE, 1)}</span>
                      <span style="font-size: 11px; color: var(--color-neutral-500);">ΔE00</span>
                      {#if seloDaFaixa(r)}
                        {@const selo = seloDaFaixa(r)}
                        <span style="font-size: 12.5px; font-weight: 600; color: {selo?.cor};">{selo?.texto}</span>
                      {/if}
                    </div>
                    <p style="margin: 0; font-size: 13.5px; color: var(--color-neutral-300); text-wrap: pretty;">{t(vc.c)}</p>
                    <p style="margin: 0; font-size: 12.5px; color: var(--color-neutral-500); text-wrap: pretty;">
                      {#if res.method}{t('regionDetailMethod')}: {res.method} · {/if}{t('regionDetailDeltaHelp')}
                    </p>

                    <div>
                      <p class="section-label" style="margin: 0 0 8px;">{t('regionDetailIngredients')}</p>
                      <div style="display: flex; flex-direction: column; gap: 6px;">
                        {#each [...res.ingredients].sort((a, b) => b.percentage - a.percentage) as ing, k (k)}
                          <div style="display: flex; align-items: center; gap: 10px;">
                            <span style="width: 22px; height: 22px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({ing.r}, {ing.g}, {ing.b}); flex-shrink: 0;"></span>
                            <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                              <span style="font-size: 14px; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.name}</span>
                              <span style="font-size: 12px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.code}{#if ing.code && ing.manufacturer} · {/if}{ing.manufacturer ?? ''}</span>
                            </span>
                            <span class="font-mono" style="width: 56px; text-align: right; font-size: 14px; font-weight: 500; color: var(--color-neutral-300); flex-shrink: 0;">{Math.round(ing.percentage)}%</span>
                          </div>
                        {/each}
                      </div>
                    </div>

                    {#if res.tips && res.tips.length > 0}
                      <div>
                        <p class="section-label" style="margin: 0 0 6px;">{t('regionDetailTips')}</p>
                        <ul style="margin: 0; padding-left: 18px; font-size: 13px; color: var(--color-neutral-400);">
                          {#each res.tips as tip, k (k)}<li>{tip}</li>{/each}
                        </ul>
                      </div>
                    {/if}

                    {#if r.override && res.faixa === 'nao-encontrei'}
                      <p role="status" style="margin: 0; font-size: 12.5px; color: var(--color-warning, #E4A11B);">{t('regionOverrideNoMatch')}</p>
                    {/if}
                  {:else}
                    <p style="margin: 0; font-size: 13.5px; color: var(--color-neutral-400);">{r.vazioMotivo ?? t('noSimilarPaint')}</p>
                  {/if}

                  <p style="margin: 0; font-size: 12.5px; color: var(--color-neutral-400);">
                    {origemDaRegiao(r)}{#if r.result?.foraDoUniverso} · {t('foraDoUniversoLabel')}{/if}
                  </p>

                  <!-- rf-16 RN19: ajuste de fornecedor só desta região. -->
                  <div style="display: flex; flex-direction: column; gap: 8px; padding-top: 12px; border-top: 1px solid var(--color-line);">
                    <p class="section-label" style="margin: 0;">{t('regionAdjustTitle')}</p>
                    <Combobox
                      options={manufacturers}
                      value={r.override ? r.regionManufacturerId : null}
                      onchange={(id) => void ajustarFornecedorDaRegiao(r, id)}
                      placeholder={t('regionFollowProject', { marca: nomeFabricante(manufacturerId) || t('allMakers') })}
                      clearLabel={t('regionFollowProject', { marca: nomeFabricante(manufacturerId) || t('allMakers') })}
                      searchPlaceholder={t('phFindMaker')}
                      emptyText={t('noMakerFound')}
                      ariaLabel={t('regionAdjustTitle')}
                      moreText={(n) => t('comboMore', { n })}
                    />
                    <label style="display: flex; align-items: center; gap: 10px; min-height: 44px; font-size: 13.5px; color: {r.override ? 'var(--color-neutral-300)' : 'var(--color-neutral-600)'}; cursor: {r.override ? 'pointer' : 'default'};">
                      <input
                        type="checkbox"
                        role="switch"
                        disabled={!r.override}
                        checked={r.override ? r.regionUseStockOnly : useStockOnly}
                        onchange={(e) => void ajustarEstoqueDaRegiao(r, e.currentTarget.checked)}
                        style="width: 20px; height: 20px; accent-color: var(--color-accent);"
                      />
                      {t('onlyMyStock')}
                    </label>
                    <div style="display: flex; gap: 8px; flex-wrap: wrap;">
                      <button class="t2-secondary-btn" disabled={r.computing} onclick={() => void recalcularUmaRegiao(r.id)}>
                        <i class="ph ph-arrows-clockwise" style="font-size: 16px;"></i>{t('regionRecalcBtn')}
                      </button>
                      <button class="t2-secondary-btn" onclick={removeSelected}>
                        <i class="ph ph-trash-simple" style="font-size: 16px;"></i>{t('removeRegionBtn')}
                      </button>
                    </div>
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        </div>

        <p class="section-label" style="margin: 26px 0 12px;">{t('piecePaints')}</p>
        <div style="display: flex; flex-direction: column;">
          {#each shoppingItems as s (s.key)}
            {@const owned = ownedPaints.has(s.key)}
            <div style="display: flex; align-items: center; gap: 12px; min-height: 56px; padding: 8px 2px; border-bottom: 1px solid var(--color-line);">
              <button
                aria-pressed={owned}
                style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: none; background: transparent; cursor: pointer; flex-shrink: 0;"
                onclick={() => toggleOwned(s.key)}
              >
                <span style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {owned ? 'var(--color-accent)' : 'var(--color-field-border)'}; border-radius: 4px; background: {owned ? 'var(--color-accent)' : 'transparent'};">
                  {#if owned}<i class="ph-bold ph-check" style="font-size: 15px; color: var(--color-accent-100);"></i>{/if}
                </span>
              </button>
              <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                <span style="font-size: 15px; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{s.name}</span>
                <span class="font-mono" style="font-size: 12.5px; color: var(--color-neutral-500);">{s.code} · {s.manufacturer}</span>
                <span style="font-size: 11.5px; color: var(--color-neutral-600);">{t('usedInTabsLabel', { tabs: s.tabs.join(', ') })}</span>
              </span>
              <span style="font-size: 12px; letter-spacing: 0.06em; text-transform: uppercase; color: {owned ? 'var(--color-accent-2)' : 'var(--color-neutral-500)'}; flex-shrink: 0;">{owned ? t('tagHave') : t('tagBuy')}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
</div>
{/if}

<style>
  .t2-newphoto-btn:hover {
    background: var(--color-accent-hover);
  }

  .t2-tab-btn:hover {
    border-color: var(--color-accent-700);
  }

  .t2-tab-btn:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }

  .t2-region-card:hover {
    border-color: var(--color-accent-700) !important;
  }

  .t2-region-head:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }

  .t2-done-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-300);
    cursor: pointer;
  }

  .t2-icon-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-icon-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .t2-menu-item {
    min-height: 44px;
    padding: 0 12px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--color-text);
    font-family: inherit;
    font-size: 14px;
    text-align: left;
    cursor: pointer;
  }

  .t2-menu-item:hover {
    background: var(--color-accent-panel);
  }

  .t2-secondary-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-300);
    font-family: inherit;
    font-size: 13px;
    cursor: pointer;
  }

  .t2-secondary-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t2-secondary-btn:disabled {
    opacity: 0.45;
    pointer-events: none;
  }

  .t2-discard-btn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }
</style>
