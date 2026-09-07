package mix

import (
	"testing"

	"paint-match-ai/api/domain/color"
)

func TestSuggestBestSubsetSingleCandidate(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Salmão", R: 240, G: 130, B: 100},
	}

	recipe := SuggestBestSubset([3]float64{60, 30, 20}, candidates, 1, 3)

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
	recipe := SuggestBestSubset(target, candidates, 1, 3)

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
	recipe := SuggestBestSubset([3]float64{targetL, targetA, targetB}, candidates, 1, 1)

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

	recipe := SuggestBestSubset([3]float64{50, 10, 10}, candidates, 1, 3)

	if len(recipe.Ingredients) == 0 {
		t.Fatal("Expected a recipe even with many candidates")
	}
	if len(recipe.Ingredients) > 3 {
		t.Errorf("Expected at most 3 ingredients, got %d", len(recipe.Ingredients))
	}
}

func TestSuggestBestSubsetEmptyCandidates(t *testing.T) {
	recipe := SuggestBestSubset([3]float64{50, 50, 50}, nil, 1, 3)

	if recipe.Method != "none" {
		t.Errorf("Expected method 'none', got '%s'", recipe.Method)
	}
}

// minIngredients=2 (caso mesma marca) deve obrigar mistura de verdade quando
// nenhuma tinta sozinha resolve.
func TestSuggestBestSubsetMinIngredientsForcesMix(t *testing.T) {
	// Alvo azul-marinho com pool sem nada parecido pronto: a resposta é uma
	// mistura real (preto + azul), nunca uma tinta só.
	candidates := []PaintInput{
		{ID: 1, Name: "Black", R: 10, G: 10, B: 10},
		{ID: 2, Name: "Blue", R: 30, G: 60, B: 200},
		{ID: 3, Name: "Red", R: 255, G: 0, B: 0},
	}
	// azul-marinho (20, 30, 80) em Lab
	l, a, b := 14.0, 15.0, -35.0

	recipe := SuggestBestSubset([3]float64{l, a, b}, candidates, 2, 3)

	if len(recipe.Ingredients) < 2 {
		t.Fatalf("Expected at least 2 ingredients with minIngredients=2, got %d", len(recipe.Ingredients))
	}
	for _, ing := range recipe.Ingredients {
		if ing.Percentage <= 0 {
			t.Fatalf("ingrediente com %.1f%% não deveria aparecer na receita", ing.Percentage)
		}
	}
}

// Proporções 0% nunca aparecem na receita; se outra tinta do pool tem a cor
// exata do alvo, a mistura "forçada" colapsa pra ela — resposta honesta.
func TestSuggestBestSubsetDropsZeroPercentages(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "White", R: 255, G: 255, B: 255},
		{ID: 2, Name: "Black", R: 0, G: 0, B: 0},
		{ID: 3, Name: "Red", R: 255, G: 0, B: 0},
	}

	recipe := SuggestBestSubset([3]float64{100, 0, 0}, candidates, 2, 3)

	for _, ing := range recipe.Ingredients {
		if ing.Percentage <= 0 {
			t.Fatalf("ingrediente com 0%% vazou pra receita: %s", ing.Paint.Name)
		}
	}
	if len(recipe.Ingredients) != 1 || recipe.Ingredients[0].Paint.ID != 1 {
		t.Fatalf("esperava colapso honesto pra White (100%%), veio %+v", recipe.Ingredients)
	}
}

// Piso maior que o pool não deve travar: rebaixa pro que houver.
func TestSuggestBestSubsetMinClampedToPool(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Only", R: 120, G: 120, B: 120},
	}

	recipe := SuggestBestSubset([3]float64{50, 0, 0}, candidates, 2, 3)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("Expected 1 ingredient (pool tem só 1), got %d", len(recipe.Ingredients))
	}
}

