<script lang="ts">
  import Sidebar from './lib/components/Sidebar.svelte';
  import HomeView from './lib/components/HomeView.svelte';
  import CatalogView from './lib/components/CatalogView.svelte';
  import ColorSearchView from './lib/components/ColorSearchView.svelte';
  import CompareView from './lib/components/CompareView.svelte';
  import MixView from './lib/components/MixView.svelte';

  type View = 'home' | 'catalog' | 'color-search' | 'compare' | 'mix';

  let currentView: View = $state('home');
  let sidebarCollapsed = $state(false);

  function handleNavigate(view: View) {
    currentView = view;
  }
</script>

<div class="app-layout">
  <Sidebar {currentView} collapsed={sidebarCollapsed} onNavigate={handleNavigate} onToggle={() => sidebarCollapsed = !sidebarCollapsed} />

  <main class="app-main">
    {#if currentView === 'home'}
      <HomeView onNavigate={handleNavigate} />
    {:else if currentView === 'catalog'}
      <CatalogView />
    {:else if currentView === 'color-search'}
      <ColorSearchView />
    {:else if currentView === 'compare'}
      <CompareView />
    {:else if currentView === 'mix'}
      <MixView />
    {/if}
  </main>
</div>
