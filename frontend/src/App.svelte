<script lang="ts">
  import Sidebar from './lib/components/Sidebar.svelte';
  import HomeView from './lib/components/HomeView.svelte';
  import CatalogView from './lib/components/CatalogView.svelte';
  import ManufacturersView from './lib/components/ManufacturersView.svelte';
  import ColorSearchView from './lib/components/ColorSearchView.svelte';
  import CompareView from './lib/components/CompareView.svelte';
  import EquivalentRecipeView from './lib/components/EquivalentRecipeView.svelte';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix';

  let currentView: View = $state('home');
  let sidebarCollapsed = $state(false);
  let recipeSourcePaintId: number | null = $state(null);

  function handleNavigate(view: View, paintId?: number) {
    currentView = view;
    if (view === 'mix' && paintId) {
      recipeSourcePaintId = paintId;
    }
  }
</script>

<div class="app-layout">
  <Sidebar {currentView} collapsed={sidebarCollapsed} onNavigate={handleNavigate} onToggle={() => sidebarCollapsed = !sidebarCollapsed} />

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
    {:else if currentView === 'mix'}
      <EquivalentRecipeView initialSourcePaintId={recipeSourcePaintId} />
    {/if}
  </main>
</div>
