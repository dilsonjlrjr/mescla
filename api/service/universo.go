package service

import (
	"fmt"

	"paint-match-ai/api/domain/color"
	"paint-match-ai/api/domain/equivalence"
	"paint-match-ai/api/domain/mix"
)

// Universo de busca de cor (rf-11).
//
// O usuário diz de onde a tinta pode sair por dois interruptores
// INDEPENDENTES: "só o que eu tenho" e "fabricante base". Ligados juntos
// significam "só o que eu tenho daquela marca" — a interseção, não a união.
//
// A montagem vive no servidor de propósito (RN10): o cliente manda os dois
// parâmetros, nunca a lista de candidatos.

// FaixaQualidade classifica o quão perto a receita chegou do alvo (RN3).
type FaixaQualidade string

const (
	FaixaOtima        FaixaQualidade = "otimo"
	FaixaAproximada   FaixaQualidade = "aproximada"
	FaixaNaoEncontrei FaixaQualidade = "nao-encontrei"

	// deltaEOtimo — abaixo disso a diferença só aparece lado a lado.
	deltaEOtimo = 3.0
	// deltaEAceitavel — acima disso o app admite que não achou e pergunta.
	// A folga até 6 cobre o erro de medida de gota, que o ΔE00 teórico não
	// captura.
	deltaEAceitavel = 6.0
)

// ClassificarFaixa devolve a faixa de qualidade de um ΔE00 (RN3).
func ClassificarFaixa(deltaE float64) FaixaQualidade {
	switch {
	case deltaE <= deltaEOtimo:
		return FaixaOtima
	case deltaE <= deltaEAceitavel:
		return FaixaAproximada
	default:
		return FaixaNaoEncontrei
	}
}

// ErrUniversoVazio diz que a combinação escolhida não tem tinta nenhuma. Não é
// erro de servidor (RN9): a tela mostra a mensagem e oferece as saídas.
type ErrUniversoVazio struct {
	Motivo string
}

func (e ErrUniversoVazio) Error() string { return e.Motivo }

// montarUniverso resolve as quatro combinações de RN1.
func (s *PaintService) montarUniverso(fabricanteID int64, soEstoque bool) ([]mix.PaintInput, error) {
	var nomeFabricante string
	if fabricanteID > 0 {
		if err := s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", fabricanteID).
			Scan(&nomeFabricante); err != nil {
			return nil, fmt.Errorf("fabricante não encontrado")
		}
	}

	switch {
	case !soEstoque && fabricanteID <= 0:
		return s.catalogoInteiroComCor()

	case !soEstoque && fabricanteID > 0:
		pool, err := s.loadPaintsByManufacturerID(fabricanteID)
		if err != nil {
			return nil, err
		}
		if len(pool) == 0 {
			return nil, ErrUniversoVazio{Motivo: nomeFabricante + " não tem tintas com cor cadastrada"}
		}
		return pool, nil

	case soEstoque && fabricanteID <= 0:
		pool, err := s.loadStockAsMixInputs()
		if err != nil {
			return nil, err
		}
		if len(pool) == 0 {
			return nil, ErrUniversoVazio{Motivo: "Seu estoque está vazio"}
		}
		return pool, nil

	default:
		pool, err := s.estoqueDoFabricante(fabricanteID)
		if err != nil {
			return nil, err
		}
		if len(pool) == 0 {
			return nil, ErrUniversoVazio{Motivo: "Você não tem tintas " + nomeFabricante + " no estoque"}
		}
		return pool, nil
	}
}

// catalogoInteiroComCor é o universo quando nenhum interruptor está ligado.
// Só tinta com RGB cadastrado entra: tinta sem cor viraria candidato "preto"
// silencioso na mistura.
func (s *PaintService) catalogoInteiroComCor() ([]mix.PaintInput, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.code, pc.rgb_r, pc.rgb_g, pc.rgb_b
		FROM paints p
		JOIN paint_colors pc ON pc.paint_id = p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.R, &p.G, &p.B); err != nil {
			return nil, err
		}
		paints = append(paints, p)
	}
	return paints, rows.Err()
}

