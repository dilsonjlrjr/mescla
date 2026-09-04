<script lang="ts">
  // Busca de tinta em TELA CHEIA — nunca dropdown: com o teclado virtual
  // aberto sobram ~150px entre campo e teclado, inutilizável pra 11.932 itens.
  // Grid de 2 colunas com amostra chapada + garrafinha, resultados ordenados
  // por matiz (arquivo de museu). Back/fechar cancela (nav.pushLayer).
  import PaintBottle from './PaintBottle.svelte';
  import { pushLayer } from '../nav.svelte';
  import { searchPaints, paintById, allPaints, type Paint } from '../services/catalog';
  import { shelf } from '../services/shelf.svelte';
  import { hueOf } from '../ui';

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

  let searching = $derived(debounced.trim().length > 0);

  let results = $derived.by(() => {
    if (searching) {
      // Ordenado por matiz: a lista vira uma cartela contínua de cor.
      return searchPaints(debounced, { limit: 60 }).slice().sort(
        (a, b) => hueOf(a.r, a.g, a.b) - hueOf(b.r, b.g, b.b)
      );
    }
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
      <input
        bind:this={input}
        bind:value={query}
        type="search"
        {placeholder}
        aria-label="Buscar tinta"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        enterkeyhint="search"
      />
      <button class="fss-cancel pressable" onclick={onClose}>fechar</button>
    </div>

    <div class="fss-results">
      {#if results.length === 0}
        <div class="empty-state">
          <p class="empty-title">Nada com esse nome</p>
          <p class="empty-hint">Tente o código do pote (ex.: 70.951) ou só parte do nome.</p>
        </div>
      {:else}
        <p class="fss-caption section-label">
          {#if searching}
            {results.length} {results.length === 1 ? 'resultado' : 'resultados'} · ordenado por matiz
          {:else}
            {recentIds.length ? 'Recentes e da sua estante' : 'Da sua estante'}
          {/if}
        </p>
        <div class="fss-grid">
          {#each results as p (p.id)}
            <button class="fss-card pressable" onclick={() => pick(p)}>
              <span class="fss-card-top">
                <span class="fss-card-swatch" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
                <PaintBottle r={p.r} g={p.g} b={p.b} size={64} />
              </span>
              <span class="fss-card-name">{p.name}</span>
              <span class="fss-card-meta font-mono">{p.code} · {p.manufacturer}</span>
            </button>
          {/each}
        </div>
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
    background: var(--bancada);
    padding-top: var(--safe-top);
    animation: fade-in 0.15s ease both;
  }

  .fss-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 14px 16px 10px;
  }

  .fss-bar input {
    flex: 1;
    min-width: 0;
    min-height: 50px;
    padding: 0 14px;
    font: inherit;
    font-size: 16px; /* <16px dispara zoom automático */
  }

  .fss-cancel {
    flex-shrink: 0;
    min-height: 44px;
    padding: 0 6px;
    font-size: 15px;
    font-weight: 500;
    color: var(--ink-500);
  }

  .fss-results {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 4px 16px calc(16px + var(--safe-bottom));
  }

  .fss-caption {
    padding: 8px 0 10px;
  }

  .fss-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px 14px;
  }

  .fss-card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    min-width: 0;
    text-align: left;
  }

  .fss-card-top {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    width: 100%;
    margin-bottom: 7px;
  }

  .fss-card-swatch {
    flex: 1;
    min-width: 0;
    aspect-ratio: 5 / 4;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
  }

  .fss-card-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--grafite);
    line-height: 1.25;
    max-width: 100%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .fss-card-meta {
    font-size: 11.5px;
    color: var(--ink-500);
    max-width: 100%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
