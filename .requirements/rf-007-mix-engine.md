# RF-007 — Mix Engine

## ID
RF-007

## Título
Mix Engine — Motor de mistura de tintas

## Status
Done

## Objetivo
Implementar motor de mistura de tintas para gerar receitas e calcular cores resultantes.

## Motivação
O sistema precisa sugerir misturas de tintas para alcançar cores desejadas, calcular proporções e prever cores resultantes.

## Escopo
- Mistura de 2 a 4 tintas
- Cálculo de proporções (peso ou volume)
- Previsão de cor resultante (mix aditivo RGB)
- Cálculo de precisão da mistura
- Geração de receitas com DeltaE
- Suporte a diferentes métodos de mistura

## O que faz
- Calcula cor resultante da mistura de N tintas
- Gera receitas com proporções percentuais
- Avalia precisão da mistura (DeltaE)
- Sugere ajustes para alcançar cor alvo

## O que não faz
- Não calcula similaridade entre tintas (RF-008)
- Não busca equivalências (RF-008)
- Não interage com IA (RF-009)
- Não modifica dados de tintas existentes

## Dependências
- RF-006 Color Converter (conversões e DeltaE)

## Pré-requisitos
- pkg/color implementado (RGB, CIELAB, DeltaE)
- Tabela recipes e recipe_ingredients no banco

## Gatilhos
- Quando usuário busca mistura para alcançar cor
- Quando sistema precisa prever cor resultante

## Fluxo esperado
1. Receber tintas disponíveis com suas cores (RGB)
2. Receber cor alvo (RGB ou CIELAB)
3. Calcular melhor combinação de proporções
4. Retornar receita com proporções e precisão

## Estrutura técnica
```go
// pkg/mix/engine.go
type PaintInput struct {
    ID    int64
    Name  string
    R, G, B uint8
}

type Recipe struct {
    Ingredients []Ingredient
    ResultR, ResultG, ResultB uint8
    DeltaE      float64
    Method      string
}

type Ingredient struct {
    Paint      PaintInput
    Percentage float64
}

type Engine struct{}

func NewEngine() *Engine
func (e *Engine) Mix(colors []PaintInput, weights []float64) (r, g, b uint8)
func (e *Engine) CalculateDeltaE(target [3]float64, recipe Recipe) float64
func (e *Engine) SuggestRecipe(target [3]float64, available []PaintInput) Recipe

// pkg/mix/proportions.go
func CalculateProportions(colors []PaintInput, target [3]float64) []float64
func ValidateProportions(proportions []float64) bool
func NormalizeProportions(proportions []float64) []float64
```

## Arquivos envolvidos
- `pkg/mix/engine.go` — motor de mistura
- `pkg/mix/proportions.go` — cálculo de proporções
- `pkg/mix/engine_test.go` — testes unitários

## Bibliotecas
- Nenhuma dependência externa (usa pkg/color)

## Banco de dados
- Leitura: paints, paint_colors
- Leitura/Escrita: recipes, recipe_ingredients

## Critérios de aceite
- [ ] Mistura de 2 tintas funciona
- [ ] Mistura de 3 tintas funciona
- [ ] Mistura de 4 tintas funciona
- [ ] Cálculo de proporções correto
- [ ] Previsão de cor resultante precisa
- [ ] Cálculo DeltaE da receita
- [ ] Normalização de proporções
- [ ] Testes unitários passam

## Critérios de qualidade
- Mistura aditiva RGB correta
- Proporções normalizadas somam 100%
- DeltaE < 5.0 para misturas ideais
- Código testável e sem efeitos colaterais

## Exemplo de uso
```go
engine := mix.NewEngine()

available := []mix.PaintInput{
    {ID: 1, Name: "Vermelho", R: 255, G: 0, B: 0},
    {ID: 2, Name: "Azul", R: 0, G: 0, B: 255},
    {ID: 3, Name: "Branco", R: 255, G: 255, B: 255},
}

// Mistura direta
r, g, b := engine.Mix(available, []float64{50, 30, 20})

// Sugerir receita para cor alvo
target := [3]float64{128, 0, 128} // Roxo
recipe := engine.SuggestRecipe(target, available)
// Resultado: {Vermelho: 50%, Azul: 50%, DeltaE: 2.3}
```

## Próximos requisitos dependentes
- RF-008 Similarity Engine
- RF-009 AI Retrieval