// estoqueDoFabricante é a INTERSEÇÃO dos dois interruptores: as tintas que o
// usuário tem, daquele fabricante. O estoque guarda `manufacturer_id`, então o
// corte é direto — não se casa por nome, que varia em grafia.
func (s *PaintService) estoqueDoFabricante(fabricanteID int64) ([]mix.PaintInput, error) {
	rows, err := s.db.Query(`
		SELECT id, name, COALESCE(code, ''), rgb_r, rgb_g, rgb_b
		FROM user_paints
		WHERE manufacturer_id = ?
	`, fabricanteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.R, &p.G, &p.B); err != nil {
			return nil, err
		}
		paints = append(paints, p)
	}
	return paints, rows.Err()
}

// ResolverCorNoUniverso é o caminho do rf-11: monta o universo pelos dois
// interruptores, resolve a cor dentro dele e classifica o acerto.
//
// `foraDoUniverso` é decidido por quem chama — a tela só o liga depois de o
// usuário autorizar a saída no diálogo (RN7/CAN4). O serviço não escolhe sair
// do universo pedido por conta própria.
func (s *PaintService) ResolverCorNoUniverso(r, g, b uint8, fabricanteID int64, soEstoque, foraDoUniverso bool) (EquivalentRecipeDTO, error) {
	pool, err := s.montarUniverso(fabricanteID, soEstoque)
	if err != nil {
		return EquivalentRecipeDTO{}, err
	}

	var nomeFabricante string
	if fabricanteID > 0 {
		s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", fabricanteID).Scan(&nomeFabricante)
	}

	l, a, bLab := color.RGBToLab(r, g, b)
	receita := mix.SuggestBestSubset([3]float64{l, a, bLab}, pool, 1, 0)

	ingredientes := make([]RecipeIngredientDTO, 0, len(receita.Ingredients))
	for _, ing := range receita.Ingredients {
		ingredientes = append(ingredientes, RecipeIngredientDTO{
			PaintID:    ing.Paint.ID,
			Name:       ing.Paint.Name,
			Code:       ing.Paint.Code,
			Percentage: ing.Percentage,
			R:          ing.Paint.R,
			G:          ing.Paint.G,
			B:          ing.Paint.B,
		})
	}

	faixa := ClassificarFaixa(receita.DeltaE)

	return EquivalentRecipeDTO{
		SourceName:         "cor alvo",
		SourceR:            r,
		SourceG:            g,
		SourceB:            b,
		TargetManufacturer: nomeFabricante,
		Ingredients:        ingredientes,
		ResultR:            receita.ResultR,
		ResultG:            receita.ResultG,
		ResultB:            receita.ResultB,
		DeltaE:             receita.DeltaE,
		Method:             receita.Method,
		// Reproducible mantém a semântica original (equivalence.MaxViableDeltaE,
		// usada por T1/T3): não é o mesmo corte da faixa de qualidade do rf-11.
		// Trocar isso mudaria "reproducible" nas telas antigas sem pedido.
		Reproducible:   receita.DeltaE <= equivalence.MaxViableDeltaE,
		Tips:           mix.GenerateTips(r, g, b, receita),
		Faixa:          faixa,
		ForaDoUniverso: foraDoUniverso,
	}, nil
}

// MelhorDeltaENoUniverso devolve só o ΔE00 alcançável numa combinação, sem
// montar a receita — é o que o diálogo de fallback mostra em cada saída antes
// de o usuário escolher (RN6/CA12).
func (s *PaintService) MelhorDeltaENoUniverso(r, g, b uint8, fabricanteID int64, soEstoque bool) (float64, error) {
	pool, err := s.montarUniverso(fabricanteID, soEstoque)
	if err != nil {
		return 0, err
	}
	l, a, bLab := color.RGBToLab(r, g, b)
	return mix.SuggestBestSubset([3]float64{l, a, bLab}, pool, 1, 0).DeltaE, nil
}
