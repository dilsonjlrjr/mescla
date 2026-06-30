# RF-009 — AI Retrieval

**ID**: rf-009-ai-retrieval
**Status**: Done
**Requisito anterior**: RF-008 (Similarity Engine)

## Objetivo

Módulo de recuperação de conhecimento que orquestra consultas ao banco de dados, engines de cor/mistura/similaridade e estrutura respostas completas para o usuário.

Este módulo é o "cérebro" que conecta todos os engines anteriores em uma pipeline unificada de recuperação de informação.

## Motivação

A aplicação precisa de uma camada central que:
- Consulte o banco de dados como fonte primária
- Orquestre os engines (color, mix, similarity) em sequência
- Estruture respostas completas com dados, cores, receitas e similaridades
- Nunca invente dados — quando não houver informação cadastrada, informa explicitamente

## Escopo

### Pacote: `pkg/ai/`

**Arquivos:**
- `retrieval.go` — Engine principal de recuperação de conhecimento
- `context.go` — Estrutura de contexto para queries
- `retrieval_test.go` — Testes unitários

### Estruturas

```go
// Query representa uma pergunta do usuário
type Query struct {
    Text         string            // texto original
    Intent       QueryIntent       // intenção detectada
    TargetColor  *color.RGB        // cor alvo (se aplicável)
    Filters      QueryFilters      // filtros extraídos
}

// QueryIntent tipos de intenção
type QueryIntent string

const (
    IntentFindSimilar    QueryIntent = "find_similar"
    IntentFindEquivalent QueryIntent = "find_equivalent"
    IntentMixRecipe      QueryIntent = "mix_recipe"
    IntentPaintInfo      QueryIntent = "paint_info"
    IntentManufacturerInfo QueryIntent = "manufacturer_info"
    IntentCompare        QueryIntent = "compare"
    IntentGeneral        QueryIntent = "general"
)

// QueryFilters filtros opcionais
type QueryFilters struct {
    ManufacturerID *int64
    PaintTypeID    *int64
    FinishTypeID   *int64
    MaxDeltaE      *float64
    MaxResults     *int
}

// Response representa a resposta estruturada
type Response struct {
    Intent       QueryIntent
    Paints       []PaintInfo       // tintas encontradas
    Equivalences []Equivalence     // equivalências
    Recipes      []Recipe          // receitas de mistura
    Similar      []SimilarResult   // resultados de similaridade
    Message      string            // mensagem para o usuário
    Sources      []string          // fontes dos dados
    IsEstimate   bool              // true se baseado em conhecimento geral
}

// PaintInfo informações completas de uma tinta
type PaintInfo struct {
    ID           int64
    Name         string
    Code         string
    Manufacturer string
    ProductLine  string
    RGB          color.RGB
    Lab          color.Lab
    SwatchPath   string
    Thumbnail    string
    ImageURL     string
    FinishType   string
    PaintType    string
    Coverage     string
    Opacity      string
    Volume       string
}

// Equivalence par de tintas equivalentes
type Equivalence struct {
    SourcePaint  PaintInfo
    TargetPaint  PaintInfo
    DeltaE       float64
    Similarity   float64
    Notes        string
}

// Recipe receita de mistura
type Recipe struct {
    TargetPaint  PaintInfo
    Ingredients  []mix.Ingredient
    Method       string
    DeltaE       float64
}

// SimilarResult resultado de busca por similaridade
type SimilarResult struct {
    Paint    PaintInfo
    DeltaE   float64
    Rank     int
}
```

### Métodos

```go
func NewRetrieval(db *sql.DB) *Retrieval

// Process executa a pipeline completa de recuperação
func (r *Retrieval) Process(query Query) (Response, error)

// FindPaintInfo busca informações detalhadas de uma tinta por ID, código ou nome
func (r *Retrieval) FindPaintInfo(identifier string) ([]PaintInfo, error)

// FindEquivalencesFor busca equivalências para uma tinta
func (r *Retrieval) FindEquivalencesFor(paintID int64) ([]Equivalence, error)

// SuggestMixes sugere misturas para uma cor alvo
func (r *Retrieval) SuggestMixes(target color.RGB, maxIngredients int) ([]Recipe, error)

// FindSimilarColors busca cores similares com contexto completo
func (r *Retrieval) FindSimilarColors(target color.RGB, opts similarity.SearchOptions) ([]SimilarResult, error)

// ComparePaints compara duas ou mais tintas
func (r *Retrieval) ComparePaints(paintIDs []int64) (Response, error)
```

### Pipeline de Recuperação (Process)

1. **Parse** — Extrair intenção, cor alvo e filtros da query
2. **Consulta DB** — Buscar tintas, fabricantes, linhas relevantes
3. **Equivalências** — Se aplicável, buscar equivalências na tabela `equivalences`
4. **Receitas** — Se aplicável, buscar ou calcular receitas de mistura
5. **Similaridade** — Se aplicável, calcular tintas similares via Delta E
6. **Estruturar** — Montar Response com todos os dados coletados
7. **Validar** — Marcar como estimativa se dados insuficientes no banco

### Integrações com Engines Anteriores

