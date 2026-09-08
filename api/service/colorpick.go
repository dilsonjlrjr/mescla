package service

import (
	"fmt"
)

// PickColorRequest é o input do color pick (usado internamente, exposto como parâmetros).
// R, G, B vêm como uint8 — o type system de Go já garante 0-255.

// ColorMatchDTO é uma tinta equivalente encontrada para a cor detectada.
type ColorMatchDTO struct {
	PaintID      int64   `json:"paintId"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	Code         string  `json:"code"`
	R            uint8   `json:"r"`
	G            uint8   `json:"g"`
	B            uint8   `json:"b"`
	Hex          string  `json:"hex"`
	DeltaE       float64 `json:"deltaE"`
}

// PickColorResponse é o resultado do color pick.
type PickColorResponse struct {
	Hex     string               `json:"hex"`
	Matches []ColorMatchDTO      `json:"matches"`
	Recipe  *EquivalentRecipeDTO `json:"recipe,omitempty"`
}

// rgbToHex converts uint8 RGB to #RRGGBB string.
func rgbToHexStr(r, g, b uint8) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// PickColor recebe RGB de um pixel e retorna as tintas mais próximas no catálogo.
// Se targetManufacturerID > 0, também gera receita de mistura para o melhor match.
func (s *PaintService) PickColor(r, g, b uint8, targetManufacturerID int64) (PickColorResponse, error) {
	resp := PickColorResponse{
		Hex: rgbToHexStr(r, g, b),
	}

	// Busca similares no catálogo
	results, err := s.FindSimilar(r, g, b, 50, 5)
	if err != nil {
		return resp, fmt.Errorf("buscando similares: %w", err)
	}

	// Converte SearchResultDTO → ColorMatchDTO (adiciona Code e Hex)
	for _, sr := range results {
		// Busca o código da tinta
		code := ""
		paint, err := s.GetPaintByID(sr.PaintID)
		if err == nil {
			code = paint.Code
		}

		resp.Matches = append(resp.Matches, ColorMatchDTO{
			PaintID:      sr.PaintID,
			Name:         sr.Name,
			Manufacturer: sr.Manufacturer,
			Code:         code,
			R:            sr.R,
			G:            sr.G,
			B:            sr.B,
			Hex:          rgbToHexStr(sr.R, sr.G, sr.B),
			DeltaE:       sr.DeltaE,
		})
	}

	// Se pediu receita, gera para o melhor match
	if targetManufacturerID > 0 && len(resp.Matches) > 0 {
		// Valida fabricante
		var mfrName string
		err := s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", targetManufacturerID).Scan(&mfrName)
		if err != nil {
			return resp, fmt.Errorf("fabricante não encontrado (id %d)", targetManufacturerID)
		}

		// Tenta gerar receita para o melhor match
		bestMatch := resp.Matches[0]
		recipe, err := s.SuggestEquivalentRecipe(bestMatch.PaintID, targetManufacturerID, 0)
		if err != nil {
			// Receita pode falhar (ex: fabricante sem tintas compatíveis) —
			// retornamos sem receita, não é erro fatal
			resp.Recipe = nil
		} else {
			resp.Recipe = &recipe
		}
	}

	return resp, nil
}
