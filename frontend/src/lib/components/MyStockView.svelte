<script lang="ts">
  // Meu estoque (Tintômetro) — master-detail: grid de cards à esquerda,
  // painel da tinta selecionada à direita. CRUD + importação/exportação CSV.
  // As tintas daqui alimentam a opção "priorizar meu estoque" na Equivalência.
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Select, { Option } from '@smui/select';
  import Dialog, { Content as DialogContent, Title as DialogTitle } from '@smui/dialog';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import { recipesWithIngredient } from '../recipes.svelte';
  import { hexOf } from '../ui';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type { UserPaintDTO, CSVImportResultDTO } from '../../../bindings/paint-match-ai/models';

  interface Props {
    // vindo da busca global: "Adicionar {tinta} ao meu estoque"
    prefillPaintId?: number | null;
  }

  let { prefillPaintId = null }: Props = $props();

  let paints: UserPaintDTO[] = $state([]);
  let manufacturers: { id: number; name: string }[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedBrands: string[] = $state([]);
  let selectedId: number | null = $state(null);

  // Form (add/editar)
  let formOpen = $state(false);
  let editingId: number | null = $state(null);
  let fMfr: number | '' = $state('');
  let fName = $state('');
  let fCode = $state('');
  let fHex = $state('#8a8a8a');
  let fVolume = $state('');
  let fNotes = $state('');
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
      const [stock, mfrs] = await Promise.all([
        PaintService.GetUserPaints(),
        PaintService.GetManufacturers(),
      ]);
      paints = stock || [];
      manufacturers = (mfrs || []).map(m => ({ id: m.id, name: m.name }));
      if (selectedId === null || !paints.some(p => p.id === selectedId)) {
        selectedId = paints[0]?.id ?? null;
      }
    } catch (e) {
      console.error('Erro carregando estoque:', e);
      toast('Não consegui carregar seu estoque.', 'error');
    } finally {
      loading = false;
    }
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

  async function remove(p: UserPaintDTO) {
    if (!confirm(`Remover "${p.name}" do seu estoque?`)) return;
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

<div class="page-container">
  <!-- Header -->
  <div class="stock-head animate-rise">
    <div>
      <h1 class="page-title">Meu estoque</h1>
      <p class="stock-count font-mono">
        {paints.length.toLocaleString('pt-BR')} tintas&nbsp;&nbsp;·&nbsp;&nbsp;{ownedBrands.length} {ownedBrands.length === 1 ? 'fabricante' : 'fabricantes'}
      </p>
    </div>
    <div class="stock-cta">
      <button class="text-link" onclick={downloadTemplate}>Baixar modelo</button>
      <button class="text-link" onclick={exportCsv} disabled={paints.length === 0}>Exportar CSV</button>
      <button class="pill-light" onclick={pickFile}>Importar CSV</button>
      <button class="pill-dark" onclick={openAdd}>+ Nova tinta</button>
    </div>
  </div>

  <input
    bind:this={fileInput}
    type="file"
    accept=".csv,text/csv"
    style="display: none;"
    onchange={onFileChosen}
  />

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
        <button class="pill-dark" onclick={openAdd}>+ Nova tinta</button>
        <button class="pill-light" onclick={pickFile}>Importar CSV</button>
      </div>
    </div>
  {:else}
    <!-- Filtros -->
    <div class="stock-filters animate-rise" style="animation-delay: 60ms;">
      <div class="brand-pills">
        <button class="filter-pill" class:active={selectedBrands.length === 0} onclick={() => (selectedBrands = [])}>Todas</button>
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
            <button class="pill-light" onclick={() => openEdit(selected!)}>Editar</button>
            <button class="pill-light" onclick={() => remove(selected!)}>Excluir</button>
          </div>
        {:else}
          <span class="label-mono">Tinta selecionada</span>
          <p class="detail-none">Escolha uma tinta na estante pra ver os detalhes aqui.</p>
        {/if}
      </aside>
    </div>
  {/if}
</div>

<!-- Form add/editar -->
<Dialog bind:open={formOpen} surface$style="background: var(--papel); border-radius: var(--radius-surface); max-width: 480px; width: 100%;">
  <DialogTitle>{editingId === null ? 'Nova tinta' : 'Editar tinta'}</DialogTitle>
  <DialogContent>
    <div style="display: flex; flex-direction: column; gap: 16px; padding-top: 8px;">
      <div style="display: flex; gap: 14px; align-items: center;">
        <span class="swatch-flat" style="width: 56px; height: 56px; background: rgb({hexToRgb(fHex).r}, {hexToRgb(fHex).g}, {hexToRgb(fHex).b});"></span>
        <div style="display: flex; flex-direction: column; gap: 6px;">
          <span class="form-label label-mono">Cor</span>
          <div style="display: flex; gap: 8px; align-items: center;">
            <input type="color" bind:value={fHex} class="color-swatch" aria-label="Escolher cor" />
            <input type="text" bind:value={fHex} class="hex-input font-mono" maxlength="7" aria-label="Hex" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
          </div>
        </div>
      </div>

      <Select variant="outlined" bind:value={fMfr} label="Fabricante" style="width: 100%;">
        {#each manufacturers as mfr (mfr.id)}
          <Option value={mfr.id}>{mfr.name}</Option>
        {/each}
      </Select>
      <Textfield variant="outlined" bind:value={fName} label="Nome da tinta" style="width: 100%;" />
      <div style="display: flex; gap: 12px;">
        <Textfield variant="outlined" bind:value={fCode} label="Código (opcional)" style="flex: 1;" />
        <Textfield variant="outlined" bind:value={fVolume} label="Volume (ex.: 17 ml)" style="flex: 1;" />
      </div>
      <Textfield variant="outlined" bind:value={fNotes} label="Notas (opcional)" textarea style="width: 100%;" />
    </div>

    <div style="display: flex; gap: 10px; margin-top: 22px; justify-content: flex-end;">
      <button class="pill-light" onclick={() => (formOpen = false)}>Cancelar</button>
      <button class="pill-dark" onclick={save} disabled={saving}>
        {saving ? 'Salvando…' : 'Salvar'}
      </button>
    </div>
  </DialogContent>
</Dialog>

<!-- Resultado da importação -->
<Dialog bind:open={importOpen} surface$style="background: var(--papel); border-radius: var(--radius-surface); max-width: 520px; width: 100%;">
  <DialogTitle>Importação de CSV</DialogTitle>
  <DialogContent>
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
    <div style="display: flex; justify-content: flex-end; margin-top: 20px;">
      <button class="pill-dark" onclick={() => (importOpen = false)}>Fechar</button>
    </div>
  </DialogContent>
</Dialog>

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
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.06);
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
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.06);
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

  /* Form */
  .form-label {
    font-size: 10px;
  }

  .color-swatch {
    width: 44px;
    height: 34px;
    padding: 0;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-control);
    background: transparent;
    cursor: pointer;
  }

  .color-swatch:focus-visible {
    outline: none;
    border-color: var(--laca);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--laca) 18%, transparent);
  }

  .hex-input {
    width: 96px;
    padding: 8px 10px;
    font-size: 13px;
    text-transform: uppercase;
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
</style>
