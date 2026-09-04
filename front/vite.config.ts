import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { VitePWA } from 'vite-plugin-pwa';

// Decisão de 04/09/2026: front consome mescla-api (fasthttp) pela rede, sem
// fallback offline — ver brain-mescla-ai/2026-09-04-reorg-api-wails-front.md.
// O precache do PWA cobre só o shell do app (JS/CSS/ícones); dado e motor de
// cor exigem API alcançável.
export default defineConfig({
  plugins: [
    svelte(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // woff2 no precache: fontes agora são self-hosted (@fontsource),
        // então o shell do app abre já com a tipografia certa desde o boot.
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
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
  server: {
    // Dev: encaminha /api pro mescla-api rodando local (api/cmd/apiserver,
    // porta padrão 8080) — front chama sempre "/api/...", nunca uma URL
    // absoluta; produção faz o mesmo via nginx (front/deploy/nginx.conf).
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
});
