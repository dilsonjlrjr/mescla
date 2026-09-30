#!/usr/bin/env bash
# Instala o Mescla AI desktop no usuário atual (sem sudo).
#
#   ./install.sh <binário>     # ex.: ./install.sh ../../dist/mescla-ai-linux-amd64
#
# No GTK4 o Wails não consegue pôr ícone na janela a partir de bytes: o
# ícone da barra de tarefas e do menu vem do arquivo .desktop, casado pelo
# id da aplicação: "org.wails." + Name de main.go em minúsculas, com
# espaço trocado por "_" (org.wails.mescla_ai).
set -euo pipefail

cd "$(dirname "$0")"
BIN="${1:?uso: $0 <binário do Mescla AI>}"
DATA="${XDG_DATA_HOME:-$HOME/.local/share}"

install -Dm755 "$BIN" "$HOME/.local/bin/mescla-ai"
install -Dm644 mescla-256.png "$DATA/icons/hicolor/256x256/apps/org.wails.mescla_ai.png"
install -Dm644 mescla-512.png "$DATA/icons/hicolor/512x512/apps/org.wails.mescla_ai.png"
sed "s|^Exec=.*|Exec=$HOME/.local/bin/mescla-ai|" org.wails.mescla_ai.desktop \
  > "$DATA/applications/org.wails.mescla_ai.desktop"

command -v update-desktop-database >/dev/null && update-desktop-database -q "$DATA/applications" || true
command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q -t "$DATA/icons/hicolor" || true

echo "Mescla AI instalado em $HOME/.local/bin/mescla-ai (menu de aplicativos: Mescla AI)."
