package ai

import (
	"database/sql"
	"fmt"
	"strings"

	"paint-match-ai/internal/color"
	"paint-match-ai/internal/mix"
	"paint-match-ai/internal/similarity"
)

// Retrieval engine principal de recuperação de conhecimento
type Retrieval struct {
	db       *sql.DB
	sim      *similarity.Engine
	mix      *mix.Engine
}

// NewRetrieval cria nova instância do engine de recuperação
func NewRetrieval(db *sql.DB) *Retrieval {
	return &Retrieval{
		db:  db,
		sim: similarity.NewEngine(db),
		mix: mix.NewEngine(),
	}
}

// Process executa a pipeline completa de recuperação
func (r *Retrieval) Process(query Query) (Response, error) {
	resp := Response{
		Intent:   query.Intent,
		Sources:  []string{"paint_knowledge.db"},
	}

	switch query.Intent {
	case IntentPaintInfo:
		paints, err := r.FindPaintInfo(query.Text)
		if err != nil {
			return resp, err
		}
		resp.Paints = paints
		if len(paints) == 0 {
			resp.IsEstimate = true
			resp.Message = fmt.Sprintf("Nenhuma tinta encontrada para '%s' no banco de dados.", query.Text)
		}

	case IntentFindEquivalent:
		paints, err := r.FindPaintInfo(query.Text)
		if err != nil {
			return resp, err
		}
		if len(paints) == 0 {
			resp.IsEstimate = true
			resp.Message = fmt.Sprintf("Tinta '%s' não encontrada no banco de dados.", query.Text)
			return resp, nil
		}
		resp.Paints = paints
		equivs, err := r.FindEquivalencesFor(paints[0].ID)
		if err != nil {
			return resp, err
		}
		resp.Equivalences = equivs

	case IntentFindSimilar:
		if query.TargetColor == nil {
			resp.IsEstimate = true
			resp.Message = "Cor alvo não especificada."
			return resp, nil
		}
		opts := r.buildSimilarOptions(query.Filters)
		simResults, err := r.FindSimilarColors(*query.TargetColor, opts)
		if err != nil {
			return resp, err
		}
		resp.Similar = simResults

	case IntentMixRecipe:
		if query.TargetColor == nil {
			resp.IsEstimate = true
			resp.Message = "Cor alvo não especificada."
			return resp, nil
		}
		maxIngredients := 3
		if query.Filters.MaxResults != nil && *query.Filters.MaxResults > 0 {
			maxIngredients = *query.Filters.MaxResults
			if maxIngredients > 4 {
				maxIngredients = 4
			}
		}
		recipes, err := r.SuggestMixes(*query.TargetColor, maxIngredients)
		if err != nil {
			return resp, err
		}
		resp.Recipes = recipes

	case IntentCompare:
		// Parse paint IDs from text (comma-separated)
		ids := parsePaintIDs(query.Text)
		if len(ids) < 2 {
			resp.IsEstimate = true
			resp.Message = "Forneça pelo menos 2 IDs de tintas para comparar."
			return resp, nil
		}
		return r.ComparePaints(ids)

	case IntentManufacturerInfo:
		paints, err := r.loadPaintsByManufacturer(query.Text)
		if err != nil {
			return resp, err
		}
		resp.Paints = paints
		if len(paints) == 0 {
			resp.IsEstimate = true
			resp.Message = fmt.Sprintf("Nenhum fabricante '%s' encontrado no banco.", query.Text)
		}

	default:
		// General: tenta encontrar tinta por nome
		paints, err := r.FindPaintInfo(query.Text)
		if err != nil {
			return resp, err
		}
		resp.Paints = paints
		if len(paints) == 0 {
			resp.IsEstimate = true
			resp.Message = fmt.Sprintf("Nenhuma informação encontrada para '%s' no banco de dados.", query.Text)
		}
	}

	return resp, nil
}

