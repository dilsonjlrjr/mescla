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

  const quickActions: { view: View; title: string; desc: string; icon: string; gradient: string; glow: string }[] = [
    {
      view: 'catalog',
      title: 'Catálogo Completo',
      desc: 'Explore todas as tintas cadastradas por fabricante',
      icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10',
      gradient: 'linear-gradient(135deg, #2e86de22, #00b89422)',
      glow: '#2e86de',
    },
    {
      view: 'color-search',
      title: 'Buscar por Cor',
      desc: 'Encontre tintas similares usando Delta E 2000',
      icon: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
      gradient: 'linear-gradient(135deg, #8e44ad22, #e1705522)',
      glow: '#8e44ad',
    },
    {
      view: 'compare',
      title: 'Comparar Tintas',
      desc: 'Compare cores lado a lado visualmente',
      icon: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4',
      gradient: 'linear-gradient(135deg, #d4a05322, #c0392b22)',
      glow: '#d4a053',
    },
    {
      view: 'mix',
      title: 'Receita de Mistura',
      desc: 'Descubra a fórmula perfeita para sua cor',
      icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z',
      gradient: 'linear-gradient(135deg, #27ae6022, #2e86de22)',
      glow: '#27ae60',
    },
  ];
</script>

<div class="min-h-full p-8 relative z-10">
  <!-- Hero Header -->
  <div class="mb-10 animate-artisan-fade">
    <div class="flex items-end gap-5 mb-3">
      <div class="w-14 h-14 rounded-2xl flex items-center justify-center relative"
        style="background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm)); box-shadow: 0 8px 32px rgba(212,160,83,0.3), 0 0 0 1px rgba(255,255,255,0.1) inset;">
        <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
        </svg>
      </div>
      <div>
        <h1 class="font-[family-name:var(--font-display)] text-4xl font-bold text-white tracking-tight leading-none">
          Paint Match <span style="color: var(--color-amber-glow);">AI</span>
        </h1>
        <p class="text-sm mt-1.5" style="color: var(--color-obsidian-400);">Base de conhecimento profissional para pintores de miniaturas</p>
      </div>
    </div>
    <!-- Decorative line -->
    <div class="mt-5 h-[1px]" style="background: linear-gradient(90deg, var(--color-amber-glow), transparent 60%); opacity: 0.2;"></div>
  </div>

  <!-- Stats Grid -->
  <div class="grid grid-cols-5 gap-4 mb-10">
    {#if loading}
      {#each Array(5) as _}
        <div class="artisan-card p-5">
          <div class="skeleton h-9 w-16 mb-2.5"></div>
          <div class="skeleton h-3 w-20"></div>
        </div>
      {/each}
    {:else if stats}
      {#each [
        { value: stats.manufacturers, label: 'Fabricantes', color: 'var(--color-amber-glow)', icon: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4' },
        { value: stats.productLines, label: 'Linhas', color: '#2e86de', icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10' },
        { value: stats.paints, label: 'Tintas', color: '#8e44ad', icon: 'M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01' },
        { value: stats.equivalences, label: 'Equivalências', color: '#27ae60', icon: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4' },
        { value: stats.recipes, label: 'Receitas', color: '#e17055', icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z' },
      ] as stat, i}
        <div class="artisan-card stat-card p-5 animate-artisan-fade" style="animation-delay: {i * 60}ms;">
          <div class="absolute -top-8 -right-8 w-24 h-24 rounded-full" style="background: {stat.color}; filter: blur(50px);"></div>
          <div class="flex items-center gap-3 mb-3">
            <div class="w-8 h-8 rounded-lg flex items-center justify-center" style="background: {stat.color}15;">
              <svg class="w-4 h-4" style="color: {stat.color};" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d={stat.icon} />
              </svg>
            </div>
          </div>
          <div class="font-[family-name:var(--font-display)] text-3xl font-bold mb-0.5" style="color: {stat.color};">{stat.value}</div>
          <div class="text-[11px] font-semibold uppercase tracking-[0.15em]" style="color: var(--color-obsidian-500);">{stat.label}</div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Quick Actions -->
  <div class="mb-8">
    <h2 class="font-[family-name:var(--font-display)] text-lg font-semibold text-white mb-5 flex items-center gap-3">
      <span>Ações Rápidas</span>
      <div class="flex-1 h-[1px]" style="background: linear-gradient(90deg, var(--color-glass-border), transparent);"></div>
    </h2>
    <div class="grid grid-cols-2 gap-4">
      {#each quickActions as action, i}
        <button
          onclick={() => onNavigate(action.view)}
          class="artisan-card p-6 text-left transition-all duration-300 group animate-artisan-fade relative overflow-hidden"
          style="animation-delay: {200 + i * 60}ms;"
        >
          <!-- Glow on hover -->
          <div class="absolute -top-12 -right-12 w-32 h-32 rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-500" style="background: {action.glow}; filter: blur(50px);"></div>

          <div class="flex items-start gap-4 relative z-10">
            <div class="w-12 h-12 rounded-xl flex items-center justify-center flex-shrink-0" style="background: {action.gradient}; border: 1px solid rgba(255,255,255,0.06);">
              <svg class="w-6 h-6 text-white/80" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d={action.icon} />
              </svg>
            </div>
            <div>
              <div class="font-semibold text-white text-[15px] mb-1 group-hover:text-[var(--color-amber-hot)] transition-colors duration-200">{action.title}</div>
              <div class="text-[13px] leading-relaxed" style="color: var(--color-obsidian-400);">{action.desc}</div>
            </div>
          </div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Info Footer -->
  <div class="artisan-card p-5 animate-artisan-fade relative overflow-hidden" style="animation-delay: 500ms;">
    <div class="absolute top-0 left-0 right-0 h-[1px]" style="background: linear-gradient(90deg, transparent, var(--color-amber-glow), transparent); opacity: 0.15;"></div>
    <div class="flex items-start gap-3">
      <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: rgba(212,160,83,0.1);">
        <svg class="w-4 h-4" style="color: var(--color-amber-glow);" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <div>
        <div class="text-sm font-semibold text-white mb-1">Sistema 100% Offline</div>
        <div class="text-[13px] leading-relaxed" style="color: var(--color-obsidian-400);">
          Todos os dados estão armazenados localmente em SQLite. Para popular o banco, execute:
          <code class="font-mono text-xs px-1.5 py-0.5 rounded-lg ml-1" style="background: var(--color-obsidian-800); color: var(--color-amber-glow);">rtk go run ./cmd/seed</code>
        </div>
      </div>
    </div>
  </div>
</div>
