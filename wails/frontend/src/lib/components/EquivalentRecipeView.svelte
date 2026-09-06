<script lang="ts">
  // Equivalência (tela principal do Tintômetro): painel esquerdo em cor
  // CHAPADA da tinta de origem, altura cheia; à direita a Fórmula como fita
  // proporcional, lista de ingredientes e a leitura de ΔE00 como instrumento.
  import { onMount } from 'svelte';
  import PaintSearchInput from './PaintSearchInput.svelte';
  import PaintBottle from './PaintBottle.svelte';
  import FormulaRibbon from './FormulaRibbon.svelte';
  import { toast } from '../toast.svelte';
  import { saveRecipe } from '../recipes.svelte';
  import { contrastOn, hexOf, deltaVerdict, deltaIsGood } from '../ui';
  import * as PaintService from '../../../bindings/paint-match-ai/paintservice';
  import type { UserPaintDTO } from '../../../bindings/paint-match-ai/models';

  interface Paint {
    id: number;
    name: string;
    manufacturer: string;
    productLine?: string;
    r: number;
    g: number;
    b: number;
  }

  interface Manufacturer {
    id: number;
    name: string;
    paintCount?: number;
  }

  interface Props {
    initialSourcePaintId?: number | null;
    initialTargetManufacturerId?: number | null;
  }

  let { initialSourcePaintId = null, initialTargetManufacturerId = null }: Props = $props();

  let allPaints: Paint[] = $state([]);
  let manufacturers: Manufacturer[] = $state([]);
  let userPaints: UserPaintDTO[] = $state([]);
  let sourcePaint = $state<Paint | null>(null);
  let targetManufacturerId: number | '' = $state('');
  let result: any = $state(null);
  let loading = $state(false);
  let errorMsg = $state('');

  // Priorizar o estoque do próprio pintor: quando ligado, a receita sai primeiro
  // do que ele tem; se o estoque não alcança a cor, cai na marca de reserva.
  let useStock = $state(false);
  let fromStock = $state(false);
  let stockFellBack = $state(false);

  onMount(async () => {
    try {
      const [paints, mfrs, stock] = await Promise.all([
        PaintService.GetAllPaints(),
        PaintService.GetManufacturers(),
        PaintService.GetUserPaints(),
      ]);
      allPaints = paints || [];
      manufacturers = mfrs || [];
      userPaints = stock || [];
      useStock = userPaints.length > 0;
      if (initialTargetManufacturerId) {
        targetManufacturerId = initialTargetManufacturerId;
        // receita salva reaberta: recalcular exatamente pela marca gravada
        useStock = false;
      }
      if (initialSourcePaintId) {
        sourcePaint = allPaints.find(p => p.id === initialSourcePaintId) || null;
        if (sourcePaint && (targetManufacturerId || useStock)) suggest();
      }
    } catch (e) {
      console.error('Erro carregando dados:', e);
      errorMsg = 'O catálogo não carregou. Feche e abra o aplicativo.';
    }
  });

  // Índices do estoque (ver userstock.go): por id e por marca+código.
  let stockById = $derived(new Map(userPaints.map(p => [p.id, p])));
  let stockKeys = $derived(
    new Set(userPaints.filter(p => p.code.trim()).map(p => `${p.manufacturer.toLowerCase()}|${p.code.trim().toLowerCase()}`))
  );

  let paintById = $derived(new Map(allPaints.map(p => [p.id, p])));

  function ingredientInStock(ing: any): boolean {
    if (fromStock) return true;
    if (!result || !ing.code) return false;
    return stockKeys.has(`${result.targetManufacturer.toLowerCase()}|${String(ing.code).trim().toLowerCase()}`);
  }

  // Linha de produto do ingrediente: no catálogo, resolvida por paintId;
  // numa receita de estoque, a marca da tinta do estoque.
  function ingredientLine(ing: any): string {
    if (fromStock) return stockById.get(ing.paintId)?.manufacturer ?? '';
    return paintById.get(ing.paintId)?.productLine || result?.targetManufacturer || '';
  }

  let targetName = $derived(
    manufacturers.find(m => m.id === Number(targetManufacturerId))?.name ?? ''
  );

  let targetCount = $derived(
    manufacturers.find(m => m.id === Number(targetManufacturerId))?.paintCount ?? 0
  );

  let canSuggest = $derived(!!sourcePaint && !loading && (useStock || !!targetManufacturerId));

  // Gotas: menor proporção inteira que mantém os percentuais (70/20/10 -> 7/2/1).
  function gcd(a: number, b: number): number {
    return b === 0 ? a : gcd(b, a % b);
  }

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

  // Proporções ~0% são ruído de arredondamento: fora da fita e da lista.
  let ingredients = $derived(
    (result?.ingredients || []).filter((i: any) => i.percentage > 0.5)
  );
  let drops = $derived(ingredients.length ? computeDrops(ingredients) : []);

  function selectSource(paint: Paint) {
    sourcePaint = paint;
    result = null;
    errorMsg = '';
    if (targetManufacturerId || useStock) suggest();
  }

  function clearSource() {
    sourcePaint = null;
    result = null;
    errorMsg = '';
  }

  function onTargetChange() {
    result = null;
    errorMsg = '';
    if (sourcePaint && (targetManufacturerId || useStock)) suggest();
  }

  function toggleStock() {
    useStock = !useStock;
    result = null;
    errorMsg = '';
    if (sourcePaint && (useStock || targetManufacturerId)) suggest();
  }

  async function suggest() {
    if (!sourcePaint) return;
    loading = true;
    errorMsg = '';
    result = null;
    fromStock = false;
    stockFellBack = false;
    try {
      if (useStock && userPaints.length > 0) {
        const fromStockRes = await PaintService.SuggestEquivalentFromStock(sourcePaint.id);
        if (fromStockRes.reproducible || !targetManufacturerId) {
          result = fromStockRes;
          fromStock = true;
        } else {
          result = await PaintService.SuggestEquivalentRecipe(sourcePaint.id, Number(targetManufacturerId));
          stockFellBack = true;
        }
      } else {
        result = await PaintService.SuggestEquivalentRecipe(sourcePaint.id, Number(targetManufacturerId));
      }
    } catch (e) {
      console.error('Erro sugerindo receita equivalente:', e);
      errorMsg = String(e).includes('estoque está vazio')
        ? 'Seu estoque está vazio. Cadastre tintas ou desligue a priorização.'
        : 'O cálculo falhou. Escolha a tinta e a marca de novo e tente outra vez.';
      toast('O cálculo falhou. Tente de novo.', 'error');
    } finally {
      loading = false;
    }
  }

  function copyRecipe() {
    if (!ingredients.length) return;
    const lines = [
      `${result.sourceName} (${result.sourceManufacturer}) em ${result.targetManufacturer}`,
      ...ingredients.map((i: any, idx: number) =>
        `${drops[idx]} ${drops[idx] === 1 ? 'gota' : 'gotas'} (${i.percentage.toFixed(1)}%)  ${i.name}${i.code ? ` (${i.code})` : ''}`),
      `dE00 ${result.deltaE.toFixed(2)}`,
    ];
    navigator.clipboard.writeText(lines.join('\n'));
    toast('Fórmula copiada');
  }

  function persistRecipe() {
    if (!result || !ingredients.length) return;
    saveRecipe({
      sourcePaintId: result.sourcePaintId,
      sourceName: result.sourceName,
      sourceHex: hexOf(result.sourceR, result.sourceG, result.sourceB),
      targetManufacturerId: fromStock ? 0 : Number(targetManufacturerId),
      targetManufacturer: fromStock ? 'meu estoque' : result.targetManufacturer,
      ingredients: ingredients.map((i: any) => ({
        name: i.name,
        code: i.code || '',
        hex: hexOf(i.r, i.g, i.b),
        percentage: i.percentage,
      })),
      deltaE: result.deltaE,
    });
    toast('Receita salva');
  }

  function tryAnotherBrand() {
    targetManufacturerId = '';
    result = null;
    document.getElementById('target-select')?.focus();
  }

  let sourceInk = $derived(sourcePaint ? contrastOn(sourcePaint.r, sourcePaint.g, sourcePaint.b) : '#1a1712');
