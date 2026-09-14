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
	// entram na mistura é decidido pelo teto de tintas (rf-20 RN1).
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
// melhor tinta sozinha. O solver decide quantas tintas recebem peso, até o
// teto: `maxIngredientes <= 0` usa o teto padrão de 4 tintas (rf-20 RN1).
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

	// rf-20 RN1: sem teto pedido, a receita tem no máximo 4 tintas. "Sem teto"
	// (rf-10) comprava décimos de ΔE com dez, quinze tintas diferentes.
	if maxIngredientes <= 0 {
		maxIngredientes = tetoPadraoDeTintas
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
	if melhorDelta <= deltaEAcertoDireto || len(available) == 1 || maxIngredientes == 1 {
		return receitaDeUmaTinta(available[melhorIdx], target, melhorDelta)
	}

	// rf-20 RN3: a dupla de referência sai antes do gradiente — é piso da
	// receita e ponto de partida, e fica pronta mesmo se o prazo apertar.
	ref := e.duplaDeReferencia(target, available, prazo)

	// Poda por relevância (não por contagem): o gradiente numérico é O(n²) por
	// iteração, então um pool de milhares precisa ser reduzido aos candidatos
	// que têm chance de entrar. O corte é por distância ao alvo, não por um
	// teto de quantos ingredientes a receita pode ter.
	podado, idxPodado := podarPorRelevancia(target, available, melhorIdx)
	melhorProp, melhorMistura := e.resolverNoPool(target, podado, idxPodado, maxIngredientes, prazo, true)
	pool := podado

	considerar := func(candPool []PaintInput, prop []float64, preferir bool) {
		if prop == nil {
			return
		}
		d := e.custo(target, candPool, prop)
		if melhorProp == nil || preferir || aceitaTroca(melhorMistura, contarIngredientes(melhorProp), d, contarIngredientes(prop)) {
			melhorProp, melhorMistura, pool = prop, d, candPool
		}
	}

	// D-019 + rf-20: segunda busca num pool ampliado — branco, preto e neutras
	// (que a poda por distância descartava) e as duas tintas da dupla de
	// referência. Parte das neutras e da dupla, compara com a própria dupla e
	// com a melhor combinação só de neutras. Cada troca passa pelo custo por
	// tinta (RN2).
	// A dupla de referência é piso mesmo com o prazo estourado na primeira
	// busca (RN6): sem isto, a receita podia sair sem piso nenhum.
	if ref.ok {
		poolRef := acrescentarSeFaltar(append([]PaintInput(nil), podado...), ref.tintas()...)
		considerar(poolRef, ref.proporcoesEm(poolRef), false)
	}

	extras := tintasDeClareamento(target, available, podado)
	ampliado := make([]PaintInput, 0, len(podado)+len(extras)+2)
	ampliado = append(append(ampliado, podado...), extras...)
	ampliado = acrescentarSeFaltar(ampliado, ref.tintas()...)

	// Alvo quase neutro: a combinação só de neutras sai ANTES da segunda busca
	// e com folga própria de prazo (RN5/RN6). Sob carga, o gradiente consumia
	// o prazo, a busca de neutras era cortada e o cinza voltava com tinta
	// colorida (guardrail rf-20, Acrilex cinza 120 com Rosa Bebê).
	quaseNeutro := alvoQuaseNeutro(target)
	var comboNeutro []float64
	if quaseNeutro {
		prazoNeutras := prazo
		if minimo := time.Now().Add(folgaDasNeutras); minimo.After(prazoNeutras) {
			prazoNeutras = minimo
		}
		comboNeutro, _ = e.melhorComboNeutro(target, ampliado, maxIngredientes, prazoNeutras)
	}

	if time.Now().Before(prazo) {
		propRef := ref.proporcoesEm(ampliado)
		var partidaRef []float64
		if propRef != nil {
			partidaRef = make([]float64, len(propRef))
			for i, v := range propRef {
				partidaRef[i] = v / 100
			}
		}
		if prop, _ := e.resolverNoPool(target, ampliado, idxPodado, maxIngredientes, prazo, false, partidaNeutra(ampliado), partidaRef); prop != nil {
			considerar(ampliado, prop, false)
		}
		if !quaseNeutro {
			if prop, _ := e.melhorComboNeutro(target, ampliado, maxIngredientes, prazo); prop != nil {
				considerar(ampliado, prop, false)
			}
		}
	}

	if comboNeutro != nil {
		// Um cinza feito de rosa e azul acerta no número e erra na bancada.
		// A neutra ganha se ficar na mesma tolerância que já faz preferir uma
		// tinta só a uma mistura — e só força a troca quando a melhor receita
		// até aqui tem tinta colorida; entre duas neutras, vale o custo por
		// tinta.
		d := e.custo(target, ampliado, comboNeutro)
		preferir := temTintaColorida(pool, melhorProp) && d <= melhorMistura+margemParaPreferirTintaUnica
		considerar(ampliado, comboNeutro, preferir)
	}

	if melhorProp == nil || !misturaCompensa(melhorDelta, melhorMistura, contarIngredientes(melhorProp)) {
		return receitaDeUmaTinta(available[melhorIdx], target, melhorDelta)
	}

	proporcoes := melhorProp
	r, g, b := e.Mix(pool, proporcoes)
	delta := melhorMistura

	return Recipe{
		Ingredients: nonZeroIngredients(pool, proporcoes),
		ResultR:     r,
		ResultG:     g,
		ResultB:     b,
		DeltaE:      delta,
		Method:      "kubelka-munk",
	}
}

