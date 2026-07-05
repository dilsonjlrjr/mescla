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
    <div class="pd-hero" style="background: rgb({paint.r}, {paint.g}, {paint.b});">
      <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={92} label={paint.code} />
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
    display: flex;
    align-items: center;
    justify-content: center;
    height: 130px;
    border-radius: 12px;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ink-100) 15%, transparent);
    margin-bottom: 16px;
  }

  .pd-name {
    font-size: 22px;
    font-weight: 600;
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
    border-radius: 10px;
    background: var(--ink-800);
    color: var(--ink-300);
    font-size: 13px;
    font-weight: 500;
  }
</style>
