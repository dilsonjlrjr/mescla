<script lang="ts">
  type View = 'home' | 'catalog' | 'color-search' | 'compare' | 'mix';

  interface Props {
    currentView: View;
    collapsed: boolean;
    onNavigate: (view: View) => void;
    onToggle: () => void;
  }

  let { currentView, collapsed, onNavigate, onToggle }: Props = $props();

  const navItems: { id: View; label: string; icon: string }[] = [
    { id: 'home', label: 'Início', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
    { id: 'catalog', label: 'Catálogo', icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10' },
    { id: 'color-search', label: 'Buscar Cor', icon: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z' },
    { id: 'compare', label: 'Comparar', icon: 'M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3' },
    { id: 'mix', label: 'Mistura', icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z' },
  ];
</script>

<aside class="h-full flex flex-col transition-all duration-300 ease-in-out {collapsed ? 'w-16' : 'w-56'}"
  style="background: var(--color-surface-900); border-right: 1px solid var(--color-glass-border);">

  <!-- Logo -->
  <div class="flex items-center gap-3 px-4 h-14 border-b border-white/5">
    <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
      style="background: linear-gradient(135deg, var(--color-accent-500), var(--color-accent-600));">
      <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
      </svg>
    </div>
    {#if !collapsed}
      <div class="animate-fadeIn">
        <span class="font-bold text-sm text-white tracking-tight">Paint Match</span>
        <span class="block text-[10px] font-medium tracking-widest uppercase" style="color: var(--color-accent-400);">AI</span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav class="flex-1 py-3 px-2 space-y-1">
    {#each navItems as item, i}
      <button
        onclick={() => onNavigate(item.id)}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 group relative
          {currentView === item.id
            ? 'text-white'
            : 'text-surface-400 hover:text-surface-200 hover:bg-white/[0.03]'}"
        style="animation-delay: {i * 50}ms;"
      >
        {#if currentView === item.id}
          <div class="absolute inset-0 rounded-lg animate-scaleIn"
            style="background: linear-gradient(135deg, rgba(232,168,76,0.15), rgba(232,168,76,0.05)); border: 1px solid rgba(232,168,76,0.2);"></div>
        {/if}
        <svg class="w-5 h-5 flex-shrink-0 relative z-10 {currentView === item.id ? 'text-accent-400' : ''}" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
        </svg>
        {#if !collapsed}
          <span class="relative z-10 animate-fadeIn">{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Toggle -->
  <div class="p-2 border-t border-white/5">
    <button onclick={onToggle}
      class="w-full flex items-center justify-center py-2 rounded-lg text-surface-500 hover:text-surface-300 hover:bg-white/[0.03] transition-colors">
      <svg class="w-5 h-5 transition-transform duration-300 {collapsed ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
      </svg>
    </button>
  </div>
</aside>
