package mix

import (
	"strings"
	"testing"
)

func TestGenerateTipsDarkResultSuggestsLightener(t *testing.T) {
	recipe := Recipe{
		Ingredients: []Ingredient{
			{Paint: PaintInput{ID: 1, Name: "Salmão", R: 240, G: 130, B: 100}, Percentage: 70},
			{Paint: PaintInput{ID: 2, Name: "Branco", R: 255, G: 255, B: 255}, Percentage: 20},
			{Paint: PaintInput{ID: 3, Name: "Rosa Chá", R: 245, G: 170, B: 190}, Percentage: 10},
		},
		ResultR: 180, ResultG: 100, ResultB: 80, // bem mais escuro que o alvo abaixo
	}

	tips := GenerateTips(240, 170, 160, recipe)

	found := false
	for _, tip := range tips {
		if strings.Contains(tip, "escuro") && strings.Contains(tip, "Branco") {
			found = true
		}
	}
	if !found {
		t.Errorf("Esperava dica de clareador ('Branco'), got: %v", tips)
	}
}

func TestGenerateTipsCloseHueNoTip(t *testing.T) {
	// Alvo e resultado com hue muito próximo — não deve gerar dica de matiz.
	recipe := Recipe{
		Ingredients: []Ingredient{
			{Paint: PaintInput{ID: 1, Name: "Vermelho", R: 200, G: 50, B: 50}, Percentage: 100},
		},
		ResultR: 200, ResultG: 52, ResultB: 48,
	}

	tips := GenerateTips(200, 50, 50, recipe)

	for _, tip := range tips {
		if strings.HasPrefix(tip, "Muito ") && !strings.Contains(tip, "escuro") && !strings.Contains(tip, "claro") {
			t.Errorf("Não esperava dica de matiz com hue tão próximo, got: %v", tips)
		}
	}
}

func TestGenerateTipsSingleIngredientNoError(t *testing.T) {
	recipe := Recipe{
		Ingredients: []Ingredient{
			{Paint: PaintInput{ID: 1, Name: "Único", R: 100, G: 100, B: 100}, Percentage: 100},
		},
		ResultR: 100, ResultG: 100, ResultB: 100,
	}

	tips := GenerateTips(255, 0, 0, recipe)
	// Não deve dar panic nem retornar nil de forma inesperada; qualquer
	// quantidade de dicas (inclusive zero) é aceitável aqui.
	_ = tips
}

func TestGenerateTipsEmptyIngredients(t *testing.T) {
	tips := GenerateTips(100, 100, 100, Recipe{})
	if len(tips) != 0 {
		t.Errorf("Esperava lista vazia sem ingredientes, got: %v", tips)
	}
}

func TestGenerateTipsHueMismatchNamesIngredients(t *testing.T) {
	// Alvo rosado, resultado puxando pra laranja (Salmão dominante) — deve
	// sugerir aumentar o Rosa Chá (mais próximo do alvo) e reduzir o Salmão.
	recipe := Recipe{
		Ingredients: []Ingredient{
			{Paint: PaintInput{ID: 1, Name: "Salmão", R: 250, G: 140, B: 90}, Percentage: 80},
			{Paint: PaintInput{ID: 2, Name: "Rosa Chá", R: 245, G: 170, B: 200}, Percentage: 20},
		},
		ResultR: 249, ResultG: 146, ResultB: 112, // resultado puxado pro laranja
	}

	// Alvo real: bem mais rosado que o resultado calculado.
	tips := GenerateTips(240, 130, 170, recipe)

	found := false
	for _, tip := range tips {
		if strings.Contains(tip, "Rosa Chá") && strings.Contains(tip, "Salmão") {
			found = true
		}
	}
	if !found {
		t.Errorf("Esperava dica nomeando Rosa Chá e Salmão, got: %v", tips)
	}
}

func TestHueBucketName(t *testing.T) {
	cases := map[float64]string{
		0:   "vermelho",
		30:  "laranja",
		60:  "amarelo",
		120: "verde",
		180: "ciano",
		230: "azul",
		290: "roxo",
		330: "rosa",
	}
	for hue, want := range cases {
		got := hueBucketName(hue)
		if got != want {
			t.Errorf("hueBucketName(%v) = %q, want %q", hue, got, want)
		}
	}
}
