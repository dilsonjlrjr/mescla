//go:build js && wasm

// Command wasm é o motor de cor do app mobile (frontend-mobile), compilado
// para WebAssembly. Expõe em window.__mescla as operações matemáticas do
// Mescla — ΔE2000, busca por proximidade e receita equivalente — usando os
// MESMOS pacotes do desktop (pkg/color, pkg/mix, pkg/equivalence), então os
// dois alvos produzem resultados idênticos para a mesma entrada.
//
// O catálogo chega via init(catalogJSON) — o JSON gerado por cmd/export — e
// vive em memória; não há SQLite aqui (modernc.org/sqlite não compila para
// js/wasm). Todas as funções retornam STRINGS JSON (o TS faz JSON.parse);
// erros vêm como {"error": "mensagem"}.
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"syscall/js"

	"paint-match-ai/internal/color"
	"paint-match-ai/internal/equivalence"
	"paint-match-ai/internal/mix"
	"paint-match-ai/internal/stock"
)

// paintRec é uma tinta do catálogo em memória, com Lab precomputado no init
// (o JSON traz só RGB pra economizar payload).
type paintRec struct {
	ID      int64
	MfrID   int64
	Name    string
	Code    string
	Line    string
	R, G, B uint8
	Lab     [3]float64
}

var (
	paints  []paintRec
	byID    map[int64]int    // paint ID → índice em paints
	mfrName map[int64]string // manufacturer ID → nome
)

func main() {
	js.Global().Set("__mescla", js.ValueOf(map[string]any{
		"init":                    js.FuncOf(jsInit),
		"findSimilar":             js.FuncOf(jsFindSimilar),
		"suggestEquivalentRecipe": js.FuncOf(jsSuggestEquivalentRecipe),
		"suggestRecipeForColor":   js.FuncOf(jsSuggestRecipeForColor),
		"suggestFromStock":        js.FuncOf(jsSuggestFromStock),
		"parseStockCSV":           js.FuncOf(jsParseStockCSV),
		"stockCSVTemplate":        js.FuncOf(jsStockCSVTemplate),
		"stockToCSV":              js.FuncOf(jsStockToCSV),
		"compareToAnchor":         js.FuncOf(jsCompareToAnchor),
		"bestBrandsFor":           js.FuncOf(jsBestBrandsFor),
	}))
	// Avisa o app que o motor está pronto (engine.ts registra este callback
	// antes de injetar o script).
	if ready := js.Global().Get("__mesclaOnReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}
	select {} // mantém o runtime Go vivo — as funções são chamadas via js.FuncOf
}

func errJSON(format string, args ...any) string {
	msg, _ := json.Marshal(fmt.Sprintf(format, args...))
	return `{"error":` + string(msg) + `}`
}

func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return errJSON("serializando resposta: %v", err)
	}
	return string(data)
}

func clamp8(f float64) uint8 {
	if f < 0 {
		return 0
	}
	if f > 255 {
		return 255
	}
	return uint8(f)
}

