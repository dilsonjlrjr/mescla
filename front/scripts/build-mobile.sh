#!/usr/bin/env bash
# Build do Mescla Mobile (PWA) — só o shell do app (Vite). Dado e motor de
# cor vêm do mescla-api em runtime (decisão de 04/09/2026, sem fallback
# offline) — não há mais banco/catalog.json/wasm embutidos no build.
# Saída: front/dist/ — pronto pra qualquer host estático (HTTPS), na frente
# de um proxy que encaminhe /api pro mescla-api (ver front/deploy/nginx.conf).
set -euo pipefail

cd "$(dirname "$0")/.."

say() { printf '\033[1;33m▸ %s\033[0m\n' "$*"; }
ok()  { printf '\033[1;32m✓ %s\033[0m\n' "$*"; }

say "Build do app…"
[ -d node_modules ] || npm install
npm run build

ok "PWA pronta em dist/ — sirva atrás de um proxy que encaminhe /api pro mescla-api"
