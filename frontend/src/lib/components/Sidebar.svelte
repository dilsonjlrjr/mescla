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
    { id: 'compare', label: 'Comparar', icon: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4' },
    { id: 'mix', label: 'Mistura', icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z' },
  ];
</script>

<aside
  class="h-full flex flex-col transition-all duration-300 ease-out relative {collapsed ? 'w-[68px]' : 'w-[220px]'}"
  style="background: linear-gradient(180deg, var(--color-obsidian-900), var(--color-obsidian-950)); border-right: 1px solid var(--color-glass-border);"
>
  <!-- Decorative gradient line at top -->
  <div class="absolute top-0 left-0 right-0 h-[1px]" style="background: linear-gradient(90deg, transparent, var(--color-amber-glow), transparent); opacity: 0.4;"></div>

  <!-- Logo -->
  <div class="flex items-center gap-3 px-4 h-[60px] border-b border-white/[0.04]">
    <div class="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 relative"
      style="background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm)); box-shadow: 0 4px 12px rgba(212,160,83,0.3);">
      <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
      </svg>
    </div>
    {#if !collapsed}
      <div class="animate-artisan-fade">
        <span class="font-[family-name:var(--font-display)] font-bold text-[15px] text-white tracking-tight">Paint Match</span>
        <span class="block text-[9px] font-semibold tracking-[0.2em] uppercase" style="color: var(--color-amber-glow);">AI</span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav class="flex-1 py-4 px-2.5 space-y-1">
    {#each navItems as item, i}
      {@const isActive = currentView === item.id}
      <button
        onclick={() => onNavigate(item.id)}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-[13px] font-medium transition-all duration-200 group relative
          {isActive
            ? 'text-white'
            : 'text-[var(--color-obsidian-400)] hover:text-[var(--color-obsidian-200)] hover:bg-white/[0.03]'}"
        style="animation-delay: {i * 40}ms;"
      >
        {#if isActive}
          <div class="absolute inset-0 rounded-xl animate-artisan-scale"
            style="background: linear-gradient(135deg, rgba(212,160,83,0.12), rgba(212,160,83,0.03)); border: 1px solid rgba(212,160,83,0.15); box-shadow: 0 2px 12px rgba(212,160,83,0.08);"></div>
        {/if}
        <svg class="w-[18px] h-[18px] flex-shrink-0 relative z-10 transition-colors duration-200 {isActive ? 'text-[var(--color-amber-glow)]' : ''}" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
        </svg>
        {#if !collapsed}
          <span class="relative z-10 animate-artisan-fade">{item.label}</span>
        {/if}
        {#if isActive && !collapsed}
          <div class="ml-auto w-1.5 h-1.5 rounded-full bg-[var(--color-amber-glow)] relative z-10" style="box-shadow: 0 0 8px rgba(212,160,83,0.5);"></div>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Version badge -->
  {#if !collapsed}
    <div class="px-4 pb-3">
      <div class="text-[10px] text-[var(--color-obsidian-600)] font-mono">v1.0.0</div>
    </div>
  {/if}

  <!-- Toggle -->
  <div class="p-2.5 border-t border-white/[0.04]">
    <button onclick={onToggle}
      class="w-full flex items-center justify-center py-2 rounded-xl text-[var(--color-obsidian-500)] hover:text-[var(--color-obsidian-300)] hover:bg-white/[0.03] transition-all duration-200">
      <svg class="w-4 h-4 transition-transform duration-300 {collapsed ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
      </svg>
    </button>
  </div>
</aside>