// jsInit carrega o catálogo (formato do cmd/export) e precomputa Lab.
func jsInit(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return errJSON("init espera 1 argumento (catalogJSON)")
	}
	var catalog struct {
		Manufacturers []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			PaintCount int    `json:"paintCount"`
		} `json:"manufacturers"`
		Paints [][]any `json:"paints"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &catalog); err != nil {
		return errJSON("catálogo inválido: %v", err)
	}

	mfrName = make(map[int64]string, len(catalog.Manufacturers))
	for _, m := range catalog.Manufacturers {
		mfrName[m.ID] = m.Name
	}

	paints = make([]paintRec, 0, len(catalog.Paints))
	byID = make(map[int64]int, len(catalog.Paints))
	for _, row := range catalog.Paints {
		// [id, mfrId, name, code, line, r, g, b] — números viram float64 no JSON
		if len(row) != 8 {
			return errJSON("linha de tinta com %d campos (esperava 8)", len(row))
		}
		rec := paintRec{
			ID:    int64(row[0].(float64)),
			MfrID: int64(row[1].(float64)),
			Name:  row[2].(string),
			Code:  row[3].(string),
			Line:  row[4].(string),
			R:     clamp8(row[5].(float64)),
			G:     clamp8(row[6].(float64)),
			B:     clamp8(row[7].(float64)),
		}
		l, a, b := color.RGBToLab(rec.R, rec.G, rec.B)
		rec.Lab = [3]float64{l, a, b}
		byID[rec.ID] = len(paints)
		paints = append(paints, rec)
	}

	return toJSON(map[string]any{"ok": true, "paints": len(paints), "manufacturers": len(mfrName)})
}

// similarResult espelha o SearchResultDTO do desktop (mesmas chaves JSON),
// acrescido de "code" — no celular o código do pote é essencial.
type similarResult struct {
	PaintID      int64   `json:"paintId"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Manufacturer string  `json:"manufacturer"`
	R            uint8   `json:"r"`
	G            uint8   `json:"g"`
	B            uint8   `json:"b"`
	DeltaE       float64 `json:"deltaE"`
	Similarity   float64 `json:"similarity"`
}

// jsFindSimilar(r, g, b, maxDeltaE, maxResults) — tintas mais próximas da cor.
func jsFindSimilar(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 5 {
		return errJSON("findSimilar espera (r, g, b, maxDeltaE, maxResults)")
	}
	r, g, b := clamp8(args[0].Float()), clamp8(args[1].Float()), clamp8(args[2].Float())
	maxDeltaE, maxResults := args[3].Float(), args[4].Int()
	if maxResults <= 0 {
		maxResults = 10
	}

	l, a, bb := color.RGBToLab(r, g, b)
	target := [3]float64{l, a, bb}

	results := make([]similarResult, 0, maxResults)
	for _, p := range paints {
		delta := color.DeltaE2000(target, p.Lab)
		if delta > maxDeltaE {
			continue
		}
		results = append(results, similarResult{
			PaintID: p.ID, Name: p.Name, Code: p.Code, Manufacturer: mfrName[p.MfrID],
			R: p.R, G: p.G, B: p.B, DeltaE: delta, Similarity: 100.0 - delta,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DeltaE < results[j].DeltaE })
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return toJSON(results)
}

// jsSuggestEquivalentRecipe(paintId, targetMfrId) — mesma forma JSON do
// EquivalentRecipeDTO do desktop; a regra vive em pkg/equivalence.
func jsSuggestEquivalentRecipe(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 2 {
		return errJSON("suggestEquivalentRecipe espera (paintId, targetManufacturerId)")
	}
	sourceID, targetMfrID := int64(args[0].Float()), int64(args[1].Float())

	idx, ok := byID[sourceID]
	if !ok {
		return errJSON("tinta de origem não encontrada: %d", sourceID)
	}
	source := paints[idx]
	targetName, ok := mfrName[targetMfrID]
	if !ok {
		return errJSON("fabricante de destino não encontrado: %d", targetMfrID)
	}

	candidates := make([]mix.PaintInput, 0, 512)
	for _, p := range paints {
		if p.MfrID == targetMfrID {
			candidates = append(candidates, mix.PaintInput{ID: p.ID, Name: p.Name, Code: p.Code, R: p.R, G: p.G, B: p.B})
		}
	}
	if len(candidates) == 0 {
		return errJSON("fabricante de destino não possui tintas cadastradas com cor")
	}

	sourceInput := mix.PaintInput{ID: source.ID, Name: source.Name, Code: source.Code, R: source.R, G: source.G, B: source.B}
	res, err := equivalence.Suggest(sourceInput, mfrName[source.MfrID], targetName, candidates)
	if err != nil {
		return errJSON("%v", err)
	}
	recipe := res.Recipe

	type ingredientJSON struct {
		PaintID    int64   `json:"paintId"`
		Name       string  `json:"name"`
		Code       string  `json:"code"`
		Percentage float64 `json:"percentage"`
		R          uint8   `json:"r"`
		G          uint8   `json:"g"`
		B          uint8   `json:"b"`
	}
	ingredients := make([]ingredientJSON, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, ingredientJSON{
			PaintID: ing.Paint.ID, Name: ing.Paint.Name, Code: ing.Paint.Code,
			Percentage: ing.Percentage, R: ing.Paint.R, G: ing.Paint.G, B: ing.Paint.B,
		})
	}
	tips := res.Tips
	if tips == nil {
		tips = []string{}
	}

	return toJSON(map[string]any{
		"sourcePaintId":      source.ID,
		"sourceName":         source.Name,
		"sourceManufacturer": mfrName[source.MfrID],
		"sourceR":            source.R,
		"sourceG":            source.G,
		"sourceB":            source.B,
		"targetManufacturer": targetName,
		"ingredients":        ingredients,
		"resultR":            recipe.ResultR,
		"resultG":            recipe.ResultG,
		"resultB":            recipe.ResultB,
		"deltaE":             recipe.DeltaE,
		"method":             recipe.Method,
		"reproducible":       res.Reproducible,
		"tips":               tips,
	})
}

