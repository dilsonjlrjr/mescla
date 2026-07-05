#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────
# Mescla — build multiplataforma (rodar num macOS)
#
#   ./scripts/build-all.sh            # mac + windows + linux (se houver docker)
#   ./scripts/build-all.sh mac        # só macOS (arm64 + amd64 + universal + .app)
#   ./scripts/build-all.sh windows    # só Windows (amd64, cross-compile)
#   ./scripts/build-all.sh linux      # só Linux (amd64, via Docker)
#
# Por que assim:
#   - macOS   → build nativo com CGO (webview do sistema exige).
#   - Windows → o Wails v3 usa syscalls puros no Windows (go-webview2) e o
#               SQLite é modernc (Go puro), então cross-compila do mac
#               com CGO_ENABLED=0 sem toolchain extra.
#   - Linux   → o webview usa GTK/WebKitGTK via CGO; não dá pra cross-
#               compilar do mac. O script builda dentro de um container
#               Debian com os headers certos (precisa de Docker).
# ──────────────────────────────────────────────────────────────────────
set -euo pipefail

cd "$(dirname "$0")/.."

APP="mescla"
DIST="dist"
VERSION="$(git describe --tags --always 2>/dev/null || echo dev)"
LDFLAGS="-s -w"

say()  { printf '\033[1;36m▸ %s\033[0m\n' "$*"; }
ok()   { printf '\033[1;32m✓ %s\033[0m\n' "$*"; }
warn() { printf '\033[1;33m! %s\033[0m\n' "$*"; }

build_frontend() {
  say "Frontend (vite build)…"
  (cd frontend && npm run build >/dev/null)
  ok "frontend/dist pronto"
}

# Banco de catálogo fresco, embutido no binário — o app instala sozinho
# no primeiro boot (auto-suficiente, sem passo de seed pro usuário).
build_database() {
  say "Banco de catálogo (migração + seed)…"
  local db="db/embedded/paint_knowledge.db"
  rm -f "$db" "$db-wal" "$db-shm"
  mkdir -p db/embedded
  sqlite3 "$db" < db/migrations/001_initial_schema.sql
  go run ./cmd/seed -db "$db" -import-catalog >/dev/null
  sqlite3 "$db" "PRAGMA journal_mode=DELETE; VACUUM;" >/dev/null
  rm -f "$db-wal" "$db-shm"
  ok "banco embutível pronto ($(du -h "$db" | cut -f1 | tr -d ' '), $(sqlite3 "$db" 'SELECT COUNT(*) FROM paints') tintas)"
}

