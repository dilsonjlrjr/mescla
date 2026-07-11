<script lang="ts">
  // Cor — "tenho uma cor de referência, que tinta chega perto?"
  // Preview gigante no topo (o usuário olha a COR, não os números), sliders
  // com gradiente real do canal, campo HEX e resultados ao vivo.
  import Icon from '../components/Icon.svelte';
  import DeltaBadge from '../components/DeltaBadge.svelte';
  import DeltaScaleSheet from '../components/DeltaScaleSheet.svelte';
  import PaintDetailSheet from '../components/PaintDetailSheet.svelte';
  import { switchTab } from '../nav.svelte';
  import { allManufacturers, paintById, type Paint } from '../services/catalog';
  import { findSimilar, type SearchResult } from '../services/engine';
  import { shelf } from '../services/shelf.svelte';
  import { appState } from '../appState.svelte';

  let r = $state(180);
  let g = $state(120);
  let b = $state(60);
  let hex = $state('');
  let onlyShelf = $state(false);
  let results: SearchResult[] = $state([]);
  let searching = $state(false);
  let deltaSheetOpen = $state(false);
  let deltaSheetValue: number | null = $state(null);
  let detailPaint: Paint | null = $state(null);

  // Preset vindo de outra aba ("ver tintas prontas mais próximas")
  $effect(() => {
    if (appState.corPreset) {
      ({ r, g, b } = appState.corPreset);
      appState.corPreset = null;
    }
  });

  let currentHex = $derived(
    '#' + [r, g, b].map(n => n.toString(16).padStart(2, '0').toUpperCase()).join('')
  );

  function applyHex() {
    const m = hex.trim().replace(/^#/, '');
    if (!/^[0-9a-fA-F]{6}$/.test(m)) return;
    r = parseInt(m.slice(0, 2), 16);
    g = parseInt(m.slice(2, 4), 16);
    b = parseInt(m.slice(4, 6), 16);
    hex = '';
  }

  // Busca ao vivo (debounce 150ms) — o resultado acompanha o dedo no slider.
  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    const rr = r, gg = g, bb = b;
    clearTimeout(timer);
    timer = setTimeout(async () => {
      searching = true;
      try {
        results = await findSimilar(rr, gg, bb, 100, 24);
      } catch (e) {
        console.error('findSimilar:', e);
        results = [];
      } finally {
        searching = false;
      }
    }, 150);
  });

  let shelfNames = $derived(
    allManufacturers().filter(m => shelf.manufacturerIds.includes(m.id)).map(m => m.name)
  );

  let visible = $derived(
    (onlyShelf && shelfNames.length > 0
      ? results.filter(x => shelfNames.includes(x.manufacturer))
      : results
    ).slice(0, 8)
  );

  function openDetail(res: SearchResult) {
    const p = paintById(res.paintId);
    if (p) detailPaint = p;
  }

  function mesclarFrom(paint: Paint) {
    detailPaint = null;
    appState.pendingMesclarPaint = paint;
    switchTab('mesclar');
  }

  const channels = [
    { key: 'r', label: 'R', color: 'var(--chan-r)' },
    { key: 'g', label: 'G', color: 'var(--chan-g)' },
    { key: 'b', label: 'B', color: 'var(--chan-b)' },
  ] as const;

  function channelGradient(ch: 'r' | 'g' | 'b'): string {
    const at = (v: number) =>
      `rgb(${ch === 'r' ? v : r}, ${ch === 'g' ? v : g}, ${ch === 'b' ? v : b})`;
    return `linear-gradient(to right, ${at(0)}, ${at(255)})`;
  }
</script>

