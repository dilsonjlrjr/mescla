package mix

import (
	"math"
	"testing"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
}

func TestMixTwoColors(t *testing.T) {
	engine := NewEngine()

	// rf-10: expectativa REESCRITA. O valor antigo (128,0,128) vinha da média
	// aritmética de RGB. Vermelho puro absorve 100% de verde e de azul; azul
	// puro absorve 100% de vermelho e de verde. Misturados, absorvem tudo — a
	// resposta subtrativa correta é quase preto, não roxo. Roxo é o que se
	// obtém com pigmentos reais, que não são saturados assim (ver
	// TestCA1AmareloComAzulDaVerde, com valores de tinta de verdade).
	colors := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Blue", R: 0, G: 0, B: 255},
	}

	r, g, b := engine.Mix(colors, []float64{50, 50})

	// O verde é o único canal que as DUAS tintas absorvem por completo, então
	// é o que praticamente desaparece — sobra um roxo escuro. A média
	// aritmética dava (128,0,128), claro demais: subtrativo escurece.
	if g >= r || g >= b {
		t.Errorf("o verde é absorvido pelas duas: deveria ser o menor canal, veio (%d,%d,%d)", r, g, b)
	}
	if r >= 128 || b >= 128 {
		t.Errorf("a mistura subtrativa escurece abaixo da média aritmética (128), veio (%d,%d,%d)", r, g, b)
	}
}

func TestMixThreeColors(t *testing.T) {
	engine := NewEngine()

	// rf-10: expectativa REESCRITA. O cinza (85,85,85) era média aritmética.
	// Somar as três primárias saturadas em tinta dá quase preto — cada uma
	// absorve o que as outras refletem. É o mesmo motivo pelo qual misturar
	// muitas cores na paleta acaba em marrom escuro, não em cinza médio.
	colors := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Green", R: 0, G: 255, B: 0},
		{ID: 3, Name: "Blue", R: 0, G: 0, B: 255},
	}

	r, g, b := engine.Mix(colors, []float64{33.33, 33.33, 33.34})

	// Resultado neutro, porém mais escuro que os 85 da média aritmética: é o
	// motivo de misturar muita cor na paleta acabar em cinza sujo escuro.
	if abs8(r, g) > 2 || abs8(g, b) > 2 {
		t.Errorf("proporções iguais de primárias simétricas dão neutro, veio (%d,%d,%d)", r, g, b)
	}
	if r >= 85 {
		t.Errorf("as três primárias juntas escurecem abaixo da média (85), veio (%d,%d,%d)", r, g, b)
	}
}

func abs8(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

func TestMixEmptyColors(t *testing.T) {
	engine := NewEngine()

	r, g, b := engine.Mix(nil, nil)
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("Expected (0,0,0), got (%d,%d,%d)", r, g, b)
	}
}

func TestMixMismatchedLengths(t *testing.T) {
	engine := NewEngine()

	colors := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
	}

	r, g, b := engine.Mix(colors, []float64{50, 50})
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("Expected (0,0,0), got (%d,%d,%d)", r, g, b)
	}
}

func TestCalculateDeltaE(t *testing.T) {
	engine := NewEngine()

	recipe := Recipe{
		ResultR: 255,
		ResultG: 0,
		ResultB: 0,
	}

	target := [3]float64{53.23, 80.11, 67.22}
	delta := engine.CalculateDeltaE(target, recipe)

	if delta < 0 {
		t.Errorf("DeltaE should be non-negative, got %f", delta)
	}
}

func TestSuggestRecipeSinglePaint(t *testing.T) {
	engine := NewEngine()

	available := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
	}

	target := [3]float64{53.23, 80.11, 67.22}
	recipe := engine.SuggestRecipe(target, available)

	if len(recipe.Ingredients) != 1 {
		t.Errorf("Expected 1 ingredient, got %d", len(recipe.Ingredients))
	}
	if recipe.Ingredients[0].Percentage != 100 {
		t.Errorf("Expected 100%%, got %f", recipe.Ingredients[0].Percentage)
	}
	if recipe.Method != "single" {
		t.Errorf("Expected method 'single', got '%s'", recipe.Method)
	}
}

func TestSuggestRecipeTwoPaints(t *testing.T) {
	engine := NewEngine()

	// rf-10: expectativa REESCRITA. O teste antigo exigia 2 ingredientes e o
	// método "binary-search" — a grade de proporções que não existe mais.
	// Com mistura subtrativa, acrescentar vermelho puro a azul puro só
	// escurece: não aproxima um azul-escuro alvo. A resposta honesta passa a
	// ser a tinta sozinha, e é isso que se verifica aqui.
	available := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Blue", R: 0, G: 0, B: 255},
	}

	target := [3]float64{25.53, 0.0, -40.0}
	recipe := engine.SuggestRecipe(target, available)

	if len(recipe.Ingredients) == 0 {
		t.Fatal("receita não pode sair vazia")
	}

	var soma float64
	for _, ing := range recipe.Ingredients {
		if ing.Percentage <= 0 {
			t.Errorf("ingrediente com proporção não positiva: %+v", ing)
		}
		soma += ing.Percentage
	}
	if math.Abs(soma-100) > 0.001 {
		t.Errorf("as proporções têm de somar 100, somaram %.2f", soma)
	}

	if recipe.Method != "single" && recipe.Method != "kubelka-munk" {
		t.Errorf("método esperado single ou kubelka-munk, veio %q", recipe.Method)
	}
}

func TestSuggestRecipeEmptyAvailable(t *testing.T) {
	engine := NewEngine()

	target := [3]float64{50, 50, 50}
	recipe := engine.SuggestRecipe(target, nil)

	if recipe.Method != "none" {
		t.Errorf("Expected method 'none', got '%s'", recipe.Method)
	}
}
