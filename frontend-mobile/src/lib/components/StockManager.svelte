<script lang="ts">
  // Meu estoque (mobile) — CRUD das tintas do pintor + importação CSV.
  // Renderizado dentro da sub-tela "Meu estoque" da aba Mais. A crítica do CSV
  // e o modelo vêm do WASM (pkg/stock), a mesma do desktop.
  import Icon from './Icon.svelte';
  import BottomSheet from './BottomSheet.svelte';
  import { allManufacturers } from '../services/catalog';
  import {
    stock,
    sortedStock,
    stockBrands,
    addStockPaint,
    updateStockPaint,
    removeStockPaint,
    addStockPaints,
  } from '../services/stock.svelte';
  import { parseStockCSV, stockCSVTemplate, stockToCSV, type StockPaint, type StockRowError } from '../services/engine';
  import { toast } from '../toast.svelte';

  let search = $state('');
  let formOpen = $state(false);
  let importOpen = $state(false);
  let saving = $state(false);
  let importErrors: StockRowError[] = $state([]);
  let importCount = $state(0);
  let fileInput: HTMLInputElement;

  // Campos do form
  let editingId: number | null = $state(null);
  let fMfr: number | '' = $state('');
  let fName = $state('');
  let fCode = $state('');
  let fHex = $state('#8a8a8a');
  let fVolume = $state('');
  let fNotes = $state('');

  let mfrs = $derived(allManufacturers());

  let filtered = $derived(
    sortedStock().filter(p => {
      if (!search.trim()) return true;
      const q = search.toLowerCase();
      return (
        p.name.toLowerCase().includes(q) ||
        p.code.toLowerCase().includes(q) ||
        p.manufacturer.toLowerCase().includes(q)
      );
    }),
  );

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
    fMfr = mfrs[0]?.id ?? '';
    fName = '';
    fCode = '';
    fHex = '#8a8a8a';
    fVolume = '';
    fNotes = '';
    formOpen = true;
  }

  function openEdit(p: StockPaint) {
    editingId = p.id;
    fMfr = p.manufacturerId;
    fName = p.name;
    fCode = p.code;
    fHex = rgbToHex(p.r, p.g, p.b);
    fVolume = p.volume;
    fNotes = p.notes;
    formOpen = true;
  }

  function save() {
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
    const mfr = mfrs.find(m => m.id === Number(fMfr));
    const base = {
      manufacturerId: Number(fMfr),
      manufacturer: mfr?.name ?? '',
      name: fName.trim(),
      code: fCode.trim(),
      r,
      g,
      b,
      volume: fVolume.trim(),
      notes: fNotes.trim(),
    };
    if (editingId === null) {
      addStockPaint(base);
      toast('Tinta adicionada ao estoque.');
    } else {
      updateStockPaint({ ...base, id: editingId });
      toast('Tinta atualizada.');
    }
    saving = false;
    formOpen = false;
  }

  function remove(p: StockPaint) {
    if (!confirm(`Remover "${p.name}" do seu estoque?`)) return;
    removeStockPaint(p.id);
    toast('Tinta removida.');
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
      downloadCsv(await stockCSVTemplate(), 'modelo-estoque-mescla.csv');
    } catch {
      toast('Não consegui gerar o modelo.', 'error');
    }
  }

  async function exportCsv() {
    if (stock.paints.length === 0) return;
    try {
      downloadCsv(await stockToCSV(stock.paints), 'meu-estoque-mescla.csv');
      toast(`${stock.paints.length} ${stock.paints.length === 1 ? 'tinta exportada' : 'tintas exportadas'}.`);
    } catch {
      toast('Não consegui exportar o estoque.', 'error');
    }
  }

  function pickFile() {
    fileInput?.click();
  }

  async function onFileChosen(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    try {
      const text = await file.text();
      const result = await parseStockCSV(text);
      importCount = addStockPaints(result.paints);
      importErrors = result.errors;
      importOpen = true;
    } catch (err) {
      console.error('Erro importando CSV:', err);
      toast('Não consegui ler esse CSV.', 'error');
    }
  }
</script>

