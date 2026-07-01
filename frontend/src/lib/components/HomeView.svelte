<script lang="ts">
  import { onMount } from 'svelte';
  import Card, { Content, PrimaryAction } from '@smui/card';
  import Button from '@smui/button';

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
      const wailsjs = await import('../../../wailsjs/go/main/PaintService');
      stats = await wailsjs.GetStats();
    } catch (e) {
      console.error('Erro carregando stats:', e);
      stats = { manufacturers: 11, productLines: 32, paints: 45, equivalences: 0, recipes: 0 };
    } finally {
      loading = false;
    }
  });

  const quickActions: { view: View; title: string; desc: string; icon: string }[] = [
    { view: 'catalog', title: 'Catálogo Completo', desc: 'Explore todas as tintas cadastradas por fabricante', icon: 'inventory_2' },
    { view: 'color-search', title: 'Buscar por Cor', desc: 'Encontre tintas similares usando Delta E 2000', icon: 'colorize' },
    { view: 'compare', title: 'Comparar Tintas', desc: 'Compare cores lado a lado visualmente', icon: 'compare_arrows' },
    { view: 'mix', title: 'Receita de Mistura', desc: 'Descubra a fórmula perfeita para sua cor', icon: 'science' },
  ];

  const statMeta = [
    { key: 'manufacturers' as const, label: 'Fabricantes', color: 'var(--color-amber-glow)', icon: 'business' },
    { key: 'productLines' as const, label: 'Linhas', color: 'var(--color-azure)', icon: 'category' },
    { key: 'paints' as const, label: 'Tintas', color: 'var(--color-violet)', icon: 'palette' },
    { key: 'equivalences' as const, label: 'Equivalências', color: 'var(--color-emerald)', icon: 'swap_horiz' },
    { key: 'recipes' as const, label: 'Receitas', color: 'var(--color-coral)', icon: 'science' },
  ];
</script>

<div class="page-container">
  <!-- Hero -->
  <div class="page-header animate-artisan-fade">
    <div class="flex items-center gap-4 mb-3">
      <div class="hero-icon">
        <span class="material-icons" style="color: white; font-size: 24px;">palette</span>
      </div>
      <div>
        <h1 class="page-title">Paint Match <span style="color: var(--color-amber-glow);">AI</span></h1>
        <p class="page-subtitle">Base de conhecimento profissional para pintores de miniaturas</p>
      </div>
    </div>
    <div class="page-divider"></div>
  </div>

  <!-- Stats -->
  <div class="grid-5 mb-10">
    {#if loading}
      {#each Array(5) as _}
        <div class="artisan-card p-5">
          <div class="skeleton" style="height: 32px; width: 56px; margin-bottom: 8px;"></div>
          <div class="skeleton" style="height: 12px; width: 64px;"></div>
        </div>
      {/each}
    {:else if stats}
      {#each statMeta as meta, i}
        <div class="artisan-card p-5 animate-artisan-fade" style="animation-delay: {i * 60}ms;">
          <div class="flex items-center gap-3 mb-3">
            <div class="stat-icon" style="background: color-mix(in srgb, {meta.color} 12%, transparent);">
              <span class="material-icons" style="color: {meta.color}; font-size: 18px;">{meta.icon}</span>
            </div>
          </div>
          <div class="font-display text-3xl font-bold mb-1" style="color: {meta.color};">{stats[meta.key]}</div>
          <div style="font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.1em; color: var(--color-obsidian-500);">{meta.label}</div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Quick Actions -->
  <div class="mb-8">
    <h2 class="font-display text-lg font-semibold text-white mb-5 flex items-center gap-3">
      <span>Ações Rápidas</span>
      <div style="flex: 1; height: 1px; background: linear-gradient(90deg, rgba(255,255,255,0.06), transparent);"></div>
    </h2>
    <div class="grid-2">
      {#each quickActions as action, i}
        <Card variant="outlined" class="artisan-card animate-artisan-fade" style="animation-delay: {200 + i * 60}ms;">
          <PrimaryAction onclick={() => onNavigate(action.view)}>
            <Content>
              <div class="flex items-start gap-4">
                <div class="action-icon">
                  <span class="material-icons" style="color: var(--color-amber-glow); font-size: 22px;">{action.icon}</span>
                </div>
                <div>
                  <div class="font-semibold text-white text-sm mb-1">{action.title}</div>
                  <div style="font-size: 13px; color: var(--color-obsidian-400);">{action.desc}</div>
                </div>
              </div>
            </Content>
          </PrimaryAction>
        </Card>
      {/each}
    </div>
  </div>

  <!-- Info Footer -->
  <div class="artisan-card p-4 animate-artisan-fade" style="animation-delay: 500ms;">
    <div class="flex items-center gap-3">
      <div class="stat-icon" style="background: rgba(212,160,83,0.1);">
        <span class="material-icons" style="color: var(--color-amber-glow); font-size: 18px;">info</span>
      </div>
      <div>
        <span class="text-sm font-semibold text-white">Sistema 100% Offline</span>
        <span style="font-size: 13px; color: var(--color-obsidian-400); margin-left: 8px;">
          Dados locais em SQLite. Seed:
          <code class="font-mono" style="font-size: 11px; padding: 2px 6px; border-radius: 4px; background: var(--color-obsidian-800); color: var(--color-amber-glow);">rtk go run ./cmd/seed</code>
        </span>
      </div>
    </div>
  </div>
</div>

<style>
  .hero-icon {
    width: 48px;
    height: 48px;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, var(--color-amber-glow), var(--color-amber-warm));
    box-shadow: 0 4px 20px rgba(212, 160, 83, 0.3);
  }

  .stat-icon {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .action-icon {
    width: 44px;
    height: 44px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background: var(--color-obsidian-800);
    border: 1px solid rgba(255, 255, 255, 0.06);
  }

  :global(.smui-card--outlined.artisan-card) {
    background: linear-gradient(145deg, rgba(255,255,255,0.035), rgba(255,255,255,0.01));
    border-color: rgba(255, 255, 255, 0.06);
    border-radius: 14px;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.smui-card--outlined.artisan-card:hover) {
    background: linear-gradient(145deg, rgba(255,255,255,0.06), rgba(255,255,255,0.025));
    border-color: rgba(212, 160, 83, 0.15);
    box-shadow: 0 8px 32px rgba(0,0,0,0.4), 0 0 0 1px rgba(212,160,83,0.08);
    transform: translateY(-1px);
  }
</style>
