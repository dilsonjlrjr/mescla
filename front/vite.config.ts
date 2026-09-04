import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    svelte(),
    VitePWA({
      registerType: 'autoUpdate',
      // wasm + catálogo entram no precache: o app inteiro funciona offline
      // (o cenário-alvo é dentro de loja, onde sinal ruim é comum).
      includeAssets: ['wasm_exec.js', 'mescla.wasm', 'data/catalog.json'],
      workbox: {
        // woff2 no precache: fontes agora são self-hosted (@fontsource),
        // então o app abre offline já com a tipografia certa desde o boot.
        globPatterns: ['**/*.{js,css,html,svg,png,wasm,json,woff2}'],
        maximumFileSizeToCacheInBytes: 8 * 1024 * 1024, // mescla.wasm ~3.3MB
      },
      manifest: {
        name: 'Mescla',
        short_name: 'Mescla',
        description: 'Cor certa, qualquer marca: equivalência de tintas para pintores de miniaturas',
        lang: 'pt-BR',
        display: 'standalone',
        orientation: 'portrait',
        start_url: '/',
        background_color: '#f6f4ef',
        theme_color: '#f6f4ef',
        icons: [
          { src: 'icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icons/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icons/icon-512-maskable.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
    }),
  ],
});