// jsSuggestRecipeForColor(r, g, b, targetMfrId) — receita equivalente para uma
// COR ARBITRÁRIA (passo da rampa da Roda) dentro de uma marca. Origem sintética
// (sem marca) → nada é excluído do pool. Mesma forma JSON do EquivalentRecipeDTO.
func jsSuggestRecipeForColor(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 4 {
		return errJSON("suggestRecipeForColor espera (r, g, b, targetManufacturerId)")
	}
	r, g, b := clamp8(args[0].Float()), clamp8(args[1].Float()), clamp8(args[2].Float())
	targetMfrID := int64(args[3].Float())

	targetName, ok := mfrName[targetMfrID]
	if !ok {
		return errJSON("fabricante de destino não encontrado: %d", targetMfrID)
	}

	candidates := make([]mix.PaintInput, 0, 512)
	for _, p := range paints {
		if p.MfrID == targetMfrID {
			candidates = append(candidates, mix.PaintInput{ID: p.ID, Name: p.Name, Code: p.Code, R: p.R, G: p.G, B: p.B})
		}
	}
	if len(candidates) == 0 {
		return errJSON("fabricante de destino não possui tintas cadastradas com cor")
	}

	sourceInput := mix.PaintInput{ID: 0, Name: "cor alvo", Code: "", R: r, G: g, B: b}
	res, err := equivalence.Suggest(sourceInput, "", targetName, candidates)
	if err != nil {
		return errJSON("%v", err)
	}
	recipe := res.Recipe

	type ingredientJSON struct {
		PaintID    int64   `json:"paintId"`
		Name       string  `json:"name"`
		Code       string  `json:"code"`
		Percentage float64 `json:"percentage"`
		R          uint8   `json:"r"`
		G          uint8   `json:"g"`
		B          uint8   `json:"b"`
	}
	ingredients := make([]ingredientJSON, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, ingredientJSON{
			PaintID: ing.Paint.ID, Name: ing.Paint.Name, Code: ing.Paint.Code,
			Percentage: ing.Percentage, R: ing.Paint.R, G: ing.Paint.G, B: ing.Paint.B,
		})
	}
	tips := res.Tips
	if tips == nil {
		tips = []string{}
	}

	return toJSON(map[string]any{
		"sourcePaintId":      int64(0),
		"sourceName":         "cor alvo",
		"sourceManufacturer": "",
		"sourceR":            r,
		"sourceG":            g,
		"sourceB":            b,
		"targetManufacturer": targetName,
		"ingredients":        ingredients,
		"resultR":            recipe.ResultR,
		"resultG":            recipe.ResultG,
		"resultB":            recipe.ResultB,
		"deltaE":             recipe.DeltaE,
		"method":             recipe.Method,
		"reproducible":       res.Reproducible,
		"tips":               tips,
	})
}

