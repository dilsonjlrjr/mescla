package httpapi

import (
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

// handleRecipeByPaint atende GET /recipes/by-paint?sourcePaintId=&targetManufacturerId=.
func handleRecipeByPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		sourceID := queryInt64(ctx, "sourcePaintId", 0)
		targetMfrID := queryInt64(ctx, "targetManufacturerId", 0)
		if sourceID == 0 || targetMfrID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "sourcePaintId e targetManufacturerId são obrigatórios")
			return
		}
		recipe, err := svc.SuggestEquivalentRecipe(sourceID, targetMfrID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// handleRecipeByColor atende GET /recipes/by-color?r=&g=&b=&targetManufacturerId=
// — usada pela Roda de cores, onde o "alvo" é um passo calculado, não uma
// tinta do catálogo.
func handleRecipeByColor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		targetMfrID := queryInt64(ctx, "targetManufacturerId", 0)
		if targetMfrID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "targetManufacturerId é obrigatório")
			return
		}
		recipe, err := svc.SuggestRecipeForColor(r, g, b, targetMfrID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
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
