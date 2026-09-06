<script lang="ts">
  // T4 — Minhas tintas (rf-04). Consolida CatalogoView + StockManager numa lista
  // única de tintas (posse alternável linha a linha) + aba Fabricantes, fiel ao
  // protótipo Nocturne (tests/fixtures/mockup/t4-tintas.html — D-001, Epic D).
  //
  // PENDÊNCIA DECLARADA (spec rf-04, seção Dependências #3 / risco R1
  // herdado de docs/mesclaai-userstory.md): esta apuração NÃO localizou rota
  // de escrita (POST/PATCH/DELETE) para tinta/fabricante em api/httpapi. O
  // catálogo servido (GET /manufacturers, GET /paints) é só-leitura aqui.
  // Por isso:
  //   - "Cadastrar/Editar/Excluir tinta" grava em services/stock.svelte.ts
  //     (localStorage, MESMO mecanismo já usado por T1 "Só o que eu tenho" e
  //     pela prioridade de estoque) — cadastro real no catálogo do servidor
  //     fica bloqueado até confirmação do contrato de escrita.
  //   - O toggle "tenho/não tenho" numa linha da lista escreve em
  //     stock.svelte.ts (por código+fabricante — RG-17), nunca no paint do
  //     servidor.
  //   - "Novo/Excluir fabricante" opera sobre uma lista LOCAL de fabricantes
  //     customizados (localStorage) somada à lista do servidor — excluir um
  //     fabricante do SERVIDOR não é possível sem a rota; "Excluir" aqui
  //     remove as tintas do ESTOQUE local daquele fabricante (confirmação
  //     declara exatamente esse efeito, RG-18/CA11).
  // Não foi inventada nenhuma rota HTTP nova.
  //
  // D-001 (fidelidade ao protótipo):
  //   - O protótipo funde a antiga lista "tenho" + catálogo virtualizado numa
  //     única lista de linhas (toggle tenho/não-tenho à esquerda, "Editar" à
  //     direita). "Editar" numa linha que ainda não é minha abre o mesmo modal
  //     de cadastro pré-preenchido com os dados do catálogo — capacidade nova,
  //     sem regressão (antes essas linhas só tinham o toggle).
  //   - O filtro "só as que eu tenho" (onlyMine) é novo nesta tela — mesmo
  //     rótulo/mecanismo já usado em T1.
  //   - O detalhe de tinta com atalho "gerar mistura equivalente"
  //     (PaintDetailSheet) NÃO existe no protótipo aprovado desta tela —
  //     removido daqui; cada linha só alterna posse ou abre o formulário.
  //   - "Ler do pote" (câmera, US-20/E9) segue fora desta rodada (Could) —
  //     botão presente, sem ação (nenhuma rota/captura foi inventada).
  //   - Os modais (cadastro/edição, novo fabricante, confirmações) ficam
  //     sempre montados no DOM e alternam via `display:none` em vez de
  //     `{#if}` — mesma técnica do protótipo (sc-if com hint-placeholder,
  //     nós sempre presentes, visibilidade que alterna).
  import Header from '../components/Header.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import VirtualList from '../components/VirtualList.svelte';
  import { allManufacturers, allPaints, hexOf, searchPaints, type Paint } from '../services/catalog';
  import { stock, sortedStock, addStockPaint, updateStockPaint, removeStockPaint } from '../services/stock.svelte';
  import type { StockPaint } from '../services/engine';
  import { appState } from '../appState.svelte';
  import { t } from '../i18n.svelte';
  import { toast } from '../toast.svelte';

  // ── Fabricantes customizados (locais — ver nota no topo do arquivo) ──
  interface CustomMfr { id: number; name: string }
  const CUSTOM_KEY = 'mescla.customMfrs.v1';
  function loadCustom(): CustomMfr[] {
    try {
      const raw = localStorage.getItem(CUSTOM_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  }
  let customMfrs: CustomMfr[] = $state(loadCustom());
  let customSeq = $state(-1);
  function persistCustom() {
    try { localStorage.setItem(CUSTOM_KEY, JSON.stringify(customMfrs)); } catch { /* indisponível */ }
  }

  type Tab = 'tintas' | 'fabricantes';
  let tab: Tab = $state('tintas');

  let search = $state('');
  let mfrFilter: number | null = $state(null);
  let onlyMine = $state(false);

  $effect(() => {
    if (appState.pendingCatalogMfrId != null) {
      mfrFilter = appState.pendingCatalogMfrId;
      appState.pendingCatalogMfrId = null;
      tab = 'tintas';
    }
  });

  let allMfrs = $derived([...allManufacturers().map(m => ({ id: m.id, name: m.name, custom: false })), ...customMfrs.map(m => ({ id: m.id, name: m.name, custom: true }))]);

  function stockCountFor(mfrName: string): number {
    return stock.paints.filter(p => p.manufacturer === mfrName).length;
  }

  function stockKeyOf(manufacturer: string, code: string): string {
    return `${manufacturer}|${code}`;
  }
  let stockKeys = $derived(new Set(stock.paints.filter(p => p.code !== '').map(p => stockKeyOf(p.manufacturer, p.code))));

  // ── Aba Tintas ──
  let filteredCatalog = $derived.by(() => {
    const base = search.trim() ? searchPaints(search, { manufacturerId: mfrFilter ?? undefined, limit: 100000 }) : allPaints().filter(p => mfrFilter == null || p.manufacturerId === mfrFilter);
    return base;
  });

  let filteredStock = $derived(
    sortedStock().filter(p => {
      if (mfrFilter != null) {
        const mfrName = allMfrs.find(m => m.id === mfrFilter)?.name;
        if (p.manufacturer !== mfrName) return false;
      }
      if (!search.trim()) return true;
      const q = search.toLowerCase();
      return p.name.toLowerCase().includes(q) || p.code.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q);
    })
  );

  function toggleHave(p: Paint) {
    const key = stockKeyOf(p.manufacturer, p.code);
    if (stockKeys.has(key)) {
      const sp = stock.paints.find(s => s.manufacturer === p.manufacturer && s.code === p.code);
      if (sp) removeStockPaint(sp.id);
    } else {
      addStockPaint({
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
  }

  let mergedRows = $derived.by(() => {
    const rows: Row[] = filteredStock.map(p => ({
      key: `s${p.id}`,
      have: true,
      name: p.name,
      meta: `${p.code || '—'} · ${p.manufacturer}${p.volume ? ` · ${p.volume}` : ''}`,
      hex: rgbToHex(p.r, p.g, p.b),
      toggle: () => removeStockPaint(p.id),
      edit: () => openEdit(p),
    }));
    if (!onlyMine) {
      for (const p of filteredCatalog) {
        if (stockKeys.has(stockKeyOf(p.manufacturer, p.code))) continue;
        rows.push({
          key: `c${p.id}`,
          have: false,
          name: p.name,
          meta: `${p.code} · ${p.manufacturer}`,
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
      (mfrFilter != null ? t('inMaker', { brand: allMfrs.find(m => m.id === mfrFilter)?.name ?? '' }) : t('inAll', { n: allMfrs.length }))
  );

  let headerCount = $derived(t('estanteCount', { a: filteredStock.length, b: filteredCatalog.length, c: allMfrs.length }));

  // ── Modal cadastro/edição (tinta) ──
  let formOpen = $state(false);
  let editingId: number | null = $state(null);
  let fMfr: number | '' = $state('');
  let fName = $state('');
  let fCode = $state('');
  let fHex = $state('#8a8a8a');
  let fVolume = $state('');
  // Toggle visual "Tenho este pote agora" (fiel ao mockup) — o estoque local
  // não tem coluna `have` própria: estar na lista já É "tenho" (RG-17). Sem
  // essa coluna, desmarcar aqui não muda o que saveForm() grava; é limitação
  // declarada, não comportamento fingido.
  let fHave = $state(true);
  let saving = $state(false);

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
  let fMfrName = $derived(allMfrs.find(m => m.id === fMfr)?.name ?? '');
  let formPreviewMeta = $derived(`${fCode || '—'} · ${fMfrName}${fVolume ? ` · ${fVolume}` : ''}`);

  function openAdd() {
    editingId = null;
    fMfr = allMfrs[0]?.id ?? '';
    fName = '';
    fCode = '';
    fHex = '#8a8a8a';
    fVolume = '';
    fHave = true;
    formOpen = true;
  }

  function openEdit(p: StockPaint) {
    editingId = p.id;
    const m = allMfrs.find(x => x.name === p.manufacturer);
    fMfr = m?.id ?? '';
    fName = p.name;
    fCode = p.code;
    fHex = rgbToHex(p.r, p.g, p.b);
    fVolume = p.volume;
    fHave = true;
    formOpen = true;
  }

  function openEditCatalog(p: Paint) {
    editingId = null;
    fMfr = p.manufacturerId;
    fName = p.name;
    fCode = p.code;
    fHex = rgbToHex(p.r, p.g, p.b);
    fVolume = '';
    fHave = true;
    formOpen = true;
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
    saving = true;
    const { r, g, b } = hexToRgb(fHex);
    const mfr = allMfrs.find(m => m.id === Number(fMfr));
    const base = {
      manufacturerId: Number(fMfr),
      manufacturer: mfr?.name ?? '',
      name: fName.trim(),
      code: fCode.trim(),
      r, g, b,
      volume: fVolume.trim(),
      notes: '',
    };
    if (editingId === null) {
      addStockPaint(base);
    } else {
      updateStockPaint({ ...base, id: editingId });
    }
    saving = false;
    formOpen = false;
    toast(t('saveChanges'));
  }

  // ── Confirmação de exclusão (RG-18/CAN1) ──
  let confirmPaintDel: StockPaint | null = $state(null);
  let confirmMakerDel: { id: number; name: string } | null = $state(null);

  function askDeletePaint() {
    if (editingId == null) return;
    const p = stock.paints.find(x => x.id === editingId);
    if (p) confirmPaintDel = p;
  }
  function doDeletePaint() {
    if (!confirmPaintDel) return;
    removeStockPaint(confirmPaintDel.id);
    confirmPaintDel = null;
    formOpen = false;
  }

  function askDeleteMaker(id: number, name: string) {
    confirmMakerDel = { id, name };
  }
  function doDeleteMaker() {
    if (!confirmMakerDel) return;
    const { id, name } = confirmMakerDel;
    for (const p of [...stock.paints].filter(sp => sp.manufacturer === name)) removeStockPaint(p.id);
    customMfrs = customMfrs.filter(m => m.id !== id);
    persistCustom();
    confirmMakerDel = null;
  }

  // ── Novo fabricante (local) ──
  let makerFormOpen = $state(false);
  let newMakerName = $state('');
  function saveNewMaker() {
    if (!newMakerName.trim()) return;
    const id = customSeq--;
    customMfrs = [...customMfrs, { id, name: newMakerName.trim() }];
    persistCustom();
    newMakerName = '';
    makerFormOpen = false;
    mfrFilter = id;
    tab = 'tintas';
  }

  function seePaints(mfrId: number) {
    mfrFilter = mfrId;
    tab = 'tintas';
  }

  function swatchesFor(mfrName: string, isCustom: boolean): string[] {
    const fromCatalog = isCustom ? [] : allPaints().filter(p => p.manufacturer === mfrName).slice(0, 5);
    const need = 5 - fromCatalog.length;
    const fromStock = need > 0 ? stock.paints.filter(p => p.manufacturer === mfrName).slice(0, need) : [];
    return [...fromCatalog, ...fromStock].map(p => `rgb(${p.r}, ${p.g}, ${p.b})`);
  }

  // ── Modal único (overlay do protótipo cobre um dos 4 conteúdos) ──
  let modalKind = $derived.by((): 'confirmPaint' | 'confirmMaker' | 'paint' | 'maker' | null => {
    if (confirmPaintDel) return 'confirmPaint';
    if (confirmMakerDel) return 'confirmMaker';
    if (formOpen) return 'paint';
    if (makerFormOpen) return 'maker';
    return null;
  });
  function closeModal() {
    if (confirmPaintDel) { confirmPaintDel = null; return; }
    if (confirmMakerDel) { confirmMakerDel = null; return; }
    if (formOpen) { formOpen = false; return; }
    if (makerFormOpen) { makerFormOpen = false; return; }
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
      </div>
    {/snippet}
    {#snippet actions()}
      <span style="font-size: 14px; color: var(--color-neutral-500); white-space: nowrap;">{headerCount}</span>
      <button
        class="t4-hover-accent"
        onclick={() => (tab === 'tintas' ? openAdd() : (makerFormOpen = true))}
        style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 18px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
      >
        <i class="ph ph-plus" style="font-size: 18px;"></i>{tab === 'tintas' ? t('addPaintBtn') : t('addMakerBtn')}
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

          <div style="display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 20px;">
            <button
              class="t4-hover-border"
              onclick={() => (mfrFilter = null)}
              style="height: 44px; padding: 0 14px; border: 1px solid {mfrFilter === null ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 999px; background: {mfrFilter === null ? 'var(--color-accent)' : 'var(--color-surface)'}; color: {mfrFilter === null ? 'var(--color-accent-100)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer; white-space: nowrap;"
            >{t('allMakers')}</button>
            {#each allMfrs as m (m.id)}
              <button
                class="t4-hover-border"
                onclick={() => (mfrFilter = m.id)}
                style="height: 44px; padding: 0 14px; border: 1px solid {mfrFilter === m.id ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 999px; background: {mfrFilter === m.id ? 'var(--color-accent)' : 'var(--color-surface)'}; color: {mfrFilter === m.id ? 'var(--color-accent-100)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer; white-space: nowrap;"
              >{m.name}</button>
            {/each}
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
                  </div>
                {/snippet}
              </VirtualList>
            </div>
          {/if}
        </div>
      {:else}
        <div style="display: flex; flex-direction: column;">
          {#each allMfrs as m (m.id)}
            {@const catalogCount = m.custom ? 0 : (allManufacturers().find(x => x.id === m.id)?.paintCount ?? 0)}
            {@const haveCount = stockCountFor(m.name)}
            <div style="display: flex; align-items: center; gap: 16px; min-height: 76px; padding: 12px 4px; border-bottom: 1px solid var(--color-line);">
              <span style="display: flex; gap: 3px; flex-shrink: 0;">
                {#each swatchesFor(m.name, m.custom) as hex, i (i)}
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
              <button class="t4-hover-ghost" onclick={() => askDeleteMaker(m.id, m.name)} style="height: 48px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('del')}</button>
            </div>
          {/each}
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

  <!-- Modal cadastro/edição de tinta -->
  <div class="t4-modal" style="{modalKind !== 'paint' ? 'display: none; ' : ''}position: relative; width: min(560px, 100%); max-height: 100%; overflow-y: auto; padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <div style="display: flex; align-items: center; gap: 14px; margin-bottom: 20px;">
      <span style="flex: 1; font-size: 22px; font-weight: 500; letter-spacing: -0.01em; color: var(--color-text);">{editingId === null ? t('newPaint') : t('editPaint')}</span>
      <button class="t4-hover-ghost" onclick={closeModal} aria-label={t('ariaClose')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;">
        <i class="ph ph-x" style="font-size: 20px;"></i>
      </button>
    </div>

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
          <input type="text" bind:value={fVolume} placeholder="17 ml" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
        </label>
      </div>

      <div style="display: flex; flex-direction: column; gap: 8px;">
        <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fMaker')}</span>
        <div style="display: flex; flex-wrap: wrap; gap: 8px;">
          {#each allMfrs as m (m.id)}
            <button
              type="button"
              class="t4-hover-border"
              onclick={() => (fMfr = m.id)}
              style="height: 44px; padding: 0 14px; border: 1px solid {fMfr === m.id ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 999px; background: {fMfr === m.id ? 'var(--color-accent)' : 'var(--color-surface)'}; color: {fMfr === m.id ? 'var(--color-accent-100)' : 'var(--color-neutral-300)'}; font-family: inherit; font-size: 13.5px; font-weight: 500; cursor: pointer; white-space: nowrap;"
            >{m.name}</button>
          {/each}
        </div>
      </div>

      <div style="display: flex; flex-direction: column; gap: 8px;">
        <span style="font-size: 13px; color: var(--color-neutral-500);">{t('fColor')}</span>
        <div style="display: flex; align-items: center; gap: 10px;">
          <span style="width: 50px; height: 50px; border-radius: 8px; border: 1px solid var(--color-neutral-800); background: rgb({fRgb.r}, {fRgb.g}, {fRgb.b}); flex-shrink: 0;"></span>
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
          <!-- "Ler do pote" (câmera, US-20/E9) — fora desta rodada (Could): botão fiel ao
               protótipo, sem captura/rota nova. -->
          <button
            type="button"
            class="t4-hover-accent"
            style="display: inline-flex; align-items: center; gap: 8px; height: 50px; padding: 0 14px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 14px; font-weight: 500; cursor: pointer; flex-shrink: 0; white-space: nowrap;"
          >
            <i class="ph ph-camera" style="font-size: 18px;"></i>{t('readPot')}
          </button>
        </div>
      </div>

      <button
        type="button"
        onclick={() => (fHave = !fHave)}
        aria-pressed={fHave}
        style="display: flex; align-items: center; gap: 12px; height: 60px; padding: 0 14px; border: 1px solid {fHave ? 'var(--color-accent)' : 'var(--color-neutral-800)'}; border-radius: 8px; background: {fHave ? 'color-mix(in srgb, var(--color-accent) 12%, transparent)' : 'transparent'}; color: var(--color-text); font-family: inherit; font-size: 15px; text-align: left; cursor: pointer;"
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
        <button class="t4-hover-accent" onclick={saveForm} disabled={saving} style="flex: 1; height: 58px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;">{editingId === null ? t('addPaintBtn') : t('saveChanges')}</button>
        {#if editingId !== null}
          <button class="t4-hover-ghost" onclick={askDeletePaint} style="height: 58px; padding: 0 18px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer; flex-shrink: 0;">{t('delPaintA')}</button>
        {/if}
      </div>
      <p style="margin: 2px 0 0; font-size: 13px; color: var(--color-neutral-500); text-wrap: pretty;">{t('formNote')}</p>
    </div>
  </div>

  <!-- Modal novo fabricante -->
  <div class="t4-modal" style="{modalKind !== 'maker' ? 'display: none; ' : ''}position: relative; width: min(460px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <div style="display: flex; align-items: center; gap: 14px; margin-bottom: 18px;">
      <span style="flex: 1; font-size: 22px; font-weight: 500; letter-spacing: -0.01em; color: var(--color-text);">{t('newMakerT')}</span>
      <button class="t4-hover-ghost" onclick={closeModal} aria-label={t('ariaClose')} style="display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;">
        <i class="ph ph-x" style="font-size: 20px;"></i>
      </button>
    </div>
    <label style="display: flex; flex-direction: column; gap: 6px;">
      <span style="font-size: 13px; color: var(--color-neutral-500);">{t('makerName')}</span>
      <input type="text" bind:value={newMakerName} placeholder="Scale75" style="height: 50px; padding: 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: var(--color-text); font-family: inherit; font-size: 15px; outline: none;" />
    </label>
    <button class="t4-hover-accent" onclick={saveNewMaker} style="width: 100%; height: 58px; margin-top: 18px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 16px; font-weight: 500; cursor: pointer;">{t('addMakerBtn')}</button>
    <p style="margin: 12px 0 0; font-size: 13px; color: var(--color-neutral-500); text-wrap: pretty;">{t('addMakerNote')}</p>
  </div>

  <!-- Confirmação exclusão de tinta -->
  <div class="t4-modal" style="{modalKind !== 'confirmPaint' ? 'display: none; ' : ''}position: relative; width: min(440px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <p style="margin: 0; font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; color: var(--color-text); text-wrap: pretty;">{t('delPaintT', { name: confirmPaintDel?.name ?? '' })}</p>
    <p style="margin: 10px 0 0; font-size: 15px; color: var(--color-neutral-400); text-wrap: pretty;">{t('delPaintB')}</p>
    <div style="display: flex; gap: 10px; margin-top: 22px;">
      <button class="t4-hover-ghost" onclick={closeModal} style="flex: 1; height: 56px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('cancel')}</button>
      <button class="t4-hover-accent" onclick={doDeletePaint} style="flex: 1; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('delPaintA')}</button>
    </div>
  </div>

  <!-- Confirmação exclusão de fabricante (RG-18/CA11): declara quantas tintas somem -->
  <div class="t4-modal" style="{modalKind !== 'confirmMaker' ? 'display: none; ' : ''}position: relative; width: min(440px, 100%); padding: 24px; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);">
    <p style="margin: 0; font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; color: var(--color-text); text-wrap: pretty;">{t('delMakerT', { name: confirmMakerDel?.name ?? '' })}</p>
    <p style="margin: 10px 0 0; font-size: 15px; color: var(--color-neutral-400); text-wrap: pretty;">
      {stockCountFor(confirmMakerDel?.name ?? '') > 0 ? t('delMakerB', { n: stockCountFor(confirmMakerDel?.name ?? '') }) : t('delMakerB0')}
    </p>
    <div style="display: flex; gap: 10px; margin-top: 22px;">
      <button class="t4-hover-ghost" onclick={closeModal} style="flex: 1; height: 56px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('cancel')}</button>
      <button class="t4-hover-accent" onclick={doDeleteMaker} style="flex: 1; height: 56px; border: 1px solid var(--color-accent); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;">{t('delMakerA')}</button>
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
  .t4-hover-border:hover {
    border-color: var(--color-accent-700);
  }
  .t4-hover-ghost:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }
  .t4-hover-accent:hover {
    background: var(--color-accent-hover);
  }
</style>
