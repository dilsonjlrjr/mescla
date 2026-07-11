<script lang="ts">
  // Sheet de seleção de marcas — nunca <select> nativo (35 itens, sem busca).
  // Dois modos: 'shelf' (multi, com persistência da estante — passo 2 da
  // Mesclar) e 'single' (filtro do Catálogo). "Minhas marcas" fixas no topo.
  import BottomSheet from './BottomSheet.svelte';
  import Icon from './Icon.svelte';
  import { allManufacturers, type Manufacturer } from '../services/catalog';
  import { shelf, toggleShelf } from '../services/shelf.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    mode: 'shelf' | 'single';
    /** modo single: marca atualmente escolhida (null = todas) */
    selectedId?: number | null;
    onPick?: (m: Manufacturer | null) => void;
  }

  let { open, onClose, mode, selectedId = null, onPick }: Props = $props();

  let filter = $state('');

  let mfrs = $derived.by(() => {
    const all = allManufacturers();
    const q = filter.trim().toLowerCase();
    return q ? all.filter(m => m.name.toLowerCase().includes(q)) : all;
  });

  let mine = $derived(mfrs.filter(m => shelf.manufacturerIds.includes(m.id)));
  let others = $derived(mfrs.filter(m => !shelf.manufacturerIds.includes(m.id)));

  function tapShelf(m: Manufacturer) {
    if (navigator.vibrate) navigator.vibrate(10);
    toggleShelf(m.id);
  }

  function tapSingle(m: Manufacturer | null) {
    if (navigator.vibrate) navigator.vibrate(10);
    onPick?.(m);
    onClose();
  }
</script>

<BottomSheet {open} {onClose} height="tall" title={mode === 'shelf' ? 'Tenho tintas de…' : 'Filtrar por marca'}>
  <div class="brand-filter">
    <Icon name="search" size={16} />
    <input bind:value={filter} type="search" placeholder="Buscar marca…" autocomplete="off" />
  </div>

  {#if mode === 'single'}
    <button class="brand-row pressable" class:checked={selectedId === null} onclick={() => tapSingle(null)}>
      <span class="brand-name">Todas as marcas</span>
      {#if selectedId === null}<span class="brand-check"><Icon name="check" size={18} /></span>{/if}
    </button>
  {/if}

  {#if mine.length > 0}
    <p class="brand-caption">Minhas marcas</p>
    {#each mine as m (m.id)}
      {#if mode === 'shelf'}
        <button class="brand-row pressable checked" onclick={() => tapShelf(m)}>
          <span class="brand-name">{m.name}</span>
          <span class="brand-count font-mono">{m.paintCount}</span>
          <span class="brand-check"><Icon name="check" size={18} /></span>
        </button>
      {:else}
        <button class="brand-row pressable" class:checked={selectedId === m.id} onclick={() => tapSingle(m)}>
          <span class="brand-name">{m.name}</span>
          <span class="brand-count font-mono">{m.paintCount}</span>
          {#if selectedId === m.id}<span class="brand-check"><Icon name="check" size={18} /></span>{/if}
        </button>
      {/if}
    {/each}
  {/if}

  <p class="brand-caption">{mine.length > 0 ? 'Outras marcas' : 'Marcas'}</p>
  {#each others as m (m.id)}
    {#if mode === 'shelf'}
      <button class="brand-row pressable" onclick={() => tapShelf(m)}>
        <span class="brand-name">{m.name}</span>
        <span class="brand-count font-mono">{m.paintCount}</span>
      </button>
    {:else}
      <button class="brand-row pressable" class:checked={selectedId === m.id} onclick={() => tapSingle(m)}>
        <span class="brand-name">{m.name}</span>
        <span class="brand-count font-mono">{m.paintCount}</span>
        {#if selectedId === m.id}<span class="brand-check"><Icon name="check" size={18} /></span>{/if}
      </button>
    {/if}
  {/each}

  {#if mode === 'shelf'}
    <div class="brand-apply">
      <button class="btn-primary" onclick={onClose}>
        {shelf.manufacturerIds.length > 0
          ? `Usar ${shelf.manufacturerIds.length} ${shelf.manufacturerIds.length === 1 ? 'marca' : 'marcas'}`
          : 'Fechar'}
      </button>
    </div>
  {/if}
</BottomSheet>

<style>
  .brand-filter {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 48px;
    padding: 4px 14px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    color: var(--ink-500);
    margin-bottom: 12px;
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .brand-filter:focus-within {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .brand-filter input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    box-shadow: none;
    background: transparent;
    font: inherit;
    font-size: 16px;
    color: var(--ink-100);
  }

  .brand-caption {
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--ink-500);
    padding: 12px 4px 6px;
  }

  .brand-row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 52px;
    padding: 6px 12px;
    border-radius: var(--radius-control);
    text-align: left;
  }

  .brand-row:active {
    background: var(--ink-800);
  }

  .brand-row.checked {
    background: color-mix(in srgb, var(--lacquer) 8%, transparent);
  }

  .brand-name {
    flex: 1;
    min-width: 0;
    font-size: 16px;
    font-weight: 500;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .brand-count {
    font-size: 12px;
    color: var(--ink-500);
    flex-shrink: 0;
  }

  .brand-check {
    color: var(--lacquer);
    display: flex;
    flex-shrink: 0;
  }

  .brand-apply {
    position: sticky;
    bottom: 0;
    padding: 12px 0 4px;
    background: linear-gradient(transparent, var(--ink-900) 30%);
  }
</style>