// jsSuggestFromStock(sourcePaintId, stockJSON) — receita da cor de origem
// usando SÓ o estoque do pintor (as tintas que ele cadastrou, de qualquer
// marca). É o "priorize o que eu tenho" do mobile; a regra vive em
// pkg/equivalence.SuggestFromStock, a mesma do desktop. O estoque chega como
// JSON porque vive no localStorage do app, não no catálogo em memória.
func jsSuggestFromStock(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 2 {
		return errJSON("suggestFromStock espera (sourcePaintId, stockJSON)")
	}
	sourceID := int64(args[0].Float())

	idx, ok := byID[sourceID]
	if !ok {
		return errJSON("tinta de origem não encontrada: %d", sourceID)
	}
	source := paints[idx]

	var stockPaints []stock.Paint
	if err := json.Unmarshal([]byte(args[1].String()), &stockPaints); err != nil {
		return errJSON("estoque inválido: %v", err)
	}
	pool := stock.ToMixInputs(stockPaints)
	if len(pool) == 0 {
		return errJSON("seu estoque está vazio — cadastre tintas primeiro")
	}

	sourceInput := mix.PaintInput{ID: source.ID, Name: source.Name, Code: source.Code, R: source.R, G: source.G, B: source.B}
	res, err := equivalence.SuggestFromStock(sourceInput, pool)
	if err != nil {
		return errJSON("%v", err)
	}
	recipe := res.Recipe

	type ingredientJSON struct {
		PaintID    int64   `json:"paintId"`
		Name       string  `json:"name"`
		Code       string  `json:"code"`
		Percentage float64 `json:"percentage"`
		R          uint8   `json:"r"`
		G          uint8   `json:"g"`
		B          uint8   `json:"b"`
	}
	ingredients := make([]ingredientJSON, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, ingredientJSON{
			PaintID: ing.Paint.ID, Name: ing.Paint.Name, Code: ing.Paint.Code,
			Percentage: ing.Percentage, R: ing.Paint.R, G: ing.Paint.G, B: ing.Paint.B,
		})
	}
	tips := res.Tips
	if tips == nil {
		tips = []string{}
	}

	return toJSON(map[string]any{
		"sourcePaintId":      source.ID,
		"sourceName":         source.Name,
		"sourceManufacturer": mfrName[source.MfrID],
		"sourceR":            source.R,
		"sourceG":            source.G,
		"sourceB":            source.B,
		"targetManufacturer": "Meu estoque",
		"ingredients":        ingredients,
		"resultR":            recipe.ResultR,
		"resultG":            recipe.ResultG,
		"resultB":            recipe.ResultB,
		"deltaE":             recipe.DeltaE,
		"method":             recipe.Method,
		"reproducible":       res.Reproducible,
		"tips":               tips,
	})
}

// jsParseStockCSV(csvText) — valida um CSV de importação contra os fabricantes
// do catálogo em memória (mesma crítica do desktop, via pkg/stock). Retorna
// {"paints":[...], "errors":[...]} — o TS insere as tintas boas no localStorage
// e mostra os erros por linha.
func jsParseStockCSV(_ js.Value, args []js.Value) any {
	if len(mfrName) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 1 {
		return errJSON("parseStockCSV espera (csvText)")
	}

	mfrs := make([]stock.Manufacturer, 0, len(mfrName))
	for id, name := range mfrName {
		mfrs = append(mfrs, stock.Manufacturer{ID: id, Name: name})
	}

	paintsOut, rowErrs := stock.ParseCSV(args[0].String(), mfrs)
	if paintsOut == nil {
		paintsOut = []stock.Paint{}
	}
	if rowErrs == nil {
		rowErrs = []stock.RowError{}
	}
	return toJSON(map[string]any{"paints": paintsOut, "errors": rowErrs})
}

