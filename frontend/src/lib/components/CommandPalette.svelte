<script lang="ts">
  // Busca global (⌘K) — padrão command palette: um campo, duas fontes de
  // resultado (navegação + tintas do catálogo). Escolher uma tinta cai direto
  // na Equivalência com ela pré-selecionada — o caminho nº 1 do app em 2 teclas.
  import Icon from './Icon.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface Paint {
    id: number;
    name: string;
    code?: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  interface Props {
    open: boolean;
    onClose: () => void;
    onNavigate: (view: View, paintId?: number) => void;
  }

  let { open, onClose, onNavigate }: Props = $props();

  const actions: { view: View; label: string; hint: string; icon: 'home' | 'flask' | 'wheel' | 'grid' | 'box' | 'pipette' | 'swap' | 'building' }[] = [
    { view: 'mix', label: 'Equivalência', hint: 'receita em outra marca', icon: 'flask' },
    { view: 'catalog', label: 'Catálogo', hint: 'todas as tintas', icon: 'grid' },
    { view: 'stock', label: 'Meu estoque', hint: 'as tintas que você tem', icon: 'box' },
    { view: 'wheel', label: 'Roda de cor', hint: 'harmonias e rampas', icon: 'wheel' },
    { view: 'color-search', label: 'Buscar cor', hint: 'da cor exata pra tinta', icon: 'pipette' },
    { view: 'compare', label: 'Comparar', hint: 'tintas lado a lado', icon: 'swap' },
    { view: 'manufacturers', label: 'Marcas', hint: 'quem fabrica o quê', icon: 'building' },
    { view: 'home', label: 'Início', hint: 'visão geral', icon: 'home' },
  ];

  let query = $state('');
  let paints: Paint[] = $state([]);
  let highlighted = $state(0);
  let inputEl: HTMLInputElement | undefined = $state();
  let timer: ReturnType<typeof setTimeout>;

  let filteredActions = $derived(
    query.trim()
      ? actions.filter(a => a.label.toLowerCase().includes(query.trim().toLowerCase()))
      : actions
  );

  // Lista plana pra navegação por teclado: ações primeiro, tintas depois.
  let total = $derived(filteredActions.length + paints.length);

  $effect(() => {
    if (!open) return;
    query = '';
    paints = [];
    highlighted = 0;
    // foco depois do render do overlay
    setTimeout(() => inputEl?.focus(), 30);
  });

  $effect(() => {
    const q = query.trim();
    clearTimeout(timer);
    if (q.length < 2) {
      paints = [];
      return;
    }
    timer = setTimeout(async () => {
      try {
        const found = await PaintService.SearchPaints(q);
        paints = (found || []).slice(0, 8);
      } catch {
        paints = [];
      }
      highlighted = 0;
    }, 120);
  });

  function pick(index: number) {
    if (index < filteredActions.length) {
      onNavigate(filteredActions[index].view);
    } else {
      const paint = paints[index - filteredActions.length];
      if (paint) onNavigate('mix', paint.id);
    }
    onClose();
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      highlighted = Math.min(highlighted + 1, total - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      highlighted = Math.max(highlighted - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (total > 0) pick(highlighted);
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div class="palette-overlay" onclick={onClose}>
    <div class="palette" role="dialog" aria-label="Busca global" tabindex="-1" onclick={e => e.stopPropagation()}>
      <div class="palette-input">
        <Icon name="search" size={17} />
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          placeholder="Tinta, código ou tela…"
          autocomplete="off"
          spellcheck="false"
          onkeydown={onKeydown}
          role="combobox"
          aria-expanded="true"
          aria-controls="palette-results"
        />
        <kbd class="font-mono">esc</kbd>
      </div>

      <div class="palette-results" id="palette-results" role="listbox">
        {#if filteredActions.length > 0}
          <p class="palette-group font-mono">ir para</p>
          {#each filteredActions as a, i (a.view)}
            <button
              class="palette-row"
              class:highlighted={highlighted === i}
              role="option"
              aria-selected={highlighted === i}
              onclick={() => pick(i)}
              onmouseenter={() => (highlighted = i)}
            >
              <span class="row-icon"><Icon name={a.icon} size={16} /></span>
              <span class="row-label">{a.label}</span>
              <span class="row-hint">{a.hint}</span>
            </button>
          {/each}
        {/if}

        {#if paints.length > 0}
          <p class="palette-group font-mono">tintas</p>
          {#each paints as p, j (p.id)}
            {@const i = filteredActions.length + j}
            <button
              class="palette-row"
              class:highlighted={highlighted === i}
              role="option"
              aria-selected={highlighted === i}
              onclick={() => pick(i)}
              onmouseenter={() => (highlighted = i)}
            >
              <span class="row-swatch" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
              <span class="row-label">{p.name}</span>
              <span class="row-hint"><span class="font-mono">{p.code ?? ''}</span> {p.manufacturer}</span>
            </button>
          {/each}
        {/if}

        {#if total === 0}
          <p class="palette-empty">Nada com esse nome. Tente o código do pote (ex.: 70.951).</p>
        {/if}
      </div>

      <div class="palette-foot font-mono">↑↓ navegar · enter abrir · esc fechar</div>
    </div>
  </div>
{/if}

<style>
  button {
    font: inherit;
    color: inherit;
    border: none;
    background: none;
    cursor: pointer;
  }

  .palette-overlay {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 12vh;
  }

  .palette {
    width: min(580px, calc(100vw - 48px));
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-surface);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.25);
    overflow: hidden;
    animation: palette-in 0.16s cubic-bezier(0.16, 1, 0.3, 1) both;
  }

  @keyframes palette-in {
    from { opacity: 0; transform: translateY(-8px) scale(0.985); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .palette-input {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 16px;
    height: 54px;
    border-bottom: 1px solid var(--ink-700);
    color: var(--ink-500);
  }

  .palette-input input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    box-shadow: none; /* a baseline global de campo (:focus com anel) não vale aqui — o "campo" é a própria barra do palette */
    font: inherit;
    font-size: 15px;
    color: var(--ink-100);
  }

  .palette-input kbd {
    font-size: 10.5px;
    padding: 3px 7px;
    border-radius: 5px;
    background: var(--ink-800);
    color: var(--ink-500);
  }

  .palette-results {
    max-height: 340px;
    overflow-y: auto;
    padding: 8px;
  }

  .palette-group {
    font-size: 10px;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--ink-500);
    padding: 8px 10px 5px;
  }

  .palette-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 9px 10px;
    border-radius: var(--radius-control);
    text-align: left;
    color: var(--ink-300);
  }

  .palette-row.highlighted {
    background: var(--ink-800);
    color: var(--ink-100);
  }

  .row-icon {
    display: flex;
    color: var(--ink-500);
    flex-shrink: 0;
  }

  .palette-row.highlighted .row-icon {
    color: var(--lacquer-deep);
  }

  .row-swatch {
    width: 22px;
    height: 22px;
    border-radius: 6px;
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.2);
  }

  .row-label {
    font-size: 14px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .row-hint {
    margin-left: auto;
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  .palette-empty {
    padding: 24px 14px;
    text-align: center;
    font-size: 13px;
    color: var(--ink-500);
  }

  .palette-foot {
    padding: 9px 16px;
    border-top: 1px solid var(--ink-700);
    font-size: 10.5px;
    color: var(--ink-500);
  }
</style>
