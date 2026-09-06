<script lang="ts">
  import { onMount } from 'svelte';
  import TopBar from './lib/components/TopBar.svelte';
  import CommandPalette from './lib/components/CommandPalette.svelte';
  import CatalogView from './lib/components/CatalogView.svelte';
  import ColorSearchView from './lib/components/ColorSearchView.svelte';
  import CompareView from './lib/components/CompareView.svelte';
  import EquivalentRecipeView from './lib/components/EquivalentRecipeView.svelte';
  import ColorWheelView from './lib/components/ColorWheelView.svelte';
  import MyStockView from './lib/components/MyStockView.svelte';
  import RecipesView from './lib/components/RecipesView.svelte';
  import PlanejadorPintura from './lib/components/PlanejadorPintura.svelte';
  import ToastRegion from './lib/components/ToastRegion.svelte';
  import GuideDialog from './lib/components/GuideDialog.svelte';

  // rf-04: T1 = 'home' (ColorSearchView reorganizada), T2 = 'planner', T3 =
  // 'receitas' (nova), T4 = 'stock' (MyStockView com abas Tintas/Fabricantes).
  // 'manufacturers' e as demais telas legadas (mix/compare/wheel/color-search
  // como ferramenta avulsa) continuam navegáveis — fora do nav principal de
  // 4 itens, mas sem quebrar quem ainda aponta pra elas (CatalogView, palette).
  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock' | 'planner' | 'receitas';

  interface NavOpts {
    paintId?: number;
    targetManufacturerId?: number; // mix: re-executar receita salva
    manufacturer?: string;         // catalog: filtro de marca
    stockPrefillPaintId?: number;  // stock: abrir form pré-preenchido
  }

  let currentView: View = $state('home');
  let recipeSourcePaintId: number | null = $state(null);
  let recipeTargetMfrId: number | null = $state(null);
  let catalogManufacturer: string | null = $state(null);
  let catalogPaintId: number | null = $state(null);
  let compareAnchorId: number | null = $state(null);
  let stockPrefillPaintId: number | null = $state(null);
  let stockInitialTab: 'tintas' | 'fabricantes' = $state('tintas');
  let guideOpen = $state(false);
  let paletteOpen = $state(false);
  // força remontagem da view quando a mesma rota é reaberta com outro contexto
  let navSeq = $state(0);

  function handleNavigate(view: View, opts?: number | NavOpts) {
    const o: NavOpts = typeof opts === 'number' ? { paintId: opts } : (opts ?? {});
    recipeSourcePaintId = view === 'mix' ? (o.paintId ?? null) : null;
    recipeTargetMfrId = view === 'mix' ? (o.targetManufacturerId ?? null) : null;
    catalogManufacturer = view === 'catalog' ? (o.manufacturer ?? null) : null;
    catalogPaintId = view === 'catalog' ? (o.paintId ?? null) : null;
    compareAnchorId = view === 'compare' ? (o.paintId ?? null) : null;
    stockPrefillPaintId = (view === 'stock' || view === 'manufacturers') ? (o.stockPrefillPaintId ?? null) : null;
    navSeq++;
    // 'manufacturers' é um alias de navegação pra T4 já na aba Fabricantes —
    // CatalogView e a busca global ainda apontam pra esse destino (T2/T4 nota
    // de construção: sem sobreposição, sem duplicar a tela).
    currentView = view === 'manufacturers' ? 'stock' : view;
    stockInitialTab = view === 'manufacturers' ? 'fabricantes' : (view === 'stock' ? 'tintas' : stockInitialTab);
  }

  // ⌘K / Ctrl+K abre a busca global de qualquer tela.
  function onKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      paletteOpen = !paletteOpen;
    }
  }

  onMount(() => {
    // Guia abre sozinho só na primeira execução.
    if (!localStorage.getItem('mescla_guided')) {
      guideOpen = true;
    }
  });

  function closeGuide() {
    localStorage.setItem('mescla_guided', '1');
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="app-layout">
  <TopBar
    {currentView}
    onNavigate={handleNavigate}
    onSearch={() => (paletteOpen = true)}
  />

  <main class="app-main">
    {#key navSeq}
      {#if currentView === 'home' || currentView === 'color-search'}
        <ColorSearchView onNavigate={handleNavigate} />
      {:else if currentView === 'catalog'}
        <CatalogView onNavigate={handleNavigate} initialManufacturer={catalogManufacturer} initialPaintId={catalogPaintId} />
      {:else if currentView === 'compare'}
        <CompareView initialAnchorId={compareAnchorId} />
      {:else if currentView === 'wheel'}
        <ColorWheelView />
      {:else if currentView === 'stock'}
        <MyStockView prefillPaintId={stockPrefillPaintId} initialTab={stockInitialTab} />
      {:else if currentView === 'receitas'}
        <RecipesView />
      {:else if currentView === 'mix'}
        <EquivalentRecipeView initialSourcePaintId={recipeSourcePaintId} initialTargetManufacturerId={recipeTargetMfrId} />
      {:else if currentView === 'planner'}
        <PlanejadorPintura />
      {/if}
    {/key}
  </main>
</div>

<CommandPalette open={paletteOpen} onClose={() => (paletteOpen = false)} onNavigate={handleNavigate} />
<GuideDialog bind:open={guideOpen} onClose={closeGuide} />
<ToastRegion />
