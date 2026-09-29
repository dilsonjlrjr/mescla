<script lang="ts">
  // rf-23 — paleta de comandos (Ctrl+K / ⌘K, ou o botão do cabeçalho). Montada
  // uma vez em App.svelte; abre em qualquer tela. Busca tinta por nome, código
  // ou hex (com ΔE00 local contra o catálogo, que já está na memória), leva a
  // qualquer tela e oferece três ações sobre a tinta destacada.
  //
  // Entra como camada de navegação (`pushLayer`): o "voltar" do Android a fecha.
  // Uma ação que troca de tela espera a camada sair (popstate) antes de
  // navegar — `switchTab` no meio do `history.back()` mandaria o app para T1.
  import { onDestroy, tick, untrack } from 'svelte';
  import { allPaints, searchPaints, type Paint } from '../services/catalog';
  import { deltaE00, hexToRgb } from '../color/circulo';
  import { switchTab, pushLayer, type Tab } from '../nav.svelte';
  import { appState } from '../appState.svelte';
  import { t, decimal } from '../i18n.svelte';

  interface Linha {
    tipo: 'nav' | 'tinta' | 'acao';
    id: string;
    rotulo: string;
    icone?: string;
    tinta?: Paint;
    delta?: number;
    executar: () => void;
  }

  let consulta = $state('');
  let consultaAtiva = $state('');
  let sel = $state(0);
  let alvoId: number | null = $state(null);
  let inputEl: HTMLInputElement | undefined = $state();
  let dialogEl: HTMLDivElement | undefined = $state();
  let listaEl: HTMLDivElement | undefined = $state();

  let layerAtiva = false;
  let fecharLayer: (() => void) | null = null;
  let origem: HTMLElement | null = null;
  let depois: (() => void) | null = null;
  let restaurarFoco = true;
  let espera: ReturnType<typeof setTimeout> | undefined;
  let estavaAberta = false;

  const HEX = /^#?[0-9a-f]{6}$/i;

  function dobrar(s: string): string {
    return s.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
  }

  // ── abrir / fechar ──
  function abrir() {
    if (layerAtiva) return;
    estavaAberta = true;
    restaurarFoco = true;
    origem = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    consulta = '';
    consultaAtiva = '';
    sel = 0;
    alvoId = null;
    fecharLayer = pushLayer(() => {
      layerAtiva = false;
      appState.paletaAberta = false;
      const f = depois;
      depois = null;
      f?.();
    });
    layerAtiva = true;
    void tick().then(() => inputEl?.focus());
  }

  function aoFechar() {
    clearTimeout(espera);
    if (!estavaAberta) return;
    estavaAberta = false;
    if (layerAtiva) {
      layerAtiva = false;
      fecharLayer?.();
    }
    const o = origem;
    origem = null;
    if (restaurarFoco && o?.isConnected) o.focus();
  }

  $effect(() => {
    if (appState.paletaAberta) untrack(abrir);
    else untrack(aoFechar);
  });

  onDestroy(() => clearTimeout(espera));

  /** Fecha; `seguinte` roda depois da camada sair. */
  function fechar(seguinte?: () => void) {
    if (seguinte) restaurarFoco = false;
    depois = seguinte ?? null;
    if (layerAtiva && fecharLayer) {
      fecharLayer();
    } else {
      appState.paletaAberta = false;
      const f = depois;
      depois = null;
      f?.();
    }
  }

  // ── atalho global ──
  function aoTeclar(e: KeyboardEvent) {
    if (e.isComposing) return;
    // Só Ctrl/⌘+K: no navegador esse atalho foca a barra de endereço.
    if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      if (appState.paletaAberta) fechar();
      else appState.paletaAberta = true;
    }
  }

  // ── consulta ──
  function aoDigitar() {
    clearTimeout(espera);
    espera = setTimeout(() => {
      consultaAtiva = consulta;
      sel = 0;
      alvoId = null;
    }, 120);
  }

  let telas = $derived.by(() => {
    const irPara = (tab: Tab, antes?: () => void) => () => fechar(() => { antes?.(); switchTab(tab); });
    const lista: Array<{ rotulo: string; icone: string; ir: () => void }> = [
      { rotulo: t('question'), icone: 'ph-chat-circle-text', ir: irPara('pergunta') },
      { rotulo: t('navPlano'), icone: 'ph-crosshair', ir: irPara('plano') },
      { rotulo: t('navReceitas'), icone: 'ph-flask', ir: irPara('receitas') },
      { rotulo: t('navTintas'), icone: 'ph-paint-bucket', ir: irPara('estante') },
      { rotulo: t('makers'), icone: 'ph-factory', ir: irPara('estante', () => (appState.pendingCatalogTab = 'fabricantes')) },
      { rotulo: t('paintTypes'), icone: 'ph-tag', ir: irPara('estante', () => (appState.pendingCatalogTab = 'tipos')) },
      { rotulo: t('navCirculo'), icone: 'ph-circle-half-tilt', ir: irPara('circulo') },
      { rotulo: t('navComparar'), icone: 'ph-columns', ir: irPara('comparar') },
      { rotulo: t('palActGuia'), icone: 'ph-question', ir: () => fechar(() => { appState.guiaAberta = true; }) },
    ];
    const q = dobrar(consultaAtiva.trim());
    return q.length < 2 ? lista : lista.filter(l => dobrar(l.rotulo).includes(q));
  });

  let tintas = $derived.by((): Array<{ tinta: Paint; delta?: number }> => {
    const q = consultaAtiva.trim();
    if (q.length < 2) return [];
    if (HEX.test(q)) {
      const alvo = hexToRgb(q);
      if (!alvo) return [];
      return allPaints()
        .map(p => ({ tinta: p, delta: deltaE00(alvo, { r: p.r, g: p.g, b: p.b }) }))
        .sort((a, b) => a.delta - b.delta)
        .slice(0, 6);
    }
    return searchPaints(q, { limit: 6 }).map(p => ({ tinta: p }));
  });

  let alvo = $derived(tintas.find(x => x.tinta.id === alvoId)?.tinta ?? tintas[0]?.tinta);

  let linhas = $derived.by((): Linha[] => {
    const out: Linha[] = telas.map((l, i) => ({
      tipo: 'nav', id: `nav-${i}`, rotulo: l.rotulo, icone: l.icone, executar: l.ir,
    }));
    for (const x of tintas) {
      out.push({
        tipo: 'tinta', id: `tinta-${x.tinta.id}`, rotulo: x.tinta.name, tinta: x.tinta, delta: x.delta,
        executar: () => gerarFormula(x.tinta),
      });
    }
    if (alvo) {
      const p = alvo;
      out.push(
        { tipo: 'acao', id: 'acao-mix', rotulo: t('palActMix'), icone: 'ph-flask', executar: () => gerarFormula(p) },
        {
          tipo: 'acao', id: 'acao-estoque', rotulo: t('palActStock'), icone: 'ph-plus',
          executar: () => fechar(() => { appState.pendingStockPrefill = p; switchTab('estante'); }),
        },
        {
          tipo: 'acao', id: 'acao-comparar', rotulo: t('palActCmp'), icone: 'ph-columns',
          executar: () => fechar(() => { appState.pendingCompareAnchor = p.id; switchTab('comparar'); }),
        },
      );
    }
    return out;
  });

  function gerarFormula(p: Paint) {
    fechar(() => { appState.pendingTargetPaint = p; switchTab('pergunta'); });
  }

  let indiceTintas = $derived(linhas.findIndex(l => l.tipo === 'tinta'));
  let indiceAcoes = $derived(linhas.findIndex(l => l.tipo === 'acao'));
  let indiceSel = $derived(Math.min(sel, Math.max(0, linhas.length - 1)));

  function selecionar(i: number, rolar = false) {
    sel = i;
    const l = linhas[i];
    if (l?.tipo === 'tinta' && l.tinta) alvoId = l.tinta.id;
    if (rolar) {
      void tick().then(() => listaEl?.querySelector(`[data-i="${i}"]`)?.scrollIntoView({ block: 'nearest' }));
    }
  }

  function aoTeclarNoDialogo(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      fechar();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (linhas.length) selecionar((indiceSel + 1) % linhas.length, true);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (linhas.length) selecionar((indiceSel - 1 + linhas.length) % linhas.length, true);
    } else if (e.key === 'Enter') {
      if (e.isComposing) return;
      e.preventDefault();
      // Digitou e apertou Enter antes da espera de 120ms: usa o que está na tela.
      linhas[indiceSel]?.executar();
    } else if (e.key === 'Tab' && dialogEl) {
      // Foco preso na paleta.
      const focaveis = [...dialogEl.querySelectorAll<HTMLElement>('input, button')].filter(el => !el.hasAttribute('disabled'));
      if (focaveis.length === 0) return;
      const primeiro = focaveis[0];
      const ultimo = focaveis[focaveis.length - 1];
      if (e.shiftKey && document.activeElement === primeiro) {
        e.preventDefault();
        ultimo.focus();
      } else if (!e.shiftKey && document.activeElement === ultimo) {
        e.preventDefault();
        primeiro.focus();
      }
    }
  }

  function cabecalho(i: number): string {
    const l = linhas[i];
    if (l.tipo === 'nav' && i === 0) return t('palGo');
    if (l.tipo === 'tinta' && i === indiceTintas) return t('palPaints');
    if (l.tipo === 'acao' && i === indiceAcoes) return t('palActs', { name: alvo?.name ?? '' });
    return '';
  }
