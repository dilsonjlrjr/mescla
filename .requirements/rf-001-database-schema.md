# RF-001 — Database Schema

## Status

Done

---

## Objetivo

Criar o schema completo do banco de dados SQLite `paint_knowledge.db`, incluindo todas as tabelas, índices, restrições e migração inicial necessários para armazenar a base de conhecimento de tintas para modelismo.

---

## Motivação

O banco de dados é a única fonte oficial de conhecimento da aplicação. Toda funcionalidade subsequente — importação de tintas, cálculo de equivalências, geração de receitas, busca por similaridade — depende de uma estrutura de dados sólida, normalizada e performática.

Sem este schema, nenhuma outra atividade pode ser iniciada.

---

## Escopo

- Definição de todas as tabelas do domínio
- Definição de chaves primárias e estrangeiras
- Definição de índices para performance
- Definição de restrições de integridade
- Script de migração inicial
- Estrutura preparada para futuras expansões

---

## O que faz

- Cria o arquivo `paint_knowledge.db`
- Cria as tabelas: `manufacturers`, `product_lines`, `paints`, `paint_colors`, `equivalences`, `recipes`, `recipe_ingredients`, `resources`, `paint_types`, `finish_types`, `coverage_types`, `opacity_types`
- Define relacionamentos entre tabelas
- Cria índices para colunas frequentemente consultadas
- Define valores padrão e restrições NOT NULL onde aplicável
- Prepara estrutura para suporte multi-idioma futuro

---

## O que não faz

- Não popula dados (importação é RF-002+)
- Não cria a aplicação Go/Svelte
- Não implementa repositórios ou serviços
- Não baixa imagens ou thumbnails
- Não calcula cores ou similaridades

---

## Dependências

Nenhuma. Este é o requisito raiz do projeto.

---

## Pré-requisitos

- SQLite 3.x instalado
- Go 1.22+ com driver `modernc.org/sqlite` ou `github.com/mattn/go-sqlite3`

---

## Gatilhos

- Executado uma única vez na inicialização do projeto
- Pode ser re-executado com `--force` para recriar o banco (destrutivo)

---

## Fluxo esperado

1. Aplicação inicia
2. Verifica se `paint_knowledge.db` existe
3. Se não existe, executa script de migração
4. Cria todas as tabelas
5. Aplica índices
6. Banco pronto para receber dados

---

## Estrutura técnica

### Tabela: `manufacturers`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| name | TEXT | NOT NULL, UNIQUE | Nome do fabricante |
| country | TEXT | | País de origem |
| website | TEXT | | Site oficial |
| logo_path | TEXT | | Caminho para logotipo local |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

### Tabela: `product_lines`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| manufacturer_id | INTEGER | FK → manufacturers.id, NOT NULL | Fabricante |
| name | TEXT | NOT NULL | Nome da linha |
| description | TEXT | | Descrição da linha |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

**Índice:** `idx_product_lines_manufacturer` em `manufacturer_id`

### Tabela: `paint_types`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| name | TEXT | NOT NULL, UNIQUE | Tipo (Acrylic, Enamel, Lacquer, etc.) |

### Tabela: `finish_types`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| name | TEXT | NOT NULL, UNIQUE | Acabamento (Matte, Gloss, Satin, etc.) |

### Tabela: `coverage_types`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| name | TEXT | NOT NULL, UNIQUE | Cobertura (Opaque, Semi-opaque, Transparent, etc.) |

### Tabela: `opacity_types`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| name | TEXT | NOT NULL, UNIQUE | Opacidade (High, Medium, Low) |

