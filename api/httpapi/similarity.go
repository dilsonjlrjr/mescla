package httpapi

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// handleFindSimilar atende GET /similar?r=&g=&b=&maxDeltaE=&maxResults=.
func handleFindSimilar(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		maxDeltaE := queryFloat64(ctx, "maxDeltaE", 10.0)
		maxResults := queryInt(ctx, "maxResults", 10)

		results, err := svc.FindSimilar(r, g, b, maxDeltaE, maxResults)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleEquivalences atende GET /paints/{id}/equivalences.
func handleEquivalences(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id, ok := pathInt64(ctx, "id")
		if !ok {
			return
		}
		results, err := svc.FindEquivalences(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleRecipeByPaint atende GET /recipes/by-paint?sourcePaintId=&targetManufacturerId=&maxIngredients=.
// rf-13: targetManufacturerId deixou de ser obrigatório — ausente é catálogo
// inteiro, cross-brand (RG-13).
func handleRecipeByPaint(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		sourceID := queryInt64(ctx, "sourcePaintId", 0)

		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}
		maxIngredients, ok := queryMaxIngredients(ctx)
		if !ok {
			writeError(ctx, fasthttp.StatusBadRequest, "maxIngredients fora da faixa")
			return
		}

		recipe, err := svc.SuggestEquivalentRecipe(sourceID, targetMfrID, maxIngredients)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// queryManufacturerID lê targetManufacturerId distinguindo "ausente" (0, sem
// fabricante — CA17) de "presente mas inválido" (400): um id gigante ou não
// numérico não pode virar silenciosamente "sem fabricante base" e cair no
// catálogo inteiro (CAN2/CAN3). queryInt64 sozinho devolveria 0 nos dois
// casos, escondendo o erro de quem chamou.
func queryManufacturerID(ctx *fasthttp.RequestCtx) (id int64, valido bool) {
	raw := strings.TrimSpace(string(ctx.QueryArgs().Peek("targetManufacturerId")))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// queryMaxIngredients lê maxIngredients (rf-13 RN5): ausente é 0, sem teto —
// preserva o comportamento de T2/T3. Presente fora de 0-8, ou não numérico, é
// inválido (CAN2).
func queryMaxIngredients(ctx *fasthttp.RequestCtx) (n int, valido bool) {
	raw := strings.TrimSpace(string(ctx.QueryArgs().Peek("maxIngredients")))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > 8 {
		return 0, false
	}
	return v, true
}

// handleRecipeByColor atende GET /recipes/by-color?r=&g=&b=&targetManufacturerId=&useStockOnly=&foraDoUniverso=&maxIngredients=
// — usada pela Roda de cores, onde o "alvo" é um passo calculado, não uma
// tinta do catálogo. rf-11: targetManufacturerId deixou de ser obrigatório —
// sem ele e sem useStockOnly, o universo é o catálogo inteiro.
func handleRecipeByColor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")

		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}
		soEstoque := queryBool(ctx, "useStockOnly")
		foraDoUniverso := queryBool(ctx, "foraDoUniverso")
		maxIngredients, ok := queryMaxIngredients(ctx)
		if !ok {
			writeError(ctx, fasthttp.StatusBadRequest, "maxIngredients fora da faixa")
			return
		}

		recipe, err := svc.ResolverCorNoUniverso(r, g, b, targetMfrID, soEstoque, foraDoUniverso, maxIngredients)
		if err != nil {
			// Universo vazio não é erro de servidor (RN9): a tela mostra a
			// mensagem e oferece as saídas do diálogo.
			var vazio service.ErrUniversoVazio
			if errors.As(err, &vazio) {
				writeJSON(ctx, fasthttp.StatusOK, map[string]any{
					"universoVazio": true,
					"motivo":        vazio.Motivo,
				})
				return
			}
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// maxDeviceStockItems é o teto de itens de estoque aceito em
// POST /recipes/by-color (rf-16 T2, CAN4): acima disso o custo do motor
// passaria do catálogo inteiro que o GET já aceita.
const maxDeviceStockItems = 2000

// colorComponentValid reporta se v cabe em 0–255 — usado na validação da cor
// alvo e de cada item do estoque de POST /recipes/by-color. Duplicado do
// homônimo em api/service/planning.go de propósito: são pacotes diferentes,
// e a função é pequena demais para justificar um pacote exportado só para
// isso.
func colorComponentValid(v int) bool {
	return v >= 0 && v <= 255
}

// recipeByColorStockItem é um item do estoque do aparelho (rf-16 T2, RN14) —
// campos de cor em int, não uint8, para o decode nunca recusar o corpo
// inteiro por um valor fora da faixa: a validação roda depois, em Go, e o
// item inválido é só descartado (CAN5).
type recipeByColorStockItem struct {
	ID             int64  `json:"id"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	R              int    `json:"r"`
	G              int    `json:"g"`
	B              int    `json:"b"`
}

// recipeByColorRequest é o corpo de POST /recipes/by-color (rf-16 T2, RN14):
// o universo vem do estoque do aparelho mandado pelo cliente, não de
// user_paints — é a correção do item 1.6 (T2 lia a fonte errada de estoque).
type recipeByColorRequest struct {
	R                    int                      `json:"r"`
	G                    int                      `json:"g"`
	B                    int                      `json:"b"`
	TargetManufacturerID int64                    `json:"targetManufacturerId"`
	ForaDoUniverso       bool                     `json:"foraDoUniverso"`
	MaxIngredients       int                      `json:"maxIngredients"`
	Stock                []recipeByColorStockItem `json:"stock"`
}

// handleRecipeByColorFromDeviceStock atende POST /recipes/by-color (rf-16
// T2, RN14). Decodifica o corpo direto com encoding/json — nunca com
// readJSON, que ecoaria o erro do decode (rf-16 CAN5) — e devolve a mesma
// forma de EquivalentRecipeDTO do GET (rf-11), calculada contra o estoque do
// corpo em vez do universo montado no servidor. GET /recipes/by-color
// (handleRecipeByColor) fica intacto: mesma rota, método diferente.
func handleRecipeByColorFromDeviceStock(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var req recipeByColorRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "pedido inválido")
			return
		}

		if len(req.Stock) > maxDeviceStockItems {
			writeError(ctx, fasthttp.StatusBadRequest, "estoque grande demais para o cálculo")
			return
		}
		if !colorComponentValid(req.R) || !colorComponentValid(req.G) || !colorComponentValid(req.B) {
			writeError(ctx, fasthttp.StatusBadRequest, "cor inválida")
			return
		}
		if req.MaxIngredients < 0 || req.MaxIngredients > 8 {
			writeError(ctx, fasthttp.StatusBadRequest, "maxIngredients fora da faixa")
			return
		}
		if req.TargetManufacturerID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}

		itens := make([]service.DeviceStockItemInput, 0, len(req.Stock))
		for _, it := range req.Stock {
			// Item inválido é descartado, nunca recusa o pedido inteiro
			// (CAN5) — nem o texto dele é ecoado na resposta.
			if !colorComponentValid(it.R) || !colorComponentValid(it.G) || !colorComponentValid(it.B) {
				continue
			}
			if utf8.RuneCountInString(it.Name) > 200 ||
				utf8.RuneCountInString(it.Code) > 60 ||
				utf8.RuneCountInString(it.Manufacturer) > 120 {
				continue
			}
			itens = append(itens, service.DeviceStockItemInput{
				ID: it.ID, ManufacturerID: it.ManufacturerID, Manufacturer: it.Manufacturer,
				Name: it.Name, Code: it.Code, R: it.R, G: it.G, B: it.B,
			})
		}

		recipe, err := svc.ResolverCorComEstoqueDoAparelho(
			uint8(req.R), uint8(req.G), uint8(req.B),
			req.TargetManufacturerID, req.ForaDoUniverso, req.MaxIngredients, itens,
		)
		if err != nil {
			var vazio service.ErrUniversoVazio
			if errors.As(err, &vazio) {
				writeJSON(ctx, fasthttp.StatusOK, map[string]any{
					"universoVazio": true,
					"motivo":        vazio.Motivo,
				})
				return
			}
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, recipe)
	}
}

// handleBestDeltaE atende GET /recipes/best-delta-e — o ΔE00 alcançável numa
// combinação, sem montar a receita. É o número que o diálogo de fallback
// mostra em cada saída antes de o usuário escolher (rf-11 RN6).
func handleBestDeltaE(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		targetMfrID, ok := queryManufacturerID(ctx)
		if !ok || targetMfrID < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "fabricante não encontrado")
			return
		}

		delta, err := svc.MelhorDeltaENoUniverso(r, g, b, targetMfrID, queryBool(ctx, "useStockOnly"))
		if err != nil {
			var vazio service.ErrUniversoVazio
			if errors.As(err, &vazio) {
				writeJSON(ctx, fasthttp.StatusOK, map[string]any{
					"universoVazio": true,
					"motivo":        vazio.Motivo,
				})
				return
			}
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"deltaE": delta})
	}
}

// handlePickColor atende GET /color-pick?r=&g=&b=&targetManufacturerId=.
func handlePickColor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, g, b := queryUint8(ctx, "r"), queryUint8(ctx, "g"), queryUint8(ctx, "b")
		targetMfrID := queryInt64(ctx, "targetManufacturerId", 0)
		resp, err := svc.PickColor(r, g, b, targetMfrID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, resp)
	}
}

// handleCompareToAnchor atende GET /compare-to-anchor?anchorId=&ids=1,2,3.
func handleCompareToAnchor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		anchorID := queryInt64(ctx, "anchorId", 0)
		ids := queryInt64List(ctx, "ids")
		if anchorID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "anchorId é obrigatório")
			return
		}
		results, err := svc.CompareToAnchor(anchorID, ids)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}

// handleBestBrandsFor atende GET /best-brands?paintId=.
func handleBestBrandsFor(svc *service.PaintService) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		paintID := queryInt64(ctx, "paintId", 0)
		if paintID == 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "paintId é obrigatório")
			return
		}
		results, err := svc.BestBrandsFor(paintID)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, results)
	}
}
