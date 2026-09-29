// rf-21 — liga os ids do modelo do círculo (color/circulo.ts) às chaves do
// dicionário, para o modelo continuar sem dependência de interface.
import type { HueId, SchemeId } from './color/circulo';
import type { DictKey } from './i18n.svelte';

const HUE: Record<HueId, DictKey> = {
  magenta: 'hueMagenta',
  violeta: 'hueVioleta',
  azulVioletado: 'hueAzulVioletado',
  azul: 'hueAzul',
  azulCiano: 'hueAzulCiano',
  turquesa: 'hueTurquesa',
  verde: 'hueVerde',
  verdeLima: 'hueVerdeLima',
  amarelo: 'hueAmarelo',
  laranja: 'hueLaranja',
  vermelho: 'hueVermelho',
  rosa: 'hueRosa',
};

const SCHEME: Record<SchemeId, DictKey> = {
  complementar: 'schComplementar',
  analoga: 'schAnaloga',
  triade: 'schTriade',
  complementarDividido: 'schDividido',
  tetrade: 'schTetrade',
  complementarDupla: 'schDupla',
  monocromatico: 'schMono',
};

const HINT: Record<SchemeId, DictKey> = {
  complementar: 'hintComplementar',
  analoga: 'hintAnaloga',
  triade: 'hintTriade',
  complementarDividido: 'hintDividido',
  tetrade: 'hintTetrade',
  complementarDupla: 'hintDupla',
  monocromatico: 'hintMono',
};

export const hueKey = (id: HueId): DictKey => HUE[id];
export const schemeKey = (id: SchemeId): DictKey => SCHEME[id];
export const hintKey = (id: SchemeId): DictKey => HINT[id];
