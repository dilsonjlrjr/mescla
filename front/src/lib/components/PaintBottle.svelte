<script lang="ts">
  // Garrafinha do Tintômetro (SVG): silhueta de frasco conta-gotas com tampa,
  // contorno grafite, corpo na cor da tinta e etiqueta branca quadrada com um
  // ponto da cor no meio — o traço dos boards do Penpot. Cobre o catálogo
  // inteiro sem depender de foto de produto por SKU.
  interface Props {
    r: number;
    g: number;
    b: number;
    size?: number; // altura em px (largura = 60% disso)
    /** aceito por compatibilidade; o traço atual usa só o ponto de cor */
    label?: string;
  }

  let { r, g, b, size = 64 }: Props = $props();

  let fill = $derived(`rgb(${r}, ${g}, ${b})`);
</script>

<svg
  width={size * 0.6}
  height={size}
  viewBox="0 0 60 100"
  xmlns="http://www.w3.org/2000/svg"
  role="img"
  aria-label="Frasco de tinta"
  style="display: block; flex-shrink: 0;"
>
  <!-- bico conta-gotas -->
  <path d="M26 4 L34 4 L37 18 L23 18 Z" fill="#1a1712" />
  <!-- anel da tampa -->
  <rect x="20" y="17" width="20" height="8" rx="2.5" fill="#1a1712" />
  <!-- corpo do frasco (a tinta), contorno grafite -->
  <rect x="12" y="25" width="36" height="71" rx="9" fill={fill} stroke="#1a1712" stroke-width="3" />
  <!-- etiqueta branca quadrada com o ponto da cor -->
  <rect x="21" y="52" width="18" height="18" rx="2" fill="#ffffff" stroke="#1a1712" stroke-width="1.5" />
  <rect x="26" y="57" width="8" height="8" rx="1.5" fill={fill} stroke="rgba(26,23,18,0.25)" stroke-width="0.75" />
</svg>