// FindPaintInfo busca informações detalhadas de uma tinta por ID, código ou nome
func (r *Retrieval) FindPaintInfo(identifier string) ([]PaintInfo, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name, COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''), COALESCE(pc.thumbnail_path, ''), COALESCE(pc.image_url, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''), COALESCE(ct.name, ''),
			   COALESCE(ot.name, ''), COALESCE(p.volume, '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE p.id = ? OR p.code = ? OR p.name LIKE ?
		LIMIT 10
	`

	rows, err := r.db.Query(query, identifier, identifier, "%"+identifier+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPaints(rows)
}

// FindEquivalencesFor busca equivalências para uma tinta
func (r *Retrieval) FindEquivalencesFor(paintID int64) ([]Equivalence, error) {
	// Busca equivalências diretas da tabela
	query := `
		SELECT p2.id, p2.name, COALESCE(p2.code, ''), m2.name, COALESCE(pl2.name, ''),
			   COALESCE(pc2.rgb_r, 0), COALESCE(pc2.rgb_g, 0), COALESCE(pc2.rgb_b, 0),
			   COALESCE(pc2.swatch_path, ''),
			   COALESCE(eq.similarity, 0), COALESCE(eq.delta_e, 0), COALESCE(eq.notes, '')
		FROM equivalences eq
		JOIN paints p2 ON p2.id = eq.paint_id_2
		JOIN manufacturers m2 ON m2.id = p2.manufacturer_id
		LEFT JOIN product_lines pl2 ON pl2.id = p2.product_line_id
		LEFT JOIN paint_colors pc2 ON pc2.paint_id = p2.id
		WHERE eq.paint_id_1 = ?
		UNION
		SELECT p1.id, p1.name, COALESCE(p1.code, ''), m1.name, COALESCE(pl1.name, ''),
			   COALESCE(pc1.rgb_r, 0), COALESCE(pc1.rgb_g, 0), COALESCE(pc1.rgb_b, 0),
			   COALESCE(pc1.swatch_path, ''),
			   COALESCE(eq.similarity, 0), COALESCE(eq.delta_e, 0), COALESCE(eq.notes, '')
		FROM equivalences eq
		JOIN paints p1 ON p1.id = eq.paint_id_1
		JOIN manufacturers m1 ON m1.id = p1.manufacturer_id
		LEFT JOIN product_lines pl1 ON pl1.id = p1.product_line_id
		LEFT JOIN paint_colors pc1 ON pc1.paint_id = p1.id
		WHERE eq.paint_id_2 = ?
	`

	rows, err := r.db.Query(query, paintID, paintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var equivs []Equivalence
	for rows.Next() {
		var target PaintInfo
		var similarityVal, deltaE float64
		var notes string

		if err := rows.Scan(
			&target.ID, &target.Name, &target.Code, &target.Manufacturer, &target.ProductLine,
			&target.RGB.R, &target.RGB.G, &target.RGB.B,
			&target.SwatchPath,
			&similarityVal, &deltaE, &notes,
		); err != nil {
			return nil, err
		}

		target.Lab = color.Lab{
			L: func() float64 { l, _, _ := color.RGBToLab(target.RGB.R, target.RGB.G, target.RGB.B); return l }(),
			A: func() float64 { _, a, _ := color.RGBToLab(target.RGB.R, target.RGB.G, target.RGB.B); return a }(),
			B: func() float64 { _, _, b := color.RGBToLab(target.RGB.R, target.RGB.G, target.RGB.B); return b }(),
		}

		equivs = append(equivs, Equivalence{
			TargetPaint: target,
			DeltaE:      deltaE,
			Similarity:  similarityVal,
			Notes:       notes,
		})
	}

	return equivs, rows.Err()
}

// SuggestMixes sugere misturas para uma cor alvo
func (r *Retrieval) SuggestMixes(target color.RGB, maxIngredients int) ([]Recipe, error) {
	// Carrega tintas disponíveis
	paints, err := r.loadAllPaints()
	if err != nil {
		return nil, err
	}

	if len(paints) == 0 {
		return nil, nil
	}

	// Converte para PaintInput do mix engine
	inputs := make([]mix.PaintInput, len(paints))
	for i, p := range paints {
		inputs[i] = mix.PaintInput{
			ID: p.ID,
			Name: p.Name,
			R: p.RGB.R,
			G: p.RGB.G,
			B: p.RGB.B,
		}
	}

	targetL, targetA, targetB := color.RGBToLab(target.R, target.G, target.B)
	targetArr := [3]float64{targetL, targetA, targetB}

	// Usa subset de tintas para performance
	subset := inputs
	if len(inputs) > 20 {
		// Seleciona as mais diversas (espaçamento no espaço RGB)
		subset = selectDiversePaints(inputs, 20)
	}

	recipe := r.mix.SuggestRecipe(targetArr, subset)

	// Converte resultado para Recipe do AI
	ingredients := make([]RecipeIngredient, len(recipe.Ingredients))
	for i, ing := range recipe.Ingredients {
		pi := PaintInfo{
			ID:   ing.Paint.ID,
			Name: ing.Paint.Name,
			RGB:  color.RGB{R: ing.Paint.R, G: ing.Paint.G, B: ing.Paint.B},
		}
		l, a, b := color.RGBToLab(ing.Paint.R, ing.Paint.G, ing.Paint.B)
		pi.Lab = color.Lab{L: l, A: a, B: b}
		ingredients[i] = RecipeIngredient{
			Paint:      pi,
			Percentage: ing.Percentage,
		}
	}

	return []Recipe{{
		Ingredients: ingredients,
		Method:      recipe.Method,
		DeltaE:      recipe.DeltaE,
	}}, nil
}

// FindSimilarColors busca cores similares com contexto completo
func (r *Retrieval) FindSimilarColors(target color.RGB, opts similarity.SearchOptions) ([]SimilarResult, error) {
	results, err := r.sim.FindSimilar(target.R, target.G, target.B, opts)
	if err != nil {
		return nil, err
	}

	similar := make([]SimilarResult, len(results))
	for i, res := range results {
		pi := PaintInfo{
			ID:           res.PaintID,
			Name:         res.Name,
			Manufacturer: res.Manufacturer,
			RGB:          color.RGB{R: res.R, G: res.G, B: res.B},
			SwatchPath:   res.SwatchPath,
		}
		l, a, b := color.RGBToLab(res.R, res.G, res.B)
		pi.Lab = color.Lab{L: l, A: a, B: b}
		similar[i] = SimilarResult{
			Paint:  pi,
			DeltaE: res.DeltaE,
			Rank:   i + 1,
		}
	}

	return similar, nil
}

// ComparePaints compara duas ou mais tintas
func (r *Retrieval) ComparePaints(paintIDs []int64) (Response, error) {
	resp := Response{
		Intent:  IntentCompare,
		Sources: []string{"paint_knowledge.db"},
	}

	var paints []PaintInfo
	for _, id := range paintIDs {
		p, err := r.FindPaintInfo(fmt.Sprintf("%d", id))
		if err != nil {
			return resp, err
		}
		if len(p) == 0 {
			resp.IsEstimate = true
			resp.Message = fmt.Sprintf("Tinta ID %d não encontrada.", id)
			return resp, nil
		}
		paints = append(paints, p[0])
	}

	resp.Paints = paints

	// Calcula similaridades par a par
	for i := 0; i < len(paints); i++ {
		for j := i + 1; j < len(paints); j++ {
			deltaE := color.DeltaE2000(
				[3]float64{paints[i].Lab.L, paints[i].Lab.A, paints[i].Lab.B},
				[3]float64{paints[j].Lab.L, paints[j].Lab.A, paints[j].Lab.B},
			)
			similarity := 100.0 - deltaE*10
			if similarity < 0 {
				similarity = 0
			}
			resp.Equivalences = append(resp.Equivalences, Equivalence{
				SourcePaint: paints[i],
				TargetPaint: paints[j],
				DeltaE:      deltaE,
				Similarity:  similarity,
			})
		}
	}

	return resp, nil
}

// loadPaintsByManufacturer carrega tintas de um fabricante pelo nome
func (r *Retrieval) loadPaintsByManufacturer(manufacturerName string) ([]PaintInfo, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name, COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''), COALESCE(pc.thumbnail_path, ''), COALESCE(pc.image_url, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''), COALESCE(ct.name, ''),
			   COALESCE(ot.name, ''), COALESCE(p.volume, '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE m.name LIKE ?
	`

	rows, err := r.db.Query(query, "%"+manufacturerName+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPaints(rows)
}

// loadAllPaints carrega todas as pintas com cor definida
func (r *Retrieval) loadAllPaints() ([]PaintInfo, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name, COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''), COALESCE(pc.thumbnail_path, ''), COALESCE(pc.image_url, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''), COALESCE(ct.name, ''),
			   COALESCE(ot.name, ''), COALESCE(p.volume, '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE pc.rgb_r IS NOT NULL
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPaints(rows)
}

// scanPaints escaneia rows de pintas para []PaintInfo
func (r *Retrieval) scanPaints(rows *sql.Rows) ([]PaintInfo, error) {
	var paints []PaintInfo
	for rows.Next() {
		var p PaintInfo
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.Manufacturer, &p.ProductLine,
			&p.RGB.R, &p.RGB.G, &p.RGB.B,
			&p.SwatchPath, &p.Thumbnail, &p.ImageURL,
			&p.FinishType, &p.PaintType, &p.Coverage,
			&p.Opacity, &p.Volume,
		); err != nil {
			return nil, err
		}
		l, a, b := color.RGBToLab(p.RGB.R, p.RGB.G, p.RGB.B)
		p.Lab = color.Lab{L: l, A: a, B: b}
		paints = append(paints, p)
	}
	return paints, rows.Err()
}

