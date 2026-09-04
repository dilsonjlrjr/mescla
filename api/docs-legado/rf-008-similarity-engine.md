# RF-008 — Similarity Engine

**ID**: rf-008-similarity-engine
**Status**: Done
**Requisito anterior**: RF-007 (Mix Engine)

## Objetivo

Motor de similaridade que encontra as tintas mais próximas de uma cor alvo usando Delta E (CIE2000). Suporta busca por:
- Cor sólida (RGB direto)
- Cor misturada (receita de RF-007)

## Escopo

### Pacote: `pkg/similarity/`

**Arquivos:**
- `engine.go` — Engine principal com métodos de busca
- `ranking.go` — Lógica de ranking e filtragem
- `engine_test.go` — Testes unitários

### Estruturas

```go
type SearchResult struct {
    PaintID    int64
    Name       string
    Manufacturer string
    RGB        color.RGB
    DeltaE     float64
    SwatchPath string
}

type SearchOptions struct {
    MaxResults  int       // default: 10
    MaxDeltaE   float64   // default: 10.0 (excluir acima disso)
    ManufacturerID *int64 // filtrar por fabricante
    PaintTypeID    *int64 // filtrar por tipo
}
```

### Métodos

```go
func NewEngine(db *sql.DB) *Engine

// FindSimilar encontra tintas similares a uma cor RGB
func (e *Engine) FindSimilar(target color.RGB, opts SearchOptions) ([]SearchResult, error)

// FindSimilarToMixed encontra tintas similares a uma mistura
func (e *Engine) FindSimilarToMixed(recipe []mix.Ingredient, opts SearchOptions) ([]SearchResult, error)

// FindEquivalent encontra equivalentes entre fabricantes (usa tabela equivalences)
func (e *Engine) FindEquivalent(paintID int64) ([]SearchResult, error)
```

### Algoritmo

1. Carrega todas as tintas do banco com `paint_colors.rgb_r/g/b`
2. Calcula Delta E 2000 entre alvo e cada tinta
3. Filtra por `MaxDeltaE`
4. Ordena por Delta E (menor = mais similar)
5. Retorna top N resultados

### Índices

Usar `idx_paint_colors_rgb` (já existe em RF-001) para performance.

## Critérios de Aceite

- [ ] `FindSimilar` retorna resultados ordenados por Delta E
- [ ] `FindSimilarToMixed` usa `mix.Engine.Mix()` para calcular cor alvo
- [ ] `FindEquivalent` usa tabela `equivalences`
- [ ] Filtros por fabricante e tipo funcionam
- [ ] Testes unitários passam
- [ ] Performance: < 100ms para 1000 tintas

## Prioridade

Alta — motor central do app (encontrar tintas equivalentes)
