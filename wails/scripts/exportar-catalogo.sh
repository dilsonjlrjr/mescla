#!/usr/bin/env bash
# Atualiza o catálogo que o app desktop embute (wails/catalogo/paint_knowledge.db)
# com o que está cadastrado na web: fabricantes, tintas, cores e tipos.
# Estoque, projetos e receitas salvas NÃO vão para o desktop.
#
#   ./wails/scripts/exportar-catalogo.sh                 # lê do container mescla-api-1
#   ./wails/scripts/exportar-catalogo.sh <banco.db>      # lê de um arquivo
#
# Depois, faça commit de wails/catalogo/paint_knowledge.db e gere o build.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$raiz"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [[ $# -ge 1 ]]; then
  origem="$1"
else
  container="${MESCLA_API_CONTAINER:-mescla-api-1}"
  # Copia banco, WAL e SHM juntos: o WAL pode ter escrita ainda não gravada no banco.
  for f in paint_knowledge.db paint_knowledge.db-wal paint_knowledge.db-shm; do
    docker cp "$container:/data/$f" "$tmp/$f" 2>/dev/null || true
  done
  origem="$tmp/paint_knowledge.db"
  [[ -f "$origem" ]] || { echo "✗ banco não encontrado em $container:/data"; exit 1; }
fi

go run ./wails/cmd/exportar-catalogo -origem "$origem" -destino wails/catalogo/paint_knowledge.db
echo "✓ wails/catalogo/paint_knowledge.db ($(du -h wails/catalogo/paint_knowledge.db | cut -f1))"
