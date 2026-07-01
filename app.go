package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"paint-match-ai/pkg/ai"
	"paint-match-ai/pkg/color"
	"paint-match-ai/pkg/mix"
	"paint-match-ai/pkg/similarity"

	_ "modernc.org/sqlite"
)

type PaintService struct {
	db  *sql.DB
	ret *ai.Retrieval
	sim *similarity.Engine
}

func NewPaintService() (*PaintService, error) {
	dbPath := "paint_knowledge.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".paint-match-ai", "paint_knowledge.db")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("pragma: %w", err)
		}
	}

	return &PaintService{
		db:  db,
		ret: ai.NewRetrieval(db),
		sim: similarity.NewEngine(db),
	}, nil
}

func (s *PaintService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

type PaintDTO struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Manufacturer string  `json:"manufacturer"`
	ProductLine  string  `json:"productLine"`
	R            uint8   `json:"r"`
	G            uint8   `json:"g"`
	B            uint8   `json:"b"`
	SwatchPath   string  `json:"swatchPath"`
	Thumbnail    string  `json:"thumbnail"`
	ImageURL     string  `json:"imageUrl"`
	FinishType   string  `json:"finishType"`
	PaintType    string  `json:"paintType"`
	Coverage     string  `json:"coverage"`
	Opacity      string  `json:"opacity"`
	Volume       string  `json:"volume"`
}

type SearchResultDTO struct {
	PaintID      int64   `json:"paintId"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	R            uint8   `json:"r"`
	G            uint8   `json:"g"`
	B            uint8   `json:"b"`
	DeltaE       float64 `json:"deltaE"`
	SwatchPath   string  `json:"swatchPath"`
	Similarity   float64 `json:"similarity"`
}

type RecipeDTO struct {
	Ingredients []RecipeIngredientDTO `json:"ingredients"`
	ResultR     uint8                 `json:"resultR"`
	ResultG     uint8                 `json:"resultG"`
	ResultB     uint8                 `json:"resultB"`
	DeltaE      float64               `json:"deltaE"`
	Method      string                `json:"method"`
}

type RecipeIngredientDTO struct {
	PaintID    int64   `json:"paintId"`
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	R          uint8   `json:"r"`
	G          uint8   `json:"g"`
	B          uint8   `json:"b"`
}

type ManufacturerDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type StatsDTO struct {
	Manufacturers int `json:"manufacturers"`
	ProductLines  int `json:"productLines"`
	Paints        int `json:"paints"`
	Equivalences  int `json:"equivalences"`
	Recipes       int `json:"recipes"`
}

func (s *PaintService) GetStats() (StatsDTO, error) {
	var stats StatsDTO
	s.db.QueryRow("SELECT COUNT(*) FROM manufacturers").Scan(&stats.Manufacturers)
	s.db.QueryRow("SELECT COUNT(*) FROM product_lines").Scan(&stats.ProductLines)
	s.db.QueryRow("SELECT COUNT(*) FROM paints").Scan(&stats.Paints)
	s.db.QueryRow("SELECT COUNT(*) FROM equivalences").Scan(&stats.Equivalences)
	s.db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&stats.Recipes)
	return stats, nil
}

func (s *PaintService) GetManufacturers() ([]ManufacturerDTO, error) {
	rows, err := s.db.Query("SELECT id, name FROM manufacturers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ManufacturerDTO
	for rows.Next() {
		var m ManufacturerDTO
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *PaintService) GetAllPaints() ([]PaintDTO, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		ORDER BY m.name, p.name
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PaintDTO
	for rows.Next() {
		var p PaintDTO
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.Manufacturer,
			&p.ProductLine,
			&p.R, &p.G, &p.B,
			&p.SwatchPath,
			&p.Thumbnail, &p.ImageURL,
			&p.FinishType, &p.PaintType,
			&p.Coverage, &p.Opacity,
			&p.Volume,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *PaintService) SearchPaints(query string) ([]PaintDTO, error) {
	sqlQuery := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE p.name LIKE ? OR p.code LIKE ? OR m.name LIKE ?
		ORDER BY m.name, p.name
		LIMIT 50
	`
	like := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PaintDTO
	for rows.Next() {
		var p PaintDTO
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.Manufacturer,
			&p.ProductLine,
			&p.R, &p.G, &p.B,
			&p.SwatchPath,
			&p.Thumbnail, &p.ImageURL,
			&p.FinishType, &p.PaintType,
			&p.Coverage, &p.Opacity,
			&p.Volume,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *PaintService) GetPaintByID(id int64) (PaintDTO, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE p.id = ?
	`
	var p PaintDTO
	err := s.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.Code, &p.Manufacturer,
		&p.ProductLine,
		&p.R, &p.G, &p.B,
		&p.SwatchPath,
		&p.Thumbnail, &p.ImageURL,
		&p.FinishType, &p.PaintType,
		&p.Coverage, &p.Opacity,
		&p.Volume,
	)
	return p, err
}

func (s *PaintService) FindSimilar(r, g, b uint8, maxDeltaE float64, maxResults int) ([]SearchResultDTO, error) {
	opts := similarity.SearchOptions{
		MaxResults: maxResults,
		MaxDeltaE:  maxDeltaE,
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}
	if opts.MaxDeltaE <= 0 {
		opts.MaxDeltaE = 10.0
	}

	results, err := s.sim.FindSimilar(r, g, b, opts)
	if err != nil {
		return nil, err
	}

	var dto []SearchResultDTO
	for i, res := range results {
		dto = append(dto, SearchResultDTO{
			PaintID:      res.PaintID,
			Name:         res.Name,
			Manufacturer: res.Manufacturer,
			R:            res.R,
			G:            res.G,
			B:            res.B,
			DeltaE:       res.DeltaE,
			SwatchPath:   res.SwatchPath,
			Similarity:   100.0 - res.DeltaE,
		})
		_ = i
	}
	return dto, nil
}

func (s *PaintService) FindEquivalences(paintID int64) ([]SearchResultDTO, error) {
	results, err := s.sim.FindEquivalent(paintID)
	if err != nil {
		return nil, err
	}

	var dto []SearchResultDTO
	for _, res := range results {
		dto = append(dto, SearchResultDTO{
			PaintID:      res.PaintID,
			Name:         res.Name,
			Manufacturer: res.Manufacturer,
			R:            res.R,
			G:            res.G,
			B:            res.B,
			DeltaE:       res.DeltaE,
			SwatchPath:   res.SwatchPath,
			Similarity:   100.0 - res.DeltaE,
		})
	}
	return dto, nil
}

func (s *PaintService) SuggestRecipe(r, g, b uint8, maxPaints int) (RecipeDTO, error) {
	if maxPaints <= 0 {
		maxPaints = 20
	}

	available, err := s.loadAvailablePaints(maxPaints)
	if err != nil {
		return RecipeDTO{}, err
	}

	engine := mix.NewEngine()
	target := [3]float64{float64(r), float64(g), float64(b)}
	recipe := engine.SuggestRecipe(target, available)

	var ingredients []RecipeIngredientDTO
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, RecipeIngredientDTO{
			PaintID:    ing.Paint.ID,
			Name:       ing.Paint.Name,
			Percentage: ing.Percentage,
			R:          ing.Paint.R,
			G:          ing.Paint.G,
			B:          ing.Paint.B,
		})
	}

	return RecipeDTO{
		Ingredients: ingredients,
		ResultR:     recipe.ResultR,
		ResultG:     recipe.ResultG,
		ResultB:     recipe.ResultB,
		DeltaE:      recipe.DeltaE,
		Method:      recipe.Method,
	}, nil
}

func (s *PaintService) loadAvailablePaints(limit int) ([]mix.PaintInput, error) {
	query := `
		SELECT p.id, p.name, COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0)
		FROM paints p
		JOIN paint_colors pc ON pc.paint_id = p.id
		WHERE pc.rgb_r IS NOT NULL
		ORDER BY RANDOM()
		LIMIT ?
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.R, &p.G, &p.B); err != nil {
			return nil, err
		}
		paints = append(paints, p)
	}
	return paints, rows.Err()
}

