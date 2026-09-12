<script lang="ts">
  // T4 — Minhas tintas (rf-04). Consolida CatalogoView + StockManager numa lista
  // única de tintas (posse alternável linha a linha) + aba Fabricantes, fiel ao
  // protótipo Nocturne (tests/fixtures/mockup/t4-tintas.html — D-001, Epic D).
  //
  // PENDÊNCIA DECLARADA (spec rf-04, seção Dependências #3 / risco R1
  // herdado de docs/mesclaai-userstory.md): não existe rota de escrita para
  // tinta em api/httpapi. O catálogo de tintas servido (GET /paints) é
  // só-leitura aqui. Por isso:
  //   - "Cadastrar/Editar/Excluir tinta" grava em services/stock.svelte.ts
  //     (localStorage, MESMO mecanismo já usado por T1 "Só o que eu tenho" e
  //     pela prioridade de estoque) — cadastro real no catálogo do servidor
  //     fica bloqueado até confirmação do contrato de escrita.
  //   - O toggle "tenho/não tenho" numa linha da lista escreve em
  //     stock.svelte.ts (por código+fabricante — RG-17), nunca no paint do
  //     servidor.
  //   - Fabricantes vivem no servidor (rf-14): buscar, cadastrar, alterar e
  //     excluir usam /manufacturers. Só sai fabricante sem nenhuma tinta —
  //     catálogo, estoque do servidor ou estoque deste aparelho (RG-18
  //     reescrita em 2026-09-11) — e nada é apagado em cascata. A lista local
  //     antiga (mescla.customMfrs.v1) sobe uma vez ao abrir T4.
  //
  // D-001 (fidelidade ao protótipo):
  //   - O protótipo funde a antiga lista "tenho" + catálogo virtualizado numa
  //     única lista de linhas (toggle tenho/não-tenho à esquerda, "Editar" à
  //     direita). "Editar" numa linha que ainda não é minha edita a própria
  //     linha: salvar grava no estoque uma cópia ligada ao catálogo por
  //     `catalogId`, e ela toma o lugar da linha do catálogo (D-013). Linha do
  //     estoque ganha "Excluir", sempre atrás da confirmação do tema.
  //   - O filtro "só as que eu tenho" (onlyMine) é novo nesta tela — mesmo
  //     rótulo/mecanismo já usado em T1.
  //   - O detalhe de tinta com atalho "gerar mistura equivalente"
  //     (PaintDetailSheet) NÃO existe no protótipo aprovado desta tela —
  //     removido daqui; cada linha só alterna posse ou abre o formulário.
  //   - "Ler do pote" (câmera, US-20/E9) lê a cor da foto no próprio
  //     navegador (canvas, média do miolo da imagem). A foto não é enviada a
  //     lugar nenhum e nenhuma rota HTTP nova foi criada.
  //   - Os modais (cadastro/edição, novo fabricante, confirmações) ficam
  //     sempre montados no DOM e alternam via `display:none` em vez de
  //     `{#if}` — mesma técnica do protótipo (sc-if com hint-placeholder,
  //     nós sempre presentes, visibilidade que alterna).
  import Header from '../components/Header.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import VirtualList from '../components/VirtualList.svelte';
  import Combobox from '../components/Combobox.svelte';
  import { onMount, tick } from 'svelte';
  import {
    allManufacturers, allPaints, hexOf, searchPaints, type Paint, type Manufacturer,
    reloadManufacturers, createManufacturer, updateManufacturer, deleteManufacturer,
    migrateLegacyManufacturers, MAX_MANUFACTURER_NAME, paintById,
    allPaintTypes, reloadPaintTypes, createPaintType, updatePaintType, deletePaintType,
    MAX_PAINT_TYPE_NAME, type PaintType,
  } from '../services/catalog';
  import {
    stock, sortedStock, addStockPaint, updateStockPaint, removeStockPaint,
    relinkStockManufacturers, localStockCountFor, fillStockPaintTypes, localStockCountForType,
    fillStockQuantities,
  } from '../services/stock.svelte';
  import { catalogRev } from '../services/catalogRev.svelte';
  import type { StockPaint } from '../services/engine';
  import { appState } from '../appState.svelte';
  import { t } from '../i18n.svelte';
  import { toast } from '../toast.svelte';


  type Tab = 'tintas' | 'fabricantes' | 'tipos';
  let tab: Tab = $state('tintas');

  let search = $state('');
  let mfrFilter: number | null = $state(null);
  let ptypeFilter: number | null = $state(null);
  let onlyMine = $state(false);

  $effect(() => {
    if (appState.pendingCatalogMfrId != null) {
      mfrFilter = appState.pendingCatalogMfrId;
      appState.pendingCatalogMfrId = null;
      tab = 'tintas';
    }
  });

  // Ao abrir T4: a lista local antiga sobe uma vez, a lista vem do servidor e o
  // estoque local é religado pelo id (rf-14, RN5/RN6).
  onMount(() => {
    void (async () => {
      const pending = await migrateLegacyManufacturers();
      if (pending > 0) toast(t('errMakerMigrate'), 'error');
      await refreshMakers();
      await refreshPTypes();
      backfillStockPaintTypes();
      fillStockQuantities();
    })();
  });

  let allMfrs: Manufacturer[] = $derived.by(() => {
    void catalogRev.n;
    return [...allManufacturers()];
  });

  // allPTypes/ptypeNameById declarados aqui (não lá embaixo, com o resto de
  // rf-15) porque listNote já precisa do nome do tipo filtrado.
  let allPTypes: PaintType[] = $derived.by(() => {
    void catalogRev.n;
    return [...allPaintTypes()];
  });
  let ptypeNameById = $derived(new Map(allPTypes.map(pt => [pt.id, pt.name])));

  let makerSearch = $state('');
  function foldText(s: string): string {
    return s.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
  }
  let filteredMakers = $derived.by(() => {
    const q = foldText(makerSearch.trim());
    return q ? allMfrs.filter(m => foldText(m.name).includes(q)) : allMfrs;
  });

  function stockKeyOf(manufacturer: string, code: string): string {
    return `${manufacturer}|${code}`;
  }
  let stockKeys = $derived(new Set(stock.paints.filter(p => p.code !== '').map(p => stockKeyOf(p.manufacturer, p.code))));

  // Tinta do estoque que nasceu de uma linha do catálogo guarda `catalogId`
  // (D-013): editar código ou fabricante não traz a linha original de volta
  // como duplicata. Estoque gravado antes do campo ainda casa por
  // código+fabricante.
  let catalogIdByKey = $derived.by(() => {
    void catalogRev.n;
    return new Map(allPaints().filter(p => p.code !== '').map(p => [stockKeyOf(p.manufacturer, p.code), p.id]));
  });
  function catalogOriginOf(sp: StockPaint): number | undefined {
    return sp.catalogId ?? (sp.code !== '' ? catalogIdByKey.get(stockKeyOf(sp.manufacturer, sp.code)) : undefined);
  }
  let linkedCatalogIds = $derived(new Set(stock.paints.map(catalogOriginOf).filter((id): id is number => id != null)));

  // ── Aba Tintas ──
  let filteredCatalog = $derived.by(() => {
    void catalogRev.n;
    const base = search.trim() ? searchPaints(search, { manufacturerId: mfrFilter ?? undefined, limit: 100000 }) : allPaints().filter(p => mfrFilter == null || p.manufacturerId === mfrFilter);
    return base.filter(p => ptypeFilter == null || p.paintTypeId === ptypeFilter);
  });

  let filteredStock = $derived(
    sortedStock().filter(p => {
      if (mfrFilter != null) {
        const mfrName = allMfrs.find(m => m.id === mfrFilter)?.name;
        if (p.manufacturer !== mfrName) return false;
      }
      if (ptypeFilter != null && p.paintTypeId !== ptypeFilter) return false;
      if (!search.trim()) return true;
      const q = search.toLowerCase();
      return p.name.toLowerCase().includes(q) || p.code.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q);
    })
  );

  function toggleHave(p: Paint) {
    const sp = stock.paints.find(s => catalogOriginOf(s) === p.id);
    if (sp) {
      removeStockPaint(sp.id);
    } else {
      addStockPaint({
        catalogId: p.id,
        manufacturerId: p.manufacturerId,
        manufacturer: p.manufacturer,
        name: p.name,
        code: p.code,
        r: p.r, g: p.g, b: p.b,
        volume: '',
        notes: '',
      });
    }
  }

  // ── Lista única (posse alternável + editar), fiel ao protótipo ──
  interface Row {
    key: string;
    have: boolean;
    name: string;
    meta: string;
    hex: string;
    toggle: () => void;
    edit: () => void;
    del?: () => void;
  }

  let mergedRows = $derived.by(() => {
    const rows: Row[] = filteredStock.map(p => ({
      key: `s${p.id}`,
      have: true,
      name: p.name,
      meta: `${p.code || '—'} · ${p.manufacturer}${typeSuffix(p.paintTypeId)}${p.volume ? ` · ${p.volume}` : ''}${(p.quantity ?? 1) > 1 ? ` · ${t('qtyPots', { n: p.quantity ?? 1 })}` : ''}`,
      hex: rgbToHex(p.r, p.g, p.b),
      // Desmarcar tinta do catálogo só devolve a linha ao catálogo; tinta
      // cadastrada à mão sumiria de vez, então passa pela confirmação.
      toggle: () => {
        if (catalogOriginOf(p) != null) removeStockPaint(p.id);
        else askDeletePaint(p);
      },
      edit: () => openEdit(p),
      del: () => askDeletePaint(p),
    }));
    if (!onlyMine) {
      for (const p of filteredCatalog) {
        if (linkedCatalogIds.has(p.id) || stockKeys.has(stockKeyOf(p.manufacturer, p.code))) continue;
        rows.push({
          key: `c${p.id}`,
          have: false,
          name: p.name,
          meta: `${p.code} · ${p.manufacturer}${typeSuffix(p.paintTypeId)}`,
          hex: hexOf(p),
          toggle: () => toggleHave(p),
          edit: () => openEditCatalog(p),
        });
      }
    }
    return rows;
  });

  let listNote = $derived(
    (onlyMine ? t('listNoteMine') : t('listNoteAll')) +
      (mfrFilter != null ? t('inMaker', { brand: allMfrs.find(m => m.id === mfrFilter)?.name ?? '' }) : t('inAll', { n: allMfrs.length })) +
      (ptypeFilter != null ? ` · ${ptypeNameById.get(ptypeFilter)}` : '')
  );

  let headerCount = $derived(t('estanteCount', { a: filteredStock.length, b: filteredCatalog.length, c: allMfrs.length }));

  // ── Modal cadastro/edição (tinta) ──
  let formOpen = $state(false);
  let editingId: number | null = $state(null);
  // Linha do catálogo (ainda não minha) aberta em "Editar" — D-013.
  let editingCatalogId: number | null = $state(null);
  let editing = $derived(editingId !== null || editingCatalogId !== null);
  let fMfr: number | '' = $state('');
  let fPType: number | '' = $state('');
  let fName = $state('');
  let fCode = $state('');
  let fHex = $state('#8a8a8a');
  let fVolume = $state('');
  let fQty = $state(1);
  // Toggle visual "Tenho este pote agora" (fiel ao mockup) — o estoque local
  // não tem coluna `have` própria: estar na lista já É "tenho" (RG-17). Sem
  // essa coluna, desmarcar aqui não muda o que saveForm() grava; é limitação
  // declarada, não comportamento fingido.
  let fHave = $state(true);
  let saving = $state(false);
  // O modal fica sempre montado (alterna só `display`) e o corpo rolável guarda
  // a posição de rolagem entre aberturas — sem isto, reabrir o formulário
  // continuaria de onde a última edição parou.
  let formBodyEl: HTMLDivElement | null = $state(null);
  async function resetFormScroll() {
    await tick(); // espera o `display` sair de `none` (scrollTop não pega em elemento escondido)
    requestAnimationFrame(() => {
      if (formBodyEl) formBodyEl.scrollTop = 0;
    });
  }

  function hexToRgb(hex: string): { r: number; g: number; b: number } {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
    if (!m) return { r: 138, g: 138, b: 138 };
    const n = parseInt(m[1], 16);
    return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
  }
  function rgbToHex(r: number, g: number, b: number): string {
    const h = (n: number) => n.toString(16).padStart(2, '0');
    return `#${h(r)}${h(g)}${h(b)}`;
  }
  function isValidHex(hex: string): boolean {
    return /^#[0-9a-fA-F]{6}$/.test(hex.trim());
  }

  let fRgb = $derived(hexToRgb(fHex));

  // O seletor nativo (<input type="color">) só aceita #rrggbb. Enquanto o campo
  // de texto está pela metade, o seletor mostra a cor do preview (cinza padrão)
  // em vez de recusar o valor.
  let pickerHex = $derived(isValidHex(fHex) ? fHex.trim().toLowerCase() : rgbToHex(fRgb.r, fRgb.g, fRgb.b));

  function onPickColor(ev: Event) {
    fHex = (ev.currentTarget as HTMLInputElement).value.toUpperCase();
  }

  // ── "Ler do pote": foto → cor (US-20/E9) ──
  // Leitura local, no navegador: nenhuma rota nova. O <input type="file"
  // capture="environment"> abre a câmera traseira no iPad e o seletor de
  // arquivo no desktop — o mesmo elemento cobre os dois.
  let potInputEl: HTMLInputElement | null = $state(null);
  let readingPot = $state(false);

  function openPotCamera() {
    potInputEl?.click();
  }

  function loadImage(src: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => resolve(img);
      img.onerror = () => reject(new Error('decode'));
      img.src = src;
    });
  }

  async function hexFromPhoto(file: File): Promise<string | null> {
    const url = URL.createObjectURL(file);
    try {
      const img = await loadImage(url);
      const side = 64;
      const canvas = document.createElement('canvas');
      canvas.width = side;
      canvas.height = side;
      const ctx = canvas.getContext('2d', { willReadFrequently: true });
      if (!ctx) return null;
      ctx.drawImage(img, 0, 0, side, side);
      // Média só do miolo (50% central): a borda da foto pega mesa e fundo,
      // não a tinta do pote.
      const box = Math.floor(side / 2);
      const origin = Math.floor(side / 4);
      const { data } = ctx.getImageData(origin, origin, box, box);
      let r = 0, g = 0, b = 0, n = 0;
      for (let i = 0; i < data.length; i += 4) {
        if (data[i + 3] < 128) continue;
        r += data[i];
        g += data[i + 1];
        b += data[i + 2];
        n++;
      }
      if (n === 0) return null;
      return rgbToHex(Math.round(r / n), Math.round(g / n), Math.round(b / n)).toUpperCase();
    } finally {
      URL.revokeObjectURL(url);
    }
  }

  async function onPotFile(ev: Event) {
    const input = ev.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    // Zera o valor para que a mesma foto possa ser escolhida de novo.
    input.value = '';
    if (!file) return;
    readingPot = true;
    try {
      const hex = await hexFromPhoto(file);
      if (!hex) {
        toast(t('potReadFail'), 'error');
        return;
      }
      fHex = hex;
      toast(t('potReadOk'));
    } catch {
      toast(t('potReadFail'), 'error');
    } finally {
      readingPot = false;
    }
  }
  let fMfrName = $derived(allMfrs.find(m => m.id === fMfr)?.name ?? '');
  let formPreviewMeta = $derived(`${fCode || '—'} · ${fMfrName}${fVolume ? ` · ${fVolume}` : ''}`);

  // Só um card por vez. `modalKind` já escolhe um por prioridade, mas deixar
  // duas flags ligadas faria a ação abrir um card e mostrar outro.
  function closeOtherModals() {
    formOpen = false;
    makerFormOpen = false;
    ptypeFormOpen = false;
    confirmPaintDel = null;
    confirmMakerDel = null;
    confirmPTypeDel = null;
  }

  function openAdd() {
    closeOtherModals();
    editingId = null;
    editingCatalogId = null;
    fMfr = mfrFilter ?? '';
    fPType = defaultPTypeId() ?? '';
    fName = '';
    fCode = '';
    fHex = '#8a8a8a';
    fVolume = '';
    fQty = 1;
    fHave = true;
    formOpen = true;
    void resetFormScroll();
  }

  function openEdit(p: StockPaint) {
    closeOtherModals();
    editingId = p.id;
    editingCatalogId = null;
    const m = allMfrs.find(x => x.name === p.manufacturer);
    fMfr = m?.id ?? '';
    fPType = p.paintTypeId ?? '';
    fName = p.name;
    fCode = p.code;
    fHex = rgbToHex(p.r, p.g, p.b);
    fVolume = p.volume;
    fQty = p.quantity ?? 1;
    fHave = true;
    formOpen = true;
    void resetFormScroll();
  }

  function openEditCatalog(p: Paint) {
    closeOtherModals();
    editingId = null;
    editingCatalogId = p.id;
    fMfr = p.manufacturerId;
    fPType = p.paintTypeId || '';
    fName = p.name;
    fCode = p.code;
    fHex = rgbToHex(p.r, p.g, p.b);
    fVolume = '';
    fQty = 1;
    fHave = true;
    formOpen = true;
    void resetFormScroll();
  }

  function saveForm() {
    if (!fName.trim()) {
      toast(t('errNameRequired'), 'error');
      return;
    }
    if (!isValidHex(fHex)) {
      toast(t('errHexInvalid'), 'error');
      return;
    }
    if (fMfr === '') {
      toast(t('errMakerRequired'), 'error');
      return;
    }
    if (!Number.isInteger(Number(fQty)) || Number(fQty) < 1) {
      toast(t('errQtyInvalid'), 'error');
      return;
    }
    saving = true;
    const { r, g, b } = hexToRgb(fHex);
    const mfr = allMfrs.find(m => m.id === Number(fMfr));
    const base = {
      manufacturerId: Number(fMfr),
      manufacturer: mfr?.name ?? '',
      paintTypeId: fPType === '' ? undefined : Number(fPType),
      name: fName.trim(),
      code: fCode.trim(),
      r, g, b,
      volume: fVolume.trim(),
      quantity: Number(fQty),
      notes: '',
    };
    if (editingId !== null) {
      // A origem sai do registro ANTES da troca: código e fabricante podem
      // mudar agora e o casamento por chave se perderia.
      const prev = stock.paints.find(x => x.id === editingId);
      updateStockPaint({ ...base, id: editingId, catalogId: prev ? catalogOriginOf(prev) : undefined });
    } else if (editingCatalogId !== null) {
      addStockPaint({ ...base, catalogId: editingCatalogId });
    } else {
      addStockPaint(base);
    }
    saving = false;
    formOpen = false;
    toast(t('saveChanges'));
  }

  // ── Confirmação de exclusão (RG-18/CAN1) ──
  let confirmPaintDel: StockPaint | null = $state(null);
  let confirmMakerDel: Manufacturer | null = $state(null);

  function askDeletePaint(p?: StockPaint) {
    const alvo = p ?? (editingId != null ? stock.paints.find(x => x.id === editingId) : undefined);
    if (alvo) confirmPaintDel = alvo;
  }
  function doDeletePaint() {
    if (!confirmPaintDel) return;
    removeStockPaint(confirmPaintDel.id);
    confirmPaintDel = null;
    formOpen = false;
  }

  // ── Excluir fabricante (RG-18 reescrita, rf-14 RN3) ──
  // Tinta prende o fabricante: com qualquer contagem > 0 o modal só explica e
  // não chama a rota. Catálogo e estoque aparecem separados, sem somar — a
  // cópia local de uma tinta do catálogo não conta duas vezes.
  let deletingMaker = $state(false);
  let makerDelCounts = $derived.by(() => {
    const m = confirmMakerDel;
    return m ? { catalog: m.paintCount, stock: m.userPaintCount + localStockCountFor(m) } : { catalog: 0, stock: 0 };
  });
  let makerDelBlocked = $derived(makerDelCounts.catalog > 0 || makerDelCounts.stock > 0);

  function statusOf(e: unknown): number {
    return (e as { status?: number } | null)?.status ?? 0;
  }

  async function refreshMakers() {
    try {
      await reloadManufacturers();
    } catch {
      // Sem rede, a lista em memória continua valendo.
    }
    relinkStockManufacturers(allManufacturers());
  }

  function askDeleteMaker(m: Manufacturer) {
    closeOtherModals();
    confirmMakerDel = m;
  }
  async function doDeleteMaker() {
    if (!confirmMakerDel || makerDelBlocked || deletingMaker) return;
    const { id } = confirmMakerDel;
    deletingMaker = true;
    try {
      await deleteManufacturer(id);
      confirmMakerDel = null;
      if (mfrFilter === id) mfrFilter = null;
      await refreshMakers();
    } catch (e) {
      const status = statusOf(e);
      if (status === 404 || status === 409) {
        confirmMakerDel = null;
        toast(t(status === 404 ? 'errMakerGone' : 'errMakerBusy'), 'error');
        await refreshMakers();
      } else {
        toast(t('errMakerNet'), 'error');
      }
    } finally {
      deletingMaker = false;
    }
  }

  // ── Cadastrar / alterar fabricante (um modal só) ──
  let makerFormOpen = $state(false);
  let editingMakerId: number | null = $state(null);
  let makerNameInput = $state('');
  let savingMaker = $state(false);

  function openNewMaker() {
    closeOtherModals();
    editingMakerId = null;
    makerNameInput = '';
    makerFormOpen = true;
  }
  function openEditMaker(m: Manufacturer) {
    closeOtherModals();
    editingMakerId = m.id;
    makerNameInput = m.name;
    makerFormOpen = true;
  }

  function makerNameProblem(name: string): 'errMakerNameRequired' | 'errMakerNameTooLong' | null {
    if (!name) return 'errMakerNameRequired';
    if ([...name].length > MAX_MANUFACTURER_NAME) return 'errMakerNameTooLong';
    return null;
  }

  async function saveMaker() {
    if (savingMaker) return;
    const name = makerNameInput.trim();
    const problem = makerNameProblem(name);
    if (problem) {
      toast(t(problem), 'error');
      return;
    }
    savingMaker = true;
    try {
      if (editingMakerId === null) {
        const created = await createManufacturer(name);
        makerFormOpen = false;
        await refreshMakers();
        mfrFilter = created.id;
        tab = 'tintas';
      } else {
        await updateManufacturer(editingMakerId, name);
        makerFormOpen = false;
        await refreshMakers();
        toast(t('saveChanges'));
      }
    } catch (e) {
      const status = statusOf(e);
      if (status === 409) {
        toast(t('errMakerDup'), 'error');
      } else if (status === 400) {
        toast(t(makerNameProblem(name) ?? 'errMakerNameRequired'), 'error');
      } else if (status === 404) {
        makerFormOpen = false;
        toast(t('errMakerGone'), 'error');
        await refreshMakers();
      } else {
        toast(t('errMakerNet'), 'error');
      }
    } finally {
      savingMaker = false;
    }
  }

  function seePaints(mfrId: number) {
    mfrFilter = mfrId;
    tab = 'tintas';
  }

  function swatchesFor(m: Manufacturer): string[] {
    const fromCatalog = m.paintCount > 0 ? allPaints().filter(p => p.manufacturerId === m.id).slice(0, 5) : [];
    const need = 5 - fromCatalog.length;
    const name = m.name.toLowerCase();
    const fromStock = need > 0 ? stock.paints.filter(p => p.manufacturerId === m.id || p.manufacturer.toLowerCase() === name).slice(0, need) : [];
    return [...fromCatalog, ...fromStock].map(p => `rgb(${p.r}, ${p.g}, ${p.b})`);
  }

  // ── Tipos de tinta (rf-15) ── (allPTypes/ptypeNameById ficam lá em cima, perto de allMfrs)
  function typeSuffix(id: number | undefined): string {
    const name = id ? ptypeNameById.get(id) : undefined;
    return name ? ` · ${name}` : '';
  }
  function acrilicaId(): number | undefined {
    return allPTypes.find(pt => pt.name.toLowerCase() === 'acrílica')?.id;
  }
  function defaultPTypeId(): number | undefined {
    return acrilicaId() ?? allPTypes[0]?.id;
  }

  // O estoque anterior ao campo não tem tipo: a cópia de uma tinta do catálogo
  // herda o tipo dela, e o resto é Acrílica.
  function backfillStockPaintTypes() {
    const acrilica = acrilicaId();
    fillStockPaintTypes(p => (p.catalogId != null ? paintById(p.catalogId)?.paintTypeId || undefined : undefined) ?? acrilica);
  }

  let ptypeSearch = $state('');
  let filteredPTypes = $derived.by(() => {
    const q = foldText(ptypeSearch.trim());
    return q ? allPTypes.filter(pt => foldText(pt.name).includes(q)) : allPTypes;
  });

  async function refreshPTypes() {
    try {
      await reloadPaintTypes();
    } catch {
      // Sem rede, a lista em memória continua valendo.
    }
  }

  let ptypeFormOpen = $state(false);
  let editingPTypeId: number | null = $state(null);
  let ptypeNameInput = $state('');
  let savingPType = $state(false);

  function openNewPType() {
    closeOtherModals();
    editingPTypeId = null;
    ptypeNameInput = '';
    ptypeFormOpen = true;
  }
  function openEditPType(pt: PaintType) {
    closeOtherModals();
    editingPTypeId = pt.id;
    ptypeNameInput = pt.name;
    ptypeFormOpen = true;
  }

  function ptypeNameProblem(name: string): 'errPTypeNameRequired' | 'errPTypeNameTooLong' | null {
    if (!name) return 'errPTypeNameRequired';
    if ([...name].length > MAX_PAINT_TYPE_NAME) return 'errPTypeNameTooLong';
    return null;
  }

  async function savePType() {
    if (savingPType) return;
    const name = ptypeNameInput.trim();
    const problem = ptypeNameProblem(name);
    if (problem) {
      toast(t(problem), 'error');
      return;
    }
    savingPType = true;
    try {
      if (editingPTypeId === null) {
        await createPaintType(name);
      } else {
        await updatePaintType(editingPTypeId, name);
        toast(t('saveChanges'));
      }
      ptypeFormOpen = false;
      await refreshPTypes();
    } catch (e) {
      const status = statusOf(e);
      if (status === 409) {
        toast(t('errPTypeDup'), 'error');
      } else if (status === 400) {
        toast(t(ptypeNameProblem(name) ?? 'errPTypeNameRequired'), 'error');
      } else if (status === 404) {
        ptypeFormOpen = false;
        toast(t('errPTypeGone'), 'error');
        await refreshPTypes();
      } else {
        toast(t('errMakerNet'), 'error');
      }
    } finally {
      savingPType = false;
    }
  }

  // Tipo usado por alguma tinta não sai: o modal só explica e não chama a rota.
  let confirmPTypeDel: PaintType | null = $state(null);
  let deletingPType = $state(false);
  let ptypeDelCounts = $derived.by(() => {
    const pt = confirmPTypeDel;
    return pt ? { catalog: pt.paintCount, stock: localStockCountForType(pt.id) } : { catalog: 0, stock: 0 };
  });
  let ptypeDelBlocked = $derived(ptypeDelCounts.catalog > 0 || ptypeDelCounts.stock > 0);

  function askDeletePType(pt: PaintType) {
    closeOtherModals();
    confirmPTypeDel = pt;
  }
  async function doDeletePType() {
    if (!confirmPTypeDel || ptypeDelBlocked || deletingPType) return;
    const { id } = confirmPTypeDel;
    deletingPType = true;
    try {
      await deletePaintType(id);
      confirmPTypeDel = null;
      if (ptypeFilter === id) ptypeFilter = null;
      await refreshPTypes();
    } catch (e) {
      const status = statusOf(e);
      if (status === 404 || status === 409) {
        confirmPTypeDel = null;
        toast(t(status === 404 ? 'errPTypeGone' : 'errPTypeBusy'), 'error');
        await refreshPTypes();
      } else {
        toast(t('errMakerNet'), 'error');
      }
    } finally {
      deletingPType = false;
    }
  }

  // ── Modal único (overlay do protótipo cobre um dos conteúdos) ──
  let modalKind = $derived.by((): 'confirmPaint' | 'confirmMaker' | 'confirmPType' | 'paint' | 'maker' | 'ptype' | null => {
    if (confirmPaintDel) return 'confirmPaint';
    if (confirmMakerDel) return 'confirmMaker';
    if (confirmPTypeDel) return 'confirmPType';
    if (formOpen) return 'paint';
    if (makerFormOpen) return 'maker';
    if (ptypeFormOpen) return 'ptype';
    return null;
  });
  function closeModal() {
    if (confirmPaintDel) { confirmPaintDel = null; return; }
    if (confirmMakerDel) { confirmMakerDel = null; return; }
    if (confirmPTypeDel) { confirmPTypeDel = null; return; }
    if (formOpen) { formOpen = false; return; }
    if (makerFormOpen) { makerFormOpen = false; return; }
    if (ptypeFormOpen) { ptypeFormOpen = false; return; }
  }
