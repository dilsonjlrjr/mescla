<script lang="ts">
  import { onMount } from 'svelte';

  type View = 'home' | 'catalog' | 'color-search' | 'compare' | 'mix';

  interface Props {
    onNavigate: (view: View) => void;
  }

  let { onNavigate }: Props = $props();

  interface Stats {
    manufacturers: number;
    productLines: number;
    paints: number;
    equivalences: number;
    recipes: number;
  }

  let stats: Stats | null = $state(null);
  let loading = $state(true);

  onMount(async () => {
    try {
      const { PaintService } = await import('../../../bindings/paint-match-ai');
      stats = await PaintService.GetStats();
    } catch (e) {
      console.error('Erro carregando stats:', e);
      stats = { manufacturers: 11, productLines: 32, paints: 45, equivalences: 0, recipes: 0 };
    } finally {
      loading = false;
    }
  });

  const quickActions: { view: View; title: string; desc: string; icon: string; gradient: string }[] = [
    {
      view: 'catalog',
      title: 'Catálogo',
      desc: 'Explore todas as tintas cadastradas',
      icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
      gradient: 'from-blue-500/20 to-cyan-500/20',
    },
    {
      view: 'color-search',
      title: 'Buscar por Cor',
      desc: 'Encontre tintas similares a uma cor',
      icon: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
      gradient: 'from-purple-500/20 to-pink-500/20',
    },
    {
      view: 'compare',
      title: 'Comparar',
      desc: 'Compare tintas lado a lado',
      icon: 'M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3',
      gradient: 'from-amber-500/20 to-orange-500/20',
    },
    {
      view: 'mix',
      title: 'Mistura',
      desc: 'Sugestão de receitas de mistura',
      icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z',
      gradient: 'from-emerald-500/20 to-teal-500/20',
    },
  ];
</script>

<div class="min-h-full p-8">
  <!-- Hero -->
  <div class="mb-10 animate-fadeIn">
    <div class="flex items-center gap-4 mb-2">
      <div class="w-12 h-12 rounded-2xl flex items-center justify-center"
        style="background: linear-gradient(135deg, var(--color-accent-500), var(--color-accent-600)); box-shadow: 0 8px 32px rgba(232,168,76,0.3);">
        <svg class="w-7 h-7 text-white" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
        </svg>
      </div>
      <div>
        <h1 class="text-3xl font-bold text-white tracking-tight">Paint Match AI</h1>
        <p class="text-sm" style="color: var(--color-surface-400);">Base de conhecimento para pintores de miniaturas</p>
      </div>
    </div>
  </div>

  <!-- Stats -->
  <div class="grid grid-cols-5 gap-4 mb-10">
    {#if loading}
      {#each Array(5) as _}
        <div class="glass rounded-xl p-4 animate-pulse">
          <div class="h-8 w-16 bg-white/5 rounded mb-2"></div>
          <div class="h-3 w-20 bg-white/5 rounded"></div>
        </div>
      {/each}
    {:else if stats}
      {#each [
        { value: stats.manufacturers, label: 'Fabricantes', color: 'var(--color-accent-400)' },
        { value: stats.productLines, label: 'Linhas', color: '#60a5fa' },
        { value: stats.paints, label: 'Tintas', color: '#a78bfa' },
        { value: stats.equivalences, label: 'Equivalências', color: '#34d399' },
        { value: stats.recipes, label: 'Receitas', color: '#f472b6' },
      ] as stat, i}
        <div class="glass rounded-xl p-4 animate-fadeIn group hover:border-white/10 transition-colors" style="animation-delay: {i * 80}ms;">
          <div class="text-2xl font-bold mb-1" style="color: {stat.color};">{stat.value}</div>
          <div class="text-xs font-medium uppercase tracking-wider" style="color: var(--color-surface-500);">{stat.label}</div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Quick Actions -->
  <div class="mb-8">
    <h2 class="text-sm font-semibold uppercase tracking-wider mb-4" style="color: var(--color-surface-500);">Ações Rápidas</h2>
    <div class="grid grid-cols-2 gap-4">
      {#each quickActions as action, i}
        <button
          onclick={() => onNavigate(action.view)}
          class="glass glass-hover rounded-xl p-5 text-left transition-all duration-200 group animate-fadeIn"
          style="animation-delay: {200 + i * 80}ms;"
        >
          <div class="flex items-start gap-4">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0 bg-gradient-to-br {action.gradient}">
              <svg class="w-5 h-5 text-white/80" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d={action.icon} />
              </svg>
            </div>
            <div>
              <div class="font-semibold text-white text-sm mb-1 group-hover:text-accent-300 transition-colors">{action.title}</div>
              <div class="text-xs" style="color: var(--color-surface-500);">{action.desc}</div>
            </div>
          </div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Info -->
  <div class="glass rounded-xl p-5 animate-fadeIn" style="animation-delay: 600ms;">
    <div class="flex items-start gap-3">
      <svg class="w-5 h-5 flex-shrink-0 mt-0.5" style="color: var(--color-accent-400);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <div>
        <div class="text-sm font-medium text-white mb-1">Sistema Offline</div>
        <div class="text-xs leading-relaxed" style="color: var(--color-surface-400);">
          Paint Match AI funciona completamente offline. Todos os dados estão armazenados localmente em SQLite.
          Use o comando <code class="font-mono text-xs px-1.5 py-0.5 rounded" style="background: var(--color-surface-800); color: var(--color-accent-400);">go run ./cmd/seed</code> para popular o banco com dados de fabricantes e tintas.
        </div>
      </div>
    </div>
  </div>
</div>
