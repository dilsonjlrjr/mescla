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

---

## Decisões Tomadas

- **Processo:** passo a passo, um requisito por vez, aprovação antes de implementar
- **Banco:** SQLite com WAL mode, foreign keys habilitadas
- **Driver Go:** a definir (modernc.org/sqlite ou mattn/go-sqlite3)
- **Estrutura de cores:** tabela separada `paint_colors` (1:1 com `paints`)
- **Lookup tables:** normalizadas para tipos, acabamentos, cobertura, opacidade

---

## Arquitetura Definida

```
paint-match-ai/
├── .requirements/          # Backlog de requisitos
│   ├── rf-001-database-schema.md
│   └── rf-002-manufacturer-importer.md
├── db/
│   ├── migrations/
│   │   └── 001_initial_schema.sql
│   └── seeds/
│       ├── seed.go                 # Executor central
│       └── 002_manufacturers.go    # Seed fabricantes
├── main.go                 # Entry point para seeds
├── go.mod                  # Módulo Go
├── go.sum                  # Dependências
├── paint_knowledge.db      # Banco SQLite
├── CONTEXT.md              # Este arquivo
└── PROMPT.md               # Especificação mestre
```

---

## Requisitos Concluídos

| ID | Título | Status |
|----|--------|--------|
| RF-001 | Database Schema | Done |
| RF-002 | Manufacturer Importer | Done |
| RF-003 | Asset Downloader | Done |

---

## Requisitos Pendentes

| ID | Título | Status |
|----|--------|--------|
| RF-004 | Paint Importer | Draft |
| RF-005 | Swatch Generator | Draft |
| RF-006 | Color Converter | Draft |
| RF-007 | Mix Engine | Draft |
| RF-008 | Similarity Engine | Draft |
| RF-009 | AI Retrieval | Draft |
| RF-010 | Desktop Interface | Draft |

---

## Próximos Passos

1. Criar RF-004 (Paint Importer)
2. Implementar importação de tintas e linhas de produtos
3. Continuar sequência até RF-010

---

## Riscos Identificados

- Dados de tintas podem precisar de atualização manual constante
- Thumbnails/imagens dependem de URLs externas (podem quebrar)
- Cálculo de DeltaE requer precisão nos valores LAB/LCH
- Volume de dados pode impactar performance de buscas (mitigado por índices)
