package service

import (
	"strings"
	"testing"

	"paint-match-ai/api/domain/stock"
)

// Regressão do guardrail do rf-13 (2026-09-08): SuggestEquivalentFromPool e
// SuggestEquivalentFromStock montavam o DTO sem CrossBrand/Manufacturers, ao
// contrário de SuggestEquivalentRecipe e ResolverCorNoUniverso. A receita
// multi-marca vinda do estoque chegava à tela com crossBrand=false, e o botão
// Salvar — que RN11/M7 manda bloquear justamente nesse caso — ficava
// habilitado.
func TestRF13GuardrailPoolDoEstoqueMarcaCrossBrand(t *testing.T) {
	svc := newRF13TestService(t, `
		INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo'), (2, 'Citadel');
		INSERT INTO paints (id, manufacturer_id, code, name) VALUES (1, 1, '70.870', 'Goblin Green');
		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES (1, 120, 90, 60);
	`)

	// Duas marcas no pool, nenhuma acertando o alvo sozinha — o solver precisa
	// combinar, e a combinação é necessariamente cross-brand.
	pool := []stock.Paint{
		{ID: 101, ManufacturerID: 1, Manufacturer: "Vallejo", Name: "Vermelho", Code: "V1", R: 220, G: 30, B: 30},
		{ID: 102, ManufacturerID: 2, Manufacturer: "Citadel", Name: "Verde", Code: "C1", R: 20, G: 150, B: 90},
	}

	dto, err := svc.SuggestEquivalentFromPool(1, pool)
	if err != nil {
		t.Fatalf("SuggestEquivalentFromPool: %v", err)
	}

	marcas := map[int64]bool{}
	for _, ing := range dto.Ingredients {
		if ing.ManufacturerID == 0 || ing.Manufacturer == "" {
			t.Errorf("ingrediente %q sem fabricante: id=%d nome=%q", ing.Name, ing.ManufacturerID, ing.Manufacturer)
		}
		marcas[ing.ManufacturerID] = true
	}

	if len(marcas) > 1 {
		if !dto.CrossBrand {
			t.Errorf("receita com %d marcas distintas veio com crossBrand=false", len(marcas))
		}
		if len(dto.Manufacturers) != len(marcas) {
			t.Errorf("manufacturers = %#v, esperado %d nomes", dto.Manufacturers, len(marcas))
		}
	}
}

// Rodada 2 do guardrail: crossBrandInfo contava ManufacturerID 0 — fabricante
// DESCONHECIDO — como marca. A receita saía com crossBrand=true e um único
// nome em manufacturers, e a tela dizia "2 marcas" nomeando uma.
func TestRF13GuardrailFabricanteDesconhecidoNaoContaComoMarca(t *testing.T) {
	cross, nomes := crossBrandInfo([]RecipeIngredientDTO{
		{ManufacturerID: 0, Manufacturer: ""},
		{ManufacturerID: 7, Manufacturer: "Citadel"},
	})
	if cross {
		t.Errorf("crossBrand = true com uma marca conhecida só (nomes=%#v)", nomes)
	}
	if len(nomes) != 1 || nomes[0] != "Citadel" {
		t.Errorf("manufacturers = %#v, esperado [Citadel]", nomes)
	}

	cross, nomes = crossBrandInfo([]RecipeIngredientDTO{
		{ManufacturerID: 7, Manufacturer: "Citadel"},
		{ManufacturerID: 9, Manufacturer: "Vallejo"},
	})
	if !cross || len(nomes) != 2 {
		t.Errorf("duas marcas conhecidas deviam dar crossBrand=true com 2 nomes, veio %v/%#v", cross, nomes)
	}
}

// Rodada 2 do guardrail: ResolverCorNoUniverso resolvia sempre sem teto, então
// o alvo hex livre de T1 escapava do teto de 3 de M6/RN5.
func TestRF13GuardrailTetoValeNoCaminhoPorCor(t *testing.T) {
	svc := newRF13TestService(t, `
		INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo'), (2, 'Citadel');
		INSERT INTO paints (id, manufacturer_id, code, name) VALUES
			(1, 1, 'A', 'A'), (2, 1, 'B', 'B'), (3, 1, 'C', 'C'),
			(4, 2, 'D', 'D'), (5, 2, 'E', 'E'), (6, 2, 'F', 'F');
		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES
			(1, 250, 10, 10), (2, 10, 250, 10), (3, 10, 10, 250),
			(4, 250, 250, 10), (5, 10, 250, 250), (6, 250, 10, 250);
	`)

	semTeto, err := svc.ResolverCorNoUniverso(90, 110, 130, 0, false, false, 0)
	if err != nil {
		t.Fatalf("sem teto: %v", err)
	}
	comTeto, err := svc.ResolverCorNoUniverso(90, 110, 130, 0, false, false, 3)
	if err != nil {
		t.Fatalf("com teto: %v", err)
	}

	if len(comTeto.Ingredients) > 3 {
		t.Errorf("maxIngredients=3 devolveu %d ingredientes", len(comTeto.Ingredients))
	}
	if len(semTeto.Ingredients) < len(comTeto.Ingredients) {
		t.Errorf("sem teto (%d) não deveria render menos que com teto (%d)",
			len(semTeto.Ingredients), len(comTeto.Ingredients))
	}
}

// Rodada 2 do guardrail: sem candidatos no cross-brand, a mensagem culpava
// "essa marca" — não há marca nenhuma em jogo nesse caminho.
func TestRF13GuardrailMensagemSemCandidatosNoCrossBrand(t *testing.T) {
	svc := newRF13TestService(t, `
		INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo');
		INSERT INTO paints (id, manufacturer_id, code, name) VALUES (1, 1, '70.870', 'Goblin Green');
		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES (1, 74, 103, 65);
	`)

	_, err := svc.SuggestEquivalentRecipe(1, 0, 3)
	if err == nil {
		t.Fatal("catálogo só com a própria origem deveria falhar")
	}
	if strings.Contains(err.Error(), "essa marca") {
		t.Errorf("mensagem do cross-brand culpa uma marca inexistente: %q", err)
	}
}
