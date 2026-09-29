<script lang="ts">
  // rf-22 RN17/RN18 — lupa de precisão 8×. Círculo de 96 px com cerca de 12×12
  // pixels da foto, sem suavização, mira no centro, o quadrado da janela 5×5 e
  // embaixo o hex da média (o valor que será gravado). Só apresentação: quem
  // lê a cor e decide quando aparece é o PlannerView.
  interface Props {
    /** Mesma fonte de pixels que `lerCorMedia` lê. */
    fonte: HTMLCanvasElement | null;
    /** Pixel da foto sob a mira. */
    ix: number;
    iy: number;
    hex: string;
    /** Ponto da tela onde está o dedo/ponteiro. */
    clientX: number;
    clientY: number;
  }

  let { fonte, ix, iy, hex, clientX, clientY }: Props = $props();

  const DIAMETRO = 96;
  const ZOOM = 8;
  const ALTURA_TOTAL = DIAMETRO + 30;
  const FOLGA = 8;

  let canvasEl: HTMLCanvasElement | undefined = $state();

  $effect(() => {
    const c = canvasEl;
    if (!c || !fonte) return;
    const dpr = window.devicePixelRatio || 1;
    c.width = Math.round(DIAMETRO * dpr);
    c.height = Math.round(DIAMETRO * dpr);
    const ctx = c.getContext('2d');
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.imageSmoothingEnabled = false;
    ctx.fillStyle = '#161826';
    ctx.fillRect(0, 0, DIAMETRO, DIAMETRO);
    // 13×13 pixels a partir de (ix-6, iy-6), deslocados meio pixel ampliado
    // para o centro da mira cair no meio do pixel `ix,iy`.
    const lado = 13;
    const desloc = -ZOOM / 2;
    ctx.drawImage(fonte, ix - 6, iy - 6, lado, lado, desloc, desloc, lado * ZOOM, lado * ZOOM);

    const c0 = DIAMETRO / 2;
    // Janela 5×5 da média (40 px).
    const janela = 5 * ZOOM;
    ctx.lineWidth = 1;
    ctx.strokeStyle = 'rgba(255,255,255,0.95)';
    ctx.strokeRect(c0 - janela / 2 + 0.5, c0 - janela / 2 + 0.5, janela - 1, janela - 1);
    ctx.strokeStyle = 'rgba(0,0,0,0.7)';
    ctx.strokeRect(c0 - janela / 2 - 0.5, c0 - janela / 2 - 0.5, janela + 1, janela + 1);
    // Mira.
    ctx.strokeStyle = 'rgba(255,255,255,0.9)';
    ctx.beginPath();
    ctx.moveTo(c0, c0 - 10); ctx.lineTo(c0, c0 - 4);
    ctx.moveTo(c0, c0 + 4); ctx.lineTo(c0, c0 + 10);
    ctx.moveTo(c0 - 10, c0); ctx.lineTo(c0 - 4, c0);
    ctx.moveTo(c0 + 4, c0); ctx.lineTo(c0 + 10, c0);
    ctx.stroke();
  });

  // RN18: acima do ponto (fora do dedo); perto do topo, abaixo; nunca sai da
  // janela na horizontal.
  let esquerda = $derived(
    Math.max(FOLGA, Math.min(clientX - DIAMETRO / 2, (typeof window === 'undefined' ? 1024 : window.innerWidth) - DIAMETRO - FOLGA)),
  );
  let topo = $derived.by(() => {
    const acima = clientY - ALTURA_TOTAL - 36;
    return acima < FOLGA ? clientY + 44 : acima;
  });
</script>

<div
  aria-hidden="true"
  style="position: fixed; left: {esquerda}px; top: {topo}px; z-index: 50; width: {DIAMETRO}px; pointer-events: none; display: flex; flex-direction: column; align-items: center; gap: 6px;"
>
  <div style="width: {DIAMETRO}px; height: {DIAMETRO}px; border-radius: 50%; overflow: hidden; box-shadow: 0 0 0 2px var(--color-neutral-100, #f3f5fe), 0 4px 16px rgba(0,0,0,0.5);">
    <canvas bind:this={canvasEl} style="width: {DIAMETRO}px; height: {DIAMETRO}px; display: block;"></canvas>
  </div>
  <span class="font-mono" style="padding: 2px 8px; border-radius: 999px; background: rgba(22,24,38,0.95); color: var(--color-text, #f3f5fe); font-size: 12px;">{hex.toUpperCase()}</span>
</div>
