package httpapi

import (
	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

func handleListPlans(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		plans, err := svc.ListPlans()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, plans)
	}
}

func handleSavePlan(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var plan service.PaintingPlanDTO
		if !readJSON(ctx, &plan) {
			return
		}
		saved, err := svc.SavePlan(plan)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, saved)
	}
}

func handleLoadPlan(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		plan, err := svc.LoadPlan(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, plan)
	}
}

func handleDeletePlan(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		if err := svc.DeletePlan(id); err != nil {
			writeError(ctx, fasthttp.StatusNotFound, err.Error())
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}
}
