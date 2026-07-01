package mix

import "math"

// SuggestBestSubset testa combinações de 1 a maxIngredients tintas dentre os
// candidatos e retorna a melhor Recipe (menor DeltaE), usando o Engine.SuggestRecipe
// já existente pra cada combinação — ele já funciona corretamente para conjuntos
// de até 4 tintas, só não escolhe sozinho QUAIS tintas usar dentre N candidatas.
func SuggestBestSubset(target [3]float64, candidates []PaintInput, maxIngredients int) Recipe {
	if len(candidates) == 0 {
		return Recipe{Method: "none"}
	}
	if maxIngredients <= 0 || maxIngredients > 4 {
		maxIngredients = 4 // Engine.SuggestRecipe só suporta até 4
	}

	pool := candidates
	const safetyLimit = 15
	if len(pool) > safetyLimit {
		pool = selectDiverseSubset(pool, safetyLimit)
	}

	maxK := maxIngredients
	if maxK > len(pool) {
		maxK = len(pool)
	}

	engine := NewEngine()
	best := Recipe{Method: "none", DeltaE: math.MaxFloat64}

	for k := 1; k <= maxK; k++ {
		forEachCombination(pool, k, func(combo []PaintInput) {
			candidateCopy := make([]PaintInput, len(combo))
			copy(candidateCopy, combo)
			recipe := engine.SuggestRecipe(target, candidateCopy)
			if recipe.DeltaE < best.DeltaE {
				best = recipe
			}
		})
	}

	return best
}

// forEachCombination itera todas as combinações de tamanho k dentre items,
// chamando fn pra cada uma (índices estritamente crescentes, sem repetição).
func forEachCombination(items []PaintInput, k int, fn func([]PaintInput)) {
	n := len(items)
	if k <= 0 || k > n {
		return
	}
	combo := make([]PaintInput, k)
	var recurse func(start, depth int)
	recurse = func(start, depth int) {
		if depth == k {
			fn(combo)
			return
		}
		for i := start; i < n; i++ {
			combo[depth] = items[i]
			recurse(i+1, depth+1)
		}
	}
	recurse(0, 0)
}

// selectDiverseSubset é uma cópia local e simplificada de selectDiversePaints
// (pkg/ai/retrieval.go) — usa distância euclidiana em RGB em vez de Lab, o
// suficiente pra reduzir o espaço de busca com boa cobertura de matiz/luminosidade
// quando um fabricante tiver muitas tintas cadastradas. Não vale extrair um
// pacote compartilhado só por essa duplicação pontual entre pkg/ai e pkg/mix.
func selectDiverseSubset(paints []PaintInput, n int) []PaintInput {
	if len(paints) <= n {
		return paints
	}

	selected := make([]PaintInput, 0, n)
	selected = append(selected, paints[0])

	for len(selected) < n {
		bestIdx, bestMinDist := -1, -1.0
		for i, p := range paints {
			skip := false
			for _, s := range selected {
				if p.ID == s.ID {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			minDist := math.MaxFloat64
			for _, s := range selected {
				dr := float64(int(p.R) - int(s.R))
				dg := float64(int(p.G) - int(s.G))
				db := float64(int(p.B) - int(s.B))
				d := dr*dr + dg*dg + db*db
				if d < minDist {
					minDist = d
				}
			}
			if minDist > bestMinDist {
				bestMinDist = minDist
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		selected = append(selected, paints[bestIdx])
	}

	return selected
}