// buildSimilarOptions converte QueryFilters para similarity.SearchOptions
func (r *Retrieval) buildSimilarOptions(filters QueryFilters) similarity.SearchOptions {
	opts := similarity.DefaultOptions()
	if filters.ManufacturerID != nil {
		opts.ManufacturerID = filters.ManufacturerID
	}
	if filters.PaintTypeID != nil {
		opts.PaintTypeID = filters.PaintTypeID
	}
	if filters.MaxDeltaE != nil {
		opts.MaxDeltaE = *filters.MaxDeltaE
	}
	if filters.MaxResults != nil {
		opts.MaxResults = *filters.MaxResults
	}
	return opts
}

// parsePaintIDs parse IDs de tintas de uma string (separada por vírgula)
func parsePaintIDs(text string) []int64 {
	parts := strings.Split(text, ",")
	var ids []int64
	for _, part := range parts {
		var id int64
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// selectDiversePaints seleciona subset diverso de tintas
func selectDiversePaints(paints []mix.PaintInput, n int) []mix.PaintInput {
	if len(paints) <= n {
		return paints
	}

	selected := make([]mix.PaintInput, 0, n)
	selected = append(selected, paints[0])

	for len(selected) < n {
		bestIdx := -1
		bestMinDist := -1.0

		for i, p := range paints {
			alreadySelected := false
			for _, s := range selected {
				if p.ID == s.ID {
					alreadySelected = true
					break
				}
			}
			if alreadySelected {
				continue
			}

			// Distância mínima para qualquer já selecionado
			minDist := 1e9
			for _, s := range selected {
				dr := float64(int(p.R) - int(s.R))
				dg := float64(int(p.G) - int(s.G))
				db := float64(int(p.B) - int(s.B))
				dist := dr*dr + dg*dg + db*db
				if dist < minDist {
					minDist = dist
				}
			}

			if minDist > bestMinDist {
				bestMinDist = minDist
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			selected = append(selected, paints[bestIdx])
		} else {
			break
		}
	}

	return selected
}
