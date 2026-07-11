<script lang="ts">
  import Textfield from '@smui/textfield';
  import Icon from './Icon.svelte';

  interface PaintOption {
    id: number;
    name: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  interface Props {
    paints: PaintOption[];
    selected: PaintOption | null;
    onSelect: (paint: PaintOption) => void;
    onClear: () => void;
    label?: string;
  }

  let { paints, selected, onSelect, onClear, label = 'Buscar tinta...' }: Props = $props();

  let query = $state('');

  let results = $derived(
    query.trim()
      ? paints
          .filter(p => {
            const q = query.toLowerCase();
            return p.name.toLowerCase().includes(q) || p.manufacturer.toLowerCase().includes(q);
          })
          .slice(0, 6)
      : []
  );

  function pick(p: PaintOption) {
    onSelect(p);
    query = '';
  }
</script>

{#if selected}
  <div class="paint-search-chip">
    <span class="swatch-flat" style="width: 40px; height: 40px; background: rgb({selected.r}, {selected.g}, {selected.b});"></span>
    <div style="flex: 1; min-width: 0;">
      <div class="font-semibold text-sm text-white truncate">{selected.name}</div>
      <div class="font-mono" style="font-size: 10.5px; color: var(--ink-500);">{selected.manufacturer}</div>
    </div>
    <button class="paint-search-swap" onclick={onClear}>Trocar</button>
  </div>
{:else}
  <div class="paint-search" style="position: relative;">
    <Textfield variant="outlined" bind:value={query} {label} style="width: 100%;">
      {#snippet leadingIcon()}
        <span class="mdc-text-field__icon mdc-text-field__icon--leading" style="color: var(--ink-500); display: flex;"><Icon name="search" size={17} /></span>
      {/snippet}
    </Textfield>

    {#if query.trim()}
      <div class="paint-search-results panel">
        {#if results.length > 0}
          {#each results as p (p.id)}
            <button class="paint-search-result" onclick={() => pick(p)}>
              <span class="swatch-flat" style="width: 30px; height: 30px; background: rgb({p.r}, {p.g}, {p.b});"></span>
              <div style="flex: 1; min-width: 0;">
                <div class="font-medium text-sm text-white truncate">{p.name}</div>
                <div class="font-mono" style="font-size: 10.5px; color: var(--ink-500);">{p.manufacturer}</div>
              </div>
            </button>
          {/each}
        {:else}
          <div style="padding: 14px; font-size: 12.5px; color: var(--ink-500);">Nenhuma tinta encontrada</div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .paint-search-results {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 5;
    max-height: 260px;
    overflow-y: auto;
    padding: 6px;
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.18);
  }

  .paint-search-result {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px;
    border: none;
    border-radius: 6px;
    background: transparent;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s ease;
  }

  .paint-search-result:hover {
    background: var(--ink-800);
  }

  .paint-search-chip {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-control);
    background: var(--ink-900);
  }

  .paint-search-swap {
    flex-shrink: 0;
    border: none;
    background: transparent;
    color: var(--lacquer-tint);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    padding: 6px 8px;
  }

  .paint-search-swap:hover {
    text-decoration: underline;
  }
</style>
