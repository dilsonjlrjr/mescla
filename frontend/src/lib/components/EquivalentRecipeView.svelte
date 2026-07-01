<script lang="ts">
  import { onMount } from 'svelte';
  import Select, { Option } from '@smui/select';
  import Button from '@smui/button';
  import LinearProgress from '@smui/linear-progress';
  import Icon from './Icon.svelte';
  import PaintSearchInput from './PaintSearchInput.svelte';
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
    } catch (e) {
      console.error('Erro carregando dados:', e);
    }
  });

  let availableTargets = $derived(
    manufacturers.filter(m => !sourcePaint || m.name !== sourcePaint.manufacturer)
  );

  function selectSource(paint: Paint) {
    sourcePaint = paint;
    if (targetManufacturerId && manufacturers.find(m => m.id === targetManufacturerId)?.name === paint.manufacturer) {
      targetManufacturerId = '';
    }
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
      errorMsg = 'Não foi possível calcular a receita equivalente.';
    } finally {
      loading = false;
    }
  }

  function deltaClass(de: number): string {
    if (de < 3) return 'delta-excellent';
    if (de < 6) return 'delta-good';
    if (de < 10) return 'delta-fair';
    return 'delta-poor';
  }
</script>

<div class="page-container">
  <div class="page-header animate-rise">
    <h1 class="page-title">Receita equivalente</h1>
    <p class="page-subtitle">Escolha uma tinta e descubra como replicar a cor com outro fabricante</p>
    <div class="page-divider"></div>
  </div>

  <div class="mix-layout">
    <!-- Form -->
    <div class="panel p-5 animate-rise" style="animation-delay: 80ms;">
      <h3 class="font-display text-sm font-semibold text-white mb-4">Tinta de origem</h3>
      <PaintSearchInput
        paints={allPaints}
        selected={sourcePaint}
        onSelect={selectSource}
        onClear={clearSource}
        label="Buscar por nome ou fabricante..."
      />

      <h3 class="font-display text-sm font-semibold text-white mb-4" style="margin-top: 24px;">Fabricante de destino</h3>
      <Select variant="outlined" bind:value={targetManufacturerId} label="Fabricante" style="width: 100%;" disabled={!sourcePaint}>
        <Option value="">Selecione...</Option>
        {#each availableTargets as mfr}
          <Option value={mfr.id}>{mfr.name}</Option>
        {/each}
      </Select>

      <Button
        variant="raised"
        onclick={suggest}
        disabled={!sourcePaint || !targetManufacturerId || loading}
        style="width: 100%; margin-top: 20px; background: var(--lacquer); color: white; font-weight: 600; border-radius: 8px; height: 46px;"
      >
        {loading ? 'Calculando…' : 'Buscar receita equivalente'}
      </Button>

      {#if errorMsg}
        <p style="margin-top: 12px; font-size: 12.5px; color: var(--delta-poor);">{errorMsg}</p>
      {/if}
    </div>

    <!-- Result -->
    <div class="animate-rise" style="animation-delay: 160ms;">
      {#if loading}
        <div style="display: flex; align-items: center; justify-content: center; padding: 80px 0;">
          <LinearProgress indeterminate style="width: 200px;" />
        </div>
      {:else if result}
        <!-- Result color -->
        <div class="panel p-5 mb-5">
          <h3 class="font-display text-sm font-semibold text-white mb-4">Cor resultante</h3>
          <div class="flex gap-4 items-center">
            <div class="swatch-flat" style="width: 76px; height: 76px; background: rgb({result.resultR}, {result.resultG}, {result.resultB}); flex-shrink: 0;"></div>
            <div>
              <div style="font-size: 12px; color: var(--ink-500);">{result.sourceName} ({result.sourceManufacturer}) → {result.targetManufacturer}</div>
              <div class="font-mono text-sm text-white" style="margin-top: 4px;">RGB({result.resultR}, {result.resultG}, {result.resultB})</div>
              <div class="font-mono font-bold {deltaClass(result.deltaE)}" style="margin-top: 4px;">
                ΔE {result.deltaE.toFixed(2)}
              </div>
            </div>
          </div>
        </div>

        <!-- Ingredients -->
        <div class="panel p-5 mb-5">
          <h3 class="font-display text-sm font-semibold text-white mb-4">Ingredientes ({result.ingredients?.length || 0})</h3>

          {#if result.ingredients}
            <div style="display: flex; flex-direction: column; gap: 12px;">
              {#each result.ingredients as ing, i}
                <div class="ingredient-row animate-slide" style="animation-delay: {i * 40}ms;">
                  <div class="flex items-center gap-4">
                    <div class="swatch-flat" style="width: 40px; height: 40px; background: rgb({ing.r}, {ing.g}, {ing.b}); flex-shrink: 0;"></div>
                    <div style="flex: 1; min-width: 0;">
                      <div class="font-semibold text-sm text-white truncate">{ing.name}</div>
                    </div>
                    <div style="text-align: right; flex-shrink: 0;">
                      <div class="font-mono font-bold text-white" style="font-size: 16px;">{ing.percentage.toFixed(1)}%</div>
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
        <div class="panel" style="text-align: center; padding: 80px 0;">
          <span style="color: var(--ink-600); display: flex; justify-content: center;"><Icon name="flask" size={40} /></span>
          <p class="font-medium mt-4" style="color: var(--ink-500);">Escolha a tinta de origem e o fabricante de destino</p>
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

  .ingredient-row {
    padding: 12px;
    border: 1px solid var(--ink-700);
    border-radius: 8px;
  }
</style>
