# CONTEXT.md — Paint Match AI

## Resumo

App desktop para pintores de miniaturas. Wails 3 + Go + Svelte 5 + TypeScript + TailwindCSS + SQLite.

10 requisitos (RF-001 a RF-010), todos concluídos.

---

## Decisões de Arquitetura

- **DB:** SQLite, WAL mode, foreign keys, `modernc.org/sqlite` (pure Go, no CGO)
- **Schema:** 12 tabelas, 16 índices. `paint_colors` 1:1 com `paints`. Lookup tables normalizadas
- **Mistura:** aditiva RGB, proporções normalizadas. 2 tintas: brute force step 5. 3-4: exhaustive step 10
- **AI Retrieval:** pipeline IntentPaintInfo → FindEquivalent → FindSimilar → MixRecipe → Compare → ManufacturerInfo → General
- **Tipos:** `color.RGB{R,G,B uint8}`, `color.Lab{L,A,B float64}` — structs no pacote `color`
- **Desktop:** Wails 3 v3.0.0-alpha2.110 + Svelte 5 (runes: $state, $derived, $effect) + TailwindCSS v4
- **Design:** tema "Atelier" — dark luxury, Playfair Display (display) + DM Sans (body), amber accent
- **Build:** `go build` + `npm run build` + `wails3 generate bindings`

---

## Estrutura do Projeto

```
paint-match-ai/
├── .requirements/          # RF-001 a RF-010
├── db/
│   ├── migrations/
│   │   └── 001_initial_schema.sql
│   └── seeds/
│       ├── seed.go
│       ├── 002_manufacturers.go    # 11 fabricantes
│       ├── 003_asset_downloader.go # Download logos (8/11 OK)
│       ├── 004_paint_importer.go   # 45 tintas, 32 product lines
│       └── 005_swatch_generator.go # 45 swatches 64x64 PNG
├── pkg/
│   ├── ai/                  # context.go, retrieval.go, retrieval_test.go
│   ├── color/               # converter.go, deltae.go, converter_test.go
│   ├── mix/                 # engine.go, proportions.go, engine_test.go
│   └── similarity/          # engine.go, ranking.go, engine_test.go
├── cmd/seed/main.go         # Seed runner (--import-paints, --download-logos, etc.)
├── main.go                  # Wails 3 entry point
├── app.go                   # PaintService (11 Go→TS binding methods)
├── frontend/
│   ├── index.html           # Playfair Display + DM Sans + JetBrains Mono
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── app.css          # Artisan theme: obsidian palette, amber accent
│       ├── App.svelte       # Sidebar + router
│       └── lib/components/
│           ├── Sidebar.svelte
│           ├── HomeView.svelte
│           ├── CatalogView.svelte
│           ├── PaintCard.svelte
│           ├── ColorSearchView.svelte
│           ├── CompareView.svelte
│           └── MixView.svelte
├── setup.sh                 # RTK: schema → manufacturers → download → paints → swatches
├── Taskfile.yml
├── build/                   # Wails build config
├── paint_knowledge.db       # SQLite: 11 manufacturers, 45 paints
├── assets/
│   ├── manufacturers/       # 8 logos (AK, Scale75, Acrilex, Tamiya, Corfix, GSW, Army Painter, Vallejo)
│   └── swatches/            # 45 PNG
└── bin/paint-match-ai       # Binary
```

---

## PaintService Methods

| Método | Retorno |
|--------|---------|
| `GetStats()` | StatsDTO |
| `GetManufacturers()` | []ManufacturerDTO |
| `GetAllPaints()` | []PaintDTO |
| `SearchPaints(query)` | []PaintDTO |
| `GetPaintByID(id)` | PaintDTO |
| `FindSimilar(r,g,b,maxDeltaE,maxResults)` | []SearchResultDTO |
| `FindEquivalences(paintID)` | []SearchResultDTO |
| `SuggestRecipe(r,g,b,maxPaints)` | RecipeDTO |
| `CompareColors(paintIDs)` | []SearchResultDTO |
| `ProcessQuery(text)` | ai.Response |

---

## Frontend Views

| View | Descrição |
|------|-----------|
| HomeView | Dashboard: stats + ações rápidas |
| CatalogView | Grid 5 colunas, busca/filtro, modal detalhe |
| ColorSearchView | Sliders RGB, busca DeltaE |
| CompareView | Seleção múltipla, comparação par-a-par |
| MixView | Sugestão de mistura com barras de proporção |

---

## Requisitos

| RF | Título | Commit |
|----|--------|--------|
| RF-001 | Database Schema | d11891b |
| RF-002 | Manufacturer Importer | 2bd75c8 |
| RF-003 | Asset Downloader | e0df05f + 682e217 |
| RF-004 | Paint Importer | 23ec62e |
| RF-005 | Swatch Generator | 682e217 |
| RF-006 | Color Converter | 628e31a |
| RF-007 | Mix Engine | 050afc7 |
| RF-008 | Similarity Engine | f86bb4d |
| RF-009 | AI Retrieval | 80dfe0e |
| RF-010 | Desktop Interface | df8232f + 93c70c7 + 42b337f |

---

## Como Executar

```bash
# Setup (schema + dados)
rtk go run ./cmd/seed --import-paints --download-logos --generate-swatches

# Desktop app
./bin/paint-match-ai

# Dev mode
wails3 dev

# Testes (33 testes, 6 pacotes)
rtk go test ./...
```

---

## Pendências

- **Logos:** 3 fabricantes sem logo (Citadel, Mr Hobby, Talento) — sites bloqueiam ou não existem
- **Wails 3:** alpha — API pode mudar
- **A11y:** warnings no build (labels sem aria)
