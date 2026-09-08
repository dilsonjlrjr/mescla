package service

import (
	"errors"
	"fmt"
	"sort"

	"paint-match-ai/api/domain/color"
	"paint-match-ai/api/domain/equivalence"
	"paint-match-ai/api/domain/mix"
	"paint-match-ai/api/domain/stock"
)

// BrandBestDTO é a tinta mais próxima de uma cor-alvo dentro de UMA marca —
// uma linha do "não sai nesta marca → veja nestas" do fluxo Mesclar.
type BrandBestDTO struct {
	ManufacturerID int64   `json:"manufacturerId"`
	Manufacturer   string  `json:"manufacturer"`
	PaintID        int64   `json:"paintId"`
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	R              uint8   `json:"r"`
	G              uint8   `json:"g"`
	B              uint8   `json:"b"`
	DeltaE         float64 `json:"deltaE"`
}

// CompareToAnchor calcula o ΔE de cada tinta de ids em relação à tinta âncora,
// ordenado da mais próxima à mais distante. Espelha o compareToAnchor do WASM
// (mesma regra, agora contra o catálogo do banco em vez do catálogo em memória).
func (s *PaintService) CompareToAnchor(anchorID int64, ids []int64) ([]SearchResultDTO, error) {
	anchor, err := s.GetPaintByID(anchorID)
	if err != nil {
		return nil, fmt.Errorf("tinta âncora não encontrada: %w", err)
	}
	anchorL, anchorA, anchorB := color.RGBToLab(anchor.R, anchor.G, anchor.B)
	anchorLab := [3]float64{anchorL, anchorA, anchorB}

	results := make([]SearchResultDTO, 0, len(ids))
	for _, id := range ids {
		if id == anchorID {
			continue
		}
		p, err := s.GetPaintByID(id)
		if err != nil {
			continue
		}
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		delta := color.DeltaE2000(anchorLab, [3]float64{l, a, b})
		results = append(results, SearchResultDTO{
			PaintID:      p.ID,
			Name:         p.Name,
			Manufacturer: p.Manufacturer,
			R:            p.R,
			G:            p.G,
			B:            p.B,
			DeltaE:       delta,
			SwatchPath:   p.SwatchPath,
			Similarity:   100.0 - delta,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DeltaE < results[j].DeltaE })
	return results, nil
}

// BestBrandsFor devolve, para cada OUTRA marca do catálogo, a tinta única mais
// próxima da cor de origem — uma passada em todo o catálogo. Espelha o
// bestBrandsFor do WASM.
func (s *PaintService) BestBrandsFor(paintID int64) ([]BrandBestDTO, error) {
	source, err := s.GetPaintByID(paintID)
	if err != nil {
		return nil, fmt.Errorf("tinta não encontrada: %w", err)
	}
	sourceL, sourceA, sourceB := color.RGBToLab(source.R, source.G, source.B)
	sourceLab := [3]float64{sourceL, sourceA, sourceB}

	all, err := s.GetAllPaints()
	if err != nil {
		return nil, err
	}

	mfrID := make(map[string]int64)
	best := make(map[string]BrandBestDTO)
	for _, p := range all {
		if p.ID == source.ID {
			continue
		}
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		delta := color.DeltaE2000(sourceLab, [3]float64{l, a, b})
		cur, seen := best[p.Manufacturer]
		if !seen || delta < cur.DeltaE {
			best[p.Manufacturer] = BrandBestDTO{
				Manufacturer: p.Manufacturer,
				PaintID:      p.ID,
				Name:         p.Name,
				Code:         p.Code,
				R:            p.R,
				G:            p.G,
				B:            p.B,
				DeltaE:       delta,
			}
		}
	}
	// Resolve manufacturerId (best[] só tem o nome, vindo do JOIN de GetAllPaints).
	if len(best) > 0 {
		rows, err := s.db.Query("SELECT id, name FROM manufacturers")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var name string
				if rows.Scan(&id, &name) == nil {
					mfrID[name] = id
				}
			}
		}
	}

	results := make([]BrandBestDTO, 0, len(best))
	for name, b := range best {
		b.ManufacturerID = mfrID[name]
		results = append(results, b)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DeltaE < results[j].DeltaE })
	return results, nil
}

// SuggestEquivalentFromPool monta a receita da tinta de origem (do catálogo)
// usando um pool de tintas fornecido pelo cliente — o "priorize o que eu
// tenho" do app mobile, cujo estoque vive no localStorage do navegador, não
// no banco do servidor (ao contrário do desktop, que tem SuggestEquivalentFromStock
// lendo user_paints). Mesma regra de negócio (api/domain/equivalence), fonte do
// pool diferente.
func (s *PaintService) SuggestEquivalentFromPool(sourcePaintID int64, pool []stock.Paint) (EquivalentRecipeDTO, error) {
	source, err := s.GetPaintByID(sourcePaintID)
	if err != nil {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não encontrada: %w", err)
	}
	if source.R == 0 && source.G == 0 && source.B == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não possui dados de cor cadastrados")
	}

	mixPool := stock.ToMixInputs(pool)
	if len(mixPool) == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("estoque vazio — cadastre tintas primeiro")
	}

	sourceInput := mix.PaintInput{ID: source.ID, Name: source.Name, Code: source.Code, R: source.R, G: source.G, B: source.B}
	res, err := equivalence.SuggestFromStock(sourceInput, mixPool)
	if err != nil {
		if errors.Is(err, equivalence.ErrNoCandidates) {
			return EquivalentRecipeDTO{}, fmt.Errorf("estoque vazio — cadastre tintas primeiro")
		}
		return EquivalentRecipeDTO{}, err
	}
	recipe := res.Recipe

	ingredients := mapIngredients(recipe.Ingredients)
	crossBrand, manufacturers := crossBrandInfo(ingredients)

	return EquivalentRecipeDTO{
		SourcePaintID:      source.ID,
		SourceName:         source.Name,
		SourceManufacturer: source.Manufacturer,
		SourceR:            source.R,
		SourceG:            source.G,
		SourceB:            source.B,
		TargetManufacturer: "Meu estoque",
		Ingredients:        ingredients,
		ResultR:            recipe.ResultR,
		ResultG:            recipe.ResultG,
		ResultB:            recipe.ResultB,
		DeltaE:             recipe.DeltaE,
		Method:             recipe.Method,
		Reproducible:       res.Reproducible,
		Tips:               res.Tips,
		CrossBrand:         crossBrand,
		Manufacturers:      manufacturers,
	}, nil
}

// ValidateStockCSV roda a mesma crítica de ImportUserPaintsCSV (fabricante
// precisa existir no catálogo, hex válido, nome obrigatório) SEM persistir —
// usada pelo app mobile, cujo estoque não vive no banco do servidor.
func (s *PaintService) ValidateStockCSV(csvText string) ([]stock.Paint, []stock.RowError, error) {
	mfrs, err := s.loadStockManufacturers()
	if err != nil {
		return nil, nil, err
	}
	paints, rowErrs := stock.ParseCSV(csvText, mfrs)
	if paints == nil {
		paints = []stock.Paint{}
	}
	if rowErrs == nil {
		rowErrs = []stock.RowError{}
	}
	return paints, rowErrs, nil
}

// StockToCSV serializa um pool de tintas fornecido pelo cliente no formato de
// importação — puro (sem banco), reexportado aqui para o handler HTTP não
// precisar importar api/domain/stock diretamente.
func StockToCSV(paints []stock.Paint) string {
	return stock.ToCSV(paints)
}