</script>

<div class="equiv-split">
  <!-- Painel da tinta de origem: cor chapada, raio 0, altura cheia -->
  {#if sourcePaint}
    <aside class="source-panel" style="background: rgb({sourcePaint.r}, {sourcePaint.g}, {sourcePaint.b}); color: {sourceInk};">
      <div class="source-top">
        <span class="label-mono inherit">Tinta de origem</span>
        <span class="source-no font-mono">No. {sourcePaint.id}</span>
      </div>
      <div class="source-bottom">
        <h2 class="source-name font-display">{sourcePaint.name}</h2>
        <p class="source-meta">{sourcePaint.manufacturer}{sourcePaint.productLine ? ` · ${sourcePaint.productLine}` : ''}</p>
        <p class="source-code font-mono">
          {hexOf(sourcePaint.r, sourcePaint.g, sourcePaint.b)}&nbsp;&nbsp;&nbsp;RGB {sourcePaint.r} {sourcePaint.g} {sourcePaint.b}
        </p>
        <button class="source-swap font-mono" style="color: {sourceInk};" onclick={clearSource}>trocar tinta ›</button>
      </div>
    </aside>
  {:else}
    <aside class="source-panel empty">
      <div class="source-top">
        <span class="label-mono">Tinta de origem</span>
      </div>
      <div class="source-pick">
        <p class="pick-title font-display">Comece pela cor que você quer.</p>
        <p class="pick-hint">Busque pelo nome, código ou marca. Ou venha do Catálogo pelo botão da tinta.</p>
        <div class="pick-input">
          <PaintSearchInput
            paints={allPaints}
            selected={null}
            onSelect={selectSource}
            onClear={clearSource}
            label="Nome, código ou marca"
          />
        </div>
      </div>
    </aside>
  {/if}

  <!-- Fórmula -->
  <section class="formula-side">
    <div class="formula-head">
      <div>
        <h1 class="formula-title font-display">Fórmula</h1>
        <p class="formula-sub">
          {#if fromStock && result}
            A cor mais próxima possível usando só o seu estoque.
          {:else if targetName}
            A cor mais próxima possível usando só o catálogo {targetName}.
          {:else}
            Escolha a marca de destino pra calcular a fórmula.
          {/if}
        </p>
      </div>
      <div class="target-wrap">
        <select
          id="target-select"
          class="target-select"
          bind:value={targetManufacturerId}
          onchange={onTargetChange}
          aria-label="Marca de destino"
        >
          <option value="">{useStock ? 'Marca de reserva' : 'Escolher marca'}</option>
          {#each manufacturers as mfr (mfr.id)}
            <option value={mfr.id}>{mfr.name}</option>
          {/each}
        </select>
      </div>
    </div>

    {#if errorMsg}
      <div class="notice-card mb-6">
        <div class="notice-title">Algo deu errado</div>
        <p class="notice-text">{errorMsg}</p>
      </div>
    {/if}

    {#if loading}
      <!-- Calculando: skeleton + linha mono (board Estados) -->
      <div class="calc-skeleton" aria-hidden="true">
        <div class="skeleton" style="height: 60px; width: 60%; border-radius: 0;"></div>
        {#each [0, 1, 2] as i (i)}
          <div class="skel-row">
            <div class="skeleton" style="width: 40px; height: 40px;"></div>
            <div class="skel-lines">
              <div class="skeleton" style="height: 12px; width: 46%;"></div>
              <div class="skeleton" style="height: 10px; width: 30%;"></div>
            </div>
            <div class="skeleton" style="width: 44px; height: 20px;"></div>
          </div>
        {/each}
      </div>
      <p class="calc-line font-mono">
        {#if useStock && userPaints.length > 0}
          Testando misturas com as {userPaints.length} tintas do seu estoque…
        {:else}
          Testando misturas de {targetCount || 'todas as'} tintas no catálogo {targetName}…
        {/if}
      </p>
    {:else if result}
      {#if stockFellBack}
        <div class="notice-card mb-6">
          <div class="notice-title">Seu estoque não alcança essa cor</div>
          <p class="notice-text">
            A fórmula abaixo usa o catálogo {result.targetManufacturer} (marca de reserva).
            Os potes que você já tem estão marcados na lista.
          </p>
        </div>
      {/if}

      {#if !result.reproducible}
        <div class="notice-card mb-6">
          <div class="notice-title">
            {fromStock ? 'Seu estoque não alcança a cor' : 'Essa marca não alcança a cor'}
          </div>
          <p class="notice-text">
            Melhor resultado ficou em ΔE {result.deltaE.toFixed(1)}.
            {fromStock ? 'Nenhuma mistura com as suas tintas chega perto.' : `Nenhuma mistura ${result.targetManufacturer} chega perto.`}
            A fórmula abaixo é só a aproximação mais próxima.
          </p>
          <button class="notice-link" onclick={tryAnotherBrand}>Tentar outra marca</button>
        </div>
      {/if}

      <div class="formula-body">
        <div class="formula-main">
          <FormulaRibbon
            segments={ingredients.map((i: any) => ({ r: i.r, g: i.g, b: i.b, code: i.code || i.name, percentage: i.percentage }))}
          />

          <div class="ingredient-list">
            {#each ingredients as ing, i (ing.paintId)}
              <div class="ingredient-row">
                <PaintBottle r={ing.r} g={ing.g} b={ing.b} size={52} />
                <div class="ing-text">
                  <div class="ing-name">
                    {ing.name}
                    {#if ingredientInStock(ing)}
                      <span class="stock-chip font-mono" title="Você tem esta tinta">no meu estoque</span>
                    {/if}
                  </div>
                  <div class="ing-meta font-mono">{ing.code ? `${ing.code} · ` : ''}{ingredientLine(ing)}</div>
                </div>
                <span class="ing-drops font-mono">{drops[i]} {drops[i] === 1 ? 'gota' : 'gotas'}</span>
                <span class="ing-pct font-display">{Math.round(ing.percentage)}%</span>
              </div>
            {/each}
          </div>

          {#if result.tips && result.tips.length > 0}
            <p class="formula-tip">{result.tips[0]}</p>
          {/if}
        </div>

        <!-- Leitura de instrumento: ΔE00 gigante + verdicto + amostras -->
        <aside class="delta-col">
          <span class="label-mono">ΔE00</span>
          <div class="delta-reading big" class:good={deltaIsGood(result.deltaE)}>{result.deltaE.toFixed(1)}</div>
          <p class="delta-verdict" class:good={deltaIsGood(result.deltaE)}>{deltaVerdict(result.deltaE)}</p>

          <div class="sample-pair">
            <div class="sample">
              <span class="sample-color" style="background: rgb({result.sourceR}, {result.sourceG}, {result.sourceB});"></span>
              <span class="sample-tag font-mono">alvo</span>
            </div>
            <div class="sample">
              <span class="sample-color" style="background: rgb({result.resultR}, {result.resultG}, {result.resultB});"></span>
              <span class="sample-tag font-mono">mistura</span>
            </div>
          </div>
        </aside>
      </div>
    {:else if sourcePaint}
      <div class="await-calc">
        <p class="await-title font-display">
          {useStock ? 'Pronto pra calcular com o seu estoque.' : 'Escolha a marca de destino.'}
        </p>
        <p class="await-hint">
          {useStock
            ? 'A fórmula prioriza as tintas que você já tem. Se não alcançar, cai na marca de reserva.'
            : 'A Mescla monta a cor usando só o catálogo da marca escolhida.'}
        </p>
        {#if useStock}
          <button class="pill-dark" onclick={suggest} disabled={!canSuggest}>Gerar fórmula</button>
        {/if}
      </div>
    {:else}
      <div class="await-calc">
        <p class="await-title font-display">A fórmula aparece aqui.</p>
        <p class="await-hint">Busque a tinta de origem no painel ao lado e escolha a marca de destino.</p>
      </div>
    {/if}

    <!-- Rodapé: estoque à esquerda, salvar à direita -->
    <div class="formula-foot">
      {#if userPaints.length > 0}
        <button class="stock-line" onclick={toggleStock} aria-pressed={useStock}>
          <span class="switch" class:on={useStock}><span class="knob"></span></span>
          <span class="stock-label">Priorizar meu estoque</span>
        </button>
      {:else}
        <span></span>
      {/if}
      <div class="foot-actions">
        {#if result && ingredients.length}
          <button class="pill-light" onclick={copyRecipe}>Copiar</button>
          <button class="pill-dark" onclick={persistRecipe}>Salvar receita</button>
        {/if}
      </div>
    </div>
  </section>
</div>

<style>
  /* Split full-bleed: o painel de cor encosta na borda esquerda/inferior */
  .equiv-split {
    display: grid;
    grid-template-columns: minmax(320px, 40%) minmax(0, 1fr);
    min-height: 100%;
  }

  @media (max-width: 900px) {
    .equiv-split {
      grid-template-columns: 1fr;
    }
    .source-panel {
      min-height: 320px;
    }
  }

  .source-panel {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    max-width: 560px;
    width: 100%;
    padding: 36px 40px 44px;
    border-radius: 0;
  }

  .source-panel.empty {
    background: var(--bancada-deep);
    color: var(--grafite);
    justify-content: flex-start;
  }

  .source-top {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }

  /* labels dentro do painel herdam a cor de contraste */
  .label-mono.inherit {
    color: inherit;
    opacity: 0.85;
  }

  .source-no {
    font-size: 17px;
    font-weight: 500;
  }

  .source-name {
    font-size: clamp(2.4rem, 4.6vw, 4rem);
    font-weight: 720;
    line-height: 1.02;
    letter-spacing: -0.02em;
    margin-bottom: 14px;
    overflow-wrap: anywhere;
  }

  .source-meta {
    font-size: 14.5px;
    font-weight: 560;
    margin-bottom: 8px;
  }

  .source-code {
    font-size: 12.5px;
    opacity: 0.9;
  }

  .source-swap {
    margin-top: 18px;
    padding: 0;
    border: none;
    background: none;
    font-size: 11.5px;
    opacity: 0.75;
    cursor: pointer;
    text-align: left;
  }

  .source-swap:hover {
    opacity: 1;
    text-decoration: underline;
  }

  /* Centraliza o convite no espaço restante (space-between jogava tudo pro fundo) */
  .source-pick {
    max-width: 380px;
    margin: auto 0;
  }

  .pick-title {
    font-size: 26px;
    font-weight: 700;
    color: var(--grafite);
    margin-bottom: 8px;
    line-height: 1.15;
  }

  .pick-hint {
    font-size: 13px;
    color: var(--text-2);
    margin-bottom: 20px;
    line-height: 1.55;
  }

  .formula-side {
    display: flex;
    flex-direction: column;
    padding: 44px 48px 36px;
    min-width: 0;
  }

  .formula-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 24px;
    margin-bottom: 36px;
  }

  .formula-title {
    font-size: 2.6rem;
    font-weight: 750;
    color: var(--grafite);
    letter-spacing: -0.015em;
    line-height: 1.05;
    margin-bottom: 8px;
  }

  .formula-sub {
    font-size: 13.5px;
    color: var(--text-2);
    max-width: 420px;
  }

  /* Dropdown do fabricante-alvo: pílula branca */
  .target-select {
    appearance: none;
    -webkit-appearance: none;
    height: 44px;
    padding: 0 40px 0 20px;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-pill);
    background: var(--papel);
    color: var(--grafite);
    font-family: var(--font-body);
    font-size: 13.5px;
    font-weight: 600;
    cursor: pointer;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%231a1712' stroke-width='1.5' stroke-linecap='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 18px center;
  }

  .target-select:hover {
    border-color: var(--grafite);
  }

  .formula-body {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 240px;
    gap: 0;
    flex: 1;
  }

  @media (max-width: 1100px) {
    .formula-body {
      grid-template-columns: 1fr;
    }
    .delta-col {
      border-left: none;
      border-top: 1px solid var(--hairline);
      padding-left: 0;
      padding-top: 24px;
      margin-top: 24px;
    }
  }

  .formula-main {
    min-width: 0;
    padding-right: 36px;
  }

  .delta-col {
    padding-left: 32px;
    border-left: 1px solid var(--hairline);
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .delta-reading.big {
    font-size: clamp(64px, 7vw, 96px);
    margin: 14px 0 8px;
  }

  .sample-pair {
    display: flex;
    gap: 4px;
    margin-top: 26px;
  }

  .sample {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .sample-color {
    width: 96px;
    height: 64px;
    border-radius: var(--radius-control);
    box-shadow: inset 0 0 0 1px var(--color-neutral-800);
    display: block;
  }

  .sample-tag {
    font-size: 10.5px;
    color: var(--text-2);
  }

  .ingredient-list {
    margin-top: 28px;
    border-top: 1px solid var(--hairline);
  }

  .ingredient-row {
    display: flex;
    align-items: center;
    gap: 18px;
    padding: 16px 0;
    border-bottom: 1px solid var(--hairline);
  }

  .ing-text {
    flex: 1;
    min-width: 0;
  }

  .ing-name {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 15px;
    font-weight: 680;
    color: var(--grafite);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ing-meta {
    font-size: 12px;
    color: var(--text-2);
    margin-top: 4px;
  }

  .ing-drops {
    flex-shrink: 0;
    font-size: 12.5px;
    color: var(--text-2);
  }

  .ing-pct {
    flex-shrink: 0;
    min-width: 76px;
    text-align: right;
    font-size: 28px;
    font-weight: 720;
    color: var(--grafite);
  }

  .stock-chip {
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 500;
    color: var(--laca-deep);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .formula-tip {
    margin-top: 22px;
    font-size: 13.5px;
    color: var(--text-2);
    line-height: 1.55;
  }

  /* Calculando (board Estados) */
  .calc-skeleton {
    display: flex;
    flex-direction: column;
    gap: 18px;
    max-width: 620px;
  }

  .skel-row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .skel-lines {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .calc-line {
    margin-top: 26px;
    font-size: 12.5px;
    color: var(--text-2);
  }

  .await-calc {
    flex: 1;
    max-width: 460px;
    padding: 24px 0;
  }

  .await-title {
    font-size: 24px;
    font-weight: 700;
    color: var(--grafite);
    margin-bottom: 8px;
  }

  .await-hint {
    font-size: 13.5px;
    color: var(--text-2);
    line-height: 1.55;
    margin-bottom: 22px;
  }

  .formula-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-top: 40px;
    padding-top: 22px;
  }

  .stock-line {
    display: flex;
    align-items: center;
    gap: 12px;
    border: none;
    background: none;
    padding: 0;
    cursor: pointer;
  }

  .stock-label {
    font-size: 13.5px;
    font-weight: 560;
    color: var(--grafite);
  }

  .foot-actions {
    display: flex;
    gap: 10px;
  }
</style>
