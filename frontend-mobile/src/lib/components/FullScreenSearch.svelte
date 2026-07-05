<script lang="ts">
  // Busca de tinta em TELA CHEIA — nunca dropdown: com o teclado virtual
  // aberto sobram ~150px entre campo e teclado, inutilizável pra 11.932 itens.
  // Padrão iFood/Spotify: campo no topo com foco automático, lista embaixo,
  // back/✕ cancela (integrado ao histórico via nav.pushLayer).
  import Icon from './Icon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { pushLayer } from '../nav.svelte';
  import { searchPaints, type Paint } from '../services/catalog';
  import { shelf } from '../services/shelf.svelte';

  interface Props {
    open: boolean;
    onSelect: (paint: Paint) => void;
    onClose: () => void;
    placeholder?: string;
    /** ids das últimas tintas usadas — mostradas antes de digitar */
    recentIds?: number[];
  }

  let { open, onSelect, onClose, placeholder = 'Nome ou código da tinta…', recentIds = [] }: Props = $props();

  let query = $state('');
  let input: HTMLInputElement | undefined = $state();
  let closeLayer: (() => void) | null = null;

  $effect(() => {
    if (open && !closeLayer) {
      query = '';
      closeLayer = pushLayer(() => {
        closeLayer = null;
        onClose();
      });
      // foco após a animação de entrada — abre o teclado
      setTimeout(() => input?.focus(), 80);
    } else if (!open && closeLayer) {
      const c = closeLayer;
      closeLayer = null;
      c();
    }
  });

  // Debounce leve: 11k itens filtram rápido, mas evita trabalho por tecla.
  let debounced = $state('');
  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    const q = query;
    clearTimeout(timer);
    timer = setTimeout(() => (debounced = q), 120);
  });

  import { paintById, allPaints } from '../services/catalog';

  let results = $derived.by(() => {
    if (debounced.trim()) return searchPaints(debounced, { limit: 50 });
    // Pré-digitação: recentes primeiro, depois tintas das marcas da estante.
    const recents = recentIds.map(id => paintById(id)).filter((p): p is Paint => !!p);
    const seen = new Set(recents.map(p => p.id));
    const fromShelf: Paint[] = [];
    if (shelf.manufacturerIds.length) {
      for (const p of allPaints()) {
        if (fromShelf.length >= 30) break;
        if (!seen.has(p.id) && shelf.manufacturerIds.includes(p.manufacturerId)) fromShelf.push(p);
      }
    }
    return [...recents, ...fromShelf].slice(0, 30);
  });

  function pick(p: Paint) {
    if (navigator.vibrate) navigator.vibrate(10);
    onSelect(p);
    onClose();
  }
</script>

{#if open}
  <div class="fss" role="dialog" aria-modal="true" aria-label="Buscar tinta">
    <div class="fss-bar">
      <span class="fss-icon"><Icon name="search" size={19} /></span>
      <input
        bind:this={input}
        bind:value={query}
        type="search"
        {placeholder}
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        enterkeyhint="search"
      />
      <button class="fss-cancel pressable" onclick={onClose}>Cancelar</button>
    </div>

    <div class="fss-results">
      {#if results.length === 0}
        <div class="empty-state">
          <span class="empty-icon"><Icon name="search-off" size={36} /></span>
          <p class="empty-title">Nada com esse nome</p>
          <p class="empty-hint">Tente o código do pote (ex.: 70.951) ou só parte do nome.</p>
        </div>
      {:else}
        {#if !debounced.trim() && results.length > 0}
          <p class="fss-caption">{recentIds.length ? 'Recentes e da sua estante' : 'Da sua estante'}</p>
        {/if}
        {#each results as p (p.id)}
          <button class="fss-row pressable" onclick={() => pick(p)}>
            <PaintBottle r={p.r} g={p.g} b={p.b} size={44} />
            <span class="fss-row-text">
              <span class="fss-row-name">{p.name}</span>
              <span class="fss-row-meta">
                <span class="font-mono">{p.code}</span> · {p.manufacturer}
              </span>
            </span>
          </button>
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  .fss {
    position: fixed;
    inset: 0;
    z-index: 70;
    display: flex;
    flex-direction: column;
    background: var(--ink-950);
    padding-top: var(--safe-top);
    animation: fade-in 0.15s ease both;
  }

  .fss-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    border-bottom: 1px solid var(--ink-700);
    background: var(--ink-900);
  }

  .fss-icon {
    color: var(--ink-500);
    display: flex;
    flex-shrink: 0;
  }

  .fss-bar input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    font: inherit;
    font-size: 16px; /* <16px dispara zoom automático */
    color: var(--ink-100);
  }

  .fss-bar input::placeholder {
    color: var(--ink-500);
  }

  .fss-cancel {
    flex-shrink: 0;
    min-height: 44px;
    padding: 0 6px;
    font-size: 15px;
    font-weight: 500;
    color: var(--lacquer-deep);
  }

  .fss-results {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 8px 12px calc(16px + var(--safe-bottom));
  }

  .fss-caption {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--ink-500);
    padding: 8px 8px 6px;
  }

  .fss-row {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    min-height: 64px;
    padding: 8px 8px;
    border-radius: 10px;
    text-align: left;
  }

  .fss-row:active {
    background: var(--ink-800);
  }

  .fss-row-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .fss-row-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .fss-row-meta {
    font-size: 13px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
