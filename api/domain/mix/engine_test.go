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

	colors := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Blue", R: 0, G: 0, B: 255},
	}

	r, g, b := engine.Mix(colors, []float64{50, 50})
	if r != 128 || g != 0 || b != 128 {
		t.Errorf("Expected purple (128,0,128), got (%d,%d,%d)", r, g, b)
	}
}

func TestMixThreeColors(t *testing.T) {
	engine := NewEngine()

	colors := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Green", R: 0, G: 255, B: 0},
		{ID: 3, Name: "Blue", R: 0, G: 0, B: 255},
	}

	r, g, b := engine.Mix(colors, []float64{33.33, 33.33, 33.34})
	if r != 85 || g != 85 || b != 85 {
		t.Errorf("Expected gray (85,85,85), got (%d,%d,%d)", r, g, b)
	}
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

	available := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "Blue", R: 0, G: 0, B: 255},
	}

	target := [3]float64{25.53, 0.0, -40.0}
	recipe := engine.SuggestRecipe(target, available)

	if len(recipe.Ingredients) != 2 {
		t.Errorf("Expected 2 ingredients, got %d", len(recipe.Ingredients))
	}
	if recipe.Method != "binary-search" {
		t.Errorf("Expected method 'binary-search', got '%s'", recipe.Method)
	}

	sum := 0.0
	for _, ing := range recipe.Ingredients {
		sum += ing.Percentage
	}
	if math.Abs(sum-100) > 1 {
		t.Errorf("Percentages should sum to 100, got %f", sum)
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
