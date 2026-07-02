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

// NewPaintService abre o banco de catálogo. Ordem de resolução:
//  1. paint_knowledge.db no diretório atual (fluxo de desenvolvimento);
//  2. banco já instalado no diretório de dados do usuário;
//  3. primeiro boot: extrai o banco embutido no binário (embeddedSeed)
//     para o diretório de dados — o app é auto-suficiente, sem instalador.
func NewPaintService(embeddedSeed []byte) (*PaintService, error) {
	dbPath := "paint_knowledge.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			cfgDir = home
		}
		dataDir := filepath.Join(cfgDir, "Mescla")
		dbPath = filepath.Join(dataDir, "paint_knowledge.db")

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			if len(embeddedSeed) == 0 {
				return nil, fmt.Errorf("banco de catálogo não encontrado (nem no diretório atual, nem em %s, nem embutido no binário)", dataDir)
			}
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return nil, fmt.Errorf("criando diretório de dados: %w", err)
			}
			if err := os.WriteFile(dbPath, embeddedSeed, 0o644); err != nil {
				return nil, fmt.Errorf("instalando banco de catálogo: %w", err)
			}
		}
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

type RecipeIngredientDTO struct {
	PaintID    int64   `json:"paintId"`
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Percentage float64 `json:"percentage"`
	R          uint8   `json:"r"`
	G          uint8   `json:"g"`
	B          uint8   `json:"b"`
}

type EquivalentRecipeDTO struct {
	SourcePaintID      int64                 `json:"sourcePaintId"`
	SourceName         string                `json:"sourceName"`
	SourceManufacturer string                `json:"sourceManufacturer"`
	SourceR            uint8                 `json:"sourceR"`
	SourceG            uint8                 `json:"sourceG"`
	SourceB            uint8                 `json:"sourceB"`
	TargetManufacturer string                `json:"targetManufacturer"`
	Ingredients        []RecipeIngredientDTO `json:"ingredients"`
	ResultR            uint8                 `json:"resultR"`
	ResultG            uint8                 `json:"resultG"`
	ResultB            uint8                 `json:"resultB"`
	DeltaE             float64               `json:"deltaE"`
	Method             string                `json:"method"`
	// Reproducible indica se a mistura chega perto o bastante da cor de origem
	// pra ser considerada uma equivalência de verdade. Quando falso, o fabricante
	// de destino não tem os pigmentos necessários e os "ingredientes" são só a
	// melhor aproximação possível — não uma receita utilizável.
	Reproducible bool     `json:"reproducible"`
	Tips         []string `json:"tips"`
}

// maxViableDeltaE é o limite de ΔE2000 acima do qual uma cor é considerada
// irreproduzível com o catálogo de destino. ΔE ~10 já é uma diferença de cor
// óbvia a olho nu; acima disso a "receita" não replica a cor, só a aproxima.
const maxViableDeltaE = 10.0

type ManufacturerDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Country    string `json:"country"`
	Website    string `json:"website"`
	LogoPath   string `json:"logoPath"`
	PaintCount int    `json:"paintCount"`
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
	rows, err := s.db.Query(`
		SELECT m.id, m.name, COALESCE(m.country, ''), COALESCE(m.website, ''), COALESCE(m.logo_path, ''),
		       (SELECT COUNT(*) FROM paints p WHERE p.manufacturer_id = m.id)
		FROM manufacturers m
		ORDER BY m.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ManufacturerDTO
	for rows.Next() {
		var m ManufacturerDTO
		if err := rows.Scan(&m.ID, &m.Name, &m.Country, &m.Website, &m.LogoPath, &m.PaintCount); err != nil {
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

// SuggestEquivalentRecipe busca a tinta de origem (de qualquer fabricante) e
// monta uma receita de mistura usando somente tintas do fabricante de destino
// que aproxima a cor da origem, junto com dicas de ajuste em texto.
func (s *PaintService) SuggestEquivalentRecipe(sourcePaintID int64, targetManufacturerID int64) (EquivalentRecipeDTO, error) {
	source, err := s.GetPaintByID(sourcePaintID)
	if err != nil {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não encontrada: %w", err)
	}
	if source.R == 0 && source.G == 0 && source.B == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não possui dados de cor cadastrados")
	}

	candidates, err := s.loadPaintsByManufacturerID(targetManufacturerID)
	if err != nil {
		return EquivalentRecipeDTO{}, err
	}
	if len(candidates) == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("fabricante de destino não possui tintas cadastradas com cor")
	}

	// Nunca usar a própria tinta-alvo como ingrediente. Quando a marca de destino
	// é a mesma da origem (ex.: montar um azul-marinho que a marca não tem, a
	// partir de preto + azul dela), a tinta-alvo estaria no pool e a "receita"
	// viraria 100% dela mesma (ΔE 0), inútil. Em marcas diferentes os ids nunca
	// coincidem, então isto não tem efeito lá.
	kept := candidates[:0]
	for _, c := range candidates {
		if c.ID != source.ID {
			kept = append(kept, c)
		}
	}
	candidates = kept
	if len(candidates) == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("essa marca não tem outras tintas com cor pra montar a mistura")
	}

	var targetMfrName string
	s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", targetManufacturerID).Scan(&targetMfrName)

	// Mesma marca: obrigar mistura de 2+ tintas. Devolver "quase 100% de uma
	// tinta só" da própria marca não ajuda — o usuário já sabe que aquela tinta
	// existe; ele quer o tom que NÃO tem, feito com as que tem.
	minIngredients := 1
	if targetMfrName != "" && targetMfrName == source.Manufacturer {
		minIngredients = 2
	}

	targetL, targetA, targetB := color.RGBToLab(source.R, source.G, source.B)
	recipe := mix.SuggestBestSubset([3]float64{targetL, targetA, targetB}, candidates, minIngredients, 3)

	ingredients := make([]RecipeIngredientDTO, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, RecipeIngredientDTO{
			PaintID:    ing.Paint.ID,
			Name:       ing.Paint.Name,
			Code:       ing.Paint.Code,
			Percentage: ing.Percentage,
			R:          ing.Paint.R,
			G:          ing.Paint.G,
			B:          ing.Paint.B,
		})
	}

	tips := mix.GenerateTips(source.R, source.G, source.B, recipe)

	return EquivalentRecipeDTO{
		SourcePaintID:      source.ID,
		SourceName:         source.Name,
		SourceManufacturer: source.Manufacturer,
		SourceR:            source.R,
		SourceG:            source.G,
		SourceB:            source.B,
		TargetManufacturer: targetMfrName,
		Ingredients:        ingredients,
		ResultR:            recipe.ResultR,
		ResultG:            recipe.ResultG,
		ResultB:            recipe.ResultB,
		DeltaE:             recipe.DeltaE,
		Method:             recipe.Method,
		Reproducible:       recipe.DeltaE <= maxViableDeltaE,
		Tips:               tips,
	}, nil
}

// loadPaintsByManufacturerID carrega as tintas com cor cadastrada de um único
// fabricante — pool de candidatos para SuggestBestSubset. Usa JOIN (não
// LEFT JOIN + COALESCE como o resto do arquivo) de propósito: tinta sem RGB
// não deve virar candidato "preto" silencioso numa receita de mistura.
func (s *PaintService) loadPaintsByManufacturerID(manufacturerID int64) ([]mix.PaintInput, error) {
	query := `
		SELECT p.id, p.name, p.code, pc.rgb_r, pc.rgb_g, pc.rgb_b
		FROM paints p
		JOIN paint_colors pc ON pc.paint_id = p.id
		WHERE p.manufacturer_id = ?
	`
	rows, err := s.db.Query(query, manufacturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.R, &p.G, &p.B); err != nil {
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
