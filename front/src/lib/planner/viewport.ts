// Viewport da foto de T2 (Plano da peça): a matemática de ajuste, zoom e
// conversão tela ↔ imagem, isolada do componente para poder ser testada
// sozinha (D-002 e rf-06).
//
// Convenção: a "caixa" é a área visível da foto em CSS px — o que o canvas
// ocupa na tela. `zoom` e `pan` levam coordenadas da IMAGEM para coordenadas
// da CAIXA. Misturar esses dois espaços foi exatamente o defeito D-002a: o
// ajuste era calculado contra o tamanho do próprio canvas, que por sua vez
// dependia da resolução da imagem.

export interface View {
  /** escala aplicada à imagem; 1 = pixel da imagem = pixel da caixa */
  zoom: number;
  /** deslocamento em px da caixa */
  panX: number;
  panY: number;
}

export interface Point {
  x: number;
  y: number;
}

export const ZOOM_MIN = 0.05;
export const ZOOM_MAX = 8;

export function clampZoom(z: number): number {
  return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, z));
}

/** Ajuste inicial: a foto inteira cabe na caixa e encosta em pelo menos uma
 *  borda. Nunca amplia além do tamanho real — ampliar é decisão do usuário. */
export function fitView(imgW: number, imgH: number, boxW: number, boxH: number): View {
  if (imgW <= 0 || imgH <= 0 || boxW <= 0 || boxH <= 0) {
    return { zoom: 1, panX: 0, panY: 0 };
  }
  const zoom = Math.min(boxW / imgW, boxH / imgH, 1);
  return {
    zoom,
    panX: (boxW - imgW * zoom) / 2,
    panY: (boxH - imgH * zoom) / 2,
  };
}

/** Ponto da caixa (tela) → ponto da imagem. */
export function toImage(v: View, x: number, y: number): Point {
  return { x: (x - v.panX) / v.zoom, y: (y - v.panY) / v.zoom };
}

/** Ponto da imagem → ponto da caixa (tela). */
export function toScreen(v: View, x: number, y: number): Point {
  return { x: x * v.zoom + v.panX, y: y * v.zoom + v.panY };
}

/** Zoom mantendo fixo o ponto da imagem que está sob a âncora — sem isso o
 *  zoom "foge" do que o pintor está olhando. */
export function zoomAround(v: View, factor: number, anchor: Point): View {
  const zoom = clampZoom(v.zoom * factor);
  const real = zoom / v.zoom; // fator efetivo depois do limite
  return {
    zoom,
    panX: anchor.x - (anchor.x - v.panX) * real,
    panY: anchor.y - (anchor.y - v.panY) * real,
  };
}

/** Zoom relativo ao ajuste inicial, para rotular o controle (ex.: "120%"). */
export function zoomPercent(v: View): number {
  return Math.round(v.zoom * 100);
}
