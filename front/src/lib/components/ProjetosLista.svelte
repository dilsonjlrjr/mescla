<script lang="ts">
  // rf-16 — lista de projetos, a entrada de T2 (RN1–RN6). Só apresentação:
  // quem busca a lista no servidor, lê o rascunho e decide abrir/excluir é o
  // PlannerView, que é dono do estado do editor.
  import Header from './Header.svelte';
  import Spinner from './Spinner.svelte';
  import { t } from '../i18n.svelte';
  import { norm } from '../texto';
  import type { ResumoPlanoDTO } from '../services/plans';

  interface Props {
    planos: ResumoPlanoDTO[];
    carregando: boolean;
    erro: boolean;
    /** Projeto sendo aberto agora (spinner no card e lista travada). */
    abrindo: boolean;
    rascunho: { planId: number | null; nome: string } | null;
    onabrir: (p: ResumoPlanoDTO) => void;
    onabrirRascunho: () => void;
    onnovo: () => void;
    onexcluir: (p: ResumoPlanoDTO) => void;
    onrecarregar: () => void;
  }

  let { planos, carregando, erro, abrindo, rascunho, onabrir, onabrirRascunho, onnovo, onexcluir, onrecarregar }: Props =
    $props();

  let busca = $state('');

  let filtrados = $derived.by(() => {
    const q = norm(busca.trim());
    return q ? planos.filter(p => norm(p.name).includes(q)) : planos;
  });

  // RN4: rascunho de projeto que está na lista vira selo no card; o resto
  // (nunca salvo, ou salvo e depois excluído) vira card próprio no topo. Com
  // a lista em erro não dá para classificar — mostra o card do rascunho.
  let rascunhoNaLista = $derived.by(() => {
    const idDoRascunho = rascunho?.planId ?? null;
    return idDoRascunho !== null && !erro && planos.some(p => p.id === idDoRascunho);
  });
  let mostrarCardRascunho = $derived(rascunho !== null && !rascunhoNaLista && (!carregando || erro));

  function formatarData(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    const p2 = (n: number) => String(n).padStart(2, '0');
    return `${p2(d.getDate())}/${p2(d.getMonth() + 1)}/${d.getFullYear()} ${p2(d.getHours())}:${p2(d.getMinutes())}`;
  }

  function figurasTexto(n: number): string {
    return n === 1 ? t('projectFigureOne') : t('projectFigures', { n });
  }
</script>

