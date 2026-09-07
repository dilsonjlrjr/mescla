package mix

import "math"

// Modelo de mistura subtrativo (rf-10).
//
// Tinta é subtrativa: pigmento absorve luz. A média ponderada de RGB é
// aditiva e prevê o oposto — amarelo (255,255,0) com azul (0,0,255) meio a
// meio dá (128,128,128), cinza, quando o pote dá verde.
//
// Kubelka-Munk de constante única resolve isso trabalhando em espaço de
// REFLETÂNCIA em vez de intensidade: para cada canal, a razão entre absorção
// e espalhamento é K/S = (1−R)² / 2R. Essa razão é o que se mistura de forma
// linear, ponderada pela proporção de cada tinta. Depois volta-se à
// refletância pela inversa.
//
// Não é colorimetria de laboratório: um catálogo com três números por tinta
// não dá para calibrar K e S separadamente. É a aproximação honesta possível
// com o dado que existe — e acerta o comportamento qualitativo (amarelo com
// azul dá verde) que a média aritmética errava por construção.

// refletanciaMinima é o piso FÍSICO de refletância de um canal. Não serve só
// para evitar divisão por zero: tinta real reflete alguns por cento mesmo no
// canal que mais absorve, e um piso perto de zero manda K/S para a casa dos
// milhares, fazendo qualquer mistura colapsar em preto. 2% é a ordem de
// grandeza de um pigmento bem escuro.
const refletanciaMinima = 0.02

// ksDaRefletancia aplica a função de Kubelka-Munk a um canal já normalizado.
func ksDaRefletancia(r float64) float64 {
	if r < refletanciaMinima {
		r = refletanciaMinima
	}
	if r > 1 {
		r = 1
	}
	return (1 - r) * (1 - r) / (2 * r)
}

// refletanciaDoKS é a inversa: dado K/S, devolve a refletância do canal.
func refletanciaDoKS(ks float64) float64 {
	if ks < 0 || math.IsNaN(ks) {
		ks = 0
	}
	if math.IsInf(ks, 1) {
		return 0
	}
	r := 1 + ks - math.Sqrt(ks*ks+2*ks)
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// MisturaSubtrativa devolve a cor resultante de misturar as tintas nas
// proporções dadas (em porcentagem, normalizadas internamente). Proporção
// negativa, NaN ou Inf conta como zero — nunca inverte a mistura.
func MisturaSubtrativa(colors []PaintInput, weights []float64) (r, g, b uint8) {
	if len(colors) == 0 || len(colors) != len(weights) {
		return 0, 0, 0
	}

	var soma float64
	limpos := make([]float64, len(weights))
	for i, w := range weights {
		if w > 0 && !math.IsNaN(w) && !math.IsInf(w, 0) {
			limpos[i] = w
			soma += w
		}
	}
	if soma == 0 {
		return 0, 0, 0
	}

	// Mistura ponderada por ESPALHAMENTO, não só por proporção:
	// K/S da mistura = Σ(cᵢ·Kᵢ) / Σ(cᵢ·Sᵢ).
	//
	// Ponderar só pela proporção subestima grosseiramente o branco, que tem
	// espalhamento altíssimo (é para isso que serve o dióxido de titânio) — e
	// clarear com branco é a mistura mais comum na bancada. Com S estimado
	// pela própria refletância do canal, vermelho com branco volta a dar rosa
	// em vez de continuar vermelho.
	var kR, sR, kG, sG, kB, sB float64
	for i, c := range colors {
		p := limpos[i] / soma
		if p == 0 {
			continue
		}
		for _, ch := range []struct {
			valor uint8
			k, s  *float64
		}{
			{c.R, &kR, &sR},
			{c.G, &kG, &sG},
			{c.B, &kB, &sB},
		} {
			r := float64(ch.valor) / 255
			if r < refletanciaMinima {
				r = refletanciaMinima
			}
			espalhamento := r
			*ch.k += p * ksDaRefletancia(r) * espalhamento
			*ch.s += p * espalhamento
		}
	}

	ksR := razao(kR, sR)
	ksG := razao(kG, sG)
	ksB := razao(kB, sB)

	return canal(refletanciaDoKS(ksR)), canal(refletanciaDoKS(ksG)), canal(refletanciaDoKS(ksB))
}

// razao devolve K/S protegendo contra espalhamento nulo.
func razao(k, s float64) float64 {
	if s <= 0 {
		return 0
	}
	return k / s
}

// canal converte refletância [0,1] para o byte do canal, com arredondamento.
func canal(r float64) uint8 {
	v := r*255 + 0.5
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
