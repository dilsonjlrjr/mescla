<script lang="ts">
  // Autocomplete de tinta (Tintômetro): input raio 8 + dropdown papel com
  // hairline; busca por nome, código ou marca no catálogo já carregado.
  import { paintMatches } from '../ui';

  interface PaintOption {
    id: number;
    name: string;
    code?: string;
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

  let { paints, selected, onSelect, onClear, label = 'Buscar tinta' }: Props = $props();

  let query = $state('');

  let results = $derived(
    query.trim()
      ? paints.filter(p => paintMatches(p, query)).slice(0, 6)
      : []
  );

  function pick(p: PaintOption) {
    onSelect(p);
    query = '';
  }
</script>

{#if selected}
  <div class="paint-chip">
    <span class="swatch-flat" style="width: 40px; height: 40px; background: rgb({selected.r}, {selected.g}, {selected.b});"></span>
    <div style="flex: 1; min-width: 0;">
      <div class="chip-name">{selected.name}</div>
      <div class="chip-meta font-mono">{selected.manufacturer}</div>
    </div>
    <button class="chip-swap" onclick={onClear}>Trocar</button>
  </div>
{:else}
  <div class="paint-search">
    <input
      type="search"
      bind:value={query}
      placeholder={label}
      aria-label={label}
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
      spellcheck="false"
    />

    {#if query.trim()}
      <div class="search-results">
        {#if results.length > 0}
          {#each results as p (p.id)}
            <button class="search-result" onclick={() => pick(p)}>
              <span class="swatch-flat" style="width: 30px; height: 30px; background: rgb({p.r}, {p.g}, {p.b});"></span>
              <div style="flex: 1; min-width: 0;">
                <div class="result-name">{p.name}</div>
                <div class="result-meta font-mono">{p.code ? `${p.code} · ` : ''}{p.manufacturer}</div>
              </div>
            </button>
          {/each}
        {:else}
          <div class="search-empty">Nenhuma tinta encontrada</div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .paint-search {
    position: relative;
  }

  .paint-search input {
    width: 100%;
    height: 46px;
    padding: 0 16px;
    font: inherit;
    font-size: 14px;
  }

  .search-results {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 5;
    max-height: 260px;
    overflow-y: auto;
    padding: 6px;
    background: var(--papel);
    border: 1px solid var(--hairline);
    border-radius: var(--radius-surface);
    box-shadow: var(--shadow-md);
  }

  .search-result {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px;
    border: none;
    border-radius: var(--radius-control);
    background: transparent;
    cursor: pointer;
    text-align: left;
    font: inherit;
    transition: background 0.15s ease;
  }

  .search-result:hover {
    background: color-mix(in srgb, var(--laca) 7%, transparent);
  }

  .result-name {
    font-size: 13.5px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-meta {
    font-size: 10.5px;
    color: var(--text-2);
  }

  .search-empty {
    padding: 14px;
    font-size: 12.5px;
    color: var(--text-2);
  }

  .paint-chip {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-control);
    background: var(--papel);
  }

  .chip-name {
    font-size: 13.5px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .chip-meta {
    font-size: 10.5px;
    color: var(--text-2);
  }

  .chip-swap {
    flex-shrink: 0;
    border: none;
    background: transparent;
    color: var(--laca-deep);
    font: inherit;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    padding: 6px 8px;
  }

  .chip-swap:hover {
    text-decoration: underline;
  }
</style>
