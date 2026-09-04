#!/usr/bin/env bash
# Build completo do Mescla Mobile (PWA):
#   banco → catalog.json → mescla.wasm → vite build
# Saída: front/dist/ — pronto pra qualquer host estático (HTTPS).
set -euo pipefail

# Script vive em front/scripts/ — sobe 2 níveis pra rodar da raiz do módulo Go
# (go.mod fica em project/, e a etapa de banco/export/wasm usa api/ como sibling).
cd "$(dirname "$0")/../.."

say() { printf '\033[1;33m▸ %s\033[0m\n' "$*"; }
ok()  { printf '\033[1;32m✓ %s\033[0m\n' "$*"; }

DB="api/db/embedded/paint_knowledge.db"

# 1. Banco de catálogo (mesmo fluxo do build-all.sh)
if [ ! -f "$DB" ]; then
  say "Banco de catálogo (migração + seed)…"
  mkdir -p api/db/embedded
  sqlite3 "$DB" < api/db/migrations/001_initial_schema.sql
  go run ./api/cmd/seed -db "$DB" -import-catalog >/dev/null
  sqlite3 "$DB" "PRAGMA journal_mode=DELETE; VACUUM;" >/dev/null
  rm -f "$DB-wal" "$DB-shm"
  ok "banco pronto ($(sqlite3 "$DB" 'SELECT COUNT(*) FROM paints') tintas)"
else
  ok "banco existente reutilizado ($(sqlite3 "$DB" 'SELECT COUNT(*) FROM paints') tintas) — apague $DB pra regenerar"
fi

# 2. Catálogo JSON pro app
say "Exportando catálogo…"
go run ./api/cmd/export -db "$DB" -out front/public/data/catalog.json

# 3. Motor de cor WASM
say "Compilando motor de cor (WASM)…"
GOOS=js GOARCH=wasm go build -o front/public/mescla.wasm ./api/cmd/wasm
WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
[ -f "$WASM_EXEC" ] || WASM_EXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
cp "$WASM_EXEC" front/public/wasm_exec.js
ok "mescla.wasm ($(du -h front/public/mescla.wasm | cut -f1 | tr -d ' '))"

# 4. Build do app (Vite + PWA)
say "Build do app…"
cd front
[ -d node_modules ] || npm install
npm run build

ok "PWA pronta em front/dist/ — sirva com HTTPS (ou 'npm run preview' pra testar)"
