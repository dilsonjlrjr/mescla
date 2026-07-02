<script lang="ts">
  import { onMount } from 'svelte';
  import Select, { Option } from '@smui/select';
  import LinearProgress from '@smui/linear-progress';
  import Icon from './Icon.svelte';
  import PaintSearchInput from './PaintSearchInput.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import DeltaBadge from './DeltaBadge.svelte';
  import { toast } from '../toast.svelte';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';

  interface Paint {
    id: number;
    name: string;
    manufacturer: string;
    r: number;
    g: number;
    b: number;
  }

  interface Manufacturer {
    id: number;
    name: string;
  }

  interface Props {
    initialSourcePaintId?: number | null;
  }

  let { initialSourcePaintId = null }: Props = $props();

  let allPaints: Paint[] = $state([]);
  let manufacturers: Manufacturer[] = $state([]);
  let sourcePaint: Paint | null = $state(null);
  let targetManufacturerId: number | '' = $state('');
  let result: any = $state(null);
  let loading = $state(false);
  let errorMsg = $state('');

  onMount(async () => {
    try {
      const [paints, mfrs] = await Promise.all([
        PaintService.GetAllPaints(),
        PaintService.GetManufacturers(),
      ]);
      allPaints = paints || [];
      manufacturers = mfrs || [];
      if (initialSourcePaintId) {
        sourcePaint = allPaints.find(p => p.id === initialSourcePaintId) || null;
      }
    } catch (e) {
      console.error('Erro carregando dados:', e);
      errorMsg = 'O catálogo não carregou. Feche e abra o aplicativo.';
    }
  });

  // Todas as marcas, incluindo a própria da tinta de origem: dá pra montar um tom
  // que a marca não tem a partir de outras tintas dela (a tinta-alvo é excluída
  // do cálculo no backend).
  let availableTargets = $derived(manufacturers);

  // Passo atual da jornada — guia o olho pro próximo campo
  let step = $derived(!sourcePaint ? 1 : !targetManufacturerId ? 2 : 3);

  // Faixas de ΔE pra legenda visível (mesmos cortes do DeltaBadge). max é o
  // teto exclusivo da faixa; a última pega tudo acima de 12.
  const deltaBands = [
    { range: '0–1',  label: 'idêntica',      cls: 'excellent', max: 1 },
    { range: '1–3',  label: 'muito próxima', cls: 'excellent', max: 3 },
    { range: '3–6',  label: 'próxima',       cls: 'good',      max: 6 },
    { range: '6–12', label: 'visível',       cls: 'fair',      max: 12 },
    { range: '12+',  label: 'diferente',     cls: 'poor',      max: Infinity },
  ];
  let activeBandIdx = $derived(result ? deltaBands.findIndex(b => result.deltaE < b.max) : -1);

  // Medir a receita em % ou em gotas. Pintor dosa na bancada por gota. As gotas
  // são só a indicação da proporção — não uma interação: o app mostra a menor
  // receita de gotas inteiras que mantém as proporções.
  let unit: 'percent' | 'drops' = $state('percent');

  function gcd(a: number, b: number): number {
    return b === 0 ? a : gcd(b, a % b);
  }

  // Converte os percentuais na menor proporção de gotas inteiras. Arredonda pra
  // inteiros que somam 100 (método do maior resto) e divide pelo MDC — assim
  // 30/60/10% vira 3/6/1 gotas, 25/75% vira 1/3, etc.
  function computeDrops(ingredients: any[]): number[] {
    const raw = ingredients.map(i => i.percentage);
    const ints = raw.map(Math.floor);
    let left = 100 - ints.reduce((a, b) => a + b, 0);
    const byFrac = raw
      .map((v, idx) => ({ idx, frac: v - Math.floor(v) }))
      .sort((a, b) => b.frac - a.frac);
    for (let k = 0; left > 0 && byFrac.length; k++, left--) {
      ints[byFrac[k % byFrac.length].idx]++;
    }
    const g = ints.filter(v => v > 0).reduce((acc, v) => gcd(acc, v), 0) || 1;
    return ints.map(v => Math.round(v / g));
  }

  let drops = $derived(result?.ingredients ? computeDrops(result.ingredients) : []);
  let totalDrops = $derived(drops.reduce((a, b) => a + b, 0));

  function selectSource(paint: Paint) {
    sourcePaint = paint;
    result = null;
    errorMsg = '';
  }

  function clearSource() {
    sourcePaint = null;
    result = null;
    errorMsg = '';
  }

  async function suggest() {
    if (!sourcePaint || !targetManufacturerId) return;
    loading = true;
    errorMsg = '';
    result = null;
    try {
      result = await PaintService.SuggestEquivalentRecipe(sourcePaint.id, Number(targetManufacturerId));
    } catch (e) {
      console.error('Erro sugerindo receita equivalente:', e);
      errorMsg = 'O cálculo falhou. Escolha a tinta e a marca de novo e tente outra vez.';
      toast('O cálculo falhou. Tente de novo.', 'error');
    } finally {
      loading = false;
    }
  }

  function copyRecipe() {
    if (!result?.ingredients) return;
    const measure = (i: any, idx: number) =>
      unit === 'drops'
        ? `${drops[idx]} ${drops[idx] === 1 ? 'gota' : 'gotas'}`
        : `${i.percentage.toFixed(1)}%`;
    const lines = [
      `${result.sourceName} (${result.sourceManufacturer}) → ${result.targetManufacturer}`,
      ...(unit === 'drops' ? [`Mistura de ${totalDrops} gotas`] : []),
      ...result.ingredients.map((i: any, idx: number) => `${measure(i, idx)}  ${i.name}${i.code ? ` (${i.code})` : ''}`),
      `ΔE2000 ${result.deltaE.toFixed(2)}`,
    ];
    navigator.clipboard.writeText(lines.join('\n'));
    toast('Receita copiada');
  }
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Equivalência</h1>
    <p class="page-subtitle">A mesma cor, feita com as tintas da marca que você tem</p>
    <div class="page-divider"></div>
  </div>

  <div class="mix-layout">
    <!-- Form: passos numerados guiam a jornada -->
    <div class="panel p-5 animate-rise" style="animation-delay: 80ms;">
      <div class="form-step" class:current={step === 1} class:done={step > 1}>
        <span class="step-n">1</span>
        <h3 class="form-step-title font-display">Tinta que você quer</h3>
      </div>
      <PaintSearchInput
        paints={allPaints}
        selected={sourcePaint}
        onSelect={selectSource}
        onClear={clearSource}
        label="Nome, código ou marca…"
      />

      <div class="form-step" class:current={step === 2} class:done={step > 2} style="margin-top: 24px;">
        <span class="step-n">2</span>
        <h3 class="form-step-title font-display">Marca que você tem</h3>
      </div>
      <Select variant="outlined" bind:value={targetManufacturerId} label="Marca" style="width: 100%;" disabled={!sourcePaint}>
        <Option value="">Selecione…</Option>
        {#each availableTargets as mfr}
          <Option value={mfr.id}>{mfr.name}</Option>
        {/each}
      </Select>

      <button
        class="btn-primary"
        onclick={suggest}
        disabled={!sourcePaint || !targetManufacturerId || loading}
        style="margin-top: 20px;"
      >
        {loading ? 'Calculando…' : 'Encontrar equivalência'}
      </button>

      {#if errorMsg}
        <p style="margin-top: 12px; font-size: 12.5px; color: var(--delta-poor);">{errorMsg}</p>
      {/if}
    </div>

    <!-- Result -->
    <div class="animate-rise" style="animation-delay: 160ms;">
      {#if loading}
        <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; padding: 80px 0;">
          <LinearProgress indeterminate style="width: 200px;" />
          <span style="font-size: 12.5px; color: var(--ink-500);">Testando misturas de até 3 tintas…</span>
        </div>
      {:else if result}
        <!-- Aviso: cor irreproduzível com o catálogo de destino -->
        {#if !result.reproducible}
          <div class="warn-banner mb-5 animate-rise">
            <span class="warn-icon"><Icon name="info" size={18} /></span>
            <div>
              <div class="warn-title">Esta cor não sai com as tintas da {result.targetManufacturer}</div>
              <p class="warn-text">
                Falta pigmento no catálogo dela pra chegar neste tom. A mistura abaixo é a
                <strong>aproximação mais próxima possível</strong> — compare o par de cores
                antes de decidir usar.
              </p>
            </div>
          </div>
        {/if}

        <!-- O par: alvo vs obtido, lado a lado — o momento da verdade -->
        <div class="panel p-5 mb-5">
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-display text-sm font-semibold text-white">Alvo × mistura</h3>
            <DeltaBadge
              deltaE={result.deltaE}
              pair={{ r1: result.sourceR, g1: result.sourceG, b1: result.sourceB, r2: result.resultR, g2: result.resultG, b2: result.resultB }}
            />
          </div>
          <div class="verdict-pair">
            <div class="verdict-half" style="background: rgb({result.sourceR}, {result.sourceG}, {result.sourceB});">
              <span class="verdict-tag">{result.sourceName} · {result.sourceManufacturer}</span>
            </div>
            <div class="verdict-half" style="background: rgb({result.resultR}, {result.resultG}, {result.resultB});">
              <span class="verdict-tag">sua mistura · {result.targetManufacturer}</span>
            </div>
          </div>

          <!-- Legenda: explica o par de cores e o ΔE pra quem nunca viu o termo -->
          <div class="verdict-legend">
            <p class="legend-desc">
              À <strong>esquerda</strong>, a cor que você quer; à <strong>direita</strong>, o que a sua mistura produz.
              O <strong>ΔE</strong> mede o quanto elas se diferenciam — quanto menor, mais parecidas (<strong>0 = idênticas</strong>).
            </p>
            <div class="delta-scale" role="img" aria-label="Escala de diferença de cor, ΔE {result.deltaE.toFixed(1)}">
              {#each deltaBands as band, i}
                <span class="scale-seg {band.cls}" class:active={activeBandIdx === i}>
                  <span class="scale-range">{band.range}</span>
                  <span class="scale-lbl">{band.label}</span>
                </span>
              {/each}
            </div>
          </div>
        </div>

        <!-- Ingredients -->
        <div class="panel p-5 mb-5">
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-display text-sm font-semibold text-white">
              {result.reproducible ? 'Receita' : 'Melhor aproximação'} ({result.ingredients?.length || 0} {result.ingredients?.length === 1 ? 'tinta' : 'tintas'})
            </h3>
            <button class="btn-ghost" onclick={copyRecipe}>
              Copiar receita
            </button>
          </div>

          <!-- Medir em % ou gotas: pintor dosa na bancada por gota -->
          <div class="recipe-controls">
            <div class="unit-toggle" role="group" aria-label="Unidade de medida">
              <button class:active={unit === 'percent'} onclick={() => unit = 'percent'}>%</button>
              <button class:active={unit === 'drops'} onclick={() => unit = 'drops'}>gotas</button>
            </div>
            {#if unit === 'drops'}
              <span class="drops-cap">mistura de {totalDrops} {totalDrops === 1 ? 'gota' : 'gotas'} — dá pra multiplicar (2×, 3×…) pra fazer mais</span>
            {/if}
          </div>

          {#if result.ingredients}
            <div style="display: flex; flex-direction: column; gap: 12px;">
              {#each result.ingredients as ing, i}
                <div class="ingredient-row animate-slide" style="animation-delay: {i * 40}ms;">
                  <div class="flex items-center gap-4">
                    <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={46} />
                    <div style="flex: 1; min-width: 0;">
                      <div class="font-semibold text-sm text-white truncate">{ing.name}</div>
                      {#if ing.code}
                        <span class="ing-code font-mono">{ing.code}</span>
                      {/if}
                    </div>
                    <div style="text-align: right; flex-shrink: 0;">
                      {#if unit === 'drops'}
                        <div class="font-mono font-bold text-white" style="font-size: 16px;">{drops[i]} <span style="font-size: 11px; font-weight: 500; color: var(--ink-500);">{drops[i] === 1 ? 'gota' : 'gotas'}</span></div>
                        <div style="font-size: 11px; color: var(--ink-500);">{ing.percentage.toFixed(1)}%</div>
                      {:else}
                        <div class="font-mono font-bold text-white" style="font-size: 16px;">{ing.percentage.toFixed(1)}%</div>
                      {/if}
                    </div>
                  </div>
                  <div style="margin-top: 10px; height: 3px; border-radius: 999px; background: var(--ink-700); overflow: hidden;">
                    <div style="height: 100%; width: {ing.percentage}%; border-radius: 999px; background: var(--lacquer); transition: width 0.5s ease;"></div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Tips -->
        {#if result.tips && result.tips.length > 0}
          <div class="panel p-5">
            <h3 class="font-display text-sm font-semibold text-white mb-4">
              <span class="flex items-center gap-2"><Icon name="info" size={16} />Dicas de ajuste</span>
            </h3>
            <ul style="display: flex; flex-direction: column; gap: 10px; padding-left: 18px; margin: 0;">
              {#each result.tips as tip}
                <li style="font-size: 13px; color: var(--ink-300); line-height: 1.5;">{tip}</li>
              {/each}
            </ul>
          </div>
        {/if}
      {:else}
        <div class="panel empty-state">
          <span class="empty-icon"><Icon name="flask" size={40} /></span>
          <p class="empty-title">
            {step === 1 ? 'Comece buscando a tinta que você quer' : 'Agora escolha a marca que você tem'}
          </p>
          <p class="empty-hint">
            {step === 1 ? 'Digite o nome no campo ao lado — ou venha do Catálogo pelo botão da tinta.' : 'A receita usa só as tintas dessa marca.'}
          </p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .mix-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 24px;
  }

  .form-step {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
    opacity: 0.55;
    transition: opacity 0.2s ease;
  }

  .form-step.current,
  .form-step.done {
    opacity: 1;
  }

  .form-step.current .step-n {
    background: var(--lacquer);
    color: white;
  }

  .form-step.done .step-n {
    background: var(--delta-excellent);
    color: white;
  }

  .form-step .step-n {
    margin-bottom: 0;
  }

  .form-step-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--paper);
  }

  /* Par alvo × mistura: as duas cores encostadas, como se compara tinta de verdade */
  .verdict-pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    height: 120px;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.22);
  }

  .verdict-half {
    position: relative;
  }

  /* Legenda do par + escala de ΔE */
  .verdict-legend {
    margin-top: 14px;
  }

  .legend-desc {
    margin: 0 0 12px;
    font-size: 12.5px;
    line-height: 1.55;
    color: var(--ink-400);
  }

  .legend-desc strong {
    color: var(--ink-200);
    font-weight: 600;
  }

  .delta-scale {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 4px;
  }

  .scale-seg {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 6px 4px;
    border-radius: 6px;
    text-align: center;
    background: var(--ink-800);
    border: 1px solid transparent;
    opacity: 0.55;
    transition: opacity 0.15s ease;
  }

  /* Faixa onde o ΔE atual cai — destacada, o resto esmaecido */
  .scale-seg.active {
    opacity: 1;
    background: color-mix(in srgb, currentColor 12%, transparent);
    border-color: color-mix(in srgb, currentColor 40%, transparent);
  }

  .scale-seg.excellent { color: var(--delta-excellent); }
  .scale-seg.good      { color: var(--delta-good); }
  .scale-seg.fair      { color: var(--delta-fair); }
  .scale-seg.poor      { color: var(--delta-poor); }

  .scale-range {
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 600;
  }

  .scale-lbl {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--ink-500);
  }

  .scale-seg.active .scale-lbl {
    color: currentColor;
  }

  .verdict-tag {
    position: absolute;
    bottom: 8px;
    left: 8px;
    right: 8px;
    font-family: var(--font-mono);
    font-size: 10px;
    padding: 3px 8px;
    border-radius: 4px;
    background: rgba(15, 13, 18, 0.55);
    color: rgba(255, 255, 255, 0.92);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    width: fit-content;
    max-width: calc(100% - 16px);
  }

  .ingredient-row {
    padding: 12px;
    border: 1px solid var(--ink-700);
    border-radius: 8px;
  }

  /* Barra de medida: alternador %/gotas + total de gotas */
  .recipe-controls {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
    margin-bottom: 16px;
  }

  .unit-toggle {
    display: inline-flex;
    padding: 2px;
    border-radius: 8px;
    background: var(--ink-800);
    border: 1px solid var(--ink-700);
  }

  .unit-toggle button {
    font: inherit;
    font-size: 12.5px;
    font-weight: 600;
    padding: 5px 14px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--ink-400);
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .unit-toggle button.active {
    background: var(--lacquer);
    color: #fff;
  }

  .drops-cap {
    font-size: 12px;
    color: var(--ink-500);
  }

  /* Referência do pote — sem o código não dá pra comprar/achar a tinta na loja */
  .ing-code {
    display: inline-block;
    margin-top: 3px;
    font-size: 11px;
    font-weight: 500;
    padding: 1px 7px;
    border-radius: 4px;
    background: var(--ink-800);
    color: var(--lacquer-tint);
  }

  .warn-banner {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 16px 18px;
    border-radius: 10px;
    border: 1px solid color-mix(in srgb, var(--delta-poor) 45%, transparent);
    background: color-mix(in srgb, var(--delta-poor) 12%, transparent);
  }

  .warn-icon {
    color: var(--delta-poor);
    flex-shrink: 0;
    margin-top: 1px;
  }

  .warn-title {
    font-weight: 600;
    font-size: 13.5px;
    color: var(--delta-poor);
    margin-bottom: 4px;
  }

  .warn-text {
    font-size: 12.5px;
    color: var(--ink-300);
    line-height: 1.55;
    margin: 0;
  }
</style>
