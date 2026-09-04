<script lang="ts">
  // Logomark da Mescla (tema Tintômetro): a gota de tinta em duas metades —
  // grafite (a tinta que você tem) e laca (a cor que você busca).
  interface Props {
    size?: number;
  }

  let { size = 30 }: Props = $props();

  const uid = `bm-${Math.random().toString(36).slice(2, 8)}`;
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
  <defs>
    <clipPath id={uid}>
      <path d="M16 2 C16 2 6 14.5 6 21 a10 10 0 0 0 20 0 C26 14.5 16 2 16 2 Z" />
    </clipPath>
  </defs>
  <g clip-path="url(#{uid})">
    <rect x="0" y="0" width="16" height="32" fill="#1a1712" />
    <rect x="16" y="0" width="16" height="32" fill="#e8542c" />
  </g>
</svg>