</script>

<div style="display: flex; flex-direction: column; height: 100%;">
  <!-- No protótipo T4 as abas ficam coladas na marca (não há kicker/título
       nesta tela) e a contagem + "Cadastrar" vão para a direita. -->
  <Header gap={14}>
    {#snippet lead()}
      <div style="display: flex; border: 1px solid var(--color-neutral-800); border-radius: 8px; overflow: hidden; flex-shrink: 0;">
        <button
          onclick={() => (tab = 'tintas')}
          style="height: 50px; padding: 0 20px; border: none; background: {tab === 'tintas' ? 'var(--color-accent-900)' : 'transparent'}; color: {tab === 'tintas' ? 'var(--color-accent-200)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; white-space: nowrap;"
        >{t('tabPaints')}</button>
        <button
          onclick={() => (tab = 'fabricantes')}
          style="height: 50px; padding: 0 20px; border: none; border-left: 1px solid var(--color-neutral-800); background: {tab === 'fabricantes' ? 'var(--color-accent-900)' : 'transparent'}; color: {tab === 'fabricantes' ? 'var(--color-accent-200)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; white-space: nowrap;"
        >{t('makers')}</button>
        <button
          onclick={() => (tab = 'tipos')}
          style="height: 50px; padding: 0 20px; border: none; border-left: 1px solid var(--color-neutral-800); background: {tab === 'tipos' ? 'var(--color-accent-900)' : 'transparent'}; color: {tab === 'tipos' ? 'var(--color-accent-200)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; white-space: nowrap;"
        >{t('paintTypes')}</button>
      </div>
    {/snippet}
    {#snippet actions()}
      <span style="font-size: 14px; color: var(--color-neutral-500); white-space: nowrap;">{headerCount}</span>
      <button
        class="t4-hover-accent"
        onclick={() => (tab === 'tintas' ? openAdd() : tab === 'fabricantes' ? openNewMaker() : openNewPType())}
        style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 18px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
      >
        <i class="ph ph-plus" style="font-size: 18px;"></i>{tab === 'tintas' ? t('addPaintBtn') : tab === 'fabricantes' ? t('addMakerBtn') : t('addPTypeBtn')}
      </button>
    {/snippet}
  </Header>

  <div style="flex: 1; min-height: 0; overflow-y: auto;">
    <div style="max-width: 880px; margin: 0 auto; padding: 22px 24px 40px;">
      {#if tab === 'tintas'}
        <div>
          <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px;">
            <div style="display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; height: 50px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field);">
              <i class="ph ph-magnifying-glass" style="font-size: 18px; color: var(--color-neutral-500);"></i>
              <input
                type="search"
                bind:value={search}
                placeholder={t('phFindPaint')}
                aria-label={t('phFindPaint')}
                autocomplete="off"
                autocorrect="off"
                autocapitalize="off"
                spellcheck="false"
                style="flex: 1; min-width: 0; height: 46px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: 15px;"
              />
            </div>
            <button
              onclick={() => (onlyMine = !onlyMine)}
              aria-pressed={onlyMine}
              style="display: inline-flex; align-items: center; gap: 10px; height: 50px; padding: 0 14px; border: 1px solid {onlyMine ? 'var(--color-accent-700)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {onlyMine ? 'var(--color-accent-panel)' : 'transparent'}; color: {onlyMine ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
            >
              <i class="ph-bold ph-check" style="font-size: 15px;"></i>{t('onlyMineShort')}
            </button>
          </div>

          <div style="display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 20px;">
            <div style="flex: 1; min-width: 200px;">
              <Combobox
                options={allMfrs}
                value={mfrFilter}
                onchange={(id) => (mfrFilter = id)}
                placeholder={t('allMakers')}
                clearLabel={t('allMakers')}
                searchPlaceholder={t('phFindMaker')}
                emptyText={t('noMakerFound')}
                ariaLabel={t('fMaker')}
                moreText={(n) => t('comboMore', { n })}
              />
            </div>
            <div style="flex: 1; min-width: 200px;">
              <Combobox
                options={allPTypes}
                value={ptypeFilter}
                onchange={(id) => (ptypeFilter = id)}
                placeholder={t('allPTypesFilter')}
                clearLabel={t('allPTypesFilter')}
                searchPlaceholder={t('phFindPType')}
                emptyText={t('noPTypeFound')}
                ariaLabel={t('fPType')}
                moreText={(n) => t('comboMore', { n })}
              />
            </div>
          </div>

          <p style="margin: 0 0 8px; font-size: 13px; color: var(--color-neutral-500);">{listNote}</p>

          {#if mergedRows.length === 0}
            <div style="display: flex; flex-direction: column; align-items: flex-start; gap: 12px; padding: 48px 4px;">
              <i class="ph ph-drop-half" style="font-size: 34px; color: var(--color-neutral-700);"></i>
              <p style="margin: 0; font-size: 20px; font-weight: 500; color: var(--color-text);">
                {mfrFilter != null ? t('emptyTitleBrand', { brand: allMfrs.find(m => m.id === mfrFilter)?.name ?? '' }) : t('emptyTitleAll')}
              </p>
              <p style="margin: 0; font-size: 15px; color: var(--color-neutral-500); max-width: 420px; text-wrap: pretty;">{t('emptyPaintsNote', { label: t('addPaintBtn') })}</p>
            </div>
          {:else}
            <div style="display: flex; flex-direction: column; flex: 1; min-height: 300px;">
              <VirtualList rows={mergedRows} rowHeight={68}>
                {#snippet row(r: Row)}
                  <div style="display: flex; align-items: center; gap: 14px; height: 100%; padding: 10px 4px; border-bottom: 1px solid var(--color-line);">
                    <button
                      onclick={r.toggle}
                      aria-pressed={r.have}
                      style="display: inline-flex; align-items: center; gap: 10px; height: 48px; padding: 0 10px 0 4px; border: none; background: transparent; color: {r.have ? 'var(--color-accent-400)' : 'var(--color-neutral-500)'}; font-family: inherit; font-size: 13px; font-weight: 500; cursor: pointer; flex-shrink: 0;"
                    >
                      <span style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {r.have ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 4px; background: {r.have ? 'var(--color-accent)' : 'transparent'};">
                        <i class="ph-bold ph-check" style="font-size: 15px; color: {r.have ? 'var(--color-accent-100)' : 'transparent'};"></i>
                      </span>
                      <span style="width: 62px; text-align: left;">{r.have ? t('have') : t('dontHave')}</span>
                    </button>
                    <span style="width: 42px; height: 42px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: {r.hex}; flex-shrink: 0;"></span>
                    <span style="display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0;">
                      <span style="font-size: 16px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{r.name}</span>
                      <span style="font-size: 13px; color: var(--color-neutral-500);">{r.meta}</span>
                    </span>
                    <button class="t4-hover-ghost" onclick={r.edit} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('edit')}</button>
                    {#if r.del}
                      <button class="t4-hover-ghost" onclick={r.del} aria-label={t('delPaintA')} title={t('delPaintA')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer; flex-shrink: 0;">
                        <i class="ph ph-trash" style="font-size: 18px;"></i>
                      </button>
                    {/if}
                  </div>
                {/snippet}
              </VirtualList>
            </div>
          {/if}
        </div>
      {:else if tab === 'fabricantes'}
        <div style="display: flex; flex-direction: column;">
          <div style="display: flex; align-items: center; gap: 10px; height: 50px; padding: 0 14px; margin-bottom: 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field);">
            <i class="ph ph-magnifying-glass" style="font-size: 18px; color: var(--color-neutral-500);"></i>
            <input
              type="search"
              bind:value={makerSearch}
              placeholder={t('phFindMaker')}
              aria-label={t('phFindMaker')}
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              spellcheck="false"
              style="flex: 1; min-width: 0; height: 46px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: 15px;"
            />
          </div>
          {#each filteredMakers as m (m.id)}
            {@const catalogCount = m.paintCount}
            {@const haveCount = localStockCountFor(m)}
            <div style="display: flex; align-items: center; gap: 16px; min-height: 76px; padding: 12px 4px; border-bottom: 1px solid var(--color-line);">
              <span style="display: flex; gap: 3px; flex-shrink: 0;">
                {#each swatchesFor(m) as hex, i (i)}
                  <span style="width: 18px; height: 40px; border-radius: 2px; background: {hex};"></span>
                {/each}
              </span>
              <span style="display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0;">
                <span style="font-size: 17px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{m.name}</span>
                <span style="font-size: 13px; color: var(--color-neutral-500);">
                  {catalogCount === 1 ? t('makerMeta1', { m: haveCount }) : t('makerMeta', { n: catalogCount, m: haveCount })}
                </span>
              </span>
              <button class="t4-hover-ghost" onclick={() => seePaints(m.id)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('seePaints')}</button>
              <button class="t4-hover-ghost" onclick={() => openEditMaker(m)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('edit')}</button>
              <button class="t4-hover-ghost" onclick={() => askDeleteMaker(m)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('del')}</button>
            </div>
          {/each}
          {#if filteredMakers.length === 0}
            <p style="margin: 24px 4px; font-size: 15px; color: var(--color-neutral-500);">{t('noMakerFound')}</p>
          {/if}
        </div>
      {:else}
        <div style="display: flex; flex-direction: column;">
          <div style="display: flex; align-items: center; gap: 10px; height: 50px; padding: 0 14px; margin-bottom: 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field);">
            <i class="ph ph-magnifying-glass" style="font-size: 18px; color: var(--color-neutral-500);"></i>
            <input
              type="search"
              bind:value={ptypeSearch}
              placeholder={t('phFindPType')}
              aria-label={t('phFindPType')}
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              spellcheck="false"
              style="flex: 1; min-width: 0; height: 46px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: 15px;"
            />
          </div>
          {#each filteredPTypes as pt (pt.id)}
            <div style="display: flex; align-items: center; gap: 16px; min-height: 68px; padding: 12px 4px; border-bottom: 1px solid var(--color-line);">
              <span style="display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0;">
                <span style="font-size: 17px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{pt.name}</span>
                <span style="font-size: 13px; color: var(--color-neutral-500);">{t('pTypeMeta', { n: pt.paintCount, m: localStockCountForType(pt.id) })}</span>
              </span>
              <button class="t4-hover-ghost" onclick={() => openEditPType(pt)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('edit')}</button>
              <button class="t4-hover-ghost" onclick={() => askDeletePType(pt)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('del')}</button>
            </div>
          {/each}
          {#if filteredPTypes.length === 0}
            <p style="margin: 24px 4px; font-size: 15px; color: var(--color-neutral-500);">{t('noPTypeFound')}</p>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<!-- Overlay único: sempre montado, visibilidade alterna por `display` (como no
     protótipo — sc-if com hint-placeholder, nós presentes, visibilidade que
     alterna) para que cada card fique disponível assim que sua ação abrir. -->
<div class="t4-scrim" style="position: absolute; inset: 0; z-index: 70; display: {modalKind ? 'flex' : 'none'}; align-items: center; justify-content: center; padding: 40px;">
  <div role="presentation" onclick={closeModal} style="position: absolute; inset: 0; background: rgba(9, 10, 16, 0.7);"></div>

  <!-- Modal cadastro/edição de tinta: topo e rodapé fixos, só o corpo rola —
       card sem padding próprio, cada faixa cuida do seu. -->
  <div class="t4-modal" style="position: relative; width: min(560px, 100%); max-height: 100%; display: {modalKind === 'paint' ? 'flex' : 'none'}; flex-direction: column; overflow: hidden; padding: 0; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <div style="flex-shrink: 0; display: flex; align-items: center; gap: 14px; padding: 20px 24px 16px; border-bottom: 1px solid var(--color-line);">
      <span style="flex: 1; font-size: 22px; font-weight: 500; letter-spacing: -0.01em; color: var(--color-text);">{editing ? t('editPaint') : t('newPaint')}</span>
      <button class="t4-hover-ghost" onclick={closeModal} aria-label={t('ariaClose')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;">
        <i class="ph ph-x" style="font-size: 20px;"></i>
      </button>
    </div>

    <div bind:this={formBodyEl} style="flex: 1; min-height: 0; overflow-y: auto; padding: 20px 24px;">
      <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 20px;">
        <PaintBottle r={fRgb.r} g={fRgb.g} b={fRgb.b} width={52} height={87} label={fName || t('fName')} />
        <span style="display: flex; flex-direction: column; gap: 3px; min-width: 0;">
          <span style="font-size: clamp(15px, 1.6cqi, 19px); font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{fName || t('fName')}</span>
          <span style="font-size: 13px; color: var(--color-neutral-500);">{formPreviewMeta}</span>
        </span>
      </div>

      <div style="display: flex; flex-direction: column; gap: 14px;">
        <label style="display: flex; flex-direction: column; gap: 6px;">
          <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fName')}</span>
          <input type="text" bind:value={fName} placeholder="Mephiston Red" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
        </label>

        <div style="display: flex; gap: 12px;">
          <label style="display: flex; flex-direction: column; gap: 6px; flex: 1; min-width: 0;">
            <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fCode')}</span>
            <input type="text" bind:value={fCode} placeholder="70.951" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
          </label>
          <label style="display: flex; flex-direction: column; gap: 6px; width: 130px; flex-shrink: 0;">
            <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fLeft')}</span>
            <input type="text" inputmode="decimal" bind:value={fVolume} placeholder="17" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
          </label>
          <label style="display: flex; flex-direction: column; gap: 6px; width: 110px; flex-shrink: 0;">
            <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fQty')}</span>
            <input type="number" inputmode="numeric" min="1" step="1" bind:value={fQty} style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
          </label>
        </div>

        <div style="display: flex; flex-direction: column; gap: 8px;">
          <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fMaker')}</span>
          <Combobox
            options={allMfrs}
            value={fMfr === '' ? null : fMfr}
            onchange={(id) => (fMfr = id ?? '')}
            placeholder={t('pickMaker')}
            searchPlaceholder={t('phFindMaker')}
            emptyText={t('noMakerFound')}
            ariaLabel={t('fMaker')}
            moreText={(n) => t('comboMore', { n })}
          />
        </div>

        <div style="display: flex; flex-direction: column; gap: 8px;">
          <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fPType')}</span>
          <Combobox
            options={allPTypes}
            value={fPType === '' ? null : fPType}
            onchange={(id) => (fPType = id ?? '')}
            placeholder={t('pickPType')}
            searchPlaceholder={t('phFindPType')}
            emptyText={t('noPTypeFound')}
            ariaLabel={t('fPType')}
            moreText={(n) => t('comboMore', { n })}
          />
        </div>

        <div style="display: flex; flex-direction: column; gap: 8px;">
          <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fColor')}</span>
          <div style="display: flex; align-items: center; gap: 10px;">
            <input
              type="color"
              class="t4-swatch-picker"
              value={pickerHex}
              oninput={onPickColor}
              aria-label={t('fColorPick')}
              title={t('fColorPick')}
              style="width: 50px; height: 50px; border-radius: 8px; border: 1px solid var(--color-neutral-800); background: rgb({fRgb.r}, {fRgb.g}, {fRgb.b}); flex-shrink: 0; cursor: pointer;"
            />
            <input
              type="text"
              bind:value={fHex}
              maxlength="7"
              placeholder="#9A1115"
              aria-label={t('fColor')}
              autocomplete="off"
              autocapitalize="off"
              spellcheck="false"
              style="flex: 1; min-width: 0; height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;"
            />
            <!-- "Ler do pote" (câmera, US-20/E9): captura local no navegador — a foto
                 nunca sai do aparelho e nenhuma rota nova foi criada. -->
            <input
              type="file"
              accept="image/*"
              capture="environment"
              bind:this={potInputEl}
              onchange={onPotFile}
              tabindex="-1"
              aria-hidden="true"
              style="position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none;"
            />
            <button
              type="button"
              class="t4-hover-accent"
              onclick={openPotCamera}
              disabled={readingPot}
              style="display: inline-flex; align-items: center; gap: 8px; height: 50px; padding: 0 14px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
            >
              <i class="ph ph-camera" style="font-size: 18px;"></i>{t('readPot')}
            </button>
          </div>
        </div>
      </div>
    </div>

    <div style="flex-shrink: 0; padding: 16px 24px 20px; border-top: 1px solid var(--color-line); background: var(--color-modal);">
      <button
        type="button"
        onclick={() => (fHave = !fHave)}
        aria-pressed={fHave}
        style="display: flex; align-items: center; gap: 12px; width: 100%; height: 60px; padding: 0 14px; border: 1px solid {fHave ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {fHave ? 'color-mix(in srgb, var(--color-accent) 12%, transparent)' : 'transparent'}; color: var(--color-text); font-family: inherit; font-size: 15px; text-align: left; cursor: pointer;"
      >
        <span style="display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid {fHave ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 4px; flex-shrink: 0;">
          <i class="ph-bold ph-check" style="font-size: 15px; color: {fHave ? 'var(--color-accent)' : 'transparent'};"></i>
        </span>
        <span style="display: flex; flex-direction: column; gap: 2px; min-width: 0;">
          <span style="font-weight: 500;">{t('haveNow')}</span>
          <span style="font-size: 12.5px; color: var(--color-neutral-500);">{t('haveNote')}</span>
        </span>
      </button>

      <div style="display: flex; gap: 10px; margin-top: 6px;">
        <button class="t4-hover-accent" onclick={saveForm} disabled={saving} style="flex: 1; height: 58px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;">{editing ? t('saveChanges') : t('addPaintBtn')}</button>
        {#if editingId !== null}
          <button class="t4-hover-ghost" onclick={() => askDeletePaint()} style="height: 58px; padding: 0 18px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('delPaintA')}</button>
        {/if}
      </div>
      <p style="margin: 10px 0 0; font-size: 13px; color: var(--color-neutral-500); text-wrap: pretty;">{t('formNote')}</p>
    </div>
  </div>

  <!-- Modal novo fabricante -->
  <div class="t4-modal" style="{modalKind !== 'maker' ? 'display: none; ' : ''}position: relative; width: min(460px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <div style="display: flex; align-items: center; gap: 14px; margin-bottom: 18px;">
      <span style="flex: 1; font-size: 22px; font-weight: 500; letter-spacing: -0.01em; color: var(--color-text);">{editingMakerId === null ? t('newMakerT') : t('editMakerT')}</span>
      <button class="t4-hover-ghost" onclick={closeModal} aria-label={t('ariaClose')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;">
        <i class="ph ph-x" style="font-size: 20px;"></i>
      </button>
    </div>
    <label style="display: flex; flex-direction: column; gap: 6px;">
      <span style="font-size: 13px; color: var(--color-neutral-500);">{t('makerName')}</span>
      <input type="text" bind:value={makerNameInput} maxlength={MAX_MANUFACTURER_NAME} placeholder="Scale75" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
    </label>
    <button class="t4-hover-accent" onclick={saveMaker} disabled={savingMaker} style="width: 100%; height: 58px; margin-top: 18px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;">{editingMakerId === null ? t('addMakerBtn') : t('saveChanges')}</button>
    {#if editingMakerId === null}
      <p style="margin: 12px 0 0; font-size: 13px; color: var(--color-neutral-500); text-wrap: pretty;">{t('addMakerNote')}</p>
    {/if}
  </div>

  <!-- Confirmação exclusão de tinta -->
  <div class="t4-modal" role="alertdialog" aria-modal="true" style="{modalKind !== 'confirmPaint' ? 'display: none; ' : ''}position: relative; width: min(440px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <p style="margin: 0; font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; color: var(--color-text); text-wrap: pretty;">{t('delPaintT', { name: confirmPaintDel?.name ?? '' })}</p>
    <p style="margin: 10px 0 0; font-size: 15px; color: var(--color-neutral-400); text-wrap: pretty;">{confirmPaintDel && catalogOriginOf(confirmPaintDel) != null ? t('delPaintBCat') : t('delPaintB')}</p>
    <div style="display: flex; gap: 10px; margin-top: 22px;">
      <button class="t4-hover-ghost" onclick={closeModal} style="flex: 1; height: 56px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('cancel')}</button>
      <button class="t4-hover-accent" onclick={doDeletePaint} style="flex: 1; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('delPaintA')}</button>
    </div>
  </div>

  <!-- Confirmação exclusão de fabricante (RG-18/CA11): declara quantas tintas somem -->
  <div class="t4-modal" role="alertdialog" aria-modal="true" style="{modalKind !== 'confirmMaker' ? 'display: none; ' : ''}position: relative; width: min(440px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <p style="margin: 0; font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; color: var(--color-text); text-wrap: pretty;">{makerDelBlocked ? t('delMakerBlockedT', { name: confirmMakerDel?.name ?? '' }) : t('delMakerT', { name: confirmMakerDel?.name ?? '' })}</p>
    <p style="margin: 10px 0 0; font-size: 15px; color: var(--color-neutral-400); text-wrap: pretty;">
      {makerDelBlocked ? t('delMakerBlockedB', { n: makerDelCounts.catalog, m: makerDelCounts.stock }) : t('delMakerB0')}
    </p>
    <div style="display: flex; gap: 10px; margin-top: 22px;">
      <button class="t4-hover-ghost" onclick={closeModal} style="flex: 1; height: 56px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{makerDelBlocked ? t('closeBtn') : t('cancel')}</button>
      {#if !makerDelBlocked}
        <button class="t4-hover-accent" onclick={doDeleteMaker} disabled={deletingMaker} style="flex: 1; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('delMakerA')}</button>
      {/if}
    </div>
  </div>

  <!-- Modal tipo de tinta (novo/editar) -->
  <div class="t4-modal" style="{modalKind !== 'ptype' ? 'display: none; ' : ''}position: relative; width: min(460px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <div style="display: flex; align-items: center; gap: 14px; margin-bottom: 18px;">
      <span style="flex: 1; font-size: 22px; font-weight: 500; letter-spacing: -0.01em; color: var(--color-text);">{editingPTypeId === null ? t('newPTypeT') : t('editPTypeT')}</span>
      <button class="t4-hover-ghost" onclick={closeModal} aria-label={t('ariaClose')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;">
        <i class="ph ph-x" style="font-size: 20px;"></i>
      </button>
    </div>
    <label style="display: flex; flex-direction: column; gap: 6px;">
      <span style="font-size: 13px; color: var(--color-neutral-500);">{t('pTypeName')}</span>
      <input type="text" bind:value={ptypeNameInput} maxlength={MAX_PAINT_TYPE_NAME} placeholder="Wash" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
    </label>
    <button class="t4-hover-accent" onclick={savePType} disabled={savingPType} style="width: 100%; height: 58px; margin-top: 18px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;">{editingPTypeId === null ? t('addPTypeBtn') : t('saveChanges')}</button>
  </div>

  <!-- Confirmação exclusão de tipo de tinta: tipo em uso não sai -->
  <div class="t4-modal" role="alertdialog" aria-modal="true" style="{modalKind !== 'confirmPType' ? 'display: none; ' : ''}position: relative; width: min(440px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <p style="margin: 0; font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; color: var(--color-text); text-wrap: pretty;">{ptypeDelBlocked ? t('delPTypeBlockedT', { name: confirmPTypeDel?.name ?? '' }) : t('delPTypeT', { name: confirmPTypeDel?.name ?? '' })}</p>
    <p style="margin: 10px 0 0; font-size: 15px; color: var(--color-neutral-400); text-wrap: pretty;">
      {ptypeDelBlocked ? t('delPTypeBlockedB', { n: ptypeDelCounts.catalog, m: ptypeDelCounts.stock }) : t('delPTypeB0')}
    </p>
    <div style="display: flex; gap: 10px; margin-top: 22px;">
      <button class="t4-hover-ghost" onclick={closeModal} style="flex: 1; height: 56px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{ptypeDelBlocked ? t('closeBtn') : t('cancel')}</button>
      {#if !ptypeDelBlocked}
        <button class="t4-hover-accent" onclick={doDeletePType} disabled={deletingPType} style="flex: 1; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('delPTypeA')}</button>
      {/if}
    </div>
  </div>
</div>

<style>
  /* Modais entram junto com o escurecimento do fundo. Os cards ficam sempre
     montados e alternam por `display`, e a animação CSS reinicia sozinha toda
     vez que o elemento sai de `display: none` — por isso funciona sem estado
     extra. */
  .t4-scrim {
    animation: scrim-in 140ms ease both;
  }

  .t4-modal {
    animation: modal-in 180ms cubic-bezier(0.4, 0, 0.2, 1) both;
  }

  @keyframes scrim-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes modal-in {
    from {
      opacity: 0;
      transform: translateY(10px) scale(0.985);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  /* `style-hover="..."` do protótipo — reproduzido aqui por classe (rule 7). */
  .t4-hover-ghost:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }
  .t4-hover-accent:hover {
    background: var(--color-accent-hover);
  }

  /* O seletor nativo desenha a cor num miolo com margem própria; sem isto o
     quadrado de 50px mostra uma moldura clara em volta da cor. */
  .t4-swatch-picker {
    appearance: none;
    -webkit-appearance: none;
    padding: 0;
    overflow: hidden;
  }
  .t4-swatch-picker::-webkit-color-swatch-wrapper {
    padding: 0;
  }
  .t4-swatch-picker::-webkit-color-swatch {
    border: none;
    border-radius: 7px;
  }
  .t4-swatch-picker::-moz-color-swatch {
    border: none;
    border-radius: 7px;
  }
</style>
