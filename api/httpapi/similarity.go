package httpapi

import (
	"errors"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// handleFindSimilar atende GET /similar?r=&g=&b=&maxDeltaE=&maxResults=.
func handleFindSimilar(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		maxDeltaE := queryFloat64(ctx, "maxDeltaE", 10.0)
		maxResults := queryInt(ctx, "maxResults", 10)

		results, err := svc.FindSimilar(r, g, b, maxDeltaE, maxResults)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleEquivalences atende GET /paints/{id}/equivalences.
func handleEquivalences(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		results, err := svc.FindEquivalences(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleRecipeByPaint atende GET /recipes/by-paint?sourcePaintId=&targetManufacturerId=&maxIngredients=.
// rf-13: targetManufacturerId deixou de ser obrigatório — ausente é catálogo
// inteiro, cross-brand (RG-13).
func handleRecipeByPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		sourceID := queryInt64(ctx, "sourcePaintId", 0)

		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}
		maxIngredients, ok := queryMaxIngredients(ctx)
		if !ok {
			writeError(ctx, fasthttp.StatusBadRequest, "maxIngredients fora da faixa")
			return
		}

		recipe, err := svc.SuggestEquivalentRecipe(sourceID, targetMfrID, maxIngredients)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// queryManufacturerID lê targetManufacturerId distinguindo "ausente" (0, sem
// fabricante — CA17) de "presente mas inválido" (400): um id gigante ou não
// numérico não pode virar silenciosamente "sem fabricante base" e cair no
// catálogo inteiro (CAN2/CAN3). queryInt64 sozinho devolveria 0 nos dois
// casos, escondendo o erro de quem chamou.
func queryManufacturerID(ctx *fasthttp.RequestCtx) (id int64, valido bool) {
	raw := strings.TrimSpace(string(ctx.QueryArgs().Peek("targetManufacturerId")))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// queryMaxIngredients lê maxIngredients (rf-13 RN5): ausente é 0, sem teto —
// preserva o comportamento de T2/T3. Presente fora de 0-8, ou não numérico, é
// inválido (CAN2).
func queryMaxIngredients(ctx *fasthttp.RequestCtx) (n int, valido bool) {
	raw := strings.TrimSpace(string(ctx.QueryArgs().Peek("maxIngredients")))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > 8 {
		return 0, false
	}
	return v, true
}

// handleRecipeByColor atende GET /recipes/by-color?r=&g=&b=&targetManufacturerId=&useStockOnly=&foraDoUniverso=&maxIngredients=
// — usada pela Roda de cores, onde o "alvo" é um passo calculado, não uma
// tinta do catálogo. rf-11: targetManufacturerId deixou de ser obrigatório —
// sem ele e sem useStockOnly, o universo é o catálogo inteiro.
func handleRecipeByColor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")

		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}
		soEstoque := queryBool(ctx, "useStockOnly")
		foraDoUniverso := queryBool(ctx, "foraDoUniverso")
		maxIngredients, ok := queryMaxIngredients(ctx)
		if !ok {
			writeError(ctx, fasthttp.StatusBadRequest, "maxIngredients fora da faixa")
			return
		}

		recipe, err := svc.ResolverCorNoUniverso(r, g, b, targetMfrID, soEstoque, foraDoUniverso, maxIngredients)
		if err != nil {
			// Universo vazio não é erro de servidor (RN9): a tela mostra a
			// mensagem e oferece as saídas do diálogo.
			var vazio service.ErrUniversoVazio
			if errors.As(err, &vazio) {
				writeJSON(ctx, fasthttp.StatusOK, map[string]any{
					"universoVazio": true,
					"motivo":        vazio.Motivo,
				})
				return
			}
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// handleBestDeltaE atende GET /recipes/best-delta-e — o ΔE00 alcançável numa
// combinação, sem montar a receita. É o número que o diálogo de fallback
// mostra em cada saída antes de o usuário escolher (rf-11 RN6).
func handleBestDeltaE(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}

		delta, err := svc.MelhorDeltaENoUniverso(r, g, b, targetMfrID, queryBool(ctx, "useStockOnly"))
		if err != nil {
			var vazio service.ErrUniversoVazio
			if errors.As(err, &vazio) {
				writeJSON(ctx, fasthttp.StatusOK, map[string]any{
					"universoVazio": true,
					"motivo":        vazio.Motivo,
				})
				return
			}
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"deltaE": delta})
	}
}

// handlePickColor atende GET /color-pick?r=&g=&b=&targetManufacturerId=.
func handlePickColor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		targetMfrID := queryInt64(ctx, "targetManufacturerId", 0)
		resp, err := svc.PickColor(r, g, b, targetMfrID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, resp)
	}
}

// handleCompareToAnchor atende GET /compare-to-anchor?anchorId=&ids=1,2,3.
func handleCompareToAnchor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		anchorID := queryInt64(ctx, "anchorId", 0)
		ids := queryInt64List(ctx, "ids")
		if anchorID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "anchorId é obrigatório")
			return
		}
		results, err := svc.CompareToAnchor(anchorID, ids)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleBestBrandsFor atende GET /best-brands?paintId=.
func handleBestBrandsFor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		paintID := queryInt64(ctx, "paintId", 0)
		if paintID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "paintId é obrigatório")
			return
		}
		results, err := svc.BestBrandsFor(paintID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}
