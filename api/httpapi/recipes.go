package httpapi

import (
	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// handleListRecipes atende GET /recipes (receitas salvas, RF-04 T3 — não
// confundir com /recipes/by-paint e /recipes/by-color, que resolvem
// equivalência entre tintas do catálogo).
func handleListRecipes(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		recipes, err := svc.ListRecipes()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipes)
	}
}

// handleSaveRecipe atende POST /recipes {"name":"...","targetHex":"#rrggbb"}.
func handleSaveRecipe(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			Name      string `json:"name"`
			TargetHex string `json:"targetHex"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		recipe, err := svc.SaveRecipe(body.Name, body.TargetHex)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// handleDeleteRecipe atende DELETE /recipes/{id}.
func handleDeleteRecipe(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		if err := svc.DeleteRecipe(id); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]bool{"ok": true})
	}
}

// handleResolveRecipe atende GET /recipes/{id}/resolve?targetManufacturerId=.
// Reabre a receita salva e resolve a fórmula de novo no fabricante escolhido
// — nunca devolve uma fórmula congelada (RG-16 de docs/mesclaai-userstory.md).
func handleResolveRecipe(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		targetMfrID := queryInt64(ctx, "targetManufacturerId", 0)
		if targetMfrID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "targetManufacturerId é obrigatório")
			return
		}
		recipe, err := svc.ResolveSavedRecipe(id, targetMfrID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// handleAddManufacturer atende POST /manufacturers {"name":"..."} (RF-04 T4,
// US-13 — fabricante fora do catálogo seedado).
func handleAddManufacturer(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			Name string `json:"name"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		mfr, err := svc.AddManufacturer(body.Name)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, mfr)
	}
}

// handleDeleteManufacturer atende DELETE /manufacturers/{id} — cascata nas
// tintas do usuário (user_paints); recusa se o fabricante tem catálogo
// seedado (RG-18, US-13).
func handleDeleteManufacturer(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		result, err := svc.DeleteManufacturer(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, result)
	}
}
