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
  import LupaPrecisao from '../components/LupaPrecisao.svelte';
  import {
    fitView, toImage, toScreen, zoomAround, zoomPercent, type View,
  } from '../planner/viewport';
  import {
    suggestRecipeForColor, melhorDeltaE, ehUniversoVazio, recipeForChosenPaint,
    type EquivalentRecipe, type UniversoBusca, type StockPaint,
  } from '../services/engine';
  import { allManufacturers, allPaints, paintById } from '../services/catalog';
  import { stock, estoqueAssentado } from '../services/stock.svelte';
  import { verdictKeys } from '../ui';
  import { t, decimal, type DictKey } from '../i18n.svelte';
  import { toast } from '../toast.svelte';
  import { baixarTexto } from '../services/download';
  import { nav } from '../nav.svelte';
  import {
    lerRascunho, agendarGravacao, apagarRascunho, gravarPendente, estadoAutoSave,
    type Rascunho, type RegiaoRascunho, type AbaRascunho, type EstadoAutoSave,
  } from '../planner/rascunho';
  import {
    montarArquivoPlano, nomeArquivoPlano, lerArquivoPlano, TETO_ARQUIVO_PLANO,
    type PlanoImportado,
  } from '../planner/arquivoPlano';
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
    /** rf-18 RN1-RN5: tinta escolhida à mão — fora de todo recálculo automático. */
    manual: boolean;
    /** rf-18 RN6/RN7: cor lida da foto antes de uma correção; `''` = nunca corrigida. */
    sampleHex: string;
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

  /** rf-22 RN1: o estado de antes de uma ação — só as regiões (cópia) e a
   *  seleção. Foto, `view` e `nextId` ficam de fora: `nextId` nunca volta, e um
   *  id nunca é reusado. `rotulo` é a chave i18n do toast. */
  interface Retrato {
    regions: PlannerRegion[];
    selectedId: number | null;
    rotulo: DictKey;
  }

  interface Historico {
    desfazer: Retrato[];
    refazer: Retrato[];
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
    /** rf-22 RN1: pilha de desfazer/refazer da aba (só em memória). */
    historico: Historico;
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
  let historico: Historico = $state({ desfazer: [], refazer: [] });

  let view: View = $state({ zoom: 1, panX: 0, panY: 0 });

  // ── rf-18 RN10-RN14: ferramenta mão e Pointer Events ──
  let ferramenta = $state<'marcar' | 'mover' | 'ajustar'>('marcar');
  let arrastando = $state(false);
  /** rf-22 RN15: arquivo sendo arrastado sobre a caixa da foto. */
  let soltando = $state(false);
  /** rf-22 RN16/RN17: lupa de precisão (null = fechada). */
  let lupa: { ix: number; iy: number; hex: string; x: number; y: number } | null = $state(null);
  /** Ponteiros ativos — não é `$state`: só orienta o cálculo do gesto a cada
   *  evento, nunca é lido pelo template. */
  const ponteiros = new Map<number, { x: number; y: number }>();
  /** Registro do gesto corrente (não reativo, mesmo motivo). */
  const gesto = {
    multi: false,
    startX: 0,
    startY: 0,
    maxDist: 0,
    lastDist: 0,
    lastMid: { x: 0, y: 0 },
  };

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
      historico: { desfazer: [], refazer: [] },
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
      imageDataUrl, hasImage, image, regions, nextId, selectedId, view, historico,
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
    historico = aba.historico;
    regiaoNoDialogo = null;
    limparPonteiros();
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

  // Achado 9 do guardrail: `{#if hasImage}` desmonta o <canvas> — um dedo no
  // chão nesse instante deixaria o id velho no mapa de ponteiros.
  $effect(() => {
    if (!hasImage) limparPonteiros();
  });

  let selectedRegion = $derived(regions.find(r => r.id === selectedId) ?? null);

  // ── rf-18 RN6/RN7: correção da cor da região selecionada ──
  let corTexto = $state('');
  let corInvalida = $state(false);
  $effect(() => {
    const reg = selectedRegion;
    corInvalida = false;
    corTexto = reg ? reg.hex : '';
  });

  // ── rf-18 RN1: opções do Combobox "Escolher a tinta eu mesmo" ──
  let opcoesTintaManual = $derived.by(() => {
    const doEstoque = stock.paints.map(p => ({
      id: p.id,
      name: p.name,
      hint: `${p.manufacturer} · ${p.code} · ${t('paintHintHave')}`,
      searchText: `${p.name} ${p.code} ${p.manufacturer}`,
    }));
    const doCatalogo = allPaints().map(p => ({
      id: p.id,
      name: p.name,
      hint: `${p.manufacturer} · ${p.code}`,
      searchText: `${p.name} ${p.code} ${p.manufacturer}`,
    }));
    return [...doEstoque, ...doCatalogo];
  });
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
  /** PDF/PNG exigem plano salvo; o arquivo JSON (rf-22 RN19) exporta o estado
   *  de trabalho, salvo ou não. */
  let relatorioDisabled = $derived(planId == null || rascunhoPendente || reportGenerating);
  let algumaAbaComFoto = $derived(tabs.some((a, i) => (i === activeTabIndex ? hasImage : a.hasImage)));
  let cursorDaFoto = $derived(
    ferramenta === 'marcar' ? 'crosshair'
      : ferramenta === 'ajustar' ? (arrastando ? 'grabbing' : 'pointer')
      : arrastando ? 'grabbing' : 'grab',
  );

  /** rf-17: a tinta do ingrediente está em "Minhas tintas"? O mesmo pote
   *  chega de três formas — id negativo do estoque, id de catálogo de onde a
   *  linha do estoque nasceu, ou só marca + código (tinta do catálogo que o
   *  pintor também tem). Sem o terceiro caso, uma receita calculada no
   *  catálogo inteiro marcava "não tenho" em pote que está na estante. */
  function temEmEstoque(ing: IngredienteDTO): boolean {
    const cod = ing.code.trim().toLowerCase();
    const nom = ing.name.trim().toLowerCase();
    return stock.paints.some(p =>
      p.id === ing.paintId ||
      (p.catalogId != null && p.catalogId === ing.paintId) ||
      (p.manufacturerId === ing.manufacturerId &&
        (cod !== '' ? p.code.trim().toLowerCase() === cod : p.name.trim().toLowerCase() === nom)));
  }

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
      // RN5: trocar a foto zera a pilha desta aba.
      historico.desfazer = [];
      historico.refazer = [];
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

  /** rf-22 RN11: a entrada única de foto — botão, colar e arrastar. As
   *  guardas ficam todas aqui, num ponto só. */
  function receberImagem(arquivo: Blob, origem: 'botao' | 'colar' | 'arrastar'): void {
    const limpaBotao = () => { if (origem === 'botao' && fileInputEl) fileInputEl.value = ''; };
    // rf-16 RN12: tipo e tamanho barrados ANTES de ler o arquivo — o usuário
    // já viu o limite na tela, e ler 20 MB só para recusar gasta memória.
    if (!TIPOS_IMAGEM_ACEITOS.includes(arquivo.type)) {
      toast(t('errImageType'), 'error');
      limpaBotao();
      return;
    }
    if (arquivo.size > tetoArquivoImagem()) {
      toast(t('errImageMax', { mb: tetoMb }), 'error');
      limpaBotao();
      return;
    }
    // RN13: a troca de foto não entra no desfazer, então pergunta antes.
    if (regions.length > 0) {
      const nome = displayTabName(tabs[activeTabIndex], activeTabIndex);
      if (!window.confirm(t('replacePhotoConfirm', { name: nome, n: regions.length }))) {
        limpaBotao();
        return;
      }
    }
    const reader = new FileReader();
    reader.onload = () => {
      const src = reader.result as string;
      // D-005: a mesma guarda da CA17 continua como rede de segurança.
      const erroImagem = validarImagemDaAba(src);
      if (erroImagem) {
        toast(t(erroImagem as DictKey, { mb: tetoMb }), 'error');
        limpaBotao();
        return;
      }
      loadImage(src);
      limpaBotao();
    };
    reader.readAsDataURL(arquivo);
  }

  function onFileInput(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    receberImagem(file, 'botao');
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

  /** rf-22 RN10: gesto de "ajustar" — o pino que está sendo arrastado. Guarda
   *  a região pela referência (não pelo binding vivo `regions`): trocar de aba
   *  no meio do gesto não pode restaurar o pino errado. Não reativo. */
  const pino = {
    reg: null as PlannerRegion | null,
    ox: 0,
    oy: 0,
    movendo: false,
    retrato: null as Retrato | null,
  };
  /** rf-22 RN16: espera de 500 ms do toque longo no "marcar", e se a lupa
   *  aberta é a dele. Não reativo. */
  let timerLupa: ReturnType<typeof setTimeout> | null = null;
  let lupaDoMarcar = false;
  const LIMIAR_TOQUE_PX = 6;
  const TOQUE_LONGO_MS = 500;

  /** RN11/RN13: a região só nasce/seleciona no `pointerup`, e só se o gesto
   *  não passou de 1 ponteiro nem de 6px CSS de distância máxima. rf-22: o
   *  toque longo (500 ms parado) abre a lupa no "marcar"; no "ajustar", o
   *  `pointerdown` sobre um pino começa o arraste dele. */
  function onPointerDown(e: PointerEvent) {
    if (!canvasEl || !image) return;
    if (e.pointerType === 'mouse' && e.button !== 0) return;
    canvasEl.setPointerCapture?.(e.pointerId);
    ponteiros.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (ponteiros.size <= 1) {
      gesto.multi = false;
      gesto.startX = e.clientX;
      gesto.startY = e.clientY;
      gesto.maxDist = 0;
      if (ferramenta === 'marcar') agendarLupa();
      else if (ferramenta === 'ajustar') iniciarPino(e);
    } else if (ponteiros.size === 2) {
      gesto.multi = true;
      // RN10/RN16: um segundo dedo vira pinça — o pino volta ao lugar e a
      // lupa fecha sem criar nada.
      cancelarPino();
      fecharLupa();
      reinitGestoPinca();
    }
  }

  /** Achado 8 do guardrail: recalcula `lastDist`/`lastMid` a partir do par
   *  ATUAL de ponteiros sempre que a contagem vira exatamente 2 — no
   *  `pointerdown` (1→2) e no `pointerup`/`pointercancel` (3→2). Sem isso, o
   *  próximo `pointermove` compara a distância do par novo com a do trio
   *  anterior e o zoom salta. */
  function reinitGestoPinca(): void {
    const pts = [...ponteiros.values()];
    gesto.lastDist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
    gesto.lastMid = { x: (pts[0].x + pts[1].x) / 2, y: (pts[0].y + pts[1].y) / 2 };
  }

  /** RN12: pinça (zoom + deslocamento pelo meio dos dedos) com 2+ ponteiros;
   *  RN13: com 1 ponteiro, "mover" arrasta a foto e "marcar" só acumula a
   *  distância máxima do gesto. rf-22: "ajustar" arrasta o pino, ou a foto
   *  quando o toque não pegou pino. */
  function onPointerMove(e: PointerEvent) {
    if (!canvasEl || !image) return;
    const anterior = ponteiros.get(e.pointerId);
    if (!anterior) return;
    ponteiros.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (ponteiros.size >= 2) {
      const pts = [...ponteiros.values()];
      const dist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
      const mid = { x: (pts[0].x + pts[1].x) / 2, y: (pts[0].y + pts[1].y) / 2 };
      if (gesto.lastDist > 0 && dist > 0) {
        const { mx, my } = canvasPos(mid.x, mid.y);
        view = zoomAround(view, dist / gesto.lastDist, { x: mx, y: my });
      }
      view = { ...view, panX: view.panX + (mid.x - gesto.lastMid.x), panY: view.panY + (mid.y - gesto.lastMid.y) };
      gesto.lastDist = dist;
      gesto.lastMid = mid;
      draw();
      return;
    }

    // Gesto que chegou a ter 2 ponteiros: o dedo que sobra não move a foto
    // até todos soltarem (observação de construção do rf-18).
    if (gesto.multi) return;

    const dist = Math.hypot(e.clientX - gesto.startX, e.clientY - gesto.startY);
    if (dist > gesto.maxDist) gesto.maxDist = dist;
    if (ferramenta === 'marcar') {
      if (lupaDoMarcar) {
        atualizarLupa(e.clientX, e.clientY);
      } else if (gesto.maxDist > LIMIAR_TOQUE_PX && timerLupa !== null) {
        clearTimeout(timerLupa);
        timerLupa = null;
      }
    } else if (ferramenta === 'ajustar' && pino.reg) {
      if (gesto.maxDist > LIMIAR_TOQUE_PX) moverPino(e);
    } else {
      view = { ...view, panX: view.panX + (e.clientX - anterior.x), panY: view.panY + (e.clientY - anterior.y) };
      arrastando = true;
      draw();
    }
  }

  function onPointerUp(e: PointerEvent) {
    const tinha = ponteiros.has(e.pointerId);
    ponteiros.delete(e.pointerId);
    if (ponteiros.size === 0) {
      if (tinha && !gesto.multi) {
        if (ferramenta === 'marcar') {
          if (lupaDoMarcar) criarRegiaoNaLupa(e.clientX, e.clientY);
          else if (gesto.maxDist <= LIMIAR_TOQUE_PX) handlePoint(e.clientX, e.clientY);
        } else if (ferramenta === 'ajustar') {
          finalizarPino(e.clientX, e.clientY);
        }
      }
      fecharLupa();
      pino.reg = null;
      pino.movendo = false;
      pino.retrato = null;
      gesto.multi = false;
      gesto.maxDist = 0;
      arrastando = false;
    } else if (ponteiros.size === 2) {
      reinitGestoPinca();
    }
  }

  /** Também usado em `onlostpointercapture` (achado 8): mesmo tratamento —
   *  tira o id do mapa sem tentar criar/mover nada. */
  function onPointerCancel(e: PointerEvent) {
    ponteiros.delete(e.pointerId);
    if (ponteiros.size === 0) {
      cancelarPino();
      fecharLupa();
      gesto.multi = false;
      gesto.maxDist = 0;
      arrastando = false;
    } else if (ponteiros.size === 2) {
      reinitGestoPinca();
    }
  }

  /** Achado 9 do guardrail: zera ponteiros e o gesto corrente por inteiro —
   *  usado ao desmontar o canvas (aba sem foto), trocar de aba e entrar no
   *  editor, para que um id de ponteiro velho nunca sobreviva e faça o
   *  próximo toque parecer multi (sem criar região). */
  function limparPonteiros(): void {
    ponteiros.clear();
    cancelarPino();
    fecharLupa();
    gesto.multi = false;
    gesto.maxDist = 0;
    arrastando = false;
  }

  // ── rf-22 RN16–RN18: lupa de precisão ──
  /** Ponto da foto sob a tela, preso às bordas da foto. */
  function pontoNaFoto(clientX: number, clientY: number): { ix: number; iy: number } | null {
    if (!image) return null;
    const { mx, my } = canvasPos(clientX, clientY);
    const p = toImage(view, mx, my);
    return {
      ix: Math.min(image.width - 1, Math.max(0, Math.round(p.x))),
      iy: Math.min(image.height - 1, Math.max(0, Math.round(p.y))),
    };
  }

  function atualizarLupa(clientX: number, clientY: number): void {
    const p = pontoNaFoto(clientX, clientY);
    if (!p) return;
    lupa = { ...p, hex: lerCorMedia(p.ix, p.iy).hex, x: clientX, y: clientY };
  }

  function fecharLupa(): void {
    if (timerLupa !== null) {
      clearTimeout(timerLupa);
      timerLupa = null;
    }
    lupaDoMarcar = false;
    if (lupa !== null) lupa = null;
  }

  /** RN16: ponteiro parado (≤ 6 px) por 500 ms no "marcar" abre a lupa. */
  function agendarLupa(): void {
    if (timerLupa !== null) clearTimeout(timerLupa);
    timerLupa = setTimeout(() => {
      timerLupa = null;
      if (ponteiros.size !== 1 || gesto.multi || gesto.maxDist > LIMIAR_TOQUE_PX) return;
      const [ponto] = [...ponteiros.values()];
      lupaDoMarcar = true;
      atualizarLupa(ponto.x, ponto.y);
    }, TOQUE_LONGO_MS);
  }

  /** RN16: soltar sobre a foto cria a região no ponto da lupa; fora dela, nada. */
  function criarRegiaoNaLupa(clientX: number, clientY: number): void {
    if (!image) return;
    const { mx, my } = canvasPos(clientX, clientY);
    const p = toImage(view, mx, my);
    const ix = Math.round(p.x);
    const iy = Math.round(p.y);
    if (ix >= 0 && iy >= 0 && ix < image.width && iy < image.height) void addRegion(ix, iy);
  }

  // ── rf-22 RN10: ferramenta "ajustar pino" ──
  function iniciarPino(e: PointerEvent): void {
    const { mx, my } = canvasPos(e.clientX, e.clientY);
    const hit = hitTest(mx, my);
    pino.reg = hit;
    pino.movendo = false;
    pino.retrato = null;
    if (hit) {
      pino.ox = hit.x;
      pino.oy = hit.y;
    }
  }

  /** Só depois de passar dos 6 px o pino sai do lugar e a lupa abre. O retrato
   *  de antes é tirado na primeira mudança. */
  function moverPino(e: PointerEvent): void {
    const reg = pino.reg;
    const p = pontoNaFoto(e.clientX, e.clientY);
    if (!reg || !p) return;
    if (!pino.movendo) {
      pino.retrato = tirarRetrato('undoLabelMovePin');
      pino.movendo = true;
    }
    reg.x = p.ix;
    reg.y = p.iy;
    atualizarLupa(e.clientX, e.clientY);
    draw();
  }

  /** Segundo dedo, cancelamento ou desmontagem: o pino volta ao lugar e nada
   *  entra na pilha. */
  function cancelarPino(): void {
    const reg = pino.reg;
    if (reg && pino.movendo) {
      reg.x = pino.ox;
      reg.y = pino.oy;
      draw();
    }
    pino.reg = null;
    pino.movendo = false;
    pino.retrato = null;
  }

  function finalizarPino(clientX: number, clientY: number): void {
    const reg = pino.reg;
    if (!reg) return;
    // ≤ 6 px: só seleciona.
    if (!pino.movendo || gesto.maxDist <= LIMIAR_TOQUE_PX) {
      selectRegion(reg.id, true);
      return;
    }
    const p = pontoNaFoto(clientX, clientY) ?? { ix: reg.x, iy: reg.y };
    const retrato = pino.retrato;
    const tabUid = tabs[activeTabIndex].uid;
    reg.x = p.ix;
    reg.y = p.iy;
    const cor = lerCorMedia(p.ix, p.iy);
    reg.r = cor.r; reg.g = cor.g; reg.b = cor.b; reg.hex = cor.hex;
    // Leitura nova: a correção antiga da cor se perde.
    reg.sampleHex = '';
    selectRegion(reg.id, true);
    void recalcularAposMoverPino(reg.id, retrato, tabUid, historicoSeq);
  }

  /** RN2: só empilha quando a conta deu certo; falha devolve o pino e a cor. */
  async function recalcularAposMoverPino(id: number, retrato: Retrato | null, tabUid: number, seq: number): Promise<void> {
    const reg = regiaoDaAba(tabUid, id);
    if (!reg) return;
    const ok = await recalcularAposCorrecao(reg);
    if (ok) {
      if (retrato) empilharSeMesmaEpoca(retrato, tabUid, seq);
      return;
    }
    const orig = retrato?.regions.find(r => r.id === id);
    const agora = regiaoDaAba(tabUid, id);
    if (orig && agora) {
      agora.x = orig.x; agora.y = orig.y;
      agora.r = orig.r; agora.g = orig.g; agora.b = orig.b; agora.hex = orig.hex;
      agora.sampleHex = orig.sampleHex;
      draw();
      marcarAlteracao();
    }
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

  /** rf-22 RN17: a fonte de pixels da leitura de cor e da lupa — a foto da
   *  aba ativa, num canvas guardado enquanto a foto for a mesma. */
  let amostra: { img: HTMLImageElement; canvas: HTMLCanvasElement } | null = null;
  function fonteDeAmostra(): HTMLCanvasElement | null {
    if (!image) return null;
    if (!amostra || amostra.img !== image) {
      const tmp = document.createElement('canvas');
      tmp.width = image.width;
      tmp.height = image.height;
      tmp.getContext('2d', { willReadFrequently: true })!.drawImage(image, 0, 0);
      amostra = { img: image, canvas: tmp };
    }
    return amostra.canvas;
  }

  /** rf-22 RN10/RN17: a média da janela 5×5 (RN9 do rf-18) — usada ao criar
   *  a região, ao ajustar o pino e pela lupa. */
  function lerCorMedia(ix: number, iy: number): { r: number; g: number; b: number; hex: string } {
    const fonte = fonteDeAmostra()!;
    const img = image!;
    const tctx = fonte.getContext('2d', { willReadFrequently: true })!;
    // RN9: média de uma janela 5×5 (3×3 no canto), recortada nas bordas da foto.
    const x0 = Math.max(0, ix - 2);
    const y0 = Math.max(0, iy - 2);
    const x1 = Math.min(img.width - 1, ix + 2);
    const y1 = Math.min(img.height - 1, iy + 2);
    const ww = x1 - x0 + 1;
    const hh = y1 - y0 + 1;
    const janela = tctx.getImageData(x0, y0, ww, hh).data;
    // Achado 3 do guardrail: pixel transparente (alfa 0) não conta na média —
    // e pesa pelo alfa os que só são parcialmente transparentes — senão um
    // ponto opaco a 1px de uma borda transparente do PNG lê cinza em vez da
    // própria cor.
    let somaR = 0, somaG = 0, somaB = 0, somaAlfa = 0;
    const n = ww * hh;
    for (let k = 0; k < n; k++) {
      const a = janela[k * 4 + 3] / 255;
      somaR += janela[k * 4] * a;
      somaG += janela[k * 4 + 1] * a;
      somaB += janela[k * 4 + 2] * a;
      somaAlfa += a;
    }
    let r: number, g: number, b: number;
    if (somaAlfa > 0) {
      r = Math.round(somaR / somaAlfa);
      g = Math.round(somaG / somaAlfa);
      b = Math.round(somaB / somaAlfa);
    } else {
      // Janela inteira transparente: usa o pixel central, como antes da média.
      const centro = tctx.getImageData(ix, iy, 1, 1).data;
      r = centro[0]; g = centro[1]; b = centro[2];
    }
    const hex = `#${[r, g, b].map(n => n.toString(16).padStart(2, '0')).join('')}`;
    return { r, g, b, hex };
  }

  async function addRegion(ix: number, iy: number) {
    if (!image) return;
    if (regions.length >= 50) {
      toast(`${displayTabName(tabs[activeTabIndex], activeTabIndex)}: ${t('errRegionsMax')}`, 'error');
      return;
    }
    const { r, g, b, hex } = lerCorMedia(ix, iy);
    const tabUid = tabs[activeTabIndex].uid;
    // rf-22 U1: o retrato de antes de nascer a região.
    empilhar(tirarRetrato('undoLabelAdd'), tabUid);

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
      manual: false,
      sampleHex: '',
    };
    regions = [...regions, region];
    selectRegion(region.id, true);
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
    // rf-22 RN5: trocar fabricante/estoque do projeto e sair do fallback
    // passam por aqui e zeram a pilha de todas as abas.
    limparHistoricos();
    await Promise.all(
      tabs.flatMap((aba, i) => tabRegions(i).filter(r => !r.override && !r.manual).map(r => computeRegion(r.id, aba.uid))),
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
      // RN23/RN5: o diálogo de fallback é só de quem segue o projeto — nunca
      // por uma região manual, nem por uma conta automática que estava em voo.
      if (resp.faixa === 'nao-encontrei' && !agora.override && !agora.manual && saidaAutorizada === null) {
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

  // ── rf-22 RN1–RN6: desfazer e refazer (uma pilha por aba, só em memória) ──
  const MAX_DESFAZER = 50;

  function copiar<T>(v: T): T {
    return $state.snapshot(v) as T;
  }

  function tirarRetrato(rotulo: DictKey): Retrato {
    return { regions: copiar(regions), selectedId, rotulo };
  }

  /** A pilha da aba `tabUid` — endereçada pela identidade da aba, como
   *  `regiaoDaAba`: uma conta que termina depois da troca de aba empilha na
   *  aba certa. */
  function historicoDaAba(tabUid: number): Historico | null {
    if (tabs[activeTabIndex]?.uid === tabUid) return historico;
    return tabs.find(a => a.uid === tabUid)?.historico ?? null;
  }

  /** RN1: empilha o retrato de ANTES e limpa o refazer; acima de 50, o mais
   *  velho sai. */
  function empilhar(retrato: Retrato, tabUid: number): void {
    const h = historicoDaAba(tabUid);
    if (!h) return;
    h.desfazer.push(retrato);
    if (h.desfazer.length > MAX_DESFAZER) h.desfazer.shift();
    h.refazer = [];
  }

  /** Época das pilhas: sobe em `limparHistoricos` e em `desfazerRefazer`.
   *  As ações que empilham depois de uma conta (U3, U4, U5) guardam a época
   *  de antes do `await`; se ela mudou, o retrato é de outro universo (RN5)
   *  ou o usuário desfez no meio (a pilha de refazer não pode ser zerada). */
  let historicoSeq = 0;

  function empilharSeMesmaEpoca(retrato: Retrato, tabUid: number, seq: number): void {
    if (seq === historicoSeq) empilhar(retrato, tabUid);
  }

  /** RN5: um retrato traria de volta misturas calculadas com outro universo. */
  function limparHistoricos(): void {
    historicoSeq++;
    historico.desfazer = [];
    historico.refazer = [];
    for (const aba of tabs) {
      aba.historico.desfazer = [];
      aba.historico.refazer = [];
    }
  }

  function mesmaEntradaDeConta(a: PlannerRegion, b: PlannerRegion): boolean {
    return a.r === b.r && a.g === b.g && a.b === b.b && a.manual === b.manual && a.override === b.override &&
      a.regionManufacturerId === b.regionManufacturerId && a.regionUseStockOnly === b.regionUseStockOnly;
  }

  /** RN3/RN4: `sentido` 'desfazer' restaura o topo de `desfazer` e guarda o
   *  estado atual em `refazer`; 'refazer' faz o inverso. */
  function desfazerRefazer(sentido: 'desfazer' | 'refazer'): void {
    const de = sentido === 'desfazer' ? historico.desfazer : historico.refazer;
    const para = sentido === 'desfazer' ? historico.refazer : historico.desfazer;
    const alvo = de[de.length - 1];
    if (!alvo) return;
    historicoSeq++;
    const tabUid = tabs[activeTabIndex].uid;
    de.pop();
    para.push(tirarRetrato(alvo.rotulo));
    if (para.length > MAX_DESFAZER) para.shift();

    const vivas = new Map(regions.map(r => [r.id, r]));
    const recalcular: number[] = [];
    const restauradas = copiar(alvo.regions).map((reg) => {
      const viva = vivas.get(reg.id);
      // RN4: época acima da região viva — resposta em voo chega velha e cai.
      reg.calcSeq = Math.max(reg.calcSeq, viva?.calcSeq ?? 0) + 1;
      if (reg.computing) {
        if (viva && !viva.computing && mesmaEntradaDeConta(viva, reg)) {
          const v = copiar(viva);
          reg.result = v.result;
          reg.salvo = v.salvo;
          reg.vazioMotivo = v.vazioMotivo;
          reg.computing = false;
        } else if (!reg.manual) {
          recalcular.push(reg.id);
        } else {
          reg.result = receitaDoSalvo(reg.salvo, reg.r, reg.g, reg.b);
          reg.computing = false;
        }
      }
      return reg;
    });
    regions = restauradas;
    selectedId = restauradas.some(r => r.id === alvo.selectedId) ? alvo.selectedId : null;
    draw();
    if (recalcular.length > 0) {
      void recalcularComMarcacao(async () => {
        await Promise.all(recalcular.map(id => computeRegion(id, tabUid)));
      });
    } else {
      marcarAlteracao();
    }
    toast(t(sentido === 'desfazer' ? 'undoDone' : 'redoDone', { rotulo: t(alvo.rotulo) }));
  }

  function removeSelected() {
    if (selectedId == null) return;
    // rf-22 U2.
    empilhar(tirarRetrato('undoLabelRemove'), tabs[activeTabIndex].uid);
    regions = regions.filter(r => r.id !== selectedId);
    selectedId = null;
    draw();
    marcarAlteracao();
  }

  // ── rf-16 RN19–RN21: ajuste por região ──
  async function ajustarFornecedorDaRegiao(r: PlannerRegion, id: number | null): Promise<void> {
    // rf-22 U6: retrato de antes do ajuste.
    empilhar(tirarRetrato('undoLabelAdjust'), tabs[activeTabIndex].uid);
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
    empilhar(tirarRetrato('undoLabelAdjust'), tabs[activeTabIndex].uid);
    r.regionUseStockOnly = valor;
    await recalcularUmaRegiao(r.id);
  }

  async function recalcularUmaRegiao(id: number): Promise<void> {
    const tabUid = tabs[activeTabIndex].uid;
    // RN3: região manual não recalcula — nem por "Recalcular", nem por ajuste
    // de fornecedor/estoque (o controle já fica desabilitado na tela).
    if (regiaoDaAba(tabUid, id)?.manual) return;
    await recalcularComMarcacao(() => computeRegion(id, tabUid));
  }

  // ── rf-18 RN2/RN15: tinta escolhida à mão ──
  /** `id` é o que vai no corpo do POST de `recipeForChosenPaint` (sempre
   *  <=0, como o servidor espera para uma tinta escolhida à mão); `appId` é
   *  como o resto do app representa a tinta — estoque negativo como está,
   *  catálogo positivo (`plans.ts:52`). Achados 1/6 do guardrail: gravar o
   *  `id` do POST direto no ingrediente colidia com a linha de estoque do
   *  id oposto. */
  type TintaEscolhidaAMao = Pick<StockPaint, 'manufacturerId' | 'manufacturer' | 'name' | 'code' | 'r' | 'g' | 'b'> & {
    id: number;
    appId: number;
  };

  /** Sobe o `calcSeq` da região ANTES de chamar o servidor — a mesma guarda
   *  de `computeRegion` (`:904-909`): uma conta automática em voo chega com
   *  época velha e é descartada, sem trocar `result`/`salvo` nem abrir
   *  fallback (RN15/CAN3). Escolher A e logo B termina com B (CAN2).
   *
   *  Achado 2 do guardrail: marca `manual = true` OTIMISTICAMENTE, antes do
   *  POST — RN3 já exclui a região de recálculos automáticos disparados em
   *  voo (troca de fabricante/estoque, saída do fallback). Falha (throw,
   *  `universoVazio` ou época velha não causada por uma escolha manual mais
   *  nova) restaura o retrato anterior de manual/result/salvo/vazioMotivo.
   *  Devolve `true` só quando a nova mistura foi gravada — quem corrige a
   *  cor da peça usa isso para saber se também precisa desfazer a cor
   *  (achado 4). */
  async function escolherTintaManual(reg: PlannerRegion, tinta: TintaEscolhidaAMao): Promise<boolean> {
    const ep = aberturaSeq;
    const tabUid = tabs[activeTabIndex].uid;
    const seq = ++reg.calcSeq;
    const antes = { manual: reg.manual, result: reg.result, salvo: reg.salvo, vazioMotivo: reg.vazioMotivo };
    reg.manual = true;
    reg.computing = true;
    // Ignora a época de abertura: usado para restaurar o retrato mesmo se o
    // editor foi reaberto em voo, contanto que nenhuma escolha manual mais
    // nova tenha subido `calcSeq` de novo (CAN2 — essa vence, sem restauro).
    const porSeq = (): PlannerRegion | null => {
      const agora = regiaoDaAba(tabUid, reg.id);
      return agora && agora.calcSeq === seq ? agora : null;
    };
    const vigente = (): PlannerRegion | null => (ep === aberturaSeq ? porSeq() : null);
    const desfazer = (): void => {
      const agora = porSeq();
      if (!agora) return;
      agora.manual = antes.manual;
      agora.result = antes.result;
      agora.salvo = antes.salvo;
      agora.vazioMotivo = antes.vazioMotivo;
    };
    try {
      const resp = await recipeForChosenPaint(reg.r, reg.g, reg.b, tinta);
      const agora = vigente();
      if (!agora) return false;
      if (ehUniversoVazio(resp)) {
        desfazer();
        toast(t('errManualCalc'), 'error');
        return false;
      }
      agora.vazioMotivo = null;
      // Achado 1/6 do guardrail: reescreve a marca e o id do único
      // ingrediente ANTES de guardar — `targetManufacturerId: 0` no corpo
      // (RN2) volta `targetManufacturer`/`manufacturerId` vazios do
      // servidor, e o `id` do corpo (sempre <=0) colide com uma linha de
      // estoque de id oposto quando a tinta é do catálogo.
      resp.targetManufacturer = tinta.manufacturer;
      for (const ing of resp.ingredients ?? []) {
        ing.manufacturer = tinta.manufacturer;
        ing.manufacturerId = tinta.manufacturerId;
        ing.paintId = tinta.appId;
      }
      agora.result = resp;
      agora.salvo = salvoDoResultado(resp);
      return true;
    } catch (e) {
      console.error('Erro ao calcular tinta manual:', e);
      desfazer();
      toast(t('errManualCalc'), 'error');
      return false;
    } finally {
      const agora = porSeq();
      if (agora) agora.computing = false;
      draw();
    }
  }

  /** RN1: escolha vinda do Combobox — id negativo é tinta do estoque (como
   *  está); id positivo é do catálogo, e vai negativado no corpo (RN2). */
  async function onEscolherTintaManual(r: PlannerRegion, id: number): Promise<void> {
    let tinta: TintaEscolhidaAMao | null = null;
    if (id < 0) {
      const p = stock.paints.find(sp => sp.id === id);
      if (p) tinta = { id: p.id, appId: p.id, manufacturerId: p.manufacturerId, manufacturer: p.manufacturer, name: p.name, code: p.code, r: p.r, g: p.g, b: p.b };
    } else {
      const p = paintById(id);
      if (p) tinta = { id: -p.id, appId: p.id, manufacturerId: p.manufacturerId, manufacturer: p.manufacturer, name: p.name, code: p.code, r: p.r, g: p.g, b: p.b };
    }
    if (!tinta) return;
    const tintaEscolhida = tinta;
    // rf-22 U5/RN2: retrato de antes; só empilha se a conta deu certo.
    const retrato = tirarRetrato('undoLabelManual');
    const tabUid = tabs[activeTabIndex].uid;
    const seq = historicoSeq;
    let ok = false;
    await recalcularComMarcacao(async () => {
      ok = await escolherTintaManual(r, tintaEscolhida);
    });
    if (ok) empilharSeMesmaEpoca(retrato, tabUid, seq);
  }

  /** RN4: desfaz o manual e recalcula com o universo da região. Achado 7 do
   *  guardrail: limpa `salvo` antes de recalcular — um recálculo vazio ou
   *  com falha (que só zera `result`) não pode deixar a tinta escolhida à
   *  mão gravada como se fosse sugestão da Mescla. */
  async function voltarSugestaoMescla(r: PlannerRegion): Promise<void> {
    empilhar(tirarRetrato('undoLabelManual'), tabs[activeTabIndex].uid);
    r.manual = false;
    r.salvo = null;
    await recalcularUmaRegiao(r.id);
  }

  // ── rf-18 RN6/RN7: correção da cor da peça ──
  function parseHexCor(hex: string): { r: number; g: number; b: number; hex: string } | null {
    const m = /^#?([0-9a-fA-F]{6})$/.exec(hex.trim());
    if (!m) return null;
    const h = m[1].toLowerCase();
    return {
      r: parseInt(h.slice(0, 2), 16),
      g: parseInt(h.slice(2, 4), 16),
      b: parseInt(h.slice(4, 6), 16),
      hex: `#${h}`,
    };
  }

  /** Não manual: recalcula (RN6) — `recalcularUmaRegiao`/`computeRegion` não
   *  devolve sinal de falha (acha 4 do guardrail: sem isso, não dá pra saber
   *  se deve desfazer a cor); mantém a cor nova, como já era. Manual: refaz
   *  só a conta da RN2 com a mesma tinta, remontada do ingrediente salvo
   *  (achado 5 — `manufacturerId` real, não mais 0). Sem ingrediente salvo,
   *  só a cor muda. Devolve `true` quando não há nada a desfazer. */
  async function recalcularAposCorrecao(r: PlannerRegion): Promise<boolean> {
    if (!r.manual) {
      await recalcularUmaRegiao(r.id);
      return true;
    }
    const ing = r.salvo?.ingredients?.[0];
    if (!ing) {
      marcarAlteracao();
      return true;
    }
    // Achado 1/6: `ing.paintId` já é o id como o app representa a tinta
    // (estoque negativo, catálogo positivo); o POST precisa dele <=0.
    const appId = ing.paintId ?? 0;
    const tinta: TintaEscolhidaAMao = {
      id: appId > 0 ? -appId : appId,
      appId,
      manufacturerId: ing.manufacturerId,
      manufacturer: ing.manufacturer,
      name: ing.name,
      code: ing.code,
      r: ing.r,
      g: ing.g,
      b: ing.b,
    };
    let ok = true;
    await recalcularComMarcacao(async () => {
      ok = await escolherTintaManual(r, tinta);
    });
    return ok;
  }

  /** RN6: hex inválido não muda nada; hex válido muda `r/g/b/hex` (minúsculas,
   *  como `addRegion`) e guarda `sampleHex` na primeira correção. Achado 4 do
   *  guardrail: guarda a cor anterior antes de aplicar e, se o recálculo
   *  manual falhar (`escolherTintaManual` já mostra o toast), restaura
   *  `r/g/b/hex/sampleHex` — sem isso a cor nova fica com o ΔE/faixa da
   *  tinta antiga (CA11 offline). */
  async function corrigirCorDaRegiao(r: PlannerRegion, hexEntrada: string): Promise<void> {
    const cor = parseHexCor(hexEntrada);
    if (!cor) {
      corInvalida = true;
      return;
    }
    corInvalida = false;
    const retrato = tirarRetrato('undoLabelColor');
    const tabUid = tabs[activeTabIndex].uid;
    const seq = historicoSeq;
    const antes = { r: r.r, g: r.g, b: r.b, hex: r.hex, sampleHex: r.sampleHex };
    if (!r.sampleHex) r.sampleHex = r.hex;
    r.r = cor.r; r.g = cor.g; r.b = cor.b; r.hex = cor.hex;
    corTexto = cor.hex;
    draw();
    const ok = await recalcularAposCorrecao(r);
    if (!ok) {
      r.r = antes.r; r.g = antes.g; r.b = antes.b; r.hex = antes.hex; r.sampleHex = antes.sampleHex;
      corTexto = antes.hex;
      draw();
    } else {
      empilharSeMesmaEpoca(retrato, tabUid, seq);
    }
  }

  /** RN7: volta à cor lida e limpa `sampleHex`. Mesmo restauro do achado 4
   *  se o recálculo manual falhar. */
  async function usarCorLidaDaFoto(r: PlannerRegion): Promise<void> {
    if (!r.sampleHex) return;
    const cor = parseHexCor(r.sampleHex);
    if (!cor) return;
    const retrato = tirarRetrato('undoLabelColor');
    const tabUid = tabs[activeTabIndex].uid;
    const seq = historicoSeq;
    const antes = { r: r.r, g: r.g, b: r.b, hex: r.hex, sampleHex: r.sampleHex };
    r.r = cor.r; r.g = cor.g; r.b = cor.b; r.hex = cor.hex;
    r.sampleHex = '';
    corTexto = cor.hex;
    corInvalida = false;
    draw();
    const ok = await recalcularAposCorrecao(r);
    if (!ok) {
      r.r = antes.r; r.g = antes.g; r.b = antes.b; r.hex = antes.hex; r.sampleHex = antes.sampleHex;
      corTexto = antes.hex;
      draw();
    } else {
      empilharSeMesmaEpoca(retrato, tabUid, seq);
    }
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
      regionManual: r.manual, sampleHex: r.sampleHex,
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
      regionManual: r.manual ? 1 : 0, sampleHex: r.sampleHex,
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
    ferramenta = 'marcar';
    limparPonteiros();
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

  /** Uma região do rascunho (ou do arquivo, rf-22) remontada como DTO. */
  function regiaoRascunhoParaDTO(rr: RegiaoRascunho): RegiaoDTO {
    return {
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
      regionManual: rr.regionManual ? 1 : 0, sampleHex: rr.sampleHex ?? '',
    };
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
        regions: rt.regions.map(regiaoRascunhoParaDTO),
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
    // RN3: restauração nunca recalcula região manual, mesmo sem mistura salva.
    if (inicio.manual) return;
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
          manual: rd.regionManual === 1,
          sampleHex: rd.sampleHex ?? '',
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
        historico: { desfazer: [], refazer: [] },
      });
    }

    // RN27/RN28: só região sem mistura salva recalcula; rascunho nunca
    // consulta o servidor (CA8 do rf-07).
    if (recalcular) {
      await Promise.all(
        novasAbas.flatMap(aba => aba.regions.filter(r => !r.manual && !r.salvo?.faixa).map(r => computeRegionInAba(aba, r.id, ctx, ep))),
      );
    }
    if (ep !== aberturaSeq) return;

    planId = plano.id ?? null;
    planName = plano.name;
    manufacturerId = ctx.manufacturerId;
    useStockOnly = ctx.useStockOnly;
    saidaAutorizada = null;
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

  // ── rf-22 RN19: exportar o plano em arquivo JSON ──
  function exportarArquivoJson(): void {
    exportMenuOpen = false;
    if (algumCalculando) {
      toast(t('saveWaitCalc'), 'error');
      return;
    }
    const dto = buildPlanoDTO();
    if (!dto.tabs.some(a => a.imageData)) return;
    const conteudo = montarArquivoPlano(dto, nomeFabricante);
    // `baixarTexto` adia o revoke: revogar logo após o clique cancela o
    // download no Safari do iPad (guardrail F5).
    baixarTexto(nomeArquivoPlano(planName), conteudo, 'application/json');
  }

  // ── rf-22 RN21–RN23: abrir plano de arquivo JSON ──
  /** RN23: o id vale se o fabricante com esse id tem o mesmo nome; senão,
   *  procura pelo nome; senão `null` (o chamador decide o que fazer). */
  function resolverFabricante(id: number | null, nome: string): number | null {
    const alvo = nome.trim().toLowerCase();
    if (id != null) {
      const porId = manufacturers.find(m => m.id === id);
      if (porId && (alvo === '' || porId.name.trim().toLowerCase() === alvo)) return porId.id;
    }
    if (alvo === '') return null;
    return manufacturers.find(m => m.name.trim().toLowerCase() === alvo)?.id ?? null;
  }

  function planoImportadoParaDTO(plano: PlanoImportado, nome: string): PlanoDTO {
    return {
      id: undefined,
      name: plano.name.trim() ? plano.name : nome,
      selectedManufacturerId: resolverFabricante(plano.selectedManufacturerId, plano.selectedManufacturerName),
      useStockOnly: plano.useStockOnly ? 1 : 0,
      tabs: plano.tabs.map((aba): AbaDTO => ({
        id: undefined,
        name: aba.name,
        imageData: aba.imageData,
        regions: aba.regions.map((rr): RegiaoDTO => {
          // Ajuste de fornecedor que não casa nesta instalação: a região perde
          // o ajuste e mantém a mistura salva.
          const marca = rr.regionOverride ? resolverFabricante(rr.regionManufacturerId ?? null, rr.regionManufacturerName) : null;
          const ajusta = rr.regionOverride === true && marca !== null;
          return {
            ...regiaoRascunhoParaDTO(rr),
            regionOverride: ajusta ? 1 : 0,
            regionManufacturerId: ajusta ? marca : null,
            regionUseStockOnly: ajusta && rr.regionUseStockOnly ? 1 : 0,
          };
        }),
      })),
    };
  }

  async function abrirArquivo(arquivo: File): Promise<void> {
    if (abrindo) return;
    if (arquivo.size > TETO_ARQUIVO_PLANO) {
      toast(t('fileTooBig'), 'error');
      return;
    }
    let conteudo: string;
    try {
      conteudo = await arquivo.text();
    } catch {
      toast(t('fileNotRecognized'), 'error');
      return;
    }
    const lido = lerArquivoPlano(conteudo);
    if (!lido.ok) {
      const msg = t(lido.chave as DictKey, { mb: tetoMb });
      toast(lido.abaNome ? `${lido.abaNome}: ${msg}` : msg, 'error');
      return;
    }
    const nomeDoArquivo = arquivo.name.replace(/\.(json|mesclaplan)$/i, '').slice(0, 200);
    const dto = planoImportadoParaDTO(lido.plano, nomeDoArquivo);

    // Tudo ou nada: a foto de cada aba tem de carregar, e x/y ficam presos às
    // dimensões dela. Nada disso mexe no editor antes da confirmação.
    for (let i = 0; i < dto.tabs.length; i++) {
      const aba = dto.tabs[i];
      if (!aba.imageData) continue;
      const img = await loadImageBitmap(aba.imageData);
      if (!img) {
        toast(`${aba.name.trim() || `Figura ${i + 1}`}: ${t('errImageType')}`, 'error');
        return;
      }
      for (const rg of aba.regions) {
        rg.x = Math.min(img.width - 1, Math.max(0, rg.x));
        rg.y = Math.min(img.height - 1, Math.max(0, rg.y));
      }
    }

    if (!confirmarDescarteDoRascunho(arquivo.name)) return;
    const ep = ++aberturaSeq;
    abrindo = true;
    try {
      await applyLoadedPlan(dto, true, 0, ep);
      if (ep !== aberturaSeq) return;
      entrarNoEditor(false);
    } catch (e) {
      if (ep !== aberturaSeq) return;
      console.error('Erro ao abrir arquivo de plano:', e);
      toast(t('fileNotRecognized'), 'error');
      return;
    } finally {
      if (ep === aberturaSeq) abrindo = false;
    }
    // Projeto novo, ainda não salvo: o rascunho local já nasce gravado.
    marcarAlteracao();
    toast(t('fileOpened'));
  }

  // ── rf-22 RN7/RN8/RN14/RN15: atalhos, colar e arrastar ──
  function emCampoDeTexto(alvo: EventTarget | null): boolean {
    return alvo instanceof Element &&
      alvo.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="listbox"], [aria-haspopup="listbox"]') !== null;
  }

  /** RN8: onde nenhum atalho vale. O `[aria-modal]` de fora cobre diálogos e
   *  paletas de outras telas (Ctrl+K); o de dentro é o do fallback. Só conta
   *  modal visível: as telas ficam montadas com `display:none`, e T4 mantém
   *  diálogos com `aria-modal` fixo — sem o filtro, atalho nenhum disparava
   *  (guardrail F5). */
  function modalVisivelAberto(): boolean {
    return [...document.querySelectorAll('[aria-modal="true"]')].some(el => el.getClientRects().length > 0);
  }

  function atalhosBloqueados(alvo: EventTarget | null): boolean {
    return modo !== 'editor' || nav.tab !== 'plano' || abrindo || regiaoNoDialogo !== null || exportMenuOpen ||
      emCampoDeTexto(alvo) || modalVisivelAberto();
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.isComposing) return;
    const k = e.key.toLowerCase();
    const mod = e.ctrlKey || e.metaKey;
    // Ctrl/⌘+K é da paleta de comandos global.
    if (mod && k === 'k') return;
    if (atalhosBloqueados(e.target)) return;
    if (mod) {
      if (e.altKey) return;
      if (k === 'z') {
        e.preventDefault();
        desfazerRefazer(e.shiftKey ? 'refazer' : 'desfazer');
      } else if (k === 'y' && !e.shiftKey) {
        e.preventDefault();
        desfazerRefazer('refazer');
      }
      return;
    }
    // Letra com Alt (ou Shift) não troca ferramenta.
    if (e.altKey || e.shiftKey || !hasImage) return;
    if (k === 'v') ferramenta = 'mover';
    else if (k === 'i') ferramenta = 'marcar';
    else if (k === 'e') ferramenta = 'ajustar';
  }

  function onPaste(e: ClipboardEvent): void {
    if (atalhosBloqueados(e.target)) return;
    const item = [...(e.clipboardData?.items ?? [])].find(i => i.kind === 'file' && i.type.startsWith('image/'));
    const arquivo = item?.getAsFile();
    if (!arquivo) return;
    e.preventDefault();
    receberImagem(arquivo, 'colar');
  }

  function temArquivo(e: DragEvent): boolean {
    return [...(e.dataTransfer?.types ?? [])].includes('Files');
  }

  /** RN15: em todo o editor, arquivo solto fora da caixa não tira o navegador
   *  do app abrindo o arquivo. */
  function onDragOverJanela(e: DragEvent): void {
    if (modo === 'editor' && temArquivo(e)) e.preventDefault();
  }

  function onDropJanela(e: DragEvent): void {
    if (modo === 'editor' && temArquivo(e)) e.preventDefault();
  }

  function onDragOverCaixa(e: DragEvent): void {
    if (!temArquivo(e)) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
    soltando = true;
  }

  function onDragLeaveCaixa(e: DragEvent): void {
    if (boxEl && e.relatedTarget instanceof Node && boxEl.contains(e.relatedTarget)) return;
    soltando = false;
  }

  function onDropCaixa(e: DragEvent): void {
    if (!temArquivo(e)) return;
    e.preventDefault();
    soltando = false;
    const arquivos = e.dataTransfer?.files;
    const primeiro = arquivos?.[0];
    if (!primeiro) return;
    if (arquivos.length > 1) toast(t('dropOnlyFirst'));
    receberImagem(primeiro, 'arrastar');
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

<svelte:window
  onkeydown={onKeydown}
  onpaste={onPaste}
  ondragover={onDragOverJanela}
  ondrop={onDropJanela}
/>

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
    onabrirArquivo={(arq) => void abrirArquivo(arq)}
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
      <!-- rf-22 RN15: arquivo de imagem solto na caixa da foto. -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        bind:this={boxEl}
        ondragover={onDragOverCaixa}
        ondragleave={onDragLeaveCaixa}
        ondrop={onDropCaixa}
        style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: repeating-conic-gradient(var(--color-panel) 0% 25%, var(--color-bg) 0% 50%) 0 0 / 40px 40px; cursor: {cursorDaFoto};"
      >
        {#if hasImage}
          <canvas
            bind:this={canvasEl}
            style="width: 100%; height: 100%; display: block; touch-action: none; cursor: {cursorDaFoto};"
            oncontextmenu={(e) => e.preventDefault()}
            onpointerdown={onPointerDown}
            onpointermove={onPointerMove}
            onpointerup={onPointerUp}
            onpointercancel={onPointerCancel}
            onlostpointercapture={onPointerCancel}
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
        {#if soltando}
          <div
            aria-hidden="true"
            style="position: absolute; inset: 12px; display: flex; align-items: center; justify-content: center; border: 2px dashed var(--color-accent); border-radius: 14px; background: rgba(22,24,38,0.82); color: var(--color-accent-400); font-size: 17px; font-weight: 500; text-align: center; padding: 20px; pointer-events: none;"
          >{t('dropHere', { name: displayTabName(tabs[activeTabIndex], activeTabIndex) })}</div>
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
          <!-- rf-22 RN6: desfazer e refazer da aba ativa. -->
          <button
            class="t2-icon-btn"
            aria-label={t('undoBtn')}
            title={`${t('undoBtn')} (Ctrl+Z)`}
            disabled={historico.desfazer.length === 0}
            onclick={() => desfazerRefazer('desfazer')}
          ><i class="ph ph-arrow-counter-clockwise" style="font-size: 18px;"></i></button>
          <button
            class="t2-icon-btn"
            aria-label={t('redoBtn')}
            title={`${t('redoBtn')} (Ctrl+Shift+Z)`}
            disabled={historico.refazer.length === 0}
            onclick={() => desfazerRefazer('refazer')}
          ><i class="ph ph-arrow-clockwise" style="font-size: 18px;"></i></button>

          <div aria-hidden="true" style="width: 1px; align-self: stretch; margin: 4px 2px; background: var(--color-neutral-800);"></div>

          <button
            class="t2-icon-btn"
            aria-label={t('toolMark')}
            title={`${t('toolMark')} (I)`}
            aria-pressed={ferramenta === 'marcar'}
            disabled={!hasImage}
            onclick={() => (ferramenta = 'marcar')}
            style="background: {ferramenta === 'marcar' ? 'var(--color-accent-panel)' : 'transparent'}; color: {ferramenta === 'marcar' ? 'var(--color-accent-400)' : 'var(--color-neutral-300)'};"
          >
            <i class="ph ph-crosshair" style="font-size: 18px;"></i>
          </button>
          <button
            class="t2-icon-btn"
            aria-label={t('toolMove')}
            title={`${t('toolMove')} (V)`}
            aria-pressed={ferramenta === 'mover'}
            disabled={!hasImage}
            onclick={() => (ferramenta = 'mover')}
            style="background: {ferramenta === 'mover' ? 'var(--color-accent-panel)' : 'transparent'}; color: {ferramenta === 'mover' ? 'var(--color-accent-400)' : 'var(--color-neutral-300)'};"
          >
            <i class="ph ph-hand" style="font-size: 18px;"></i>
          </button>
          <button
            class="t2-icon-btn"
            aria-label={t('toolAdjust')}
            title={`${t('toolAdjust')} (E)`}
            aria-pressed={ferramenta === 'ajustar'}
            disabled={!hasImage}
            onclick={() => (ferramenta = 'ajustar')}
            style="background: {ferramenta === 'ajustar' ? 'var(--color-accent-panel)' : 'transparent'}; color: {ferramenta === 'ajustar' ? 'var(--color-accent-400)' : 'var(--color-neutral-300)'};"
          >
            <i class="ph ph-arrows-out-cardinal" style="font-size: 18px;"></i>
          </button>

          <div aria-hidden="true" style="width: 1px; align-self: stretch; margin: 4px 2px; background: var(--color-neutral-800);"></div>

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
              aria-label={t('exportReportBtn')}
              title={t('exportReportBtn')}
              aria-haspopup="menu"
              aria-expanded={exportMenuOpen}
              disabled={reportGenerating}
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
                <button role="menuitem" class="t2-menu-item" disabled={relatorioDisabled} title={relatorioDisabled ? t('exportSaveFirst') : undefined} onclick={() => void handleExport('pdf')}>{t('exportPdf')}</button>
                <button role="menuitem" class="t2-menu-item" disabled={relatorioDisabled} title={relatorioDisabled ? t('exportSaveFirst') : undefined} onclick={() => void handleExport('png')}>{t('exportPng')}</button>
                <button role="menuitem" class="t2-menu-item" disabled={!algumaAbaComFoto} onclick={exportarArquivoJson}>{t('exportJson')}</button>
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
                  {#if r.manual}
                    <span style="align-self: flex-start; font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 999px; background: var(--color-accent-panel); color: var(--color-accent-400);">{t('regionManualBadge')}</span>
                  {/if}
                  {#if seloDaFaixa(r)}
                    {@const selo = seloDaFaixa(r)}
                    <span style="font-size: 11.5px; font-weight: 600; color: {selo?.cor};">{selo?.texto}</span>
                  {/if}
                  {#if r.result?.foraDoUniverso}
                    <span style="font-size: 11.5px; color: var(--color-warning, #E4A11B);">{t('foraDoUniversoLabel')}</span>
                  {/if}
                </span>
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
                              {#if temEmEstoque(ing)}
                                <span style="align-self: flex-start; margin-top: 2px; font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 999px; background: var(--color-accent-panel); color: var(--color-accent-400);">{t('ingredientInStock')}</span>
                              {:else}
                                <span style="align-self: flex-start; margin-top: 2px; font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 999px; border: 1px solid var(--color-neutral-800); color: var(--color-neutral-500);">{t('ingredientNoStock')}</span>
                              {/if}
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

                  <!-- rf-18 RN6/RN7: correção da cor da peça. -->
                  <div style="display: flex; flex-direction: column; gap: 8px; padding-top: 12px; border-top: 1px solid var(--color-line);">
                    <p class="section-label" style="margin: 0;">{t('regionColorTitle')}</p>
                    <div style="display: flex; align-items: center; gap: 10px;">
                      <input
                        type="color"
                        value={r.hex}
                        aria-label={t('regionColorTitle')}
                        onchange={(e) => void corrigirCorDaRegiao(r, e.currentTarget.value)}
                        style="width: 44px; height: 44px; padding: 0; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; cursor: pointer; flex-shrink: 0;"
                      />
                      <input
                        type="text"
                        bind:value={corTexto}
                        aria-label={t('regionColorTitle')}
                        maxlength="7"
                        class="font-mono"
                        style="flex: 1; min-width: 0; height: 44px; padding: 0 10px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-size: 14px;"
                      />
                      <button class="t2-secondary-btn" onclick={() => void corrigirCorDaRegiao(r, corTexto)}>{t('regionColorApply')}</button>
                    </div>
                    {#if corInvalida}
                      <p role="alert" style="margin: 0; font-size: 12.5px; color: var(--color-danger, #D1495B);">{t('regionColorInvalid')}</p>
                    {/if}
                    {#if r.sampleHex}
                      <button class="t2-secondary-btn" style="align-self: flex-start;" onclick={() => void usarCorLidaDaFoto(r)}>
                        <i class="ph ph-arrow-counter-clockwise" style="font-size: 16px;"></i>{t('regionColorRestore')}
                      </button>
                    {/if}
                  </div>

                  <!-- rf-18 RN1/RN4: tinta escolhida à mão. -->
                  <div style="display: flex; flex-direction: column; gap: 8px; padding-top: 12px; border-top: 1px solid var(--color-line);">
                    <p class="section-label" style="margin: 0;">{t('regionManualTitle')}</p>
                    <Combobox
                      options={opcoesTintaManual}
                      value={null}
                      onchange={(id) => { if (id != null) void onEscolherTintaManual(r, id); }}
                      placeholder={t('regionManualTitle')}
                      searchPlaceholder={t('phFindPaint')}
                      emptyText={t('noPaintFound')}
                      ariaLabel={t('regionManualTitle')}
                      moreText={(n) => t('comboMore', { n })}
                    />
                    {#if r.manual}
                      <button class="t2-secondary-btn" style="align-self: flex-start;" onclick={() => void voltarSugestaoMescla(r)}>
                        <i class="ph ph-arrow-counter-clockwise" style="font-size: 16px;"></i>{t('regionBackToSuggestion')}
                      </button>
                    {/if}
                  </div>

                  <!-- rf-16 RN19: ajuste de fornecedor só desta região. -->
                  <div style="display: flex; flex-direction: column; gap: 8px; padding-top: 12px; border-top: 1px solid var(--color-line);">
                    <p class="section-label" style="margin: 0;">{t('regionAdjustTitle')}</p>
                    {#if r.manual}
                      <p style="margin: 0; font-size: 12px; color: var(--color-neutral-500);">{t('regionManualNote')}</p>
                    {/if}
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
                      disabled={r.manual}
                    />
                    <label style="display: flex; align-items: center; gap: 10px; min-height: 44px; font-size: 13.5px; color: {r.override && !r.manual ? 'var(--color-neutral-300)' : 'var(--color-neutral-600)'}; cursor: {r.override && !r.manual ? 'pointer' : 'default'};">
                      <input
                        type="checkbox"
                        role="switch"
                        disabled={!r.override || r.manual}
                        checked={r.override ? r.regionUseStockOnly : useStockOnly}
                        onchange={(e) => void ajustarEstoqueDaRegiao(r, e.currentTarget.checked)}
                        style="width: 20px; height: 20px; accent-color: var(--color-accent);"
                      />
                      {t('onlyMyStock')}
                    </label>
                    <div style="display: flex; gap: 8px; flex-wrap: wrap;">
                      <button class="t2-secondary-btn" disabled={r.computing || r.manual} onclick={() => void recalcularUmaRegiao(r.id)}>
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

      </div>
    </div>
  </div>
</div>
{/if}

{#if lupa}
  <LupaPrecisao fonte={fonteDeAmostra()} ix={lupa.ix} iy={lupa.iy} hex={lupa.hex} clientX={lupa.x} clientY={lupa.y} />
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

  .t2-menu-item:disabled {
    opacity: 0.45;
    pointer-events: none;
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
