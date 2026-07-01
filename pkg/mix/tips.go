package mix

import (
	"fmt"
	"math"

	"paint-match-ai/pkg/color"
)

// Thresholds heurísticos, não têm base científica rígida — ponto de partida
// pra ajustar empiricamente conforme o feedback de uso real com tintas físicas.
const (
	lightnessThreshold = 0.06 // ~6% de diferença de luminosidade já é perceptível
	saturationFloor     = 0.12 // abaixo disso a cor é quase neutra; hue vira ruído
	hueThresholdDeg     = 12.0 // graus de diferença de matiz pra valer uma dica
)

// GenerateTips gera dicas de ajuste em texto comparando o HSL da cor alvo
// (a tinta de origem que se quer replicar) com o HSL do resultado calculado
// da receita — não compara os ingredientes entre si, já que o que importa é o
// desvio real entre o que a receita produz e o que se queria. Retorna de 0 a
// 3 dicas: só gera uma quando o desvio é perceptível E existe um ingrediente
// plausível pra corrigi-lo.
func GenerateTips(targetR, targetG, targetB uint8, recipe Recipe) []string {
	var tips []string
	if len(recipe.Ingredients) == 0 {
		return tips
	}

	targetHue, targetSat, targetL := color.RGBToHSL(targetR, targetG, targetB)
	resultHue, resultSat, resultL := color.RGBToHSL(recipe.ResultR, recipe.ResultG, recipe.ResultB)

	lightener, hasLightener := findLightener(recipe.Ingredients)

	if hasLightener {
		if targetL-resultL > lightnessThreshold {
			low, high := lightenRange(lightener.Percentage)
			tips = append(tips, fmt.Sprintf(
				"Muito escuro: aumente o %s para %.0f–%.0f%%.",
				lightener.Paint.Name, low, high))
		} else if resultL-targetL > lightnessThreshold && lightener.Percentage > 10 {
			tips = append(tips, fmt.Sprintf(
				"Muito claro: reduza o %s em cerca de 5%%.",
				lightener.Paint.Name))
		}
	}

	if targetSat > saturationFloor {
		if resultSat <= saturationFloor {
			// O alvo tem uma cor de verdade, mas a receita só conseguiu chegar
			// num tom neutro/acinzentado (ex: só branco+preto disponíveis pra
			// aproximar um azul saturado). Não tem ingrediente pra "ajustar" —
			// falta um pigmento saturado no catálogo do fabricante de destino.
			tips = append(tips, fmt.Sprintf(
				"O fabricante de destino não tem nenhuma tinta na família do %s — a melhor mistura possível com o catálogo disponível fica num tom neutro/acinzentado.",
				hueBucketName(targetHue)))
		} else {
			hueDiff := hueDelta(targetHue, resultHue)
			if math.Abs(hueDiff) > hueThresholdDeg {
				mover, others, ok := findHueMover(recipe.Ingredients, targetHue, lightener, hasLightener)
				resultBucket := hueBucketName(resultHue)
				switch {
				case ok && len(others) == 1:
					// Há um ingrediente mais próximo do matiz do alvo e outro pra
					// compensar — dica de rebalanceamento de verdade.
					tips = append(tips, fmt.Sprintf(
						"Muito %s: aumente o %s e reduza o %s.",
						resultBucket, mover.Paint.Name, others[0].Paint.Name))
				case ok && len(others) > 1:
					tips = append(tips, fmt.Sprintf(
						"Muito %s: aumente o %s e reduza os demais.",
						resultBucket, mover.Paint.Name))
				case ok:
					// Só existe UM ingrediente saturado na receita: ele já é o
					// responsável pelo desvio de matiz, não tem com o que
					// rebalancear. "Aumentar" ele só pioraria — a mistura desse
					// fabricante simplesmente não tem o pigmento certo pra chegar
					// nesse tom (ex: falta um amarelo/laranja pra sair de um
					// vermelho+branco). Ser honesto em vez de inventar uma dica.
					tips = append(tips, "O fabricante escolhido não tem uma tinta com o matiz certo pra chegar nessa cor — o resultado é a melhor aproximação possível com o catálogo disponível.")
				}
			}
		}
	}

	if len(tips) > 3 {
		tips = tips[:3]
	}
	return tips
}

