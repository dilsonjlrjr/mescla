<script lang="ts">
  // Home (Tintômetro): "Bancada pronta." + linha de stats em mono gigante,
  // ações em pílulas e a lista de receitas salvas com mini fita de fórmula.
  import { onMount } from 'svelte';
  import FormulaRibbon from './FormulaRibbon.svelte';
  import BrandMark from './BrandMark.svelte';
  import { recipes, removeRecipe, type SavedRecipe } from '../recipes.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface NavOpts {
    paintId?: number;
    targetManufacturerId?: number;
  }

  interface Props {
    onNavigate: (view: View, opts?: number | NavOpts) => void;
  }

  let { onNavigate }: Props = $props();

  let paintCount: number | null = $state(null);
  let mfrCount: number | null = $state(null);
  let stockCount: number | null = $state(null);

  onMount(async () => {
    try {
      const [s, stock] = await Promise.all([
        PaintService.GetStats(),
        PaintService.GetUserPaints(),
      ]);
      paintCount = s.paints;
      mfrCount = s.manufacturers;
      stockCount = (stock || []).length;
    } catch (e) {
      console.error('Erro carregando stats:', e);
    }
  });

  function fmt(n: number | null): string {
    return n === null ? '·' : n.toLocaleString('pt-BR');
  }

  function hexToRgb(hex: string): { r: number; g: number; b: number } {
    const n = parseInt(hex.replace('#', ''), 16) || 0;
    return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
  }

  function openRecipe(r: SavedRecipe) {
    onNavigate('mix', {
      paintId: r.sourcePaintId,
      targetManufacturerId: r.targetManufacturerId || undefined,
    });
  }
</script>

