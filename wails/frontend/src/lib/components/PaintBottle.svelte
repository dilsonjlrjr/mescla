<script lang="ts">
  // Garrafinha (tema Tintômetro): silhueta de frasco conta-gotas com tampa
  // grafite, corpo na cor da tinta com contorno grafite e etiqueta branca
  // quadrada com o ponto da cor no meio. Cobre o catálogo inteiro sem foto.
  interface Props {
    r: number;
    g: number;
    b: number;
    size?: number; // altura em px (largura = 50% disso)
  }

  let { r, g, b, size = 64 }: Props = $props();

  let fill = $derived(`rgb(${r}, ${g}, ${b})`);
</script>

<svg
  width={size * 0.5}
  height={size}
  viewBox="0 0 50 100"
  xmlns="http://www.w3.org/2000/svg"
  role="img"
  aria-label="Frasco de tinta"
  style="display: block; flex-shrink: 0;"
>
  <!-- bico conta-gotas -->
  <path d="M22 2 L28 2 L31 14 L19 14 Z" fill="#1a1712" />
  <!-- tampa -->
  <rect x="15" y="13" width="20" height="11" rx="3" fill="#1a1712" />
  <!-- corpo do frasco (a tinta), contorno grafite -->
  <rect x="7" y="26" width="36" height="70" rx="9" fill={fill} />
  <rect x="7" y="26" width="36" height="70" rx="9" fill="none" stroke="#1a1712" stroke-width="2.5" />
  <!-- etiqueta branca quadrada com o ponto da cor -->
  <rect x="15" y="52" width="20" height="20" rx="2.5" fill="#ffffff" />
  <rect x="15" y="52" width="20" height="20" rx="2.5" fill="none" stroke="rgba(26,23,18,0.18)" stroke-width="1" />
  <rect x="21" y="58" width="8" height="8" rx="2" fill={fill} stroke="rgba(26,23,18,0.25)" stroke-width="0.75" />
</svg>