// findLightener acha, entre os ingredientes da receita, um candidato plausível
// a "clareador" (tipicamente um branco): precisa ter a maior luminosidade do
// grupo E ser claro/neutro o bastante (L alto ou saturação baixa) pra não
// confundir uma cor saturada-mas-clara com um branco de verdade.
func findLightener(ings []Ingredient) (Ingredient, bool) {
	if len(ings) == 0 {
		return Ingredient{}, false
	}

	best := ings[0]
	_, bestS, bestL := color.RGBToHSL(best.Paint.R, best.Paint.G, best.Paint.B)
	for _, ing := range ings[1:] {
		_, s, l := color.RGBToHSL(ing.Paint.R, ing.Paint.G, ing.Paint.B)
		if l > bestL {
			bestL, bestS = l, s
			best = ing
		}
	}

	// Só conta como "clareador" de verdade se for dessaturado o suficiente pra
	// ser um neutro/branco, ou praticamente branco puro independente da
	// saturação — evita confundir um pastel vibrante (ex: um rosa bem claro)
	// com um clareador de verdade só porque tem luminosidade alta.
	if bestS > 0.30 && bestL < 0.92 {
		return Ingredient{}, false
	}
	return best, true
}

// findHueMover acha, entre os ingredientes saturados (excluindo o clareador),
// aquele cujo matiz mais se aproxima do matiz do alvo — candidato a "puxar" a
// mistura na direção certa. Retorna também os demais ingredientes saturados
// relevantes (pra nomear qual reduzir quando sobra só um).
func findHueMover(ings []Ingredient, targetHue float64, lightener Ingredient, hasLightener bool) (mover Ingredient, others []Ingredient, ok bool) {
	type candidate struct {
		ing  Ingredient
		diff float64
	}
	var saturated []candidate

	for _, ing := range ings {
		if hasLightener && ing.Paint.ID == lightener.Paint.ID {
			continue
		}
		h, s, _ := color.RGBToHSL(ing.Paint.R, ing.Paint.G, ing.Paint.B)
		if s < saturationFloor {
			continue
		}
		saturated = append(saturated, candidate{ing: ing, diff: math.Abs(hueDelta(targetHue, h))})
	}

	if len(saturated) == 0 {
		return Ingredient{}, nil, false
	}

	bestIdx := 0
	for i, c := range saturated {
		if c.diff < saturated[bestIdx].diff {
			bestIdx = i
		}
	}

	mover = saturated[bestIdx].ing
	for i, c := range saturated {
		if i != bestIdx {
			others = append(others, c.ing)
		}
	}
	return mover, others, true
}

// hueDelta retorna a diferença angular assinada (a - b) normalizada pra -180..180.
func hueDelta(a, b float64) float64 {
	return math.Mod(a-b+540, 360) - 180
}

// hueBucketName mapeia um matiz em graus (0-360) pro nome de cor em português
// mais próximo — usado só pra dar nome legível à dica, não é uma classificação
// rigorosa de teoria de cor.
func hueBucketName(h float64) string {
	switch {
	case h < 15 || h >= 345:
		return "vermelho"
	case h < 45:
		return "laranja"
	case h < 70:
		return "amarelo"
	case h < 160:
		return "verde"
	case h < 200:
		return "ciano"
	case h < 260:
		return "azul"
	case h < 320:
		return "roxo"
	default:
		return "rosa"
	}
}

// lightenRange sugere uma faixa de percentual pra aumentar o clareador,
// capada em 35% (acima disso a receita perde o "corpo" da cor original).
func lightenRange(current float64) (low, high float64) {
	high = current + 10
	if high > 35 {
		high = 35
	}
	low = high - 5
	if low < current {
		low = current
	}
	return low, high
}
