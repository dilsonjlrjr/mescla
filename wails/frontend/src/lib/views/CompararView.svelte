<script lang="ts">
  // rf-23 — T6 Comparar. Uma tinta âncora contra até 6 candidatas: cada uma
  // aparece numa amostra dividida (metade âncora, metade candidata) com ΔE00 e
  // veredito, da mais próxima à mais distante. O cálculo vem do servidor
  // (`/compare-to-anchor`, que ordena por ΔE); o veredito reaproveita a escala
  // de T1/T3 (`verdictKeys`) e, com ΔE ≥ 4, soma a direção (mais clara/escura;
  // acima de 8, o matiz da roda do círculo cromático).
  import Header from '../components/Header.svelte';
  import Combobox from '../components/Combobox.svelte';
  import Spinner from '../components/Spinner.svelte';
  import { allPaints, paintById, type Paint } from '../services/catalog';
  import { compareToAnchor, type SearchResult } from '../services/engine';
  import { catalogRev } from '../services/catalogRev.svelte';
  import { appState } from '../appState.svelte';
  import { WHEEL, nearestIndex, rgbToLab, wheelDegOf } from '../color/circulo';
  import { hueKey } from '../circuloTexto';
  import { contrastOn, verdictKeys } from '../ui';
  import { t, decimal } from '../i18n.svelte';
  import { toast } from '../toast.svelte';

  const MAX_CANDIDATAS = 6;
  const STORE = 'mescla-comparar';

  // Só conveniência: falha de armazenamento nunca bloqueia a tela.
  function restaurar(): { anchor: number | null; cands: number[] } {
    try {
      const raw = JSON.parse(localStorage.getItem(STORE) ?? 'null');
      if (raw && typeof raw === 'object') {
        // Id que sumiu do catálogo é descartado sem aviso.
        const existe = (id: unknown): id is number => typeof id === 'number' && paintById(id) !== undefined;
        const anchor = existe(raw.anchor) ? raw.anchor : null;
        const cands = Array.isArray(raw.cands)
          ? [...new Set((raw.cands as unknown[]).filter(existe))].filter(id => id !== anchor).slice(0, MAX_CANDIDATAS)
          : [];
        return { anchor, cands };
      }
    } catch {
      // armazenamento indisponível: começa vazio
    }
    return { anchor: null, cands: [] };
  }

  const salvo = restaurar();
  let anchorId: number | null = $state(salvo.anchor);
  let candIds: number[] = $state(salvo.cands);
  let trocando = $state(false);
  let carregando = $state(false);
  let resultados: SearchResult[] = $state([]);

  // Candidata tirada some já, sem esperar a nova resposta do servidor.
  let visiveis = $derived(resultados.filter(r => candIds.includes(r.paintId)));

  let anchor: Paint | undefined = $derived(anchorId !== null ? paintById(anchorId) : undefined);

  let opcoes = $derived.by(() => {
    void catalogRev.n;
    return allPaints().map(p => ({
      id: p.id,
      name: p.name,
      hint: `${p.code} · ${p.manufacturer}`,
      searchText: `${p.name} ${p.code} ${p.manufacturer}`,
    }));
  });

  // Âncora vinda de fora (T4 ou paleta).
  $effect(() => {
    const id = appState.pendingCompareAnchor;
    if (id == null) return;
    appState.pendingCompareAnchor = null;
    if (paintById(id)) definirAncora(id);
  });

  $effect(() => {
    try {
      localStorage.setItem(STORE, JSON.stringify({ anchor: anchorId, cands: candIds }));
    } catch {
      // conveniência
    }
  });

  let ultimaBusca = 0;
  $effect(() => {
    const a = anchorId;
    const ids = [...candIds];
    const minha = ++ultimaBusca;
    if (a === null || ids.length === 0) {
      resultados = [];
      carregando = false;
      return;
    }
    carregando = true;
    compareToAnchor(a, ids)
      .then(r => {
        if (minha === ultimaBusca) resultados = r;
      })
      .catch(() => {
        if (minha === ultimaBusca) toast(t('cmpFailed'), 'error');
      })
      .finally(() => {
        if (minha === ultimaBusca) carregando = false;
      });
  });

  function definirAncora(id: number) {
    anchorId = id;
    candIds = candIds.filter(c => c !== id);
    trocando = false;
  }

  function adicionar(id: number | null) {
    if (id === null || anchorId === null) return;
    if (id === anchorId || candIds.includes(id)) {
      toast(t('cmpDup'), 'error');
      return;
    }
    if (candIds.length >= MAX_CANDIDATAS) {
      toast(t('cmpMaxMsg', { max: MAX_CANDIDATAS }), 'error');
      return;
    }
    candIds = [...candIds, id];
  }

  function remover(id: number) {
    candIds = candIds.filter(c => c !== id);
  }

  // RN8: escala de T1/T3 + direção quando a diferença já se vê.
  function veredito(r: SearchResult): string {
    const partes = [t(verdictKeys(r.deltaE).v)];
    if (anchor && r.deltaE >= 4) {
      const dL = rgbToLab({ r: r.r, g: r.g, b: r.b }).L - rgbToLab({ r: anchor.r, g: anchor.g, b: anchor.b }).L;
      const dir: string[] = [];
      if (Math.abs(dL) > 3) dir.push(t(dL > 0 ? 'cmpLighter' : 'cmpDarker'));
      if (r.deltaE >= 8) {
        const deg = wheelDegOf({ r: r.r, g: r.g, b: r.b });
        if (deg !== null) {
          dir.push(t('cmpPulls', { hue: t(hueKey(WHEEL[nearestIndex(deg)].id)).toLocaleLowerCase() }));
        }
      }
      if (dir.length > 0) partes.push(dir.join(', '));
    }
    return partes.join(' · ');
  }

  function cor(p: { r: number; g: number; b: number }): string {
    return `rgb(${p.r}, ${p.g}, ${p.b})`;
  }
