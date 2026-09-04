package ai

import (
	"testing"

	"paint-match-ai/internal/color"
	"paint-match-ai/internal/mix"
)

func TestQueryIntentConstants(t *testing.T) {
	intents := []QueryIntent{
		IntentFindSimilar,
		IntentFindEquivalent,
		IntentMixRecipe,
		IntentPaintInfo,
		IntentManufacturerInfo,
		IntentCompare,
		IntentGeneral,
	}
	for _, intent := range intents {
		if intent == "" {
			t.Error("Intent vazio")
		}
	}
}

func TestParsePaintIDs(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1,2,3", 3},
		{"1", 1},
		{"", 0},
		{"abc", 0},
		{"1, abc, 3", 2},
	}
	for _, tt := range tests {
		ids := parsePaintIDs(tt.input)
		if len(ids) != tt.want {
			t.Errorf("parsePaintIDs(%q) = %d ids, want %d", tt.input, len(ids), tt.want)
		}
	}
}

func TestSelectDiversePaints(t *testing.T) {
	paints := []mix.PaintInput{
		{ID: 1, Name: "White", R: 255, G: 255, B: 255},
		{ID: 2, Name: "Black", R: 0, G: 0, B: 0},
		{ID: 3, Name: "Red", R: 255, G: 0, B: 0},
		{ID: 4, Name: "Green", R: 0, G: 255, B: 0},
		{ID: 5, Name: "Blue", R: 0, G: 0, B: 255},
		{ID: 6, Name: "Yellow", R: 255, G: 255, B: 0},
		{ID: 7, Name: "Cyan", R: 0, G: 255, B: 255},
		{ID: 8, Name: "Magenta", R: 255, G: 0, B: 255},
	}

	// Seleciona 3 de 8
	selected := selectDiversePaints(paints, 3)
	if len(selected) != 3 {
		t.Errorf("selectDiversePaints retornou %d, esperado 3", len(selected))
	}

	// Seleciona 10 de 8 (retorna todos)
	all := selectDiversePaints(paints, 10)
	if len(all) != 8 {
		t.Errorf("selectDiversePaints retornou %d, esperado 8", len(all))
	}

	// Primeiro deve ser o primeiro da lista
	if selected[0].ID != 1 {
		t.Errorf("Primeiro selecionado ID=%d, esperado 1", selected[0].ID)
	}
}

func TestBuildSimilarOptions(t *testing.T) {
	r := &Retrieval{}

	maxDE := 5.0
	maxRes := 5
	mfgID := int64(2)
	filters := QueryFilters{
		MaxDeltaE:      &maxDE,
		MaxResults:     &maxRes,
		ManufacturerID: &mfgID,
	}

	opts := r.buildSimilarOptions(filters)

	if opts.MaxDeltaE != 5.0 {
		t.Errorf("MaxDeltaE = %f, esperado 5.0", opts.MaxDeltaE)
	}
	if opts.MaxResults != 5 {
		t.Errorf("MaxResults = %d, esperado 5", opts.MaxResults)
	}
	if opts.ManufacturerID == nil || *opts.ManufacturerID != 2 {
		t.Errorf("ManufacturerID = %v, esperado 2", opts.ManufacturerID)
	}
}

func TestResponseStructure(t *testing.T) {
	resp := Response{
		Intent:    IntentPaintInfo,
		Paints:    []PaintInfo{{ID: 1, Name: "Test"}},
		Sources:   []string{"paint_knowledge.db"},
		IsEstimate: false,
	}

	if resp.Intent != IntentPaintInfo {
		t.Errorf("Intent = %s, esperado %s", resp.Intent, IntentPaintInfo)
	}
	if len(resp.Paints) != 1 {
		t.Errorf("Paints len = %d, esperado 1", len(resp.Paints))
	}
	if resp.IsEstimate {
		t.Error("IsEstimate deveria ser false")
	}
}

func TestPaintInfoLabCalculation(t *testing.T) {
	// Preto puro: Lab ≈ (0, 0, 0)
	l, a, b := color.RGBToLab(0, 0, 0)
	if l > 1 || a > 1 || b > 1 {
		t.Errorf("Preto RGB(0,0,0) Lab=(%f,%f,%f), esperado próximo de (0,0,0)", l, a, b)
	}

	// Branco puro: Lab ≈ (100, 0, 0)
	l, a, b = color.RGBToLab(255, 255, 255)
	if l < 95 {
		t.Errorf("Branco RGB(255,255,255) Lab L=%f, esperado ~100", l)
	}
}
