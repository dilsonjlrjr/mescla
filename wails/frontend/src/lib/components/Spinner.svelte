<script lang="ts">
  // Indicador de carregamento da Mescla: o próprio d20 da marca girando no
  // eixo Y, como um dado rolando na bancada. Não é um spinner genérico — é a
  // logomarca, então o app não precisa de um segundo vocabulário visual.
  //
  // Acessibilidade: `role="status"` com rótulo, e o giro morre sozinho sob
  // `prefers-reduced-motion` (regra global em app.css) — o desenho continua
  // legível parado.
  import BrandMark from './BrandMark.svelte';

  interface Props {
    size?: number;
    /** rótulo lido por leitor de tela; também vira `title` */
    label?: string;
    /** volta do giro em ms — maior = mais calmo */
    duration?: number;
  }

  let { size = 28, label = 'Carregando', duration = 1800 }: Props = $props();
</script>

<span
  class="spin-wrap"
  role="status"
  aria-label={label}
  title={label}
  style="--spin-size: {size}px; --spin-duration: {duration}ms;"
>
  <span class="spin-die">
    <BrandMark {size} />
  </span>
</span>

<style>
  .spin-wrap {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--spin-size);
    height: var(--spin-size);
    flex-shrink: 0;
    /* a perspectiva é o que faz o giro parecer um dado rolando, não um
       adesivo girando no plano */
    perspective: calc(var(--spin-size) * 4);
  }

  .spin-die {
    display: block;
    transform-style: preserve-3d;
    animation: die-roll var(--spin-duration) cubic-bezier(0.62, 0.03, 0.35, 0.98) infinite;
  }

  @keyframes die-roll {
    0% {
      transform: rotateY(0deg) rotateX(0deg);
    }
    50% {
      transform: rotateY(180deg) rotateX(8deg);
    }
    100% {
      transform: rotateY(360deg) rotateX(0deg);
    }
  }
</style>
