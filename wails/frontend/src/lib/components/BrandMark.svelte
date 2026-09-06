<script lang="ts">
  // Logomark da Mescla: d20 (dado de 20 faces) — fonte: docs/Mescla AI.html
  // (protótipo, viewBox 0 0 48 48, 7 facetas). Hex literal, não token: é o
  // desenho de referência do design, não chrome de UI (mesma exceção pontual
  // já aplicada em DeltaScaleSheet).
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
  <polygon points="24,2 43,13 43,35 24,46 5,35 5,13" fill="#75798c" />
  <polygon points="5,13 11,15 24,36 24,46 5,35" fill="#5f6373" />
  <polygon points="43,13 37,15 24,36 24,46 43,35" fill="#9184d9" />
  <polygon points="24,2 5,13 11,15" fill="#6b6f80" />
  <polygon points="24,2 43,13 37,15" fill="#7d6fca" />
  <polygon points="24,2 11,15 37,15" fill="#8b8fa3" />
  <polygon points="11,15 37,15 24,36" fill="#d2cefd" />
</svg>
