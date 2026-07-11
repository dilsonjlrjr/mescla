<script lang="ts">
  // Comparar (Tintômetro): todas contra a âncora, ordenadas por distância de
  // cor. Painel-âncora chapado à esquerda; à direita, amostras duplas
  // (metade âncora, metade candidata) com ΔE00 e verdicto.
  import { onMount } from 'svelte';
  import PaintSearchInput from './PaintSearchInput.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { contrastOn, hexOf, deltaIsGood } from '../ui';
  import { rgbToHsl } from '../color/theory';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    productLine?: string;
    r: number;
    g: number;
    b: number;
  }

  interface Props {
    initialAnchorId?: number | null;
  }

  let { initialAnchorId = null }: Props = $props();

  let allPaints: Paint[] = $state([]);
  let anchor: Paint | null = $state(null);
  let candidateIds: number[] = $state([]);
  let results: any[] = $state([]);
  let comparing = $state(false);
  let pickingAnchor = $state(false);
  let addingPaint = $state(false);

  onMount(async () => {
    try {
      allPaints = (await PaintService.GetAllPaints()) || [];
      if (initialAnchorId) {
        anchor = allPaints.find(p => p.id === initialAnchorId) || null;
      }
    } catch (e) {
      console.error('Erro:', e);
    }
  });

  async function doCompare() {
    if (!anchor || candidateIds.length === 0) {
      results = [];
      return;
    }
    comparing = true;
    try {
      const raw = (await PaintService.CompareColors([anchor.id, ...candidateIds])) || [];
      results = raw
        .filter((r: any) => r.paintId !== anchor!.id)
        .sort((a: any, b: any) => a.deltaE - b.deltaE);
    } catch (e) {
      console.error('Erro comparando:', e);
      results = [];
    } finally {
      comparing = false;
    }
  }

  function setAnchor(p: Paint) {
    anchor = p;
    pickingAnchor = false;
    candidateIds = candidateIds.filter(id => id !== p.id);
    doCompare();
  }

  function addCandidate(p: Paint) {
    addingPaint = false;
    if (!anchor) {
      anchor = p;
      return;
    }
    if (p.id === anchor.id || candidateIds.includes(p.id) || candidateIds.length >= 6) return;
    candidateIds = [...candidateIds, p.id];
    doCompare();
  }

  function removeCandidate(paintId: number) {
    candidateIds = candidateIds.filter(id => id !== paintId);
    results = results.filter((r: any) => r.paintId !== paintId);
  }

  const hueNames: { max: number; name: string }[] = [
    { max: 20, name: 'o vermelho' },
    { max: 50, name: 'o laranja' },
    { max: 70, name: 'o amarelo' },
    { max: 160, name: 'o verde' },
    { max: 200, name: 'o ciano' },
    { max: 250, name: 'o azul' },
    { max: 290, name: 'o violeta' },
    { max: 345, name: 'o magenta' },
    { max: 361, name: 'o vermelho' },
  ];

  // Verdicto: escala de ΔE00 + comparação HSL simples (claridade, matiz).
  function verdictOf(r: any): string {
    if (!anchor) return '';
    if (r.deltaE < 2) return 'Equivalência excelente';
    if (r.deltaE < 4) return 'Boa sob luz de bancada';
    const a = rgbToHsl({ r: anchor.r, g: anchor.g, b: anchor.b });
    const c = rgbToHsl({ r: r.r, g: r.g, b: r.b });
    if (r.deltaE < 8) {
      const dl = c.l - a.l;
      if (dl > 0.06) return 'Visivelmente mais clara';
      if (dl < -0.06) return 'Visivelmente mais escura';
      return 'Diferença perceptível';
    }
    const hue = hueNames.find(h => c.h < h.max)?.name ?? 'outro matiz';
    return `Distante: puxa pra ${hue}`;
  }
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Comparar</h1>
    <p class="page-subtitle">Todas contra a âncora, ordenadas por distância de cor.</p>
  </div>

  <div class="compare-layout">
    <!-- Âncora -->
    <div class="anchor-side animate-rise" style="animation-delay: 60ms;">
      {#if anchor && !pickingAnchor}
        <div class="anchor-panel" style="background: rgb({anchor.r}, {anchor.g}, {anchor.b}); color: {contrastOn(anchor.r, anchor.g, anchor.b)};">
          <span class="label-mono inherit">Âncora</span>
          <div>
            <h2 class="anchor-name font-display">{anchor.name}</h2>
            <p class="anchor-meta">{anchor.manufacturer}{anchor.productLine ? ` · ${anchor.productLine}` : ''}</p>
            <p class="anchor-hex font-mono">{hexOf(anchor.r, anchor.g, anchor.b)}</p>
          </div>
        </div>
        <button class="pill-light mt-4" onclick={() => (pickingAnchor = true)}>Trocar âncora</button>
      {:else}
        <div class="anchor-empty">
          <span class="label-mono">Âncora</span>
          <p class="anchor-pick-title font-display">Escolha a tinta de referência.</p>
          <PaintSearchInput
            paints={allPaints}
            selected={null}
            onSelect={setAnchor}
            onClear={() => {}}
            label="Nome, código ou marca"
          />
          {#if anchor}
            <button class="cancel-pick font-mono" onclick={() => (pickingAnchor = false)}>cancelar</button>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Candidatas -->
    <div class="candidates animate-rise" style="animation-delay: 120ms;">
      <p class="label-mono cand-label">ΔE00</p>

      {#if comparing}
        <div class="cand-skel">
          {#each Array(3) as _, i (i)}
            <div class="skel-row">
              <div class="skeleton" style="width: 168px; height: 76px; border-radius: 0;"></div>
              <div style="flex: 1; display: flex; flex-direction: column; gap: 8px;">
                <div class="skeleton" style="height: 12px; width: 40%;"></div>
                <div class="skeleton" style="height: 10px; width: 28%;"></div>
              </div>
              <div class="skeleton" style="width: 60px; height: 26px;"></div>
            </div>
          {/each}
        </div>
      {:else}
        {#each results as r (r.paintId)}
          <div class="cand-row">
            <span class="dual-swatch" aria-hidden="true">
              <span style="background: rgb({anchor?.r}, {anchor?.g}, {anchor?.b});"></span>
              <span style="background: rgb({r.r}, {r.g}, {r.b});"></span>
            </span>
            <PaintBottle r={r.r} g={r.g} b={r.b} size={48} />
            <div class="cand-text">
              <span class="cand-name">{r.name}</span>
              <span class="cand-meta font-mono">{r.manufacturer}</span>
            </div>
            <span class="delta-reading" class:good={deltaIsGood(r.deltaE)} style="font-size: 30px;">{r.deltaE.toFixed(1)}</span>
            <span class="cand-verdict" class:good-text={deltaIsGood(r.deltaE)}>{verdictOf(r)}</span>
            <button class="cand-del font-mono" onclick={() => removeCandidate(r.paintId)} aria-label="Remover da comparação" title="Remover">×</button>
          </div>
        {/each}

        {#if anchor && results.length === 0}
          <div class="cand-empty">
            <p class="empty-title font-display">Adicione tintas pra comparar</p>
            <p class="empty-hint">Cada tinta entra na lista com a distância de cor até a âncora.</p>
          </div>
        {/if}

        {#if !anchor}
          <div class="cand-empty">
            <p class="empty-title font-display">Comece pela âncora</p>
            <p class="empty-hint">Escolha a tinta de referência no painel ao lado.</p>
          </div>
        {/if}

        {#if anchor}
          {#if addingPaint}
            <div class="add-picker">
              <PaintSearchInput
                paints={allPaints}
                selected={null}
                onSelect={addCandidate}
                onClear={() => {}}
                label="Nome, código ou marca"
              />
              <button class="cancel-pick font-mono" onclick={() => (addingPaint = false)}>cancelar</button>
            </div>
          {:else}
            <button class="pill-light add-row" onclick={() => (addingPaint = true)} disabled={candidateIds.length >= 6}>
              + Adicionar tinta à comparação
            </button>
          {/if}
        {/if}
      {/if}
    </div>
  </div>
</div>

<style>
  .compare-layout {
    display: grid;
    grid-template-columns: 360px minmax(0, 1fr);
    gap: 56px;
    align-items: start;
  }

  @media (max-width: 900px) {
    .compare-layout {
      grid-template-columns: 1fr;
    }
  }

  .anchor-panel {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    height: 560px;
    border-radius: var(--radius-surface);
    padding: 24px 26px 28px;
  }

  .label-mono.inherit {
    color: inherit;
    opacity: 0.85;
  }

  .anchor-name {
    font-size: clamp(1.9rem, 3vw, 2.6rem);
    font-weight: 720;
    line-height: 1.05;
    letter-spacing: -0.02em;
    margin-bottom: 10px;
    overflow-wrap: anywhere;
  }

  .anchor-meta {
    font-size: 13.5px;
    font-weight: 560;
    margin-bottom: 6px;
  }

  .anchor-hex {
    font-size: 12px;
    opacity: 0.9;
  }

  .anchor-empty {
    padding: 24px;
    border: 1px dashed var(--hairline);
    border-radius: var(--radius-surface);
  }

  .anchor-pick-title {
    font-size: 20px;
    font-weight: 700;
    color: var(--grafite);
    margin: 14px 0 16px;
  }

  .cancel-pick {
    margin-top: 12px;
    border: none;
    background: none;
    padding: 0;
    font-size: 11.5px;
    color: var(--text-2);
    cursor: pointer;
  }

  .cancel-pick:hover {
    color: var(--grafite);
  }

  .cand-label {
    display: block;
    text-align: right;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--hairline);
  }

  .cand-row {
    display: flex;
    align-items: center;
    gap: 18px;
    padding: 16px 0;
    border-bottom: 1px solid var(--hairline);
  }

  /* Amostra dupla: metade âncora, metade candidata — raio 0, junção limpa */
  .dual-swatch {
    display: flex;
    width: 168px;
    height: 76px;
    flex-shrink: 0;
  }

  .dual-swatch > span {
    display: block;
    width: 50%;
    height: 100%;
  }

  .cand-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .cand-name {
    font-size: 14.5px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cand-meta {
    font-size: 11.5px;
    color: var(--text-2);
  }

  .cand-verdict {
    flex-shrink: 0;
    width: 190px;
    font-size: 12.5px;
    color: var(--text-2);
  }

  .cand-verdict.good-text {
    color: var(--laca-deep);
    font-weight: 600;
  }

  .cand-del {
    flex-shrink: 0;
    border: none;
    background: none;
    padding: 4px 6px;
    font-size: 14px;
    color: var(--text-3);
    cursor: pointer;
  }

  .cand-del:hover {
    color: var(--grafite);
  }

  .add-row {
    width: 100%;
    margin-top: 18px;
  }

  .add-picker {
    margin-top: 18px;
  }

  .cand-empty {
    padding: 40px 0;
  }

  .cand-empty .empty-title {
    font-size: 19px;
    font-weight: 700;
    color: var(--grafite);
    margin-bottom: 6px;
  }

  .cand-empty .empty-hint {
    font-size: 13px;
    color: var(--text-2);
  }

  .cand-skel {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding-top: 18px;
  }

  .skel-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }
</style>
