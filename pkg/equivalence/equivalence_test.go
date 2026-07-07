package equivalence

import (
	"errors"
	"testing"

	"paint-match-ai/pkg/mix"
)

func TestSuggestExcludesSourcePaint(t *testing.T) {
	// A tinta-alvo (ID 1) está no pool; sem a exclusão a receita seria 100%
	// dela mesma com ΔE 0.
	source := mix.PaintInput{ID: 1, Name: "Azul Marinho", R: 20, G: 30, B: 80}
	candidates := []mix.PaintInput{
		source,
		{ID: 2, Name: "Preto", R: 10, G: 10, B: 10},
		{ID: 3, Name: "Azul", R: 30, G: 60, B: 180},
	}

	res, err := Suggest(source, "Acrilex", "Acrilex", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ing := range res.Recipe.Ingredients {
		if ing.Paint.ID == source.ID {
			t.Fatalf("a tinta-alvo (ID %d) entrou como ingrediente", source.ID)
		}
	}
}

func TestSuggestSameManufacturerForcesMix(t *testing.T) {
	// O cenário-título da feature: a marca não tem azul-marinho pronto, mas
	// tem preto e azul — na mesma marca a resposta é a mistura, nunca 1 tinta.
	source := mix.PaintInput{ID: 99, Name: "Azul Marinho", R: 20, G: 30, B: 80}
	candidates := []mix.PaintInput{
		{ID: 1, Name: "Black", R: 10, G: 10, B: 10},
		{ID: 2, Name: "Blue", R: 30, G: 60, B: 200},
		{ID: 3, Name: "Red", R: 255, G: 0, B: 0},
	}

	res, err := Suggest(source, "Vallejo", "Vallejo", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Recipe.Ingredients) < 2 {
		t.Fatalf("mesma marca deveria forçar 2+ ingredientes, veio %d", len(res.Recipe.Ingredients))
	}
}

func TestSuggestSameManufacturerCollapsesToTwin(t *testing.T) {
	// Se a marca tem OUTRA tinta de cor idêntica, a mistura "forçada" colapsa
	// pra ela (proporção 0% é descartada) — a resposta honesta é "use essa".
	source := mix.PaintInput{ID: 99, Name: "Branco Alvo", R: 255, G: 255, B: 255}
	candidates := []mix.PaintInput{
		{ID: 1, Name: "White Twin", R: 255, G: 255, B: 255},
		{ID: 2, Name: "Black", R: 0, G: 0, B: 0},
		{ID: 3, Name: "Red", R: 255, G: 0, B: 0},
	}

	res, err := Suggest(source, "Vallejo", "Vallejo", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Recipe.Ingredients) != 1 || res.Recipe.Ingredients[0].Paint.ID != 1 {
		t.Fatalf("esperava colapso pra White Twin (100%%), veio %+v", res.Recipe.Ingredients)
	}
	for _, ing := range res.Recipe.Ingredients {
		if ing.Percentage <= 0 {
			t.Fatalf("ingrediente com 0%% vazou: %s", ing.Paint.Name)
		}
	}
}

func TestSuggestDifferentManufacturerAllowsSingle(t *testing.T) {
	// Entre marcas diferentes a equivalência 1:1 é a resposta certa quando
	// existe uma tinta idêntica.
	source := mix.PaintInput{ID: 99, Name: "Branco Alvo", R: 255, G: 255, B: 255}
	candidates := []mix.PaintInput{
		{ID: 1, Name: "White", R: 255, G: 255, B: 255},
		{ID: 2, Name: "Black", R: 0, G: 0, B: 0},
	}

	res, err := Suggest(source, "Citadel", "Vallejo", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Recipe.Ingredients) != 1 {
		t.Fatalf("esperava equivalência 1:1, veio %d ingredientes", len(res.Recipe.Ingredients))
	}
	if res.Recipe.Ingredients[0].Paint.ID != 1 {
		t.Fatalf("esperava a tinta branca (ID 1), veio ID %d", res.Recipe.Ingredients[0].Paint.ID)
	}
	if !res.Reproducible {
		t.Fatal("ΔE 0 deveria ser reproduzível")
	}
}

func TestSuggestReproducibleFlag(t *testing.T) {
	// Alvo vermelho saturado com pool só de cinzas: melhor tentativa fica longe
	// (ΔE > MaxViableDeltaE) e a flag deve refletir isso.
	source := mix.PaintInput{ID: 99, Name: "Vermelho Vivo", R: 230, G: 20, B: 20}
	candidates := []mix.PaintInput{
		{ID: 1, Name: "Cinza Claro", R: 200, G: 200, B: 200},
		{ID: 2, Name: "Cinza Escuro", R: 60, G: 60, B: 60},
	}

	res, err := Suggest(source, "Citadel", "Talento", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reproducible {
		t.Fatalf("ΔE %.1f não deveria ser reproduzível (limite %.1f)", res.Recipe.DeltaE, MaxViableDeltaE)
	}
	if len(res.Recipe.Ingredients) == 0 {
		t.Fatal("mesmo irreproduzível, a melhor tentativa deve vir preenchida")
	}
}

func TestSuggestNoCandidates(t *testing.T) {
	// Pool contém só a própria tinta-alvo: após a exclusão não sobra nada.
	source := mix.PaintInput{ID: 1, Name: "Única", R: 100, G: 100, B: 100}
	candidates := []mix.PaintInput{source}

	_, err := Suggest(source, "Corfix", "Corfix", candidates)
	if !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("esperava ErrNoCandidates, veio %v", err)
	}
}

func TestSuggestGeneratesTips(t *testing.T) {
	// Alvo saturado com pool só de neutros: a mistura sai acinzentada e a dica
	// de "falta pigmento no catálogo" deve disparar (condição de tips.go).
	source := mix.PaintInput{ID: 99, Name: "Vermelho Vivo", R: 230, G: 20, B: 20}
	candidates := []mix.PaintInput{
		{ID: 1, Name: "Cinza Claro", R: 200, G: 200, B: 200},
		{ID: 2, Name: "Cinza Escuro", R: 60, G: 60, B: 60},
	}

	res, err := Suggest(source, "Citadel", "Talento", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Tips) == 0 {
		t.Fatal("esperava a dica de pigmento faltante para alvo saturado com pool neutro")
	}
}

func TestSuggestFromStockAllowsSingleTwin(t *testing.T) {
	// O estoque já tem a cor exata: a resposta certa é "use essa tinta" (1:1),
	// não uma mistura forçada — e a origem NÃO é excluída por ID.
	source := mix.PaintInput{ID: 2, Name: "Branco Alvo", R: 255, G: 255, B: 255}
	stock := []mix.PaintInput{
		{ID: 2, Name: "Meu Branco", R: 255, G: 255, B: 255}, // ID coincide com a origem de propósito
		{ID: 7, Name: "Meu Preto", R: 0, G: 0, B: 0},
	}

	res, err := SuggestFromStock(source, stock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Reproducible {
		t.Fatalf("branco idêntico no estoque deveria ser reproduzível (ΔE %.2f)", res.Recipe.DeltaE)
	}
	if len(res.Recipe.Ingredients) != 1 || res.Recipe.Ingredients[0].Paint.ID != 2 {
		t.Fatalf("esperava 1:1 com a tinta do estoque (ID 2), veio %+v", res.Recipe.Ingredients)
	}
}

func TestSuggestFromStockEmpty(t *testing.T) {
	source := mix.PaintInput{ID: 1, Name: "Qualquer", R: 100, G: 100, B: 100}
	if _, err := SuggestFromStock(source, nil); !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("estoque vazio deveria dar ErrNoCandidates, veio %v", err)
	}
}
