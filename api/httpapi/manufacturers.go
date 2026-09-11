package httpapi

import (
	"errors"
	"log"
	"strconv"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// Rotas de escrita de fabricante (rf-14). O corpo aceita só "name": o id vem
// do caminho e o resto do payload é ignorado. As mensagens de erro são fixas —
// o texto do SQLite nunca chega ao cliente.

type manufacturerBody struct {
	Name string `json:"name"`
}

// handleAddManufacturer atende POST /manufacturers {"name":"..."}.
func handleAddManufacturer(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body manufacturerBody
		if !readJSON(ctx, &body) {
			return
		}
		mfr, err := svc.AddManufacturer(body.Name)
		if err != nil {
			writeManufacturerError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, mfr)
	}
}

// handleUpdateManufacturer atende PUT /manufacturers/{id} {"name":"..."}.
func handleUpdateManufacturer(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := positiveID(ctx)
		if !ok {
			return
		}
		var body manufacturerBody
		if !readJSON(ctx, &body) {
			return
		}
		mfr, err := svc.UpdateManufacturer(id, body.Name)
		if err != nil {
			writeManufacturerError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, mfr)
	}
}

// handleDeleteManufacturer atende DELETE /manufacturers/{id} — sem cascata:
// fabricante com tinta ou em uso responde 409 (RG-18).
func handleDeleteManufacturer(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := positiveID(ctx)
		if !ok {
			return
		}
		if err := svc.DeleteManufacturer(id); err != nil {
			writeManufacturerError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]int64{"id": id})
	}
}

func positiveID(ctx *fasthttp.RequestCtx) (int64, bool) {
	id, ok := pathInt64(ctx, "id")
	if !ok {
		return 0, false
	}
	if id <= 0 {
		writeError(ctx, fasthttp.StatusBadRequest, "id inválido: "+strconv.FormatInt(id, 10))
		return 0, false
	}
	return id, true
}

func writeManufacturerError(ctx *fasthttp.RequestCtx, err error) {
	var nameErr *service.NameError
	var paintsErr *service.ManufacturerHasPaintsError
	switch {
	case errors.As(err, &nameErr):
		writeError(ctx, fasthttp.StatusBadRequest, nameErr.Msg)
	case errors.As(err, &paintsErr):
		writeError(ctx, fasthttp.StatusConflict, paintsErr.Error())
	case errors.Is(err, service.ErrManufacturerNotFound):
		writeError(ctx, fasthttp.StatusNotFound, "Fabricante não encontrado.")
	case errors.Is(err, service.ErrManufacturerDuplicate):
		writeError(ctx, fasthttp.StatusConflict, "Já existe um fabricante com esse nome.")
	case errors.Is(err, service.ErrManufacturerInUse):
		writeError(ctx, fasthttp.StatusConflict, "O fabricante está em uso e não pode ser excluído.")
	default:
		log.Printf("[manufacturers] %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "Não foi possível gravar o fabricante.")
	}
}
