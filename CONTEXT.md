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
│   └── rf-009-ai-retrieval.md
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
├── main.go                 # Entry point para seeds
├── setup.sh                # Script RTK para setup
├── go.mod                  # Módulo Go
├── go.sum                  # Dependências
├── paint_knowledge.db      # Banco SQLite
├── assets/
│   ├── manufacturers/      # Logos dos fabricantes
│   └── swatches/           # Swatches gerados
├── CONTEXT.md              # Este arquivo
└── PROMPT.md               # Especificação mestre
```

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

---

## Requisitos Pendentes

| ID | Título | Status |
|----|--------|--------|
| RF-010 | Desktop Interface | Draft |

---

## Próximos Passos

1. Criar RF-010 (Desktop Interface — Wails 3 + Svelte 5)
2. Implementar interface desktop completa

---

## Riscos Identificados

- Dados de tintas podem precisar de atualização manual constante
- Thumbnails/imagens dependem de URLs externas (podem quebrar)
- Cálculo de DeltaE requer precisão nos valores LAB/LCH
- Volume de dados pode impactar performance de buscas (mitigado por índices)
