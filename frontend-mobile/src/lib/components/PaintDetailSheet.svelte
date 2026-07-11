<script lang="ts">
  // Detalhe de tinta em bottom sheet (não página): swatch grande, código
  // copiável e o corredor pro fluxo principal — "Mesclar esta cor".
  import BottomSheet from './BottomSheet.svelte';
  import Icon from './Icon.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import { hexOf, type Paint } from '../services/catalog';
  import { toast } from '../toast.svelte';
  import { addToCompare, appState } from '../appState.svelte';

  interface Props {
    paint: Paint | null;
    onClose: () => void;
    onMesclar: (paint: Paint) => void;
  }

  let { paint, onClose, onMesclar }: Props = $props();

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
</script>

<BottomSheet open={paint !== null} {onClose}>
  {#if paint}
    <!-- A cor chapada com a garrafinha em pé na frente, apoiada na base. -->
    <div class="pd-hero" style="background: rgb({paint.r}, {paint.g}, {paint.b});">
      <span class="pd-bottle">
        <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={96} label={paint.code} />
      </span>
    </div>

    <h2 class="pd-name font-display">{paint.name}</h2>
    <p class="pd-mfr">{paint.manufacturer}{paint.line ? ` · ${paint.line}` : ''}</p>

    <div class="pd-chips">
      <button class="pd-chip pressable font-mono" onclick={copyCode}>
        {paint.code}
        <Icon name="copy" size={13} />
      </button>
      <button class="pd-chip pressable font-mono" onclick={copyColor}>
        {hexOf(paint)} · {paint.r}, {paint.g}, {paint.b}
        <Icon name="copy" size={13} />
      </button>
    </div>

    <button class="btn-primary" style="margin-top: 18px;" onclick={() => onMesclar(paint!)}>
      <Icon name="droplet" size={19} />
      Mesclar esta cor
    </button>
    <button class="btn-ghost" style="width: 100%; margin-top: 10px;" onclick={compare}>
      <Icon name="swap" size={17} />
      Adicionar à comparação
    </button>
  {/if}
</BottomSheet>

<style>
  .pd-hero {
    position: relative;
    height: 130px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink-100) 15%, transparent);
    margin-bottom: 16px;
  }

  .pd-bottle {
    position: absolute;
    right: 20px;
    bottom: 0;
    display: flex;
    filter: drop-shadow(0 4px 10px rgba(0, 0, 0, 0.3));
  }

  /* Furo de catálogo no canto — a assinatura da cartela (à esquerda,
     a garrafinha mora à direita). */
  .pd-hero::after {
    content: '';
    position: absolute;
    top: 8px;
    left: 8px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--ink-950);
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.35);
  }

  .pd-name {
    font-size: 22px;
    font-weight: 660;
    color: var(--ink-100);
  }

  .pd-mfr {
    font-size: 14px;
    color: var(--ink-500);
    margin-top: 2px;
    margin-bottom: 14px;
  }

  .pd-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .pd-chip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    padding: 0 14px;
    border-radius: var(--radius-pill);
    background: var(--ink-800);
    color: var(--ink-300);
    font-size: 13px;
    font-weight: 500;
  }
</style>
