<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    visible: boolean;
    hasImage: boolean;
    hasRegions: boolean;
    onExportPNG: () => void;
    onExportPDF: () => void;
    onExportJSON: () => void;
    onClose: () => void;
  }

  let { visible, hasImage, hasRegions, onExportPNG, onExportPDF, onExportJSON, onClose }: Props = $props();
</script>

{#if visible}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="export-overlay" onclick={onClose}></div>
  <div class="export-menu">
    <button class="export-item" onclick={onExportPNG} disabled={!hasImage || !hasRegions}>
      <Icon name="image" size={18} />
      <div class="export-info">
        <span class="export-label">PNG</span>
        <span class="export-desc">Imagem com pins e legenda</span>
      </div>
    </button>
    <button class="export-item" onclick={onExportPDF} disabled={!hasImage || !hasRegions}>
      <Icon name="download" size={18} />
      <div class="export-info">
        <span class="export-label">PDF</span>
        <span class="export-desc">Layout imprimível A4</span>
      </div>
    </button>
    <button class="export-item" onclick={onExportJSON} disabled={!hasImage || !hasRegions}>
      <Icon name="box" size={18} />
      <div class="export-info">
        <span class="export-label">JSON</span>
        <span class="export-desc">Backup e restauração</span>
      </div>
    </button>
  </div>
{/if}

<style>
  .export-overlay {
    position: fixed;
    inset: 0;
    z-index: 40;
  }

  .export-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    min-width: 200px;
    z-index: 41;
    overflow: hidden;
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
