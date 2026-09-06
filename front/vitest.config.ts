import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// TDD unit-only com mocks (regra de projeto): jsdom + componentes isolados,
// serviços HTTP mockados. Nenhum servidor, nenhum banco.
export default defineConfig({
  plugins: [svelte({ configFile: false })],
  resolve: {
    // O plugin do Svelte 5 precisa da condição "browser" pra resolver o runtime
    // client em ambiente de teste.
    conditions: ['browser'],
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.ts'],
    include: ['tests/**/*.test.ts'],
  },
});
