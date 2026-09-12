<script lang="ts">
  // Cabeçalho fixo das telas secundárias (T2 Plano, T3 Receitas, T4 Minhas
  // tintas) — métrica do protótipo docs/oficial/Mescla AI.html: 76px de altura
  // mínima, padding 10px 20px, volta pra T1 à esquerda, marca + kicker/título,
  // ações da tela no fim e o seletor de idioma. T1 tem cabeçalho próprio (o
  // campo de busca ocupa o lugar do título), por isso não usa este componente.
  import type { Snippet } from 'svelte';
  import BrandMark from './BrandMark.svelte';
  import LangSwitch from './LangSwitch.svelte';
  import { switchTab } from '../nav.svelte';
  import { t } from '../i18n.svelte';

  interface Props {
    /** linha pequena em caixa alta acima do título */
    kicker?: string;
    title?: string;
    /** conteúdo colado à marca, antes do espaçador (T4 põe as abas aqui) */
    lead?: Snippet;
    /** ações à direita, antes do seletor de idioma */
    actions?: Snippet;
    /** T4 usa 14px entre os itens; T2/T3, 16px */
    gap?: number;
    /** rf-16: botão de voltar próprio (o editor de T2 volta para a lista de
     *  projetos). Ausente, volta para T1 com o rótulo "Pergunta". */
    back?: { label: string; onclick: () => void };
    /** rf-16: substitui o título em texto (o editor de T2 põe o campo de
     *  renomear aqui). */
    titleSlot?: Snippet;
  }

  let { kicker, title, lead, actions, gap = 16, back, titleSlot }: Props = $props();
</script>

<div
  style="display: flex; align-items: center; flex-wrap: wrap; row-gap: 8px; gap: {gap}px; min-height: 76px; flex-shrink: 0; padding: 10px 20px; border-bottom: 1px solid var(--color-line);"
>
  <button
    onclick={() => (back ? back.onclick() : switchTab('pergunta'))}
    style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 16px 0 12px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
  >
    <i class="ph ph-arrow-left" style="font-size: 19px;"></i>{back ? back.label : t('question')}
  </button>

  <!-- A logo também volta pra T1: nas telas secundárias "página principal" é a
       própria T1, sem mexer no estado dela. -->
  <button
    class="pressable"
    onclick={() => switchTab('pergunta')}
    aria-label={t('homeAria')}
    style="display: inline-flex; align-items: center; min-height: 44px; padding: 0 4px; border: none; background: transparent; cursor: pointer;"
  >
    <BrandMark size={26} />
  </button>

  {#if kicker || title || titleSlot}
    <div style="display: flex; flex-direction: column; min-width: 0;">
      <span
        style="font-size: 12px; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);"
        >{kicker}</span
      >
      {#if titleSlot}
        {@render titleSlot()}
      {:else}
        <span
          style="font-size: clamp(16px, 1.8cqi, 21px); font-weight: 500; letter-spacing: -0.01em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
          >{title}</span
        >
      {/if}
    </div>
  {/if}

  {#if lead}{@render lead()}{/if}

  <span style="flex: 1;"></span>

  {#if actions}{@render actions()}{/if}

  <LangSwitch />
</div>
