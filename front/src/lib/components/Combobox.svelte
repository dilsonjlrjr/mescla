<script lang="ts">
  // Seletor único padrão (substitui os "chips" de fabricante/tipo, que estouram a
  // tela quando a lista cresce). Gatilho + popover com busca, pensado pra toque
  // (iPad) e teclado.
  import { untrack } from 'svelte';
  import { norm } from '../texto';

  interface ComboOption {
    id: number;
    name: string;
    hint?: string;
  }

  interface Props {
    options: ComboOption[];
    value: number | null;
    onchange: (id: number | null) => void;
    placeholder: string;
    searchPlaceholder: string;
    emptyText: string;
    clearLabel?: string;
    ariaLabel?: string;
    disabled?: boolean;
    width?: string;
    moreText?: (n: number) => string;
  }

  let {
    options,
    value,
    onchange,
    placeholder,
    searchPlaceholder,
    emptyText,
    clearLabel,
    ariaLabel,
    disabled = false,
    width = '100%',
    moreText,
  }: Props = $props();

  // Teto de opções renderizadas por vez — lista pode ter milhares de tintas.
  const MAX_VISIBLE = 300;

  let wrapperEl: HTMLDivElement | undefined = $state();
  let triggerEl: HTMLButtonElement | undefined = $state();
  let searchEl: HTMLInputElement | undefined = $state();
  let listEl: HTMLDivElement | undefined = $state();

  let open = $state(false);
  let search = $state('');
  let openUpward = $state(false);
  // Posição em coordenada de tela, recalculada a cada abertura.
  let popStyle = $state('');
  let highlight = $state(0);

  // A lista sai do lugar onde foi declarada e vira filha de <body>. Motivo: o
  // card do modal de T4 anima com `animation: … both`, e o último keyframe
  // (`transform: scale(1)`) fica aplicado para sempre. Transform diferente de
  // `none` cria bloco de contenção para descendente `position: fixed` — a lista
  // passava a ser posicionada em relação ao card, não à janela, e o
  // `overflow: hidden` do card a cortava por inteiro. Fora do card, `fixed`
  // volta a significar coordenada de tela (D-017).
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return {
      destroy() {
        node.remove();
      },
    };
  }


  let selected = $derived(options.find((o) => o.id === value) ?? null);

  let filtered = $derived.by(() => {
    const q = norm(search.trim());
    if (!q) return options;
    return options.filter((o) => norm(o.name).includes(q));
  });

  let visible = $derived(filtered.slice(0, MAX_VISIBLE));
  let hiddenCount = $derived(Math.max(0, filtered.length - MAX_VISIBLE));

  // Linha "limpar" some durante a busca, a não ser que o próprio texto dela case.
  let showClear = $derived(
    !!clearLabel && (!search.trim() || norm(clearLabel).includes(norm(search.trim())))
  );

  interface Row {
    id: number | null;
    name: string;
    hint?: string;
  }

  // Lista achatada (linha de limpar + opções visíveis) pra navegação por teclado
  // andar num único índice, sem distinguir as duas origens.
  let rows = $derived.by((): Row[] => {
    const r: Row[] = [];
    if (showClear) r.push({ id: null, name: clearLabel as string });
    for (const o of visible) r.push({ id: o.id, name: o.name, hint: o.hint });
    return r;
  });

  // O popover é `position: fixed` em coordenada de tela, não `absolute` no
  // wrapper: dentro do corpo rolável do modal de tinta um filho absoluto seria
  // cortado pelo `overflow-y: auto` da faixa. Fixo escapa do corte, mas passa a
  // não acompanhar rolagem — por isso a lista fecha ao rolar (efeito abaixo).
  function decidePosition() {
    if (!triggerEl) return;
    const rect = triggerEl.getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom - 12;
    const spaceAbove = rect.top - 12;
    // Só inverte se faltar espaço embaixo E sobrar mais em cima.
    openUpward = spaceBelow < 320 && spaceAbove > spaceBelow;
    const room = Math.max(160, Math.min(320, openUpward ? spaceAbove : spaceBelow));
    const vertical = openUpward
      ? `bottom: ${Math.round(window.innerHeight - rect.top + 6)}px;`
      : `top: ${Math.round(rect.bottom + 6)}px;`;
    popStyle = `left: ${Math.round(rect.left)}px; width: ${Math.round(rect.width)}px; ${vertical} max-height: ${Math.round(room)}px;`;
  }

  function openPopover() {
    if (disabled) return;
    decidePosition();
    search = '';
    const idx = rows.findIndex((r) => r.id === value);
    highlight = idx >= 0 ? idx : 0;
    open = true;
  }

  function closePopover(focusTrigger: boolean) {
    open = false;
    search = '';
    if (focusTrigger) triggerEl?.focus();
  }

  function pick(id: number | null) {
    onchange(id);
    closePopover(true);
  }

  function onTriggerClick() {
    if (open) closePopover(false);
    else openPopover();
  }

  function onSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      highlight = Math.min(rows.length - 1, highlight + 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      highlight = Math.max(0, highlight - 1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const row = rows[highlight];
      if (row) pick(row.id);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      closePopover(true);
    } else if (e.key === 'Tab') {
      closePopover(false);
    }
  }

  // Abrir foca a busca e rola até a seleção — só na transição fechado→aberto,
  // por isso lê `rows`/`value` com untrack (não deve reagir a cada tecla digitada).
  $effect(() => {
    if (!open) return;
    // `preventScroll`: o campo vive num popover `position: fixed`, e o foco sem
    // essa opção faz o navegador rolar o ancestral rolável pra "revelar" o campo.
    // Essa rolagem disparava o efeito de baixo e fechava a lista no mesmo quadro
    // em que ela abria — a lista parecia travada (D-016).
    searchEl?.focus({ preventScroll: true });
    untrack(() => {
      const idx = rows.findIndex((r) => r.id === value);
      if (idx < 0) return;
      const el = listEl?.querySelector(`[data-idx="${idx}"]`);
      el?.scrollIntoView({ block: 'nearest' });
    });
  });

  // Clique fora fecha — captura em `document` pra pegar o clique antes de outros
  // handlers, ignorando o que acontece dentro do próprio wrapper.
  $effect(() => {
    if (!open) return;
    function onPointerDown(e: PointerEvent) {
      if (!(e.target instanceof Node)) return;
      // A lista vive em <body> (ação `portal`), então não basta olhar o wrapper.
      if (wrapperEl?.contains(e.target) || listEl?.contains(e.target)) return;
      closePopover(false);
    }
    document.addEventListener('pointerdown', onPointerDown, true);
    return () => document.removeEventListener('pointerdown', onPointerDown, true);
  });

  // Lista fixa na tela não acompanha sozinha a rolagem de quem está atrás dela.
  // Rolar ou redimensionar REPOSICIONA a lista no campo; só fecha quando o campo
  // sai da janela. Fechar direto travava a lista no iPad, onde o teclado que sobe
  // dispara `resize` assim que a busca recebe foco (D-016).
  $effect(() => {
    if (!open) return;
    function onMove(e: Event) {
      // Rolar a própria lista de opções não mexe em nada.
      if (e.target instanceof Node && listEl?.contains(e.target)) return;
      if (!triggerEl) return;
      const rect = triggerEl.getBoundingClientRect();
      if (rect.bottom < 0 || rect.top > window.innerHeight) {
        closePopover(false);
        return;
      }
      decidePosition();
    }
    window.addEventListener('scroll', onMove, true);
    window.addEventListener('resize', onMove);
    return () => {
      window.removeEventListener('scroll', onMove, true);
      window.removeEventListener('resize', onMove);
    };
  });
