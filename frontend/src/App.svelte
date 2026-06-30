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

<div class="noise-overlay flex h-full">
  <!-- Ambient glows -->
  <div class="ambient-glow" style="top: -200px; left: -200px; background: radial-gradient(circle, var(--color-amber-glow), transparent);"></div>
  <div class="ambient-glow" style="bottom: -300px; right: -200px; background: radial-gradient(circle, #6c5ce7, transparent); opacity: 0.02;"></div>

  <Sidebar {currentView} collapsed={sidebarCollapsed} onNavigate={handleNavigate} onToggle={() => sidebarCollapsed = !sidebarCollapsed} />

  <main class="flex-1 overflow-y-auto relative z-10">
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
