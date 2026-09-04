package main

import (
	"database/sql"
	"errors"
	"fmt"

	"paint-match-ai/internal/equivalence"
	"paint-match-ai/internal/mix"
	"paint-match-ai/internal/stock"
)

// ensureUserSchema cria a tabela do estoque do usuário se ela ainda não existe.
// É idempotente (roda a cada boot) e usa FK real para manufacturers: uma tinta
// de estoque sempre aponta para um fabricante do catálogo.
func ensureUserSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_paints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			manufacturer_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			code TEXT,
			rgb_r INTEGER NOT NULL,
			rgb_g INTEGER NOT NULL,
			rgb_b INTEGER NOT NULL,
			volume TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id)
		);
		CREATE INDEX IF NOT EXISTS idx_user_paints_manufacturer ON user_paints(manufacturer_id);
	`)
	return err
}

// UserPaintDTO é uma tinta do estoque do usuário, no formato trocado com o
// frontend. A cor viaja como R/G/B (o color-picker do form converte de/para hex).
type UserPaintDTO struct {
	ID             int64  `json:"id"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	R              uint8  `json:"r"`
	G              uint8  `json:"g"`
	B              uint8  `json:"b"`
	Volume         string `json:"volume"`
	Notes          string `json:"notes"`
}

// CSVImportResultDTO resume uma importação: quantas tintas entraram e os erros
// por linha (linhas ruins não bloqueiam as boas).
type CSVImportResultDTO struct {
	Imported int              `json:"imported"`
	Errors   []stock.RowError `json:"errors"`
}

