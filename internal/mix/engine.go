package mix

import (
	"paint-match-ai/internal/color"
)

type PaintInput struct {
	ID      int64
	Name    string
	Code    string
	R, G, B uint8
}

type Ingredient struct {
	Paint      PaintInput
	Percentage float64
}

type Recipe struct {
	Ingredients []Ingredient
	ResultR     uint8
	ResultG     uint8
	ResultB     uint8
	DeltaE      float64
	Method      string
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Mix(colors []PaintInput, weights []float64) (r, g, b uint8) {
	if len(colors) == 0 || len(colors) != len(weights) {
		return 0, 0, 0
	}

	normalized := NormalizeProportions(weights)

	var sumR, sumG, sumB float64
	for i, c := range colors {
		w := normalized[i] / 100.0
		sumR += float64(c.R) * w
		sumG += float64(c.G) * w
		sumB += float64(c.B) * w
	}

	return uint8(sumR + 0.5), uint8(sumG + 0.5), uint8(sumB + 0.5)
}

func (e *Engine) CalculateDeltaE(target [3]float64, recipe Recipe) float64 {
	l, a, b := color.RGBToLab(recipe.ResultR, recipe.ResultG, recipe.ResultB)
	return color.DeltaE2000(target, [3]float64{l, a, b})
}

func (e *Engine) SuggestRecipe(target [3]float64, available []PaintInput) Recipe {
	if len(available) == 0 {
		return Recipe{Method: "none"}
	}

	if len(available) == 1 {
		r, g, b := available[0].R, available[0].G, available[0].B
		l, a, bOut := color.RGBToLab(r, g, b)
		return Recipe{
			Ingredients: []Ingredient{
				{Paint: available[0], Percentage: 100},
			},
			ResultR: r,
			ResultG: g,
			ResultB: b,
			DeltaE:  color.DeltaE2000(target, [3]float64{l, a, bOut}),
			Method:  "single",
		}
	}

	if len(available) == 2 {
		bestProportions := []float64{50, 50}
		bestDeltaE := 1e9
		bestR, bestG, bestB := uint8(0), uint8(0), uint8(0)

		for p := 0; p <= 100; p += 5 {
			proportions := []float64{float64(p), float64(100 - p)}
			r, g, b := e.Mix(available, proportions)
			l, a, bOut := color.RGBToLab(r, g, b)
			delta := color.DeltaE2000(target, [3]float64{l, a, bOut})

			if delta < bestDeltaE {
				bestDeltaE = delta
				bestProportions = proportions
				bestR, bestG, bestB = r, g, b
			}
		}

		return Recipe{
			Ingredients: nonZeroIngredients(available, bestProportions),
			ResultR:     bestR,
			ResultG:     bestG,
			ResultB:     bestB,
			DeltaE:      bestDeltaE,
			Method:      "binary-search",
		}
	}

	bestProportions := make([]float64, len(available))
	bestDeltaE := 1e9
	bestR, bestG, bestB := uint8(0), uint8(0), uint8(0)

	for i := 0; i <= 100; i += 10 {
		for j := 0; j <= 100-i; j += 10 {
			k := 100 - i - j
			if len(available) == 3 {
				proportions := []float64{float64(i), float64(j), float64(k)}
				r, g, b := e.Mix(available, proportions)
				l, a, bOut := color.RGBToLab(r, g, b)
				delta := color.DeltaE2000(target, [3]float64{l, a, bOut})

				if delta < bestDeltaE {
					bestDeltaE = delta
					copy(bestProportions, proportions)
					bestR, bestG, bestB = r, g, b
				}
			} else {
				for lIdx := 0; lIdx <= 100-i-j; lIdx += 10 {
					m := 100 - i - j - lIdx
					if len(available) == 4 && m >= 0 {
						proportions := []float64{float64(i), float64(j), float64(k), float64(lIdx)}
						r, g, b := e.Mix(available, proportions)
						labL, labA, labB := color.RGBToLab(r, g, b)
						delta := color.DeltaE2000(target, [3]float64{labL, labA, labB})

						if delta < bestDeltaE {
							bestDeltaE = delta
							copy(bestProportions, proportions)
							bestR, bestG, bestB = r, g, b
						}
					}
				}
			}
		}
	}

	return Recipe{
		Ingredients: nonZeroIngredients(available, bestProportions),
		ResultR:     bestR,
		ResultG:     bestG,
		ResultB:     bestB,
		DeltaE:      bestDeltaE,
		Method:      "exhaustive",
	}
}

// nonZeroIngredients monta a lista de ingredientes descartando proporções 0%.
// A busca de proporções inclui extremos como 0/100 — uma linha "0% de X" na
// receita é ruído (e vira "0 gotas" na conversão), então some. Efeito colateral
// intencional: uma mistura "forçada" de 2 tintas pode colapsar pra 1 quando o
// catálogo tem outra tinta de cor idêntica — e essa É a resposta certa
// ("use essa tinta"), não uma mistura fantasma.
func nonZeroIngredients(paints []PaintInput, proportions []float64) []Ingredient {
	ingredients := make([]Ingredient, 0, len(paints))
	for i, p := range paints {
		if proportions[i] > 0 {
			ingredients = append(ingredients, Ingredient{Paint: p, Percentage: proportions[i]})
		}
	}
	return ingredients
}
