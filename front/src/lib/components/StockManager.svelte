<script lang="ts">
  // Meu estoque (mobile) — CRUD das tintas do pintor + importação CSV.
  // Renderizado dentro da sub-tela "Meu estoque" da aba Mais. A crítica do CSV
  // e o modelo vêm do WASM (pkg/stock), a mesma do desktop.
  import Icon from './Icon.svelte';
  import PaintBottle from './PaintBottle.svelte';
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
  import { appState } from '../appState.svelte';
  import { toast } from '../toast.svelte';

  interface Props {
    /** volta pro menu Mais (o header "‹ Mais" mora aqui dentro). */
    onBack?: () => void;
  }

  let { onBack }: Props = $props();

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

  // "Tenho outra parecida" (detalhe da tinta): abre o form já com cor + marca.
  $effect(() => {
    if (appState.pendingStockPrefill) {
      const pre = appState.pendingStockPrefill;
      appState.pendingStockPrefill = null;
      openAdd();
      fMfr = pre.manufacturerId;
      fHex = pre.hex;
    }
  });

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

  function removeEditing() {
    if (editingId === null) return;
    const p = stock.paints.find(x => x.id === editingId);
    if (!p) return;
    if (!confirm(`Remover "${p.name}" do seu estoque?`)) return;
    removeStockPaint(p.id);
    formOpen = false;
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
  {#if onBack}
    <button class="stock-back pressable" onclick={onBack}>
      <Icon name="chevron-left" size={16} /> Mais
    </button>
  {/if}

  <div class="stock-head">
    <h1 class="screen-title">Meu estoque</h1>
    <button class="stock-add pressable" onclick={openAdd} aria-label="Nova tinta do estoque">
      <Icon name="plus" size={20} />
    </button>
  </div>
  <p class="stock-count font-mono">
    {stock.paints.length} {stock.paints.length === 1 ? 'tinta' : 'tintas'} · {stockBrands().length} {stockBrands().length === 1 ? 'fabricante' : 'fabricantes'}
  </p>

  <div class="stock-actions">
    <button class="btn-ghost" onclick={pickFile}>Importar CSV</button>
    <button class="btn-ghost" onclick={exportCsv} disabled={stock.paints.length === 0}>Exportar CSV</button>
    <button class="stock-template pressable" onclick={downloadTemplate}>modelo</button>
  </div>
  <input bind:this={fileInput} type="file" accept=".csv,text/csv" style="display: none;" onchange={onFileChosen} aria-label="Arquivo CSV do estoque" />

  {#if stock.paints.length > 0}
    <input
      type="search"
      class="stock-search"
      bind:value={search}
      placeholder="Buscar no meu estoque…"
      aria-label="Buscar no estoque"
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
      spellcheck="false"
    />
  {/if}

  {#if stock.paints.length === 0}
    <!-- Board Estados: estoque vazio -->
    <div class="stock-empty">
      <span class="stock-empty-art" aria-hidden="true">
        <span class="stock-empty-dash"></span>
        <PaintBottle r={224} g={218} b={200} size={64} />
      </span>
      <p class="stock-empty-title font-display">Sua estante ainda está vazia</p>
      <p class="stock-empty-hint">Cadastre a primeira tinta ou importe um CSV com tudo de uma vez.</p>
      <button class="btn-primary" style="margin-top: 14px;" onclick={openAdd}>+ Nova tinta</button>
    </div>
  {:else if filtered.length === 0}
    <div class="empty-state">
      <p class="empty-title">Nada com esse nome</p>
    </div>
  {:else}
    <div class="stock-list">
      {#each filtered as p (p.id)}
        <button class="stock-row pressable" onclick={() => openEdit(p)}>
          <span class="stock-swatch" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
          <PaintBottle r={p.r} g={p.g} b={p.b} size={42} />
          <span class="stock-text">
            <span class="stock-name">{p.name}</span>
            <span class="stock-meta font-mono">
              {p.code || 's/ código'} · {p.manufacturer}{p.volume ? ` · ${p.volume}` : ''}
            </span>
          </span>
          <span class="stock-chev"><Icon name="chevron-right" size={16} /></span>
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Form add/editar -->
<BottomSheet open={formOpen} onClose={() => (formOpen = false)} title={editingId === null ? 'Nova tinta' : 'Editar tinta'}>
  <div class="form">
    <div class="form-color">
      <span class="swatch-flat" style="width: 52px; height: 52px; background: rgb({hexToRgb(fHex).r}, {hexToRgb(fHex).g}, {hexToRgb(fHex).b});"></span>
      <div class="color-inputs">
        <span class="form-label">Cor</span>
        <div class="color-row">
          <input type="color" bind:value={fHex} class="color-swatch" aria-label="Escolher cor" />
          <input type="text" bind:value={fHex} class="hex-input font-mono" maxlength="7" aria-label="Hex" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" />
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
      {saving ? 'Salvando…' : 'Salvar'}
    </button>
    {#if editingId !== null}
      <button class="btn-ghost" style="width: 100%;" onclick={removeEditing}>Excluir do estoque</button>
    {/if}
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
  }

  .stock-back {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    align-self: flex-start;
    min-height: 40px;
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-500);
  }

  .stock-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  /* botão circular grafite (board Estoque) */
  .stock-add {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 46px;
    border-radius: 50%;
    background: var(--grafite);
    color: var(--papel);
    flex-shrink: 0;
  }

  .stock-count {
    font-size: 12px;
    color: var(--ink-500);
    margin: 4px 0 14px;
  }

  .stock-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }

  .stock-actions .btn-ghost {
    min-height: 44px;
    padding: 0 16px;
    font-size: 14px;
  }

  .stock-actions .btn-ghost:disabled {
    opacity: 0.4;
    pointer-events: none;
  }

  .stock-template {
    min-height: 44px;
    padding: 0 6px;
    font-size: 14px;
    font-weight: 500;
    color: var(--ink-500);
  }

  .stock-search {
    width: 100%;
    min-height: 48px;
    padding: 0 12px;
    font-size: 16px;
    margin-bottom: 6px;
  }

  .stock-list {
    display: flex;
    flex-direction: column;
  }

  .stock-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 64px;
    padding: 9px 0;
    border-bottom: 1px solid var(--hairline);
    text-align: left;
  }

  .stock-swatch {
    width: 48px;
    height: 42px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
    flex-shrink: 0;
  }

  .stock-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .stock-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-meta {
    font-size: 12px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stock-chev {
    display: flex;
    color: var(--hairline);
    flex-shrink: 0;
  }

  /* ── Estado vazio (board Estados) ── */
  .stock-empty {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    padding: 24px 0;
  }

  .stock-empty-art {
    display: flex;
    align-items: flex-end;
    gap: 14px;
    margin-bottom: 16px;
  }

  .stock-empty-dash {
    width: 110px;
    height: 76px;
    border: 2px dashed var(--hairline);
    border-radius: var(--radius-control);
  }

  .stock-empty-title {
    font-size: 20px;
    font-weight: 750;
    color: var(--grafite);
  }

  .stock-empty-hint {
    font-size: 14px;
    color: var(--ink-500);
    margin-top: 4px;
    max-width: 300px;
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
    border: 1px solid var(--hairline);
    border-radius: var(--radius-control);
    background: transparent;
  }

  .color-swatch:focus {
    outline: none;
    border-color: var(--laca);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--laca) 20%, transparent);
  }

  .hex-input {
    width: 104px;
    min-height: 44px;
    padding: 0 10px;
    font-size: 16px;
    text-transform: uppercase;
  }

  /* Label-etiqueta: mono, caixa alta, espaçada. */
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
    font-size: 16px;
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
    color: var(--ink-500);
    font-size: 14px;
  }

  .import-summary.ok {
    color: var(--grafite);
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
    color: var(--ink-500);
  }
</style>
