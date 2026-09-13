package httpapi

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/domain/stock"
	"paint-match-ai/api/service"
)

// --- Estoque ad-hoc (front/ manda uma lista arbitrária no corpo pra calcular
// em cima dela, sem tocar o banco). rf-17: o estoque do usuário DEIXOU de
// viver só no localStorage — agora mora em user_paints, servido pelas rotas
// PERSISTIDAS mais abaixo; estas aqui continuam stateless de propósito. ---

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

// --- Estoque PERSISTIDO (rf-17): fonte de verdade do estoque do usuário —
// front/ migra o que estava no localStorage e passa a ler/gravar só aqui;
// mesmo caminho que o binding Wails do desktop chama direto, sem HTTP. ---

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

// decodeUserPaintBody decodifica o corpo em v SEM eco: erro de JSON vira 400
// "pedido inválido" fixo — readJSON (json.go) ecoa err.Error(), proibido aqui
// pelo contrato do rf-17 (nunca devolver o que o cliente mandou).
func decodeUserPaintBody(ctx *fasthttp.RequestCtx, v any) bool {
	if err := json.Unmarshal(ctx.PostBody(), v); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "pedido inválido")
		return false
	}
	return true
}

// writeUserPaintError traduz o erro tipado do service (sentinela not-found e
// erro de entrada) pro código HTTP certo, sem casar por texto.
func writeUserPaintError(ctx *fasthttp.RequestCtx, err error) {
	var inputErr *service.UserPaintInputError
	switch {
	case errors.As(err, &inputErr):
		writeError(ctx, fasthttp.StatusBadRequest, inputErr.Msg)
	case errors.Is(err, service.ErrUserPaintNotFound):
		writeError(ctx, fasthttp.StatusNotFound, err.Error())
	default:
		log.Printf("[user-paints] %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "não foi possível gravar a tinta")
	}
}

func handleAddUserPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var p service.UserPaintDTO
		if !decodeUserPaintBody(ctx, &p) {
			return
		}
		p.ID = 0 // contrato: id do corpo é ignorado
		saved, err := svc.AddUserPaint(p)
		if err != nil {
			writeUserPaintError(ctx, err)
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
		if !decodeUserPaintBody(ctx, &p) {
			return
		}
		p.ID = id
		saved, err := svc.UpdateUserPaint(p)
		if err != nil {
			writeUserPaintError(ctx, err)
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
			writeUserPaintError(ctx, err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}
}

// handleMigrateUserPaints atende POST /user-paints/migrate — grava o estoque
// de um navegador (rf-17, RN5/RN6). Corpo externo decodificado sem eco;
// deviceId/paints[] repassados ao service, que valida e roda a transação.
func handleMigrateUserPaints(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			DeviceID string            `json:"deviceId"`
			Paints   []json.RawMessage `json:"paints"`
		}
		if !decodeUserPaintBody(ctx, &body) {
			return
		}
		result, err := svc.MigrateUserPaints(body.DeviceID, body.Paints)
		if err != nil {
			var inputErr *service.UserPaintInputError
			if errors.As(err, &inputErr) {
				writeError(ctx, fasthttp.StatusBadRequest, inputErr.Msg)
				return
			}
			log.Printf("[user-paints] migrate: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "não foi possível migrar o estoque")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, result)
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
