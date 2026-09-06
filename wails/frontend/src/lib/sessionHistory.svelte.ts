// US-06 — histórico da sessão ("Na mesa hoje"). Vive só em memória: reiniciar
// o app começa uma mesa nova, de propósito (é o histórico DESTA sessão).

export interface HistoryEntry {
  id: number;
  label: string; // nome do pote ou o hex perguntado
  hex: string;
  r: number;
  g: number;
  b: number;
  paintId?: number; // presente quando o alvo veio de um pote do catálogo
}

const MAX_ENTRIES = 24;

export const sessionHistory: HistoryEntry[] = $state([]);

let nextId = 1;

export function pushHistory(entry: Omit<HistoryEntry, 'id'>) {
  // mesma cor consultada de novo: sobe para o topo em vez de duplicar
  const dup = sessionHistory.findIndex(h => h.hex === entry.hex && h.label === entry.label);
  if (dup !== -1) sessionHistory.splice(dup, 1);
  sessionHistory.unshift({ ...entry, id: nextId++ });
  if (sessionHistory.length > MAX_ENTRIES) sessionHistory.length = MAX_ENTRIES;
}
