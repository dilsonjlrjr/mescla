<script lang="ts">
  // rf-24 — guia didático do círculo (T5): 5 cartões com amostras vivas da
  // cor-base atual, todas calculadas por color/circulo.ts. "Ver no disco" leva
  // a aba ou a harmonia de T5 ao que o cartão ensina e fecha a folha.
  import FolhaCirculo from './FolhaCirculo.svelte';
  import {
    SCHEMES,
    WHEEL,
    WHITE,
    harmonyOf,
    mixPaint,
    nearestIndex,
    rgbToHex,
    withBlack,
    withComplement,
    type RGB,
    type SchemeId,
  } from '../color/circulo';
  import { hueKey, schemeKey } from '../circuloTexto';
  import { t, type DictKey } from '../i18n.svelte';

  export type VerNoDisco = { scheme?: SchemeId; tab?: 'escurecer' | 'clarear' | 'saturacao' | 'valor' };

  interface Props {
    base: RGB;
    baseWheel: number;
    fromRing: boolean;
    /** nome do matiz mais perto da base */
    matiz: string;
    onfechar: () => void;
    onver: (v: VerNoDisco) => void;
  }

  let { base, baseWheel, fromRing, matiz, onfechar, onver }: Props = $props();

  const css = (c: RGB) => `rgb(${c.r}, ${c.g}, ${c.b})`;

  let baseHex = $derived(rgbToHex(base));

  // 3. harmonias — complementar, análoga e tríade.
  const HARMONIAS: SchemeId[] = ['complementar', 'analoga', 'triade'];
  let harmonias = $derived(
    HARMONIAS.map(id => {
      const s = SCHEMES.find(x => x.id === id)!;
      return { id, sw: harmonyOf(s, baseWheel, base, fromRing) };
    })
  );

  // 4. clarear e escurecer: preto x complementar; branco x cor vizinha.
  let complementar = $derived(harmonyOf(SCHEMES[0], baseWheel, base, fromRing)[1].rgb);
  let comBranco = $derived(mixPaint(base, WHITE, 0.4));
  let comPreto = $derived(withBlack(base, 0.6));
  let comComp = $derived(withComplement(base, complementar, 0.6));

  // 5. miniatura: luz quente no alto, sombra fria embaixo.
  let luz = $derived(mixPaint(mixPaint(base, WHITE, 0.25), WHEEL[8].rgb, 0.25));
  let sombra = $derived(withBlack(mixPaint(base, WHEEL[3].rgb, 0.3), 0.3));

  let cartoes = $derived<Array<{ t: DictKey; d: DictKey; ver: VerNoDisco }>>([
    { t: 'cgC1T', d: 'cgC1D', ver: {} },
    { t: 'cgC2T', d: 'cgC2D', ver: {} },
    { t: 'cgC3T', d: 'cgC3D', ver: { scheme: 'complementar' } },
    { t: 'cgC4T', d: 'cgC4D', ver: { tab: 'escurecer' } },
    { t: 'cgC5T', d: 'cgC5D', ver: { tab: 'valor' } },
  ]);
</script>

<FolhaCirculo titulo={t('cgTitle')} {onfechar}>
  {#snippet children(fechar)}
    <p class="gc-intro">{t('cgIntro')}</p>

    {#each cartoes as c, i (c.t)}
      <section class="gc-card">
        <h3>{i + 1}. {t(c.t)}</h3>
        <p>{t(c.d)}</p>

        <div class="gc-amostras">
          {#if i === 0}
            <div class="gc-anel" aria-hidden="true">
              {#each WHEEL as w (w.id)}
                <span class:base={w.id === WHEEL[nearestIndex(baseWheel)].id} style="background: {w.hex};"></span>
              {/each}
            </div>
            <span class="gc-legenda">{matiz} · <span class="font-mono">{baseHex}</span></span>
          {:else if i === 1}
            <div class="gc-anel" aria-hidden="true">
              {#each WHEEL as w (w.id)}
                <span class:base={w.id === WHEEL[nearestIndex(baseWheel)].id} style="background: {w.hex}; opacity: {w.warm ? 1 : 0.55};"></span>
              {/each}
            </div>
            <span class="gc-legenda">{t('cwWarm')} · {t('cwCool')}</span>
          {:else if i === 2}
            <div class="gc-linhas">
              {#each harmonias as h (h.id)}
                <div class="gc-linha">
                  <span class="gc-rot">{t(schemeKey(h.id))}</span>
                  <span class="gc-par">
                    {#each h.sw as s (s.offset)}
                      <span title={t(hueKey(s.hue.id))} style="background: {s.hex};"></span>
                    {/each}
                  </span>
                </div>
              {/each}
            </div>
          {:else if i === 3}
            <div class="gc-linhas">
              <div class="gc-linha">
                <span class="gc-rot">{t('pathPreto')}</span>
                <span class="gc-par"><span style="background: {baseHex};"></span><span style="background: {css(comPreto)};"></span></span>
              </div>
              <div class="gc-linha">
                <span class="gc-rot">{t('pathComp')}</span>
                <span class="gc-par"><span style="background: {baseHex};"></span><span style="background: {css(comComp)};"></span></span>
              </div>
              <div class="gc-linha">
                <span class="gc-rot">{t('pathBranco')}</span>
                <span class="gc-par"><span style="background: {baseHex};"></span><span style="background: {css(comBranco)};"></span></span>
              </div>
            </div>
          {:else}
            <div class="gc-linhas">
              <div class="gc-linha">
                <span class="gc-rot">{t('cgLight')}</span>
                <span class="gc-par"><span style="background: {css(luz)};"></span></span>
              </div>
              <div class="gc-linha">
                <span class="gc-rot">{matiz}</span>
                <span class="gc-par"><span style="background: {baseHex};"></span></span>
              </div>
              <div class="gc-linha">
                <span class="gc-rot">{t('cgShadow')}</span>
                <span class="gc-par"><span style="background: {css(sombra)};"></span></span>
              </div>
            </div>
          {/if}
        </div>

        <button class="pressable gc-ver" onclick={() => { onver(c.ver); fechar(); }}>
          <i class="ph ph-circle-half-tilt" style="font-size: 17px;"></i>{t('cgSee')}
        </button>
      </section>
    {/each}
  {/snippet}
</FolhaCirculo>

<style>
  .gc-intro {
    margin: 0;
    font-size: 14px;
    color: var(--color-neutral-400);
  }

  .gc-card {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px;
    border: 1px solid var(--color-rule);
    border-radius: 14px;
    background: var(--color-panel);
  }

  .gc-card h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
  }

  .gc-card p {
    margin: 0;
    font-size: 14px;
    color: var(--color-neutral-400);
    text-wrap: pretty;
  }

  .gc-amostras {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .gc-anel {
    display: flex;
    gap: 3px;
  }

  .gc-anel span {
    flex: 1;
    height: 26px;
    border-radius: 4px;
    border: 1px solid transparent;
  }

  .gc-anel span.base {
    height: 34px;
    align-self: flex-end;
    border-color: var(--color-text);
  }

  .gc-legenda {
    font-size: 13px;
    color: var(--color-neutral-300);
  }

  .gc-linhas {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .gc-linha {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .gc-rot {
    width: 42%;
    flex-shrink: 0;
    font-size: 13px;
    color: var(--color-neutral-300);
  }

  .gc-par {
    flex: 1;
    display: flex;
    height: 30px;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid var(--color-rule);
  }

  .gc-par span {
    flex: 1;
  }

  .gc-ver {
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
</style>
