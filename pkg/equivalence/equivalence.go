// Package equivalence orquestra a sugestão de receita equivalente: dada uma
// tinta de origem e o pool de tintas de um fabricante de destino, escolhe a
// mistura que melhor aproxima a cor. É a fonte única dessa regra de negócio,
// compartilhada pelo desktop (app.go, dados via SQL) e pelo mobile (cmd/wasm,
// dados via catálogo JSON em memória) — os dois alvos DEVEM produzir a mesma
// receita para a mesma entrada.
package equivalence

import (
	"errors"

	"paint-match-ai/pkg/color"
	"paint-match-ai/pkg/mix"
)

// MaxViableDeltaE é o limite de ΔE2000 acima do qual uma cor é considerada
// irreproduzível com o catálogo de destino. ΔE ~10 já é uma diferença de cor
// óbvia a olho nu; acima disso a "receita" não replica a cor, só a aproxima.
const MaxViableDeltaE = 10.0

// ErrNoCandidates indica que, após excluir a própria tinta-alvo, não sobrou
// nenhuma tinta com cor no fabricante de destino para montar a mistura.
var ErrNoCandidates = errors.New("nenhuma tinta candidata para montar a mistura")

// Result é a saída completa da sugestão.
type Result struct {
	Recipe       mix.Recipe
	Tips         []string
	Reproducible bool
}

// Suggest monta a receita equivalente da cor de source usando somente as
// tintas de candidates (o catálogo do fabricante de destino).
//
// Regras centralizadas aqui:
//   - A própria tinta-alvo nunca entra como ingrediente (por ID) — na mesma
//     marca a "receita" viraria 100% dela mesma (ΔE 0), inútil.
//   - Mesma marca (sourceManufacturer == targetManufacturer) força mistura de
//     2+ tintas: o usuário já sabe que a tinta existe; ele quer o tom que NÃO
//     tem, feito com as que tem.
//   - Reproducible reflete MaxViableDeltaE.
func Suggest(source mix.PaintInput, sourceManufacturer, targetManufacturer string, candidates []mix.PaintInput) (Result, error) {
	kept := make([]mix.PaintInput, 0, len(candidates))
	for _, c := range candidates {
		if c.ID != source.ID {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		return Result{}, ErrNoCandidates
	}

	minIngredients := 1
	if targetManufacturer != "" && targetManufacturer == sourceManufacturer {
		minIngredients = 2
	}

	l, a, b := color.RGBToLab(source.R, source.G, source.B)
	recipe := mix.SuggestBestSubset([3]float64{l, a, b}, kept, minIngredients, 3)
	tips := mix.GenerateTips(source.R, source.G, source.B, recipe)

	return Result{
		Recipe:       recipe,
		Tips:         tips,
		Reproducible: recipe.DeltaE <= MaxViableDeltaE,
	}, nil
}

// SuggestFromStock monta a receita equivalente da cor de source usando o
// estoque do próprio pintor (tintas de qualquer marca que ele possui). É o
// "priorize o que eu tenho": se sai da estante do usuário, essa é a melhor
// resposta; caso contrário a UI cai no fluxo por fabricante (Suggest).
//
// Difere de Suggest em dois pontos, por o estoque ser multi-marca e livre:
//   - Não exclui nada por ID: a origem vem do catálogo e o estoque tem espaço
//     de IDs próprio, então excluir por ID removeria tinta legítima por acaso.
//     Se o estoque já contém a cor exata, a equivalência 1:1 ("você já tem
//     essa tinta") é justamente a resposta desejada.
//   - Nunca força 2+ ingredientes: usar uma única tinta que o pintor possui é
//     um resultado válido, não uma marca a esgotar.
func SuggestFromStock(source mix.PaintInput, stock []mix.PaintInput) (Result, error) {
	if len(stock) == 0 {
		return Result{}, ErrNoCandidates
	}

	l, a, b := color.RGBToLab(source.R, source.G, source.B)
	recipe := mix.SuggestBestSubset([3]float64{l, a, b}, stock, 1, 3)
	tips := mix.GenerateTips(source.R, source.G, source.B, recipe)

	return Result{
		Recipe:       recipe,
		Tips:         tips,
		Reproducible: recipe.DeltaE <= MaxViableDeltaE,
	}, nil
}