- `color.RGBToLab()` + `color.DeltaE2000()` — conversões e Delta E
- `mix.Engine.Mix()` + `mix.Engine.SuggestRecipe()` — cálculo de misturas
- `similarity.Engine.FindSimilar()` + `FindEquivalent()` — busca de similares

## O que faz

- Consulta o banco de dados como fonte primária
- Orquestra color, mix e similarity engines
- Estrutura respostas completas com dados verificados
- Marca respostas como estimativa quando dados insuficientes
- Fornece PaintInfo completo com todos os campos visuais

## O que NÃO faz

- Não gera respostas usando apenas conhecimento do modelo
- Não executa operações de escrita no banco
- Não baixa ou gerencia imagens
- Não processa linguagem natural (NLP) — recebe Query estruturada
- Não implementa chat/conversation — apenas retrieval pontual

## Dependências

- RF-006 (Color Converter) — conversões RGB/Lab/DeltaE
- RF-007 (Mix Engine) — cálculo de misturas
- RF-008 (Similarity Engine) — busca de tintas similares
- RF-001 (Database Schema) — tabelas `paints`, `paint_colors`, `manufacturers`, `product_lines`, `equivalences`, `recipes`

## Pré-requisitos

- Banco populado com tintas (RF-004)
- Swatches gerados (RF-005)
- Engines de cor, mistura e similaridade implementados

## Gatilhos

- Usuário busca tinta por nome/código
- Usuário quer saber equivalências entre fabricantes
- Usuário quer sugestão de mistura para uma cor
- Usuário quer comparar tintas
- Usuário pergunta sobre características de uma tinta

## Fluxo Esperado

```
Usuário: "Qual a equivalência da Citadel Macragge Blue?"
    ↓
Query{Intent: IntentFindEquivalent, Text: "Macragge Blue"}
    ↓
Retrieval.Process()
    ↓
1. FindPaintInfo("Macragge Blue") → PaintInfo da Citadel
2. FindEquivalencesFor(paintID) → lista de equivalentes
3. Busca similaridade via Delta E para complementar
    ↓
Response{
    Paints: [Macragge Blue info],
    Equivalences: [AK-xxx, Vallejo-yyy, ...],
    Similar: [...],
    Sources: ["paint_knowledge.db"],
    IsEstimate: false
}
```

## Estrutura Técnica

```
pkg/ai/
├── retrieval.go      // Engine principal
├── context.go        // Estruturas de Query/Response
└── retrieval_test.go // Testes
```

## Arquivos Envolvidos

- `pkg/ai/retrieval.go` — novo
- `pkg/ai/context.go` — novo
- `pkg/ai/retrieval_test.go` — novo
- `pkg/color/converter.go` — existente (uso)
- `pkg/color/deltae.go` — existente (uso)
- `pkg/mix/engine.go` — existente (uso)
- `pkg/similarity/engine.go` — existente (uso)

## Bibliotecas

- `database/sql` — acesso ao banco
- `fmt`, `strings`, `strconv` — parsing de queries
- Pacotes internos: `pkg/color`, `pkg/mix`, `pkg/similarity`

## Banco de Dados

Tabelas consultadas (somente leitura):
- `paints` + `paint_colors` — informações de tintas
- `manufacturers` — dados de fabricantes
- `product_lines` — linhas de produtos
- `equivalences` — equivalências cadastradas
- `recipes` + `recipe_ingredients` — receitas de mistura
- `finish_types`, `paint_types`, `coverage_types`, `opacity_types` — lookups

## Critérios de Aceite

- [ ] `Process()` orquestra pipeline completa
- [ ] `FindPaintInfo()` retorna PaintInfo completo com todos os campos
- [ ] `FindEquivalencesFor()` usa tabela `equivalences` + similaridade
- [ ] `SuggestMixes()` usa `mix.Engine.SuggestRecipe()`
- [ ] `FindSimilarColors()` usa `similarity.Engine.FindSimilar()`
- [ ] `ComparePaints()` compara tintas lado a lado
- [ ] Response marca `IsEstimate=true` quando dados insuficientes
- [ ] Testes unitários passam
- [ ] Performance: < 200ms para queries típicas

## Critérios de Qualidade

- Código limpo e bem documentado
- Erros tratados adequadamente
- Nenhum panic ou crash em dados ausentes
- Testes cobrem cenários: tinta encontrada, não encontrada, estimativa
- Segue padrões dos pacotes anteriores (construtor New*, métodos públicos)

## Exemplo de Uso

```go
r := ai.NewRetrieval(db)

// Buscar equivalências
query := ai.Query{
    Text:   "Macragge Blue equivalente",
    Intent: ai.IntentFindEquivalent,
}
resp, err := r.Process(query)
// resp.Equivalences contém tintas equivalentes
// resp.IsEstimate = false (dados do banco)

// Sugerir mistura
query := ai.Query{
    Text:        "misturar azul RGB(30, 60, 120)",
    Intent:      ai.IntentMixRecipe,
    TargetColor: &color.RGB{R: 30, G: 60, B: 120},
}
resp, err = r.Process(query)
// resp.Recipes contém receitas sugeridas
```

## Próximos Requisitos Dependentes

- RF-010 (Desktop Interface) — usará Retrieval para todas as queries do usuário
