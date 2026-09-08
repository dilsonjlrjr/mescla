package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

func ctxByPaint(q string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/recipes/by-paint?" + q)
	return ctx
}

// primeiraTintaComCor devolve o id da primeira tinta do catálogo embutido com
// cor cadastrada — evita depender de um id fixo do catálogo real.
func primeiraTintaComCor(t *testing.T, svc *service.PaintService) int64 {
	t.Helper()
	paints, err := svc.GetAllPaints()
	if err != nil {
		t.Fatalf("GetAllPaints: %v", err)
	}
	for _, p := range paints {
		if p.R != 0 || p.G != 0 || p.B != 0 {
			return p.ID
		}
	}
	t.Fatal("nenhuma tinta com cor no catálogo embutido")
	return 0
}

// CA3 — GET /recipes/by-paint sem targetManufacturerId: pool é o catálogo
// inteiro, status 200 (antes do rf-13 a rota exigia o fabricante).
func TestRF13CA3ByPaintSemTargetManufacturerIdUsaCatalogoInteiro(t *testing.T) {
	svc := servicoDeTeste(t)
	sourceID := primeiraTintaComCor(t, svc)

	ctx := ctxByPaint(fmt.Sprintf("sourcePaintId=%d", sourceID))
	handleRecipeByPaint(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if _, ok := resp["ingredients"]; !ok {
		t.Errorf("resposta sem ingredients: %s", ctx.Response.Body())
	}
}

// CAN1 — targetManufacturerId gigante ou não numérico é 400 "fabricante não
// encontrado", nunca 200 com o catálogo inteiro. A guarda roda antes de tocar
// o serviço, então nil é seguro (mesmo padrão de CAN2 em by-color).
func TestRF13CAN1TargetManufacturerIdInvalidoNuncaViraCatalogoInteiro(t *testing.T) {
	for _, valor := range []string{"99999999999999999999", "abc"} {
		ctx := ctxByPaint("sourcePaintId=1&targetManufacturerId=" + valor)
		handleRecipeByPaint(nil)(ctx)

		if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
			t.Errorf("targetManufacturerId=%s: esperava 400, veio %d", valor, ctx.Response.StatusCode())
		}
		corpo := string(ctx.Response.Body())
		if !strings.Contains(corpo, "fabricante não encontrado") {
			t.Errorf("targetManufacturerId=%s: mensagem inesperada: %s", valor, corpo)
		}
	}
}

// CAN2 — maxIngredients fora de 0-8, ou não numérico, é 400 e nenhuma receita
// é montada.
func TestRF13CAN2MaxIngredientsForaDaFaixa(t *testing.T) {
	for _, valor := range []string{"-1", "99", "três"} {
		ctx := ctxByPaint("sourcePaintId=1&maxIngredients=" + valor)
		handleRecipeByPaint(nil)(ctx)

		if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
			t.Errorf("maxIngredients=%s: esperava 400, veio %d", valor, ctx.Response.StatusCode())
		}
	}
}

// CAN5 — parâmetro fora da allowlist (ex.: poolIds) é ignorado; o pool
// continua montado no servidor pelos parâmetros de universo.
func TestRF13CAN5ParametroForaDaAllowlistEIgnorado(t *testing.T) {
	svc := servicoDeTeste(t)
	sourceID := primeiraTintaComCor(t, svc)

	ctx := ctxByPaint(fmt.Sprintf("sourcePaintId=%d&poolIds=1,2,3", sourceID))
	handleRecipeByPaint(svc)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("parâmetro fora da allowlist não pode quebrar a rota: %d: %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}
