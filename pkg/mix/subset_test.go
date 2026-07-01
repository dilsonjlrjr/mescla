package mix

import "testing"

func TestSuggestBestSubsetSingleCandidate(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Salmão", R: 240, G: 130, B: 100},
	}

	recipe := SuggestBestSubset([3]float64{60, 30, 20}, candidates, 3)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("Expected 1 ingredient, got %d", len(recipe.Ingredients))
	}
	if recipe.Method != "single" {
		t.Errorf("Expected method 'single', got '%s'", recipe.Method)
	}
}

func TestSuggestBestSubsetUsesAllWhenFew(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 2, Name: "White", R: 255, G: 255, B: 255},
		{ID: 3, Name: "Blue", R: 0, G: 0, B: 255},
	}

	target := [3]float64{50, 40, -10}
	recipe := SuggestBestSubset(target, candidates, 3)

	if len(recipe.Ingredients) == 0 {
		t.Fatal("Expected at least 1 ingredient")
	}
	if len(recipe.Ingredients) > 3 {
		t.Errorf("Expected at most 3 ingredients, got %d", len(recipe.Ingredients))
	}
}

// Regressão do bug original: com mais de 4 candidatos, o Engine.SuggestRecipe
// sozinho só varia os 4 primeiros do slice recebido. SuggestBestSubset deve
// escolher o MELHOR subconjunto dentre todos os candidatos, não só os 4 primeiros.
func TestSuggestBestSubsetPicksBestNotFirst(t *testing.T) {
	// Os 4 primeiros candidatos são todos pretos (péssima aproximação pro alvo
	// branco); o candidato de índice 4 é branco puro — só aparece se o algoritmo
	// de fato considerar combinações além dos 4 primeiros.
	candidates := []PaintInput{
		{ID: 1, Name: "Black A", R: 10, G: 10, B: 10},
		{ID: 2, Name: "Black B", R: 12, G: 12, B: 12},
		{ID: 3, Name: "Black C", R: 8, G: 8, B: 8},
		{ID: 4, Name: "Black D", R: 15, G: 15, B: 15},
		{ID: 5, Name: "White", R: 255, G: 255, B: 255},
	}

	targetL, targetA, targetB := 100.0, 0.0, 0.0 // branco em Lab
	recipe := SuggestBestSubset([3]float64{targetL, targetA, targetB}, candidates, 1)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("Expected 1 ingredient, got %d", len(recipe.Ingredients))
	}
	if recipe.Ingredients[0].Paint.ID != 5 {
		t.Errorf("Expected the white paint (ID 5) to win, got ID %d — algoritmo só considerou os primeiros candidatos", recipe.Ingredients[0].Paint.ID)
	}
}

func TestSuggestBestSubsetSafetyLimit(t *testing.T) {
	candidates := make([]PaintInput, 30)
	for i := range candidates {
		candidates[i] = PaintInput{
			ID:   int64(i + 1),
			Name: "Paint",
			R:    uint8((i * 8) % 256),
			G:    uint8((i * 16) % 256),
			B:    uint8((i * 32) % 256),
		}
	}

	recipe := SuggestBestSubset([3]float64{50, 10, 10}, candidates, 3)

	if len(recipe.Ingredients) == 0 {
		t.Fatal("Expected a recipe even with many candidates")
	}
	if len(recipe.Ingredients) > 3 {
		t.Errorf("Expected at most 3 ingredients, got %d", len(recipe.Ingredients))
	}
}

func TestSuggestBestSubsetEmptyCandidates(t *testing.T) {
	recipe := SuggestBestSubset([3]float64{50, 50, 50}, nil, 3)

	if recipe.Method != "none" {
		t.Errorf("Expected method 'none', got '%s'", recipe.Method)
	}
}