// jsStockCSVTemplate() — conteúdo do CSV-modelo, para o app oferecer o download.
func jsStockCSVTemplate(_ js.Value, _ []js.Value) any {
	return toJSON(map[string]any{"csv": stock.CSVTemplate()})
}

// jsStockToCSV(stockJSON) — serializa o estoque no formato de importação
// (backup/exportação). Mesma função do desktop (stock.ToCSV).
func jsStockToCSV(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return errJSON("stockToCSV espera (stockJSON)")
	}
	var stockPaints []stock.Paint
	if err := json.Unmarshal([]byte(args[0].String()), &stockPaints); err != nil {
		return errJSON("estoque inválido: %v", err)
	}
	return toJSON(map[string]any{"csv": stock.ToCSV(stockPaints)})
}

// jsCompareToAnchor(anchorId, ids[]) — ΔE de cada tinta em relação à âncora,
// ordenado da mais próxima à mais distante (lista-âncora do Comparar mobile).
func jsCompareToAnchor(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 2 {
		return errJSON("compareToAnchor espera (anchorId, ids)")
	}
	anchorIdx, ok := byID[int64(args[0].Float())]
	if !ok {
		return errJSON("tinta âncora não encontrada")
	}
	anchor := paints[anchorIdx]

	idsVal := args[1]
	results := make([]similarResult, 0, idsVal.Length())
	for i := 0; i < idsVal.Length(); i++ {
		id := int64(idsVal.Index(i).Float())
		if id == anchor.ID {
			continue
		}
		idx, ok := byID[id]
		if !ok {
			continue
		}
		p := paints[idx]
		delta := color.DeltaE2000(anchor.Lab, p.Lab)
		results = append(results, similarResult{
			PaintID: p.ID, Name: p.Name, Code: p.Code, Manufacturer: mfrName[p.MfrID],
			R: p.R, G: p.G, B: p.B, DeltaE: delta, Similarity: 100.0 - delta,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DeltaE < results[j].DeltaE })
	return toJSON(results)
}

// jsBestBrandsFor(paintId) — para cada OUTRA marca, a tinta única mais próxima
// da cor-alvo (1 passada no catálogo). Alimenta o "não sai nesta marca → veja
// nestas" do fluxo Mesclar; a receita completa da marca escolhida é calculada
// depois, sob demanda.
func jsBestBrandsFor(_ js.Value, args []js.Value) any {
	if len(paints) == 0 {
		return errJSON("catálogo não inicializado — chame init primeiro")
	}
	if len(args) != 1 {
		return errJSON("bestBrandsFor espera (paintId)")
	}
	idx, ok := byID[int64(args[0].Float())]
	if !ok {
		return errJSON("tinta não encontrada")
	}
	source := paints[idx]

	type brandBest struct {
		ManufacturerID int64   `json:"manufacturerId"`
		Manufacturer   string  `json:"manufacturer"`
		PaintID        int64   `json:"paintId"`
		Name           string  `json:"name"`
		Code           string  `json:"code"`
		R              uint8   `json:"r"`
		G              uint8   `json:"g"`
		B              uint8   `json:"b"`
		DeltaE         float64 `json:"deltaE"`
	}
	best := make(map[int64]brandBest)
	for _, p := range paints {
		if p.ID == source.ID {
			continue
		}
		delta := color.DeltaE2000(source.Lab, p.Lab)
		cur, seen := best[p.MfrID]
		if !seen || delta < cur.DeltaE {
			best[p.MfrID] = brandBest{
				ManufacturerID: p.MfrID, Manufacturer: mfrName[p.MfrID],
				PaintID: p.ID, Name: p.Name, Code: p.Code,
				R: p.R, G: p.G, B: p.B, DeltaE: delta,
			}
		}
	}
	results := make([]brandBest, 0, len(best))
	for _, b := range best {
		results = append(results, b)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DeltaE < results[j].DeltaE })
	return toJSON(results)
}
