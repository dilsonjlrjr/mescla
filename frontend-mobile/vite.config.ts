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
        globPatterns: ['**/*.{js,css,html,svg,png,wasm,json}'],
        maximumFileSizeToCacheInBytes: 8 * 1024 * 1024, // mescla.wasm ~3.3MB
        runtimeCaching: [
          {
            // Google Fonts: cache-first com fallback — depois da primeira
            // visita as fontes funcionam offline.
            urlPattern: /^https:\/\/fonts\.(googleapis|gstatic)\.com\/.*/,
            handler: 'CacheFirst',
            options: {
              cacheName: 'google-fonts',
              expiration: { maxEntries: 24, maxAgeSeconds: 60 * 60 * 24 * 365 },
            },
          },
        ],
      },
      manifest: {
        name: 'Mescla',
        short_name: 'Mescla',
        description: 'Cor certa, qualquer marca — equivalência de tintas para pintores de miniaturas',
        lang: 'pt-BR',
        display: 'standalone',
        orientation: 'portrait',
        start_url: '/',
        background_color: '#f6f4f0',
        theme_color: '#f6f4f0',
        icons: [
          { src: 'icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icons/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icons/icon-512-maskable.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
    }),
  ],
});
