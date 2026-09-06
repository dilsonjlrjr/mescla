<script lang="ts">
  // Busca global (⌘K) — command palette do Tintômetro: um campo grande,
  // tintas do catálogo + ações sobre a tinta escolhida + navegação.
  // Query em hex (#8A1518) busca por cor e mostra o ΔE de cada resultado.
  import PaintBottle from './PaintBottle.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import { t } from '../i18n.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock' | 'planner' | 'receitas';

  interface NavOpts {
    paintId?: number;
    targetManufacturerId?: number;
    manufacturer?: string;
    stockPrefillPaintId?: number;
  }

  interface Row {
    id: number;
    name: string;
    code?: string;
    manufacturer: string;
    productLine?: string;
    r: number;
    g: number;
    b: number;
    deltaE?: number;
  }

  interface Props {
    open: boolean;
    onClose: () => void;
    onNavigate: (view: View, opts?: number | NavOpts) => void;
  }

  let { open, onClose, onNavigate }: Props = $props();

  // $derived (não const): RG-23 — troca de idioma não pode deixar rótulo
  // parado no antigo enquanto o componente (montado uma vez em App.svelte)
  // continua vivo.
  let navActions = $derived<{ view: View; label: string }[]>([
    { view: 'home', label: t('navPergunta') },
    { view: 'planner', label: t('navPlano') },
    { view: 'catalog', label: t('navCatalogo') },
    { view: 'receitas', label: t('navReceitas') },
    { view: 'stock', label: t('navTintas') },
    { view: 'manufacturers', label: t('makers') },
    { view: 'mix', label: 'Equivalência' },
    { view: 'compare', label: 'Comparar' },
    { view: 'wheel', label: 'Roda de cor' },
  ]);

  let query = $state('');
  let paints: Row[] = $state([]);
  let highlighted = $state(0);
  let inputEl: HTMLInputElement | undefined = $state();
  let timer: ReturnType<typeof setTimeout>;

  let hexQuery = $derived(/^#?[0-9a-fA-F]{6}$/.test(query.trim()));

  let filteredNav = $derived(
    query.trim()
      ? navActions.filter(a => a.label.toLowerCase().includes(query.trim().toLowerCase()))
      : navActions
  );

  // Tinta "escolhida" pras ações: a destacada, senão a primeira da lista.
  let activePaint = $derived.by(() => {
    if (paints.length === 0) return null;
    const idx = highlighted - filteredNav.length;
    return idx >= 0 && idx < paints.length ? paints[idx] : paints[0];
  });

  let paintActions = $derived(
    activePaint
      ? [
          { kbd: '↵', label: `Gerar fórmula equivalente com ${activePaint.name}` },
          { kbd: 'E', label: `Adicionar ${activePaint.name} ao meu estoque` },
          { kbd: 'C', label: 'Comparar com outra tinta' },
        ]
      : []
  );

  // Lista plana pra navegação por teclado: nav, tintas, ações.
  let total = $derived(filteredNav.length + paints.length + paintActions.length);

  $effect(() => {
    if (!open) return;
    query = '';
    paints = [];
    highlighted = 0;
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
        if (/^#?[0-9a-fA-F]{6}$/.test(q)) {
          const n = parseInt(q.replace('#', ''), 16);
          const found = await PaintService.FindSimilar((n >> 16) & 255, (n >> 8) & 255, n & 255, 100, 6);
          paints = (found || []).map(f => ({
            id: f.paintId, name: f.name, manufacturer: f.manufacturer,
            r: f.r, g: f.g, b: f.b, deltaE: f.deltaE,
          }));
        } else {
          const found = await PaintService.SearchPaints(q);
          paints = (found || []).slice(0, 6).map(p => ({
            id: p.id, name: p.name, code: p.code, manufacturer: p.manufacturer,
            productLine: p.productLine, r: p.r, g: p.g, b: p.b,
          }));
        }
      } catch {
        paints = [];
      }
      highlighted = paints.length > 0 ? filteredNav.length : 0;
    }, 120);
  });

  function runPaintAction(k: number) {
    const p = activePaint;
    if (!p) return;
    if (k === 0) onNavigate('mix', { paintId: p.id });
    else if (k === 1) onNavigate('stock', { stockPrefillPaintId: p.id });
    else onNavigate('compare', { paintId: p.id });
    onClose();
  }

  function pick(index: number) {
    if (index < filteredNav.length) {
      onNavigate(filteredNav[index].view);
      onClose();
      return;
    }
    const pIdx = index - filteredNav.length;
    if (pIdx < paints.length) {
      // abrir ficha da tinta no Catálogo
      onNavigate('catalog', { paintId: paints[pIdx].id });
      onClose();
      return;
    }
    runPaintAction(pIdx - paints.length);
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
        <span class="kbd-mark font-mono">⌘K</span>
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          placeholder="buscar tinta"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          onkeydown={onKeydown}
          role="combobox"
          aria-expanded="true"
          aria-controls="palette-results"
        />
      </div>

      <div class="palette-results" id="palette-results" role="listbox">
        {#if filteredNav.length > 0}
          <p class="palette-group label-mono">Ir para</p>
          {#each filteredNav as a, i (a.view)}
            <button
              class="palette-row"
              class:highlighted={highlighted === i}
              role="option"
              aria-selected={highlighted === i}
              onclick={() => pick(i)}
              onmouseenter={() => (highlighted = i)}
            >
              <span class="row-label">{a.label}</span>
            </button>
          {/each}
        {/if}

        {#if paints.length > 0}
          <p class="palette-group label-mono">Tintas</p>
          {#each paints as p, j (p.id)}
            {@const i = filteredNav.length + j}
            <button
              class="palette-row paint"
              class:highlighted={highlighted === i}
              role="option"
              aria-selected={highlighted === i}
              onclick={() => pick(i)}
              onmouseenter={() => (highlighted = i)}
            >
              <span class="row-swatch" style="background: rgb({p.r}, {p.g}, {p.b});"></span>
              <PaintBottle r={p.r} g={p.g} b={p.b} size={34} />
              <span class="row-main">
                <span class="row-label">{p.name}</span>
                <span class="row-meta font-mono">{p.manufacturer}{p.productLine ? ` · ${p.productLine}` : ''}</span>
              </span>
              {#if highlighted === i}
                <span class="row-open font-mono">↵&nbsp;&nbsp;abrir ficha</span>
              {:else if hexQuery && p.deltaE !== undefined}
                <span class="row-delta font-mono">ΔE {p.deltaE.toFixed(1)} da busca</span>
              {/if}
            </button>
          {/each}
        {/if}

        {#if paintActions.length > 0}
          <p class="palette-group label-mono">Ações</p>
          {#each paintActions as act, k}
            {@const i = filteredNav.length + paints.length + k}
            <button
              class="palette-row"
              class:highlighted={highlighted === i}
              role="option"
              aria-selected={highlighted === i}
              onclick={() => pick(i)}
              onmouseenter={() => (highlighted = i)}
            >
              <span class="kbd-chip font-mono">{act.kbd}</span>
              <span class="row-label">{act.label}</span>
            </button>
          {/each}
        {/if}

        {#if total === 0}
          <p class="palette-empty">Nada com esse nome. Tente o código do pote (ex.: 70.951) ou um hex (#8A1518).</p>
        {/if}
      </div>

      <div class="palette-foot font-mono">↑↓ navegar&nbsp;&nbsp;&nbsp;↵ abrir&nbsp;&nbsp;&nbsp;esc fechar</div>
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
    background: color-mix(in srgb, var(--bancada) 82%, transparent);
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 15vh;
  }

  .palette {
    width: min(760px, calc(100vw - 48px));
    background: var(--papel);
    border-radius: var(--radius-surface);
    box-shadow: var(--shadow-lg);
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
    gap: 18px;
    padding: 0 26px;
    height: 76px;
    border-bottom: 1px solid var(--hairline);
  }

  .kbd-mark {
    font-size: 13px;
    color: var(--text-3);
    flex-shrink: 0;
  }

  .palette-input input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    box-shadow: none;
    font-family: var(--font-display);
    font-size: 26px;
    font-weight: 640;
    color: var(--grafite);
  }

  .palette-input input::placeholder {
    color: var(--text-3);
  }

  .palette-results {
    max-height: 420px;
    overflow-y: auto;
    padding: 10px 14px 14px;
  }

  .palette-group {
    padding: 12px 12px 7px;
  }

  .palette-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 10px 12px;
    border-radius: var(--radius-control);
    text-align: left;
    color: var(--grafite);
  }

  .palette-row.highlighted {
    background: color-mix(in srgb, var(--laca) 7%, transparent);
  }

  .row-swatch {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-control);
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px var(--color-neutral-800);
  }

  .row-main {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    flex: 1;
  }

  .row-label {
    font-size: 14.5px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .row-meta {
    font-size: 11.5px;
    color: var(--text-2);
  }

  .row-open {
    margin-left: auto;
    flex-shrink: 0;
    font-size: 11.5px;
    color: var(--laca-deep);
  }

  .row-delta {
    margin-left: auto;
    flex-shrink: 0;
    font-size: 11.5px;
    color: var(--text-2);
  }

  .kbd-chip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-control);
    font-size: 11.5px;
    color: var(--text-2);
    flex-shrink: 0;
  }

  .palette-empty {
    padding: 26px 14px;
    text-align: center;
    font-size: 13px;
    color: var(--text-2);
  }

  .palette-foot {
    padding: 11px 26px;
    border-top: 1px solid var(--hairline);
    font-size: 11px;
    color: var(--text-3);
  }
</style>
