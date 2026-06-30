<script lang="ts">
  import { onMount } from 'svelte';
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

<div class="flex h-full">
  <Sidebar {currentView} {sidebarCollapsed} onNavigate={handleNavigate} onToggle={() => sidebarCollapsed = !sidebarCollapsed} />

  <main class="flex-1 overflow-y-auto" class:ml-0={sidebarCollapsed}>
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