// resolverNoPool roda o solver sobre um pool já podado e devolve as proporções
// (em %) e o ΔE00 da melhor mistura. Parte do vértice da melhor tinta e das
// proporções iguais (RN3) — descida de gradiente cai em mínimo local, e partir
// de um lugar só perdia para uma busca trivial de duas tintas. `comPiso`
// compara com a melhor dupla exaustiva do pool.
func (e *Engine) resolverNoPool(target [3]float64, available []PaintInput, melhorIdx, maxIngredientes int, prazo time.Time, comPiso bool, outrasPartidas ...[]float64) ([]float64, float64) {
	partidas := make([][]float64, 0, 3+len(outrasPartidas))
	for _, p := range outrasPartidas {
		if p != nil {
			partidas = append(partidas, p)
		}
	}

	vertice := make([]float64, len(available))
	vertice[melhorIdx] = 1
	partidas = append(partidas, vertice)

	uniforme := make([]float64, len(available))
	for i := range uniforme {
		uniforme[i] = 1 / float64(len(uniforme))
	}
	partidas = append(partidas, uniforme)

	// rf-20 RN2: entre as partidas, tinta a mais só vence se pagar o custo.
	melhorProp, melhorMistura := []float64(nil), math.Inf(1)
	for _, inicio := range partidas {
		pesos := e.descerGradiente(target, available, inicio, prazo)
		pesos = e.podarEResolver(target, available, pesos, maxIngredientes, prazo)
		prop := arredondarEmPassos(pesos)
		d := e.custo(target, available, prop)
		if melhorProp == nil || aceitaTroca(melhorMistura, contarIngredientes(melhorProp), d, contarIngredientes(prop)) {
			melhorProp, melhorMistura = prop, d
		}
	}

	// Piso de qualidade: a melhor dupla em passos de 5%. O gradiente não pode
	// entregar menos que a busca trivial que qualquer um faria à mão — foi
	// exatamente isso que o guardrail mediu perdendo em 14 de 20 alvos. Na
	// segunda busca (pool ampliado) o piso é a dupla de referência, que já
	// cobre um pool maior; repetir a dupla exaustiva ali só custava tempo.
	if comPiso {
		propPar, dPar := e.melhorPar(target, available, maxIngredientes, prazo)
		if propPar != nil && (melhorProp == nil || aceitaTroca(melhorMistura, contarIngredientes(melhorProp), dPar, contarIngredientes(propPar))) {
			melhorProp, melhorMistura = propPar, dPar
		}
	}
	return melhorProp, melhorMistura
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

const (
	// vagasDeClareamento limita quantas tintas de ajuste de valor (branco,
	// preto, neutras) a segunda busca acrescenta ao pool podado (D-019).
	vagasDeClareamento = 8
	// cromaNeutra é o limite de croma (C*ab) abaixo do qual a tinta conta como
	// neutra.
	cromaNeutra = 6.0
	// margemSegundaBusca: a segunda busca só troca a receita se acertar pelo
	// menos isto a mais — empate fica com a primeira, que já era a resposta.
	margemSegundaBusca = 0.05
)

// custoPorTintaExtra é quanto de ΔE00 cada tinta a mais precisa comprar para a
// segunda busca trocar a receita: onze verdes para ganhar 0,08 não pagam o
// trabalho na bancada (D-019).
const custoPorTintaExtra = 0.25

// aceitaTroca decide se a receita candidata (ΔE `d2`, `n2` tintas) substitui a
// atual (`d1`, `n1`) — rf-20 RN2. As duas são comparadas pela pontuação
// ΔE00 + custoPorTintaExtra × tintas: tinta a mais precisa comprar 0,25 de
// ΔE00, e tinta a menos pode custar até isso. A troca exige ganho mínimo de
// `margemSegundaBusca`, para empate ficar com a atual.
func aceitaTroca(d1 float64, n1 int, d2 float64, n2 int) bool {
	return d2+custoPorTintaExtra*float64(n2) < d1+custoPorTintaExtra*float64(n1)-margemSegundaBusca
}

// misturaCompensa diz se a mistura (`dMistura`, `n` tintas) vale mais que a
// melhor tinta sozinha (`dUma`): precisa ganhar a margem de sempre (0,5) ou o
// custo das tintas extras, o que for maior (rf-20 RN2).
func misturaCompensa(dUma, dMistura float64, n int) bool {
	ganho := math.Max(margemParaPreferirTintaUnica, custoPorTintaExtra*float64(n-1))
	return dMistura < dUma-ganho
}

// tetoPadraoDeTintas é o teto de tintas da receita quando quem chama não pede
// outro (rf-20 RN1).
const tetoPadraoDeTintas = 4

// acrescentarSeFaltar devolve o pool com as tintas que ainda não estão nele
// (identidade ID + nome, como no resto do motor).
func acrescentarSeFaltar(pool []PaintInput, tintas ...PaintInput) []PaintInput {
	for _, t := range tintas {
		achou := false
		for _, p := range pool {
			if p.ID == t.ID && p.Name == t.Name {
				achou = true
				break
			}
		}
		if !achou {
			pool = append(pool, t)
		}
	}
	return pool
}

// ── Dupla de referência (rf-20 RN3) ──

const (
	// candidatosDaDupla: quantas tintas mais próximas do alvo entram na busca
	// exaustiva de dupla (M3 da macro: 200, não mais 48).
	candidatosDaDupla = 200
	// neutrasDaDupla: neutras extras (mais próximas da luminosidade do alvo).
	neutrasDaDupla = 12
	// finalistasDaDupla: combinações da triagem rápida que ganham ΔE00 exato.
	finalistasDaDupla = 40
	// paresRefinados: melhores duplas refinadas em passos de 5%.
	paresRefinados = 5
)

type duplaRef struct {
	a, b   PaintInput
	pa, pb float64
	delta  float64
	ok     bool
}

func (d duplaRef) tintas() []PaintInput {
	if !d.ok {
		return nil
	}
	return []PaintInput{d.a, d.b}
}

// proporcoesEm devolve as proporções (em %) da dupla sobre `pool`, ou nil.
func (d duplaRef) proporcoesEm(pool []PaintInput) []float64 {
	if !d.ok {
		return nil
	}
	prop := make([]float64, len(pool))
	// Cada tinta da dupla entra uma vez só: com identidade repetida no pool
	// (estoque do corpo sem dedupe), a soma passaria de 100.
	achouA, achouB := false, false
	for i, p := range pool {
		switch {
		case !achouA && p.ID == d.a.ID && p.Name == d.a.Name:
			prop[i] = d.pa
			achouA = true
		case !achouB && p.ID == d.b.ID && p.Name == d.b.Name:
			prop[i] = d.pb
			achouB = true
		}
	}
	if !achouA || !achouB {
		return nil
	}
	return prop
}

// linearSRGB é a tabela de linearização sRGB dos 256 valores de canal — a
// triagem avalia centenas de milhares de misturas e `math.Pow` por canal
// dominaria o custo.
var linearSRGB = func() [256]float64 {
	var t [256]float64
	for i := range t {
		v := float64(i) / 255
		if v <= 0.04045 {
			t[i] = v / 12.92
		} else {
			t[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return t
}()

// labRapido converte RGB para CIELAB (D65) com a tabela acima. Serve só para
// a TRIAGEM da dupla de referência; o ΔE00 que decide a receita vem sempre de
// `custo`, pelo mesmo caminho do resto do motor.
func labRapido(r, g, b uint8) [3]float64 {
	R, G, B := linearSRGB[r], linearSRGB[g], linearSRGB[b]
	x := (0.4124564*R + 0.3575761*G + 0.1804375*B) / 0.95047
	y := 0.2126729*R + 0.7151522*G + 0.0721750*B
	z := (0.0193339*R + 0.1191920*G + 0.9503041*B) / 1.08883
	f := func(t float64) float64 {
		if t > 216.0/24389.0 {
			return math.Cbrt(t)
		}
		return (24389.0/27.0*t + 16) / 116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return [3]float64{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

// duplaDeReferencia busca a melhor dupla entre as `candidatosDaDupla` tintas
// mais próximas do alvo, mais os extremos de luminosidade e até
// `neutrasDaDupla` neutras (rf-20 RN3):
//  1. triagem de todas as duplas em passos de 10% com mistura Kubelka-Munk
//     pré-calculada e distância Lab simples;
//  2. ΔE00 exato das `finalistasDaDupla` melhores;
//  3. refino em passos de 5%, entre −10% e +10%, das `paresRefinados` duplas
//     com menor ΔE00.
func (e *Engine) duplaDeReferencia(target [3]float64, pool []PaintInput, prazo time.Time) duplaRef {
	if len(pool) < 2 {
		return duplaRef{}
	}

	type cand struct {
		p     PaintInput
		delta float64
		l     float64
		croma float64
	}
	todos := make([]cand, len(pool))
	for i, p := range pool {
		lab := labRapido(p.R, p.G, p.B)
		todos[i] = cand{p: p, delta: color.DeltaE2000(target, lab), l: lab[0], croma: math.Hypot(lab[1], lab[2])}
	}
	ordem := make([]int, len(todos))
	for i := range ordem {
		ordem[i] = i
	}
	sort.SliceStable(ordem, func(a, b int) bool { return todos[ordem[a]].delta < todos[ordem[b]].delta })

	escolhido := make(map[int]bool, candidatosDaDupla+neutrasDaDupla+2)
	sel := make([]int, 0, candidatosDaDupla+neutrasDaDupla+2)
	incluir := func(i int) {
		if !escolhido[i] {
			escolhido[i] = true
			sel = append(sel, i)
		}
	}
	for k := 0; k < len(ordem) && k < candidatosDaDupla; k++ {
		incluir(ordem[k])
	}
	maisClara, maisEscura := 0, 0
	for i, c := range todos {
		if c.l > todos[maisClara].l {
			maisClara = i
		}
		if c.l < todos[maisEscura].l {
			maisEscura = i
		}
	}
	incluir(maisClara)
	incluir(maisEscura)
	neutras := make([]int, 0)
	for i, c := range todos {
		if c.croma < cromaNeutra {
			neutras = append(neutras, i)
		}
	}
	sort.SliceStable(neutras, func(a, b int) bool {
		return math.Abs(todos[neutras[a]].l-target[0]) < math.Abs(todos[neutras[b]].l-target[0])
	})
	for k := 0; k < len(neutras) && k < neutrasDaDupla; k++ {
		incluir(neutras[k])
	}
	if len(sel) < 2 {
		return duplaRef{}
	}

	// K e S por canal, pré-calculados (mesma fórmula de MisturaSubtrativa).
	type ks struct{ k, s [3]float64 }
	pre := make([]ks, len(sel))
	for n, i := range sel {
		p := todos[i].p
		for c, v := range [3]uint8{p.R, p.G, p.B} {
			r := float64(v) / 255
			if r < refletanciaMinima {
				r = refletanciaMinima
			}
			pre[n].k[c] = ksDaRefletancia(r) * r
			pre[n].s[c] = r
		}
	}

	type combo struct {
		a, b int
		p    float64
		d    float64
	}
	finalistas := make([]combo, 0, finalistasDaDupla+1)
	pior := math.Inf(1)
	for a := 0; a < len(sel); a++ {
		if time.Now().After(prazo) {
			break
		}
		for b := a + 1; b < len(sel); b++ {
			for passo := 1; passo <= 9; passo++ {
				p := float64(passo) / 10
				var rgb [3]uint8
				for c := 0; c < 3; c++ {
					k := p*pre[a].k[c] + (1-p)*pre[b].k[c]
					s := p*pre[a].s[c] + (1-p)*pre[b].s[c]
					rgb[c] = canal(refletanciaDoKS(razao(k, s)))
				}
				lab := labRapido(rgb[0], rgb[1], rgb[2])
				dl, da, db := lab[0]-target[0], lab[1]-target[1], lab[2]-target[2]
				d := dl*dl + da*da + db*db
				if len(finalistas) < finalistasDaDupla || d < pior {
					finalistas = append(finalistas, combo{a, b, p * 100, d})
					if len(finalistas) > finalistasDaDupla {
						sort.Slice(finalistas, func(i, j int) bool { return finalistas[i].d < finalistas[j].d })
						finalistas = finalistas[:finalistasDaDupla]
					}
					if len(finalistas) == finalistasDaDupla {
						pior = 0
						for _, f := range finalistas {
							if f.d > pior {
								pior = f.d
							}
						}
					}
				}
			}
		}
	}
	if len(finalistas) == 0 {
		return duplaRef{}
	}

	// ΔE00 exato, pelo caminho do motor.
	par := make([]PaintInput, 2)
	prop := make([]float64, 2)
	exato := func(a, b int, p float64) float64 {
		par[0], par[1] = todos[sel[a]].p, todos[sel[b]].p
		prop[0], prop[1] = p, 100-p
		return e.custo(target, par, prop)
	}
	for i := range finalistas {
		finalistas[i].d = exato(finalistas[i].a, finalistas[i].b, finalistas[i].p)
	}
	sort.SliceStable(finalistas, func(i, j int) bool { return finalistas[i].d < finalistas[j].d })

	melhor := finalistas[0]
	refinados := make(map[[2]int]bool, paresRefinados)
	for _, f := range finalistas {
		if len(refinados) >= paresRefinados || time.Now().After(prazo) {
			break
		}
		chave := [2]int{f.a, f.b}
		if refinados[chave] {
			continue
		}
		refinados[chave] = true
		for p := f.p - 10; p <= f.p+10; p += passoDeProporcao {
			if p < passoDeProporcao || p > 100-passoDeProporcao {
				continue
			}
			if d := exato(f.a, f.b, p); d < melhor.d {
				melhor = combo{f.a, f.b, p, d}
			}
		}
	}

	return duplaRef{
		a: todos[sel[melhor.a]].p, b: todos[sel[melhor.b]].p,
		pa: melhor.p, pb: 100 - melhor.p,
		delta: melhor.d, ok: true,
	}
}

// alvoQuaseNeutro diz se o alvo é um cinza (croma C*ab abaixo de cromaNeutra).
func alvoQuaseNeutro(target [3]float64) bool {
	return math.Hypot(target[1], target[2]) < cromaNeutra
}

// temTintaColorida diz se a receita usa alguma tinta com croma acima de
// cromaNeutra.
func temTintaColorida(pool []PaintInput, prop []float64) bool {
	for i, v := range prop {
		if v <= 0 || i >= len(pool) {
			continue
		}
		_, a, b := color.RGBToLab(pool[i].R, pool[i].G, pool[i].B)
		if math.Hypot(a, b) >= cromaNeutra {
			return true
		}
	}
	return false
}

func contarIngredientes(prop []float64) int {
	n := 0
	for _, v := range prop {
		if v > 0 {
			n++
		}
	}
	return n
}

// maxNeutrasNaBusca limita as tintas da busca exaustiva de neutras.
const maxNeutrasNaBusca = 10

// folgaDasNeutras garante à busca de neutras de um alvo quase neutro um tempo
// mínimo mesmo com o prazo geral já gasto (rf-20 RN6).
const folgaDasNeutras = 250 * time.Millisecond

// melhorComboNeutro testa, em passos de 5%, todas as duplas e trios entre a
// tinta mais clara, a mais escura e as neutras do pool mais próximas da
// luminosidade do alvo. É a busca que o pintor faz para um cinza — branco +
// cinza escuro + preto — e que o gradiente, partindo de tintas coloridas, não
// alcança (D-019). Devolve as proporções (em %) sobre o pool inteiro.
func (e *Engine) melhorComboNeutro(target [3]float64, pool []PaintInput, maxIngredientes int, prazo time.Time) ([]float64, float64) {
	if len(pool) < 2 || (maxIngredientes > 0 && maxIngredientes < 2) {
		return nil, math.Inf(1)
	}

	type candidato struct {
		idx int
		l   float64
	}
	lums := make([]float64, len(pool))
	neutra := make([]bool, len(pool))
	maisClara, maisEscura := 0, 0
	for i, p := range pool {
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		lums[i] = l
		neutra[i] = math.Hypot(a, b) < cromaNeutra
		if l > lums[maisClara] {
			maisClara = i
		}
		if l < lums[maisEscura] {
			maisEscura = i
		}
	}
	neutra[maisClara], neutra[maisEscura] = true, true

	lista := make([]candidato, 0)
	for i := range pool {
		if neutra[i] {
			lista = append(lista, candidato{idx: i, l: lums[i]})
		}
	}
	sort.SliceStable(lista, func(i, j int) bool {
		// Os extremos ficam sempre; o resto por proximidade de luminosidade.
		ei := lista[i].idx == maisClara || lista[i].idx == maisEscura
		ej := lista[j].idx == maisClara || lista[j].idx == maisEscura
		if ei != ej {
			return ei
		}
		return math.Abs(lista[i].l-target[0]) < math.Abs(lista[j].l-target[0])
	})
	if len(lista) > maxNeutrasNaBusca {
		lista = lista[:maxNeutrasNaBusca]
	}
	if len(lista) < 2 {
		return nil, math.Inf(1)
	}

	melhor, melhorDelta := []float64(nil), math.Inf(1)
	registrar := func(idx []int, prop []float64) {
		sub := make([]PaintInput, len(idx))
		for k, i := range idx {
			sub[k] = pool[i]
		}
		if d := e.custo(target, sub, prop); d < melhorDelta {
			melhorDelta = d
			melhor = make([]float64, len(pool))
			for k, i := range idx {
				melhor[i] = prop[k]
			}
		}
	}

	permiteTrio := maxIngredientes <= 0 || maxIngredientes >= 3
	for a := 0; a < len(lista); a++ {
		if time.Now().After(prazo) {
			break
		}
		for b := a + 1; b < len(lista); b++ {
			for p := passoDeProporcao; p <= 100-passoDeProporcao; p += passoDeProporcao {
				registrar([]int{lista[a].idx, lista[b].idx}, []float64{p, 100 - p})
			}
			if !permiteTrio {
				continue
			}
			for c := b + 1; c < len(lista); c++ {
				for p1 := passoDeProporcao; p1 <= 100-2*passoDeProporcao; p1 += passoDeProporcao {
					for p2 := passoDeProporcao; p1+p2 <= 100-passoDeProporcao; p2 += passoDeProporcao {
						registrar([]int{lista[a].idx, lista[b].idx, lista[c].idx}, []float64{p1, p2, 100 - p1 - p2})
					}
				}
			}
		}
	}
	return melhor, melhorDelta
}

// partidaNeutra é um ponto de partida com peso igual na tinta mais clara, na
// mais escura e em todas as neutras do pool. Cinza médio em Acrilex tem como
// melhor DUPLA Rosa Bebê + Grafite, então partir só dela não chega em Branco +
// Grafite + Preto; partir das neutras chega (D-019). Devolve nil com menos de
// duas tintas elegíveis.
func partidaNeutra(pool []PaintInput) []float64 {
	if len(pool) < 2 {
		return nil
	}
	pesos := make([]float64, len(pool))
	maisClara, maisEscura := 0, 0
	lums := make([]float64, len(pool))
	for i, p := range pool {
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		lums[i] = l
		if l > lums[maisClara] {
			maisClara = i
		}
		if l < lums[maisEscura] {
			maisEscura = i
		}
		if math.Hypot(a, b) < cromaNeutra {
			pesos[i] = 1
		}
	}
	pesos[maisClara], pesos[maisEscura] = 1, 1
	total := 0.0
	for _, w := range pesos {
		total += w
	}
	if total < 2 {
		return nil
	}
	for i := range pesos {
		pesos[i] /= total
	}
	return pesos
}

// tintasDeClareamento devolve, fora do pool podado, a tinta mais clara, a mais
// escura e as neutras mais próximas da luminosidade do alvo — as que ajustam
// valor e saturação. Sozinhas ficam longe de quase todo alvo, por isso a poda
// por distância as descartava (D-019).
func tintasDeClareamento(target [3]float64, pool, podado []PaintInput) []PaintInput {
	type chave struct {
		id   int64
		nome string
	}
	jaTem := make(map[chave]bool, len(podado))
	for _, p := range podado {
		jaTem[chave{p.ID, p.Name}] = true
	}

	type candidato struct {
		p     PaintInput
		l     float64
		croma float64
	}
	lista := make([]candidato, 0, len(pool))
	for _, p := range pool {
		if jaTem[chave{p.ID, p.Name}] {
			continue
		}
		l, a, b := color.RGBToLab(p.R, p.G, p.B)
		lista = append(lista, candidato{p: p, l: l, croma: math.Hypot(a, b)})
	}
	if len(lista) == 0 {
		return nil
	}

	extras := make([]PaintInput, 0, vagasDeClareamento)
	usado := make(map[chave]bool, vagasDeClareamento)
	incluir := func(c candidato) {
		k := chave{c.p.ID, c.p.Name}
		if !usado[k] && len(extras) < vagasDeClareamento {
			usado[k] = true
			extras = append(extras, c.p)
		}
	}

	maisClara, maisEscura := lista[0], lista[0]
	for _, c := range lista {
		if c.l > maisClara.l {
			maisClara = c
		}
		if c.l < maisEscura.l {
			maisEscura = c
		}
	}
	incluir(maisClara)
	incluir(maisEscura)

	neutras := make([]candidato, 0)
	for _, c := range lista {
		if c.croma < cromaNeutra {
			neutras = append(neutras, c)
		}
	}
	sort.SliceStable(neutras, func(i, j int) bool {
		return math.Abs(neutras[i].l-target[0]) < math.Abs(neutras[j].l-target[0])
	})
	for _, c := range neutras {
		incluir(c)
	}
	return extras
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