func (s *PaintService) ProcessQuery(text string) (ai.Response, error) {
	query := ai.Query{
		Text:   text,
		Intent: ai.IntentGeneral,
	}

	paints, _ := s.SearchPaints(text)
	if len(paints) > 0 {
		query.Intent = ai.IntentPaintInfo
	}

	return s.ret.Process(query)
}

func (s *PaintService) CompareColors(paintIDs []int64) ([]SearchResultDTO, error) {
	if len(paintIDs) < 2 {
		return nil, fmt.Errorf("precisa de pelo menos 2 tintas para comparar")
	}

	var paints []PaintDTO
	for _, id := range paintIDs {
		p, err := s.GetPaintByID(id)
		if err != nil {
			continue
		}
		paints = append(paints, p)
	}

	if len(paints) < 2 {
		return nil, fmt.Errorf("poucas tintas encontradas para comparar")
	}

	var results []SearchResultDTO
	for i := 0; i < len(paints); i++ {
		for j := i + 1; j < len(paints); j++ {
			l1, a1, b1 := color.RGBToLab(paints[i].R, paints[i].G, paints[i].B)
			l2, a2, b2 := color.RGBToLab(paints[j].R, paints[j].G, paints[j].B)
			deltaE := color.DeltaE2000(
				[3]float64{l1, a1, b1},
				[3]float64{l2, a2, b2},
			)
			results = append(results, SearchResultDTO{
				PaintID:      paints[j].ID,
				Name:         fmt.Sprintf("%s vs %s", paints[i].Name, paints[j].Name),
				Manufacturer: fmt.Sprintf("%s / %s", paints[i].Manufacturer, paints[j].Manufacturer),
				R:            paints[j].R,
				G:            paints[j].G,
				B:            paints[j].B,
				DeltaE:       deltaE,
				Similarity:   100.0 - deltaE,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DeltaE < results[j].DeltaE
	})

	return results, nil
}