build_mac() {
  say "macOS arm64…"
  GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
    go build -ldflags "$LDFLAGS" -o "$DIST/${APP}-darwin-arm64" .

  say "macOS amd64…"
  GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
    go build -ldflags "$LDFLAGS" -o "$DIST/${APP}-darwin-amd64" .

  say "macOS universal (lipo)…"
  lipo -create -output "$DIST/${APP}-darwin-universal" \
    "$DIST/${APP}-darwin-arm64" "$DIST/${APP}-darwin-amd64"

  # Bundle .app mínimo — ícone no dock, nome certo, clicável no Finder
  say "Bundle Mescla.app…"
  local app_dir="$DIST/Mescla.app/Contents"
  rm -rf "$DIST/Mescla.app"
  mkdir -p "$app_dir/MacOS" "$app_dir/Resources"
  cp "$DIST/${APP}-darwin-universal" "$app_dir/MacOS/$APP"
  if [ -f build/darwin/icons.icns ]; then
    cp build/darwin/icons.icns "$app_dir/Resources/icons.icns"
  elif [ -f build/appicon.png ]; then
    # gera .icns a partir do png (ferramentas nativas do macOS)
    local iconset; iconset="$(mktemp -d)/icon.iconset"
    mkdir -p "$iconset"
    for s in 16 32 128 256 512; do
      sips -z $s $s build/appicon.png --out "$iconset/icon_${s}x${s}.png" >/dev/null
      sips -z $((s*2)) $((s*2)) build/appicon.png --out "$iconset/icon_${s}x${s}@2x.png" >/dev/null
    done
    iconutil -c icns "$iconset" -o "$app_dir/Resources/icons.icns"
  fi
  cat > "$app_dir/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleName</key><string>Mescla</string>
  <key>CFBundleDisplayName</key><string>Mescla</string>
  <key>CFBundleIdentifier</key><string>br.com.mescla.app</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key><string>${APP}</string>
  <key>CFBundleIconFile</key><string>icons</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
</dict></plist>
PLIST
  ok "macOS: ${APP}-darwin-{arm64,amd64,universal} + Mescla.app"

  # Assinatura: usa Developer ID se houver um no keychain; senão ad-hoc.
  # Ad-hoc não passa no Gatekeeper (botão direito → Abrir na 1ª vez), mas
  # evita o "app danificado" em Apple Silicon, que exige código assinado.
  local identity
  identity="$(security find-identity -v -p codesigning 2>/dev/null \
    | { grep -o '"Developer ID Application[^"]*"' || true; } | head -1 | tr -d '"')"
  if [ -n "$identity" ]; then
    say "Assinando com: $identity"
    codesign --force --deep --options runtime --sign "$identity" "$DIST/Mescla.app"
    warn "Pra distribuir sem aviso do Gatekeeper, falta notarizar (xcrun notarytool)."
  else
    say "Assinando ad-hoc (nenhum Developer ID no keychain)…"
    codesign --force --deep --sign - "$DIST/Mescla.app"
    warn "Ad-hoc: em outro Mac, abrir com botão direito → Abrir na primeira vez."
  fi
  codesign --verify --deep "$DIST/Mescla.app" && ok "assinatura válida"

  say "DMG…"
  local staging; staging="$(mktemp -d)"
  cp -R "$DIST/Mescla.app" "$staging/"
  ln -s /Applications "$staging/Applications"
  rm -f "$DIST/Mescla-${VERSION}.dmg"
  hdiutil create -volname "Mescla" -srcfolder "$staging" -ov -format UDZO \
    "$DIST/Mescla-${VERSION}.dmg" >/dev/null
  rm -rf "$staging"
  ok "DMG: Mescla-${VERSION}.dmg (arraste pro Applications)"
}

build_windows() {
  say "Windows amd64 (cross-compile, CGO off)…"
  # Ícone, manifest (DPI aware) e info de versão viram um recurso .syso que o
  # go build linka sozinho no PE quando o arquivo está na raiz do pacote —
  # sem ele o exe sai sem ícone no Explorer/barra de tarefas.
  if command -v wails3 >/dev/null 2>&1; then
    wails3 generate syso -arch amd64 \
      -icon build/windows/icon.ico \
      -manifest build/windows/wails.exe.manifest \
      -info build/windows/info.json \
      -out rsrc_windows_amd64.syso
  else
    warn "wails3 CLI ausente — exe sairá SEM ícone (go install github.com/wailsapp/wails/v3/cmd/wails3@latest)"
  fi
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags "$LDFLAGS -H windowsgui" -o "$DIST/${APP}-windows-amd64.exe" .
  rm -f rsrc_windows_amd64.syso
  ok "Windows: ${APP}-windows-amd64.exe"
}

build_linux() {
  if ! docker info >/dev/null 2>&1; then
    warn "Linux pulado: Docker indisponível (instale/abra o Docker Desktop)."
    warn "O webview Linux usa GTK/WebKitGTK via CGO — não cross-compila do mac."
    warn "Alternativas: rodar 'go build' numa máquina Linux, ou CI (GitHub Actions)."
    return 0
  fi
  # No Apple Silicon o Docker roda arm64 por padrão; fixamos a plataforma
  # de cada build pra o nome do artefato dizer a verdade. O amd64 roda
  # emulado (Rosetta/QEMU) — mais lento, mas correto.
  local arch
  for arch in arm64 amd64; do
    say "Linux ${arch} (via Docker, pode demorar na primeira vez)…"
    docker run --rm --platform "linux/${arch}" \
      -v "$PWD":/src -w /src \
      -v "mescla-go-cache-${arch}":/root/go \
      -e GOFLAGS=-buildvcs=false \
      golang:1.26-trixie \
      bash -c "
        set -e
        apt-get update -qq >/dev/null
        apt-get install -y -qq libgtk-4-dev libwebkitgtk-6.0-dev >/dev/null
        CGO_ENABLED=1 go build -ldflags '-s -w' -o 'dist/${APP}-linux-${arch}' .
      "
    ok "Linux: ${APP}-linux-${arch}"
  done
}

# ── main ──────────────────────────────────────────────────────────────
TARGET="${1:-all}"
mkdir -p "$DIST"

say "Mescla ${VERSION} → ${TARGET}"
build_frontend
build_database

case "$TARGET" in
  mac|darwin)  build_mac ;;
  win|windows) build_windows ;;
  linux)       build_linux ;;
  all)         build_mac; build_windows; build_linux ;;
  *) echo "uso: $0 [mac|windows|linux|all]"; exit 1 ;;
esac

echo
say "Artefatos em ./$DIST:"
ls -lh "$DIST" | awk 'NR>1 {print "   " $5 "\t" $9}'
