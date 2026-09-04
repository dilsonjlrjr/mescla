<script lang="ts">
  import { Move, Hand, Pipette, Edit, Undo2, Redo2, ZoomIn, ZoomOut, Maximize, FolderOpen, Save, Download, GripHorizontal, FileImage, FileText, FileJson } from 'lucide-svelte';
  import { slide, fly } from 'svelte/transition';

  interface Props {
    mode: 'view' | 'pick' | 'edit';
    canUndo: boolean;
    canRedo: boolean;
    hasImage: boolean;
    hasRegions: boolean;
    onModeChange: (mode: 'view' | 'pick' | 'edit') => void;
    onUndo: () => void;
    onRedo: () => void;
    onZoomIn: () => void;
    onZoomOut: () => void;
    onZoomFit: () => void;
    onExportPNG: () => void;
    onExportPDF: () => void;
    onExportJSON: () => void;
    onSave: () => void;
    onOpen: () => void;
    savedIndicator: boolean;
  }

  let {
    mode, canUndo, canRedo, hasImage, hasRegions,
    onModeChange, onUndo, onRedo,
    onZoomIn, onZoomOut, onZoomFit,
    onExportPNG, onExportPDF, onExportJSON,
    onSave, onOpen,
    savedIndicator,
  }: Props = $props();

  let position: 'footer' | 'floating' = $state('footer');
  let showExportMenu = $state(false);
  let dragOffset = $state({ x: 0, y: 0 });
  let isDragging = $state(false);

  function startDrag(e: MouseEvent) {
    isDragging = true;
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    dragOffset = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    };
  }

  function onDrag(e: MouseEvent) {
    if (!isDragging) return;
    const toolbar = document.querySelector('.toolbar-draggable') as HTMLElement | null;
    if (toolbar) {
      toolbar.style.left = `${e.clientX - dragOffset.x}px`;
      toolbar.style.top = `${e.clientY - dragOffset.y}px`;
    }
  }

  function stopDrag() {
    isDragging = false;
  }

  function toggleExport() {
    showExportMenu = !showExportMenu;
  }

  function closeExport() {
    showExportMenu = false;
  }
</script>

<svelte:window onmousemove={onDrag} onmouseup={stopDrag} />

