<script lang="ts">
  // Meu estoque — o banco de tintas do próprio pintor. CRUD + importação CSV.
  // As tintas daqui alimentam a opção "priorizar meu estoque" na Equivalência.
  import { onMount } from 'svelte';
  import Textfield from '@smui/textfield';
  import Select, { Option } from '@smui/select';
  import Dialog, { Content as DialogContent, Title as DialogTitle } from '@smui/dialog';
  import Icon from './Icon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { toast } from '../toast.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type { UserPaintDTO, CSVImportResultDTO } from '../../../bindings/paint-match-ai/models';

  let paints: UserPaintDTO[] = $state([]);
  let manufacturers: { id: number; name: string }[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedManufacturer = $state('');

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

  onMount(load);

  async function load() {
    loading = true;
    try {
      const [stock, mfrs] = await Promise.all([
        PaintService.GetUserPaints(),
        PaintService.GetManufacturers(),
      ]);
      paints = stock || [];
      manufacturers = (mfrs || []).map(m => ({ id: m.id, name: m.name }));
    } catch (e) {
      console.error('Erro carregando estoque:', e);
      toast('Não consegui carregar seu estoque.', 'error');
    } finally {
      loading = false;
    }
  }

  let filtered = $derived(
    paints.filter(p => {
      if (selectedManufacturer && p.manufacturer !== selectedManufacturer) return false;
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        p.name.toLowerCase().includes(q) ||
        p.code.toLowerCase().includes(q) ||
        p.manufacturer.toLowerCase().includes(q)
      );
    })
  );

  // Marcas que o pintor realmente tem em estoque — resumo no topo.
  let ownedBrands = $derived([...new Set(paints.map(p => p.manufacturer))].sort());

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
    fHex = rgbToHex(p.r, p.g, p.b);
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
        await PaintService.AddUserPaint(dto);
        toast('Tinta adicionada ao estoque.');
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
  <div class="page-header animate-rise">
    <h1 class="page-title">Meu estoque</h1>
    <p class="page-subtitle">
      {#if paints.length === 0}
        Cadastre as tintas que você tem em casa — depois a Equivalência pode priorizar o que já é seu.
      {:else}
        {filtered.length.toLocaleString('pt-BR')} de {paints.length.toLocaleString('pt-BR')} tintas · {ownedBrands.length} {ownedBrands.length === 1 ? 'marca' : 'marcas'}
      {/if}
    </p>
    <div class="page-divider"></div>
  </div>

  <!-- Ações -->
  <div class="panel p-3 mb-6 animate-rise stock-toolbar" style="animation-delay: 60ms; position: relative; z-index: 1;">
    <div class="stock-filters">
      <div class="flex-1" style="min-width: 200px;">
        <Textfield variant="outlined" bind:value={searchQuery} label="Buscar no meu estoque..." style="width: 100%;">
          {#snippet leadingIcon()}
            <span class="mdc-text-field__icon mdc-text-field__icon--leading" style="color: var(--ink-500); display: flex;"><Icon name="search" size={17} /></span>
          {/snippet}
        </Textfield>
      </div>
      <div style="min-width: 200px;">
        <Select variant="outlined" bind:value={selectedManufacturer} label="Fabricante" style="width: 100%;">
          <Option value="">Todas as marcas</Option>
          {#each ownedBrands as name}
            <Option value={name}>{name}</Option>
          {/each}
        </Select>
      </div>
    </div>

    <div class="stock-actions">
      <button class="btn-primary stock-add" onclick={openAdd}>
        <Icon name="plus" size={17} /> Adicionar tinta
      </button>
      <div class="stock-csv-buttons">
        <button class="btn-ghost" onclick={pickFile} title="Importar tintas de um CSV">
          <Icon name="upload" size={15} /> Importar
        </button>
        <button class="btn-ghost" onclick={exportCsv} disabled={paints.length === 0} title="Exportar seu estoque (backup)">
          <Icon name="download" size={15} /> Exportar
        </button>
        <button class="btn-ghost" onclick={downloadTemplate} title="Baixar CSV de exemplo">
          <Icon name="book" size={15} /> Modelo
        </button>
      </div>
    </div>

    <input
      bind:this={fileInput}
      type="file"
      accept=".csv,text/csv"
      style="display: none;"
      onchange={onFileChosen}
    />
  </div>

  {#if loading}
    <div class="stock-list">
      {#each Array(6) as _}
        <div class="panel p-3" style="display: flex; gap: 12px; align-items: center;">
          <div class="skeleton" style="width: 40px; height: 52px; border-radius: 6px;"></div>
          <div style="flex: 1; display: flex; flex-direction: column; gap: 8px;">
            <div class="skeleton" style="height: 12px; width: 40%;"></div>
            <div class="skeleton" style="height: 10px; width: 25%;"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if paints.length === 0}
    <div class="empty-state">
      <div class="empty-icon"><Icon name="box" size={40} /></div>
      <p class="empty-title">Seu estoque está vazio</p>
      <p class="empty-hint">Adicione uma tinta ou importe um CSV. Depois, na Equivalência, ligue "priorizar meu estoque" pra montar a receita só com o que você tem.</p>
    </div>
  {:else if filtered.length === 0}
    <div class="empty-state">
      <div class="empty-icon"><Icon name="search-off" size={40} /></div>
      <p class="empty-title">Nada com esse filtro</p>
      <p class="empty-hint">Limpe a busca ou o filtro de marca.</p>
    </div>
  {:else}
    <div class="stock-list">
      {#each filtered as p, i (p.id)}
        <div class="stock-row panel p-3 animate-rise" style="animation-delay: {Math.min(i * 15, 180)}ms;">
          <PaintBottle r={p.r} g={p.g} b={p.b} size={44} label={p.code} />
          <div class="stock-main">
            <div class="stock-name">{p.name}</div>
            <div class="stock-meta">
              <span>{p.manufacturer}</span>
              {#if p.code}<span class="dot">·</span><span class="font-mono">{p.code}</span>{/if}
              {#if p.volume}<span class="dot">·</span><span>{p.volume}</span>{/if}
            </div>
            {#if p.notes}<div class="stock-notes">{p.notes}</div>{/if}
          </div>
          <div class="stock-actions">
            <button class="icon-btn" onclick={() => openEdit(p)} aria-label="Editar" title="Editar"><Icon name="edit" size={16} /></button>
            <button class="icon-btn danger" onclick={() => remove(p)} aria-label="Remover" title="Remover"><Icon name="trash" size={16} /></button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Form add/editar -->
<Dialog bind:open={formOpen} surface$style="background: var(--ink-900); border: 1px solid var(--ink-700); border-radius: 10px; max-width: 480px; width: 100%;">
  <DialogTitle>{editingId === null ? 'Adicionar tinta' : 'Editar tinta'}</DialogTitle>
  <DialogContent>
    <div style="display: flex; flex-direction: column; gap: 16px; padding-top: 8px;">
      <div style="display: flex; gap: 14px; align-items: center;">
        <PaintBottle r={hexToRgb(fHex).r} g={hexToRgb(fHex).g} b={hexToRgb(fHex).b} size={56} label={fCode} />
        <div style="display: flex; flex-direction: column; gap: 6px;">
          <span class="detail-label">Cor</span>
          <div style="display: flex; gap: 8px; align-items: center;">
            <input type="color" bind:value={fHex} class="color-swatch" aria-label="Escolher cor" />
            <input type="text" bind:value={fHex} class="hex-input font-mono" maxlength="7" aria-label="Hex" />
          </div>
        </div>
      </div>

      <Select variant="outlined" bind:value={fMfr} label="Fabricante" style="width: 100%;">
        {#each manufacturers as mfr}
          <Option value={mfr.id}>{mfr.name}</Option>
        {/each}
      </Select>
      <Textfield variant="outlined" bind:value={fName} label="Nome da tinta" style="width: 100%;" />
      <div style="display: flex; gap: 12px;">
        <Textfield variant="outlined" bind:value={fCode} label="Código (opcional)" style="flex: 1;" />
        <Textfield variant="outlined" bind:value={fVolume} label="Volume (ex.: 17ml)" style="flex: 1;" />
      </div>
      <Textfield variant="outlined" bind:value={fNotes} label="Notas (opcional)" textarea style="width: 100%;" />
    </div>

    <div style="display: flex; gap: 10px; margin-top: 22px; justify-content: flex-end;">
      <button class="btn-ghost" onclick={() => (formOpen = false)}>Cancelar</button>
      <button class="btn-primary" style="width: auto;" onclick={save} disabled={saving}>
        <Icon name="check" size={16} /> {saving ? 'Salvando…' : 'Salvar'}
      </button>
    </div>
  </DialogContent>
</Dialog>

<!-- Resultado da importação -->
<Dialog bind:open={importOpen} surface$style="background: var(--ink-900); border: 1px solid var(--ink-700); border-radius: 10px; max-width: 520px; width: 100%;">
  <DialogTitle>Importação de CSV</DialogTitle>
  <DialogContent>
    {#if importResult}
      <div class="import-summary" class:ok={importResult.imported > 0}>
        <Icon name={importResult.imported > 0 ? 'check' : 'info'} size={18} />
        <span><strong>{importResult.imported}</strong> {importResult.imported === 1 ? 'tinta importada' : 'tintas importadas'}</span>
      </div>

      {#if importResult.errors && importResult.errors.length > 0}
        <p class="detail-label" style="margin: 18px 0 8px;">
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
        <p class="empty-hint" style="margin-top: 12px;">Corrija essas linhas no arquivo e importe de novo — as que já entraram não duplicam se você remover as boas do CSV.</p>
      {:else}
        <p class="empty-hint" style="margin-top: 14px;">Tudo certo — nenhuma linha com erro.</p>
      {/if}
    {/if}
    <div style="display: flex; justify-content: flex-end; margin-top: 20px;">
      <button class="btn-primary" style="width: auto;" onclick={() => (importOpen = false)}>Fechar</button>
    </div>
  </DialogContent>
</Dialog>

<style>
  /* Barra de ações: filtros à esquerda, ações à direita; quebra em telas estreitas */
  .stock-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .stock-filters {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 260px;
  }

  .stock-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  /* O btn-primary global é width:100% sem padding lateral — aqui ele é auto,
     então precisa de respiro nas laterais pra não colar o texto na borda. */
  .stock-add {
    width: auto;
    padding: 0 20px;
    white-space: nowrap;
  }

  .stock-csv-buttons {
    display: flex;
    gap: 8px;
  }

  .stock-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .stock-row {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .stock-main {
    flex: 1;
    min-width: 0;
  }

  .stock-name {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 600;
    color: var(--paper);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12.5px;
    color: var(--ink-500);
    margin-top: 2px;
  }

  .stock-meta .dot {
    color: var(--ink-600);
  }

  .stock-notes {
    font-size: 12px;
    color: var(--ink-400);
    margin-top: 4px;
    font-style: italic;
  }

  .stock-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .icon-btn {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--ink-700);
    border-radius: 7px;
    background: transparent;
    color: var(--ink-400);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .icon-btn:hover {
    color: var(--paper);
    border-color: var(--ink-500);
  }

  .icon-btn.danger:hover {
    color: #f08a8a;
    border-color: #6b3030;
  }

  .color-swatch {
    width: 44px;
    height: 34px;
    padding: 0;
    border: 1px solid var(--ink-700);
    border-radius: 7px;
    background: transparent;
    cursor: pointer;
  }

  .hex-input {
    width: 96px;
    padding: 8px 10px;
    border: 1px solid var(--ink-700);
    border-radius: 7px;
    background: var(--ink-800);
    color: var(--paper);
    font-size: 13px;
    text-transform: uppercase;
  }

  .hex-input:focus {
    outline: none;
    border-color: var(--lacquer);
  }

  .detail-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-weight: 600;
    color: var(--ink-500);
  }

  .import-summary {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    border-radius: 8px;
    background: var(--ink-800);
    color: var(--ink-300);
    font-size: 14px;
  }

  .import-summary.ok {
    color: var(--paper);
  }

  .err-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 260px;
    overflow-y: auto;
  }

  .err-row {
    display: flex;
    gap: 10px;
    align-items: baseline;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--ink-800);
    font-size: 12.5px;
  }

  .err-line {
    color: #f0a86a;
    flex-shrink: 0;
    font-size: 11.5px;
  }

  .err-msg {
    color: var(--ink-300);
  }
</style>