### Tabela: `paints`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| manufacturer_id | INTEGER | FK → manufacturers.id, NOT NULL | Fabricante |
| product_line_id | INTEGER | FK → product_lines.id, NOT NULL | Linha |
| code | TEXT | NOT NULL | Código da tinta |
| name | TEXT | NOT NULL | Nome da tinta |
| paint_type_id | INTEGER | FK → paint_types.id | Tipo de tinta |
| finish_type_id | INTEGER | FK → finish_types.id | Acabamento |
| coverage_type_id | INTEGER | FK → coverage_types.id | Cobertura |
| opacity_type_id | INTEGER | FK → opacity_types.id | Opacidade |
| volume_ml | REAL | | Volume em ml |
| thumbnail_path | TEXT | | Caminho thumbnail local |
| image_path | TEXT | | Caminho imagem oficial local |
| notes | TEXT | | Observações |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

**Índices:**
- `idx_paints_manufacturer` em `manufacturer_id`
- `idx_paints_line` em `product_line_id`
- `idx_paints_code` em `code`
- `idx_paints_name` em `name`
- `idx_paints_manufacturer_line_code` em `(manufacturer_id, product_line_id, code)` — UNIQUE

### Tabela: `paint_colors`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| paint_id | INTEGER | FK → paints.id, NOT NULL, UNIQUE | Tinta |
| rgb_r | INTEGER | NOT NULL, 0-255 | Componente Red |
| rgb_g | INTEGER | NOT NULL, 0-255 | Componente Green |
| rgb_b | INTEGER | NOT NULL, 0-255 | Componente Blue |
| hsv_h | REAL | NOT NULL, 0-360 | Hue (HSV) |
| hsv_s | REAL | NOT NULL, 0-100 | Saturation (HSV) |
| hsv_v | REAL | NOT NULL, 0-100 | Value (HSV) |
| hsl_h | REAL | NOT NULL, 0-360 | Hue (HSL) |
| hsl_s | REAL | NOT NULL, 0-100 | Saturation (HSL) |
| hsl_l | REAL | NOT NULL, 0-100 | Lightness (HSL) |
| lab_l | REAL | NOT NULL | L* (CIELAB) |
| lab_a | REAL | NOT NULL | a* (CIELAB) |
| lab_b | REAL | NOT NULL | b* (CIELAB) |
| lch_l | REAL | NOT NULL | L (CIELCH) |
| lch_c | REAL | NOT NULL | C (CIELCH) |
| lch_h | REAL | NOT NULL | H (CIELCH) |
| swatch_path | TEXT | | Caminho swatch gerado |
| delta_e_method | TEXT | DEFAULT '2000' | Método DeltaE padrão |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

**Índice:** `idx_paint_colors_paint` em `paint_id`

### Tabela: `equivalences`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| source_paint_id | INTEGER | FK → paints.id, NOT NULL | Tinta origem |
| target_paint_id | INTEGER | FK → paints.id, NOT NULL | Tinta destino |
| similarity | REAL | NOT NULL, 0-100 | Percentual de similaridade |
| delta_e | REAL | NOT NULL | Valor DeltaE |
| delta_e_method | TEXT | NOT NULL, DEFAULT '2000' | Método (76, 94, 2000) |
| source | TEXT | | Fonte da informação |
| notes | TEXT | | Observações |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

**Índices:**
- `idx_equivalences_source` em `source_paint_id`
- `idx_equivalences_target` em `target_paint_id`
- `idx_equivalences_pair` em `(source_paint_id, target_paint_id)` — UNIQUE

### Tabela: `recipes`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| target_paint_id | INTEGER | FK → paints.id, NOT NULL | Tinta desejada |
| source_manufacturer_id | INTEGER | FK → manufacturers.id | Fabricante origem |
| target_manufacturer_id | INTEGER | FK → manufacturers.id | Fabricante destino |
| precision | REAL | 0-100 | Precisão da receita |
| delta_e | REAL | | DeltaE da mistura |
| method | TEXT | | Método de cálculo |
| source | TEXT | | Fonte da informação |
| notes | TEXT | | Observações |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de atualização |

**Índices:**
- `idx_recipes_target` em `target_paint_id`
- `idx_recipes_source_manufacturer` em `source_manufacturer_id`
- `idx_recipes_target_manufacturer` em `target_manufacturer_id`

