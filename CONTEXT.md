# CONTEXT.md — Paint Match AI

## Resumo da Interação

- Inicializado repositório Git
- Criada estrutura `.requirements/`
- Criado e aprovado requisito RF-001 (Database Schema)
- Implementado schema completo do banco `paint_knowledge.db`
- Testado script SQL: 12 tabelas, 16 índices, foreign keys e constraints funcionais
- Criado e aprovado requisito RF-002 (Manufacturer Importer)
- Implementado seed Go com 10 fabricantes reais
- Testado idempotência: inserção e atualização funcionais
- Criado executor central de seeds (`db/seeds/seed.go`)
- Inicializado módulo Go com dependência `modernc.org/sqlite`
- Implementado RF-003 (Asset Downloader) com URLs reais
- Implementado RF-004 (Paint Importer) com 45 tintas
- Implementado RF-005 (Swatch Generator) com 45 swatches
- Implementado RF-006 (Color Converter) com conversões RGB/HSV/HSL/XYZ/LAB/LCH e DeltaE
- Implementado RF-007 (Mix Engine) com motor de mistura de 2-4 tintas
- Implementado RF-008 (Similarity Engine) com busca por Delta E 2000
- Implementado RF-009 (AI Retrieval) com pipeline de recuperação de conhecimento
- Implementado RF-010 (Desktop Interface) com Wails 3 + Svelte 5 + TypeScript + TailwindCSS

---

## Decisões Tomadas

- **Processo:** passo a passo, um requisito por vez, aprovação antes de implementar
- **Banco:** SQLite com WAL mode, foreign keys habilitadas
- **Driver Go:** modernc.org/sqlite (puro Go, sem CGO)
- **Estrutura de cores:** tabela separada `paint_colors` (1:1 com `paints`)
- **Lookup tables:** normalizadas para tipos, acabamentos, cobertura, opacidade
- **Mistura:** aditiva RGB, proporções normalizadas, busca exaustiva para 3-4 tintas
- **AI Retrieval:** orquestra color/mix/similarity engines em pipeline unificada
- **Tipos RGB/Lab:** adicionados ao pacote `color` para uso como structs
- **Desktop:** Wails 3 v3.0.0-alpha2.110 + Svelte 5 (runes) + TailwindCSS v4
- **Design:** tema "Atelier" — dark luxury, amber/gold accent, glass-morphism
- **Fontes:** Plus Jakarta Sans (display) + JetBrains Mono (code)

---

## Arquitetura Definida

```
paint-match-ai/
├── .requirements/          # Backlog de requisitos
│   ├── rf-001-database-schema.md
│   ├── rf-002-manufacturer-importer.md
│   ├── rf-003-asset-downloader.md
│   ├── rf-004-paint-importer.md
│   ├── rf-005-swatch-generator.md
│   ├── rf-006-color-converter.md
│   ├── rf-007-mix-engine.md
│   ├── rf-008-similarity-engine.md
│   ├── rf-009-ai-retrieval.md
│   └── rf-010-desktop-interface.md
├── db/
│   ├── migrations/
│   │   └── 001_initial_schema.sql
│   └── seeds/
│       ├── seed.go                 # Executor central
│       ├── 002_manufacturers.go    # Seed fabricantes
│       ├── 003_asset_downloader.go # Download logos
│       ├── 004_paint_importer.go   # Importar tintas
│       └── 005_swatch_generator.go # Gerar swatches
├── pkg/
│   ├── ai/
│   │   ├── context.go              # Estruturas Query/Response/PaintInfo
│   │   ├── retrieval.go            # Engine de recuperação de conhecimento
│   │   └── retrieval_test.go       # Testes
│   ├── color/
│   │   ├── converter.go            # Conversões RGB/HSV/HSL/XYZ/LAB/LCH + tipos RGB/Lab
│   │   ├── deltae.go               # DeltaE 76/94/2000
│   │   └── converter_test.go       # Testes
│   ├── mix/
│   │   ├── engine.go               # Motor de mistura
│   │   ├── proportions.go          # Cálculo de proporções
│   │   └── engine_test.go          # Testes
│   └── similarity/
│       ├── engine.go               # Motor de similaridade
│       ├── ranking.go              # Ranking e filtros
│       └── engine_test.go          # Testes
├── cmd/seed/main.go        # Seed runner (entry point antigo)
├── main.go                 # Wails 3 app entry point
├── app.go                  # PaintService (Go → TS bindings)
├── frontend/               # Svelte 5 + TypeScript + TailwindCSS
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── svelte.config.js
│   ├── tsconfig.json
│   └── src/
│       ├── main.ts
│       ├── app.css
│       ├── App.svelte
│       └── lib/components/
│           ├── Sidebar.svelte
│           ├── HomeView.svelte
│           ├── CatalogView.svelte
│           ├── PaintCard.svelte
│           ├── ColorSearchView.svelte
│           ├── CompareView.svelte
│           └── MixView.svelte
├── setup.sh                # Script RTK para setup
├── Taskfile.yml            # Build/dev/seed tasks
├── build/
│   ├── config.yml          # Wails build config
│   ├── Taskfile.yml
│   ├── common/Taskfile.yml
│   └── darwin/Taskfile.yml
├── go.mod                  # Módulo Go
├── go.sum                  # Dependências
├── paint_knowledge.db      # Banco SQLite
├── assets/
│   ├── manufacturers/      # Logos dos fabricantes
│   └── swatches/           # Swatches gerados
├── bin/                    # Binário compilado
├── CONTEXT.md              # Este arquivo
└── PROMPT.md               # Especificação mestre
```