// GetUserPaints lista o estoque do usuário, com o nome do fabricante resolvido.
func (s *PaintService) GetUserPaints() ([]UserPaintDTO, error) {
	rows, err := s.db.Query(`
		SELECT up.id, up.manufacturer_id, m.name,
		       up.name, COALESCE(up.code, ''),
		       up.rgb_r, up.rgb_g, up.rgb_b,
		       COALESCE(up.volume, ''), COALESCE(up.notes, '')
		FROM user_paints up
		JOIN manufacturers m ON m.id = up.manufacturer_id
		ORDER BY m.name, up.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserPaintDTO, 0)
	for rows.Next() {
		var p UserPaintDTO
		if err := rows.Scan(
			&p.ID, &p.ManufacturerID, &p.Manufacturer,
			&p.Name, &p.Code, &p.R, &p.G, &p.B, &p.Volume, &p.Notes,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// validateManufacturer garante que o fabricante existe e devolve seu nome
// canônico — a mesma regra que o CSV aplica, agora no cadastro manual.
func (s *PaintService) validateManufacturer(manufacturerID int64) (string, error) {
	var name string
	err := s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", manufacturerID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("fabricante informado não existe no catálogo")
	}
	if err != nil {
		return "", err
	}
	return name, nil
}

func validateUserPaintFields(p UserPaintDTO) error {
	if p.Name == "" {
		return fmt.Errorf("o nome da tinta é obrigatório")
	}
	return nil
}

// AddUserPaint cadastra uma tinta no estoque. Valida fabricante e nome.
func (s *PaintService) AddUserPaint(p UserPaintDTO) (UserPaintDTO, error) {
	if err := validateUserPaintFields(p); err != nil {
		return UserPaintDTO{}, err
	}
	mfrName, err := s.validateManufacturer(p.ManufacturerID)
	if err != nil {
		return UserPaintDTO{}, err
	}

	res, err := s.db.Exec(`
		INSERT INTO user_paints (manufacturer_id, name, code, rgb_r, rgb_g, rgb_b, volume, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ManufacturerID, p.Name, p.Code, p.R, p.G, p.B, p.Volume, p.Notes)
	if err != nil {
		return UserPaintDTO{}, err
	}
	id, _ := res.LastInsertId()
	p.ID = id
	p.Manufacturer = mfrName
	return p, nil
}

// UpdateUserPaint edita uma tinta do estoque.
func (s *PaintService) UpdateUserPaint(p UserPaintDTO) (UserPaintDTO, error) {
	if err := validateUserPaintFields(p); err != nil {
		return UserPaintDTO{}, err
	}
	mfrName, err := s.validateManufacturer(p.ManufacturerID)
	if err != nil {
		return UserPaintDTO{}, err
	}

	res, err := s.db.Exec(`
		UPDATE user_paints
		SET manufacturer_id = ?, name = ?, code = ?, rgb_r = ?, rgb_g = ?, rgb_b = ?,
		    volume = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.ManufacturerID, p.Name, p.Code, p.R, p.G, p.B, p.Volume, p.Notes, p.ID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return UserPaintDTO{}, fmt.Errorf("tinta de estoque não encontrada (id %d)", p.ID)
	}
	p.Manufacturer = mfrName
	return p, nil
}

// DeleteUserPaint remove uma tinta do estoque.
func (s *PaintService) DeleteUserPaint(id int64) error {
	res, err := s.db.Exec("DELETE FROM user_paints WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("tinta de estoque não encontrada (id %d)", id)
	}
	return nil
}

// UserPaintCSVTemplate devolve o conteúdo do arquivo-modelo para download.
func (s *PaintService) UserPaintCSVTemplate() (string, error) {
	return stock.CSVTemplate(), nil
}

// ExportUserPaintsCSV serializa o estoque atual no mesmo formato do modelo —
// backup ou base pra reimportar em outra máquina. Usa pkg/stock.ToCSV (o inverso
// exato de ParseCSV), então o arquivo reentra sem erros.
func (s *PaintService) ExportUserPaintsCSV() (string, error) {
	rows, err := s.db.Query(`
		SELECT m.name, up.name, COALESCE(up.code, ''), up.rgb_r, up.rgb_g, up.rgb_b,
		       COALESCE(up.volume, ''), COALESCE(up.notes, '')
		FROM user_paints up
		JOIN manufacturers m ON m.id = up.manufacturer_id
		ORDER BY m.name, up.name
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var paints []stock.Paint
	for rows.Next() {
		var p stock.Paint
		if err := rows.Scan(&p.Manufacturer, &p.Name, &p.Code, &p.R, &p.G, &p.B, &p.Volume, &p.Notes); err != nil {
			return "", err
		}
		paints = append(paints, p)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return stock.ToCSV(paints), nil
}

// ImportUserPaintsCSV importa um lote de tintas de um CSV. A crítica (fabricante
// precisa existir, hex válido, nome obrigatório) vive em pkg/stock — a mesma que
// o app mobile roda via WASM. Linhas válidas são inseridas mesmo se outras
// falharem; os erros voltam para o usuário corrigir.
func (s *PaintService) ImportUserPaintsCSV(csvText string) (CSVImportResultDTO, error) {
	mfrs, err := s.loadStockManufacturers()
	if err != nil {
		return CSVImportResultDTO{}, err
	}

	paints, rowErrs := stock.ParseCSV(csvText, mfrs)

	tx, err := s.db.Begin()
	if err != nil {
		return CSVImportResultDTO{}, err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO user_paints (manufacturer_id, name, code, rgb_r, rgb_g, rgb_b, volume, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return CSVImportResultDTO{}, err
	}
	defer stmt.Close()

	imported := 0
	for _, p := range paints {
		if _, err := stmt.Exec(p.ManufacturerID, p.Name, p.Code, p.R, p.G, p.B, p.Volume, p.Notes); err != nil {
			_ = tx.Rollback()
			return CSVImportResultDTO{}, err
		}
		imported++
	}
	if err := tx.Commit(); err != nil {
		return CSVImportResultDTO{}, err
	}

	return CSVImportResultDTO{Imported: imported, Errors: rowErrs}, nil
}

// loadStockManufacturers carrega os fabricantes no formato mínimo que a crítica
// de CSV precisa.
func (s *PaintService) loadStockManufacturers() ([]stock.Manufacturer, error) {
	rows, err := s.db.Query("SELECT id, name FROM manufacturers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mfrs []stock.Manufacturer
	for rows.Next() {
		var m stock.Manufacturer
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}
		mfrs = append(mfrs, m)
	}
	return mfrs, rows.Err()
}

// loadStockAsMixInputs carrega o estoque do usuário como pool de mistura.
func (s *PaintService) loadStockAsMixInputs() ([]mix.PaintInput, error) {
	rows, err := s.db.Query("SELECT id, name, COALESCE(code, ''), rgb_r, rgb_g, rgb_b FROM user_paints")
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

// SuggestEquivalentFromStock monta a receita da cor de origem usando SÓ o
// estoque do usuário — o "priorize o que eu tenho". Quando não é reproduzível,
// a UI cai no fluxo por fabricante (SuggestEquivalentRecipe). O ingrediente
// carrega o id da tinta de estoque (mix.PaintInput.ID), então o frontend
// reencontra marca/volume pelo GetUserPaints.
func (s *PaintService) SuggestEquivalentFromStock(sourcePaintID int64) (EquivalentRecipeDTO, error) {
	source, err := s.GetPaintByID(sourcePaintID)
	if err != nil {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não encontrada: %w", err)
	}
	if source.R == 0 && source.G == 0 && source.B == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não possui dados de cor cadastrados")
	}

	pool, err := s.loadStockAsMixInputs()
	if err != nil {
		return EquivalentRecipeDTO{}, err
	}
	if len(pool) == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("seu estoque está vazio — cadastre tintas primeiro")
	}

	sourceInput := mix.PaintInput{ID: source.ID, Name: source.Name, Code: source.Code, R: source.R, G: source.G, B: source.B}
	res, err := equivalence.SuggestFromStock(sourceInput, pool)
	if err != nil {
		if errors.Is(err, equivalence.ErrNoCandidates) {
			return EquivalentRecipeDTO{}, fmt.Errorf("seu estoque está vazio — cadastre tintas primeiro")
		}
		return EquivalentRecipeDTO{}, err
	}
	recipe := res.Recipe

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

	return EquivalentRecipeDTO{
		SourcePaintID:      source.ID,
		SourceName:         source.Name,
		SourceManufacturer: source.Manufacturer,
		SourceR:            source.R,
		SourceG:            source.G,
		SourceB:            source.B,
		TargetManufacturer: "Meu estoque",
		Ingredients:        ingredients,
		ResultR:            recipe.ResultR,
		ResultG:            recipe.ResultG,
		ResultB:            recipe.ResultB,
		DeltaE:             recipe.DeltaE,
		Method:             recipe.Method,
		Reproducible:       res.Reproducible,
		Tips:               res.Tips,
	}, nil
}
