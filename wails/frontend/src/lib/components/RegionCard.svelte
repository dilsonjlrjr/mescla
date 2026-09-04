<script lang="ts">
  import type { ColorMatchDTO } from '../../../bindings/paint-match-ai/models';
  import type { ManufacturerDTO } from '../../../bindings/paint-match-ai/models';
  import Icon from './Icon.svelte';
  import DeltaBadge from './DeltaBadge.svelte';

  interface PaintingRegion {
    id: number;
    x: number; y: number;
    r: number; g: number; b: number;
    hex: string;
    regionName: string;
    note: string;
    targetMfrId: number;
    match: ColorMatchDTO | null;
    allMatches: ColorMatchDTO[];
    recipe: any;
  }

  interface Props {
    region: PaintingRegion;
    manufacturers: ManufacturerDTO[];
    isStockMatch: boolean;
    selected: boolean;
    onSelect: (id: number) => void;
    onRemove: (id: number) => void;
    onUpdateName: (id: number, name: string) => void;
    onUpdateNote: (id: number, note: string) => void;
    onChangeMfr: (region: PaintingRegion, mfrId: number) => void;
  }

  let {
    region: r, manufacturers, isStockMatch, selected,
    onSelect, onRemove, onUpdateName, onUpdateNote, onChangeMfr,
  }: Props = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="region-card"
  class:selected
  onclick={() => onSelect(r.id)}
>
  <div class="region-top">
    <button
      class="swatch-btn"
      style:background={r.hex}
      title="Destacar no canvas"
      onclick={(e) => { e.stopPropagation(); onSelect(r.id); }}
    ></button>
    <span class="region-id">#{r.id}</span>
    <span class="region-hex">{r.hex}</span>
    {#if isStockMatch}
      <span class="stock-badge">no meu estoque</span>
    {/if}
    <button class="icon-btn danger" onclick={(e) => { e.stopPropagation(); onRemove(r.id); }} title="Remover">
      <Icon name="trash" size={14} />
    </button>
  </div>

  {#if r.match}
    <div class="match-line">
      <span class="swatch-sm" style:background={r.hex}></span>
      <strong>{r.match.manufacturer} {r.match.name}</strong>
      <span class="meta">({r.match.code})</span>
      <DeltaBadge deltaE={r.match.deltaE} />
    </div>
  {/if}

  {#if r.recipe?.ingredients?.length}
    <div class="recipe-line">
      <Icon name="flask" size={12} />
      <strong>Receita:</strong>
      {#each r.recipe.ingredients as ing}
        <span class="ingredient">{ing.name} ({ing.percentage.toFixed(0)}%)</span>
      {/each}
      {#if r.recipe.tips?.length}
        <details class="recipe-tips">
          <summary>Dicas</summary>
          {#each r.recipe.tips as tip}
            <span class="tip">• {tip}</span>
          {/each}
        </details>
      {/if}
    </div>
  {/if}

  <select
    class="field-sm"
    value={r.targetMfrId}
    onchange={(e: Event) => onChangeMfr(r, Number((e.target as HTMLSelectElement).value))}
  >
    <option value={0}>Melhor match (todos)</option>
    {#each manufacturers as m (m.id)}
      <option value={m.id}>{m.name}</option>
    {/each}
  </select>

  <input
    type="text"
    class="field-sm"
    placeholder="Nome da região (ex: ombro esquerdo)"
    value={r.regionName}
    oninput={(e: Event) => onUpdateName(r.id, (e.target as HTMLInputElement).value)}
  />

  <textarea
    class="field-sm"
    rows={2}
    placeholder="Nota (ex: aplicar wash depois)"
    value={r.note}
    oninput={(e: Event) => onUpdateNote(r.id, (e.target as HTMLTextAreaElement).value)}
  ></textarea>
</div>

<style>
  .region-card {
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    padding: 10px;
    background: var(--papel);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .region-card:hover { border-color: var(--grafite); }
  .region-card.selected { border-color: var(--laca); border-width: 2px; }

  .region-top {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  .region-top .icon-btn { margin-left: auto; }

  .swatch-btn {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid var(--hairline);
    cursor: pointer;
    flex-shrink: 0;
    transition: transform 0.15s;
  }

  .swatch-btn:hover { transform: scale(1.15); }

  .region-id {
    font-weight: 600;
    font-size: 13px;
  }

  .region-hex {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 12px;
    color: var(--grafite);
  }

  .stock-badge {
    font-size: 10px;
    padding: 1px 6px;
    background: var(--bancada);
    border: 1px solid var(--hairline);
    border-radius: 4px;
    color: var(--grafite);
    white-space: nowrap;
  }

  .match-line {
    font-size: 12px;
    margin-bottom: 6px;
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: wrap;
  }

  .swatch-sm {
    display: inline-block;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 1px solid var(--hairline);
    flex-shrink: 0;
  }

  .recipe-line {
    font-size: 11px;
    margin-bottom: 6px;
    padding: 6px 8px;
    background: var(--bancada);
    border-radius: var(--radius-surface);
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;
  }

  .ingredient {
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: 4px;
    padding: 1px 6px;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 11px;
  }

  .recipe-tips {
    width: 100%;
    font-size: 11px;
    margin-top: 4px;
  }

  .tip {
    display: block;
    color: var(--grafite);
    opacity: 0.7;
  }

  .meta {
    font-size: 11px;
    color: var(--grafite);
    opacity: 0.6;
  }

  .field-sm {
    width: 100%;
    padding: 6px 8px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    font-size: 12px;
    font-family: inherit;
    margin-top: 4px;
    box-sizing: border-box;
  }

  .field-sm:focus { outline: none; border-color: var(--laca); }
  textarea.field-sm { resize: vertical; }
</style>
