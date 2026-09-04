# ── Mescla — interface de build ──────────────────────────────────────────────
# Alvos no padrão do painel-chamados, adaptados ao fluxo do Mescla (Wails v3):
# a lógica pesada vive em wails/scripts/build-all.sh (desktop) e
# scripts/build-mobile.sh (PWA Android) — este Makefile é só a porta de entrada.
#
# Diferenças pro Makefile original (Wails v2), de propósito:
#   - Sem `wails build -platform`: Wails v3 builda com `go build` direto;
#     o bundle .app, assinatura e DMG são montados pelo build-all.sh.
#   - Windows NÃO precisa de mingw: o v3 usa syscalls puros no Windows e o
#     SQLite é modernc (Go puro) → cross-compila do mac com CGO_ENABLED=0.
#   - Linux não cross-compila do mac (webview GTK/WebKitGTK exige CGO):
#     o build roda dentro de Docker (build-all.sh cuida disso).
#   - DMG via hdiutil (nativo) — dispensa `brew install create-dmg`.
#   - `install` NÃO reassina: o build-all.sh já assinou (Developer ID se
#     houver no keychain, senão ad-hoc); reassinar ad-hoc aqui destruiria
#     uma assinatura Developer ID.

APP    := mescla
BUNDLE := Mescla.app
OUTDIR := dist

.PHONY: all macos windows linux mobile clean generate-icns install run

all: macos

# ── macOS: arm64 + amd64 + universal + .app assinado + DMG ──────────────────
macos:
	./wails/scripts/build-all.sh mac

# ── Windows amd64 (cross-compile do mac, sem toolchain extra) ───────────────
windows:
	./wails/scripts/build-all.sh windows

# ── Linux arm64 + amd64 (via Docker; avisa e pula se não houver) ────────────
linux:
	./wails/scripts/build-all.sh linux

# ── PWA Android (banco → catalog.json → wasm → vite build) ──────────────────
mobile:
	./scripts/build-mobile.sh

# ── Gera wails/build/darwin/icons.icns a partir de wails/build/appicon.png ──
# Utilitário explícito (o build-all.sh já tem esse fallback embutido).
# Requer: sips + iconutil (nativos no macOS)
generate-icns:
	@test -f wails/build/appicon.png || { echo "✗ wails/build/appicon.png não existe"; exit 1; }
	@echo "▶ Gerando icns a partir de wails/build/appicon.png..."
	@ICONSET="$$(mktemp -d)/appicon.iconset"; \
	mkdir -p "$$ICONSET" wails/build/darwin; \
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s wails/build/appicon.png --out "$$ICONSET/icon_$${s}x$${s}.png" >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) wails/build/appicon.png --out "$$ICONSET/icon_$${s}x$${s}@2x.png" >/dev/null; \
	done; \
	iconutil -c icns "$$ICONSET" -o wails/build/darwin/icons.icns; \
	rm -rf "$$ICONSET"
	@echo "✓ icns → wails/build/darwin/icons.icns"

# ── Instala em /Applications (preserva a assinatura feita no build) ──────────
install: macos
	@echo "▶ Instalando em /Applications..."
	@rm -rf "/Applications/$(BUNDLE)"
	@ditto "$(OUTDIR)/$(BUNDLE)" "/Applications/$(BUNDLE)"
	@xattr -cr "/Applications/$(BUNDLE)"
	@echo "✓ Instalado → /Applications/$(BUNDLE)"
	@echo ""
	@echo "⚠️  Se a assinatura foi ad-hoc (sem Developer ID no keychain):"
	@echo "    clique direito → Abrir no Finder (primeira vez apenas)"
	@echo "    ou: sudo spctl --add \"/Applications/$(BUNDLE)\""

# ── Roda direto do dist ──────────────────────────────────────────────────────
run:
	@open "$(OUTDIR)/$(BUNDLE)"

# ── Limpeza (desktop + artefatos gerados do mobile) ─────────────────────────
clean:
	rm -rf wails/build/bin $(OUTDIR) \
		front/dist front/dev-dist \
		front/public/mescla.wasm \
		front/public/wasm_exec.js \
		front/public/data/catalog.json
