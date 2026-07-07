<script lang="ts">
  // Guia didático da Roda cromática. Os visuais são gerados pela MESMA lógica
  // de cor da ferramenta (theory.ts) e refletem a cor que o usuário escolheu —
  // então o guia "conversa" com o que ele está fazendo.
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

  // Mini-roda (mesma fórmula do conic da ferramenta).
  const wheelBg = (() => {
    const stops: string[] = [];
    for (let h = 0; h <= 360; h += 20) stops.push(`hsl(${h} 92% 55%) ${(h / 360) * 100}%`);
    return `conic-gradient(from 0deg, ${stops.join(', ')})`;
  })();

  function polar(h: number, s: number, r: number) {
    const rad = (h * Math.PI) / 180;
    return { x: r + r * s * Math.sin(rad), y: r - r * s * Math.cos(rad) };
  }
</script>

<div class="guide panel">
  <button class="guide-toggle" onclick={() => (open = !open)} aria-expanded={open}>
    <span class="guide-toggle-left">
      <span class="guide-icon"><Icon name="book" size={17} /></span>
      <span>
        <span class="guide-title">Como usar a roda cromática</span>
        <span class="guide-hint">Um guia rápido pra sair da tentativa e erro</span>
      </span>
    </span>
    <span class="guide-chevron" class:flipped={open}><Icon name="chevron-down" size={18} /></span>
  </button>

  {#if open}
    <div class="guide-body">
      <!-- 1. O que é a roda -->
      <div class="card">
        <div class="card-visual">
          <span class="mini-wheel" style="background: {wheelBg};"></span>
        </div>
        <div class="card-text">
          <h4>O matiz vira um círculo</h4>
          <p>
            As cores são organizadas pelo <strong>matiz</strong>, do vermelho ao roxo e de volta. As
            três <strong>primárias</strong> — vermelho, amarelo e azul — se misturam pra gerar todas
            as outras. Quem está <em>lado a lado</em> se parece; quem está <em>de frente</em> se completa.
          </p>
        </div>
      </div>

      <!-- 2. Temperatura -->
      <div class="card">
        <div class="card-visual">
          <span class="temp-bar">
            <span class="temp-warm">quente</span>
            <span class="temp-cool">fria</span>
          </span>
        </div>
        <div class="card-text">
          <h4>Metade quente, metade fria</h4>
          <p>
            Vermelho, laranja e amarelo são <strong>quentes</strong> — avançam e chamam atenção. Verde,
            azul e roxo são <strong>frias</strong> — recuam. Essa é a base do próximo truque: luz
            costuma ser quente, sombra costuma ser fria.
          </p>
        </div>
      </div>

      <!-- 3. Harmonias -->
      <div class="card card-wide">
        <div class="card-text">
          <h4>As três harmonias que resolvem tudo</h4>
          <p>Combinações que funcionam — testadas aqui na sua cor:</p>
        </div>
        <div class="harmony-rows">
          {#each [{ label: 'Complementar', sw: comp, note: 'opostas → contraste que vibra' }, { label: 'Análogas', sw: analog, note: 'vizinhas → combinam sem esforço' }, { label: 'Tríade', sw: triad, note: '3 espaçadas → colorido equilibrado' }] as row}
            <div class="harmony-row">
              <span class="harmony-name">{row.label}</span>
              <span class="harmony-chips">
                {#each row.sw as s}<span class="chip" style="background: {s.hex};"></span>{/each}
              </span>
              <span class="harmony-note">{row.note}</span>
            </div>
          {/each}
        </div>
      </div>

      <!-- 4. Luz e sombra sem lama -->
      <div class="card card-wide highlight-card">
        <div class="card-text">
          <h4>O pulo do gato: clarear e escurecer sem "lama"</h4>
          <p>
            Branco e preto puros <strong>matam</strong> a cor — ela fica leitosa ou suja. O jeito do
            pintor é <strong>girar o matiz</strong>: pra iluminar, caminhe pro <em>amarelo</em> e suba
            o tom; pra sombrear, caminhe pro <em>azul</em> (ou puxe o complementar) e desça. Assim a
            cor continua viva em toda a escala.
          </p>
        </div>
        <div class="shift-strip">
          <span class="shift-label shift-shadow">← sombra puxa pro azul</span>
          <div class="shift-swatches">
            {#each ramp as step}
              <span class="shift-sw" class:base={step.kind === 'base'} style="background: {step.hex};"></span>
            {/each}
          </div>
          <span class="shift-label shift-light">luz puxa pro amarelo →</span>
        </div>
      </div>

      <!-- 5. Na prática -->
      <div class="card card-wide">
        <div class="card-text">
          <h4>Na hora de pintar a miniatura</h4>
          <ol class="steps">
            <li>Escolha a <strong>cor base</strong> e pinte a peça inteira com ela (camadas finas).</li>
            <li>Nas <strong>dobras e fundos</strong>, aplique a sombra (o tom que puxa pro azul/complementar).</li>
            <li>Nas <strong>saliências e bordas</strong> que pegam luz, aplique o realce (o tom que puxa pro amarelo).</li>
            <li>Quanto mais <strong>gradual</strong> a transição entre os tons, mais natural o volume.</li>
          </ol>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .guide {
    padding: 0;
    overflow: hidden;
    margin-bottom: 24px;
  }

  .guide-toggle {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 16px 18px;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
  }
  .guide-toggle-left {
    display: flex;
    align-items: center;
    gap: 13px;
  }
  .guide-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--lacquer) 14%, transparent);
    color: var(--lacquer-deep);
    flex-shrink: 0;
  }
  .guide-title {
    display: block;
    font-family: var(--font-display);
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
    color: var(--ink-400);
    transition: transform 0.22s ease;
  }
  .guide-chevron.flipped {
    transform: rotate(180deg);
  }

  .guide-body {
    padding: 4px 18px 20px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .card {
    display: flex;
    gap: 14px;
    align-items: flex-start;
    padding: 14px;
    border: 1px solid var(--ink-800);
    border-radius: 12px;
    background: var(--ink-850);
  }
  .card-wide {
    grid-column: 1 / -1;
    flex-direction: column;
    gap: 12px;
  }
  .highlight-card {
    border-color: color-mix(in srgb, var(--lacquer) 35%, var(--ink-700));
    background: color-mix(in srgb, var(--lacquer) 6%, var(--ink-850));
  }

  .card-visual {
    flex-shrink: 0;
  }
  .card-text h4 {
    font-family: var(--font-display);
    font-size: 14px;
    font-weight: 600;
    color: var(--ink-100);
    margin-bottom: 5px;
  }
  .card-text p {
    font-size: 13px;
    line-height: 1.5;
    color: var(--ink-400);
  }
  .card-text strong {
    color: var(--ink-100);
    font-weight: 600;
  }
  .card-text em {
    font-style: normal;
    color: var(--lacquer-deep);
    font-weight: 600;
  }

  .mini-wheel {
    display: block;
    width: 64px;
    height: 64px;
    border-radius: 50%;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }

  .temp-bar {
    display: flex;
    width: 84px;
    height: 64px;
    border-radius: 10px;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .temp-warm,
  .temp-cool {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    flex: 1;
    padding-bottom: 6px;
    font-size: 9.5px;
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
    gap: 8px;
  }
  .harmony-row {
    display: grid;
    grid-template-columns: 110px auto 1fr;
    align-items: center;
    gap: 12px;
  }
  .harmony-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--ink-100);
  }
  .harmony-chips {
    display: flex;
    gap: 4px;
  }
  .chip {
    width: 26px;
    height: 26px;
    border-radius: 7px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .harmony-note {
    font-size: 12px;
    color: var(--ink-500);
  }

  .shift-strip {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .shift-swatches {
    display: flex;
    gap: 4px;
    flex: 1;
  }
  .shift-sw {
    flex: 1;
    height: 34px;
    border-radius: 7px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
  }
  .shift-sw.base {
    box-shadow: inset 0 0 0 2px var(--lacquer);
  }
  .shift-label {
    font-size: 11px;
    font-weight: 600;
    white-space: nowrap;
  }
  .shift-shadow {
    color: var(--chan-b);
  }
  .shift-light {
    color: var(--delta-good);
  }

  .steps {
    margin: 4px 0 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .steps li {
    font-size: 13px;
    line-height: 1.5;
    color: var(--ink-400);
  }
  .steps strong {
    color: var(--ink-100);
    font-weight: 600;
  }

  @media (max-width: 900px) {
    .guide-body {
      grid-template-columns: 1fr;
    }
    .shift-strip {
      flex-direction: column;
      align-items: stretch;
    }
    .harmony-row {
      grid-template-columns: 92px auto;
      grid-template-areas: 'name chips' 'note note';
      gap: 6px 10px;
    }
    .harmony-note {
      grid-area: note;
    }
  }
</style>
