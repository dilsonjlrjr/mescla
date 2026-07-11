<script lang="ts">
  // Cor (Tintômetro): parta de uma cor exata e encontre o que existe em tinta.
  // Painel da cor-alvo chapado à esquerda; à direita, as mais próximas por ΔE00.
  import PaintBottle from './PaintBottle.svelte';
  import { contrastOn, hexOf, deltaIsGood } from '../ui';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  let targetR = $state(46);
  let targetG = $state(125);
  let targetB = $state(107);
  let hexInput = $state('#2E7D6B');
  let results: any[] = $state([]);
  let searching = $state(false);
  let hasSearched = $state(false);
  let shown = $state(6);

  let targetHex = $derived(hexOf(targetR, targetG, targetB));

  function applyHex() {
    const m = /^#?([0-9a-fA-F]{6})$/.exec(hexInput.trim());
    if (!m) return;
    const n = parseInt(m[1], 16);
    targetR = (n >> 16) & 255;
    targetG = (n >> 8) & 255;
    targetB = n & 255;
  }

  function syncHex() {
    hexInput = targetHex;
  }

  async function doSearch() {
    applyHex();
    searching = true;
    hasSearched = true;
    shown = 6;
    try {
      results = (await PaintService.FindSimilar(targetR, targetG, targetB, 100, 30)) || [];
    } catch (e) {
      console.error('Erro buscando similar:', e);
      results = [];
    } finally {
      searching = false;
    }
  }

  const channels = [
    { key: 'R', get: () => targetR, set: (v: number) => (targetR = v) },
    { key: 'G', get: () => targetG, set: (v: number) => (targetG = v) },
    { key: 'B', get: () => targetB, set: (v: number) => (targetB = v) },
  ];
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Cor</h1>
    <p class="page-subtitle">Parta de uma cor exata e encontre o que existe em tinta.</p>
  </div>

  <div class="color-layout">
    <!-- Cor alvo -->
    <div class="picker animate-rise" style="animation-delay: 60ms;">
      <div class="target-panel" style="background: rgb({targetR}, {targetG}, {targetB}); color: {contrastOn(targetR, targetG, targetB)};">
        <span class="label-mono inherit">Cor alvo</span>
        <span class="target-hex font-mono">{targetHex}</span>
      </div>

      <div class="sliders">
        {#each channels as ch (ch.key)}
          <label class="slider-row">
            <span class="chan">{ch.key}</span>
            <input
              type="range"
              min="0"
              max="255"
              step="1"
              value={ch.get()}
              oninput={(e) => { ch.set(Number((e.currentTarget as HTMLInputElement).value)); syncHex(); }}
              aria-label="Canal {ch.key}"
            />
            <span class="chan-val font-mono">{ch.get()}</span>
          </label>
        {/each}
      </div>

      <div class="hex-row">
        <input
          type="text"
          class="hex-field font-mono"
          bind:value={hexInput}
          maxlength="7"
          spellcheck="false"
          aria-label="Cor em hexadecimal"
          onchange={applyHex}
          onkeydown={(e) => e.key === 'Enter' && doSearch()}
        />
        <button class="pill-dark" onclick={doSearch} disabled={searching}>
          {searching ? 'Buscando…' : 'Buscar no catálogo'}
        </button>
      </div>

      <p class="picker-hint">Dica: cole um hex direto do seu software de pintura digital.</p>
    </div>

    <!-- Resultados -->
    <div class="results animate-rise" style="animation-delay: 120ms;">
      <p class="label-mono results-label">Mais próximas · ΔE00</p>

      {#if searching}
        <div class="results-skel">
          {#each Array(4) as _, i (i)}
            <div class="skel-row">
              <div class="skeleton" style="width: 52px; height: 52px;"></div>
              <div style="flex: 1; display: flex; flex-direction: column; gap: 8px;">
                <div class="skeleton" style="height: 12px; width: 40%;"></div>
                <div class="skeleton" style="height: 10px; width: 28%;"></div>
              </div>
              <div class="skeleton" style="width: 48px; height: 22px;"></div>
            </div>
          {/each}
        </div>
      {:else if hasSearched && results.length === 0}
        <div class="notice-card mt-4">
          <div class="notice-title">Nenhuma tinta por perto</div>
          <p class="notice-text">O catálogo não tem nada próximo dessa cor. Ajuste os canais e busque de novo.</p>
        </div>
      {:else if results.length > 0}
        <div class="result-list">
          {#each results.slice(0, shown) as r, i (r.paintId)}
            <div class="result-row">
              <span class="swatch-flat" style="width: 52px; height: 52px; background: rgb({r.r}, {r.g}, {r.b});"></span>
              <PaintBottle r={r.r} g={r.g} b={r.b} size={44} />
              <div class="result-text">
                <span class="result-name">{r.name}</span>
                <span class="result-meta font-mono">{r.manufacturer}</span>
              </div>
              <div class="result-delta">
                <span class="delta-reading" class:good={i === 0 && deltaIsGood(r.deltaE)} style="font-size: 26px;">{r.deltaE.toFixed(1)}</span>
                {#if i === 0}
                  <span class="best-tag font-mono">melhor</span>
                {/if}
              </div>
            </div>
          {/each}

          {#if results.length > shown}
            <button class="more-row" onclick={() => (shown = results.length)}>
              Ver mais {results.length - shown} resultados
            </button>
          {/if}
        </div>
      {:else}
        <div class="empty-state">
          <p class="empty-title">Monte a cor ao lado</p>
          <p class="empty-hint">Arraste os canais R, G e B ou cole um hex. Depois, busque no catálogo.</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .color-layout {
    display: grid;
    grid-template-columns: minmax(320px, 460px) minmax(0, 1fr);
    gap: 56px;
  }

  @media (max-width: 900px) {
    .color-layout {
      grid-template-columns: 1fr;
    }
  }

  /* Painel da cor-alvo: raio 12, chapado */
  .target-panel {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    height: 240px;
    border-radius: var(--radius-surface);
    padding: 22px 24px;
    margin-bottom: 28px;
  }

  .label-mono.inherit {
    color: inherit;
    opacity: 0.85;
  }

  .target-hex {
    font-size: 15px;
    font-weight: 600;
  }

  .sliders {
    display: flex;
    flex-direction: column;
    gap: 20px;
    margin-bottom: 28px;
  }

  .slider-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .chan {
    width: 16px;
    font-size: 13px;
    font-weight: 700;
    color: var(--grafite);
    flex-shrink: 0;
  }

  .chan-val {
    width: 40px;
    text-align: right;
    font-size: 13px;
    color: var(--grafite);
    flex-shrink: 0;
  }

  .hex-row {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .hex-field {
    width: 132px;
    height: 46px;
    padding: 0 14px;
    font-size: 14px;
    text-transform: uppercase;
  }

  .picker-hint {
    margin-top: 16px;
    font-size: 12.5px;
    color: var(--text-3);
  }

  .results-label {
    display: block;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--hairline);
  }

  .result-list {
    display: flex;
    flex-direction: column;
  }

  .result-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 15px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .result-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .result-name {
    font-size: 14.5px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-meta {
    font-size: 11.5px;
    color: var(--text-2);
  }

  .result-delta {
    display: flex;
    align-items: baseline;
    gap: 10px;
    flex-shrink: 0;
  }

  .best-tag {
    font-size: 10.5px;
    color: var(--laca-deep);
  }

  .more-row {
    padding: 16px 0;
    border: none;
    border-bottom: 1px solid var(--hairline);
    background: none;
    text-align: left;
    font: inherit;
    font-size: 14px;
    font-weight: 700;
    color: var(--grafite);
    cursor: pointer;
  }

  .more-row:hover {
    color: var(--laca-deep);
  }

  .results-skel {
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