</script>

<div bind:this={wrapperEl} style="position: relative; width: {width};">
  <button
    type="button"
    bind:this={triggerEl}
    class="combo-trigger"
    onclick={onTriggerClick}
    {disabled}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-label={ariaLabel}
    style="display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; height: 50px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-field); color: {selected ? 'var(--color-text)' : 'var(--color-neutral-500)'}; font-family: inherit; font-size: 15px; cursor: {disabled ? 'not-allowed' : 'pointer'}; opacity: {disabled ? 0.6 : 1};"
  >
    <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; text-align: left;">
      {selected ? selected.name : placeholder}
    </span>
    <i
      class="ph ph-caret-down combo-caret"
      style="font-size: 16px; flex-shrink: 0; transform: rotate({open ? 180 : 0}deg);"
    ></i>
  </button>

  {#if open}
    <div
      bind:this={listEl}
      use:portal
      role="listbox"
      style="position: fixed; {popStyle} overflow-y: auto; z-index: 90; background: var(--color-modal); border: 1px solid var(--color-neutral-800); border-radius: 10px; box-shadow: 0 18px 44px rgba(0, 0, 0, 0.5);"
    >
      <input
        bind:this={searchEl}
        type="search"
        bind:value={search}
        onkeydown={onSearchKeydown}
        placeholder={searchPlaceholder}
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        style="position: sticky; top: 0; z-index: 1; width: 100%; height: 46px; padding: 0 14px; border: none; outline: none; background: var(--color-modal); color: var(--color-text); font-family: inherit; font-size: 15px;"
      />

      {#if rows.length === 0}
        <p style="margin: 0; padding: 14px; font-size: 13px; color: var(--color-neutral-500);">{emptyText}</p>
      {:else}
        {#each rows as r, i (r.id ?? '__clear__')}
          <button
            type="button"
            role="option"
            aria-selected={r.id === value}
            data-idx={i}
            onmouseenter={() => (highlight = i)}
            onclick={() => pick(r.id)}
            style="display: flex; align-items: center; justify-content: space-between; gap: 10px; width: 100%; min-height: 48px; padding: 0 14px; border: none; text-align: left; background: {r.id === value
              ? 'var(--color-accent-panel)'
              : i === highlight
                ? 'color-mix(in srgb, var(--color-accent-panel) 50%, transparent)'
                : 'transparent'}; color: {r.id === value ? 'var(--color-accent-400)' : 'var(--color-text)'}; font-family: inherit; font-size: 14.5px; cursor: pointer;"
          >
            <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
              {r.name}
              {#if r.hint}
                <span style="margin-left: 6px; font-size: 12.5px; color: var(--color-neutral-500);">{r.hint}</span>
              {/if}
            </span>
            {#if r.id === value}
              <i class="ph-bold ph-check" style="font-size: 15px; flex-shrink: 0;"></i>
            {/if}
          </button>
        {/each}
        {#if hiddenCount > 0 && moreText}
          <p style="margin: 0; padding: 10px 14px; font-size: 12.5px; color: var(--color-neutral-500);">{moreText(hiddenCount)}</p>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  /* Mesmo padrão de hover dos outros controles do catálogo (rule 7: só
     :hover/animação/pseudo-elemento entram aqui). */
  .combo-trigger:hover:not(:disabled) {
    border-color: var(--color-accent-700);
  }

  .combo-caret {
    transition: transform 160ms ease;
  }
</style>
