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
        // Phosphor publica fallbacks legados (svg/ttf/eot) de ~3 MB que
        // navegador nenhum desta década baixa — fora do precache.
        globIgnores: ['**/Phosphor*.svg', '**/Phosphor*.ttf', '**/Phosphor*.eot'],
      },
      manifest: {
        name: 'Mescla',
        short_name: 'Mescla',
        description: 'Cor certa, qualquer marca: equivalência de tintas para pintores de miniaturas',
        lang: 'pt-BR',
        display: 'standalone',
        // D-001: o protótipo é iPad em paisagem ("iPad 11 landscape"); travar
        // em portrait contradizia o desenho aprovado.
        orientation: 'any',
        start_url: '/',
        background_color: '#161826',
        theme_color: '#161826',
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
