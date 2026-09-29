<script lang="ts">
  // rf-24 — faixa proporcional da fórmula: um segmento por tinta, com largura
  // igual ao percentual. 12px de altura, a mesma medida da barra de T1 no
  // protótipo (T1 mantém o markup inline por causa do teste de fidelidade).
  import { t } from '../i18n.svelte';

  interface Props {
    ingredientes: Array<{ r: number; g: number; b: number; code?: string; name?: string; percentage: number }>;
  }

  let { ingredientes }: Props = $props();

  function rotulo(i: Props['ingredientes'][number]): string {
    return `${i.code || i.name || ''} · ${Math.round(i.percentage)}%`;
  }

  let descricao = $derived(t('faixaAria', { list: ingredientes.map(rotulo).join(', ') }));
</script>

<div
  role="img"
  aria-label={descricao}
  style="display: flex; height: 12px; border-radius: 4px; overflow: hidden; border: 1px solid var(--color-rule);"
>
  {#each ingredientes as ing, i (i)}
    <!-- flex-grow proporcional ao percentual; 2px de piso mantém visível a tinta de 3% -->
    <span
      title={rotulo(ing)}
      style="height: 12px; flex: {Math.max(ing.percentage, 0.01)} 1 0; min-width: 2px; background: rgb({ing.r}, {ing.g}, {ing.b});"
    ></span>
  {/each}
</div>