### Tabela: `recipe_ingredients`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| recipe_id | INTEGER | FK → recipes.id, NOT NULL | Receita |
| paint_id | INTEGER | FK → paints.id, NOT NULL | Tinta ingrediente |
| percentage | REAL | NOT NULL, 0-100 | Percentual na mistura |
| order | INTEGER | NOT NULL, DEFAULT 0 | Ordem de adição |

**Índices:**
- `idx_recipe_ingredients_recipe` em `recipe_id`
- `idx_recipe_ingredients_paint` em `paint_id`

### Tabela: `resources`

| Coluna | Tipo | Restrição | Descrição |
|--------|------|-----------|-----------|
| id | INTEGER | PK, AUTOINCREMENT | Identificador único |
| paint_id | INTEGER | FK → paints.id | Tinta relacionada (opcional) |
| manufacturer_id | INTEGER | FK → manufacturers.id | Fabricante relacionado (opcional) |
| type | TEXT | NOT NULL | Tipo (thumbnail, image, catalog, url) |
| path | TEXT | | Caminho local |
| url | TEXT | | URL original |
| description | TEXT | | Descrição do recurso |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | Data de criação |

**Índices:**
- `idx_resources_paint` em `paint_id`
- `idx_resources_manufacturer` em `manufacturer_id`
- `idx_resources_type` em `type`

---

## Arquivos envolvidos

- `db/migrations/001_initial_schema.sql` — Script SQL completo
- `db/schema.go` — Definições Go para referência
- `paint_knowledge.db` — Arquivo do banco (gerado)

---

## Bibliotecas

- `modernc.org/sqlite` (driver SQLite puro Go, sem CGO) OU
- `github.com/mattn/go-sqlite3` (driver CGO, mais rápido)

---

## Banco de dados

- Engine: SQLite 3.x
- Arquivo: `paint_knowledge.db`
- Encoding: UTF-8
- Journal mode: WAL (Write-Ahead Logging) para performance
- Foreign keys: habilitadas (`PRAGMA foreign_keys = ON`)

---

## Critérios de aceite

1. Script SQL executa sem erros em SQLite 3.x
2. Todas as tabelas são criadas corretamente
3. Todos os índices são aplicados
4. Foreign keys funcionam (inserção de referência inválida deve falhar)
5. Constraints UNIQUE funcionam (duplicata deve falhar)
6. Valores DEFAULT são aplicados corretamente
7. Tabelas de lookup (`paint_types`, `finish_types`, etc.) podem ser populadas
8. Banco pode ser aberto e consultado via CLI `sqlite3`

---

## Critérios de qualidade

- Nomes de tabelas em snake_case
- Nomes de colunas em snake_case
- Chaves estrangeiras nomeadas explicitamente
- Índices nomeados com prefixo `idx_`
- Timestamps em UTC
- Comentários explicativos no SQL para tabelas e colunas principais
- Script idempotent (verifica existência antes de criar)

---

## Exemplo de uso

```sql
-- Verificar estrutura
sqlite3 paint_knowledge.db ".schema"

-- Contar tabelas
sqlite3 paint_knowledge.db "SELECT count(*) FROM sqlite_master WHERE type='table';"

-- Listar fabricantes (após população)
sqlite3 paint_knowledge.db "SELECT * FROM manufacturers;"
```

---

## Próximos requisitos dependentes

- **RF-002** — Manufacturer Importer (popula tabela `manufacturers`)
- **RF-003** — Thumbnail Downloader (popula `resources` e `paints.thumbnail_path`)
- **RF-004** — Paint Importer (popula `paints`, `product_lines`, tabelas de lookup)
- **RF-005** — Swatch Generator (popula `paint_colors`, gera `swatch_path`)
- **RF-006** — Color Converter (calcula RGB/HSV/HSL/LAB/LCH para `paint_colors`)
