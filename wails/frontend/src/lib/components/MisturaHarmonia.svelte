<script lang="ts">
  // rf-24 — receita de mistura para uma cor de harmonia sem tinta pronta (T5).
  // Universo repete as 3 opções de T1 (rf-11). Sem `stock` no corpo: a chamada
  // vai pelo GET, que lê `user_paints` no servidor e já filtra `ignore_in_mix`
  // (rf-19). Teto de 3 tintas, o mesmo de T1. Não salva a receita.
  import { onMount } from 'svelte';
  import FolhaCirculo from './FolhaCirculo.svelte';
  import FaixaFormula from './FaixaFormula.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import Spinner from './Spinner.svelte';
  import { allManufacturers } from '../services/catalog';
  import { catalogRev } from '../services/catalogRev.svelte';
  import { stock, estoqueAssentado } from '../services/stock.svelte';
  import { suggestRecipeForColor, ehUniversoVazio, type EquivalentRecipe } from '../services/engine';
  import { verdictKeys, deltaIsGood } from '../ui';
  import { t, decimal } from '../i18n.svelte';

  interface Props {
    r: number;
    g: number;
    b: number;
    /** nome da cor de harmonia, para o texto de contexto */
    nome: string;
    onfechar: () => void;
  }

  let { r, g, b, nome, onfechar }: Props = $props();

  type Opcao = 'mix' | 'marca' | 'estoque';

  let opcao: Opcao = $state('mix');
  let manufacturerId: number | undefined = $state();
  let receita: EquivalentRecipe | null = $state(null);
  let motivo = $state('');
  let carregando = $state(false);
  let erro = $state(false);
  let escolheu = false;
  let seq = 0;
  let vivo = true;

  let mfrs = $derived.by(() => {
    void catalogRev.n;
    return allManufacturers();
  });

  async function calcular(fora = false): Promise<void> {
    const minha = ++seq;
    receita = null;
    motivo = '';
    erro = false;
    if (opcao === 'marca' && !manufacturerId) {
      carregando = false;
      return;
    }
    carregando = true;
    try {
      const resp = await suggestRecipeForColor(r, g, b, {
        targetManufacturerId: opcao === 'marca' ? manufacturerId : undefined,
        useStockOnly: opcao === 'estoque',
        foraDoUniverso: fora,
        maxIngredients: 3,
      });
      // RN5: resposta de pedido velho (ou de folha já fechada) é descartada.
      if (!vivo || minha !== seq) return;
      if (ehUniversoVazio(resp)) motivo = resp.motivo;
      else receita = resp;
    } catch {
      if (!vivo || minha !== seq) return;
      erro = true;
    } finally {
      if (vivo && minha === seq) carregando = false;
    }
  }

  function escolher(o: Opcao) {
    escolheu = true;
    opcao = o;
    void calcular();
  }

  function escolherMarca(id: number | undefined) {
    escolheu = true;
    manufacturerId = id;
    void calcular();
  }

  // RN2: o padrão espera o estoque assentar (boot/migração) antes de decidir.
  onMount(() => {
    void estoqueAssentado().then(() => {
      if (!vivo) return;
      if (!escolheu && stock.paints.length > 0) opcao = 'estoque';
      void calcular();
    });
    return () => {
      vivo = false;
    };
  });

  let veredito = $derived.by(() => (receita ? verdictKeys(receita.deltaE) : null));
  const FAIXA: Record<string, 'faixaOtima' | 'faixaAproximada' | 'faixaNaoEncontrei'> = {
    otimo: 'faixaOtima',
    aproximada: 'faixaAproximada',
    'nao-encontrei': 'faixaNaoEncontrei',
  };

  const OPCOES: Array<{ id: Opcao; chave: 'universoMix' | 'universoBrand' | 'onlyStock' }> = [
    { id: 'mix', chave: 'universoMix' },
    { id: 'marca', chave: 'universoBrand' },
    { id: 'estoque', chave: 'onlyStock' },
  ];
</script>

