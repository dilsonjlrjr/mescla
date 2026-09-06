<script lang="ts">
  // Logomark da Mescla: d20 de 7 facetas — fonte: docs/oficial/Mescla AI.html
  // (viewBox 0 0 48 48). As faces saem dos tokens --brand-1..7 (NFR-10).
  interface Props {
    size?: number;
  }

  let { size = 30 }: Props = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 48 48"
  xmlns="http://www.w3.org/2000/svg"
  role="img"
  aria-label="Mescla"
  style="display: block; flex-shrink: 0;"
>
  <polygon points="24,2 43,13 43,35 24,46 5,35 5,13" fill="var(--brand-1)" />
  <polygon points="5,13 11,15 24,36 24,46 5,35" fill="var(--brand-2)" />
  <polygon points="43,13 37,15 24,36 24,46 43,35" fill="var(--brand-3)" />
  <polygon points="24,2 5,13 11,15" fill="var(--brand-4)" />
  <polygon points="24,2 43,13 37,15" fill="var(--brand-5)" />
  <polygon points="24,2 11,15 37,15" fill="var(--brand-6)" />
  <polygon points="11,15 37,15 24,36" fill="var(--brand-7)" />
</svg>
