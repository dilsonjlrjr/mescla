package mix

import (
	"sort"

	"paint-match-ai/api/domain/color"
)

// SuggestBestSubset escolhe as tintas e as proporções que mais aproximam o
// alvo dentre os candidatos.
//
// rf-10: o solver recebe o pool INTEIRO e decide sozinho quantas tintas
// entram na receita — nada de enumerar C(n,k). A enumeração anterior só
// funcionava até 4 componentes e custava O((100/passo)^(k−1)) por combinação;
// com 15 candidatos e 5 tintas dava ~30 milhões de avaliações por região.
//
// minIngredients força um piso de tintas: com 1, aceita "use só a tinta X"
// (equivalência 1:1, boa entre marcas diferentes); com 2, obriga mistura de
// verdade (usado dentro da mesma marca, onde devolver ~100% de uma tinta é
// inútil). maxIngredients <= 0 significa SEM TETO.
func SuggestBestSubset(target [3]float64, candidates []PaintInput, minIngredients, maxIngredients int) Recipe {
	if len(candidates) == 0 {
		return Recipe{Method: "none"}
	}

	engine := NewEngine()
	receita := engine.SuggestRecipeLimited(target, candidates, maxIngredients)

	// Forçar mistura só faz sentido quando a tinta sozinha NÃO resolve. Se ela
	// já acerta o alvo, misturar outra coisa por cima piora a cor — a resposta
	// honesta é "use essa tinta", mesmo com piso de 2 ingredientes.
	podeForcar := minIngredients > 1 &&
		len(receita.Ingredients) < minIngredients &&
		len(candidates) >= minIngredients &&
		receita.DeltaE > deltaEAcertoDireto

	if podeForcar {
		if forcada := misturaForcada(engine, target, candidates, receita, minIngredients, maxIngredients); forcada != nil {
			// E só vale se não piorar de forma perceptível o que já se tinha.
			if forcada.DeltaE <= receita.DeltaE+deltaEAcertoDireto {
				return *forcada
			}
		}
	}

	return receita
}

// misturaForcada resolve de novo sem a tinta que o solver escolheu sozinha e
// combina as duas metades, para a receita sair com mais de um componente
// quando quem chama exige mistura. Devolve nil se não conseguir.
func misturaForcada(engine *Engine, target [3]float64, candidates []PaintInput, unica Recipe, minIngredients, maxIngredients int) *Recipe {
	if len(unica.Ingredients) != 1 {
		return nil
	}
	escolhida := unica.Ingredients[0].Paint

	resto := make([]PaintInput, 0, len(candidates))
	for _, c := range candidates {
		if c.ID == escolhida.ID && c.Name == escolhida.Name {
			continue
		}
		resto = append(resto, c)
	}
	if len(resto) == 0 {
		return nil
	}

	segunda := engine.SuggestRecipeLimited(target, resto, maxIngredients)
	if len(segunda.Ingredients) == 0 {
		return nil
	}

	combinada := make([]PaintInput, 0, 1+len(segunda.Ingredients))
	pesos := make([]float64, 0, 1+len(segunda.Ingredients))
	combinada = append(combinada, escolhida)
	pesos = append(pesos, 0.5)
	for _, ing := range segunda.Ingredients {
		combinada = append(combinada, ing.Paint)
		pesos = append(pesos, 0.5*ing.Percentage/100)
	}

	// O teto de ingredientes de quem chamou vale também aqui: sem isto,
	// min=2/max=1 devolvia 2 componentes.
	if maxIngredients > 0 && len(combinada) > maxIngredients {
		type item struct {
			p PaintInput
			w float64
		}
		itens := make([]item, len(combinada))
		for i := range combinada {
			itens[i] = item{p: combinada[i], w: pesos[i]}
		}
		sort.SliceStable(itens, func(i, j int) bool { return itens[i].w > itens[j].w })
		itens = itens[:maxIngredients]

		combinada = combinada[:0]
		pesos = pesos[:0]
		for _, it := range itens {
			combinada = append(combinada, it.p)
			pesos = append(pesos, it.w)
		}
	}

	prop := arredondarEmPassos(pesos)
	r, g, b := engine.Mix(combinada, prop)
	l, a, bLab := color.RGBToLab(r, g, b)

	receita := Recipe{
		Ingredients: nonZeroIngredients(combinada, prop),
		ResultR:     r,
		ResultG:     g,
		ResultB:     b,
		DeltaE:      color.DeltaE2000(target, [3]float64{l, a, bLab}),
		Method:      "kubelka-munk",
	}
	if len(receita.Ingredients) < minIngredients {
		return nil
	}
	return &receita
}
