# CONTEXT.md — Paint Match AI

## Resumo

App desktop para pintores de miniaturas. Wails 3 (alpha) + Go + Svelte 5 + TypeScript + TailwindCSS + SQLite.

11 requisitos (RF-001 a RF-011), todos concluídos. O projeto passou por uma correção crítica de
dados de cor no seed, trocou a mistura genérica por uma "receita equivalente" restrita ao catálogo
de um fabricante de destino, e fechou a 1.0 com o **banco de tintas do usuário** ("Meu estoque")
integrado à mistura — detalhes em [Banco de Tintas do Usuário](#banco-de-tintas-do-usuário-meu-estoque)
e [Manutenção / Correções](#manutenção--correções).

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
- **Desktop:** Wails v3 (v3.0.0-alpha) + Svelte 5 (runes: $state, $derived, $effect) + TailwindCSS v4.
  `main.go` usa `application.New`; o `PaintService` é registrado como service em `Options.Services`
- **Design:** tema "Artisan" — dark luxury, Playfair Display (display) + DM Sans (body), amber accent
- **Build:** `go build` + `npm run build` + `wails3 generate bindings -ts` (bindings gerados em
  `frontend/bindings/paint-match-ai/`, gitignorado — importados como `* as PaintService from '../../../bindings/paint-match-ai/paintservice'`)

---

## Mescla Mobile (PWA Android)

Wails não compila pra mobile — o app Android é um **segundo frontend** (`frontend-mobile/`,
Svelte 5 + Vite, sem SMUI) empacotado como **PWA offline-first** (fase futura: APK via Capacitor).
A lógica de cor continua no Go, compilada pra **WebAssembly**; o catálogo vai como JSON em memória.

- **`pkg/equivalence`** — orquestração pura da receita equivalente extraída de `app.go`
  (exclusão da tinta-alvo, mistura forçada 2+ na mesma marca, tips, `MaxViableDeltaE`).
  **Fonte única**: desktop (`app.go`, via SQL) e mobile (`cmd/wasm`, via JSON) chamam a mesma
  função e produzem receita/ΔE idênticos (paridade verificada).
- **`cmd/export`** — `paint_knowledge.db` → `catalog.json` compacto (~736 KB / ~200 KB gzip).
- **`cmd/wasm`** — `GOOS=js GOARCH=wasm`; expõe `window.__mescla`: `init`, `findSimilar`,
  `suggestEquivalentRecipe`, `suggestRecipeForColor`, `suggestFromStock`, `parseStockCSV`,
  `stockCSVTemplate`, `stockToCSV`, `compareToAnchor`, `bestBrandsFor`. Binário ~3,5 MB, precacheado.
- **UX mobile:** 4 abas (Mesclar=inicial, Catálogo, Cor, Mais), sem Home; busca de tinta em
  tela cheia (nunca dropdown); "minha estante" (marcas em localStorage) — receita calculada
  por marca da estante, melhor primeiro; gotas por padrão; **Modo Bancada** (gotas gigantes,
  ×1×2×3, Wake Lock); back do Android integrado via pilha de History API (`nav.svelte.ts`);
  Comparar = lista-âncora em Mais; catálogo virtualizado (11.932 itens, ~12 renderizados).
- **Regra de mistura:** proporções 0% são descartadas da receita (`nonZeroIngredients`) — se a
  marca tem outra tinta idêntica, a resposta colapsa honestamente pra "use essa tinta" (ΔE 0).
- **Build:** `scripts/build-mobile.sh` (db → export → wasm → vite build) → `frontend-mobile/dist/`.
  Dev: `npm run dev` em `frontend-mobile` (launch configs `mescla-mobile` e `mescla-mobile-prod`).
  Artefatos gerados (wasm, catalog.json, wasm_exec.js, dist) são gitignorados.

---

## Banco de Tintas do Usuário (Meu Estoque) — RF-011

O "Meu estoque" é o banco das tintas que o **próprio pintor** tem em casa. Uma tinta de estoque é
**livre** (nome, código, cor em hex, volume, notas) vinculada a um **fabricante que precisa existir**
no catálogo — não referencia uma tinta do catálogo (pode ser um tom que nem está lá). Sem quantidade:
só posse. Existe nos **dois** frontends.

- **`pkg/stock`** — fonte única, neutra de plataforma: `ParseCSV` (crítica linha a linha —
  fabricante existe, hex válido, nome obrigatório; nunca aborta no 1º erro), `CSVTemplate`,
  `ToCSV` (export — inverso exato de `ParseCSV`, round-trip garantido), `ToMixInputs`. Desktop e
  mobile usam a mesma (de)serialização de CSV.
- **`pkg/equivalence.SuggestFromStock(source, stock)`** — o "pulo do gato": receita usando SÓ o
  estoque. Difere de `Suggest` — não exclui por ID (espaços de ID distintos; cor idêntica no estoque
  vira 1:1 "você já tem essa tinta") e nunca força 2+ ingredientes. O **fallback** (estoque não
  alcança → algoritmo normal por fabricante) vive na UI.
- **Desktop:** tabela `user_paints` criada idempotente no boot (`ensureUserSchema`, FK →
  `manufacturers`, **não** toca no catálogo read-only). `userstock.go`: `GetUserPaints`,
  `AddUserPaint`, `UpdateUserPaint`, `DeleteUserPaint`, `ImportUserPaintsCSV`, `UserPaintCSVTemplate`,
  `SuggestEquivalentFromStock`. UI: `MyStockView.svelte` (CRUD + filtro + importar/baixar CSV);
  toggle "Priorizar meu estoque" em `EquivalentRecipeView` (com marca de reserva de fallback + selo
  "no meu estoque" nos ingredientes que o pintor já tem).
- **Mobile:** `stock.svelte.ts` guarda as tintas inteiras em localStorage (`mescla.stock.v1`) com
  **IDs locais negativos** (nunca colidem com o catálogo, positivo). `StockManager.svelte` (CRUD +
  CSV) numa sub-tela de "Mais". Na `MesclarView`, o toggle "priorizar meu estoque" injeta a receita
  do estoque na disputa — se alcança a cor (menor ΔE), aparece primeiro.
- **Match do selo "no meu estoque":** por `fabricante + código` (a tinta de estoque é livre); numa
  receita feita a partir do estoque, todos os ingredientes já são marcados.
- **CSV:** colunas `fabricante,nome,codigo,hex,volume,notas` (hex `#RRGGBB`), cabeçalho opcional.

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
│   ├── equivalence/         # Suggest + SuggestFromStock (fonte única desktop/mobile)
│   ├── stock/               # Paint, ParseCSV, CSVTemplate, ToMixInputs (+ testes) — CSV compartilhado
│   └── similarity/          # engine.go, ranking.go, engine_test.go
├── cmd/seed/main.go         # Seed runner (--import-paints, --download-logos, etc.)
├── main.go                  # Wails 3 entry point (application.New, PaintService como service)
├── app.go                   # PaintService (catálogo + receita equivalente)
├── userstock.go             # PaintService: estoque do usuário (CRUD, CSV, receita de estoque) + testes
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
│           ├── EquivalentRecipeView.svelte  # + toggle "Priorizar meu estoque"
│           ├── MyStockView.svelte           # CRUD do estoque + importação CSV (RF-011)
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
| `GetUserPaints()` | []UserPaintDTO |
| `AddUserPaint(dto)` / `UpdateUserPaint(dto)` | UserPaintDTO |
| `DeleteUserPaint(id)` | error |
| `ImportUserPaintsCSV(csvText)` | CSVImportResultDTO |
| `ExportUserPaintsCSV()` / `UserPaintCSVTemplate()` | string |
| `SuggestEquivalentFromStock(paintID)` | EquivalentRecipeDTO |

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
| EquivalentRecipeView | Tinta de origem + fabricante de destino → receita; toggle "Priorizar meu estoque" (marca de reserva no fallback, selo "no meu estoque" nos ingredientes) |
| MyStockView | CRUD do estoque do usuário: busca/filtro por marca, form (fabricante, nome, código, cor, volume, notas), importar/exportar/baixar-modelo CSV |

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
| RF-011 | Banco de tintas do usuário (Meu estoque) — desktop + mobile | (não commitado) |

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
