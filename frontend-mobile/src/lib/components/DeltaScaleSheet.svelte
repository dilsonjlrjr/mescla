<script lang="ts">
  // Sheet da escala ΔE00 — leitura de instrumento: faixa mono bold, descrição
  // e um par de amostras ilustrativas (cor base + cor deslocada) por linha.
  import BottomSheet from './BottomSheet.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    /** ΔE atual — destaca a faixa onde o resultado caiu. */
    deltaE?: number | null;
  }

  let { open, onClose, deltaE = null }: Props = $props();

  // Pares ilustrativos: o mesmo vermelho de bancada com deslocamentos crescentes.
  const bands = [
    { range: '0 - 1', desc: 'Indistinguível a olho nu', a: '#8a1518', b: '#8b161a', max: 1 },
    { range: '1 - 2', desc: 'Excelente: some na mini', a: '#8a1518', b: '#8f181c', max: 2 },
    { range: '2 - 4', desc: 'Boa: passa sob luz de bancada', a: '#8a1518', b: '#96201f', max: 4 },
    { range: '4 - 8', desc: 'Perceptível lado a lado', a: '#8a1518', b: '#a53527', max: 8 },
    { range: '8 +', desc: 'Outra cor, na prática', a: '#8a1518', b: '#c05c2e', max: Infinity },
  ];

  let activeIdx = $derived(deltaE == null ? -1 : bands.findIndex(x => deltaE! < x.max));
</script>

<BottomSheet {open} {onClose} title="O que significa o ΔE?">
  <p class="scale-intro">Distância entre duas cores, como o olho vê.</p>

  <div class="scale-list">
    {#each bands as band, i}
      <div class="scale-row" class:active={activeIdx === i}>
        <span class="scale-range font-mono">{band.range}</span>
        <span class="scale-desc">{band.desc}</span>
        {#if activeIdx === i && deltaE != null}
          <span class="scale-you font-mono">ΔE {deltaE.toFixed(1)}</span>
        {/if}
        <span class="scale-pair" aria-hidden="true">
          <span style="background: {band.a};"></span>
          <span style="background: {band.b};"></span>
        </span>
      </div>
    {/each}
  </div>

  <p class="scale-note">O Mescla só sugere fórmula quando ΔE fica abaixo de 8.</p>
  <button class="btn-primary" style="margin-top: 14px;" onclick={onClose}>Entendi</button>
</BottomSheet>

<style>
  .scale-intro {
    font-size: 15px;
    color: var(--ink-500);
    line-height: 1.55;
    margin-bottom: 6px;
  }

  .scale-list {
    display: flex;
    flex-direction: column;
  }

  .scale-row {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 60px;
    padding: 10px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .scale-row.active .scale-range,
  .scale-row.active .scale-desc {
    color: var(--laca);
  }

  .scale-range {
    font-size: 15px;
    font-weight: 700;
    min-width: 52px;
    color: var(--grafite);
    white-space: nowrap;
  }

  .scale-desc {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    color: var(--grafite);
  }

  .scale-you {
    font-size: 12px;
    font-weight: 700;
    color: var(--laca);
    flex-shrink: 0;
  }

  .scale-pair {
    display: inline-flex;
    gap: 3px;
    flex-shrink: 0;
  }

  .scale-pair span {
    width: 34px;
    height: 28px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px rgba(26, 23, 18, 0.1);
  }

  .scale-note {
    margin-top: 14px;
    font-size: 13px;
    color: var(--ink-500);
  }
</style>
