<script lang="ts">
  // Adaptado de frontend/src/lib/components/DeltaBadge.svelte (desktop).
  // No mobile a explicação não vive em tooltip (não existe hover): o badge é
  // tocável e o dono abre o sheet da escala ΔE via onclick.
  interface Props {
    deltaE: number;
    pair?: { r1: number; g1: number; b1: number; r2: number; g2: number; b2: number };
    size?: 'sm' | 'md' | 'lg';
    onclick?: () => void;
  }

  let { deltaE, pair, size = 'md', onclick }: Props = $props();

  let grade = $derived(
    deltaE < 1 ? { cls: 'excellent', label: 'Idêntica' } :
    deltaE < 3 ? { cls: 'excellent', label: 'Muito próxima' } :
    deltaE < 6 ? { cls: 'good', label: 'Próxima' } :
    deltaE < 12 ? { cls: 'fair', label: 'Diferença visível' } :
    { cls: 'poor', label: 'Cor diferente' }
  );
</script>

<svelte:element
  this={onclick ? 'button' : 'span'}
  role={onclick ? 'button' : undefined}
  class="delta-badge {grade.cls} pressable"
  class:tappable={!!onclick}
  style={size === 'sm' ? 'font-size: 12px; padding: 4px 9px;' : size === 'lg' ? 'font-size: 15px;' : ''}
  {onclick}
>
  {#if pair}
    <span class="color-pair" style="width: {size === 'sm' ? 18 : 24}px; height: {size === 'sm' ? 12 : 15}px;">
      <span style="background: rgb({pair.r1}, {pair.g1}, {pair.b1});"></span>
      <span style="background: rgb({pair.r2}, {pair.g2}, {pair.b2});"></span>
    </span>
  {/if}
  {grade.label}
  <span class="delta-value">ΔE {deltaE.toFixed(1)}</span>
  {#if onclick}
    <!-- affordance de botão: rótulo explícito, não só um iconezinho -->
    <span class="delta-cta">
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" aria-hidden="true">
        <circle cx="12" cy="12" r="8.5" />
        <path d="M12 11v5.3" />
        <circle cx="12" cy="7.9" r="1" fill="currentColor" stroke="none" />
      </svg>
      {#if size !== 'sm'}o que é?{/if}
    </span>
  {/if}
</svelte:element>