</script>

<div style="display: flex; flex-direction: column; height: 100%;">
  <Header kicker={t('navComparar')} title={anchor ? anchor.name : t('cmpTitleNone')} />

  <div style="flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 34%) minmax(0, 1fr);">
    <!-- Âncora: cor chapada -->
    <div
      style="display: flex; flex-direction: column; justify-content: space-between; gap: 16px; padding: 24px; border-right: 1px solid var(--color-line); min-height: 0; overflow-y: auto; background: {anchor ? cor(anchor) : 'var(--color-raised)'}; color: {anchor ? contrastOn(anchor.r, anchor.g, anchor.b) : 'var(--color-text)'};"
    >
      {#if anchor && !trocando}
        <div style="display: flex; flex-direction: column; gap: 6px; min-width: 0;">
          <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; opacity: 0.75;">{t('cmpAnchorLbl')}</span>
          <span style="font-size: clamp(20px, 2.4cqi, 28px); font-weight: 500; letter-spacing: -0.01em; text-wrap: balance;">{anchor.name}</span>
          <span class="font-mono" style="font-size: 13px; opacity: 0.85;">{anchor.code}</span>
          <span style="font-size: 14px; opacity: 0.85;">{anchor.manufacturer}</span>
        </div>
        <button
          class="pressable"
          onclick={() => (trocando = true)}
          style="align-self: flex-start; height: 48px; padding: 0 18px; border: 1px solid currentColor; border-radius: 8px; background: transparent; color: inherit; font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
        >{t('cmpChange')}</button>
      {:else}
        <div style="display: flex; flex-direction: column; gap: 12px; color: var(--color-text);">
          <span style="font-size: clamp(18px, 2cqi, 24px); font-weight: 500; letter-spacing: -0.01em;">{t('cmpPickAnchor')}</span>
          <span style="font-size: 14px; color: var(--color-neutral-400); text-wrap: pretty;">{t('cmpPickAnchorN')}</span>
          <Combobox
            options={opcoes}
            value={null}
            onchange={id => id !== null && definirAncora(id)}
            placeholder={t('cmpPickAnchor')}
            searchPlaceholder={t('cmpAnchorPh')}
            emptyText={t('palNone')}
            ariaLabel={t('cmpPickAnchor')}
          />
          {#if anchor}
            <button
              class="pressable"
              onclick={() => (trocando = false)}
              style="align-self: flex-start; height: 44px; padding: 0 16px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); font-family: inherit; font-size: 14px; cursor: pointer;"
            >{t('cancel')}</button>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Candidatas -->
    <div style="display: flex; flex-direction: column; min-height: 0; min-width: 0;">
      <div style="flex: 1; min-height: 0; overflow-y: auto;">
        <div style="max-width: 720px; margin: 0 auto; padding: 22px 24px 40px; display: flex; flex-direction: column; gap: 14px;">
          {#if anchor}
            <div style="display: flex; align-items: center; gap: 12px; flex-wrap: wrap;">
              <span style="font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{t('cmpCandT', { n: candIds.length, max: MAX_CANDIDATAS })}</span>
              {#if carregando}<Spinner size={18} label={t('cmpCandT', { n: candIds.length, max: MAX_CANDIDATAS })} />{/if}
            </div>
            <Combobox
              options={opcoes}
              value={null}
              onchange={adicionar}
              placeholder={t('cmpAddPh')}
              searchPlaceholder={t('cmpAddPh')}
              emptyText={t('palNone')}
              ariaLabel={t('cmpAddPh')}
            />

            {#if candIds.length === 0}
              <p style="margin: 8px 0 0; font-size: 15px; color: var(--color-neutral-500); text-wrap: pretty;">{t('cmpEmptyCand', { max: MAX_CANDIDATAS })}</p>
            {/if}

            {#each visiveis as r (r.paintId)}
              {@const cand = paintById(r.paintId)}
              <div style="display: flex; flex-direction: column; gap: 10px; padding: 14px; border: 1px solid var(--color-neutral-800); border-radius: 12px; background: var(--color-raised);">
                <div style="display: flex; height: 72px; border-radius: 8px; overflow: hidden; border: 1px solid var(--color-line);" aria-hidden="true">
                  <div style="flex: 1; background: {cor(anchor)};"></div>
                  <div style="flex: 1; background: {cor(r)};"></div>
                </div>
                <div style="display: flex; align-items: flex-start; gap: 12px;">
                  <span style="display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0;">
                    <span style="font-size: 16px; font-weight: 500; color: var(--color-text); overflow-wrap: anywhere;">{r.name}</span>
                    <span style="font-size: 13px; color: var(--color-neutral-500);">{cand?.code ? cand.code + ' · ' : ''}{r.manufacturer}</span>
                  </span>
                  <span class="font-mono" style="flex-shrink: 0; font-size: 18px; color: var(--color-text);">ΔE00 {decimal(r.deltaE)}</span>
                  <button
                    class="pressable"
                    onclick={() => remover(r.paintId)}
                    aria-label={t('cmpRemove')}
                    title={t('cmpRemove')}
                    style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; flex-shrink: 0; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;"
                  ><i class="ph ph-x" style="font-size: 18px;"></i></button>
                </div>
                <span style="font-size: 14px; color: var(--color-neutral-300); text-wrap: pretty;">{veredito(r)}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
