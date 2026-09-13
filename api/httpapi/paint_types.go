package httpapi

import (
	"errors"
	"log"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// Rotas de tipo de tinta (rf-15), com o mesmo contrato das de fabricante: o
// corpo aceita só "name", o id vem do caminho e as mensagens de erro são fixas.

func handlePaintTypes(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		types, err := svc.GetPaintTypes()
		if err != nil {
			writePaintTypeError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, types)
	}
}

func handleAddPaintType(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body manufacturerBody
		if !readJSON(ctx, &body) {
			return
		}
		pt, err := svc.AddPaintType(body.Name)
		if err != nil {
			writePaintTypeError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, pt)
	}
}

func handleUpdatePaintType(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := positiveID(ctx)
		if !ok {
			return
		}
		var body manufacturerBody
		if !readJSON(ctx, &body) {
			return
		}
		pt, err := svc.UpdatePaintType(id, body.Name)
		if err != nil {
			writePaintTypeError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, pt)
	}
}

func handleDeletePaintType(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := positiveID(ctx)
		if !ok {
			return
		}
		if err := svc.DeletePaintType(id); err != nil {
			writePaintTypeError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]int64{"id": id})
	}
}

func writePaintTypeError(ctx *fasthttp.RequestCtx, err error) {
	var nameErr *service.NameError
	var inUseErr *service.PaintTypeInUseError
	switch {
	case errors.As(err, &nameErr):
		writeError(ctx, fasthttp.StatusBadRequest, nameErr.Msg)
	case errors.As(err, &inUseErr):
		writeJSON(ctx, fasthttp.StatusConflict, map[string]any{
			"error":  inUseErr.Error(),
			"paints": inUseErr.Paints,
			"stock":  inUseErr.Stock,
		})
	case errors.Is(err, service.ErrPaintTypeNotFound):
		writeError(ctx, fasthttp.StatusNotFound, "Tipo de tinta não encontrado.")
	case errors.Is(err, service.ErrPaintTypeDuplicate):
		writeError(ctx, fasthttp.StatusConflict, "Já existe um tipo de tinta com esse nome.")
	default:
		log.Printf("[paint-types] %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "Não foi possível gravar o tipo de tinta.")
	}
}
