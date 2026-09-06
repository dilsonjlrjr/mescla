<script lang="ts">
  // Pote de tinta — desenho do protótipo docs/oficial/Mescla AI.html
  // (viewBox 0 0 60 100: tampa, gargalo, corpo na cor da tinta e rótulo branco
  // com a amostra dentro). Aparece na sugestão de busca (34×57), no "você
  // perguntou" (30×50), no pote pronto (72×120), no ingrediente da fórmula
  // (44×74) e no rodapé "Na mesa hoje" (28×47).
  interface Props {
    r: number;
    g: number;
    b: number;
    /** largura em px; a altura sai da proporção 3:5 do protótipo se omitida */
    width?: number;
    height?: number;
    label?: string;
  }

  let { r, g, b, width = 44, height, label }: Props = $props();

  let h = $derived(height ?? Math.round((width * 100) / 60));
  let fill = $derived(`rgb(${r}, ${g}, ${b})`);
</script>

<svg
  viewBox="0 0 60 100"
  width={width}
  height={h}
  role="img"
  aria-label={label ?? 'Pote de tinta'}
  style="flex-shrink: 0; display: block;"
>
  <path d="M25 4 L35 4 L38 18 L22 18 Z" fill="var(--color-neutral-700)" />
  <rect x="19" y="17" width="22" height="9" rx="3" fill="var(--color-neutral-600)" />
  <rect
    x="11"
    y="26"
    width="38"
    height="70"
    rx="10"
    fill={fill}
    stroke="var(--color-bottle-stroke)"
    stroke-width="2"
  />
  <rect x="19" y="56" width="22" height="20" rx="2" fill="var(--color-text)" />
  <rect x="24" y="61" width="12" height="10" rx="1.5" fill={fill} />
</svg>
