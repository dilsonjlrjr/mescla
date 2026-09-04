<script lang="ts">
  // Guia didático da Roda (mobile). Visuais gerados pela mesma lógica de cor
  // (theory.ts) e ligados à cor escolhida pelo usuário.
  import Icon from './Icon.svelte';
  import { rgbToHsl, buildRamp, harmony, type RGB } from '../color/theory';

  interface Props {
    baseRgb: RGB;
  }
  let { baseRgb }: Props = $props();

  let open = $state(false);

  let hsl = $derived(rgbToHsl(baseRgb));
  let ramp = $derived(buildRamp(baseRgb, { highlights: 3, shadows: 2 }));
  let comp = $derived(harmony(hsl, 'complementary'));
  let analog = $derived(harmony(hsl, 'analogous'));
  let triad = $derived(harmony(hsl, 'triad'));

  const wheelBg = (() => {
    const stops: string[] = [];
    for (let h = 0; h <= 360; h += 20) stops.push(`hsl(${h} 92% 55%) ${(h / 360) * 100}%`);
    return `conic-gradient(from 0deg, ${stops.join(', ')})`;
  })();
</script>

<div class="guide">
  <button class="guide-toggle pressable" onclick={() => (open = !open)} aria-expanded={open}>
    <span class="guide-icon"><Icon name="book" size={17} /></span>
    <span class="guide-labels">
      <span class="guide-title font-display">Como usar a roda</span>
      <span class="guide-hint">Guia rápido pra sair da tentativa e erro</span>
    </span>
    <span class="guide-chevron" class:flipped={open}><Icon name="chevron-down" size={18} /></span>
  </button>

  {#if open}
    <div class="guide-body">
      <div class="card">
        <div class="card-head">
          <span class="mini-wheel" style="background: {wheelBg};"></span>
          <h4>O matiz vira um círculo</h4>
        </div>
        <p>
          As cores vão do vermelho ao roxo e voltam. As <strong>primárias</strong> (vermelho, amarelo,
          azul) misturam pra formar todas. Quem está <em>lado a lado</em> se parece; quem está
          <em>de frente</em> se completa.
        </p>
      </div>

      <div class="card">
        <div class="card-head">
          <span class="temp-bar">
            <span class="temp-warm">quente</span><span class="temp-cool">fria</span>
          </span>
          <h4>Metade quente, metade fria</h4>
        </div>
        <p>
          Vermelho, laranja e amarelo <strong>avançam</strong> (quentes); verde, azul e roxo
          <strong>recuam</strong> (frias). Guarde isso: luz costuma ser quente, sombra costuma ser fria.
        </p>
      </div>

      <div class="card">
        <h4>As três harmonias</h4>
        <div class="harmony-rows">
          {#each [{ label: 'Complementar', sw: comp, note: 'opostas → contraste que vibra' }, { label: 'Análogas', sw: analog, note: 'vizinhas → combinam fácil' }, { label: 'Tríade', sw: triad, note: '3 espaçadas → equilíbrio vivo' }] as row}
            <div class="harmony-row">
              <span class="harmony-chips">
                {#each row.sw as s}<span class="chip" style="background: {s.hex};"></span>{/each}
              </span>
              <span class="harmony-text">
                <strong>{row.label}</strong>
                <span class="harmony-note">{row.note}</span>
              </span>
            </div>
          {/each}
        </div>
      </div>

      <div class="card highlight-card">
        <h4>Clarear e escurecer sem "lama"</h4>
        <p>
          Branco e preto puros <strong>matam</strong> a cor. Gire o matiz: pra iluminar caminhe pro
          <em>amarelo</em> e suba o tom; pra sombrear caminhe pro <em>azul</em> e desça.
        </p>
        <div class="shift-swatches">
          {#each ramp as step}
            <span class="shift-sw" class:base={step.kind === 'base'} style="background: {step.hex};"></span>
          {/each}
        </div>
        <div class="shift-labels">
          <span class="shift-shadow">← sombra (azul)</span>
          <span class="shift-light">luz (amarelo) →</span>
        </div>
      </div>

      <div class="card">
        <h4>Na hora de pintar</h4>
        <ol class="steps">
          <li>Pinte a peça toda com a <strong>cor base</strong> (camadas finas).</li>
          <li>Nas <strong>dobras e fundos</strong>, a sombra (tom que puxa pro azul).</li>
          <li>Nas <strong>saliências</strong> que pegam luz, o realce (tom que puxa pro amarelo).</li>
          <li>Transição <strong>gradual</strong> = volume natural.</li>
        </ol>
      </div>
    </div>
  {/if}
</div>

<style>
  .guide {
    margin: 12px 16px 4px;
    border: 1px solid var(--ink-700);
    border-radius: 14px;
    overflow: hidden;
    background: var(--ink-900);
  }
  .guide-toggle {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 14px;
    background: transparent;
    text-align: left;
  }
  .guide-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--lacquer) 14%, transparent);
    color: var(--lacquer);
    flex-shrink: 0;
  }
  .guide-labels {
    flex: 1;
    min-width: 0;
  }
  .guide-title {
    display: block;
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
  }
  .guide-hint {
    display: block;
    font-size: 12px;
    color: var(--ink-500);
  }
  .guide-chevron {
    display: flex;
    color: var(--ink-500);
    transition: transform 0.22s ease;
  }
  .guide-chevron.flipped {
    transform: rotate(180deg);
  }

  .guide-body {
    padding: 0 14px 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .card {
    padding: 14px;
    border: 1px solid var(--ink-800);
    border-radius: 12px;
    background: var(--ink-950);
  }
  .highlight-card {
    border-color: color-mix(in srgb, var(--lacquer) 35%, var(--ink-700));
    background: color-mix(in srgb, var(--lacquer) 6%, var(--ink-950));
  }
  .card-head {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }
  .card h4 {
    font-size: 15px;
    font-weight: 600;
    color: var(--ink-100);
    margin-bottom: 8px;
  }
  .card-head h4 {
    margin-bottom: 0;
  }
  .card p {
    font-size: 13.5px;
    line-height: 1.5;
    color: var(--ink-300);
  }
  .card strong {
    color: var(--ink-100);
    font-weight: 600;
  }
  .card em {
    font-style: normal;
    color: var(--lacquer);
    font-weight: 600;
  }

  .mini-wheel {
    width: 46px;
    height: 46px;
    border-radius: 50%;
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }

  .temp-bar {
    display: flex;
    width: 76px;
    height: 46px;
    border-radius: 9px;
    overflow: hidden;
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .temp-warm,
  .temp-cool {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    flex: 1;
    padding-bottom: 4px;
    font-size: 8.5px;
    font-weight: 700;
    color: rgba(255, 255, 255, 0.95);
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);
  }
  .temp-warm {
    background: linear-gradient(160deg, #e8542c, #f0a020);
  }
  .temp-cool {
    background: linear-gradient(160deg, #2f7d92, #3a5bd0);
  }

  .harmony-rows {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .harmony-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .harmony-chips {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }
  .chip {
    width: 26px;
    height: 26px;
    border-radius: 7px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .harmony-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .harmony-text strong {
    font-size: 13.5px;
    color: var(--ink-100);
  }
  .harmony-note {
    font-size: 12.5px;
    color: var(--ink-500);
  }

  .shift-swatches {
    display: flex;
    gap: 4px;
    margin-top: 12px;
  }
  .shift-sw {
    flex: 1;
    height: 32px;
    border-radius: 7px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .shift-sw.base {
    box-shadow: inset 0 0 0 2px var(--lacquer);
  }
  .shift-labels {
    display: flex;
    justify-content: space-between;
    margin-top: 6px;
    font-size: 11px;
    font-weight: 600;
  }
  .shift-shadow {
    color: var(--chan-b);
  }
  .shift-light {
    color: var(--delta-good);
  }

  .steps {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .steps li {
    font-size: 13.5px;
    line-height: 1.5;
    color: var(--ink-300);
  }
  .steps strong {
    color: var(--ink-100);
    font-weight: 600;
  }
</style>