<div class="stock">
  <p class="stock-sub">
    {#if stock.paints.length === 0}
      Cadastre as tintas que você tem — a aba Mesclar pode priorizar o que já é seu.
    {:else}
      {stock.paints.length} {stock.paints.length === 1 ? 'tinta' : 'tintas'} · {stockBrands().length} {stockBrands().length === 1 ? 'marca' : 'marcas'}
    {/if}
  </p>

  <div class="stock-actions">
    <button class="btn-primary" onclick={openAdd}>
      <Icon name="plus" size={18} /> Adicionar tinta
    </button>
    <div class="csv-row">
      <button class="btn-ghost" onclick={pickFile}><Icon name="upload" size={15} /> Importar</button>
      {#if stock.paints.length > 0}
        <button class="btn-ghost" onclick={exportCsv}><Icon name="download" size={15} /> Exportar</button>
      {/if}
      <button class="btn-ghost" onclick={downloadTemplate}><Icon name="book" size={15} /> Modelo</button>
    </div>
    <input bind:this={fileInput} type="file" accept=".csv,text/csv" style="display: none;" onchange={onFileChosen} />
  </div>

  {#if stock.paints.length > 0}
    <div class="stock-search">
      <Icon name="search" size={17} />
      <input type="text" bind:value={search} placeholder="Buscar no meu estoque…" />
    </div>
  {/if}

  {#if stock.paints.length === 0}
    <div class="empty-state">
      <span class="empty-icon"><Icon name="box" size={36} /></span>
      <p class="empty-title">Estoque vazio</p>
      <p class="empty-hint">Adicione uma tinta ou importe um CSV.</p>
    </div>
  {:else if filtered.length === 0}
    <div class="empty-state">
      <span class="empty-icon"><Icon name="search-off" size={36} /></span>
      <p class="empty-title">Nada com esse nome</p>
    </div>
  {:else}
    <div class="stock-list">
      {#each filtered as p (p.id)}
        <div class="stock-row">
          <span class="swatch-flat" style="width: 44px; height: 44px; background: rgb({p.r}, {p.g}, {p.b});"></span>
          <button class="stock-main pressable" onclick={() => openEdit(p)}>
            <span class="stock-name">{p.name}</span>
            <span class="stock-meta">
              {p.manufacturer}{#if p.code} · <span class="font-mono">{p.code}</span>{/if}{#if p.volume} · {p.volume}{/if}
            </span>
          </button>
          <button class="row-trash pressable" onclick={() => remove(p)} aria-label="Remover">
            <Icon name="trash" size={17} />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Form add/editar -->
<BottomSheet open={formOpen} onClose={() => (formOpen = false)} title={editingId === null ? 'Adicionar tinta' : 'Editar tinta'}>
  <div class="form">
    <div class="form-color">
      <span class="swatch-flat" style="width: 52px; height: 52px; background: rgb({hexToRgb(fHex).r}, {hexToRgb(fHex).g}, {hexToRgb(fHex).b});"></span>
      <div class="color-inputs">
        <span class="form-label">Cor</span>
        <div class="color-row">
          <input type="color" bind:value={fHex} class="color-swatch" aria-label="Escolher cor" />
          <input type="text" bind:value={fHex} class="hex-input font-mono" maxlength="7" aria-label="Hex" />
        </div>
      </div>
    </div>

    <label class="form-label" for="stk-mfr">Fabricante</label>
    <select id="stk-mfr" bind:value={fMfr} class="form-field">
      {#each mfrs as m (m.id)}
        <option value={m.id}>{m.name}</option>
      {/each}
    </select>

    <label class="form-label" for="stk-name">Nome</label>
    <input id="stk-name" type="text" bind:value={fName} class="form-field" placeholder="Ex.: Model Color Black" />

    <div class="form-two">
      <div>
        <label class="form-label" for="stk-code">Código</label>
        <input id="stk-code" type="text" bind:value={fCode} class="form-field" placeholder="70.950" />
      </div>
      <div>
        <label class="form-label" for="stk-vol">Volume</label>
        <input id="stk-vol" type="text" bind:value={fVolume} class="form-field" placeholder="17ml" />
      </div>
    </div>

    <label class="form-label" for="stk-notes">Notas</label>
    <input id="stk-notes" type="text" bind:value={fNotes} class="form-field" placeholder="opcional" />

    <button class="btn-primary" style="margin-top: 8px;" onclick={save} disabled={saving}>
      <Icon name="check" size={18} /> {saving ? 'Salvando…' : 'Salvar'}
    </button>
  </div>
</BottomSheet>

<!-- Resultado da importação -->
<BottomSheet open={importOpen} onClose={() => (importOpen = false)} title="Importação de CSV">
  <div class="import">
    <div class="import-summary" class:ok={importCount > 0}>
      <Icon name={importCount > 0 ? 'check' : 'info'} size={18} />
      <span><strong>{importCount}</strong> {importCount === 1 ? 'tinta importada' : 'tintas importadas'}</span>
    </div>
    {#if importErrors.length > 0}
      <p class="form-label" style="margin: 16px 0 8px;">{importErrors.length} {importErrors.length === 1 ? 'linha ignorada' : 'linhas ignoradas'}</p>
      <div class="err-list">
        {#each importErrors as err}
          <div class="err-row">
            <span class="err-line font-mono">linha {err.line}</span>
            <span class="err-msg">{err.message}</span>
          </div>
        {/each}
      </div>
    {/if}
    <button class="btn-primary" style="margin-top: 16px;" onclick={() => (importOpen = false)}>Fechar</button>
  </div>
</BottomSheet>

<style>
  .stock {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .stock-sub {
    font-size: 13px;
    color: var(--ink-500);
    line-height: 1.5;
  }

  .stock-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .csv-row {
    display: flex;
    gap: 10px;
  }

  .csv-row .btn-ghost {
    flex: 1;
  }

  .stock-search {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 48px;
    padding: 0 12px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    color: var(--ink-500);
  }

  .stock-search:focus-within {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .stock-search input {
    flex: 1;
    min-width: 0;
    border: none;
    background: transparent;
    color: var(--ink-100);
    font-size: 16px;
    outline: none;
    box-shadow: none;
  }

  .stock-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .stock-row {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 60px;
    padding: 8px 12px;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-surface);
    background: var(--ink-900);
  }

  .stock-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    text-align: left;
    background: transparent;
    border: none;
  }

  .stock-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-meta {
    font-size: 12px;
    color: var(--ink-500);
  }

  .row-trash {
    flex-shrink: 0;
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: var(--radius-pill);
    background: transparent;
    color: var(--ink-500);
  }

  /* Form */
  .form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding-bottom: 8px;
  }

  .form-color {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 6px;
  }

  .color-inputs {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .color-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .color-swatch {
    width: 52px;
    height: 44px;
    padding: 0;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: transparent;
  }

  .color-swatch:focus {
    outline: none;
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .hex-input {
    width: 104px;
    min-height: 44px;
    padding: 0 10px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    color: var(--ink-100);
    font-size: 16px;
    text-transform: uppercase;
  }

  .hex-input:focus {
    outline: none;
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  /* Label-etiqueta: mono, caixa alta, espaçada — padrão de rótulo. */
  .form-label {
    font-family: var(--font-mono);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 600;
    color: var(--ink-500);
  }

  .form-field {
    width: 100%;
    min-height: 48px;
    padding: 11px 12px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    color: var(--ink-100);
    font-size: 16px;
    outline: none;
  }

  .form-field:focus {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .form-two {
    display: flex;
    gap: 12px;
  }

  .form-two > div {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  /* Importação */
  .import {
    display: flex;
    flex-direction: column;
  }

  .import-summary {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    border-radius: var(--radius-surface);
    background: var(--ink-800);
    color: var(--ink-300);
    font-size: 14px;
  }

  .import-summary.ok {
    color: var(--ink-100);
  }

  .err-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 40vh;
    overflow-y: auto;
  }

  .err-row {
    display: flex;
    gap: 10px;
    align-items: baseline;
    padding: 8px 10px;
    border-radius: var(--radius-control);
    background: var(--ink-800);
    font-size: 12.5px;
  }

  .err-line {
    color: var(--lacquer-deep);
    flex-shrink: 0;
    font-size: 11.5px;
  }

  .err-msg {
    color: var(--ink-300);
  }
</style>
