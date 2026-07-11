<script lang="ts">
  // Card de catálogo (Tintômetro): sem borda nem caixa. Amostra chapada
  // raio 8 com índice No. mono no canto + garrafinha ao lado; abaixo,
  // nome bold e código · fabricante em mono.
  import PaintBottle from './PaintBottle.svelte';
  import { contrastOn } from '../ui';

  interface Paint {
    id: number;
    name: string;
    code: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  interface Props {
    paint: Paint;
    index: number; // posição na ordenação atual (No. do arquivo)
    onclick: () => void;
  }

  let { paint, index, onclick }: Props = $props();

  let no = $derived(String(index).padStart(2, '0'));
</script>

<button class="card" {onclick}>
  <span class="card-visual">
    <span class="card-swatch" style="background: rgb({paint.r}, {paint.g}, {paint.b});">
      <span class="card-no font-mono" style="color: {contrastOn(paint.r, paint.g, paint.b)};">No. {no}</span>
    </span>
    <span class="card-bottle">
      <PaintBottle r={paint.r} g={paint.g} b={paint.b} size={78} />
    </span>
  </span>
  <span class="card-name">{paint.name}</span>
  <span class="card-meta font-mono">{paint.code ? `${paint.code} · ` : ''}{paint.manufacturer}</span>
</button>

<style>
  .card {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 3px;
    width: 100%;
    padding: 0;
    border: none;
    background: none;
    cursor: pointer;
    text-align: left;
    min-width: 0;
  }

  .card-visual {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    margin-bottom: 8px;
  }

  .card-swatch {
    position: relative;
    flex: 1;
    min-width: 0;
    aspect-ratio: 4 / 3;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.06);
    transition: transform 0.15s ease;
  }

  .card:hover .card-swatch {
    transform: translateY(-2px);
  }

  .card-no {
    position: absolute;
    top: 10px;
    left: 12px;
    font-size: 10.5px;
    font-weight: 500;
    opacity: 0.9;
  }

  .card-bottle {
    flex-shrink: 0;
    display: flex;
  }

  .card-name {
    font-size: 13.5px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .card:hover .card-name {
    color: var(--laca-deep);
  }

  .card-meta {
    font-size: 11px;
    color: var(--text-2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
