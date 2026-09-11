package httpapi

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// NewRouter registra todas as rotas do PaintService. front/ consome isto pela
// rede; wails/ nunca importa este pacote — continua com bind direto e
// offline. Prefixo /api/ é responsabilidade de quem publica (nginx faz o
// reverse proxy em produção; ver front/deploy/nginx.conf).
func NewRouter(svc *service.PaintService) *router.Router {
	r := router.New()

	r.GET("/healthz", handleHealthz)

	r.GET("/stats", handleStats(svc))
	r.GET("/manufacturers", handleManufacturers(svc))
	r.POST("/manufacturers", handleAddManufacturer(svc))
	r.PUT("/manufacturers/{id}", handleUpdateManufacturer(svc))
	r.DELETE("/manufacturers/{id}", handleDeleteManufacturer(svc))
	r.GET("/paint-types", handlePaintTypes(svc))
	r.POST("/paint-types", handleAddPaintType(svc))
	r.PUT("/paint-types/{id}", handleUpdatePaintType(svc))
	r.DELETE("/paint-types/{id}", handleDeletePaintType(svc))
	r.GET("/paints", handlePaints(svc))
	r.GET("/paints/{id}", handlePaintByID(svc))
	r.GET("/paints/{id}/equivalences", handleEquivalences(svc))
	r.POST("/paints/compare", handleCompareColors(svc))
	r.POST("/query", handleQuery(svc))

	r.GET("/similar", handleFindSimilar(svc))
	r.GET("/recipes/by-paint", handleRecipeByPaint(svc))
	r.GET("/recipes/by-color", handleRecipeByColor(svc))
	r.GET("/recipes/best-delta-e", handleBestDeltaE(svc))

	// Receitas salvas (RF-04, tela Receitas) — guardam o alvo, nunca a
	// fórmula; resolve-se de novo em /recipes/{id}/resolve.
	r.GET("/recipes", handleListRecipes(svc))
	r.POST("/recipes", handleSaveRecipe(svc))
	r.DELETE("/recipes/{id}", handleDeleteRecipe(svc))
	r.GET("/recipes/{id}/resolve", handleResolveRecipe(svc))
	r.GET("/color-pick", handlePickColor(svc))
	r.GET("/compare-to-anchor", handleCompareToAnchor(svc))
	r.GET("/best-brands", handleBestBrandsFor(svc))

	// Estoque stateless — usado pelo front (estoque no localStorage do navegador).
	r.GET("/stock/csv-template", handleStockCSVTemplate(svc))
	r.POST("/stock/parse-csv", handleStockParseCSV(svc))
	r.POST("/stock/to-csv", handleStockToCSV)
	r.POST("/stock/suggest-recipe", handleStockSuggestRecipe(svc))

	// Estoque persistido no servidor — paridade com o binding Wails do desktop.
	r.GET("/user-paints", handleListUserPaints(svc))
	r.POST("/user-paints", handleAddUserPaint(svc))
	r.PUT("/user-paints/{id}", handleUpdateUserPaint(svc))
	r.DELETE("/user-paints/{id}", handleDeleteUserPaint(svc))
	r.GET("/user-paints/export-csv", handleExportUserPaintsCSV(svc))
	r.POST("/user-paints/import-csv", handleImportUserPaintsCSV(svc))
	r.GET("/user-paints/suggest-recipe", handleSuggestFromStock(svc))

	r.GET("/plans", handleListPlans(svc))
	r.POST("/plans", handleSavePlan(svc))
	r.GET("/plans/{id}", handleLoadPlan(svc))
	r.DELETE("/plans/{id}", handleDeletePlan(svc))
	r.GET("/plans/{id}/report", handleReportPlan(svc))

	return r
}

func handleHealthz(ctx *fasthttp.RequestCtx) {
	writeJSON(ctx, fasthttp.StatusOK, map[string]bool{"ok": true})
}
