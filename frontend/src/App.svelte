<script lang="ts">
  import { onMount } from 'svelte';
  import TopBar from './lib/components/TopBar.svelte';
  import CommandPalette from './lib/components/CommandPalette.svelte';
  import HomeView from './lib/components/HomeView.svelte';
  import CatalogView from './lib/components/CatalogView.svelte';
  import ManufacturersView from './lib/components/ManufacturersView.svelte';
  import ColorSearchView from './lib/components/ColorSearchView.svelte';
  import CompareView from './lib/components/CompareView.svelte';
  import EquivalentRecipeView from './lib/components/EquivalentRecipeView.svelte';
  import ColorWheelView from './lib/components/ColorWheelView.svelte';
  import MyStockView from './lib/components/MyStockView.svelte';
  import ToastRegion from './lib/components/ToastRegion.svelte';
  import GuideDialog from './lib/components/GuideDialog.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  let currentView: View = $state('home');
  let recipeSourcePaintId: number | null = $state(null);
  let guideOpen = $state(false);
  let paletteOpen = $state(false);

  function handleNavigate(view: View, paintId?: number) {
    currentView = view;
    if (view === 'mix' && paintId) {
      recipeSourcePaintId = paintId;
    }
  }

  // ⌘K / Ctrl+K abre a busca global de qualquer tela.
  function onKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      paletteOpen = !paletteOpen;
    }
  }

  onMount(() => {
    // Guia abre sozinho só na primeira execução; depois fica no "?" da barra.
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
    onHelp={() => (guideOpen = true)}
  />

  <main class="app-main">
    {#if currentView === 'home'}
      <HomeView onNavigate={handleNavigate} />
    {:else if currentView === 'catalog'}
      <CatalogView onNavigate={handleNavigate} />
    {:else if currentView === 'manufacturers'}
      <ManufacturersView />
    {:else if currentView === 'color-search'}
      <ColorSearchView />
    {:else if currentView === 'compare'}
      <CompareView />
    {:else if currentView === 'wheel'}
      <ColorWheelView />
    {:else if currentView === 'stock'}
      <MyStockView />
    {:else if currentView === 'mix'}
      <EquivalentRecipeView initialSourcePaintId={recipeSourcePaintId} />
    {/if}
  </main>
</div>

<CommandPalette open={paletteOpen} onClose={() => (paletteOpen = false)} onNavigate={handleNavigate} />
<GuideDialog bind:open={guideOpen} onClose={closeGuide} />
<ToastRegion />
