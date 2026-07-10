<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import BrandMark from './BrandMark.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix';

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
      const stride = Math.max(1, Math.floor(pool.length / 10));
      shelf = pool.filter((_: Paint, i: number) => i % stride === 0).slice(0, 10);
    } catch (e) {
      console.error('Erro carregando stats:', e);
    } finally {
      loading = false;
    }
  });

  const steps = [
    { n: '1', title: 'Escolha a tinta', desc: 'A cor que você viu num tutorial, numa caixa ou que acabou no pote.' },
    { n: '2', title: 'Escolha sua marca', desc: 'A Mescla busca a melhor mistura usando só as tintas dela.' },
    { n: '3', title: 'Misture com confiança', desc: 'Percentuais, selo de proximidade honesto e dicas de ajuste.' },
  ];

  const tools: { view: View; title: string; desc: string; icon: 'grid' | 'pipette' | 'swap' | 'building' }[] = [
    { view: 'catalog', title: 'Catálogo', desc: 'Todas as tintas, filtráveis por marca', icon: 'grid' },
    { view: 'color-search', title: 'Buscar cor', desc: 'Da cor exata pra tinta mais próxima', icon: 'pipette' },
    { view: 'compare', title: 'Comparar', desc: 'Até 6 tintas lado a lado', icon: 'swap' },
    { view: 'manufacturers', title: 'Marcas', desc: 'Quem fabrica o quê', icon: 'building' },
  ];
</script>

<div class="page-container">
  <!-- Hero: a tese do produto -->
  <div class="hero animate-rise">
    <div class="hero-brand">
      <BrandMark size={52} />
      <div class="hero-word font-display">Mescla</div>
    </div>
    <h1 class="hero-thesis font-display">
      Você tem a cor em <em>uma</em> marca.<br />
      Precisa dela em <em>outra</em>.
    </h1>
    <p class="hero-sub">
      {#if loading}
        Carregando o catálogo…
      {:else if stats}
        {stats.paints.toLocaleString('pt-BR')} tintas de {stats.manufacturers} marcas, comparadas como o olho vê — offline.
      {/if}
    </p>
    <button class="btn-primary hero-cta" onclick={() => onNavigate('mix')}>
      <Icon name="flask" size={17} />
      Encontrar equivalência
    </button>
  </div>

  <!-- Como funciona (guia inline, sempre visível) -->
  <div class="mb-10 animate-rise" style="animation-delay: 80ms;">
    <h2 class="section-title">Como funciona</h2>
    <div class="steps">
      {#each steps as s}
        <div class="step">
          <div class="step-n">{s.n}</div>
          <div class="step-title">{s.title}</div>
          <div class="step-desc">{s.desc}</div>
        </div>
      {/each}
    </div>
  </div>

  <!-- Prateleira: amostra viva do catálogo -->
  {#if shelf.length > 0}
    <div class="mb-10 animate-rise" style="animation-delay: 140ms;">
      <h2 class="section-title">Na prateleira</h2>
      <div class="shelf">
        {#each shelf as paint, i}
          <button class="shelf-swatch" style="animation-delay: {i * 30}ms;" title="{paint.name} — {paint.manufacturer}" onclick={() => onNavigate('catalog')}>
            <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={58} />
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Demais ferramentas -->
  <div class="mb-8 animate-rise" style="animation-delay: 200ms;">
    <h2 class="section-title">Ferramentas</h2>
    <div class="tools-grid">
      {#each tools as tool}
        <button class="row-card" onclick={() => onNavigate(tool.view)}>
          <div class="action-icon"><Icon name={tool.icon} size={20} /></div>
          <div class="flex-1" style="min-width: 0;">
            <div class="action-title">{tool.title}</div>
            <div class="action-desc">{tool.desc}</div>
          </div>
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .hero {
    padding: 36px 0 40px;
    margin-bottom: 40px;
    border-bottom: 1px solid var(--ink-700);
  }

  .hero-brand {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 26px;
  }

  .hero-word {
    font-size: 21px;
    font-weight: 700;
    color: var(--paper);
    letter-spacing: -0.01em;
  }

  .hero-thesis {
    font-size: clamp(1.9rem, 4.5vw, 2.9rem);
    font-weight: 680;
    line-height: 1.14;
    color: var(--paper);
    letter-spacing: -0.02em;
    margin-bottom: 14px;
    max-width: 640px;
  }

  .hero-thesis em {
    font-style: italic;
    color: var(--lacquer-deep);
  }

  .hero-sub {
    font-size: 14px;
    color: var(--ink-500);
    margin-bottom: 26px;
  }

  .hero-cta {
    max-width: 300px;
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
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 0;
    transition: transform 0.15s ease;
    animation: rise-in 0.4s cubic-bezier(0.4, 0, 0.2, 1) both;
  }

  .shelf-swatch:hover {
    transform: translateY(-3px);
  }

  .tools-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
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
    color: var(--ink-100);
    margin-bottom: 2px;
  }

  .action-desc {
    font-size: 12.5px;
    color: var(--ink-500);
  }
</style>
