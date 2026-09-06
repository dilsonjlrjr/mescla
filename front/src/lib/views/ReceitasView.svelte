<script lang="ts">
  // T3 — Receitas (rf-04). Fiel ao protótipo tests/fixtures/mockup/t3-receitas.html
  // (D-001): lista à esquerda (par alvo/resultado + ΔE00); detalhe à direita com
  // seletor de fabricante, manchete, consequência, par de cores, passos da
  // fórmula e "a mesma cor nas outras marcas".
  //
  // Persistência: local (services/recipes.svelte.ts) — não há rota de escrita
  // de receita confirmada contra api/httpapi (pendência 3 da spec, R1
  // herdado); ver comentário no topo do serviço. Semântica (RG-16): a receita
  // guarda o ALVO — reabrir/trocar de fabricante resolve de novo (US-15).
  import Header from '../components/Header.svelte';
  import PaintBottle from '../components/PaintBottle.svelte';
  import { recipes, removeRecipe, updateRecipeManufacturer, type Recipe } from '../services/recipes.svelte';
  import { allManufacturers, paintById } from '../services/catalog';
  import { suggestEquivalentRecipe, suggestRecipeForColor, bestBrandsFor, type EquivalentRecipe, type BrandBest } from '../services/engine';
  import { verdictKeys, deltaIsGood } from '../ui';
  import { switchTab } from '../nav.svelte';
  import { t, decimal } from '../i18n.svelte';

  let selectedId: number | null = $state(null);
  let resolved: Record<number, EquivalentRecipe | 'error'> = $state({});
  let resolving = new Set<number>();
  let otherBrands: BrandBest[] = $state([]);
  let confirmDeleteId: number | null = $state(null);

  $effect(() => {
    if (selectedId == null) {
      if (recipes.lastSavedId != null && recipes.items.some(r => r.id === recipes.lastSavedId)) {
        selectedId = recipes.lastSavedId;
        recipes.lastSavedId = null;
      } else if (recipes.items.length > 0) {
        selectedId = recipes.items[0].id;
      }
    }
  });

  async function resolveRecipe(r: Recipe) {
    if (resolving.has(r.id)) return;
    resolving.add(r.id);
    try {
      const res = r.targetPaintId != null
        ? await suggestEquivalentRecipe(r.targetPaintId, r.manufacturerId)
        : await suggestRecipeForColor(r.targetR, r.targetG, r.targetB, r.manufacturerId);
      resolved = { ...resolved, [r.id]: res };
    } catch {
      resolved = { ...resolved, [r.id]: 'error' };
    } finally {
      resolving.delete(r.id);
    }
  }

  // RG-14: no máximo 12 resoluções em fila por ciclo — listas de receitas
  // costumam ser pequenas, mas o teto protege catálogos grandes.
  $effect(() => {
    let queued = 0;
    for (const r of recipes.items) {
      if (queued >= 12) break;
      if (!(r.id in resolved) && !resolving.has(r.id)) {
        queued++;
        void resolveRecipe(r);
      }
    }
  });

  let selected = $derived(recipes.items.find(r => r.id === selectedId) ?? null);
  let selectedResult = $derived(selected ? resolved[selected.id] : undefined);

  // Painel de detalhe (manchete + par de cores) mostra sempre a mesma
  // moldura — inclusive sem nenhuma receita salva ainda — só o CONTEÚDO
  // muda conforme resolvendo/erro/pronto (fiel à métrica do protótipo, que
  // não modela um estado "vazio" à parte).
  let recipeResolved = $derived(selectedResult && selectedResult !== 'error' ? selectedResult : null);
  let recipeIsError = $derived(selectedResult === 'error');
  let recipeVerdict = $derived(recipeResolved ? verdictKeys(recipeResolved.deltaE) : null);
  let pairTargetBg = $derived(
    recipeResolved
      ? `rgb(${recipeResolved.sourceR}, ${recipeResolved.sourceG}, ${recipeResolved.sourceB})`
      : selected
        ? `rgb(${selected.targetR}, ${selected.targetG}, ${selected.targetB})`
        : 'var(--color-rule)'
  );
  let pairResultBg = $derived(recipeResolved ? `rgb(${recipeResolved.resultR}, ${recipeResolved.resultG}, ${recipeResolved.resultB})` : pairTargetBg);

  $effect(() => {
    const s = selected;
    otherBrands = [];
    if (s?.targetPaintId != null) {
      bestBrandsFor(s.targetPaintId)
        .then(res => (otherBrands = [...res].sort((a, b) => a.deltaE - b.deltaE)))
        .catch(() => (otherBrands = []));
    }
  });

  function pickManufacturer(mfrId: number) {
    if (!selected) return;
    updateRecipeManufacturer(selected.id, mfrId);
    resolved = { ...resolved };
    delete resolved[selected.id];
    void resolveRecipe(selected);
  }

  function askDelete(id: number) {
    confirmDeleteId = id;
  }

  function confirmDelete() {
    if (confirmDeleteId == null) return;
    removeRecipe(confirmDeleteId);
    if (selectedId === confirmDeleteId) selectedId = null;
    confirmDeleteId = null;
  }

  function summaryName(r: Recipe): string {
    if (r.targetPaintId != null) return paintById(r.targetPaintId)?.name ?? r.name;
    return r.name;
  }

  function cardMeta(r: Recipe): string {
    const res = resolved[r.id];
    if (res && res !== 'error') return res.targetManufacturer;
    if (res === 'error') return t('naoAlcanca', { brand: allManufacturers().find(m => m.id === r.manufacturerId)?.name ?? '' });
    return t('calculating');
  }

  function cardDelta(r: Recipe): string {
    const res = resolved[r.id];
    return res && res !== 'error' ? `ΔE ${decimal(res.deltaE, 1)}` : '—';
  }
