package stock

import "testing"

// Regressão do guardrail do rf-13 (2026-09-08): ToMixInputs descartava o
// fabricante, então todo ingrediente de receita montada com o estoque chegava
// à tela com manufacturerId 0 e manufacturer vazio — o universo "só o que eu
// tenho" perdia a marca (M5/RN12) e o check de posse alternava o id 0.
func TestToMixInputsCarregaOFabricante(t *testing.T) {
	in := []Paint{
		{ID: 7, ManufacturerID: 42, Manufacturer: "Citadel", Name: "Castellan Green", Code: "CG", R: 74, G: 103, B: 65},
		{ID: 8, ManufacturerID: 9, Manufacturer: "Vallejo", Name: "Goblin Green", Code: "70.870", R: 80, G: 110, B: 70},
	}

	out := ToMixInputs(in)

	if len(out) != 2 {
		t.Fatalf("esperava 2 entradas, veio %d", len(out))
	}
	for i, p := range out {
		if p.ManufacturerID != in[i].ManufacturerID {
			t.Errorf("entrada %d: ManufacturerID = %d, esperado %d", i, p.ManufacturerID, in[i].ManufacturerID)
		}
		if p.Manufacturer != in[i].Manufacturer {
			t.Errorf("entrada %d: Manufacturer = %q, esperado %q", i, p.Manufacturer, in[i].Manufacturer)
		}
		if p.ID != in[i].ID {
			t.Errorf("entrada %d: ID do estoque = %d, esperado %d", i, p.ID, in[i].ID)
		}
	}
}
