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
  import {
    suggestRecipeForColor, melhorDeltaE, ehUniversoVazio,
    type EquivalentRecipe, type UniversoBusca,
  } from '../services/engine';
  import { allManufacturers } from '../services/catalog';
  import { saveRecipe } from '../services/recipes.svelte';
  import { verdictKeys } from '../ui';
  import { t, decimal, type DictKey } from '../i18n.svelte';
  import { toast } from '../toast.svelte';
  import { nav } from '../nav.svelte';
  import {
    lerRascunho, agendarGravacao, apagarRascunho,
    lerPlanoAtivo, gravarPlanoAtivo, limparPlanoAtivo, estadoAutoSave,
    type Rascunho, type RegiaoRascunho, type AbaRascunho, type EstadoAutoSave,
  } from '../planner/rascunho';
  import {
    validarPlano, salvarPlano, carregarPlano, baixarRelatorio, PlanoError,
    type PlanoDTO, type RegiaoDTO, type AbaDTO, type ErroValidacaoPlano,
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
    foraDoUniverso: boolean;
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
   *  `imageDataUrl`, `nextId`, `selectedId`, `view` e `manufacturerId` só
   *  pertencem à aba ATIVA nas variáveis de topo (abaixo); o resto do tempo
   *  moram aqui. `uid` é a chave estável do `{#each}` — nunca o índice, que
   *  muda ao reordenar/excluir. */
  interface AbaState {
    uid: number;
    serverId?: number;
    name: string;
    useStockOnly: boolean;
    imageDataUrl: string;
    hasImage: boolean;
    image: HTMLImageElement | null;
    regions: PlannerRegion[];
    nextId: number;
    selectedId: number | null;
    view: View;
    manufacturerId: number | null;
    /** rf-11 RN4: resposta do diálogo de fallback, lembrada por aba.
     *  null = ainda não perguntou nesta aba. */
    saidaAutorizada: 'marca' | 'todos' | null;
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
  let useStockOnly = $state(false);
  let saidaAutorizada: 'marca' | 'todos' | null = $state(null);
  /** rf-11: região que estourou o limiar e está esperando a resposta do
   *  diálogo. null = diálogo fechado. */
  let regiaoNoDialogo: number | null = $state(null);
  let dialogoMelhorMarca: number | null = $state(null);
  let dialogoMelhorTodos: number | null = $state(null);
  let universoVazioMotivo: string | null = $state(null);
  let dialogoElemento: HTMLDivElement | undefined = $state();
  let dialogoDisparador: HTMLElement | null = null;

  // Checklist local de "tenho"/"comprar" das tintas do PLANO INTEIRO — não há
  // conceito de posse por tinta no app hoje (só por marca, na estante); este
  // estado vive só nesta tela. Chave = marca+código+nome (RN7), sobrevive à
  // troca de aba e de foto (RN8/CA14) — só zera ao (re)carregar outro plano.
  let ownedPaints: Set<string> = $state(new Set());

  // ── rf-09: abas de figura ──
  // `tabs[activeTabIndex]` é a aba "estacionada" — as variáveis acima (regions,
  // image, hasImage, imageDataUrl, nextId, selectedId, view, manufacturerId)
  // são o espelho de trabalho da aba ATIVA; troca de aba grava o espelho na
  // aba de origem e relê da aba de destino (RN4: nunca marca alteração).
  let abaUidSeq = 1;
  function emptyAba(): AbaState {
    return {
      uid: abaUidSeq++,
      serverId: undefined,
      name: '',
      useStockOnly: false,
      imageDataUrl: '',
      hasImage: false,
      image: null,
      regions: [],
      nextId: 1,
      selectedId: null,
      view: { zoom: 1, panX: 0, panY: 0 },
      manufacturerId: null,
      saidaAutorizada: null,
    };
  }
  let tabs: AbaState[] = $state([emptyAba()]);
  let activeTabIndex = $state(0);
  let renamingTabIndex: number | null = $state(null);
  const MAX_ABAS_UI = 10;

  /** Nome efetivo pra exibir — mesma convenção de `plans.ts`
   *  (`nomeEfetivoDaAba`), de propósito não traduzida (RN3: "Figura N" é
   *  convenção fixa, igual na mensagem de erro do servidor). */
  function displayTabName(aba: AbaState, i: number): string {
    const nome = aba.name.trim();
    return nome.length > 0 ? nome : `Figura ${i + 1}`;
  }

  /** Regiões "ao vivo" de uma aba: a ativa lê do espelho de trabalho — que
   *  pode estar mais fresco que `tabs[i].regions` entre uma mutação e o
   *  próximo `snapshotActiveIntoTabs()`. */
  function tabRegions(i: number): PlannerRegion[] {
    return i === activeTabIndex ? regions : tabs[i].regions;
  }

  /** `hasImage` "ao vivo" de uma aba — mesma convenção de `tabRegions`. Usada
   *  pelo cabeçalho (achado 3 do guardrail rf-09): o progresso é do plano
   *  inteiro, não pode sumir só porque a aba ATIVA está sem foto. */
  function tabHasImage(i: number): boolean {
    return i === activeTabIndex ? hasImage : tabs[i].hasImage;
  }

  function tabProgressLabel(i: number): string {
    const rs = tabRegions(i);
    return t('tabProgressShort', { a: rs.filter(r => r.painted).length, b: rs.length });
  }

  /** Grava o espelho de trabalho de volta na aba ativa — chamado antes de
   *  trocar de aba, montar o DTO/rascunho, ou qualquer leitura que precise do
   *  estado mais recente de todas as abas. */
  function snapshotActiveIntoTabs(): void {
    const atual = tabs[activeTabIndex];
    if (!atual) return;
    tabs[activeTabIndex] = {
      ...atual,
      imageDataUrl, hasImage, image, regions, nextId, selectedId, view, manufacturerId,
      useStockOnly, saidaAutorizada,
    };
  }

  /** Relê o espelho de trabalho a partir da aba `i` — nunca marca alteração
   *  (RN4/CA21: trocar de aba não é alteração de conteúdo). */
  function loadTabIntoWorkingState(i: number): void {
    const aba = tabs[i];
    imageDataUrl = aba.imageDataUrl;
    hasImage = aba.hasImage;
    image = aba.image;
    regions = aba.regions;
    nextId = aba.nextId;
    selectedId = aba.selectedId;
    view = aba.view;
    manufacturerId = aba.manufacturerId;
    useStockOnly = aba.useStockOnly;
    saidaAutorizada = aba.saidaAutorizada;
    // CAN5: o diálogo é da região da aba anterior — fecha ao trocar.
    regiaoNoDialogo = null;
    universoVazioMotivo = null;
  }

  function switchTab(i: number): void {
    if (i === activeTabIndex || i < 0 || i >= tabs.length) return;
    snapshotActiveIntoTabs();
    activeTabIndex = i;
    loadTabIntoWorkingState(i);
  }

  // RN1/CA8: 11ª aba é negada na hora, com a mesma mensagem do servidor.
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

  // RN6/CA6/CA7: excluir a única aba é negado; as demais pedem confirmação.
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

  /** CA29: seta esquerda/direita move o foco (e a seleção) entre as abas. */
  function focusTabButton(i: number): void {
    const el = document.getElementById(`t2-tab-btn-${i}`);
    el?.focus();
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
  /** data URL da foto — guardada à parte de `image` (HTMLImageElement) porque
   *  o rascunho e o PlanoDTO precisam da string crua, não do bitmap. */
  let imageDataUrl = $state('');

  type SaveState = 'idle' | 'saving' | 'saved' | 'error';
  let saveState: SaveState = $state('idle');
  let savedAtLabel = $state('');
  /** Chave i18n do erro de Salvar — nunca texto cru do servidor (RN11). */
  let saveErrorKey: DictKey | null = $state(null);
  /** Erro de validação do formulário (CA11/CA13/CA17-CA20) — `abaNome`
   *  (RN15) compõe a mensagem final quando o erro é de uma aba específica. */
  let validationError: ErroValidacaoPlano | null = $state(null);
  let validationErrorText = $derived.by((): string | null => {
    const erro = validationError;
    if (!erro) return null;
    const msg = t(erro.chave as DictKey);
    return erro.abaNome ? `${erro.abaNome}: ${msg}` : msg;
  });

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

  // CA19: foco vai para o diálogo assim que ele aparece.
  $effect(() => {
    if (regiaoNoDialogo !== null) dialogoElemento?.focus();
  });

  let selectedRegion = $derived(regions.find(r => r.id === selectedId) ?? null);
  // CA13/RN9: o progresso do cabeçalho soma as regiões pintadas de TODAS as
  // abas — cada aba mostra o seu próprio na tira (tabProgressLabel).
  let totalRegionsCount = $derived(tabs.reduce((acc, _a, i) => acc + tabRegions(i).length, 0));
  let paintedCount = $derived(
    tabs.reduce((acc, _a, i) => acc + tabRegions(i).filter(r => r.painted).length, 0)
  );
  let progressPct = $derived(totalRegionsCount ? Math.round((paintedCount / totalRegionsCount) * 100) : 0);

  // CA13/achado 3: o cabeçalho mostra o progresso do PLANO INTEIRO sempre
  // que alguma aba tem foto — mesmo que a aba ativa não tenha.
  let planHasImage = $derived(tabs.some((_a, i) => tabHasImage(i)));
  let plannerTitle = $derived(
    planHasImage ? t('progress', { a: paintedCount, b: totalRegionsCount }) : t('t2EmptyTitle')
  );

  // rf-08/RN11: plano nunca salvo ou com rascunho local pendente bloqueia a exportação.
  let rascunhoPendente = $derived(draftBannerVisible || alteracaoSeq !== alteracaoSeqSalva);
  let exportDisabled = $derived(planId == null || rascunhoPendente || reportGenerating);

  function paintKey(brand: string, code: string, name: string): string {
    return `${brand} ${code} ${name}`;
  }

  // RN7/CA10-CA12: dedup por marca+código+nome (o mesmo descritor gravado por
  // região — mapRegionCommon), atravessando TODAS as abas do plano; região
  // sem tinta (descritor vazio) não entra; cada item cita as abas onde aparece.
  let shoppingItems = $derived.by((): ShoppingItem[] => {
    const map = new Map<string, ShoppingItem>();
    tabs.forEach((aba, i) => {
      const abaNome = displayTabName(aba, i);
      tabRegions(i).forEach((r, idx) => {
        const desc = mapRegionCommon(r, idx);
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
    if (!r.result) return t('noSimilarPaint');
    return `ΔE ${decimal(r.result.deltaE, 1)} · ${r.result.ingredients.map(ing => ing.name).join(' + ')}`;
  }

  /** rf-11 RN3: selo da faixa de qualidade. Nunca só por cor — o texto e o
   *  ΔE00 andam juntos. */
  function seloDaFaixa(r: PlannerRegion): { texto: string; cor: string } | null {
    const faixa = r.result?.faixa;
    if (!faixa) return null;
    if (faixa === 'otimo') return { texto: t('faixaOtima'), cor: 'var(--color-success, #3FA34D)' };
    if (faixa === 'aproximada') return { texto: t('faixaAproximada'), cor: 'var(--color-warning, #E4A11B)' };
    return { texto: t('faixaNaoEncontrei'), cor: 'var(--color-danger, #D1495B)' };
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

  function toggleOwned(key: string) {
    const next = new Set(ownedPaints);
    if (next.has(key)) next.delete(key);
    else next.add(key);
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

  // RN8/CA14: o checklist "tenho" é do PLANO — trocar a foto de uma aba não
  // zera mais `ownedPaints` (a marcação sobrevive, como manda a spec).
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

  /** Carrega o bitmap de uma foto sem tocar no estado de trabalho — usado na
   *  hidratação, para as abas que ainda não são a ativa. */
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
    // RN2/CA9: teto de 50 regiões É POR ABA — a mensagem cita a aba (RN15).
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
    };
    regions = [...regions, region];
    selectedId = region.id;
    draw();
    await computeRegion(region.id);
    marcarAlteracao();
  }

  /** Acha a região `id` pela identidade da aba (`tabUid`) — nunca pelo
   *  binding vivo `regions`, que passa a apontar para outra aba assim que o
   *  usuário troca de aba. Os ids de região reiniciam em 1 por aba: ler
   *  direto de `regions` gravaria o resultado na região de mesmo id da aba
   *  que virou ativa (achado 1 do guardrail rf-09). Enquanto a aba de
   *  origem continuar ativa, `regions` É o espelho dela; se deixou de ser,
   *  o espelho já foi estacionado em `tabs` por `snapshotActiveIntoTabs`. */
  function regiaoDaAba(tabUid: number, id: number): PlannerRegion | null {
    const arr = tabs[activeTabIndex]?.uid === tabUid ? regions : (tabs.find(a => a.uid === tabUid)?.regions ?? null);
    return arr?.find(r => r.id === id) ?? null;
  }

  /** D-002b: o cálculo é endereçado por ID, nunca por referência.
   *  `regions` é `$state`, então o que está no array é o PROXY da região —
   *  escrever no objeto cru que `addRegion` criou não dispara reatividade
   *  nenhuma, e a tela fica presa em "calculando" com a resposta já em mãos.
   *  `tabUidChamada` (mesmo padrão de época do `onManufacturerChange`, com
   *  `salvamentoSeq`) fixa a aba de origem no instante da chamada — troca de
   *  aba durante o cálculo nunca escreve na aba errada nem deixa a região de
   *  origem presa em "calculando" (achado 1 do guardrail rf-09). */
  /** rf-11 RN1/RN10: o universo vai para o servidor como dois parâmetros; a
   *  interseção é montada lá, nunca aqui. */
  function universoAtual(): UniversoBusca {
    return {
      targetManufacturerId: saidaAutorizada === 'todos' ? undefined : manufacturerId ?? undefined,
      useStockOnly: saidaAutorizada === null && useStockOnly,
      foraDoUniverso: saidaAutorizada !== null,
    };
  }

  /** rf-11 RN4/RN5/RN6: o diálogo de fallback. Abre UMA vez por aba, na
   *  primeira região que estoura o limiar, e mostra o melhor ΔE00 alcançável
   *  em cada saída antes de o usuário escolher. */
  async function abrirDialogoDeFallback(regiaoId: number, r: number, g: number, b: number): Promise<void> {
    if (regiaoNoDialogo !== null) return; // já tem um aberto
    dialogoDisparador = document.activeElement as HTMLElement | null;
    regiaoNoDialogo = regiaoId;
    dialogoMelhorMarca = null;
    dialogoMelhorTodos = null;

    // As duas saídas, na ordem da RN5: primeiro todo o catálogo do fabricante
    // base (comprar uma tinta que falta), depois os outros fabricantes.
    if (manufacturerId != null) {
      dialogoMelhorMarca = await melhorDeltaE(r, g, b, { targetManufacturerId: manufacturerId });
    }
    dialogoMelhorTodos = await melhorDeltaE(r, g, b, {});
  }

  /** A escolha vale para as demais regiões da aba (RN4) e dispara o recálculo. */
  async function escolherSaida(saida: 'marca' | 'todos'): Promise<void> {
    saidaAutorizada = saida;
    tabs[activeTabIndex].saidaAutorizada = saida;
    regiaoNoDialogo = null;
    devolverFoco();
    marcarAlteracao();
    await Promise.all(regions.map(r => computeRegion(r.id)));
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
    // CA19: fechar sem escolher não decide nada.
    regiaoNoDialogo = null;
    devolverFoco();
  }

  /** RN8: trocar qualquer interruptor limpa a resposta lembrada e recalcula. */
  async function onUseStockOnlyChange(valor: boolean): Promise<void> {
    useStockOnly = valor;
    saidaAutorizada = null;
    tabs[activeTabIndex].useStockOnly = valor;
    tabs[activeTabIndex].saidaAutorizada = null;
    universoVazioMotivo = null;
    marcarAlteracao();
    await Promise.all(regions.map(r => computeRegion(r.id)));
  }

  async function computeRegion(id: number, tabUidChamada: number = tabs[activeTabIndex].uid) {
    const alvo = () => regiaoDaAba(tabUidChamada, id);
    const inicio = alvo();
    if (!inicio) return;
    inicio.computing = true;
    try {
      const resp = await suggestRecipeForColor(inicio.r, inicio.g, inicio.b, universoAtual());
      if (ehUniversoVazio(resp)) {
        // RN9: universo sem tinta não é erro — a tela diz o motivo e o
        // usuário decide se abre o universo.
        universoVazioMotivo = resp.motivo;
        const vazio = alvo();
        if (vazio) vazio.result = null;
        return;
      }
      universoVazioMotivo = null;
      const agora = alvo();
      if (agora) agora.result = resp;
      // RN3/RN4: estourou o limiar e a aba ainda não respondeu o diálogo.
      if (resp.faixa === 'nao-encontrei' && saidaAutorizada === null) {
        void abrirDialogoDeFallback(id, inicio.r, inicio.g, inicio.b);
      }
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
    // RN8: trocar de fabricante base limpa a resposta lembrada — a mesma
    // disciplina do estoque em onUseStockOnlyChange.
    saidaAutorizada = null;
    tabs[activeTabIndex].saidaAutorizada = null;
    universoVazioMotivo = null;
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
        foraDoUniverso: s?.foraDoUniverso ?? false,
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
      // rf-11 RN7: sem foraDoUniverso na resposta (T1/T3 não o enviam), a
      // marcação é falsa — só true quando o rf-11 explicitamente autorizou.
      foraDoUniverso: r.result.foraDoUniverso ?? false,
    };
  }

  function regiaoParaRascunho(r: PlannerRegion, i: number): RegiaoRascunho {
    return { ...mapRegionCommon(r, i), painted: r.painted };
  }

  function regiaoParaDTO(r: PlannerRegion, i: number): RegiaoDTO {
    const comum = mapRegionCommon(r, i);
    return { ...comum, painted: r.painted ? 1 : 0, foraDoUniverso: comum.foraDoUniverso ? 1 : 0 };
  }

  function buildAbaDTO(aba: AbaState): AbaDTO {
    return {
      id: aba.serverId,
      name: aba.name,
      imageData: aba.imageDataUrl,
      selectedManufacturerId: aba.manufacturerId,
      useStockOnly: aba.useStockOnly ? 1 : 0,
      regions: aba.regions.map((r, i) => regiaoParaDTO(r, i)),
    };
  }

  function buildAbaRascunho(aba: AbaState): AbaRascunho {
    return {
      id: aba.serverId ?? null,
      name: aba.name,
      imageData: aba.imageDataUrl,
      selectedManufacturerId: aba.manufacturerId,
      useStockOnly: aba.useStockOnly,
      regions: aba.regions.map((r, i) => regiaoParaRascunho(r, i)),
    };
  }

  function buildPlanoDTO(): PlanoDTO {
    snapshotActiveIntoTabs();
    return {
      id: planId ?? undefined,
      name: planName,
      tabs: tabs.map(buildAbaDTO),
    };
  }

  // ── rf-07/rf-09: auto save local (RN3/RN4/RN5) ──
  /** Chamada explícita nos mutadores de CONTEÚDO — nunca em zoom/pan/view
   *  (RN4/CA21) e nunca antes da hidratação terminar (RN5). Criar, renomear,
   *  reordenar e excluir aba TAMBÉM chamam esta função (RN5/CA22); trocar de
   *  aba nunca chama (switchTab só faz snapshot). */
  function marcarAlteracao() {
    if (!hidratado) {
      alteradoDuranteHidratacao = true;
      return;
    }
    snapshotActiveIntoTabs();
    const payload: Rascunho = {
      v: 2,
      planId,
      name: planName,
      abaAtiva: activeTabIndex,
      tabs: tabs.map(buildAbaRascunho),
      salvoEm: new Date().toISOString(),
    };
    alteracaoSeq++;
    agendarGravacao(payload);
    // A escada de cota (RN8) só se resolve dentro do módulo depois do
    // debounce de 1500 ms — relê um pouco depois para refletir na tela.
    setTimeout(() => { autoSaveStatus = estadoAutoSave(); }, 1600);
  }

  // ── rf-07/rf-09: hidratar / aplicar plano carregado ──
  function resetToEmpty() {
    planId = null;
    planName = '';
    ownedPaints = new Set();
    tabs = [emptyAba()];
    activeTabIndex = 0;
    loadTabIntoWorkingState(0);
  }

  async function computeRegionInAba(aba: AbaState, id: number): Promise<void> {
    const alvo = () => aba.regions.find(r => r.id === id) ?? null;
    const inicio = alvo();
    if (!inicio) return;
    inicio.computing = true;
    try {
      const resp = await suggestRecipeForColor(inicio.r, inicio.g, inicio.b, {
        targetManufacturerId: aba.saidaAutorizada === 'todos' ? undefined : aba.manufacturerId ?? undefined,
        useStockOnly: aba.saidaAutorizada === null && aba.useStockOnly,
        foraDoUniverso: aba.saidaAutorizada !== null,
      });
      const agora = alvo();
      if (agora) agora.result = ehUniversoVazio(resp) ? null : resp;
    } catch (e) {
      console.error('Erro ao calcular região:', e);
      const agora = alvo();
      if (agora) agora.result = null;
    } finally {
      const agora = alvo();
      if (agora) agora.computing = false;
    }
  }

  async function recomputeAba(aba: AbaState): Promise<void> {
    if (aba.manufacturerId == null) return;
    await Promise.all(aba.regions.map(r => computeRegionInAba(aba, r.id)));
  }

  /** Aplica um `PlanoDTO` (do servidor ou remontado do rascunho) ao estado
   *  da tela — uma `AbaState` por `AbaDTO`. Nunca chama `marcarAlteracao` —
   *  é hidratação, não input. `abaAtivaIndex` (rascunho) escolhe qual aba
   *  fica em foco; planos vindos do servidor sempre abrem na primeira. */
  async function applyLoadedPlan(plano: PlanoDTO, recalcular = true, abaAtivaIndex = 0): Promise<void> {
    planId = plano.id ?? null;
    planName = plano.name;
    // rf-09/"Não faz": o checklist "tenho" não sobrevive ao recarregar o plano.
    ownedPaints = new Set();

    const tabsDTO = plano.tabs.length > 0 ? plano.tabs : [
      { name: '', imageData: '', selectedManufacturerId: null, useStockOnly: 0 as const, regions: [] },
    ];

    const novasAbas: AbaState[] = [];
    for (const abaDTO of tabsDTO) {
      let nextIdLocal = 1;
      const abaRegions: PlannerRegion[] = abaDTO.regions.map((rd): PlannerRegion => ({
        id: nextIdLocal++,
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
          foraDoUniverso: rd.foraDoUniverso === 1,
        },
      }));
      const image = abaDTO.imageData ? await loadImageBitmap(abaDTO.imageData) : null;
      novasAbas.push({
        uid: abaUidSeq++,
        serverId: abaDTO.id,
        name: abaDTO.name,
        useStockOnly: abaDTO.useStockOnly === 1,
        saidaAutorizada: null,
        imageDataUrl: abaDTO.imageData || '',
        hasImage: !!abaDTO.imageData,
        image,
        regions: abaRegions,
        nextId: nextIdLocal,
        selectedId: null,
        view: { zoom: 1, panX: 0, panY: 0 },
        manufacturerId: abaDTO.selectedManufacturerId ?? null,
      });
    }

    // CA8: restaurar rascunho não consulta o servidor — a tinta já veio no
    // próprio rascunho. Só o plano vindo do banco recalcula (por aba).
    if (recalcular) {
      await Promise.all(novasAbas.map(recomputeAba));
    }

    tabs = novasAbas;
    activeTabIndex = Math.min(Math.max(abaAtivaIndex, 0), tabs.length - 1);
    loadTabIntoWorkingState(activeTabIndex);
  }

  /** RN5 — entrada em T2: rascunho vence; senão plano ativo; senão vazia. */
  async function hidratarT2() {
    try {
      const { rascunho, corrompido, truncado } = lerRascunho();
      if (corrompido) {
        toast(t('draftCorrupted'), 'error');
      }
      if (rascunho) {
        // RN8 cortou a foto de TODAS as abas para caber na cota: a imagem
        // continua no banco, então busca de lá (por posição) em vez de
        // reabrir o plano sem foto — sem ela o Salvar seguinte apagaria a
        // foto das abas no servidor.
        const semFoto = rascunho.tabs.some(aba => !aba.imageData);
        let abasServidor: AbaDTO[] | null = null;
        if (semFoto && rascunho.planId != null) {
          try {
            const salvo = await carregarPlano(rascunho.planId);
            abasServidor = salvo.tabs;
          } catch {
            toast(t('draftNoPhoto'), 'error');
          }
        }
        // achado 2 do guardrail rf-09: casa a aba do rascunho com a do
        // servidor por identidade estável — o `id` da aba quando existir, e
        // o nome como segundo critério — nunca pela posição na tira, que
        // pode ter sido reordenada depois do último Salvar.
        const abaServidorPorIdentidade = (rt: AbaRascunho): AbaDTO | undefined => {
          if (!abasServidor) return undefined;
          if (rt.id != null) {
            const porId = abasServidor.find(a => a.id === rt.id);
            if (porId) return porId;
          }
          return abasServidor.find(a => a.name === rt.name);
        };
        const tabsDTO: AbaDTO[] = rascunho.tabs.map((rt) => ({
          id: undefined,
          name: rt.name,
          imageData: rt.imageData || abaServidorPorIdentidade(rt)?.imageData || '',
          selectedManufacturerId: rt.selectedManufacturerId,
          useStockOnly: rt.useStockOnly ? 1 : 0,
          regions: rt.regions.map((rr): RegiaoDTO => ({
            x: rr.x, y: rr.y, r: rr.r, g: rr.g, b: rr.b, hex: rr.hex,
            regionName: rr.regionName, note: rr.note,
            paintId: rr.paintId, paintBrand: rr.paintBrand,
            paintName: rr.paintName, paintCode: rr.paintCode,
            deltaE: rr.deltaE, painted: rr.painted ? 1 : 0,
          })),
        }));
        await applyLoadedPlan(
          { id: rascunho.planId ?? undefined, name: rascunho.name, tabs: tabsDTO },
          false,
          rascunho.abaAtiva,
        );
        draftBannerVisible = true;
        // CAN5/CAN8: reaproveita a mensagem de teto de regiões — mesmo limite,
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
          if (!alteradoDuranteHidratacao) await applyLoadedPlan(plano);
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
        await applyLoadedPlan(plano);
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
      // validarPlano devolve a chave i18n e, para erro de aba (RN15), o nome
      // dela — validationErrorText compõe a mensagem final.
      validationError = erro;
      return;
    }
    validationError = null;
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

  <!-- rf-09: tira de abas — uma por figura. Aba ativa se distingue por texto
       E contorno (nunca só cor). Botões de mover/excluir seguem o alvo de
       toque de 44×44 do resto de T2; nenhum componente visual novo. -->
  <div
    role="tablist"
    aria-label={t('tabsListLabel')}
    style="flex-shrink: 0; display: flex; align-items: center; gap: 6px; padding: 8px 20px; border-bottom: 1px solid var(--color-line); overflow-x: auto;"
  >
    {#each tabs as aba, i (aba.uid)}
      <div style="display: flex; align-items: center; gap: 2px; flex-shrink: 0;">
        {#if renamingTabIndex === i}
          <!-- achado 4 do guardrail rf-09/CA28: o `tabpanel` referencia
               `t2-tab-btn-{i}` por `aria-labelledby`; enquanto o botão vira
               o campo de rename esse id some do DOM. Este span invisível
               mantém a referência válida durante a edição. -->
          <span id={`t2-tab-btn-${i}`} style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{displayTabName(aba, i)}</span>
          <label for={`t2-tab-rename-${aba.uid}`} style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{t('renameTabLabel')}</label>
          <input
            id={`t2-tab-rename-${aba.uid}`}
            type="text"
            bind:value={tabs[i].name}
            maxlength="80"
            onblur={commitRename}
            onkeydown={onRenameKeydown}
            style="min-width: 120px; height: 44px; padding: 0 10px; border: 2px solid var(--color-accent); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-family: inherit; font-size: 13.5px;"
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
            style="display: inline-flex; flex-direction: column; align-items: flex-start; justify-content: center; gap: 2px; min-width: 44px; min-height: 44px; padding: 4px 12px; border: 2px solid {i === activeTabIndex ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {i === activeTabIndex ? 'var(--color-accent-panel)' : 'var(--color-bg)'}; color: {i === activeTabIndex ? 'var(--color-accent-400)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: {i === activeTabIndex ? '700' : '500'}; cursor: pointer;"
          >
            <span>{displayTabName(aba, i)}</span>
            <span class="font-mono" style="font-size: 11px; opacity: 0.85;">{tabProgressLabel(i)}</span>
          </button>
        {/if}
        <button
          aria-label={t('renameTabLabel')}
          title={t('renameTabLabel')}
          onclick={() => startRename(i)}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;"
        ><i class="ph ph-pencil-simple" style="font-size: 15px;"></i></button>
        <button
          aria-label={t('moveTabLeft')}
          title={t('moveTabLeft')}
          disabled={i === 0}
          onclick={() => moveTab(i, -1)}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer; opacity: {i === 0 ? 0.4 : 1};"
        ><i class="ph ph-caret-left" style="font-size: 15px;"></i></button>
        <button
          aria-label={t('moveTabRight')}
          title={t('moveTabRight')}
          disabled={i === tabs.length - 1}
          onclick={() => moveTab(i, 1)}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer; opacity: {i === tabs.length - 1 ? 0.4 : 1};"
        ><i class="ph ph-caret-right" style="font-size: 15px;"></i></button>
        <button
          aria-label={t('deleteTabBtn')}
          title={t('deleteTabBtn')}
          onclick={() => onDeleteTabClick(i)}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-500); cursor: pointer;"
        ><i class="ph ph-trash-simple" style="font-size: 15px;"></i></button>
      </div>
    {/each}
    <button
      aria-label={t('addTabBtn')}
      title={t('addTabBtn')}
      onclick={addTab}
      style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); cursor: pointer; flex-shrink: 0;"
    ><i class="ph ph-plus" style="font-size: 18px;"></i></button>
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
        <span style="font-size: 15px; color: var(--color-neutral-400);">{t('progress', { a: paintedCount, b: totalRegionsCount })}</span>
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

        <!-- rf-11: os dois interruptores são independentes. Ligados juntos,
             o universo é a interseção: só o que eu tenho daquela marca. -->
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

        {#if universoVazioMotivo}
          <p role="alert" style="margin: 8px 0 0; font-size: 12.5px; color: var(--color-warning, #E4A11B);">
            {universoVazioMotivo}
          </p>
        {/if}

        {#if regiaoNoDialogo !== null}
          <!-- rf-11 RN4/RN5: uma vez por aba, com as duas saídas na ordem e o
               melhor ΔE00 alcançável em cada uma. -->
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
                >
                  <i class="ph-bold ph-check" style="font-size: 14px;"></i>{r.painted ? t('painted') : t('toPaint')}
                </button>
              </div>
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

        {#if validationErrorText}
          <div role="alert" style="font-size: 13px; color: var(--color-neutral-300);">
            <i class="ph ph-warning-circle" style="margin-right: 6px;"></i>{validationErrorText}
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

  .t2-tab-btn:hover {
    border-color: var(--color-accent-700);
  }

  .t2-tab-btn:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
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
