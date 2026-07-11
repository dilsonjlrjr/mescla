<script lang="ts">
  // Frasco conta-gotas sintético (SVG) preenchido com a cor real da tinta.
  // Usado como "thumbnail de frasco" em todo o app — cobre o catálogo inteiro
  // sem depender de foto de produto por SKU.
  interface Props {
    r: number;
    g: number;
    b: number;
    size?: number;      // altura em px (largura = 60% disso)
    label?: string;     // texto curto no rótulo (ex: código)
  }

  let { r, g, b, size = 64, label = '' }: Props = $props();

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
  <path d="M27 2 L33 2 L36 15 L24 15 Z" fill="#3a3a40" />
  <!-- anel da tampa -->
  <rect x="21" y="14" width="18" height="9" rx="2.5" fill="#2c2c32" />
  <!-- corpo do frasco (a tinta) -->
  <rect x="13" y="23" width="34" height="73" rx="8" fill={fill} />
  <!-- contorno sutil pro corpo não sumir em cores escuras -->
  <rect x="13" y="23" width="34" height="73" rx="8" fill="none" stroke="rgba(0,0,0,0.28)" stroke-width="1" />
  <!-- reflexo -->
  <rect x="18" y="28" width="5" height="62" rx="2.5" fill="rgba(255,255,255,0.30)" />
  <!-- rótulo -->
  <rect x="13" y="57" width="34" height="28" fill="#f3f0e8" />
  <rect x="13" y="57" width="34" height="28" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.75" />
  <!-- faixa da cor no rótulo -->
  <rect x="17" y="61" width="26" height="9" rx="1.5" fill={fill} stroke="rgba(0,0,0,0.2)" stroke-width="0.5" />
  {#if label}
    <text
      x="30" y="79"
      text-anchor="middle"
      font-family="ui-monospace, monospace"
      font-size="7.5"
      fill="#4a4640"
    >{label.length > 8 ? label.slice(0, 8) : label}</text>
  {:else}
    <!-- linhas de "texto" decorativas -->
    <rect x="18" y="74" width="24" height="2.5" rx="1.25" fill="#c9c4b8" />
    <rect x="21" y="79" width="18" height="2.5" rx="1.25" fill="#d6d2c6" />
  {/if}
</svg>
