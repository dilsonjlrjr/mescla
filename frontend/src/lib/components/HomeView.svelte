<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

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

  interface Paint {
    id: number;
    name: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  let stats: Stats | null = $state(null);
  let shelf: Paint[] = $state([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      const [s, all] = await Promise.all([
        PaintService.GetStats(),
        PaintService.GetAllPaints(),
      ]);
      stats = s;
      const pool = all || [];
      const stride = Math.max(1, Math.floor(pool.length / 8));
      shelf = pool.filter((_: Paint, i: number) => i % stride === 0).slice(0, 8);
    } catch (e) {
      console.error('Erro carregando stats:', e);
      stats = { manufacturers: 11, productLines: 32, paints: 45, equivalences: 0, recipes: 0 };
    } finally {
      loading = false;
    }
  });

  const quickActions: { view: View; title: string; desc: string; icon: 'grid' | 'pipette' | 'swap' | 'flask' }[] = [
    { view: 'catalog', title: 'Catálogo completo', desc: 'Explore todas as tintas cadastradas por fabricante', icon: 'grid' },
    { view: 'color-search', title: 'Buscar por cor', desc: 'Encontre tintas similares usando Delta E 2000', icon: 'pipette' },
    { view: 'compare', title: 'Comparar tintas', desc: 'Compare cores lado a lado, até 6 por vez', icon: 'swap' },
    { view: 'mix', title: 'Receita de mistura', desc: 'Descubra a fórmula pra chegar em qualquer cor', icon: 'flask' },
  ];

  const ledger: { key: keyof Stats; label: string; icon: 'building' | 'layers' | 'palette' | 'swap' | 'flask'; view: View }[] = [
    { key: 'manufacturers', label: 'Fabricantes', icon: 'building', view: 'catalog' },
    { key: 'productLines', label: 'Linhas', icon: 'layers', view: 'catalog' },
    { key: 'paints', label: 'Tintas', icon: 'palette', view: 'catalog' },
    { key: 'equivalences', label: 'Equivalências', icon: 'swap', view: 'compare' },
    { key: 'recipes', label: 'Receitas', icon: 'flask', view: 'mix' },
  ];
</script>

<div class="page-container">
  <!-- Hero -->
  <div class="page-header animate-rise">
    <div class="eyebrow mb-2">Bancada de pintura</div>
    <div class="flex items-center gap-4">
      <div class="hero-mark"></div>
      <div>
        <h1 class="page-title">Paint Match AI</h1>
        <p class="page-subtitle">Base de conhecimento profissional pra pintores de miniaturas</p>
      </div>
    </div>
    <div class="page-divider"></div>
  </div>

  <!-- Ledger -->
  <div class="ledger mb-10 animate-rise" style="animation-delay: 60ms;">
    {#each ledger as item, i}
      <button class="ledger-col" style={i > 0 ? 'border-left: 1px solid var(--ink-700);' : ''} onclick={() => onNavigate(item.view)}>
        <span class="ledger-icon"><Icon name={item.icon} size={15} /></span>
        {#if loading}
          <div class="skeleton" style="height: 30px; width: 40px; margin: 6px 0;"></div>
        {:else if stats}
          <div class="ledger-value">{stats[item.key]}</div>
        {/if}
        <div class="ledger-label">{item.label}</div>
      </button>
    {/each}
  </div>

  <!-- Shelf -->
  {#if shelf.length > 0}
    <div class="mb-10 animate-rise" style="animation-delay: 100ms;">
      <h2 class="section-title">Na prateleira</h2>
      <div class="shelf">
        {#each shelf as paint, i}
          <button class="shelf-swatch swatch-flat" style="background: rgb({paint.r}, {paint.g}, {paint.b}); animation-delay: {i * 30}ms;" title="{paint.name} — {paint.manufacturer}" onclick={() => onNavigate('catalog')}></button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Quick actions -->
  <div class="mb-8">
    <h2 class="section-title">Ferramentas</h2>
    <div class="actions-list">
      {#each quickActions as action, i}
        <button class="row-card animate-rise" style="animation-delay: {140 + i * 50}ms;" onclick={() => onNavigate(action.view)}>
          <div class="action-icon"><Icon name={action.icon} size={20} /></div>
          <div class="flex-1">
            <div class="action-title">{action.title}</div>
            <div class="action-desc">{action.desc}</div>
          </div>
          <div class="action-arrow"><Icon name="chevron-left" size={16} /></div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Info footer -->
  <div class="panel p-4 animate-rise" style="animation-delay: 380ms;">
    <div class="flex items-center gap-3">
      <span style="color: var(--ink-500);"><Icon name="info" size={18} /></span>
      <div>
        <span class="text-sm font-semibold text-white">Sistema 100% offline</span>
        <span style="font-size: 13px; color: var(--ink-500); margin-left: 8px;">
          Dados locais em SQLite. Seed:
          <code class="font-mono" style="font-size: 11px; padding: 2px 6px; border-radius: 4px; background: var(--ink-800); color: var(--lacquer-tint);">rtk go run ./cmd/seed</code>
        </span>
      </div>
    </div>
  </div>
</div>

<style>
  .hero-mark {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--lacquer);
    box-shadow: inset 0 -4px 7px rgba(0, 0, 0, 0.25), inset 0 3px 4px rgba(255, 255, 255, 0.18);
  }

  .ledger {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    border-radius: 10px;
  }

  .ledger-col {
    display: block;
    width: 100%;
    padding: 18px 20px;
    text-align: left;
    background: transparent;
    border: none;
    border-radius: 0;
    cursor: pointer;
    font: inherit;
    color: inherit;
    transition: background 0.15s ease;
  }

  .ledger-col:hover {
    background: var(--ink-850);
  }

  .ledger-icon {
    display: inline-flex;
    color: var(--ink-500);
    margin-bottom: 10px;
  }

  .ledger-value {
    font-family: var(--font-display);
    font-size: 1.875rem;
    font-weight: 600;
    color: var(--paper);
    line-height: 1;
    margin-bottom: 6px;
  }

  .ledger-label {
    font-size: 10.5px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--ink-500);
  }

  .section-title {
    font-family: var(--font-display);
    font-size: 1.05rem;
    font-weight: 600;
    color: var(--paper);
    margin-bottom: 16px;
  }

  .shelf {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }

  .shelf-swatch {
    width: 46px;
    height: 46px;
    border: none;
    cursor: pointer;
    padding: 0;
    transition: transform 0.15s ease;
    animation: rise-in 0.4s cubic-bezier(0.4, 0, 0.2, 1) both;
  }

  .shelf-swatch:hover {
    transform: translateY(-3px);
  }

  .actions-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .action-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: 8px;
    flex-shrink: 0;
    background: var(--ink-800);
    color: var(--lacquer);
  }

  .action-title {
    font-weight: 600;
    font-size: 14px;
    color: var(--paper);
    margin-bottom: 2px;
  }

  .action-desc {
    font-size: 12.5px;
    color: var(--ink-500);
  }

  .action-arrow {
    display: flex;
    flex-shrink: 0;
    color: var(--ink-600);
    transform: rotate(180deg);
    transition: transform 0.15s ease, color 0.15s ease;
  }

  :global(.row-card:hover) .action-arrow {
    transform: rotate(180deg) translateX(3px);
    color: var(--lacquer);
  }
</style>
