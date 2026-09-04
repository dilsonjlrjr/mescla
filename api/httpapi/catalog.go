package httpapi

import (
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
