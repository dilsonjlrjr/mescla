package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

func handleStats(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		stats, err := svc.GetStats()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, stats)
	}
}

func handleManufacturers(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		mfrs, err := svc.GetManufacturers()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, mfrs)
	}
}

// handlePaints atende GET /paints (lista completa, usada pelo front pra
// popular o catálogo em memória, como antes fazia com catalog.json) e
// GET /paints?q=busca (mesma busca do backend desktop).
func handlePaints(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		q := string(ctx.QueryArgs().Peek("q"))
		var (
			paints []service.PaintDTO
			err    error
		)
		if q != "" {
			paints, err = svc.SearchPaints(q)
		} else {
			paints, err = svc.GetAllPaints()
		}
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, paints)
	}
}

func handlePaintByID(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		p, err := svc.GetPaintByID(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, p)
	}
}

// handleCompareColors atende POST /paints/compare {"paintIds":[1,2,3]} — ΔE de
// cada par, a comparação livre (não contra âncora) que o desktop usa.
func handleCompareColors(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			PaintIDs []int64 `json:"paintIds"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		results, err := svc.CompareColors(body.PaintIDs)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handlePaintIgnoreInMix atende PUT /paints/{id}/ignore-in-mix (rf-19,
// RN11) — rota nova, sem autenticação (risco aceito), só este campo. id
// parseado direto com strconv (não pathInt64, que ecoa o valor bruto no erro)
// para a mensagem de erro ficar fixa e sem eco.
func handlePaintIgnoreInMix(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		raw, _ := ctx.UserValue("id").(string)
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "id inválido")
			return
		}

		var body struct {
			IgnoreInMix *bool `json:"ignoreInMix"`
		}
		if err := json.Unmarshal(ctx.PostBody(), &body); err != nil || body.IgnoreInMix == nil {
			writeError(ctx, fasthttp.StatusBadRequest, "pedido inválido")
			return
		}

		name, err := svc.SetPaintIgnoreInMix(id, *body.IgnoreInMix)
		if err != nil {
			if errors.Is(err, service.ErrPaintNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "tinta não encontrada")
				return
			}
			log.Printf("[paints] ignore-in-mix: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "não foi possível gravar a marca")
			return
		}
		log.Printf("[paints] ignore-in-mix id=%d name=%q ignoreInMix=%v", id, name, *body.IgnoreInMix)
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"id": id, "ignoreInMix": *body.IgnoreInMix})
	}
}

func handleQuery(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var body struct {
			Text string `json:"text"`
		}
		if !readJSON(ctx, &body) {
			return
		}
		resp, err := svc.ProcessQuery(body.Text)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, resp)
	}
}