<FolhaCirculo titulo={t('mxTitle')} {onfechar}>
  {#snippet children()}
    <p class="mx-why">{t('mxWhy', { name: nome })}</p>

    <div class="mx-univ" role="group" aria-label={t('universeLabel')}>
      {#each OPCOES as o (o.id)}
        <button class="pressable mx-op" class:on={opcao === o.id} aria-pressed={opcao === o.id} onclick={() => escolher(o.id)}>
          {t(o.chave)}
        </button>
      {/each}
    </div>
    {#if opcao === 'marca'}
      <select
        class="mx-sel"
        aria-label={t('chooseBrandAria')}
        value={manufacturerId ?? ''}
        onchange={e => escolherMarca(Number((e.currentTarget as HTMLSelectElement).value) || undefined)}
      >
        <option value="" disabled>{t('chooseSupplierPlaceholder')}</option>
        {#each mfrs as m (m.id)}
          <option value={m.id}>{m.name}</option>
        {/each}
      </select>
    {/if}

    {#if carregando}
      <div class="mx-espera"><Spinner size={22} label={t('calculating')} />{t('calculating')}</div>
    {:else if erro}
      <p class="mx-msg">{t('mxErr')}</p>
    {:else if motivo}
      <p class="mx-msg">{motivo}</p>
      <button class="pressable mx-fora" onclick={() => void calcular(true)}>
        <i class="ph ph-magnifying-glass" style="font-size: 17px;"></i>{t('mxOut')}
      </button>
    {:else if receita && veredito}
      <div class="mx-par">
        <span class="mx-cores">
          <span style="background: rgb({r}, {g}, {b});"></span>
          <span style="background: rgb({receita.resultR}, {receita.resultG}, {receita.resultB});"></span>
        </span>
        <span class="mx-de">
          <span class="font-mono mx-num" style="color: {deltaIsGood(receita.deltaE) ? 'var(--color-accent-400)' : 'var(--color-text)'};">{decimal(receita.deltaE, 1)}</span>
          <span class="mx-un">ΔE00</span>
        </span>
        <span class="mx-ver">{t(veredito.v)}</span>
        {#if receita.faixa}
          <span class="mx-selo">{t(FAIXA[receita.faixa] ?? 'faixaAproximada')}</span>
        {/if}
      </div>

      <FaixaFormula ingredientes={receita.ingredients} />

      <div class="mx-ings">
        {#each receita.ingredients as ing (ing.paintId)}
          <div class="mx-ing">
            <PaintBottle r={ing.r} g={ing.g} b={ing.b} width={30} />
            <span class="mx-nome">
              <span class="mx-n1">{ing.name}</span>
              <span class="font-mono mx-n2">{ing.code} · {ing.manufacturer}</span>
            </span>
            <span class="mx-pct">{Math.round(ing.percentage)}%</span>
          </div>
        {/each}
      </div>
    {/if}
  {/snippet}
</FolhaCirculo>

<style>
  .mx-why {
    margin: 0;
    font-size: 14px;
    color: var(--color-neutral-400);
    text-wrap: pretty;
  }

  .mx-univ {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .mx-op {
    min-height: 44px;
    padding: 0 14px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: transparent;
    color: var(--color-text);
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }

  .mx-op.on {
    border-color: var(--color-accent);
    background: var(--color-accent);
    color: var(--color-accent-100);
  }

  .mx-sel {
    height: 44px;
    padding: 0 12px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 8px;
    background: var(--color-field);
    color: var(--color-text);
    font-family: inherit;
    font-size: 14px;
  }

  .mx-espera {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 14px;
    color: var(--color-neutral-400);
  }

  .mx-msg {
    margin: 0;
    font-size: 15px;
    color: var(--color-neutral-300);
    text-wrap: pretty;
  }

  .mx-fora {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    padding: 0 14px;
    border: 1px solid var(--color-accent-700);
    border-radius: 8px;
    background: transparent;
    color: var(--color-accent-400);
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }

  .mx-par {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 12px 16px;
    padding: 12px 14px;
    border: 1px solid var(--color-rule);
    border-radius: 14px;
    background: var(--color-panel);
  }

  .mx-cores {
    display: flex;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--color-neutral-800);
  }

  .mx-cores span {
    width: 48px;
    height: 56px;
  }

  .mx-de {
    display: flex;
    align-items: baseline;
    gap: 6px;
  }

  .mx-num {
    font-size: 28px;
    font-weight: 500;
    line-height: 1;
  }

  .mx-un {
    font-size: 13px;
    color: var(--color-neutral-500);
  }

  .mx-ver {
    font-size: 15px;
    font-weight: 500;
  }

  .mx-selo {
    padding: 3px 10px;
    border: 1px solid var(--color-neutral-800);
    border-radius: 999px;
    font-size: 12px;
    color: var(--color-neutral-300);
  }

  .mx-ings {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .mx-ing {
    display: flex;
    align-items: center;
    gap: 14px;
    min-height: 64px;
    padding: 8px 14px;
    border: 1px solid var(--color-rule);
    border-radius: 14px;
    background: var(--color-panel);
  }

  .mx-nome {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .mx-n1 {
    font-size: 16px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mx-n2 {
    font-size: 12px;
    color: var(--color-neutral-500);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mx-pct {
    font-size: 20px;
    font-weight: 500;
    flex-shrink: 0;
  }
</style>
