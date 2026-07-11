<script lang="ts">
  // Fabricantes (Tintômetro): lista com hairlines — inicial em círculo
  // grafite, nome display, país mono, 5 amostras representativas do catálogo
  // da marca, contagem mono grande e link "ver catálogo".
  import { onMount } from 'svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  type View = 'home' | 'catalog' | 'manufacturers' | 'color-search' | 'compare' | 'mix' | 'wheel' | 'stock';

  interface NavOpts {
    manufacturer?: string;
  }

  interface Props {
    onNavigate: (view: View, opts?: number | NavOpts) => void;
  }

  let { onNavigate }: Props = $props();

  interface Manufacturer {
    id: number;
    name: string;
    country: string;
    website: string;
    logoPath: string;
    paintCount: number;
  }

  interface Paint {
    id: number;
    manufacturer: string;
    productLine: string;
    r: number;
    g: number;
    b: number;
  }

  let manufacturers: Manufacturer[] = $state([]);
  let loading = $state(true);
  // amostras + linhas por marca, derivadas do catálogo completo
  let chipsByBrand: Map<string, Paint[]> = $state(new Map());
  let linesByBrand: Map<string, string[]> = $state(new Map());

  onMount(async () => {
    try {
      const [mfrs, paints] = await Promise.all([
        PaintService.GetManufacturers(),
        PaintService.GetAllPaints(),
      ]);
      manufacturers = mfrs || [];
      const chips = new Map<string, Paint[]>();
      const lines = new Map<string, Set<string>>();
      for (const p of (paints || []) as Paint[]) {
        const arr = chips.get(p.manufacturer) ?? [];
        // espalha as amostras pelo catálogo (não só as 5 primeiras iguais)
        if (arr.length < 5) {
          arr.push(p);
          chips.set(p.manufacturer, arr);
        }
        if (p.productLine) {
          const ls = lines.get(p.manufacturer) ?? new Set();
          ls.add(p.productLine);
          lines.set(p.manufacturer, ls);
        }
      }
      chipsByBrand = chips;
      linesByBrand = new Map([...lines].map(([k, v]) => [k, [...v].slice(0, 4)]));
    } catch (e) {
      console.error('Erro carregando fabricantes:', e);
    } finally {
      loading = false;
    }
  });

  function metaOf(m: Manufacturer): string {
    const lines = linesByBrand.get(m.name) ?? [];
    const parts = [m.country, lines.join(', ')].filter(Boolean);
    return parts.join(' · ');
  }
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Fabricantes</h1>
    <p class="page-subtitle">{manufacturers.length} marcas, cada uma com seu catálogo e suas linhas.</p>
  </div>

  {#if loading}
    <div class="mfr-list">
      {#each Array(6) as _, i (i)}
        <div class="mfr-row">
          <div class="skeleton" style="width: 46px; height: 46px; border-radius: 50%;"></div>
          <div style="flex: 1; display: flex; flex-direction: column; gap: 8px;">
            <div class="skeleton" style="height: 14px; width: 30%;"></div>
            <div class="skeleton" style="height: 10px; width: 45%;"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="mfr-list animate-rise" style="animation-delay: 60ms;">
      {#each manufacturers as mfr (mfr.id)}
        <div class="mfr-row">
          <span class="mfr-badge font-display">
            {#if mfr.logoPath}
              <img
                src="file://{mfr.logoPath}"
                alt=""
                loading="lazy"
                onerror={(e) => ((e.currentTarget as HTMLImageElement).style.display = 'none')}
              />
            {/if}
            <span class="mfr-initial">{mfr.name.charAt(0).toUpperCase()}</span>
          </span>

          <div class="mfr-id">
            <span class="mfr-name font-display">{mfr.name}</span>
            <span class="mfr-meta font-mono">{metaOf(mfr)}</span>
          </div>

          <div class="mfr-chips" aria-hidden="true">
            {#each chipsByBrand.get(mfr.name) ?? [] as chip (chip.id)}
              <span class="mfr-chip" style="background: rgb({chip.r}, {chip.g}, {chip.b});"></span>
            {/each}
          </div>

          <div class="mfr-count">
            <span class="mfr-count-n font-mono">{mfr.paintCount.toLocaleString('pt-BR')}</span>
            <span class="mfr-count-label">tintas</span>
          </div>

          <button class="mfr-open" onclick={() => onNavigate('catalog', { manufacturer: mfr.name })}>
            ver catálogo ›
          </button>
        </div>
      {/each}
    </div>

    <p class="mfr-foot font-mono">Mostrando {manufacturers.length} de {manufacturers.length} fabricantes</p>
  {/if}
</div>

<style>
  .mfr-list {
    border-top: 1px solid var(--hairline);
  }

  .mfr-row {
    display: flex;
    align-items: center;
    gap: 22px;
    padding: 20px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .mfr-badge {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 46px;
    border-radius: 50%;
    background: var(--grafite);
    overflow: hidden;
    flex-shrink: 0;
  }

  .mfr-badge img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: contain;
    padding: 8px;
    background: var(--grafite);
    z-index: 1;
  }

  .mfr-initial {
    color: var(--papel);
    font-size: 18px;
    font-weight: 700;
  }

  .mfr-id {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    width: 240px;
    flex-shrink: 0;
  }

  .mfr-name {
    font-size: 17px;
    font-weight: 700;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mfr-meta {
    font-size: 11px;
    color: var(--text-2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mfr-chips {
    display: flex;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }

  .mfr-chip {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.06);
    flex-shrink: 0;
  }

  .mfr-count {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
    flex-shrink: 0;
    min-width: 76px;
  }

  .mfr-count-n {
    font-size: 22px;
    font-weight: 600;
    color: var(--grafite);
    letter-spacing: -0.02em;
  }

  .mfr-count-label {
    font-size: 11px;
    color: var(--text-2);
  }

  .mfr-open {
    flex-shrink: 0;
    border: none;
    background: none;
    padding: 4px 0 4px 12px;
    font-size: 13px;
    font-weight: 560;
    color: var(--grafite);
    cursor: pointer;
    white-space: nowrap;
  }

  .mfr-open:hover {
    color: var(--laca-deep);
  }

  .mfr-foot {
    margin-top: 16px;
    font-size: 12px;
    color: var(--text-2);
  }
</style>