<div class="cor">
  <!-- Cor chapada, sem nada por cima — a etiqueta fica abaixo, fora da cor. -->
  <div class="cor-preview" style="background: rgb({r}, {g}, {b});">
    <!-- slot reservado: captura por câmera entra na fase APK (Capacitor) -->
  </div>
  <p class="cor-readout font-mono">{currentHex} · {r}, {g}, {b}</p>

  <div class="cor-controls">
    {#each channels as ch (ch.key)}
      <label class="cor-slider">
        <span class="cor-chan font-mono" style="color: {ch.color};">{ch.label}</span>
        <input
          type="range"
          min="0"
          max="255"
          style="--track: {channelGradient(ch.key)};"
          value={ch.key === 'r' ? r : ch.key === 'g' ? g : b}
          oninput={e => {
            const v = +(e.currentTarget as HTMLInputElement).value;
            if (ch.key === 'r') r = v;
            else if (ch.key === 'g') g = v;
            else b = v;
          }}
        />
        <span class="cor-val font-mono">{ch.key === 'r' ? r : ch.key === 'g' ? g : b}</span>
      </label>
    {/each}

    <div class="cor-hex-row">
      <input
        bind:value={hex}
        type="text"
        class="font-mono"
        placeholder="#A67B4F"
        maxlength="7"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        onkeydown={e => e.key === 'Enter' && applyHex()}
      />
      <button class="btn-ghost" onclick={applyHex}>Aplicar HEX</button>
    </div>

    {#if shelfNames.length > 0}
      <label class="cor-only-shelf pressable">
        <input type="checkbox" bind:checked={onlyShelf} />
        <span>Só minhas marcas ({shelfNames.length})</span>
      </label>
    {/if}
  </div>

  <div class="cor-results">
    <h2 class="cor-results-title font-display">
      Tintas mais próximas
      {#if searching}<span class="cor-live">atualizando…</span>{/if}
    </h2>
    {#if visible.length === 0}
      <div class="empty-state">
        <span class="empty-icon"><Icon name="pipette" size={34} /></span>
        <p class="empty-title">Nenhuma tinta próxima</p>
        <p class="empty-hint">{onlyShelf ? 'Tente desligar o filtro "só minhas marcas".' : 'Ajuste os sliders ou o HEX.'}</p>
      </div>
    {:else}
      {#each visible as res (res.paintId)}
        <button class="cor-row pressable" onclick={() => openDetail(res)}>
          <span class="swatch-flat" style="width: 44px; height: 44px; background: rgb({res.r}, {res.g}, {res.b});"></span>
          <span class="cor-row-text">
            <span class="cor-row-name">{res.name}</span>
            <span class="cor-row-meta"><span class="font-mono">{res.code}</span> · {res.manufacturer}</span>
          </span>
          <DeltaBadge
            deltaE={res.deltaE}
            size="sm"
            onclick={() => { deltaSheetValue = res.deltaE; deltaSheetOpen = true; }}
          />
        </button>
      {/each}
    {/if}
  </div>
</div>

<DeltaScaleSheet open={deltaSheetOpen} onClose={() => (deltaSheetOpen = false)} deltaE={deltaSheetValue} />
<PaintDetailSheet paint={detailPaint} onClose={() => (detailPaint = null)} onMesclar={mesclarFrom} />

<style>
  .cor {
    padding: 0 0 16px;
  }

  .cor-preview {
    position: relative;
    height: 30dvh;
    min-height: 160px;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink-100) 12%, transparent);
  }

  /* Etiqueta impressa abaixo da cor — nunca por cima dela. */
  .cor-readout {
    font-size: 12.5px;
    letter-spacing: 0.04em;
    color: var(--ink-500);
    padding: 10px 16px 0;
  }

  .cor-controls {
    padding: 12px 16px 16px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .cor-slider {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .cor-chan {
    width: 18px;
    font-size: 14px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .cor-slider input[type='range'] {
    flex: 1;
    appearance: none;
    -webkit-appearance: none;
    height: 12px;
    border-radius: 999px;
    background: var(--track);
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
    outline: none;
  }

  .cor-slider input[type='range']::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--ink-900);
    border: 2px solid var(--paper);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
    cursor: pointer;
  }

  .cor-slider input[type='range']::-moz-range-thumb {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--ink-900);
    border: 2px solid var(--paper);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
    cursor: pointer;
  }

  .cor-val {
    width: 38px;
    text-align: right;
    font-size: 14px;
    color: var(--ink-300);
    flex-shrink: 0;
  }

  .cor-hex-row {
    display: flex;
    gap: 10px;
  }

  .cor-hex-row input {
    flex: 1;
    min-width: 0;
    min-height: 48px;
    padding: 0 14px;
    border: 1px solid var(--ink-600);
    border-radius: var(--radius-control);
    background: var(--ink-850);
    font-size: 16px;
    color: var(--ink-100);
    outline: none;
  }

  .cor-hex-row input:focus {
    border-color: var(--lacquer);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--lacquer) 20%, transparent);
  }

  .cor-only-shelf {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 44px;
    font-size: 14.5px;
    font-weight: 500;
    color: var(--ink-300);
  }

  .cor-only-shelf input {
    width: 20px;
    height: 20px;
    accent-color: var(--lacquer);
  }

  .cor-results {
    padding: 0 16px;
  }

  .cor-results-title {
    display: flex;
    align-items: baseline;
    gap: 10px;
    font-size: 17px;
    font-weight: 600;
    color: var(--ink-100);
    margin-bottom: 8px;
  }

  .cor-live {
    font-family: var(--font-body);
    font-size: 12px;
    font-weight: 400;
    color: var(--ink-500);
  }

  .cor-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 60px;
    padding: 8px 4px;
    border-bottom: 1px solid var(--ink-800);
    text-align: left;
  }

  .cor-row:active {
    background: var(--ink-800);
  }

  .cor-row-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .cor-row-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cor-row-meta {
    font-size: 12.5px;
    color: var(--ink-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
