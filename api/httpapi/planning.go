package httpapi

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// handleListPlans atende GET /plans com o resumo da lista de projetos
// (rf-16 T2): nunca imagem nem regiões completas. Chama ListPlanSummaries,
// não ListPlans — este último fica intacto porque o binding Wails
// (project/wails) é gerado contra a assinatura dele.
func handleListPlans(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		plans, err := svc.ListPlanSummaries()
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

// reportTimeout é o teto de geração do relatório (rf-08, RN13): estourado, a
// resposta é 500 com a mensagem fixa e nada fica pendurado no servidor.
const reportTimeout = 10 * time.Second

// reportGenErrMsg é a mensagem fixa de falha de geração — nunca repassa
// err.Error() (rf-08, evita repetir o achado S-001).
const reportGenErrMsg = "não foi possível gerar o relatório"

// handleReportPlan exporta o relatório do plano salvo (rf-08). GET sem corpo,
// sem gravação: lê o plano já persistido e devolve PDF ou PNG como download,
// sem guardar nada em disco.
func handleReportPlan(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		idRaw, _ := ctx.UserValue("id").(string)
		id, convErr := strconv.ParseInt(idRaw, 10, 64)
		if convErr != nil || id <= 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "id inválido")
			return
		}

		format := string(ctx.QueryArgs().Peek("format"))
		if format == "" {
			format = "pdf"
		}
		if format != "pdf" && format != "png" {
			writeError(ctx, fasthttp.StatusBadRequest, "formato inválido")
			return
		}

		reqCtx, cancel := context.WithTimeout(context.Background(), reportTimeout)
		defer cancel()

		type reportResult struct {
			body     []byte
			filename string
			err      error
		}
		resultCh := make(chan reportResult, 1) // bufferizado: goroutine nunca fica presa no envio
		go func() {
			body, filename, err := svc.BuildPlanReport(reqCtx, id, format)
			resultCh <- reportResult{body: body, filename: filename, err: err}
		}()

		var res reportResult
		select {
		case res = <-resultCh:
		case <-reqCtx.Done():
			writeError(ctx, fasthttp.StatusInternalServerError, reportGenErrMsg)
			return
		}

		if res.err != nil {
			if errors.Is(res.err, service.ErrPlanNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "plano não encontrado")
				return
			}
			writeError(ctx, fasthttp.StatusInternalServerError, reportGenErrMsg)
			return
		}

		contentType := "application/pdf"
		if format == "png" {
			contentType = "image/png"
		}

		filename := safeReportFilename(res.filename, format)
		ctx.Response.Header.Set("Content-Disposition", contentDispositionHeader(filename))
		ctx.Response.Header.Set("Cache-Control", "no-store")
		ctx.SetContentType(contentType)
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(res.body)
	}
}

// safeReportFilename valida o nome vindo do serviço antes de virar cabeçalho
// HTTP (rf-08, CAN2/CAN3): nada de CR, LF ou aspas. Fora disso, cai num nome
// padrão em vez de recusar o download inteiro.
func safeReportFilename(filename, format string) string {
	ext := "pdf"
	if format == "png" {
		ext = "png"
	}
	if filename == "" || strings.ContainsAny(filename, "\r\n\"") {
		return "mescla-plano-" + time.Now().UTC().Format("2006-01-02") + "." + ext
	}
	return filename
}

// contentDispositionHeader monta o cabeçalho attachment com a forma percent-
// encoded (RFC 5987) para nomes fora de ASCII — o filename já passou por
// safeReportFilename.
func contentDispositionHeader(filename string) string {
	encoded := strings.ReplaceAll(url.QueryEscape(filename), "+", "%20")
	return `attachment; filename="` + filename + `"; filename*=UTF-8''` + encoded
}
