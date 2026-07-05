<script lang="ts">
  // Mais — Comparar (lista-âncora), Marcas, Guia ΔE, Instalar app, Sobre.
  // Comparar e Marcas abrem como sub-telas dentro da aba (camadas de
  // histórico: o back volta pro menu).
  import Icon from '../components/Icon.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import DeltaBadge from '../components/DeltaBadge.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import FullScreenSearch from '../components/FullScreenSearch.svelte';
  import { pushLayer, switchTab } from '../nav.svelte';
  import { allManufacturers, paintById, type Paint } from '../services/catalog';
  import { compareToAnchor, type SearchResult } from '../services/engine';
  import { appState, addToCompare, removeFromCompare } from '../appState.svelte';
  import { recents } from '../recents.svelte';
  import { pwa, promptInstall } from '../pwa.svelte';
  import { toast } from '../toast.svelte';

  let subView: 'menu' | 'comparar' | 'marcas' = $state('menu');
  let deltaSheetOpen = $state(false);
  let searchOpen = $state(false);
  let anchorId: number | null = $state(null);
  let compared: SearchResult[] = $state([]);

  let closeSubLayer: (() => void) | null = null;

  function openSub(view: 'comparar' | 'marcas') {
    subView = view;
    closeSubLayer = pushLayer(() => {
      closeSubLayer = null;
      subView = 'menu';
    });
  }

  function backToMenu() {
    closeSubLayer?.();
  }

  // ── Comparar: lista-âncora ──
  let anchor = $derived(anchorId ? paintById(anchorId) : null);

  $effect(() => {
    // âncora padrão: primeira tinta adicionada
    if (appState.compareIds.length > 0 && (anchorId === null || !appState.compareIds.includes(anchorId))) {
      anchorId = appState.compareIds[0];
    }
    if (appState.compareIds.length === 0) {
      anchorId = null;
      compared = [];
    }
  });

  $effect(() => {
    const a = anchorId;
    const ids = [...appState.compareIds];
    if (a === null || ids.length < 2) {
      compared = [];
      return;
    }
    compareToAnchor(a, ids)
      .then(res => (compared = res))
      .catch(() => (compared = []));
  });

  function addPaint(p: Paint) {
    if (!addToCompare(p.id)) {
      toast(appState.compareIds.includes(p.id) ? 'Já está na comparação' : 'Máximo de 6 tintas', 'error');
    }
  }

  function promote(id: number) {
    if (navigator.vibrate) navigator.vibrate(10);
    anchorId = id;
  }

  function remove(id: number) {
    removeFromCompare(id);
  }

  function openCatalogFiltered(mfrId: number) {
    appState.pendingCatalogMfrId = mfrId;
    switchTab('catalogo');
  }
</script>

