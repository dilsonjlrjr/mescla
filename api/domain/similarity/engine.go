package similarity

import (
	"database/sql"
	"sort"

	"paint-match-ai/api/domain/color"
	"paint-match-ai/api/domain/mix"
)

type SearchResult struct {
	PaintID      int64
	Name         string
	Manufacturer string
	R, G, B      uint8
	DeltaE       float64
	SwatchPath   string
}

type SearchOptions struct {
	MaxResults     int
	MaxDeltaE      float64
	ManufacturerID *int64
	PaintTypeID    *int64
}

func DefaultOptions() SearchOptions {
	return SearchOptions{
		MaxResults: 10,
		MaxDeltaE:  10.0,
	}
}

type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

type paintRow struct {
	ID           int64
	Name         string
	Manufacturer string
	R, G, B      int64
	SwatchPath   string
}

func (e *Engine) FindSimilar(targetR, targetG, targetB uint8, opts SearchOptions) ([]SearchResult, error) {
	paints, err := e.loadPaints(opts)
	if err != nil {
		return nil, err
	}

	targetL, targetA, targetBVal := color.RGBToLab(targetR, targetG, targetB)
	targetLab := [3]float64{targetL, targetA, targetBVal}

	var results []SearchResult
	for _, p := range paints {
		l, a, b := color.RGBToLab(uint8(p.R), uint8(p.G), uint8(p.B))
		paintLab := [3]float64{l, a, b}
		deltaE := color.DeltaE2000(targetLab, paintLab)

		if deltaE <= opts.MaxDeltaE {
			results = append(results, SearchResult{
				PaintID:      p.ID,
				Name:         p.Name,
				Manufacturer: p.Manufacturer,
				R:            uint8(p.R),
				G:            uint8(p.G),
				B:            uint8(p.B),
				DeltaE:       deltaE,
				SwatchPath:   p.SwatchPath,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DeltaE < results[j].DeltaE
	})

	if opts.MaxResults > 0 && len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results, nil
}

func (e *Engine) FindSimilarToMixed(ingredients []mix.Ingredient, opts SearchOptions) ([]SearchResult, error) {
	engine := mix.NewEngine()

	colors := make([]mix.PaintInput, len(ingredients))
	weights := make([]float64, len(ingredients))
	for i, ing := range ingredients {
		colors[i] = ing.Paint
		weights[i] = ing.Percentage
	}

	r, g, b := engine.Mix(colors, weights)
	return e.FindSimilar(r, g, b, opts)
}

func (e *Engine) FindEquivalent(paintID int64) ([]SearchResult, error) {
	query := `
		SELECT p.id, p.name, m.name,
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, '')
		FROM equivalences eq
		JOIN paints p ON (p.id = eq.paint_id_1 OR p.id = eq.paint_id_2) AND p.id != ?
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		WHERE eq.paint_id_1 = ? OR eq.paint_id_2 = ?
	`

	rows, err := e.db.Query(query, paintID, paintID, paintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r paintRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Manufacturer, &r.R, &r.G, &r.B, &r.SwatchPath); err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			PaintID:      r.ID,
			Name:         r.Name,
			Manufacturer: r.Manufacturer,
			R:            uint8(r.R),
			G:            uint8(r.G),
			B:            uint8(r.B),
			SwatchPath:   r.SwatchPath,
		})
	}

	return results, rows.Err()
}

func (e *Engine) loadPaints(opts SearchOptions) ([]paintRow, error) {
	query := `
		SELECT p.id, p.name, m.name,
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		WHERE pc.rgb_r IS NOT NULL
	`

	args := []interface{}{}

	if opts.ManufacturerID != nil {
		query += " AND p.manufacturer_id = ?"
		args = append(args, *opts.ManufacturerID)
	}

	if opts.PaintTypeID != nil {
		query += " AND p.paint_type_id = ?"
		args = append(args, *opts.PaintTypeID)
	}

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []paintRow
	for rows.Next() {
		var p paintRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Manufacturer, &p.R, &p.G, &p.B, &p.SwatchPath); err != nil {
			return nil, err
		}
		paints = append(paints, p)
	}

	return paints, rows.Err()
}