</script>

<svelte:window onkeydown={aoTeclar} />

{#if appState.paletaAberta}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    style="position: fixed; inset: 0; z-index: 200; display: flex; align-items: flex-start; justify-content: center; padding: 10dvh 16px 16px; background: rgba(9, 10, 16, 0.7);"
    onclick={(e) => { if (e.target === e.currentTarget) fechar(); }}
  >
    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-label={t('palTitle')}
      tabindex="-1"
      onkeydown={aoTeclarNoDialogo}
      style="width: min(640px, 100%); max-height: 100%; display: flex; flex-direction: column; overflow: hidden; border: 1px solid var(--color-neutral-800); border-radius: 14px; background: var(--color-modal); box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55); color: var(--color-text); font-family: var(--font-body); user-select: none;"
    >
      <div style="display: flex; align-items: center; gap: 10px; padding: 10px 12px 10px 16px; border-bottom: 1px solid var(--color-line);">
        <i class="ph ph-magnifying-glass" style="font-size: 20px; color: var(--color-neutral-500);"></i>
        <input
          bind:this={inputEl}
          bind:value={consulta}
          oninput={aoDigitar}
          type="text"
          role="combobox"
          aria-expanded="true"
          aria-controls="paleta-lista"
          aria-activedescendant={linhas[indiceSel] ? `paleta-${linhas[indiceSel].id}` : undefined}
          aria-label={t('palPh')}
          placeholder={t('palPh')}
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          style="flex: 1; min-width: 0; height: 48px; background: transparent; border: none; outline: none; color: var(--color-text); font-family: inherit; font-size: 16px; user-select: text;"
        />
        <button
          class="pressable"
          onclick={() => fechar()}
          aria-label={t('ariaClose')}
          style="display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; flex-shrink: 0; border: 1px solid var(--color-neutral-800); border-radius: 8px; background: transparent; color: var(--color-neutral-400); cursor: pointer;"
        ><i class="ph ph-x" style="font-size: 18px;"></i></button>
      </div>

      <div bind:this={listaEl} id="paleta-lista" role="listbox" aria-label={t('palTitle')} style="flex: 1; min-height: 0; overflow-y: auto; padding: 6px 8px 8px;">
        {#each linhas as l, i (l.id)}
          {@const titulo = cabecalho(i)}
          {#if titulo}
            <div role="presentation" style="padding: 10px 8px 4px; font-size: 12px; font-weight: 500; letter-spacing: 0.12em; text-transform: uppercase; color: var(--color-neutral-500);">{titulo}</div>
          {/if}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div
            id="paleta-{l.id}"
            data-i={i}
            role="option"
            tabindex="-1"
            aria-selected={i === indiceSel}
            onclick={() => l.executar()}
            onmousemove={() => { if (i !== indiceSel) selecionar(i); }}
            style="display: flex; align-items: center; gap: 12px; min-height: 48px; padding: 6px 10px; border-radius: 8px; cursor: pointer; background: {i === indiceSel ? 'var(--color-accent-panel)' : 'transparent'};"
          >
            {#if l.tipo === 'tinta' && l.tinta}
              <span style="width: 28px; height: 28px; flex-shrink: 0; border-radius: 6px; border: 1px solid var(--color-line); background: rgb({l.tinta.r}, {l.tinta.g}, {l.tinta.b});" aria-hidden="true"></span>
              <span style="display: flex; flex-direction: column; gap: 1px; flex: 1; min-width: 0;">
                <span style="font-size: 15px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{l.tinta.name}</span>
                <span class="font-mono" style="font-size: 12px; color: var(--color-neutral-500); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{l.tinta.code} · {l.tinta.manufacturer}{l.tinta.ignoreInMix ? ` · ${t('palOut')}` : ''}</span>
              </span>
              {#if l.delta !== undefined}
                <span class="font-mono" style="flex-shrink: 0; font-size: 13px; color: var(--color-neutral-300);">ΔE {decimal(l.delta)}</span>
              {/if}
            {:else}
              <i class="ph {l.icone}" style="font-size: 20px; color: {l.tipo === 'acao' ? 'var(--color-accent-400)' : 'var(--color-neutral-400)'}; flex-shrink: 0;"></i>
              <span style="flex: 1; min-width: 0; font-size: 15px;">{l.rotulo}</span>
            {/if}
          </div>
        {/each}
        {#if consultaAtiva.trim().length >= 2 && tintas.length === 0 && telas.length === 0}
          <p style="margin: 0; padding: 16px 10px; font-size: 14px; color: var(--color-neutral-500);">{t('palNone')}</p>
        {/if}
      </div>

      <div style="padding: 8px 16px; border-top: 1px solid var(--color-line); font-size: 12px; color: var(--color-neutral-500);">{t('palHint')}</div>
    </div>
  </div>
{/if}
