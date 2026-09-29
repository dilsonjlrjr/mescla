#!/usr/bin/env bash
# Copia a interface da web (front/src) para o desktop. A web é a referência:
# o desktop tem a mesma tela. Ficam de fora só os arquivos próprios do Wails:
# src/desktop/ (ponte da API e download nativo), src/main.ts e src/vite-env.d.ts.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/../.." && pwd)"
web="$raiz/front"
desk="$raiz/wails/frontend"

# Apaga a cópia anterior (menos os arquivos do Wails) e copia de novo, para
# arquivo removido na web sumir também daqui.
find "$desk/src" -mindepth 1 -maxdepth 1 \
  ! -name desktop ! -name main.ts ! -name vite-env.d.ts -exec rm -rf {} +
( cd "$web/src" && find . -mindepth 1 -maxdepth 1 \
  ! -name main.ts ! -name vite-env.d.ts -exec cp -r {} "$desk/src/" \; )
cp -r "$web/public/." "$desk/public/"

echo "Interface do desktop sincronizada com front/."
diff -rq "$web/src" "$desk/src" || true