{#if position === 'footer'}
  <div class="toolbar-footer">
    <div class="toolbar-group">
      <button class="tool-btn" class:active={mode === 'view'} onclick={() => onModeChange('view')} title="Mover (V)">
        <Hand size={18} />
      </button>
      <button class="tool-btn" class:active={mode === 'pick'} onclick={() => onModeChange('pick')} title="Eyedropper (I)">
        <Pipette size={18} />
      </button>
      <button class="tool-btn" class:active={mode === 'edit'} onclick={() => onModeChange('edit')} title="Editar (E)">
        <Edit size={18} />
      </button>
    </div>

    <div class="toolbar-divider"></div>

    <div class="toolbar-group">
      <button class="tool-btn" onclick={onUndo} disabled={!canUndo} title="Desfazer (Ctrl+Z)">
        <Undo2 size={18} />
      </button>
      <button class="tool-btn" onclick={onRedo} disabled={!canRedo} title="Refazer (Ctrl+Shift+Z)">
        <Redo2 size={18} />
      </button>
    </div>

    <div class="toolbar-divider"></div>

    <div class="toolbar-group">
      <button class="tool-btn" onclick={onZoomIn} title="Zoom +">
        <ZoomIn size={18} />
      </button>
      <button class="tool-btn" onclick={onZoomOut} title="Zoom -">
        <ZoomOut size={18} />
      </button>
      <button class="tool-btn" onclick={onZoomFit} title="Ajustar à tela">
        <Maximize size={18} />
      </button>
    </div>

    <div class="toolbar-spacer"></div>

    <div class="toolbar-group">
      <button class="tool-btn" onclick={onOpen} title="Abrir plano">
        <FolderOpen size={18} />
      </button>
      <button class="tool-btn" onclick={onSave} disabled={!hasImage || !hasRegions} title="Salvar plano">
        <Save size={18} />
      </button>
      <div class="export-wrapper">
        <button class="tool-btn" onclick={toggleExport} disabled={!hasImage || !hasRegions} title="Exportar">
          <Download size={18} />
        </button>
        {#if showExportMenu}
          <div class="export-menu" transition:fly={{ duration: 150, y: 10 }}>
            <button class="export-item" onclick={() => { onExportPNG(); closeExport(); }} disabled={!hasImage || !hasRegions}>
              <FileImage size={18} />
              <div class="export-info">
                <span class="export-label">PNG</span>
                <span class="export-desc">Imagem com pins e legenda</span>
              </div>
            </button>
            <button class="export-item" onclick={() => { onExportPDF(); closeExport(); }} disabled={!hasImage || !hasRegions}>
              <FileText size={18} />
              <div class="export-info">
                <span class="export-label">PDF</span>
                <span class="export-desc">Layout imprimível A4</span>
              </div>
            </button>
            <button class="export-item" onclick={() => { onExportJSON(); closeExport(); }} disabled={!hasImage || !hasRegions}>
              <FileJson size={18} />
              <div class="export-info">
                <span class="export-label">JSON</span>
                <span class="export-desc">Backup e restauração</span>
              </div>
            </button>
          </div>
        {/if}
      </div>
    </div>

    {#if savedIndicator}
      <div class="saved-indicator">✓ Salvo</div>
    {/if}

    <button class="tool-btn move-btn" onclick={() => { position = 'floating'; }} title="Mover toolbar">
      <Move size={16} />
    </button>
  </div>
{:else}
  <div class="toolbar-floating toolbar-draggable" transition:slide={{ duration: 200 }}>
    <div class="drag-handle" onmousedown={startDrag} role="button" tabindex={0} aria-label="Arrastar toolbar">
      <GripHorizontal size={14} />
    </div>

    <div class="toolbar-content">
      <div class="toolbar-group">
        <button class="tool-btn sm" class:active={mode === 'view'} onclick={() => onModeChange('view')}>
          <Hand size={16} />
        </button>
        <button class="tool-btn sm" class:active={mode === 'pick'} onclick={() => onModeChange('pick')}>
          <Pipette size={16} />
        </button>
      </div>

      <div class="toolbar-group">
        <button class="tool-btn sm" onclick={onUndo} disabled={!canUndo}>
          <Undo2 size={16} />
        </button>
        <button class="tool-btn sm" onclick={onRedo} disabled={!canRedo}>
          <Redo2 size={16} />
        </button>
      </div>

      <div class="toolbar-group">
        <button class="tool-btn sm" onclick={onZoomIn}>
          <ZoomIn size={16} />
        </button>
        <button class="tool-btn sm" onclick={onZoomOut}>
          <ZoomOut size={16} />
        </button>
      </div>

      <div class="toolbar-group">
        <button class="tool-btn sm" onclick={onSave} disabled={!hasImage || !hasRegions}>
          <Save size={16} />
        </button>
        <button class="tool-btn sm" onclick={toggleExport} disabled={!hasImage || !hasRegions}>
          <Download size={16} />
        </button>
      </div>

      <button class="tool-btn sm" onclick={() => { position = 'footer'; }} title="Voltar ao footer">
        <Move size={14} />
      </button>
    </div>

    {#if showExportMenu}
      <div class="export-menu export-menu-floating" transition:fly={{ duration: 150, y: 10 }}>
        <button class="export-item" onclick={() => { onExportPNG(); closeExport(); }} disabled={!hasImage || !hasRegions}>
          <FileImage size={16} />
          <span>PNG</span>
        </button>
        <button class="export-item" onclick={() => { onExportPDF(); closeExport(); }} disabled={!hasImage || !hasRegions}>
          <FileText size={16} />
          <span>PDF</span>
        </button>
        <button class="export-item" onclick={() => { onExportJSON(); closeExport(); }} disabled={!hasImage || !hasRegions}>
          <FileJson size={16} />
          <span>JSON</span>
        </button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .toolbar-footer {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 6px 12px;
    background: var(--papel);
    border-top: 1px solid var(--hairline);
    z-index: 10;
  }

  .toolbar-floating {
    position: fixed;
    top: 80px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    padding: 4px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: 12px;
    z-index: 100;
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  }

  .drag-handle {
    display: flex;
    justify-content: center;
    padding: 2px;
    cursor: move;
    color: var(--grafite);
    opacity: 0.5;
  }

  .drag-handle:hover { opacity: 1; }

  .toolbar-content {
    display: flex;
    gap: 2px;
  }

  .toolbar-group {
    display: flex;
    gap: 2px;
  }

  .toolbar-divider {
    width: 1px;
    height: 24px;
    background: var(--hairline);
    margin: 0 4px;
  }

  .toolbar-spacer {
    flex: 1;
  }

  .tool-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    color: var(--grafite);
    transition: background 0.15s;
  }

  .tool-btn.sm {
    width: 32px;
    height: 32px;
  }

  .tool-btn:hover { background: var(--bancada); }
  .tool-btn.active { background: var(--laca); color: #fff; }
  .tool-btn:disabled { opacity: 0.3; cursor: default; }
  .tool-btn:disabled:hover { background: transparent; }

  .move-btn {
    margin-left: 8px;
    opacity: 0.5;
  }

  .move-btn:hover { opacity: 1; }

  .saved-indicator {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-left: 12px;
    padding: 4px 8px;
    font-size: 12px;
    color: #3a7d3a;
    background: rgba(58, 125, 58, 0.1);
    border-radius: 6px;
  }

  .export-wrapper {
    position: relative;
    display: inline-flex;
  }

  .export-menu {
    position: absolute;
    bottom: 100%;
    right: 0;
    margin-bottom: 4px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    min-width: 200px;
    overflow: hidden;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    z-index: 50;
  }

  .export-menu-floating {
    bottom: auto;
    top: 100%;
    margin-top: 4px;
    margin-bottom: 0;
  }

  .export-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 14px;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    color: var(--grafite);
    transition: background 0.15s;
  }

  .export-item:hover { background: var(--bancada); }
  .export-item:disabled { opacity: 0.4; cursor: default; }
  .export-item:disabled:hover { background: transparent; }

  .export-item + .export-item {
    border-top: 1px solid var(--hairline);
  }

  .export-info {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .export-label {
    font-size: 13px;
    font-weight: 600;
  }

  .export-desc {
    font-size: 11px;
    color: var(--grafite);
    opacity: 0.6;
  }
</style>