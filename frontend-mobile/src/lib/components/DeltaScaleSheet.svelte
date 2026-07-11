<script lang="ts">
  // Sheet da escala ΔE — no mobile a explicação é tocável (tap no badge),
  // não tooltip. Mesmos cortes do DeltaBadge.
  import BottomSheet from './BottomSheet.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    /** ΔE atual — destaca a faixa onde o resultado caiu. */
    deltaE?: number | null;
  }

  let { open, onClose, deltaE = null }: Props = $props();

  const bands = [
    { range: '0–1', label: 'Idêntica', desc: 'Olho humano não distingue.', cls: 'excellent', max: 1 },
    { range: '1–3', label: 'Muito próxima', desc: 'Diferença só aparece com as cores encostadas.', cls: 'excellent', max: 3 },
    { range: '3–6', label: 'Próxima', desc: 'Diferença pequena, aceitável na maioria dos usos.', cls: 'good', max: 6 },
    { range: '6–12', label: 'Diferença visível', desc: 'Diferença clara a olho nu.', cls: 'fair', max: 12 },
    { range: '12+', label: 'Cor diferente', desc: 'A mistura não reproduz esta cor.', cls: 'poor', max: Infinity },
  ];

  let activeIdx = $derived(deltaE == null ? -1 : bands.findIndex(b => deltaE! < b.max));
</script>

<BottomSheet {open} {onClose} title="O que é o ΔE?">
  <p class="scale-intro">
    O <strong>ΔE</strong> mede a diferença entre duas cores como o olho humano percebe —
    quanto menor, mais parecidas. <strong>0 = idênticas.</strong>
  </p>
  <div class="scale-list">
    {#each bands as band, i}
      <div class="scale-row {band.cls}" class:active={activeIdx === i}>
        <span class="scale-range font-mono">{band.range}</span>
        <span class="scale-text">
          <span class="scale-label">{band.label}</span>
          <span class="scale-desc">{band.desc}</span>
        </span>
        {#if activeIdx === i && deltaE != null}
          <span class="scale-you font-mono">ΔE {deltaE.toFixed(1)}</span>
        {/if}
      </div>
    {/each}
  </div>
  <p class="scale-note">Cores de tela são aproximadas — confie no ΔE.</p>
</BottomSheet>

<style>
  .scale-intro {
    font-size: 15px;
    color: var(--ink-300);
    line-height: 1.55;
    margin-bottom: 16px;
  }

  .scale-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .scale-row {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 14px;
    border-radius: var(--radius-surface);
    background: var(--ink-900);
    border: 1px solid var(--ink-700);
    opacity: 0.75;
  }

  .scale-row.active {
    opacity: 1;
    background: color-mix(in srgb, currentColor 10%, var(--ink-900));
    border-color: color-mix(in srgb, currentColor 40%, transparent);
  }

  .scale-row.excellent { color: var(--delta-excellent); }
  .scale-row.good { color: var(--delta-good); }
  .scale-row.fair { color: var(--delta-fair); }
  .scale-row.poor { color: var(--delta-poor); }

  .scale-range {
    font-size: 14px;
    font-weight: 600;
    min-width: 46px;
  }

  .scale-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    min-width: 0;
  }

  .scale-label {
    font-size: 15px;
    font-weight: 600;
  }

  .scale-desc {
    font-size: 13px;
    color: var(--ink-500);
  }

  .scale-you {
    font-size: 13px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .scale-note {
    margin-top: 16px;
    font-size: 13px;
    color: var(--ink-500);
    text-align: center;
  }
</style>
