package mix

import "math"

// SuggestBestSubset testa combinações de minIngredients a maxIngredients tintas
// dentre os candidatos e retorna a melhor Recipe (menor DeltaE), usando o
// Engine.SuggestRecipe já existente pra cada combinação — ele já funciona
// corretamente para conjuntos de até 4 tintas, só não escolhe sozinho QUAIS
// tintas usar dentre N candidatas.
//
// minIngredients força um piso de tintas na receita: com 1, aceita "use só a
// tinta X" (equivalência 1:1, boa entre marcas diferentes); com 2, obriga uma
// mistura de verdade (usado na mesma marca, onde devolver ~100% de uma tinta é
// inútil). É rebaixado se o pool for menor que o piso.
func SuggestBestSubset(target [3]float64, candidates []PaintInput, minIngredients, maxIngredients int) Recipe {
	if len(candidates) == 0 {
		return Recipe{Method: "none"}
	}
	if maxIngredients <= 0 || maxIngredients > 4 {
		maxIngredients = 4 // Engine.SuggestRecipe só suporta até 4
	}
	if minIngredients < 1 {
		minIngredients = 1
	}
	if minIngredients > maxIngredients {
		minIngredients = maxIngredients
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
	minK := minIngredients
	if minK > len(pool) {
		minK = len(pool) // pool menor que o piso: usa o que tem
	}

	engine := NewEngine()
	best := Recipe{Method: "none", DeltaE: math.MaxFloat64}

	for k := minK; k <= maxK; k++ {
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
// (api/domain/ai/retrieval.go) — usa distância euclidiana em RGB em vez de Lab, o
// suficiente pra reduzir o espaço de busca com boa cobertura de matiz/luminosidade
// quando um fabricante tiver muitas tintas cadastradas. Não vale extrair um
// pacote compartilhado só por essa duplicação pontual entre api/domain/ai e api/domain/mix.
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
