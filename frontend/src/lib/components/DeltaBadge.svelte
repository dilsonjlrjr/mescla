<script lang="ts">
  // ΔE2000 traduzido pra gente: número + rótulo humano + par de cores.
  // Faixas baseadas na percepção: <1 imperceptível, <3 só lado a lado,
  // <6 perceptível, <12 visível, acima disso é outra cor.
  interface Props {
    deltaE: number;
    // par opcional: cor alvo vs cor obtida — o motivo da marca
    pair?: { r1: number; g1: number; b1: number; r2: number; g2: number; b2: number };
    size?: 'sm' | 'md';
  }

  let { deltaE, pair, size = 'md' }: Props = $props();

  let grade = $derived(
    deltaE < 1 ? { cls: 'excellent', label: 'Idêntica' } :
    deltaE < 3 ? { cls: 'excellent', label: 'Muito próxima' } :
    deltaE < 6 ? { cls: 'good', label: 'Próxima' } :
    deltaE < 12 ? { cls: 'fair', label: 'Diferença visível' } :
    { cls: 'poor', label: 'Cor diferente' }
  );

  let explain = $derived(
    `ΔE2000 = ${deltaE.toFixed(2)}. ` +
    (deltaE < 1 ? 'Olho humano não distingue.' :
     deltaE < 3 ? 'Diferença só aparece com as cores encostadas.' :
     deltaE < 6 ? 'Diferença pequena, aceitável na maioria dos usos.' :
     deltaE < 12 ? 'Diferença clara a olho nu.' :
     'A mistura não reproduz esta cor.')
  );
</script>

<span class="delta-badge {grade.cls}" title={explain} style={size === 'sm' ? 'font-size: 11px; padding: 3px 8px;' : ''}>
  {#if pair}
    <span class="color-pair" style="width: {size === 'sm' ? 18 : 22}px; height: {size === 'sm' ? 12 : 14}px;">
      <span style="background: rgb({pair.r1}, {pair.g1}, {pair.b1});"></span>
      <span style="background: rgb({pair.r2}, {pair.g2}, {pair.b2});"></span>
    </span>
  {/if}
  {grade.label}
  <span class="delta-value">ΔE {deltaE.toFixed(1)}</span>
</span>