<div class="mais">
  {#if subView === 'menu'}
    <header style="padding: 8px 0 18px;">
      <h1 class="screen-title">Mais</h1>
    </header>

    <button class="menu-row pressable" onclick={() => openSub('comparar')}>
      <span class="menu-icon"><Icon name="swap" size={20} /></span>
      <span class="menu-text">Comparar tintas</span>
      {#if appState.compareIds.length > 0}
        <span class="menu-badge font-mono">{appState.compareIds.length}</span>
      {/if}
    </button>

    <button class="menu-row pressable" onclick={() => openSub('marcas')}>
      <span class="menu-icon"><Icon name="building" size={20} /></span>
      <span class="menu-text">Marcas</span>
      <span class="menu-badge font-mono muted">{allManufacturers().length}</span>
    </button>

    <button class="menu-row pressable" onclick={() => (deltaSheetOpen = true)}>
      <span class="menu-icon"><Icon name="info" size={20} /></span>
      <span class="menu-text">O que é o ΔE?</span>
    </button>

    {#if pwa.canInstall}
      <button class="menu-row pressable" onclick={promptInstall}>
        <span class="menu-icon"><Icon name="plus" size={20} /></span>
        <span class="menu-text">Instalar o app</span>
        <span class="menu-hint">funciona offline na loja</span>
      </button>
    {/if}

    <div class="about">
      <p class="about-name font-display">Mescla</p>
      <p class="about-tag font-mono">cor certa, qualquer marca · v1.0.0</p>
      <p class="about-note">Cores de tela são aproximadas — confie no ΔE.</p>
    </div>
  {:else if subView === 'comparar'}
    <header class="sub-head">
      <button class="sub-back pressable" onclick={backToMenu} aria-label="Voltar">
        <Icon name="chevron-left" size={20} />
      </button>
      <h1 class="screen-title">Comparar</h1>
    </header>

    {#if appState.compareIds.length === 0}
      <div class="empty-state">
        <span class="empty-icon"><Icon name="swap" size={36} /></span>
        <p class="empty-title">Nenhuma tinta na comparação</p>
        <p class="empty-hint">Adicione pelo Catálogo (tinta → "Adicionar à comparação") ou pelo botão abaixo.</p>
      </div>
    {:else if anchor}
      <!-- Âncora: a pergunta é "qual destas é mais parecida com ELA?" -->
      <div class="anchor-card panel">
        <PaintBottle r={anchor.r} g={anchor.g} b={anchor.b} size={52} label={anchor.code} />
        <div class="anchor-text">
          <span class="anchor-eyebrow">comparando com</span>
          <span class="anchor-name">{anchor.name}</span>
          <span class="anchor-meta"><span class="font-mono">{anchor.code}</span> · {anchor.manufacturer}</span>
        </div>
        <button class="row-trash pressable" onclick={() => remove(anchor!.id)} aria-label="Remover âncora">
          <Icon name="trash" size={17} />
        </button>
      </div>

      {#if compared.length > 0}
        <p class="compare-caption">da mais próxima à mais distante — toque pra virar âncora</p>
      {/if}
      {#each compared as res (res.paintId)}
        <div class="compare-row">
          <button class="compare-main pressable" onclick={() => promote(res.paintId)}>
            <span class="color-pair" style="width: 34px; height: 22px;">
              <span style="background: rgb({anchor.r}, {anchor.g}, {anchor.b});"></span>
              <span style="background: rgb({res.r}, {res.g}, {res.b});"></span>
            </span>
            <span class="compare-text">
              <span class="compare-name">{res.name}</span>
              <span class="compare-meta"><span class="font-mono">{res.code}</span> · {res.manufacturer}</span>
            </span>
            <DeltaBadge deltaE={res.deltaE} size="sm" />
          </button>
          <button class="row-trash pressable" onclick={() => remove(res.paintId)} aria-label="Remover {res.name}">
            <Icon name="trash" size={17} />
          </button>
        </div>
      {/each}
    {/if}

    {#if appState.compareIds.length < 6}
      <button class="btn-ghost" style="width: 100%; margin-top: 14px;" onclick={() => (searchOpen = true)}>
        <Icon name="plus" size={17} />
        Adicionar tinta ({appState.compareIds.length}/6)
      </button>
    {/if}
  {:else if subView === 'marcas'}
    <header class="sub-head">
      <button class="sub-back pressable" onclick={backToMenu} aria-label="Voltar">
        <Icon name="chevron-left" size={20} />
      </button>
      <h1 class="screen-title">Marcas</h1>
    </header>

    {#each allManufacturers() as m (m.id)}
      <button class="menu-row pressable" onclick={() => openCatalogFiltered(m.id)}>
        <span class="menu-text">{m.name}</span>
        <span class="menu-badge font-mono muted">{m.paintCount} tintas</span>
      </button>
    {/each}
  {/if}
</div>

<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} />
<FullScreenSearch
  open={searchOpen}
  onClose={() => (searchOpen = false)}
  onSelect={addPaint}
  recentIds={recents.paintIds}
  placeholder="Tinta pra comparar…"
/>

<style>
  .mais {
    padding: 16px;
  }

  .menu-row {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    min-height: 60px;
    padding: 8px 14px;
    border: 1px solid var(--ink-700);
    border-radius: 12px;
    background: var(--ink-900);
    margin-bottom: 10px;
    text-align: left;
  }

  .menu-icon {
    display: flex;
    color: var(--lacquer-deep);
    flex-shrink: 0;
  }

  .menu-text {
    flex: 1;
    min-width: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .menu-badge {
    flex-shrink: 0;
    font-size: 12px;
    font-weight: 600;
    padding: 4px 10px;
    border-radius: 999px;
    background: var(--lacquer);
    color: white;
  }

  .menu-badge.muted {
    background: var(--ink-800);
    color: var(--ink-500);
    font-weight: 500;
  }

  .menu-hint {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--ink-500);
  }

  .about {
    text-align: center;
    padding: 32px 0 8px;
  }

  .about-name {
    font-size: 20px;
    font-weight: 600;
    color: var(--ink-300);
  }

  .about-tag {
    font-size: 11px;
    letter-spacing: 0.1em;
    color: var(--ink-500);
    margin-top: 2px;
  }

  .about-note {
    font-size: 12px;
    color: var(--ink-500);
    margin-top: 14px;
  }

  .sub-head {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0 16px;
  }

  .sub-back {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: 12px;
    border: 1px solid var(--ink-700);
    background: var(--ink-900);
    color: var(--ink-300);
    flex-shrink: 0;
  }

  .anchor-card {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px;
    margin-bottom: 14px;
    border-left: 3px solid var(--lacquer);
  }

  .anchor-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .anchor-eyebrow {
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--lacquer-deep);
  }

  .anchor-name {
    font-size: 17px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .anchor-meta {
    font-size: 13px;
    color: var(--ink-500);
  }

  .compare-caption {
    font-size: 12px;
    color: var(--ink-500);
    padding: 0 4px 8px;
  }

  .compare-row {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  }

  .compare-main {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;
    min-height: 60px;
    padding: 8px 12px;
    border: 1px solid var(--ink-700);
    border-radius: 12px;
    background: var(--ink-900);
    text-align: left;
  }

  .compare-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .compare-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .compare-meta {
    font-size: 12.5px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .row-trash {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: 10px;
    color: var(--ink-500);
    flex-shrink: 0;
  }

  .row-trash:active {
    color: var(--delta-poor);
  }
</style>
