# CONTEXT.md — Paint Match AI

## Resumo

App desktop para pintores de miniaturas. Wails 2 + Go + Svelte 5 + TypeScript + TailwindCSS + SQLite.

10 requisitos (RF-001 a RF-010), todos concluídos. Projeto migrou de Wails 3 (alpha) para
Wails 2 (estável), passou por uma correção crítica de dados de cor no seed, e trocou a
mistura genérica por uma "receita equivalente" restrita ao catálogo de um fabricante de
destino — detalhes em [Manutenção / Correções](#manutenção--correções) e [Branches](#branches).

---

## Decisões de Arquitetura

- **DB:** SQLite, WAL mode, foreign keys, `modernc.org/sqlite` (pure Go, no CGO)
- **Schema:** 12 tabelas, 16 índices. `paint_colors` 1:1 com `paints`. Lookup tables normalizadas
- **Mistura:** aditiva RGB, proporções normalizadas. 2 tintas: brute force step 5. 3-4: exhaustive step 10
- **Receita equivalente:** `SuggestBestSubset` testa combinações de 1-N tintas de um fabricante-alvo (pool
  reduzido a 15 via `selectDiverseSubset` se necessário) e escolhe o subconjunto com menor DeltaE.
  `GenerateTips` compara HSL da cor-alvo com o resultado e gera até 3 dicas textuais de ajuste manual
  (clareador, pigmento faltante, matiz)
- **AI Retrieval:** pipeline IntentPaintInfo → FindEquivalent → FindSimilar → MixRecipe → Compare → ManufacturerInfo → General
- **Tipos:** `color.RGB{R,G,B uint8}`, `color.Lab{L,A,B float64}` — structs no pacote `color`
- **Desktop:** Wails v2 (v2.12.0) + Svelte 5 (runes: $state, $derived, $effect) + TailwindCSS v4
- **Design:** tema "Artisan" — dark luxury, Playfair Display (display) + DM Sans (body), amber accent
- **Build:** `go build` + `npm run build` + `wails generate module` (bindings gerados em `frontend/wailsjs/`, gitignorado)

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
│       ├── 004_paint_importer.go   # 45 tintas, 32 product lines (chama SeedPaintColors após SeedPaints)
│       ├── 005_swatch_generator.go # 45 swatches 64x64 PNG (usa RGB real de paint_colors; fallback cinza só se faltar)
│       └── 006_paint_colors.go     # RGB real das 45 tintas + HSV/HSL/Lab/LCH derivados (ver Manutenção)
├── pkg/
│   ├── ai/                  # context.go, retrieval.go, retrieval_test.go
│   ├── color/               # converter.go, deltae.go, converter_test.go
│   ├── mix/                 # engine.go, proportions.go, subset.go, tips.go (+ testes)
│   └── similarity/          # engine.go, ranking.go, engine_test.go
├── cmd/seed/main.go         # Seed runner (--import-paints, --download-logos, etc.)
├── main.go                  # Wails 2 entry point
├── app.go                   # PaintService (11 Go→TS binding methods)
├── frontend/
│   ├── index.html           # Playfair Display + DM Sans + JetBrains Mono
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── app.css          # Artisan theme: obsidian palette, amber accent
│       ├── App.svelte       # Sidebar + router (repassa paintId pré-selecionado entre rotas)
│       └── lib/components/
│           ├── Sidebar.svelte
│           ├── HomeView.svelte
│           ├── CatalogView.svelte      # botão "Buscar receita equivalente" no detalhe da tinta
│           ├── PaintCard.svelte
│           ├── ColorSearchView.svelte
│           ├── CompareView.svelte
│           ├── EquivalentRecipeView.svelte  # substitui MixView.svelte
│           ├── PaintSearchInput.svelte      # autocomplete de tintas, reutilizável
│           └── Icon.svelte   # componente de ícone compartilhado
├── setup.sh                 # RTK: schema → manufacturers → download → paints → swatches
├── Taskfile.yml
├── build/                   # Wails build config
├── paint_knowledge.db       # SQLite: 11 manufacturers, 45 paints, cores reais em paint_colors
├── assets/
│   ├── manufacturers/       # 8-10 logos (AK, Scale75, Acrilex, Tamiya, Corfix, GSW, Army Painter, Vallejo, ...)
│   └── swatches/            # 45 PNG, com a cor real de cada tinta (antes: placeholder cinza 128,128,128)
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
| `SuggestEquivalentRecipe(paintID, targetManufacturerID, maxPaints)` | EquivalentRecipeDTO |
| `CompareColors(paintIDs)` | []SearchResultDTO |
| `ProcessQuery(text)` | ai.Response |

`EquivalentRecipeDTO`: SourcePaintID, SourceName, SourceR/G/B, TargetManufacturer, Ingredients[],
ResultR/G/B, DeltaE, Method, Tips[]. Substituiu o `RecipeDTO` genérico — a receita agora é sempre
calculada dentro do catálogo de um fabricante de destino específico (`FindByColor` removido).

---

## Frontend Views

| View | Descrição |
|------|-----------|
| HomeView | Dashboard: stats + ações rápidas |
| CatalogView | Grid 5 colunas, busca/filtro, modal detalhe |
| ColorSearchView | Sliders RGB, busca DeltaE |
| CompareView | Seleção múltipla, comparação par-a-par |
| EquivalentRecipeView | Tinta de origem (via `PaintSearchInput` ou deep-link do Catálogo) + fabricante de destino → receita com ingredientes, DeltaE e dicas de ajuste |

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
# Setup (schema + dados: fabricantes → tintas → cores reais → logos → swatches)
rtk go run ./cmd/seed --import-paints --download-logos --generate-swatches

# Desktop app
./bin/paint-match-ai

# Dev mode
wails dev

# Testes (46 testes, pacotes: ai, color, mix, similarity, seeds)
rtk go test ./...
```

Se o `paint_knowledge.db` local ainda tiver sido gerado **antes** da correção de cores
(veja Manutenção abaixo), rode o `cmd/seed` de novo com `--import-paints` — o
`ON CONFLICT DO UPDATE` sobrescreve os RGBs cinza pelos reais sem duplicar nada.

---

## Branches

| Branch | Estado | Descrição |
|--------|--------|-----------|
| `main` | estável (`42b337f`) | Última versão antes da migração Wails 3 → 2 |
| `refactor/wails-v2` | em dia (`a7a4303`) | Downgrade Wails 3 (alpha) → Wails 2 (estável) + fix de cores do seed + bump v2.9.1→v2.12.0 + refino UI |
| `refactor/wails-v3` | ativa, checkout no worktree principal (`a7a4303`, criada a partir de `refactor/wails-v2`) | Branch de trabalho atual para a próxima etapa |

Sem remoto configurado (`git remote -v` vazio) — tudo local por enquanto.

---

## Manutenção / Correções

**1. Migração Wails 3 → Wails 2** (`5e9e8e7`, `2354f8a`)
O projeto começou no Wails 3 (alpha) e foi rebaixado para Wails 2 (estável) por
instabilidade de toolchain. `wails.json` passou a usar o schema v2, `main.go` usa
`github.com/wailsapp/wails/v2`. Depois disso o `go.mod` foi atualizado de v2.9.1 → v2.12.0
(`a7a4303`), junto com um refino do tema Artisan e um novo componente `Icon.svelte`.

**2. Bug de cor placeholder em `paint_colors`** (`25d57db`)
Os 45 registros de `paint_colors` estavam todos com RGB `(128,128,128)` — cinza
placeholder. Causa raiz: `005_swatch_generator.go` fazia `LEFT JOIN` em `paint_colors`
(sempre vazio, pois nada nunca o populava) e caía no fallback cinza para *toda* tinta,
gravando isso permanentemente no banco. Não existia nenhum passo de extração de cor real
a partir de imagem/SKU — só logos de fabricante eram baixados.

Correção: novo seed `006_paint_colors.go` com RGB real (Vallejo, Citadel, Army Painter,
Tamiya, Mr Hobby — rastreado via encycolorpedia.com/Scalemates.com/chart Warpaints) ou
tons realistas de tinta para SKUs fictícios do seed (AK Interactive, Scale75, Acrilex,
Corfix, Talento), computando HSV/HSL/Lab/LCH via `pkg/color`. Encadeado em
`004_paint_importer.go` logo após `SeedPaints`, antes da geração de swatches — validado
com 45/45 cores gravadas, 0 sobras em cinza.

**3. Limpeza de branches e config local**
Removida a branch `claude/zen-easley-ecd7f4` (usada só para desenvolver a correção de
cores). `.claude/` (config local do Claude Code — permissões e launch.json do preview)
foi adicionado ao `.gitignore`, pois é específico de máquina, não de projeto.

**4. Mistura genérica → Receita equivalente** (não commitado)
`MixView.svelte` (sugestão de mistura livre, sem fabricante-alvo) foi substituída por
`EquivalentRecipeView.svelte`: o usuário parte de uma tinta existente (Catálogo ou busca
via `PaintSearchInput`) e pede a receita equivalente **dentro do catálogo de um fabricante
de destino específico**. Motivação: uma mistura sem restrição de fabricante gera receitas
com tintas de marcas diferentes, pouco úteis na prática — o pintor normalmente quer saber
"como chego nessa cor só com o que a marca X vende".

Isso exigiu dois módulos novos em `pkg/mix`: `subset.go` (`SuggestBestSubset` testa
combinações de 1-N tintas do fabricante-alvo e escolhe a de menor DeltaE) e `tips.go`
(`GenerateTips` gera até 3 dicas textuais comparando HSL da cor-alvo com o resultado —
clareador, pigmento faltante, matiz). `SuggestRecipe`/`RecipeDTO`/`FindByColor` foram
substituídos por `SuggestEquivalentRecipe`/`EquivalentRecipeDTO`.

---

## Pendências

- **Logos:** fabricantes sem logo podem ainda faltar (checar `assets/manufacturers/`) — sites bloqueiam ou não existem
- **A11y:** warnings no build (labels sem aria)
- **`paint_knowledge.db` local:** se foi gerado antes de `25d57db`, precisa rodar o seed de novo (ver "Como Executar")
- **Receita equivalente:** mudanças de `app.go`, `pkg/mix` e componentes do frontend ainda não commitadas — ver item 4 de Manutenção