<div style="display: flex; flex-direction: column; height: 100%; min-height: 0;">
  <Header kicker={t('navPlano')} title={t('projectsTitle')}>
    {#snippet actions()}
      <button
        class="t2l-new-btn"
        disabled={abrindo}
        onclick={onnovo}
        style="display: inline-flex; align-items: center; gap: 10px; height: 52px; padding: 0 18px; border: 1px solid var(--color-accent-700); border-radius: 8px; background: transparent; color: var(--color-accent-400); font-family: inherit; font-size: 15px; font-weight: 500; cursor: pointer;"
      >
        <i class="ph ph-plus" style="font-size: 19px;"></i>{t('newProjectBtn')}
      </button>
    {/snippet}
  </Header>

  <div style="flex: 1; min-height: 0; overflow-y: auto; padding: 20px;">
    <label for="t2-projetos-busca" style="position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap;">{t('projectsSearchPh')}</label>
    <div style="position: relative; max-width: 520px; margin-bottom: 18px;">
      <i class="ph ph-magnifying-glass" style="position: absolute; left: 14px; top: 50%; transform: translateY(-50%); font-size: 18px; color: var(--color-neutral-500);"></i>
      <input
        id="t2-projetos-busca"
        type="search"
        bind:value={busca}
        placeholder={t('projectsSearchPh')}
        style="width: 100%; height: 48px; padding: 0 14px 0 42px; border: 1px solid var(--color-field-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-family: inherit; font-size: 15px;"
      />
    </div>

    {#if erro}
      <div role="alert" style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px; padding: 12px 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: var(--color-raised); font-size: 14px; color: var(--color-neutral-300);">
        <i class="ph ph-warning-circle" style="font-size: 18px;"></i>
        <span style="flex: 1;">{t('projectsLoadError')}</span>
        <button
          onclick={onrecarregar}
          style="min-height: 44px; padding: 0 14px; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-300); font-family: inherit; font-size: 13.5px; cursor: pointer;"
        >{t('retryBtn')}</button>
      </div>
    {/if}

    {#if carregando && planos.length === 0 && !erro}
      <div style="display: flex; align-items: center; justify-content: center; gap: 12px; height: 120px; color: var(--color-neutral-500);">
        <Spinner size={24} label={t('calculating')} />
      </div>
    {/if}

    <div class="t2l-grid">
      {#if mostrarCardRascunho && rascunho}
        <div class="t2l-card" style="border-color: var(--color-accent-700); background: var(--color-accent-panel);">
          <button class="t2l-open" disabled={abrindo} onclick={onabrirRascunho}>
            <span style="font-size: 12px; letter-spacing: 0.1em; text-transform: uppercase; color: var(--color-accent-400);">{t('projectNewUnsaved')}</span>
            <span class="t2l-name">{rascunho.nome.trim() || t('projectNoName')}</span>
          </button>
        </div>
      {/if}

      {#each filtrados as p (p.id)}
        {@const comRascunho = rascunhoNaLista && rascunho?.planId === p.id}
        <div class="t2l-card">
          <button class="t2l-open" disabled={abrindo} onclick={() => (comRascunho ? onabrirRascunho() : onabrir(p))} aria-label={t('openProjectAria', { name: p.name })}>
            {#if comRascunho}
              <span style="align-self: flex-start; padding: 2px 8px; border: 1px solid var(--color-accent-700); border-radius: 999px; font-size: 11.5px; color: var(--color-accent-400);">{t('projectUnsavedBadge')}</span>
            {/if}
            <span class="t2l-name">{p.name}</span>
            <span style="font-size: 13.5px; color: var(--color-neutral-400);">{figurasTexto(p.tabCount)} · {t('projectPaintedProgress', { a: p.paintedCount, b: p.regionCount })}</span>
            <span class="font-mono" style="font-size: 12.5px; color: var(--color-neutral-500);">{t('projectUpdatedAt', { data: formatarData(p.updatedAt) })}</span>
          </button>
          <button
            class="t2l-del"
            disabled={abrindo}
            aria-label={t('deleteProjectBtn')}
            title={t('deleteProjectBtn')}
            onclick={() => onexcluir(p)}
          ><i class="ph ph-trash-simple" style="font-size: 17px;"></i></button>
        </div>
      {/each}
    </div>

    {#if !carregando && !erro && planos.length === 0 && !mostrarCardRascunho}
      <p style="margin: 24px 0 0; font-size: 15px; color: var(--color-neutral-500);">{t('projectsEmpty')}</p>
    {:else if !carregando && planos.length > 0 && filtrados.length === 0}
      <p style="margin: 24px 0 0; font-size: 15px; color: var(--color-neutral-500);">{t('projectsNoMatch')}</p>
    {/if}
  </div>
</div>

<style>
  .t2l-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: 14px;
  }

  .t2l-card {
    position: relative;
    display: flex;
    border: 1px solid var(--color-neutral-800);
    border-radius: 14px;
    background: var(--color-panel);
  }

  .t2l-card:hover {
    border-color: var(--color-accent-700);
  }

  .t2l-open {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
    min-height: 96px;
    padding: 16px;
    border: none;
    background: transparent;
    color: inherit;
    font-family: inherit;
    text-align: left;
    cursor: pointer;
  }

  .t2l-name {
    max-width: 100%;
    font-size: 17px;
    font-weight: 500;
    color: var(--color-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .t2l-del {
    flex-shrink: 0;
    align-self: flex-start;
    width: 44px;
    height: 44px;
    margin: 8px 8px 0 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--color-neutral-500);
    cursor: pointer;
  }

  .t2l-del:hover {
    color: var(--color-neutral-300);
  }

  .t2l-new-btn:hover {
    background: var(--color-accent-hover);
  }

  button:disabled {
    opacity: 0.5;
    pointer-events: none;
  }
</style>