</script>

<div style="display: flex; flex-direction: column; height: 100%;">
  <Header kicker={t('savedRecipes')} title={recipes.items.length === 1 ? t('recipeCount1') : t('recipeCount', { n: recipes.items.length })}>
    {#snippet actions()}
      <button class="t3-actionbtn" onclick={() => switchTab('plano')} style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 18px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 15px; font-weight: 500;">
        <i class="ph ph-crosshair" style="font-size: 18px;"></i>{t('navPlano')}
      </button>
    {/snippet}
  </Header>

  <div style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 34%) minmax(0, 1fr);">
    <div style="min-height: 0; overflow-y: auto; padding: 20px; border-right: 1px solid var(--color-line); display: flex; flex-direction: column; gap: 10px;">
      {#if recipes.items.length === 0}
        <div style="display: flex; flex-direction: column; align-items: center; text-align: center; padding: 32px 16px; color: var(--color-neutral-500);">
          <p style="font-size: 15px; font-weight: 500; color: var(--color-text);">{t('recipesEmptyTitle')}</p>
          <p style="margin-top: 8px; font-size: 13px;">{t('recipesEmptyHint')}</p>
        </div>
      {:else}
        {#each recipes.items as r (r.id)}
          {@const res = resolved[r.id]}
          <button
            class="t3-card"
            class:active={selectedId === r.id}
            onclick={() => (selectedId = r.id)}
            style="display: flex; align-items: center; gap: 16px; padding: 16px; border-radius: 14px; text-align: left; font-family: inherit;"
          >
            <span style="display: flex; border-radius: 4px; overflow: hidden; border: 1px solid var(--color-neutral-800); flex-shrink: 0;">
              <span style="width: 30px; height: 54px; background: rgb({r.targetR}, {r.targetG}, {r.targetB});"></span>
              <span style="width: 30px; height: 54px; background: rgb({res && res !== 'error' ? res.resultR : r.targetR}, {res && res !== 'error' ? res.resultG : r.targetG}, {res && res !== 'error' ? res.resultB : r.targetB});"></span>
            </span>
            <span style="display: flex; flex-direction: column; gap: 3px; flex: 1; min-width: 0;">
              <span style="font-size: 16px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{summaryName(r)}</span>
              <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">{cardMeta(r)}</span>
            </span>
            <span class="font-mono" style="font-size: 14px; color: var(--color-neutral-400); flex-shrink: 0;">{cardDelta(r)}</span>
          </button>
        {/each}
      {/if}
    </div>

    <div style="min-height: 0; overflow-y: auto; padding: 24px;">
      {#if selected}
        <p style="margin: 0; font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{t('reproduceIn')}</p>
        <div style="display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 22px;">
          {#each allManufacturers() as m (m.id)}
            <button
              class="t3-pill"
              class:active={selected.manufacturerId === m.id}
              onclick={() => pickManufacturer(m.id)}
              style="height: 48px; padding: 0 12px; border-radius: 8px; font-family: inherit; font-size: 13.5px; font-weight: 500; flex-shrink: 0; white-space: nowrap;"
            >{m.name}</button>
          {/each}
        </div>
      {/if}

      <!-- Manchete/consequência/par renderizam sempre — fiéis à métrica do
           protótipo, que não modela um estado "sem receita" à parte. Só o
           CONTEÚDO varia: sem receita selecionada, resolvendo, erro, ou
           fórmula pronta. -->
      <h2 style="margin: 0; font-size: clamp(19px, 2.4cqi, 28px); font-weight: 500; letter-spacing: -0.02em; line-height: 1.14; color: var(--color-text); text-wrap: pretty;">
        {#if recipeResolved}
          {recipeResolved.ingredients.length <= 1 ? t('recPote', { brand: recipeResolved.targetManufacturer }) : t('recMix', { brand: recipeResolved.targetManufacturer, n: recipeResolved.ingredients.length })}
        {:else if recipeIsError}
          {t('recNone', { brand: allManufacturers().find(m => m.id === selected?.manufacturerId)?.name ?? '' })}
        {:else if selected}
          {t('calculating')}
        {:else}
          {t('recipesEmptyTitle')}
        {/if}
      </h2>
      <p style="margin: 10px 0 0; font-size: 16px; color: var(--color-neutral-400); text-wrap: pretty;">
        {recipeVerdict ? t(recipeVerdict.c) : (selected ? '' : t('recipesEmptyHint'))}
      </p>

      <div style="display: flex; align-items: center; gap: 20px; margin: 20px 0 24px; padding: 16px 18px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);">
        <span style="display: flex; border-radius: 8px; overflow: hidden; border: 1px solid var(--color-neutral-800); flex-shrink: 0;">
          <span style="width: 62px; height: 72px; background: {pairTargetBg};"></span>
          <span style="width: 62px; height: 72px; background: {pairResultBg};"></span>
        </span>
        <span style="display: flex; align-items: baseline; gap: 8px;">
          <span class="font-mono" style="font-size: clamp(28px, 3.4cqi, 40px); font-weight: 500; line-height: 1; letter-spacing: -0.03em; color: {recipeResolved && deltaIsGood(recipeResolved.deltaE) ? 'var(--color-accent-400)' : 'var(--color-text)'};">{recipeResolved ? decimal(recipeResolved.deltaE, 1) : '—'}</span>
          <span style="font-size: 14px; color: var(--color-neutral-500);">ΔE00</span>
        </span>
        <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{recipeVerdict ? t(recipeVerdict.v) : ''}</span>
      </div>

      {#if recipeResolved}
        <div style="display: flex; flex-direction: column; gap: 10px;">
          {#each recipeResolved.ingredients as ing (ing.paintId)}
            <div style="display: flex; align-items: center; gap: 16px; min-height: 84px; padding: 12px 18px; border: 1px solid var(--color-rule); border-radius: 14px; background: var(--color-panel);">
              <PaintBottle r={ing.r} g={ing.g} b={ing.b} width={40} />
              <span style="display: flex; flex-direction: column; gap: 4px; flex: 1; min-width: 0;">
                <span style="font-size: 18px; font-weight: 500; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{ing.name}</span>
                <span class="font-mono" style="font-size: 13px; color: var(--color-neutral-500);">{ing.code}</span>
              </span>
              <span style="font-size: 24px; font-weight: 500; color: var(--color-text); flex-shrink: 0;">{Math.round(ing.percentage)}</span>
              <span style="width: 54px; font-size: 12px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--color-neutral-500); flex-shrink: 0;">%</span>
            </div>
          {/each}
        </div>

        {#if otherBrands.length > 0}
          <p style="margin: 26px 0 12px; font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{t('sameOtherBrands')}</p>
          <div style="display: flex; flex-direction: column;">
            {#each otherBrands as b (b.manufacturerId)}
              <button
                class="t3-other-row"
                onclick={() => pickManufacturer(b.manufacturerId)}
                style="display: flex; align-items: center; gap: 16px; min-height: 62px; padding: 10px 4px; border: none; border-bottom: 1px solid var(--color-line); background: transparent; text-align: left; font-family: inherit;"
              >
                <span style="width: 44px; height: 34px; border-radius: 4px; border: 1px solid var(--color-neutral-800); background: rgb({b.r}, {b.g}, {b.b}); flex-shrink: 0;"></span>
                <span style="display: flex; flex-direction: column; flex: 1; min-width: 0;">
                  <span style="font-size: 16px; font-weight: 500; color: var(--color-text);">{b.manufacturer}</span>
                  <span style="font-size: 13px; color: var(--color-neutral-500); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{b.name}</span>
                </span>
                <span class="font-mono" style="font-size: 14px; color: {deltaIsGood(b.deltaE) ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; flex-shrink: 0;">ΔE {decimal(b.deltaE, 1)}</span>
              </button>
            {/each}
          </div>
        {/if}
      {/if}

      {#if selected}
        <button class="t3-delete" onclick={() => askDelete(selected!.id)} style="margin-top: 24px; min-height: 44px; font-weight: 600; font-size: 13.5px; color: var(--color-accent-400);">
          {t('del')}
        </button>
      {/if}
    </div>
  </div>
</div>

{#if confirmDeleteId != null}
  <div class="t3-scrim" role="presentation" onclick={() => (confirmDeleteId = null)}></div>
  <div class="t3-confirm" role="alertdialog" aria-modal="true">
    <p style="font-size: 16px; font-weight: 500; color: var(--color-text);">
      {t('deleteRecipeConfirm', { name: recipes.items.find(r => r.id === confirmDeleteId)?.name ?? '' })}
    </p>
    <div style="display: flex; gap: 12px; margin-top: 20px;">
      <button
        onclick={() => (confirmDeleteId = null)}
        style="flex: 1; min-height: 48px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; font-weight: 500;"
      >{t('cancel')}</button>
      <button
        onclick={confirmDelete}
        style="flex: 1; min-height: 48px; border: none; border-radius: 8px; background: var(--color-accent); color: var(--color-accent-100); font-family: inherit; font-size: 14px; font-weight: 500;"
      >{t('confirmBtn')}</button>
    </div>
  </div>
{/if}

<style>
  /* style-hover do protótipo: reproduzido aqui, por classe (regra 6). */
  .t3-actionbtn:hover {
    border-color: var(--color-accent-700);
    color: var(--color-accent-400);
  }

  .t3-card {
    border: 1px solid var(--color-rule);
    background: var(--color-panel);
  }

  .t3-card:hover {
    border-color: var(--color-accent-700);
  }

  .t3-card.active {
    border-color: var(--color-accent-700);
    background: var(--color-accent-panel);
  }

  .t3-pill {
    border: 1px solid var(--color-neutral-800);
    background: transparent;
    color: var(--color-neutral-400);
  }

  .t3-pill:hover {
    border-color: var(--color-accent-700);
  }

  .t3-pill.active {
    border-color: var(--color-accent);
    background: var(--color-accent);
    color: var(--color-accent-100);
  }

  .t3-other-row:hover {
    background: var(--color-panel);
  }

  .t3-delete:hover {
    color: var(--color-accent-300);
  }

  .t3-scrim {
    position: fixed;
    inset: 0;
    z-index: 89;
    background: rgba(0, 0, 0, 0.55);
  }

  .t3-confirm {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 90;
    width: min(420px, 90vw);
    padding: 24px;
    background: var(--color-surface);
    border-radius: 14px;
    box-shadow: var(--shadow-lg);
  }
</style>
