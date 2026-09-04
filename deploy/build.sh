#!/usr/bin/env bash
# Roda dentro do container `builder`. Gera o banco no volume (/data), exporta o
# catálogo, compila o WASM e builda o PWA, publicando o resultado em /out (volume
# servido pelo nginx). O banco persiste em /data entre deploys — se já existe, é
# reaproveitado (não regenera).
set -euo pipefail

DB=/data/paint_knowledge.db
PUB=/app/front/public

if [ ! -f "$DB" ]; then
  echo "[build] gerando banco de catálogo…"
  sqlite3 "$DB" < /app/api/db/migrations/001_initial_schema.sql
  ( cd /app && mescla-seed -db "$DB" -import-catalog )
  sqlite3 "$DB" "PRAGMA journal_mode=DELETE; VACUUM;"
  rm -f "$DB-wal" "$DB-shm"
else
  echo "[build] banco existente reaproveitado do volume ($(sqlite3 "$DB" 'SELECT COUNT(*) FROM paints') tintas)"
fi

echo "[build] exportando catálogo…"
mescla-export -db "$DB" -out "$PUB/data/catalog.json"

echo "[build] compilando motor de cor (WASM)…"
cd /app
GOOS=js GOARCH=wasm go build -o "$PUB/mescla.wasm" ./api/cmd/wasm
WEXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
[ -f "$WEXEC" ] || WEXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
cp "$WEXEC" "$PUB/wasm_exec.js"

echo "[build] build do PWA…"
cd /app/front
npm run build

echo "[build] publicando no volume web…"
rm -rf /out/* 2>/dev/null || true
cp -a dist/. /out/
echo "[build] pronto — $(find /out -type f | wc -l) arquivos publicados."