// Com piso de 2 ingredientes e nenhuma tinta que acerte o alvo sozinha, a
// receita sai como mistura de verdade — é o caso da equivalência dentro da
// mesma marca, onde devolver "use 100% da tinta X" não ajuda ninguém.
func TestSuggestBestSubsetForcaMisturaQuandoNenhumaTintaAcerta(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Amarelo", R: 0xF2, G: 0xE2, B: 0x00},
		{ID: 2, Name: "Azul", R: 0x3F, G: 0x48, B: 0xCC},
		{ID: 3, Name: "Branco", R: 255, G: 255, B: 255},
	}

	// Um verde-oliva, longe de qualquer uma das três.
	alvo := [3]float64{50, -20, 30}
	recipe := SuggestBestSubset(alvo, candidates, 2, 0)

	if len(recipe.Ingredients) < 2 {
		t.Fatalf("piso de 2 ingredientes não foi respeitado: %+v", recipe.Ingredients)
	}
	var soma float64
	for _, ing := range recipe.Ingredients {
		if ing.Percentage <= 0 {
			t.Errorf("ingrediente com proporção não positiva: %+v", ing)
		}
		soma += ing.Percentage
	}
	if soma < 99.9 || soma > 100.1 {
		t.Errorf("as proporções têm de somar 100, somaram %.2f", soma)
	}
}

// O piso de ingredientes nunca piora a cor a ponto de valer a pena: se a
// mistura forçada ficar sensivelmente pior que a tinta sozinha, vence a tinta.
func TestSuggestBestSubsetNaoPioraACorParaCumprirOPiso(t *testing.T) {
	exata := PaintInput{ID: 1, Name: "Exata", R: 0x4A, G: 0x7C, B: 0x2F}
	candidates := []PaintInput{
		exata,
		{ID: 2, Name: "Preto", R: 0, G: 0, B: 0},
		{ID: 3, Name: "Branco", R: 255, G: 255, B: 255},
	}

	l, a, b := color.RGBToLab(exata.R, exata.G, exata.B)
	recipe := SuggestBestSubset([3]float64{l, a, b}, candidates, 2, 0)

	if len(recipe.Ingredients) != 1 || recipe.Ingredients[0].Paint.ID != exata.ID {
		t.Fatalf("com a tinta exata no pool, o piso não pode forçar mistura: %+v", recipe.Ingredients)
	}
}

// Teto de ingredientes vindo de quem chama continua valendo.
func TestSuggestBestSubsetRespeitaTetoDeIngredientes(t *testing.T) {
	candidates := make([]PaintInput, 0, 20)
	for i := 0; i < 20; i++ {
		candidates = append(candidates, PaintInput{
			ID:   int64(i + 1),
			Name: string(rune('A' + i)),
			R:    uint8((i * 37) % 256),
			G:    uint8((i * 91) % 256),
			B:    uint8((i * 143) % 256),
		})
	}

	recipe := SuggestBestSubset([3]float64{50, -20, 30}, candidates, 1, 3)

	if len(recipe.Ingredients) > 3 {
		t.Fatalf("teto de 3 ingredientes desrespeitado: %d", len(recipe.Ingredients))
	}
}

// Pool vazio não estoura.
func TestSuggestBestSubsetPoolVazio(t *testing.T) {
	if rec := SuggestBestSubset([3]float64{50, 0, 0}, nil, 1, 0); rec.Method != "none" {
		t.Fatalf("pool vazio deveria devolver Method none, veio %q", rec.Method)
	}
}

