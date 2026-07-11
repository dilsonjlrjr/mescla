<script lang="ts">
  // Detalhe de tinta em bottom sheet (não página): amostra grande + garrafinha,
  // grid mono HEX / RGB / HSL, parecidas no catálogo e os corredores do fluxo:
  // "Gerar fórmula equivalente", "Comparar" e "Tenho outra parecida" (estoque).
  import BottomSheet from './BottomSheet.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { hexOf, type Paint } from '../services/catalog';
  import { findSimilar, type SearchResult } from '../services/engine';
  import { rgbToHsl } from '../color/theory';
  import { stock } from '../services/stock.svelte';
  import { toast } from '../toast.svelte';
  import { addToCompare, appState } from '../appState.svelte';
  import { switchTab } from '../nav.svelte';

  interface Props {
    paint: Paint | null;
    onClose: () => void;
    onMesclar: (paint: Paint) => void;
  }

  let { paint, onClose, onMesclar }: Props = $props();

  let hsl = $derived(paint ? rgbToHsl({ r: paint.r, g: paint.g, b: paint.b }) : null);

  // Selo "no meu estoque": match por fabricante + código (a tinta de estoque é livre).
  let inStock = $derived(
    paint !== null &&
      stock.paints.some(sp => sp.manufacturer === paint!.manufacturer && sp.code !== '' && sp.code === paint!.code)
  );

  // Parecidas no catálogo (exclui a própria tinta).
  let similar: SearchResult[] = $state([]);
  $effect(() => {
    const p = paint;
    similar = [];
    if (!p) return;
    findSimilar(p.r, p.g, p.b, 100, 8)
      .then(res => {
        if (paint?.id === p.id) similar = res.filter(x => x.paintId !== p.id).slice(0, 6);
      })
      .catch(() => (similar = []));
  });

  function copyColor() {
    if (!paint) return;
    const hex = hexOf(paint);
    navigator.clipboard.writeText(hex);
    toast(`${hex} copiado`);
  }

  function copyCode() {
    if (!paint) return;
    navigator.clipboard.writeText(paint.code);
    toast(`Código ${paint.code} copiado`);
  }

  function compare() {
    if (!paint) return;
    if (appState.compareIds.includes(paint.id)) {
      toast('Já está na comparação');
      return;
    }
    if (addToCompare(paint.id)) {
      toast(`Adicionada à comparação (${appState.compareIds.length}/6)`);
      onClose();
    } else {
      toast('Máximo de 6 tintas na comparação', 'error');
    }
  }

  // "Tenho outra parecida": abre o cadastro do estoque com a cor pré-preenchida.
  function haveSimilar() {
    if (!paint) return;
    appState.pendingStockPrefill = { manufacturerId: paint.manufacturerId, hex: hexOf(paint) };
    onClose();
    switchTab('mais');
  }
</script>

<BottomSheet open={paint !== null} {onClose}>
  {#if paint}
    <div class="pd-top">
      <span class="pd-swatch" style="background: rgb({paint.r}, {paint.g}, {paint.b});"></span>
      <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={84} />
      <div class="pd-id">
        <h2 class="pd-name font-display">{paint.name}</h2>
        <p class="pd-mfr">{paint.manufacturer}{paint.line ? ` · ${paint.line}` : ''}</p>
        <button class="pd-code font-mono pressable" onclick={copyCode}>{paint.code}</button>
        {#if inStock}
          <span class="pd-stock font-mono">no meu estoque</span>
        {/if}
      </div>
    </div>

    <div class="pd-hairline"></div>

    <button class="pd-values pressable" onclick={copyColor} aria-label="Copiar cor">
      <span class="pd-val">
        <span class="pd-val-label font-mono">HEX</span>
        <span class="pd-val-num font-mono">{hexOf(paint)}</span>
      </span>
      <span class="pd-val">
        <span class="pd-val-label font-mono">RGB</span>
        <span class="pd-val-num font-mono">{paint.r} {paint.g} {paint.b}</span>
      </span>
      <span class="pd-val">
        <span class="pd-val-label font-mono">HSL</span>
        <span class="pd-val-num font-mono">{Math.round(hsl!.h)}° {Math.round(hsl!.s * 100)}% {Math.round(hsl!.l * 100)}%</span>
      </span>
    </button>

    {#if similar.length > 0}
      <p class="section-label" style="margin-top: 14px;">Parecidas no catálogo</p>
      <div class="pd-similar">
        {#each similar as s (s.paintId)}
          <div class="pd-sim">
            <span class="pd-sim-top">
              <span class="pd-sim-swatch" style="background: rgb({s.r}, {s.g}, {s.b});"></span>
              <PaintBottle r={s.r} g={s.g} b={s.b} size={40} />
            </span>
            <span class="pd-sim-delta font-mono">ΔE {s.deltaE.toFixed(1)}</span>
          </div>
        {/each}
      </div>
    {/if}

    <button class="btn-primary" style="margin-top: 18px;" onclick={() => onMesclar(paint!)}>
      Gerar fórmula equivalente
    </button>
    <div class="pd-actions">
      <button class="btn-ghost" style="flex: 1;" onclick={compare}>Comparar</button>
      <button class="btn-ghost" style="flex: 1;" onclick={haveSimilar}>Tenho outra parecida</button>
    </div>
  {/if}
</BottomSheet>

<style>
  .pd-top {
    display: flex;
    align-items: flex-start;
    gap: 14px;
  }

  .pd-swatch {
    width: 108px;
    height: 96px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
    flex-shrink: 0;
  }

  .pd-id {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .pd-name {
    font-size: 21px;
    font-weight: 750;
    color: var(--grafite);
    line-height: 1.15;
  }

  .pd-mfr {
    font-size: 13px;
    color: var(--ink-500);
  }

  .pd-code {
    font-size: 12.5px;
    color: var(--ink-500);
    padding: 2px 0;
  }

  .pd-stock {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--laca);
  }

  .pd-hairline {
    height: 1px;
    background: var(--hairline);
    margin: 14px 0;
  }

  .pd-values {
    display: grid;
    grid-template-columns: 1fr 1fr 1.2fr;
    gap: 10px;
    width: 100%;
    text-align: left;
  }

  .pd-val {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .pd-val-label {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--ink-500);
  }

  .pd-val-num {
    font-size: 13.5px;
    font-weight: 600;
    color: var(--grafite);
    white-space: nowrap;
  }

  .pd-similar {
    display: flex;
    gap: 10px;
    overflow-x: auto;
    margin-top: 10px;
    padding-bottom: 4px;
    scrollbar-width: none;
  }

  .pd-similar::-webkit-scrollbar {
    display: none;
  }

  .pd-sim {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex-shrink: 0;
  }

  .pd-sim-top {
    display: flex;
    align-items: flex-end;
    gap: 4px;
  }

  .pd-sim-swatch {
    width: 76px;
    height: 52px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
  }

  .pd-sim-delta {
    font-size: 11px;
    color: var(--ink-500);
  }

  .pd-actions {
    display: flex;
    gap: 10px;
    margin-top: 10px;
  }
</style>
