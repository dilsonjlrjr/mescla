package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"

	apidb "paint-match-ai/api/db"
	"paint-match-ai/api/service"
)

func ctxComQuery(q string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/recipes/by-color?" + q)
	return ctx
}

// CAN1 — parâmetro de modo com lixo conta como DESLIGADO. Um valor que o
// cliente errou nunca pode ligar um modo sozinho.
func TestCAN1QueryBoolSoAceitaValorExplicito(t *testing.T) {
	casos := map[string]bool{
		"useStockOnly=1":       true,
		"useStockOnly=true":    true,
		"useStockOnly=TRUE":    true,
		"useStockOnly=%201%20": true,
		"useStockOnly=0":       false,
		"useStockOnly=false":   false,
		"useStockOnly=banana":  false,
		"useStockOnly=":        false,
		"outroParam=1":         false,
	}
	for q, esperado := range casos {
		if got := queryBool(ctxComQuery(q), "useStockOnly"); got != esperado {
			t.Errorf("%s: esperava %v, veio %v", q, esperado, got)
		}
	}
}

// CAN2 — fabricante negativo é recusado sem ecoar o valor recebido. A guarda
// roda antes de o handler tocar o serviço, então `nil` aqui é seguro.
func TestCAN2FabricanteNegativoRecusadoSemEcoar(t *testing.T) {
	for _, rota := range []struct {
		nome    string
		handler fasthttp.RequestHandler
	}{
		{"by-color", handleRecipeByColor(nil)},
		{"best-delta-e", handleBestDeltaE(nil)},
	} {
		ctx := ctxComQuery("r=10&g=20&b=30&targetManufacturerId=-7")
		rota.handler(ctx)

		if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
			t.Errorf("%s: esperava 400, veio %d", rota.nome, ctx.Response.StatusCode())
		}
		corpo := string(ctx.Response.Body())
		if !strings.Contains(corpo, "fabricante não encontrado") {
			t.Errorf("%s: mensagem inesperada: %s", rota.nome, corpo)
		}
		if strings.Contains(corpo, "-7") {
			t.Errorf("%s: a resposta não pode ecoar o id recebido: %s", rota.nome, corpo)
		}
	}
}

// Testes de ponta a ponta dos dois handlers com um serviço real, sobre o
// catálogo embutido — a única forma de exercitar o caminho feliz (o resto do
// arquivo já testava só os desvios de validação, que rodam antes de tocar o
// serviço).
func servicoDeTeste(t *testing.T) *service.PaintService {
	t.Helper()
	svc, err := service.NewPaintService(apidb.EmbeddedSeed())
	if err != nil {
		t.Fatalf("NewPaintService: %v", err)
	}
	return svc
}

func TestHandleRecipeByColorCaminhoFeliz(t *testing.T) {
	svc := servicoDeTeste(t)

	ctx := ctxComQuery("r=74&g=124&b=47")
	handleRecipeByColor(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if _, ok := resp["deltaE"]; !ok {
		t.Errorf("resposta sem deltaE: %s", ctx.Response.Body())
	}
}

func TestHandleRecipeByColorFabricanteInexistente(t *testing.T) {
	svc := servicoDeTeste(t)

	ctx := ctxComQuery("r=74&g=124&b=47&targetManufacturerId=999999")
	handleRecipeByColor(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d", ctx.Response.StatusCode())
	}
}

func TestHandleBestDeltaECaminhoFeliz(t *testing.T) {
	svc := servicoDeTeste(t)

	ctx := ctxComQuery("r=74&g=124&b=47")
	handleBestDeltaE(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if _, ok := resp["deltaE"]; !ok {
		t.Errorf("resposta sem deltaE: %s", ctx.Response.Body())
	}
}

func TestHandleBestDeltaEFabricanteInexistente(t *testing.T) {
	svc := servicoDeTeste(t)

	ctx := ctxComQuery("r=74&g=124&b=47&targetManufacturerId=999999")
	handleBestDeltaE(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d", ctx.Response.StatusCode())
	}
}

// Universo vazio (estoque ligado e sem tinta nenhuma) não é erro de servidor
// (RN9): resposta 200 com universoVazio/motivo.
func TestHandleRecipeByColorUniversoVazio(t *testing.T) {
	svc := servicoDeTeste(t)

	ctx := ctxComQuery("r=74&g=124&b=47&useStockOnly=1")
	handleRecipeByColor(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("universo vazio não é erro de servidor, veio %d", ctx.Response.StatusCode())
	}
	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if resp["universoVazio"] != true {
		t.Errorf("esperava universoVazio:true, veio %s", ctx.Response.Body())
	}
}

// CAN2/CAN3 — id gigante ou lixo não pode virar "sem fabricante" (200,
// catálogo inteiro): é presente mas inválido, não ausente.
func TestCAN2CAN3IDInvalidoNaoViraSemFabricante(t *testing.T) {
	for _, valor := range []string{"99999999999999999999999", "abc", "-1"} {
		ctx := ctxComQuery("r=10&g=20&b=30&targetManufacturerId=" + valor)
		handleRecipeByColor(nil)(ctx)
		if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
			t.Errorf("targetManufacturerId=%s deveria ser 400, veio %d", valor, ctx.Response.StatusCode())
		}
	}
}
