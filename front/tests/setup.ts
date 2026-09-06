import '@testing-library/jest-dom/vitest';

// jsdom não implementa matchMedia nem scrollIntoView; as telas usam os dois.
if (!window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }),
  });
}

// Esta versão do jsdom sobe sem Storage; as telas persistem estante/estoque/receitas nele.
if (!window.localStorage) {
  const store = new Map<string, string>();
  const mem: Storage = {
    get length() { return store.size; },
    clear: () => store.clear(),
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    key: (i: number) => [...store.keys()][i] ?? null,
    removeItem: (k: string) => void store.delete(k),
    setItem: (k: string, v: string) => void store.set(k, String(v)),
  };
  Object.defineProperty(window, 'localStorage', { value: mem, writable: true });
  Object.defineProperty(globalThis, 'localStorage', { value: mem, writable: true });
}

// VirtualList usa ResizeObserver; jsdom não tem.
if (!('ResizeObserver' in globalThis)) {
  class RO {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  Object.defineProperty(globalThis, 'ResizeObserver', { value: RO, writable: true });
  Object.defineProperty(window, 'ResizeObserver', { value: RO, writable: true });
}

if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}
