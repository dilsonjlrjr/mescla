// Store mínimo de toasts — feedback imediato de ações (copiar, erros de backend).
// Runes ($state) exigem extensão .svelte.ts.

export interface Toast {
  id: number;
  message: string;
  kind: 'info' | 'error';
}

let nextId = 1;

export const toasts: Toast[] = $state([]);

export function toast(message: string, kind: Toast['kind'] = 'info') {
  const id = nextId++;
  toasts.push({ id, message, kind });
  setTimeout(() => {
    const idx = toasts.findIndex(t => t.id === id);
    if (idx !== -1) toasts.splice(idx, 1);
  }, kind === 'error' ? 5000 : 2600);
}