---

## PaintService Methods (Go → TypeScript)

| Método | Descrição |
|--------|-----------|
| `GetStats()` | Estatísticas do banco |
| `GetManufacturers()` | Lista fabricantes |
| `GetAllPaints()` | Lista todas as tintas |
| `SearchPaints(query)` | Busca tintas por texto |
| `GetPaintByID(id)` | Detalhe de tinta |
| `FindSimilar(r,g,b,maxDeltaE,maxResults)` | Busca por cor similar |
| `FindEquivalences(paintID)` | Equivalências entre fabricantes |
| `SuggestRecipe(r,g,b,maxPaints)` | Sugestão de mistura |
| `CompareColors(paintIDs)` | Comparação de cores |
| `ProcessQuery(text)` | Query AI genérica |

---

## Frontend Views

| View | Descrição |
|------|-----------|
| HomeView | Dashboard com stats + ações rápidas |
| CatalogView | Grid de tintas com busca/filtro + modal de detalhe |
| ColorSearchView | Sliders RGB + busca por DeltaE |
| CompareView | Seleção múltipla + comparação DeltaE |
| MixView | Sugestão de mistura com resultado visual |

---

## Requisitos Concluídos

| ID | Título | Status | Commit |
|----|--------|--------|--------|
| RF-001 | Database Schema | Done | d11891b |
| RF-002 | Manufacturer Importer | Done | 2bd75c8 |
| RF-003 | Asset Downloader | Done | e0df05f |
| RF-004 | Paint Importer | Done | 23ec62e |
| RF-005 | Swatch Generator | Done | 682e217 |
| RF-006 | Color Converter | Done | 628e31a |
| RF-007 | Mix Engine | Done | - |
| RF-008 | Similarity Engine | Done | - |
| RF-009 | AI Retrieval | Done | - |
| RF-010 | Desktop Interface | Done | - |

---

## Requisitos Pendentes

Nenhum — todos os requisitos foram concluídos.

---

## Como Executar

### Seed (popular banco)
```bash
rtk go run ./cmd/seed --import-paints --download-logos --generate-swatches
```

### Desktop App
```bash
./bin/paint-match-ai
# ou
wails3 dev
```

### Testes
```bash
rtk go test ./...
```

---

## Riscos Identificados

- Dados de tintas podem precisar de atualização manual constante
- Thumbnails/imagens dependem de URLs externas (podem quebrar)
- Cálculo de DeltaE requer precisão nos valores LAB/LCH
- Volume de dados pode impactar performance de buscas (mitigado por índices)
- Wails 3 em alpha — API pode mudar
