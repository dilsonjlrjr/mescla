<script lang="ts">
  // Amostra de cor — fiel ao protótipo docs/Mescla AI.html: NENHUMA tela ali
  // desenha frasco/garrafinha. Todo swatch de tinta é um retângulo achatado
  // com borda fina neutra (ex.: linha de fabricante, sugestão de busca,
  // ingrediente da fórmula, painel alvo/resultado) — nunca contorno preto
  // (aquele traço era do app anterior, "Tintômetro", claro/laca; incompatível
  // com fundo escuro Nocturne). Mantém o nome do componente e a prop `size`
  // pra não mexer nos ~14 call sites que já a usam como altura=largura.
  interface Props {
    r: number;
    g: number;
    b: number;
    size?: number;
  }

  let { r, g, b, size = 44 }: Props = $props();

  let fill = $derived(`rgb(${r}, ${g}, ${b})`);
  // Protótipo usa radius 4 nos swatches de lista/ingrediente e radius 8 nos
  // painéis grandes de comparação (alvo/resultado, ~74-86px) — mesmo corte.
  let radius = $derived(size >= 72 ? 8 : 4);
</script>

<span
  role="img"
  aria-label="Amostra de cor"
  style="display: block; flex-shrink: 0; width: {size}px; height: {size}px; border-radius: {radius}px; border: 1px solid var(--color-neutral-800); background: {fill};"
></span>