<div class="page-container">
  <div class="home-hero animate-rise">
    <div class="hero-text">
      <h1 class="home-title font-display">Bancada pronta.</h1>
      <p class="home-sub">Tudo que você precisa pra chegar na cor certa antes do pincel.</p>
    </div>
    <div class="hero-mark" aria-hidden="true">
      <BrandMark size={230} />
    </div>
  </div>

  <!-- Stats: números gigantes mono, separados por hairlines verticais -->
  <div class="stat-row animate-rise" style="animation-delay: 60ms;">
    <div class="stat">
      <span class="stat-n font-mono">{fmt(paintCount)}</span>
      <span class="stat-label">tintas no catálogo</span>
    </div>
    <div class="stat">
      <span class="stat-n font-mono">{fmt(mfrCount)}</span>
      <span class="stat-label">fabricantes</span>
    </div>
    <div class="stat">
      <span class="stat-n font-mono">{fmt(stockCount)}</span>
      <span class="stat-label">no meu estoque</span>
    </div>
    <div class="stat">
      <span class="stat-n font-mono">{recipes.length.toLocaleString('pt-BR')}</span>
      <span class="stat-label">receitas salvas</span>
    </div>
  </div>

  <!-- Ações -->
  <div class="home-actions animate-rise" style="animation-delay: 120ms;">
    <button class="pill-dark" onclick={() => onNavigate('mix')}>Gerar fórmula equivalente</button>
    <button class="pill-light" onclick={() => onNavigate('color-search')}>Buscar por cor</button>
    <button class="pill-light" onclick={() => onNavigate('stock')}>Importar estoque CSV</button>
  </div>

  <!-- Receitas salvas -->
  <div class="animate-rise" style="animation-delay: 180ms;">
    <p class="label-mono recipes-label">Receitas salvas</p>

    {#if recipes.length === 0}
      <div class="recipes-empty">
        <div class="empty-ribbon" aria-hidden="true">
          <span style="background: var(--bancada-deep); width: 55%;"></span>
          <span style="background: var(--hairline); width: 30%;"></span>
          <span style="background: var(--bancada-deep); width: 15%;"></span>
        </div>
        <p class="empty-title font-display">Nenhuma receita salva ainda</p>
        <p class="empty-hint">Gere uma fórmula na Equivalência e toque em "Salvar receita". Ela fica aqui, pronta pra reabrir na bancada.</p>
        <button class="pill-dark" onclick={() => onNavigate('mix')}>Gerar a primeira fórmula</button>
      </div>
    {:else}
      <div class="recipes-list">
        {#each recipes as r (r.id)}
          <div class="recipe-row">
            <div class="recipe-ribbon">
              <FormulaRibbon
                segments={r.ingredients.map(i => ({ ...hexToRgb(i.hex), code: i.code, percentage: i.percentage }))}
                height={30}
                ruler={false}
                labels={false}
              />
            </div>
            <div class="recipe-id">
              <span class="recipe-name">{r.sourceName}</span>
              <span class="recipe-target font-mono">em {r.targetManufacturer}</span>
            </div>
            <span class="recipe-delta font-mono">{r.deltaE.toFixed(1)}</span>
            <button class="recipe-open" onclick={() => openRecipe(r)}>abrir ›</button>
            <button class="recipe-del font-mono" onclick={() => removeRecipe(r.id)} aria-label="Excluir receita" title="Excluir">×</button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .home-hero {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 32px;
    padding-top: 20px;
    margin-bottom: 44px;
  }

  /* A gota da marca ancora o lado direito do hero (decorativa).
     Margem negativa: a gota pode vazar do fluxo sem empurrar os stats pra baixo. */
  .hero-mark {
    flex-shrink: 0;
    padding-right: 24px;
    margin: -20px 0 -40px;
  }

  @media (max-width: 860px) {
    .hero-mark {
      display: none;
    }
  }

  .home-title {
    font-size: clamp(2.6rem, 5.4vw, 3.5rem);
    font-weight: 760;
    color: var(--grafite);
    letter-spacing: -0.02em;
    line-height: 1.05;
    margin-bottom: 12px;
  }

  .home-sub {
    font-size: 14.5px;
    color: var(--text-2);
  }

  .stat-row {
    display: flex;
    margin-bottom: 44px;
    flex-wrap: wrap;
    row-gap: 20px;
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-right: 40px;
    margin-right: 40px;
    border-right: 1px solid var(--hairline);
  }

  .stat:last-child {
    border-right: none;
    margin-right: 0;
    padding-right: 0;
  }

  .stat-n {
    font-size: clamp(34px, 4vw, 48px);
    font-weight: 600;
    color: var(--grafite);
    letter-spacing: -0.03em;
    line-height: 1;
  }

  .stat-label {
    font-size: 12.5px;
    color: var(--text-2);
  }

  .home-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 56px;
  }

  .recipes-label {
    padding-bottom: 14px;
    border-bottom: 1px solid var(--hairline);
    display: block;
  }

  .recipes-list {
    display: flex;
    flex-direction: column;
  }

  .recipe-row {
    display: flex;
    align-items: center;
    gap: 24px;
    padding: 18px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .recipe-ribbon {
    width: 220px;
    flex-shrink: 0;
  }

  .recipe-id {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .recipe-name {
    font-size: 15px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recipe-target {
    font-size: 11.5px;
    color: var(--text-2);
  }

  .recipe-delta {
    flex-shrink: 0;
    font-size: 22px;
    font-weight: 600;
    color: var(--grafite);
    letter-spacing: -0.02em;
  }

  .recipe-open {
    flex-shrink: 0;
    border: none;
    background: none;
    padding: 4px 2px;
    font-size: 13px;
    font-weight: 600;
    color: var(--laca-deep);
    cursor: pointer;
  }

  .recipe-open:hover {
    text-decoration: underline;
  }

  .recipe-del {
    flex-shrink: 0;
    border: none;
    background: none;
    padding: 4px 6px;
    font-size: 14px;
    color: var(--text-3);
    cursor: pointer;
  }

  .recipe-del:hover {
    color: var(--grafite);
  }

  /* Estado vazio composto */
  .recipes-empty {
    padding: 40px 0 24px;
    max-width: 420px;
  }

  .empty-ribbon {
    display: flex;
    width: 220px;
    height: 30px;
    margin-bottom: 22px;
    overflow: hidden;
  }

  .recipes-empty .empty-title {
    font-size: 19px;
    font-weight: 700;
    color: var(--grafite);
    margin-bottom: 6px;
  }

  .recipes-empty .empty-hint {
    font-size: 13px;
    color: var(--text-2);
    line-height: 1.55;
    margin-bottom: 20px;
  }
</style>
