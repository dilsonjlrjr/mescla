package httpapi

import (
	"testing"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// rf-09 CA18 — o plano viaja com as fotos das abas em base64 (até 10 × 2 MB).
// O default do fasthttp é 4 MB e cortaria o corpo antes de a API vê-lo.
func TestNewServerAceitaCorpoDe32MB(t *testing.T) {
	srv := NewServer(&service.PaintService{})

	const esperado = 32 * 1024 * 1024
	if srv.fast.MaxRequestBodySize != esperado {
		t.Fatalf("MaxRequestBodySize = %d, esperado %d", srv.fast.MaxRequestBodySize, esperado)
	}
}

// rf-08 RN12 — o PWA baixa o relatório por fetch e precisa ler o nome do
// arquivo do Content-Disposition; sem expor o cabeçalho, o navegador o
// esconde do JavaScript em origem cruzada (o caso do dev, fora do nginx).
func TestWithCORSExpoeContentDisposition(t *testing.T) {
	handler := withCORS(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
	})

	ctx := &fasthttp.RequestCtx{}
	handler(ctx)

	if got := string(ctx.Response.Header.Peek("Access-Control-Expose-Headers")); got != "Content-Disposition" {
		t.Fatalf("Access-Control-Expose-Headers = %q, esperado Content-Disposition", got)
	}
	if got := string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, esperado *", got)
	}
}

// Preflight responde 204 sem chegar ao handler de trás.
func TestWithCORSPreflightRespondeSemChamarOHandler(t *testing.T) {
	chamou := false
	handler := withCORS(func(ctx *fasthttp.RequestCtx) { chamou = true })

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	handler(ctx)

	if chamou {
		t.Error("preflight não pode alcançar o handler de trás")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, esperado 204", ctx.Response.StatusCode())
	}
}
