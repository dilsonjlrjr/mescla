#!/usr/bin/env bash
# Roda dentro do container `builder`. Gera o banco no volume (/data) — o
# mescla-api (serviço `api` do compose) lê o mesmo volume — e builda o PWA,
# publicando o resultado em /out (volume servido pelo nginx). O banco persiste
# em /data entre deploys — se já existe, é reaproveitado (não regenera).
set -euo pipefail

DB=/data/paint_knowledge.db

if [ ! -f "$DB" ]; then
  echo "[build] gerando banco de catálogo…"
  sqlite3 "$DB" < /app/api/db/migrations/001_initial_schema.sql
  ( cd /app && mescla-seed -db "$DB" -import-catalog )
  sqlite3 "$DB" "PRAGMA journal_mode=DELETE; VACUUM;"
  rm -f "$DB-wal" "$DB-shm"
else
  echo "[build] banco existente reaproveitado do volume ($(sqlite3 "$DB" 'SELECT COUNT(*) FROM paints') tintas)"
fi

echo "[build] build do PWA…"
cd /app/front
npm run build

echo "[build] publicando no volume web…"
rm -rf /out/* 2>/dev/null || true
cp -a dist/. /out/
echo "[build] pronto — $(find /out -type f | wc -l) arquivos publicados."
