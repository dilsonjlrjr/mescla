<script lang="ts">
  // Logomark da Mescla: duas metades de cor que se encontram — a tinta de
  // origem e a de destino. O motivo "par de cores" se repete no app inteiro.
  interface Props {
    size?: number;
  }

  let { size = 30 }: Props = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 32 32"
  xmlns="http://www.w3.org/2000/svg"
  role="img"
  aria-label="Mescla"
  style="display: block; flex-shrink: 0;"
>
  <!-- metade origem (laca) -->
  <path d="M16 2 A14 14 0 0 0 16 30 Z" fill="var(--lacquer)" />
  <!-- metade destino (grafite de rótulo) -->
  <path d="M16 2 A14 14 0 0 1 16 30 Z" fill="var(--paper)" />
  <!-- costura central: o ponto onde as cores se mesclam -->
  <rect x="14.6" y="7" width="2.8" height="18" rx="1.4" fill="var(--ink-950)" />
</svg>