// Caminho da mistura forçada: o solver devolve uma tinta só (misturar preto em
// vermelho só escurece, não aproxima um vermelho mais claro), o piso de 2
// ingredientes tenta forçar, e a mistura forçada é RECUSADA por piorar a cor.
// A resposta honesta continua sendo a tinta sozinha.
func TestMisturaForcadaERecusadaQuandoPioraACor(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Vermelho", R: 0xD8, G: 0x1E, B: 0x05},
		{ID: 2, Name: "Preto", R: 0, G: 0, B: 0},
	}

	// Um vermelho bem mais claro que a tinta: nenhuma das duas acerta, e
	// escurecer com preto afasta ainda mais.
	l, a, b := color.RGBToLab(0xFF, 0x8A, 0x7A)
	recipe := SuggestBestSubset([3]float64{l, a, b}, candidates, 2, 0)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("forçar mistura que piora a cor não vale: %+v", recipe.Ingredients)
	}
	if recipe.Ingredients[0].Paint.Name != "Vermelho" {
		t.Fatalf("a tinta mais próxima é o vermelho, veio %q", recipe.Ingredients[0].Paint.Name)
	}
}

// Mistura forçada ACEITA: com um branco no pool, clarear o vermelho aproxima
// de verdade um vermelho claro — a receita sai com dois componentes.
func TestMisturaForcadaAceitaQuandoMelhoraACor(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Vermelho", R: 0xD8, G: 0x1E, B: 0x05},
		{ID: 2, Name: "Branco", R: 255, G: 255, B: 255},
	}

	l, a, b := color.RGBToLab(0xEE, 0x9A, 0x8A)
	recipe := SuggestBestSubset([3]float64{l, a, b}, candidates, 2, 0)

	if len(recipe.Ingredients) < 2 {
		t.Fatalf("clarear com branco aproxima: esperava mistura, veio %+v", recipe.Ingredients)
	}
}

// O teto de ingredientes vale também dentro da mistura forçada: com piso 2 e
// teto 1, a receita sai com um componente só — o teto vence o piso.
func TestSuggestBestSubsetTetoVenceOPisoDeIngredientes(t *testing.T) {
	candidates := []PaintInput{
		{ID: 1, Name: "Vermelho", R: 0xD8, G: 0x1E, B: 0x05},
		{ID: 2, Name: "Branco", R: 255, G: 255, B: 255},
		{ID: 3, Name: "Preto", R: 0, G: 0, B: 0},
	}

	l, a, b := color.RGBToLab(0xEE, 0x9A, 0x8A)
	recipe := SuggestBestSubset([3]float64{l, a, b}, candidates, 2, 1)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("teto de 1 ingrediente tem de vencer o piso de 2, veio %d", len(recipe.Ingredients))
	}
}

// Piso maior que o pool não trava: rebaixa para o que houver.
func TestSuggestBestSubsetPisoMaiorQueOPool(t *testing.T) {
	candidates := []PaintInput{{ID: 1, Name: "Única", R: 100, G: 100, B: 100}}

	recipe := SuggestBestSubset([3]float64{50, 10, 10}, candidates, 3, 0)

	if len(recipe.Ingredients) != 1 {
		t.Fatalf("com uma tinta no pool a receita tem um ingrediente, veio %d", len(recipe.Ingredients))
	}
}

// Mistura forçada com teto que corta componentes: a receita respeita o teto e
// as proporções continuam somando 100.
func TestMisturaForcadaComTetoMantemProporcoesValidas(t *testing.T) {
	candidates := make([]PaintInput, 0, 8)
	for i := 0; i < 8; i++ {
		candidates = append(candidates, PaintInput{
			ID: int64(i + 1), Name: string(rune('A' + i)),
			R: uint8(30 + i*25), G: uint8(200 - i*20), B: uint8(60 + i*15),
		})
	}

	recipe := SuggestBestSubset([3]float64{45, 12, -8}, candidates, 2, 2)

	if len(recipe.Ingredients) > 2 {
		t.Fatalf("teto de 2 desrespeitado: %d", len(recipe.Ingredients))
	}
	var soma float64
	for _, ing := range recipe.Ingredients {
		soma += ing.Percentage
	}
	if soma < 99.9 || soma > 100.1 {
		t.Fatalf("proporções têm de somar 100, somaram %.2f", soma)
	}
}
