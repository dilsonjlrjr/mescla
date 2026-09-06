<script lang="ts">
  // Meu estoque (Tintômetro) — master-detail: grid de cards à esquerda,
  // painel da tinta selecionada à direita. CRUD + importação/exportação CSV.
  // As tintas daqui alimentam a opção "priorizar meu estoque" na Equivalência.
  import { onMount } from 'svelte';
  import PaintBottle from './PaintBottle.svelte';
  import Icon from './Icon.svelte';
  import { toast } from '../toast.svelte';
  import { recipesWithIngredient } from '../recipes.svelte';
  import { hexOf } from '../ui';
  import { t } from '../i18n.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type { UserPaintDTO, CSVImportResultDTO, ManufacturerDTO, PaintDTO } from '../../../bindings/paint-match-ai/models';

  interface Props {
    // vindo da busca global: "Adicionar {tinta} ao meu estoque"
    prefillPaintId?: number | null;
    // T4: duas abas no mesmo componente (rf-04) — Tintas (estoque, CRUD real
    // via UserPaint*) e Fabricantes (leitura; cadastro/exclusão de fabricante
    // não têm rota de escrita exposta pelo Wails nesta apuração — ver Não faz
    // da spec e CAN2/R1; a aba avisa a lacuna em vez de fingir persistir).
    initialTab?: 'tintas' | 'fabricantes';
  }

  let { prefillPaintId = null, initialTab = 'tintas' }: Props = $props();

  // $state captura só o valor de montagem — de propósito: o App.svelte troca
  // a key da view (navSeq) a cada handleNavigate, então uma navegação nova
  // pra 'manufacturers' remonta este componente do zero com o initialTab novo.
  // svelte-ignore state_referenced_locally
  let activeTab: 'tintas' | 'fabricantes' = $state(initialTab);

  let paints: UserPaintDTO[] = $state([]);
  let manufacturers: { id: number; name: string }[] = $state([]);
  let mfrFull: ManufacturerDTO[] = $state([]);
  let catalogPaints: PaintDTO[] = $state([]);
  let chipsByBrand: Map<string, PaintDTO[]> = $state(new Map());
  let linesByBrand: Map<string, string[]> = $state(new Map());
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedBrands: string[] = $state([]);
  let selectedId: number | null = $state(null);
  let confirmDeletePaint: UserPaintDTO | null = $state(null);
  let confirmDeleteMaker: ManufacturerDTO | null = $state(null);

  // Form (add/editar)
  let formOpen = $state(false);
  let editingId: number | null = $state(null);
  let fMfr: number | '' = $state('');
  let fName = $state('');
  let fCode = $state('');
  let fHex = $state('#8a8a8a');
  let fVolume = $state('');
  let fNotes = $state('');
  // Toggle visual "Tenho este pote agora" (fiel ao mockup) — o backend não
  // tem coluna `have` própria: estar em user_paints já É "tenho" (RG-17). Sem
  // essa coluna, desmarcar aqui não muda o que save() grava (fica sempre
  // adicionado à estante); é limitação declarada, não comportamento fingido.
  let fHave = $state(true);
  let saving = $state(false);

  // Importação CSV
  let importOpen = $state(false);
  let importResult: CSVImportResultDTO | null = $state(null);
  let fileInput: HTMLInputElement;

  onMount(async () => {
    await load();
    if (prefillPaintId) {
      try {
        const p = await PaintService.GetPaintByID(prefillPaintId);
        const mfr = manufacturers.find(m => m.name === p.manufacturer);
        editingId = null;
        fMfr = mfr?.id ?? manufacturers[0]?.id ?? '';
        fName = p.name;
        fCode = p.code;
        fHex = hexOf(p.r, p.g, p.b).toLowerCase();
        fVolume = p.volume || '';
        fNotes = '';
        formOpen = true;
      } catch (e) {
        console.error('Erro pré-preenchendo tinta:', e);
      }
    }
  });

  async function load() {
    loading = true;
    try {
      const [stock, mfrs, catalog] = await Promise.all([
        PaintService.GetUserPaints(),
        PaintService.GetManufacturers(),
        PaintService.GetAllPaints(),
      ]);
      paints = stock || [];
      mfrFull = mfrs || [];
      manufacturers = mfrFull.map(m => ({ id: m.id, name: m.name }));
      catalogPaints = catalog || [];
      if (selectedId === null || !paints.some(p => p.id === selectedId)) {
        selectedId = paints[0]?.id ?? null;
      }
      buildBrandChips();
    } catch (e) {
      console.error('Erro carregando estoque:', e);
      toast('Não consegui carregar seu estoque.', 'error');
    } finally {
      loading = false;
    }
  }

  // Amostras/linhas por fabricante (aba Fabricantes) — derivadas do catálogo,
  // não da posse (mesma lógica que ManufacturersView já usava).
  function buildBrandChips() {
    const chips = new Map<string, PaintDTO[]>();
    const lines = new Map<string, Set<string>>();
    for (const p of catalogPaints) {
      const arr = chips.get(p.manufacturer) ?? [];
      if (arr.length < 5) { arr.push(p); chips.set(p.manufacturer, arr); }
      if (p.productLine) {
        const ls = lines.get(p.manufacturer) ?? new Set();
        ls.add(p.productLine);
        lines.set(p.manufacturer, ls);
      }
    }
    chipsByBrand = chips;
    linesByBrand = new Map([...lines].map(([k, v]) => [k, [...v].slice(0, 4)]));
  }

  function ownedCountFor(mfrName: string): number {
    return paints.filter(p => p.manufacturer === mfrName).length;
  }

  function metaOf(m: ManufacturerDTO): string {
    const lines = linesByBrand.get(m.name) ?? [];
    const parts = [m.country, lines.join(', ')].filter(Boolean);
    return parts.join(' · ');
  }

  function viewMakerPaints(m: ManufacturerDTO) {
    activeTab = 'tintas';
    selectedBrands = [m.name];
  }

  function addMaker() {
    // Sem rota de escrita de fabricante exposta pelo Wails (Não faz da spec,
    // R1/CAN2 herdado) — avisa a lacuna em vez de fingir persistir.
    toast(t('mfrWriteGapAdd'), 'error');
  }

  function askDeleteMaker(m: ManufacturerDTO) {
    confirmDeleteMaker = m;
  }

  function confirmMakerDelete() {
    confirmDeleteMaker = null;
    toast(t('mfrWriteGapDel'), 'error');
  }

  function cadastrarLabel(): string {
    return activeTab === 'tintas' ? t('addPaintBtn') : t('addMakerBtn');
  }

  function onCadastrar() {
    if (activeTab === 'tintas') openAdd();
    else addMaker();
  }

  let filtered = $derived(
    paints.filter(p => {
      if (selectedBrands.length > 0 && !selectedBrands.includes(p.manufacturer)) return false;
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        p.name.toLowerCase().includes(q) ||
        p.code.toLowerCase().includes(q) ||
        p.manufacturer.toLowerCase().includes(q)
      );
    })
  );

  let ownedBrands = $derived([...new Set(paints.map(p => p.manufacturer))].sort());

  let selected = $derived(paints.find(p => p.id === selectedId) ?? null);

  let selectedRecipeCount = $derived(
    selected ? recipesWithIngredient(selected.manufacturer, selected.code, selected.name) : 0
  );

  function hexToRgb(hex: string): { r: number; g: number; b: number } {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
    if (!m) return { r: 138, g: 138, b: 138 };
    const n = parseInt(m[1], 16);
    return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
  }

  function openAdd() {
    editingId = null;
    fMfr = manufacturers[0]?.id ?? '';
    fName = '';
    fCode = '';
    fHex = '#8a8a8a';
    fVolume = '';
    fNotes = '';
    fHave = true;
    formOpen = true;
  }

  function openEdit(p: UserPaintDTO) {
    editingId = p.id;
    fMfr = p.manufacturerId;
    fName = p.name;
    fCode = p.code;
    fHex = hexOf(p.r, p.g, p.b).toLowerCase();
    fVolume = p.volume;
    fNotes = p.notes;
    fHave = true;
    formOpen = true;
  }

  async function save() {
    if (!fName.trim()) {
      toast('Dê um nome à tinta.', 'error');
      return;
    }
    if (fMfr === '') {
      toast('Escolha o fabricante.', 'error');
      return;
    }
    saving = true;
    const { r, g, b } = hexToRgb(fHex);
    const dto: UserPaintDTO = {
      id: editingId ?? 0,
      manufacturerId: Number(fMfr),
      manufacturer: '',
      name: fName.trim(),
      code: fCode.trim(),
      r,
      g,
      b,
      volume: fVolume.trim(),
      notes: fNotes.trim(),
    };
    try {
      if (editingId === null) {
        const created = await PaintService.AddUserPaint(dto);
        toast('Tinta adicionada ao estoque.');
        selectedId = created?.id ?? selectedId;
      } else {
        await PaintService.UpdateUserPaint(dto);
        toast('Tinta atualizada.');
      }
      formOpen = false;
      await load();
    } catch (e) {
      console.error('Erro salvando tinta:', e);
      toast(String(e), 'error');
    } finally {
      saving = false;
    }
  }

  function askRemove(p: UserPaintDTO) {
    confirmDeletePaint = p;
  }

  async function confirmRemove() {
    const p = confirmDeletePaint;
    confirmDeletePaint = null;
    if (!p) return;
    try {
      await PaintService.DeleteUserPaint(p.id);
      toast('Tinta removida.');
      if (selectedId === p.id) selectedId = null;
      await load();
    } catch (e) {
      console.error('Erro removendo tinta:', e);
      toast(String(e), 'error');
    }
  }

  function downloadCsv(csv: string, filename: string) {
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  async function downloadTemplate() {
    try {
      downloadCsv(await PaintService.UserPaintCSVTemplate(), 'modelo-estoque-mescla.csv');
    } catch (e) {
      console.error('Erro gerando modelo:', e);
      toast('Não consegui gerar o modelo.', 'error');
    }
  }

  async function exportCsv() {
    if (paints.length === 0) return;
    try {
      downloadCsv(await PaintService.ExportUserPaintsCSV(), 'meu-estoque-mescla.csv');
      toast(`${paints.length} ${paints.length === 1 ? 'tinta exportada' : 'tintas exportadas'}.`);
    } catch (e) {
      console.error('Erro exportando estoque:', e);
      toast('Não consegui exportar o estoque.', 'error');
    }
  }

  function pickFile() {
    importResult = null;
    fileInput?.click();
  }

  async function onFileChosen(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = ''; // permite reimportar o mesmo arquivo
    if (!file) return;
    try {
      const text = await file.text();
      const result = await PaintService.ImportUserPaintsCSV(text);
      importResult = result;
      importOpen = true;
      if (result.imported > 0) await load();
    } catch (err) {
      console.error('Erro importando CSV:', err);
      toast(String(err), 'error');
    }
  }
</script>

<div class="page-container view-shell">
  <!-- Header -->
  <div class="stock-head animate-rise view-fixed">
    <div>
      <h1 class="page-title">{t('navTintas')}</h1>
      <p class="stock-count font-mono">
        {t('estanteCount', { a: paints.length, b: paints.length, c: ownedBrands.length })}
      </p>
    </div>
    <div class="stock-cta">
      {#if activeTab === 'tintas'}
        <button class="text-link" onclick={downloadTemplate}>Baixar modelo</button>
        <button class="text-link" onclick={exportCsv} disabled={paints.length === 0}>Exportar CSV</button>
        <button class="pill-light" onclick={pickFile}>Importar CSV</button>
      {/if}
      <button class="pill-dark" onclick={onCadastrar}>{cadastrarLabel()}</button>
    </div>
  </div>

  <div class="tab-bar view-fixed">
    <button class="tab-btn" class:active={activeTab === 'tintas'} onclick={() => (activeTab = 'tintas')}>{t('tabPaints')}</button>
    <button class="tab-btn" class:active={activeTab === 'fabricantes'} onclick={() => (activeTab = 'fabricantes')}>{t('makers')}</button>
  </div>

  <input
    bind:this={fileInput}
    type="file"
    accept=".csv,text/csv"
    style="display: none;"
    onchange={onFileChosen}
  />

  {#if activeTab === 'fabricantes'}
    <div class="view-scroll">
      <div class="mfr-list animate-rise">
        {#each mfrFull as mfr (mfr.id)}
          <div class="mfr-row">
            <span class="mfr-badge font-display">
              <span class="mfr-initial">{mfr.name.charAt(0).toUpperCase()}</span>
            </span>
            <div class="mfr-id">
              <span class="mfr-name font-display">{mfr.name}</span>
              <span class="mfr-meta font-mono">{metaOf(mfr)}</span>
            </div>
            <div class="mfr-chips" aria-hidden="true">
              {#each chipsByBrand.get(mfr.name) ?? [] as chip (chip.id)}
                <span class="mfr-chip" style="background: rgb({chip.r}, {chip.g}, {chip.b});"></span>
              {/each}
            </div>
            <div class="mfr-count">
              <span class="mfr-count-n font-mono">{mfr.paintCount.toLocaleString('pt-BR')}</span>
              <span class="mfr-count-label">{t('tabPaints')}</span>
              <span class="mfr-count-owned font-mono">{ownedCountFor(mfr.name)} {t('haveOnShelf')}</span>
            </div>
            <button class="mfr-open" onclick={() => viewMakerPaints(mfr)}>{t('seePaints')} ›</button>
            <button class="mfr-del" onclick={() => askDeleteMaker(mfr)} aria-label={t('del')}>{t('del')}</button>
          </div>
        {/each}
      </div>
    </div>
  {:else}
  <div class="view-scroll">
    {#if loading}
    <div class="stock-body">
      <div class="stock-grid">
        {#each Array(8) as _, i (i)}
          <div>
            <div class="skeleton" style="aspect-ratio: 4/3; margin-bottom: 10px;"></div>
            <div class="skeleton" style="height: 12px; width: 70%;"></div>
          </div>
        {/each}
      </div>
    </div>
    {:else if paints.length === 0}
    <!-- Estado vazio (board Estados) -->
    <div class="stock-empty animate-rise">
      <div class="empty-visual" aria-hidden="true">
        <span class="empty-swatch"></span>
        <PaintBottle r={234} g={230} b={220} size={92} />
      </div>
      <p class="empty-big font-display">Sua estante ainda está vazia</p>
      <p class="empty-sub">Cadastre a primeira tinta ou importe um CSV com tudo de uma vez.</p>
      <div class="empty-actions">
        <button class="pill-dark" onclick={onCadastrar}>{cadastrarLabel()}</button>
        <button class="pill-light" onclick={pickFile}>Importar CSV</button>
      </div>
    </div>
    {:else}
    <!-- Filtros -->
    <div class="stock-filters animate-rise" style="animation-delay: 60ms;">
      <div class="brand-pills">
        <button class="filter-pill" class:active={selectedBrands.length === 0} onclick={() => (selectedBrands = [])}>{t('allBrands')}</button>
        {#each ownedBrands as name (name)}
          <button
            class="filter-pill"
            class:active={selectedBrands.includes(name)}
            aria-pressed={selectedBrands.includes(name)}
            onclick={() => (selectedBrands = selectedBrands.includes(name) ? selectedBrands.filter(b => b !== name) : [...selectedBrands, name])}
          >{name}</button>
        {/each}
      </div>
      <input type="search" class="stock-search" bind:value={searchQuery} placeholder="Buscar no estoque" aria-label="Buscar no estoque" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
    </div>

    <div class="stock-body animate-rise" style="animation-delay: 120ms;">
      <!-- Master: grid de cards -->
      <div class="stock-master">
        {#if filtered.length === 0}
          <div class="empty-state">
            <p class="empty-title">Nada com esse filtro</p>
            <p class="empty-hint">Limpe a busca ou o filtro de marca.</p>
          </div>
        {:else}
          <div class="stock-grid">
            {#each filtered as p (p.id)}
              <button class="stock-card" class:selected={selectedId === p.id} onclick={() => (selectedId = p.id)}>
                <span class="stock-visual">
                  <span class="stock-swatch" style="background: rgb({p.r}, {p.g}, {p.b});">
                    {#if p.volume}
                      <span class="vol-badge font-mono">{p.volume}</span>
                    {/if}
                  </span>
                  <PaintBottle r={p.r} g={p.g} b={p.b} size={64} />
                </span>
                <span class="stock-card-name">{p.name}</span>
                <span class="stock-card-meta font-mono">{p.code ? `${p.code} · ` : ''}{p.manufacturer}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Detail -->
      <aside class="stock-detail">
        {#if selected}
          <span class="label-mono">Tinta selecionada</span>
          <div class="detail-visual">
            <span class="detail-swatch" style="background: rgb({selected.r}, {selected.g}, {selected.b});"></span>
            <PaintBottle r={selected.r} g={selected.g} b={selected.b} size={96} />
          </div>
          <h2 class="detail-name font-display">{selected.name}</h2>
          <div class="detail-meta font-mono">
            <span>{selected.manufacturer}{selected.code ? ` · código ${selected.code}` : ''}</span>
            <span>{hexOf(selected.r, selected.g, selected.b)}</span>
            <span>RGB {selected.r} {selected.g} {selected.b}</span>
          </div>

          {#if selected.volume}
            <span class="label-mono detail-sec">Volume</span>
            <p class="detail-text">{selected.volume}</p>
          {/if}

          {#if selected.notes}
            <span class="label-mono detail-sec">Notas</span>
            <p class="detail-text">{selected.notes}</p>
          {/if}

          <div class="hairline detail-line"></div>

          {#if selectedRecipeCount > 0}
            <p class="detail-recipes">Aparece em {selectedRecipeCount} {selectedRecipeCount === 1 ? 'receita salva' : 'receitas salvas'}</p>
          {/if}

          <div class="detail-actions">
            <button class="pill-light" onclick={() => openEdit(selected!)}>{t('edit')}</button>
            <button class="pill-light" onclick={() => askRemove(selected!)}>{t('del')}</button>
          </div>
        {:else}
          <span class="label-mono">Tinta selecionada</span>
          <p class="detail-none">Escolha uma tinta na estante pra ver os detalhes aqui.</p>
        {/if}
      </aside>
    </div>
    {/if}
  </div>
  {/if}
</div>

{#if confirmDeletePaint}
  <div class="confirm-overlay">
    <div class="confirm-dialog">
      <div class="notice-title">{t('delPaintT', { name: confirmDeletePaint.name })}</div>
      <p class="notice-text">{t('delPaintB')}</p>
      <div class="confirm-actions">
        <button class="pill-light" onclick={() => (confirmDeletePaint = null)}>{t('cancel')}</button>
        <button class="pill-dark" onclick={confirmRemove}>{t('delPaintA')}</button>
      </div>
    </div>
  </div>
{/if}

{#if confirmDeleteMaker}
  <div class="confirm-overlay">
    <div class="confirm-dialog">
      <div class="notice-title">{t('delMakerT', { name: confirmDeleteMaker.name })}</div>
      <p class="notice-text">
        {confirmDeleteMaker.paintCount > 0 ? t('delMakerB', { n: confirmDeleteMaker.paintCount }) : t('delMakerB0')}
      </p>
      <div class="confirm-actions">
        <button class="pill-light" onclick={() => (confirmDeleteMaker = null)}>{t('cancel')}</button>
        <button class="pill-dark" onclick={confirmMakerDelete}>{t('delMakerA')}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Form add/editar — fiel a docs/Mescla AI.html: nome → código+restante lado
     a lado → fabricante em chips → cor (swatch+hex) → toggle "tenho" → ações.
     "Ler do pote" (câmera, US-20/E9) fica fora desta rodada, Could. -->
{#if formOpen}
  <div class="confirm-overlay">
    <div class="confirm-dialog form-dialog">
      <div class="notice-title">{editingId === null ? t('newPaint') : t('editPaint')}</div>

      <div class="paint-form">
        <label class="form-row">
          <span class="form-field-label">{t('fName')}</span>
          <input type="text" bind:value={fName} placeholder="Mephiston Red" class="form-input" />
        </label>

        <div class="form-row-split">
          <label class="form-row" style="flex: 1;">
            <span class="form-field-label">{t('fCode')}</span>
            <input type="text" bind:value={fCode} placeholder="70.951" class="form-input" />
          </label>
          <label class="form-row" style="width: 130px; flex-shrink: 0;">
            <span class="form-field-label">{t('fLeft')}</span>
            <input type="text" bind:value={fVolume} placeholder="17 ml" class="form-input" />
          </label>
        </div>

        <div class="form-row">
          <span class="form-field-label">{t('fMaker')}</span>
          <div class="form-chip-wrap">
            {#each manufacturers as mfr (mfr.id)}
              <button
                type="button"
                class="filter-pill"
                class:active={fMfr === mfr.id}
                onclick={() => (fMfr = mfr.id)}
              >{mfr.name}</button>
            {/each}
          </div>
        </div>

        <div class="form-row">
          <span class="form-field-label">{t('fColor')}</span>
          <div class="form-color-row">
            <span
              class="swatch-flat"
              style="width: 50px; height: 50px; background: rgb({hexToRgb(fHex).r}, {hexToRgb(fHex).g}, {hexToRgb(fHex).b});"
            ></span>
            <input
              type="text"
              bind:value={fHex}
              class="form-input font-mono"
              style="flex: 1;"
              maxlength="7"
              placeholder="#9A1115"
              aria-label={t('fColor')}
              autocomplete="off"
              autocorrect="off"
              autocapitalize="off"
              spellcheck="false"
            />
          </div>
        </div>

        <button type="button" class="have-toggle" class:checked={fHave} aria-pressed={fHave} onclick={() => (fHave = !fHave)}>
          <span class="have-check" class:checked={fHave}>
            {#if fHave}<Icon name="check" size={14} />{/if}
          </span>
          <span class="have-copy">
            <span class="have-title">{t('haveNow')}</span>
            <span class="have-note">{t('haveNote')}</span>
          </span>
        </button>
      </div>

      <div class="confirm-actions">
        <button class="pill-light" onclick={() => (formOpen = false)}>{t('cancel')}</button>
        <button class="pill-dark" onclick={save} disabled={saving}>
          {saving ? t('calculating') : (editingId === null ? t('addPaintBtn') : t('saveChanges'))}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Resultado da importação (fora do escopo do mockup — recurso próprio do
     app; convertido ao mesmo wrapper de diálogo pra sair do MDC) -->
{#if importOpen}
  <div class="confirm-overlay">
    <div class="confirm-dialog form-dialog">
      <div class="notice-title">Importação de CSV</div>
      {#if importResult}
        <p class="import-summary">
          <strong>{importResult.imported}</strong>&nbsp;{importResult.imported === 1 ? 'tinta importada' : 'tintas importadas'}
        </p>

        {#if importResult.errors && importResult.errors.length > 0}
          <p class="label-mono" style="margin: 18px 0 8px;">
            {importResult.errors.length} {importResult.errors.length === 1 ? 'linha ignorada' : 'linhas ignoradas'}
          </p>
          <div class="err-list">
            {#each importResult.errors as err}
              <div class="err-row">
                <span class="err-line font-mono">linha {err.line}</span>
                <span class="err-msg">{err.message}</span>
              </div>
            {/each}
          </div>
          <p class="err-hint">Corrija essas linhas no arquivo e importe de novo. As que já entraram não duplicam se você remover as boas do CSV.</p>
        {:else}
          <p class="err-hint" style="margin-top: 14px;">Tudo certo, nenhuma linha com erro.</p>
        {/if}
      {/if}
      <div class="confirm-actions">
        <button class="pill-dark" onclick={() => (importOpen = false)}>{t('close')}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .stock-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 24px;
    margin-bottom: 26px;
  }

  .stock-count {
    font-size: 12.5px;
    color: var(--text-2);
    margin-top: 4px;
  }

  .stock-cta {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }

  .text-link {
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    font-size: 13px;
    color: var(--text-2);
    cursor: pointer;
  }

  .text-link:hover:not(:disabled) {
    color: var(--grafite);
    text-decoration: underline;
  }

  .text-link:disabled {
    color: var(--text-3);
    cursor: not-allowed;
  }

  .stock-filters {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 28px;
    flex-wrap: wrap;
  }

  .brand-pills {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .stock-search {
    width: 220px;
    height: 38px;
    padding: 0 14px;
    font: inherit;
    font-size: 13px;
  }

  .stock-body {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 430px;
    gap: 0;
    align-items: start;
  }

  @media (max-width: 1000px) {
    .stock-body {
      grid-template-columns: 1fr;
    }
    .stock-detail {
      border-left: none;
      border-top: 1px solid var(--hairline);
      padding-left: 0;
      padding-top: 28px;
      margin-top: 28px;
    }
  }

  .stock-master {
    padding-right: 36px;
    min-width: 0;
  }

  .stock-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 22px 16px;
  }

  .stock-card {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 0;
    border: none;
    background: none;
    cursor: pointer;
    text-align: left;
    min-width: 0;
  }

  .stock-visual {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    margin-bottom: 8px;
  }

  .stock-swatch {
    position: relative;
    flex: 1;
    min-width: 0;
    aspect-ratio: 4 / 3;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px var(--color-neutral-800);
  }

  /* selecionada = borda laca 2px */
  .stock-card.selected .stock-swatch {
    box-shadow: 0 0 0 2px var(--laca);
  }

  .vol-badge {
    position: absolute;
    left: 10px;
    bottom: 8px;
    font-size: 10px;
    color: #fff;
    background: rgba(26, 23, 18, 0.5);
    padding: 2px 7px;
    border-radius: var(--radius-pill);
  }

  .stock-card-name {
    font-size: 13px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-card-meta {
    font-size: 10.5px;
    color: var(--text-2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-detail {
    border-left: 1px solid var(--hairline);
    padding-left: 36px;
    min-height: 380px;
  }

  .detail-visual {
    display: flex;
    align-items: flex-end;
    gap: 16px;
    margin: 18px 0 20px;
  }

  .detail-swatch {
    width: 180px;
    height: 130px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px var(--color-neutral-800);
  }

  .detail-name {
    font-size: 26px;
    font-weight: 720;
    color: var(--grafite);
    margin-bottom: 10px;
  }

  .detail-meta {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: var(--text-2);
    margin-bottom: 24px;
  }

  .detail-sec {
    display: block;
    margin-bottom: 6px;
  }

  .detail-text {
    font-size: 13.5px;
    color: var(--grafite);
    line-height: 1.5;
    margin-bottom: 20px;
  }

  .detail-line {
    margin: 4px 0 18px;
  }

  .detail-recipes {
    font-size: 13px;
    font-weight: 600;
    color: var(--laca-deep);
    margin-bottom: 20px;
  }

  .detail-actions {
    display: flex;
    gap: 10px;
  }

  .detail-none {
    margin-top: 16px;
    font-size: 13px;
    color: var(--text-2);
  }

  /* Estado vazio */
  .stock-empty {
    padding: 48px 0;
    max-width: 460px;
  }

  .empty-visual {
    display: flex;
    align-items: flex-end;
    gap: 18px;
    margin-bottom: 26px;
  }

  .empty-swatch {
    width: 150px;
    height: 100px;
    border: 2px dashed var(--hairline);
    border-radius: var(--radius-control);
  }

  .empty-big {
    font-size: 24px;
    font-weight: 720;
    color: var(--grafite);
    margin-bottom: 8px;
  }

  .empty-sub {
    font-size: 13.5px;
    color: var(--text-2);
    margin-bottom: 24px;
  }

  .empty-actions {
    display: flex;
    gap: 12px;
  }

  /* Form — fiel a docs/Mescla AI.html (seção "Cadastrar tinta") */
  .form-dialog {
    width: min(480px, 100%);
  }

  .paint-form {
    display: flex;
    flex-direction: column;
    gap: 14px;
    margin-top: 8px;
  }

  .form-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-row-split {
    display: flex;
    gap: 12px;
  }

  .form-field-label {
    font-size: 13px;
    color: var(--color-neutral-500);
  }

  .form-input {
    height: 50px;
    padding: 0 12px;
    border: 1px solid var(--color-neutral-800);
    border-radius: var(--radius-md);
    background: #1b1d2a;
    color: var(--color-text);
    font-family: inherit;
    font-size: 15px;
    outline: none;
  }

  .form-input:focus-visible {
    border-color: var(--color-accent);
  }

  .form-chip-wrap {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .form-color-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .have-toggle {
    display: flex;
    align-items: center;
    gap: 12px;
    height: 60px;
    padding: 0 14px;
    border: 1px solid var(--color-neutral-800);
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--color-text);
    font-family: inherit;
    font-size: 15px;
    text-align: left;
    cursor: pointer;
  }

  .have-toggle.checked {
    border-color: var(--color-accent);
    background: color-mix(in srgb, var(--color-accent) 12%, transparent);
  }

  .have-check {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: 1px solid var(--color-neutral-800);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    color: var(--color-accent);
  }

  .have-check.checked {
    border-color: var(--color-accent);
  }

  .have-copy {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .have-title {
    font-weight: 500;
  }

  .have-note {
    font-size: 12.5px;
    color: var(--color-neutral-500);
  }

  .import-summary {
    font-size: 14px;
    color: var(--grafite);
  }

  .err-list {
    display: flex;
    flex-direction: column;
    max-height: 260px;
    overflow-y: auto;
    border-top: 1px solid var(--hairline);
  }

  .err-row {
    display: flex;
    gap: 10px;
    align-items: baseline;
    padding: 8px 0;
    border-bottom: 1px solid var(--hairline);
    font-size: 12.5px;
  }

  .err-line {
    color: var(--laca-deep);
    flex-shrink: 0;
    font-size: 11.5px;
  }

  .err-msg {
    color: var(--grafite);
  }

  .err-hint {
    margin-top: 12px;
    font-size: 12.5px;
    color: var(--text-2);
  }

  /* ── Aba Fabricantes ── */
  .mfr-list {
    border-top: 1px solid var(--color-divider);
    padding: 0 var(--space-6);
  }

  .mfr-row {
    display: flex;
    align-items: center;
    gap: var(--space-6);
    padding: var(--space-4) 0;
    border-bottom: 1px solid var(--color-divider);
  }

  .mfr-badge {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: var(--color-accent-800);
    flex-shrink: 0;
  }

  .mfr-initial {
    color: var(--color-neutral-100);
    font-size: 17px;
    font-weight: 700;
  }

  .mfr-id {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    width: 220px;
    flex-shrink: 0;
  }

  .mfr-name {
    font-size: 16px;
    font-weight: 700;
    color: var(--color-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mfr-meta {
    font-size: 11px;
    color: var(--text-2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mfr-chips {
    display: flex;
    gap: var(--space-2);
    flex: 1;
    min-width: 0;
  }

  .mfr-chip {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.1);
    flex-shrink: 0;
  }

  .mfr-count {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
    flex-shrink: 0;
    min-width: 96px;
  }

  .mfr-count-n {
    font-size: 20px;
    font-weight: 600;
    color: var(--color-text);
    letter-spacing: -0.02em;
  }

  .mfr-count-label {
    font-size: 11px;
    color: var(--text-2);
  }

  .mfr-count-owned {
    font-size: 10.5px;
    color: var(--color-accent-2);
  }

  .mfr-open, .mfr-del {
    flex-shrink: 0;
    min-height: 44px;
    border: none;
    background: none;
    padding: 0 var(--space-2);
    font-size: 13px;
    font-weight: 560;
    color: var(--color-text);
    cursor: pointer;
    white-space: nowrap;
  }

  .mfr-open:hover { color: var(--color-accent-2); }
  .mfr-del { color: var(--color-danger); }
  .mfr-del:hover { text-decoration: underline; }
</style>
