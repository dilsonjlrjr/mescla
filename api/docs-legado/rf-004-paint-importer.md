# RF-004 — Paint Importer

## Status

Done

---

## Objetivo

Criar o processo de importação e população das tabelas `product_lines`, `paint_types`, `finish_types`, `coverage_types`, `opacity_types` e `paints` com dados reais de tintas para modelismo.

---

## Motivação

Tintas são o núcleo da aplicação. Sem dados de tintas, não é possível calcular equivalências, gerar receitas ou sugerir misturas. Esta etapa popula o banco com o catálogo completo de tintas dos fabricantes suportados.

---

## Escopo

- Definição de linhas de produtos por fabricante
- Definição de tipos de tinta, acabamento, cobertura, opacidade
- Importação de tintas com código, nome, volume, tipo, acabamento
- Dados hardcoded ou via arquivo JSON/YAML
- Script Go para inserção idempotent

---

## O que faz

- Popula tabela `paint_types` (Acrylic, Enamel, Lacquer, etc.)
- Popula tabela `finish_types` (Matte, Gloss, Satin, etc.)
- Popula tabela `coverage_types` (Opaque, Semi-opaque, Transparent, etc.)
- Popula tabela `opacity_types` (High, Medium, Low)
- Popula tabela `product_lines` com linhas por fabricante
- Popula tabela `paints` com catálogo de tintas
- Execução idempotent (não duplica se já existe)

---

## O que não faz

- Não calcula cores (RF-006)
- Não gera swatches (RF-005)
- Não baixa imagens (RF-003)
- Não calcula equivalências (RF-007)
- Não implementa interface gráfica

---

## Dependências

- **RF-001** — Database Schema (concluído)
- **RF-002** — Manufacturer Importer (concluído)

---

## Pré-requisitos

- Banco `paint_knowledge.db` existente
- Tabelas criadas e fabricantes cadastrados

---

## Gatilhos

- Executado uma vez na configuração inicial
- Pode ser re-executado para adicionar novas tintas

---

## Fluxo esperado

1. Popular tabelas de lookup (tipos, acabamentos, etc.)
2. Popular linhas de produtos por fabricante
3. Para cada fabricante:
   a. Carregar lista de tintas
   b. Para cada tinta, inserir com dados completos
   c. Verificar duplicatas antes de inserir
4. Retornar relatório de importação

---

## Estrutura técnica

### Tabelas de Lookup

```sql
-- paint_types
INSERT INTO paint_types (name) VALUES ('Acrylic'), ('Enamel'), ('Lacquer'), ('Oil'), ('Watercolor');

-- finish_types
INSERT INTO finish_types (name) VALUES ('Matte'), ('Gloss'), ('Satin'), ('Metallic'), ('Fluorescent');

-- coverage_types
INSERT INTO coverage_types (name) VALUES ('Opaque'), ('Semi-opaque'), ('Transparent'), ('Semi-transparent');

-- opacity_types
INSERT INTO opacity_types (name) VALUES ('High'), ('Medium'), ('Low');
```

### Linhas de Produtos

| Fabricante | Linhas |
|------------|--------|
| AK Interactive | 3GEN, Real Colors, Figure Series, Interactive |
| Vallejo | Model Color, Game Color, Model Air, Mecha Color |
| Citadel | Base, Layer, Shade, Dry, Technical |
| Army Painter | Warpaints, Speedpaints, Air |
| Scale75 | Scalecolor, Fantasy & Games, Instant Colors |
| Tamiya | Acrylic, Lacquer |
| Mr Hobby | Aqueous, Color, GX |
| Acrilex | Acrílica, Spray, Esmalte |
| Corfix | Acrílica, Esmalte, Spray |
| Talento | Acrílica, Esmalte |

### Exemplo de Tinta

```go
type PaintSeed struct {
    Manufacturer string
    Line         string
    Code         string
    Name         string
    PaintType    string
    Finish       string
    Coverage     string
    Opacity      string
    VolumeML     float64
}
```

### SQL de Inserção

```sql
-- Product line
INSERT INTO product_lines (manufacturer_id, name)
SELECT id, ? FROM manufacturers WHERE name = ?
ON CONFLICT DO NOTHING;

-- Paint
INSERT INTO paints (manufacturer_id, product_line_id, code, name, paint_type_id, finish_type_id, coverage_type_id, opacity_type_id, volume_ml)
SELECT m.id, pl.id, ?, ?, pt.id, ft.id, ct.id, ot.id, ?
FROM manufacturers m
JOIN product_lines pl ON pl.manufacturer_id = m.id AND pl.name = ?
LEFT JOIN paint_types pt ON pt.name = ?
LEFT JOIN finish_types ft ON ft.name = ?
LEFT JOIN coverage_types ct ON ct.name = ?
LEFT JOIN opacity_types ot ON ot.name = ?
WHERE m.name = ?
ON CONFLICT(manufacturer_id, product_line_id, code) DO UPDATE SET
    name = excluded.name,
    updated_at = CURRENT_TIMESTAMP;
```

---

## Arquivos envolvidos

- `db/seeds/004_paint_importer.go` — Dados e lógica de importação
- `db/seeds/data/` — Dados de tintas (JSON ou hardcoded)

---

## Bibliotecas

- `database/sql` (padrão Go)
- `encoding/json` (se usar JSON)

---

## Banco de dados

- Tabelas: `paint_types`, `finish_types`, `coverage_types`, `opacity_types`, `product_lines`, `paints`
- Operações: INSERT com ON CONFLICT (upsert)

---

## Critérios de Aceite

1. Tabelas de lookup são populadas corretamente
2. Linhas de produtos são criadas por fabricante
3. Tintas são inseridas com dados completos
4. Execução dupla não cria duplicatas
5. Dados podem ser consultados via SQL
6. Script executa sem erros

---

## Critérios de Qualidade

- Código Go idiomático
- Tratamento de erros adequado
- Logs de progresso durante importação
- Função reutilizável para futuros imports
- Dados organizados por fabricante/linha

---

## Exemplo de Uso

```bash
# Executar importação
go run main.go --import-paints

# Verificar resultados
sqlite3 paint_knowledge.db "SELECT COUNT(*) FROM paints;"
sqlite3 paint_knowledge.db "SELECT p.code, p.name, m.name, pl.name FROM paints p JOIN manufacturers m ON p.manufacturer_id = m.id JOIN product_lines pl ON p.product_line_id = pl.id LIMIT 10;"
```

---

## Próximos Requisitos Dependentes

- **RF-005** — Swatch Generator (gera swatches visuais)
- **RF-006** — Color Converter (calcula RGB/HSV/HSL/LAB/LCH)
- **RF-007** — Similarity Engine (calcula equivalências)
