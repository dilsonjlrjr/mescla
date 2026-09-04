package httpapi

import (
	"github.com/valyala/fasthttp"

	"paint-match-ai/api/domain/stock"
	"paint-match-ai/api/service"
)

// --- Estoque STATELESS (front/ — o estoque vive no localStorage do
// navegador, nunca no banco do servidor; estas rotas só validam/calculam) ---

// handleStockCSVTemplate atende GET /stock/csv-template.
func handleStockCSVTemplate(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		csv, err := svc.UserPaintCSVTemplate()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]string{"csv": csv})
	}
}

// handleStockParseCSV atende POST /stock/parse-csv {"csv": "..."} — crítica
// contra os fabricantes do catálogo, sem persistir.
func handleStockParseCSV(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			CSV string `json:"csv"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		paints, rowErrs, err := svc.ValidateStockCSV(body.CSV)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"paints": paints, "errors": rowErrs})
	}
}

// handleStockToCSV atende POST /stock/to-csv {"stock": [...]}.
func handleStockToCSV(ctx *fasthttp.RequestCtx) {
	var body struct {
		Stock []stock.Paint `json:"stock"`
	}
	if !readJSON(ctx, &body) {
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"csv": service.StockToCSV(body.Stock)})
}

// handleStockSuggestRecipe atende POST /stock/suggest-recipe
// {"sourcePaintId": N, "stock": [...]} — receita priorizando o estoque que o
// cliente já tem em mãos (não o do banco).
func handleStockSuggestRecipe(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			SourcePaintID int64         `json:"sourcePaintId"`
			Stock         []stock.Paint `json:"stock"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		recipe, err := svc.SuggestEquivalentFromPool(body.SourcePaintID, body.Stock)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// --- Estoque PERSISTIDO (paridade com o binding Wails do desktop — útil
// para qualquer cliente HTTP que queira estoque guardado no servidor) ---

func handleListUserPaints(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		paints, err := svc.GetUserPaints()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, paints)
	}
}

func handleAddUserPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var p service.UserPaintDTO
		if !readJSON(ctx, &p) {
			return
		}
		saved, err := svc.AddUserPaint(p)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, saved)
	}
}

func handleUpdateUserPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		var p service.UserPaintDTO
		if !readJSON(ctx, &p) {
			return
		}
		p.ID = id
		saved, err := svc.UpdateUserPaint(p)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, saved)
	}
}

func handleDeleteUserPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		if err := svc.DeleteUserPaint(id); err != nil {
			writeError(ctx, fasthttp.StatusNotFound, err.Error())
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}
}

func handleExportUserPaintsCSV(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		csv, err := svc.ExportUserPaintsCSV()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]string{"csv": csv})
	}
}

func handleImportUserPaintsCSV(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			CSV string `json:"csv"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		result, err := svc.ImportUserPaintsCSV(body.CSV)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, result)
	}
}

func handleSuggestFromStock(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		sourceID := queryInt64(ctx, "sourcePaintId", 0)
		if sourceID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "sourcePaintId é obrigatório")
			return
		}
		recipe, err := svc.SuggestEquivalentFromStock(sourceID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}
