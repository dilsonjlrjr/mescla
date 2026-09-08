package mix

import (
	"math"
	"paint-match-ai/api/domain/color"
	"sort"
	"time"
)

type PaintInput struct {
	ID      int64
	Name    string
	Code    string
	R, G, B uint8
	// ManufacturerID e Manufacturer são carga para o DTO de saída (rf-13,
	// marca por ingrediente) — o solver (Mix, custo, descerGradiente) nunca os
	// lê, só R/G/B decidem a mistura. ManufacturerID é sempre o id de
	// `manufacturers`, mesmo no caminho do estoque (`user_paints` tem espaço
	// de IDs próprio, que fica só em PaintInput.ID).
	ManufacturerID int64
	Manufacturer   string
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

const (
	// proporcaoMinima é o menor componente praticável na bancada: 5% é ~1 gota
	// em 20. Abaixo disso ninguém mede, então a receita não o propõe (RN4).
	proporcaoMinima = 0.05
	// passoDeProporcao é a granularidade da receita final, em porcentagem.
	passoDeProporcao = 5.0
	// deltaEAcertoDireto — abaixo disso a tinta sozinha JÁ É a resposta e nem
	// vale rodar o solver. Tem de ser bem apertado: com 1,0 o atalho devolvia
	// uma tinta de ΔE 0,96 onde a mistura acertava em cheio (0,00).
	deltaEAcertoDireto = 0.1
	// margemParaPreferirTintaUnica — em empate técnico, menos componentes
	// ganha: misturar para ganhar um décimo de ΔE não paga o trabalho.
	margemParaPreferirTintaUnica = 0.5

	maxIteracoesSolver = 200
	// maxCandidatosNoSolver limita o CUSTO, não a receita: quantas tintas
	// entram na mistura segue sem teto.
	maxCandidatosNoSolver = 48
	convergenciaSolver    = 0.001
	epsilonGradiente      = 0.005
	tetoDeTempoPorReceita = 2 * time.Second
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Mix(colors []PaintInput, weights []float64) (r, g, b uint8) {
	// rf-10: mistura subtrativa (Kubelka-Munk). A média ponderada de RGB que
	// existia aqui é aditiva e previa cinza onde a tinta dá verde.
	return MisturaSubtrativa(colors, weights)
}

func (e *Engine) CalculateDeltaE(target [3]float64, recipe Recipe) float64 {
	l, a, b := color.RGBToLab(recipe.ResultR, recipe.ResultG, recipe.ResultB)
	return color.DeltaE2000(target, [3]float64{l, a, b})
}

func (e *Engine) SuggestRecipe(target [3]float64, available []PaintInput) Recipe {
	return e.SuggestRecipeLimited(target, available, 0)
}

// SuggestRecipeLimited resolve as proporções por otimização sobre o pool
// inteiro (rf-10, RN3): descida de gradiente projetada no simplex, partindo da
// melhor tinta sozinha. O solver decide quantas tintas recebem peso — não há
// teto de contagem. `maxIngredientes <= 0` significa sem teto.
//
// A enumeração anterior (grade de proporções por número fixo de tintas) só
// funcionava até 4 componentes e custava O((100/passo)^(k−1)) por combinação.
func (e *Engine) SuggestRecipeLimited(target [3]float64, available []PaintInput, maxIngredientes int) Recipe {
	return e.suggestAte(target, available, maxIngredientes, time.Now().Add(tetoDeTempoPorReceita))
}

// suggestAte é o corpo do solver com prazo EXPLÍCITO, para quem encadeia mais
// de uma chamada não somar um teto novo a cada uma (RN8).
func (e *Engine) suggestAte(target [3]float64, available []PaintInput, maxIngredientes int, prazo time.Time) Recipe {
	if len(available) == 0 {
		return Recipe{Method: "none"}
	}

	for _, v := range target {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return Recipe{Method: "none"}
		}
	}

	melhorIdx, melhorDelta := 0, math.Inf(1)
	for i, p := range available {
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		if d := color.DeltaE2000(target, [3]float64{l, a, b}); d < melhorDelta {
			melhorIdx, melhorDelta = i, d
		}
	}

	// Uma tinta que já acerta o alvo vence qualquer mistura (RN7): não faz
	// sentido mandar o usuário misturar para chegar onde já se está.
	if melhorDelta <= deltaEAcertoDireto || len(available) == 1 {
		return receitaDeUmaTinta(available[melhorIdx], target, melhorDelta)
	}

	// Poda por relevância (não por contagem): o gradiente numérico é O(n²) por
	// iteração, então um pool de milhares precisa ser reduzido aos candidatos
	// que têm chance de entrar. O corte é por distância ao alvo, não por um
	// teto de quantos ingredientes a receita pode ter.
	available, melhorIdx = podarPorRelevancia(target, available, melhorIdx)

	// Dois pontos de partida: o vértice da melhor tinta e as proporções iguais
	// (RN3). Descida de gradiente cai em mínimo local, e partir de um lugar só
	// perdia para uma busca trivial de duas tintas.
	partidas := make([][]float64, 0, 2)

	vertice := make([]float64, len(available))
	vertice[melhorIdx] = 1
	partidas = append(partidas, vertice)

	uniforme := make([]float64, len(available))
	for i := range uniforme {
		uniforme[i] = 1 / float64(len(uniforme))
	}
	partidas = append(partidas, uniforme)

	melhorProp, melhorMistura := []float64(nil), math.Inf(1)
	for _, inicio := range partidas {
		pesos := e.descerGradiente(target, available, inicio, prazo)
		pesos = e.podarEResolver(target, available, pesos, maxIngredientes, prazo)
		prop := arredondarEmPassos(pesos)
		if d := e.custo(target, available, prop); d < melhorMistura {
			melhorProp, melhorMistura = prop, d
		}
	}

	// Piso de qualidade: a melhor dupla em passos de 5%. O gradiente não pode
	// entregar menos que a busca trivial que qualquer um faria à mão — foi
	// exatamente isso que o guardrail mediu perdendo em 14 de 20 alvos.
	if propPar, dPar := e.melhorPar(target, available, maxIngredientes, prazo); dPar < melhorMistura {
		melhorProp, melhorMistura = propPar, dPar
	}

	if melhorProp == nil || melhorMistura >= melhorDelta-margemParaPreferirTintaUnica {
		return receitaDeUmaTinta(available[melhorIdx], target, melhorDelta)
	}

	proporcoes := melhorProp
	r, g, b := e.Mix(available, proporcoes)
	delta := melhorMistura

	return Recipe{
		Ingredients: nonZeroIngredients(available, proporcoes),
		ResultR:     r,
		ResultG:     g,
		ResultB:     b,
		DeltaE:      delta,
		Method:      "kubelka-munk",
	}
}

// melhorPar faz a busca trivial que qualquer pintor faz na bancada: testar
// duas tintas em proporções de 5 em 5. É barata sobre o pool já podado
// (C(48,2) × 21 avaliações) e serve de PISO — o gradiente pode ser melhor, e
// muitas vezes é, mas nunca pode entregar menos que isto.
func (e *Engine) melhorPar(target [3]float64, paints []PaintInput, maxIngredientes int, prazo time.Time) ([]float64, float64) {
	if len(paints) < 2 || (maxIngredientes > 0 && maxIngredientes < 2) {
		return nil, math.Inf(1)
	}

	melhor, melhorDelta := []float64(nil), math.Inf(1)
	par := make([]PaintInput, 2)
	prop := []float64{0, 0}

	for i := 0; i < len(paints); i++ {
		if time.Now().After(prazo) {
			break
		}
		for j := i + 1; j < len(paints); j++ {
			par[0], par[1] = paints[i], paints[j]
			for p := passoDeProporcao; p <= 100-passoDeProporcao; p += passoDeProporcao {
				prop[0], prop[1] = p, 100-p
				if d := e.custo(target, par, prop); d < melhorDelta {
					melhorDelta = d
					melhor = make([]float64, len(paints))
					melhor[i], melhor[j] = p, 100-p
				}
			}
		}
	}
	return melhor, melhorDelta
}

// podarPorRelevancia reduz o pool aos candidatos mais próximos do alvo,
// preservando a melhor tinta. Devolve o pool podado e o novo índice dela.
func podarPorRelevancia(target [3]float64, pool []PaintInput, melhorIdx int) ([]PaintInput, int) {
	if len(pool) <= maxCandidatosNoSolver {
		return pool, melhorIdx
	}

	type candidato struct {
		p     PaintInput
		delta float64
	}
	lista := make([]candidato, len(pool))
	for i, p := range pool {
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		lista[i] = candidato{p: p, delta: color.DeltaE2000(target, [3]float64{l, a, b})}
	}
	sort.SliceStable(lista, func(i, j int) bool { return lista[i].delta < lista[j].delta })

	melhor := pool[melhorIdx]
	podado := make([]PaintInput, 0, maxCandidatosNoSolver)
	novoIdx := 0
	for i := 0; i < maxCandidatosNoSolver; i++ {
		if lista[i].p.ID == melhor.ID && lista[i].p.Name == melhor.Name {
			novoIdx = len(podado)
		}
		podado = append(podado, lista[i].p)
	}
	return podado, novoIdx
}

func receitaDeUmaTinta(p PaintInput, target [3]float64, delta float64) Recipe {
	return Recipe{
		Ingredients: []Ingredient{{Paint: p, Percentage: 100}},
		ResultR:     p.R,
		ResultG:     p.G,
		ResultB:     p.B,
		DeltaE:      delta,
		Method:      "single",
	}
}

// custo é o ΔE00 entre o alvo e a mistura das proporções dadas.
func (e *Engine) custo(target [3]float64, paints []PaintInput, pesos []float64) float64 {
	r, g, b := e.Mix(paints, pesos)
	l, a, bLab := color.RGBToLab(r, g, b)
	return color.DeltaE2000(target, [3]float64{l, a, bLab})
}

// descerGradiente minimiza o ΔE00 movendo os pesos no simplex. O gradiente é
// numérico: não existe forma fechada para o ΔE00 composto com Kubelka-Munk, e
// a diferença finita custa uma avaliação por tinta.
func (e *Engine) descerGradiente(target [3]float64, paints []PaintInput, pesos []float64, prazo time.Time) []float64 {
	atual := append([]float64(nil), pesos...)
	custoAtual := e.custo(target, paints, atual)
	passo := 0.25

	for iter := 0; iter < maxIteracoesSolver; iter++ {
		if time.Now().After(prazo) {
			break
		}

		grad := make([]float64, len(atual))
		estourou := false
		for i := range atual {
			// O gradiente numérico custa uma avaliação por tinta, e cada
			// avaliação percorre o pool: é O(n²) por iteração. Com pool
			// grande, o prazo tem de ser checado aqui dentro também.
			if i%256 == 0 && time.Now().After(prazo) {
				estourou = true
				break
			}
			tentativa := append([]float64(nil), atual...)
			tentativa[i] += epsilonGradiente
			grad[i] = (e.custo(target, paints, tentativa) - custoAtual) / epsilonGradiente
		}
		if estourou {
			break
		}

		candidato := make([]float64, len(atual))
		for i := range atual {
			candidato[i] = atual[i] - passo*grad[i]
		}
		projetarNoSimplex(candidato)

		novoCusto := e.custo(target, paints, candidato)
		if novoCusto < custoAtual-convergenciaSolver {
			atual, custoAtual = candidato, novoCusto
			// Passo que só encolhe mata a busca em poucas iterações: depois de
			// um acerto ele volta a crescer, dentro de um teto.
			passo = math.Min(passo*1.6, 0.5)
			continue
		}

		// Sem ganho: encurta o passo. Passo pequeno demais é convergência.
		passo /= 2
		if passo < 1e-4 {
			break
		}
	}
	return atual
}

// podarEResolver zera componente abaixo do mínimo praticável (RN4) e, quando
// há teto de contagem, mantém só os maiores. Depois de cada poda o solver roda
// de novo sobre o que sobrou — redistribuir sem re-otimizar erraria a cor.
func (e *Engine) podarEResolver(target [3]float64, paints []PaintInput, pesos []float64, maxIngredientes int, prazo time.Time) []float64 {
	// Índices que ainda podem receber peso. Reotimizar sobre o pool inteiro
	// depois de podar RESSUSCITA a tinta podada — a projeção no simplex deixa
	// qualquer coordenada voltar a crescer. Por isso o solver passa a rodar
	// sobre o subconjunto vivo, não sobre a lista toda com zeros.
	ativos := make([]int, 0, len(pesos))
	for i, w := range pesos {
		if w > 0 {
			ativos = append(ativos, i)
		}
	}

	for volta := 0; volta <= len(pesos); volta++ {
		if len(ativos) <= 1 {
			break
		}

		subPesos := make([]float64, len(ativos))
		for k, idx := range ativos {
			subPesos[k] = pesos[idx]
		}
		normalizarParaUm(subPesos)

		menorK, menor := 0, math.Inf(1)
		for k, w := range subPesos {
			if w < menor {
				menor, menorK = w, k
			}
		}

		precisaPodar := menor < proporcaoMinima
		if !precisaPodar && maxIngredientes > 0 && len(ativos) > maxIngredientes {
			precisaPodar = true
		}
		if !precisaPodar {
			break
		}

		ativos = append(ativos[:menorK], ativos[menorK+1:]...)

		subPaints := make([]PaintInput, len(ativos))
		novoSub := make([]float64, len(ativos))
		for k, idx := range ativos {
			subPaints[k] = paints[idx]
			// Continua da solução que já convergiu, em vez de reiniciar do
			// uniforme — reiniciar jogava fora até 6 ΔE de trabalho.
			novoSub[k] = pesos[idx]
		}
		normalizarParaUm(novoSub)
		novoSub = e.descerGradiente(target, subPaints, novoSub, prazo)

		for i := range pesos {
			pesos[i] = 0
		}
		for k, idx := range ativos {
			pesos[idx] = novoSub[k]
		}
	}

	normalizarParaUm(pesos)
	return pesos
}

// projetarNoSimplex devolve o ponto mais próximo com pesos não negativos
// somando 1 — algoritmo de Duchi et al., que é o que mantém o passo do
// gradiente dentro do espaço de proporções válidas.
func projetarNoSimplex(v []float64) {
	n := len(v)
	if n == 0 {
		return
	}
	ordenado := append([]float64(nil), v...)
	sort.Sort(sort.Reverse(sort.Float64Slice(ordenado)))

	var acumulado, theta float64
	rho := 0
	for i := 0; i < n; i++ {
		acumulado += ordenado[i]
		t := (acumulado - 1) / float64(i+1)
		if ordenado[i]-t > 0 {
			rho = i + 1
			theta = t
		}
	}
	if rho == 0 {
		// Nenhuma coordenada sobrevive: distribui igualmente.
		for i := range v {
			v[i] = 1 / float64(n)
		}
		return
	}
	for i := range v {
		v[i] = math.Max(v[i]-theta, 0)
	}
}

func normalizarParaUm(v []float64) {
	var soma float64
	for _, w := range v {
		if w > 0 {
			soma += w
		} else {
			soma += 0
		}
	}
	if soma == 0 {
		return
	}
	for i, w := range v {
		if w > 0 {
			v[i] = w / soma
		} else {
			v[i] = 0
		}
	}
}

// arredondarEmPassos converte os pesos [0,1] em porcentagens múltiplas do
// passo praticável (RN5), corrigindo a sobra no maior componente para a soma
// fechar exatamente em 100.
func arredondarEmPassos(pesos []float64) []float64 {
	prop := make([]float64, len(pesos))
	vivos := make([]int, 0, len(pesos))
	for i, w := range pesos {
		if w > 0 {
			prop[i] = math.Floor(w*100/passoDeProporcao) * passoDeProporcao
			vivos = append(vivos, i)
		}
	}
	if len(vivos) == 0 {
		return prop
	}

	var soma float64
	for _, i := range vivos {
		soma += prop[i]
	}

	// Distribui a sobra passo a passo, do maior resto para o menor. Somar tudo
	// num componente só produzia receita de 105% (e componente negativo)
	// quando havia muitos pesos iguais.
	sobra := 100 - soma
	ordem := append([]int(nil), vivos...)
	sort.SliceStable(ordem, func(a, b int) bool {
		ra := pesos[ordem[a]]*100 - prop[ordem[a]]
		rb := pesos[ordem[b]]*100 - prop[ordem[b]]
		return ra > rb
	})
	for k := 0; sobra >= passoDeProporcao && len(ordem) > 0; k++ {
		i := ordem[k%len(ordem)]
		prop[i] += passoDeProporcao
		sobra -= passoDeProporcao
	}
	for k := 0; sobra <= -passoDeProporcao && len(ordem) > 0; k++ {
		i := ordem[len(ordem)-1-(k%len(ordem))]
		if prop[i] >= passoDeProporcao {
			prop[i] -= passoDeProporcao
			sobra += passoDeProporcao
		}
	}

	return prop
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
