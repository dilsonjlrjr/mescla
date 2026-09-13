package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"paint-match-ai/api/domain/equivalence"
	"paint-match-ai/api/domain/mix"
	"paint-match-ai/api/domain/stock"
)

// ensureUserSchema cria a tabela do estoque do usuário se ela ainda não existe.
// É idempotente (roda a cada boot) e usa FK real para manufacturers: uma tinta
// de estoque sempre aponta para um fabricante do catálogo.
//
// rf-17: quantity/paint_type_id/catalog_id são colunas aditivas — a tabela já
// existe em produção, então usa addColumnIfMissing (não CREATE TABLE) e a
// ordem importa: user_paints precisa existir antes do ALTER. user_paint_origins
// é a lápide da migração: sobrevive à exclusão da tinta (user_paint_id vira
// NULL) para reenviar a mesma linha local responder ja-migrada, nunca
// ressuscitar o que o dono apagou.
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
	if err != nil {
		return err
	}

	if err := addColumnIfMissing(db, "user_paints", "quantity", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "user_paints", "paint_type_id", "INTEGER"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "user_paints", "catalog_id", "INTEGER"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_paint_origins (
			origin_ref TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			user_paint_id INTEGER REFERENCES user_paints(id) ON DELETE SET NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_paint_origins_paint ON user_paint_origins(user_paint_id);
		CREATE INDEX IF NOT EXISTS idx_user_paint_origins_device ON user_paint_origins(device_id);
	`); err != nil {
		return err
	}
	// Impressão do conteúdo da linha local: a mesma lista vinda por outro
	// caminho (arquivo importado, outro navegador) reconhece a origem já
	// migrada — sem ela, um deviceId novo duplicaria tinta sem código e
	// ressuscitaria tinta apagada (achado do guardrail rf-17).
	if err := addColumnIfMissing(db, "user_paint_origins", "fingerprint", "TEXT"); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_paint_origins_fingerprint ON user_paint_origins(fingerprint)`); err != nil {
		return err
	}
	return nil
}

// UserPaintDTO é uma tinta do estoque do usuário, no formato trocado com o
// frontend. A cor viaja como R/G/B (o color-picker do form converte de/para
// hex). Quantity/PaintTypeID/CatalogID são ponteiros (rf-17, RN17): no PUT,
// ausente (nil) mantém o valor gravado; no POST, ausente vira o padrão
// (1 / NULL). Na resposta os três sempre vêm preenchidos — quantity ≥ 1,
// paintTypeId/catalogId como número ou null.
type UserPaintDTO struct {
	ID             int64  `json:"id"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	R              int    `json:"r"`
	G              int    `json:"g"`
	B              int    `json:"b"`
	Volume         string `json:"volume"`
	Notes          string `json:"notes"`
	Quantity       *int   `json:"quantity"`
	PaintTypeID    *int64 `json:"paintTypeId"`
	CatalogID      *int64 `json:"catalogId"`
}

// CSVImportResultDTO resume uma importação: quantas tintas entraram e os erros
// por linha (linhas ruins não bloqueiam as boas).
type CSVImportResultDTO struct {
	Imported int              `json:"imported"`
	Errors   []stock.RowError `json:"errors"`
}

// ErrUserPaintNotFound é a sentinela de "tinta de estoque não encontrada" —
// o handler HTTP faz errors.Is nela para responder 404, nunca casando por texto.
var ErrUserPaintNotFound = errors.New("tinta de estoque não encontrada")

// UserPaintInputError é a sentinela de entrada inválida (400) do CRUD de
// estoque e da migração — Msg é a mensagem fixa devolvida ao cliente, nunca o
// texto do SQLite.
type UserPaintInputError struct{ Msg string }

func (e *UserPaintInputError) Error() string { return e.Msg }

const (
	maxUserPaintNameRunes   = 200
	maxUserPaintCodeRunes   = 60
	maxUserPaintVolumeRunes = 40
	maxUserPaintNotesRunes  = 2000
)

// userPaintSelect resolve manufacturer/paint_type/catalog por LEFT JOIN: um
// paint_type_id ou catalog_id que aponte para um registro apagado sai como
// null na resposta, em vez de quebrar a leitura (Decisões de E1).
const userPaintSelect = `
	SELECT up.id, up.manufacturer_id, m.name,
	       up.name, COALESCE(up.code, ''),
	       up.rgb_r, up.rgb_g, up.rgb_b,
	       COALESCE(up.volume, ''), COALESCE(up.notes, ''),
	       up.quantity, pt.id, ca.id
	FROM user_paints up
	JOIN manufacturers m ON m.id = up.manufacturer_id
	LEFT JOIN paint_types pt ON pt.id = up.paint_type_id
	LEFT JOIN paints ca ON ca.id = up.catalog_id`

type rowScanner interface{ Scan(dest ...any) error }

func scanUserPaint(sc rowScanner) (UserPaintDTO, error) {
	var p UserPaintDTO
	var quantity int
	var paintTypeID, catalogID sql.NullInt64
	if err := sc.Scan(
		&p.ID, &p.ManufacturerID, &p.Manufacturer,
		&p.Name, &p.Code, &p.R, &p.G, &p.B, &p.Volume, &p.Notes,
		&quantity, &paintTypeID, &catalogID,
	); err != nil {
		return UserPaintDTO{}, err
	}
	p.Quantity = &quantity
	if paintTypeID.Valid {
		p.PaintTypeID = &paintTypeID.Int64
	}
	if catalogID.Valid {
		p.CatalogID = &catalogID.Int64
	}
	return p, nil
}

// GetUserPaints lista o estoque do usuário, com o nome do fabricante resolvido.
func (s *PaintService) GetUserPaints() ([]UserPaintDTO, error) {
	rows, err := s.db.Query(userPaintSelect + ` ORDER BY m.name, up.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserPaintDTO, 0)
	for rows.Next() {
		p, err := scanUserPaint(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// userPaintByID relê uma tinta pelo mesmo SELECT do GET — usado depois de
// INSERT/UPDATE para devolver o DTO já resolvido, sem casar campo a campo.
func (s *PaintService) userPaintByID(id int64) (UserPaintDTO, error) {
	p, err := scanUserPaint(s.db.QueryRow(userPaintSelect+` WHERE up.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return UserPaintDTO{}, ErrUserPaintNotFound
	}
	return p, err
}

// validateManufacturer garante que o fabricante existe e devolve seu nome
// canônico — a mesma regra que o CSV aplica, agora no cadastro manual.
func (s *PaintService) validateManufacturer(manufacturerID int64) (string, error) {
	var name string
	err := s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", manufacturerID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", &UserPaintInputError{Msg: "fabricante informado não existe no catálogo"}
	}
	if err != nil {
		return "", err
	}
	return name, nil
}

func validRGBComponent(v int) bool { return v >= 0 && v <= 255 }

// normalizeUserPaintFields valida nome/código/volume/notas/cor (contrato do
// POST/PUT /user-paints, mensagens fixas, sem eco) e grava o nome limpo em p.
func normalizeUserPaintFields(p *UserPaintDTO) error {
	name := cleanName(p.Name)
	if name == "" {
		return &UserPaintInputError{Msg: "o nome da tinta é obrigatório"}
	}
	if utf8.RuneCountInString(name) > maxUserPaintNameRunes {
		return &UserPaintInputError{Msg: "nome da tinta com mais de 200 caracteres"}
	}
	if utf8.RuneCountInString(p.Code) > maxUserPaintCodeRunes {
		return &UserPaintInputError{Msg: "código com mais de 60 caracteres"}
	}
	if utf8.RuneCountInString(p.Volume) > maxUserPaintVolumeRunes {
		return &UserPaintInputError{Msg: "volume com mais de 40 caracteres"}
	}
	if utf8.RuneCountInString(p.Notes) > maxUserPaintNotesRunes {
		return &UserPaintInputError{Msg: "notas com mais de 2000 caracteres"}
	}
	if !validRGBComponent(p.R) || !validRGBComponent(p.G) || !validRGBComponent(p.B) {
		return &UserPaintInputError{Msg: "cor inválida"}
	}
	p.Name = name
	return nil
}

// resolveQuantity aplica RN17: campo ausente (nil) mantém existing — que já
// vem como 1 no caminho do POST, já que não há linha gravada ainda.
func resolveQuantity(existing int, in *int) (int, error) {
	if in == nil {
		return existing, nil
	}
	if *in < 1 || *in > 9999 {
		return 0, &UserPaintInputError{Msg: "quantidade inválida"}
	}
	return *in, nil
}

// resolvePaintTypeID aplica RN17: ausente mantém existing (NULL no POST); 0
// limpa para NULL; qualquer outro valor precisa existir em paint_types.
func (s *PaintService) resolvePaintTypeID(existing sql.NullInt64, in *int64) (sql.NullInt64, error) {
	if in == nil {
		return existing, nil
	}
	if *in == 0 {
		return sql.NullInt64{}, nil
	}
	if _, err := s.paintTypeByID(*in); err != nil {
		if errors.Is(err, ErrPaintTypeNotFound) {
			return sql.NullInt64{}, &UserPaintInputError{Msg: "tipo de tinta não encontrado"}
		}
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: *in, Valid: true}, nil
}

func (s *PaintService) catalogPaintExists(id int64) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM paints WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

// resolveCatalogID é o mesmo padrão de resolvePaintTypeID, contra o catálogo.
func (s *PaintService) resolveCatalogID(existing sql.NullInt64, in *int64) (sql.NullInt64, error) {
	if in == nil {
		return existing, nil
	}
	if *in == 0 {
		return sql.NullInt64{}, nil
	}
	ok, err := s.catalogPaintExists(*in)
	if err != nil {
		return sql.NullInt64{}, err
	}
	if !ok {
		return sql.NullInt64{}, &UserPaintInputError{Msg: "tinta do catálogo não encontrada"}
	}
	return sql.NullInt64{Int64: *in, Valid: true}, nil
}

// AddUserPaint cadastra uma tinta no estoque. Ignora p.ID (o handler já zera).
// Ordem de validação segue o contrato: campos de texto/cor, quantidade, tipo,
// catálogo, fabricante por último.
func (s *PaintService) AddUserPaint(p UserPaintDTO) (UserPaintDTO, error) {
	if err := normalizeUserPaintFields(&p); err != nil {
		return UserPaintDTO{}, err
	}
	quantity, err := resolveQuantity(1, p.Quantity)
	if err != nil {
		return UserPaintDTO{}, err
	}
	paintTypeID, err := s.resolvePaintTypeID(sql.NullInt64{}, p.PaintTypeID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	catalogID, err := s.resolveCatalogID(sql.NullInt64{}, p.CatalogID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	if _, err := s.validateManufacturer(p.ManufacturerID); err != nil {
		return UserPaintDTO{}, err
	}

	res, err := s.db.Exec(`
		INSERT INTO user_paints (manufacturer_id, name, code, rgb_r, rgb_g, rgb_b, volume, notes, quantity, paint_type_id, catalog_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ManufacturerID, p.Name, p.Code, p.R, p.G, p.B, p.Volume, p.Notes, quantity, paintTypeID, catalogID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return UserPaintDTO{}, err
	}
	return s.userPaintByID(id)
}

// userPaintEditableFields lê os 3 campos aditivos da linha atual — base da
// resolução em memória do PUT parcial (Decisões de E1).
func (s *PaintService) userPaintEditableFields(id int64) (quantity int, paintTypeID, catalogID sql.NullInt64, err error) {
	err = s.db.QueryRow(`SELECT quantity, paint_type_id, catalog_id FROM user_paints WHERE id = ?`, id).
		Scan(&quantity, &paintTypeID, &catalogID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.NullInt64{}, sql.NullInt64{}, ErrUserPaintNotFound
	}
	return quantity, paintTypeID, catalogID, err
}

// UpdateUserPaint edita uma tinta do estoque. Lê a linha, resolve os campos
// aditivos em memória (RN17) e faz um único UPDATE.
func (s *PaintService) UpdateUserPaint(p UserPaintDTO) (UserPaintDTO, error) {
	if err := normalizeUserPaintFields(&p); err != nil {
		return UserPaintDTO{}, err
	}
	existingQty, existingType, existingCatalog, err := s.userPaintEditableFields(p.ID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	quantity, err := resolveQuantity(existingQty, p.Quantity)
	if err != nil {
		return UserPaintDTO{}, err
	}
	paintTypeID, err := s.resolvePaintTypeID(existingType, p.PaintTypeID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	catalogID, err := s.resolveCatalogID(existingCatalog, p.CatalogID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	if _, err := s.validateManufacturer(p.ManufacturerID); err != nil {
		return UserPaintDTO{}, err
	}

	res, err := s.db.Exec(`
		UPDATE user_paints
		SET manufacturer_id = ?, name = ?, code = ?, rgb_r = ?, rgb_g = ?, rgb_b = ?,
		    volume = ?, notes = ?, quantity = ?, paint_type_id = ?, catalog_id = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.ManufacturerID, p.Name, p.Code, p.R, p.G, p.B, p.Volume, p.Notes,
		quantity, paintTypeID, catalogID, p.ID)
	if err != nil {
		return UserPaintDTO{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return UserPaintDTO{}, ErrUserPaintNotFound
	}
	return s.userPaintByID(p.ID)
}

// DeleteUserPaint remove uma tinta do estoque. A lápide em user_paint_origins
// é desligada explicitamente ANTES do DELETE (não depende só da FK) — assim
// reenviar a mesma linha local sempre acha a origem e responde ja-migrada.
func (s *PaintService) DeleteUserPaint(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE user_paint_origins SET user_paint_id = NULL WHERE user_paint_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM user_paints WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserPaintNotFound
	}
	return tx.Commit()
}

// UserPaintCSVTemplate devolve o conteúdo do arquivo-modelo para download.
func (s *PaintService) UserPaintCSVTemplate() (string, error) {
	return stock.CSVTemplate(), nil
}

// ExportUserPaintsCSV serializa o estoque atual no mesmo formato do modelo —
// backup ou base pra reimportar em outra máquina. Usa api/domain/stock.ToCSV (o inverso
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
// precisa existir, hex válido, nome obrigatório) vive em api/domain/stock — a mesma que
// o app mobile roda via WASM. Linhas válidas são inseridas mesmo se outras
// falharem; os erros voltam para o usuário corrigir. quantity/paint_type_id/
// catalog_id ficam de fora do INSERT de propósito — os padrões da coluna
// (1 / NULL) cobrem.
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
	rows, err := s.db.Query(`
		SELECT up.id, up.name, COALESCE(up.code, ''), up.rgb_r, up.rgb_g, up.rgb_b, up.manufacturer_id, m.name
		FROM user_paints up
		JOIN manufacturers m ON m.id = up.manufacturer_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.R, &p.G, &p.B, &p.ManufacturerID, &p.Manufacturer); err != nil {
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

	ingredients := mapIngredients(recipe.Ingredients)
	crossBrand, manufacturers := crossBrandInfo(ingredients)

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
		CrossBrand:         crossBrand,
		Manufacturers:      manufacturers,
	}, nil
}

// --- rf-17: migração do estoque de um navegador (POST /user-paints/migrate) ---

var (
	migrateDeviceIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,80}$`)
	migrateLocalRefRe = regexp.MustCompile(`^(-?\d{1,15}(#\d{1,5})?|i\d{1,6})$`)
	migrateNullCodeRe = regexp.MustCompile(`(?i)^null-\d+$`)
	// Impressão gerada pelo cliente (cyrb53 ×2): fora do formato é ignorada.
	migrateFingerprintRe = regexp.MustCompile(`^[0-9a-f]{26}$`)
)

const maxMigrateBatchLines = 500

// MigrateLineDTO é uma linha do estoque local mandada por POST
// /user-paints/migrate — o mesmo formato de StockPaint do cliente, mais
// localRef (RN3).
type MigrateLineDTO struct {
	LocalRef       string `json:"localRef"`
	Fingerprint    string `json:"fingerprint"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	R              int    `json:"r"`
	G              int    `json:"g"`
	B              int    `json:"b"`
	Volume         string `json:"volume"`
	Notes          string `json:"notes"`
	Quantity       int    `json:"quantity"`
	PaintTypeID    *int64 `json:"paintTypeId"`
	CatalogID      *int64 `json:"catalogId"`
}

// MigrateLineResultDTO é o resultado de uma linha (RN6): motivo é um código
// estável, nunca eco do que foi mandado.
type MigrateLineResultDTO struct {
	LocalRef string `json:"localRef"`
	Status   string `json:"status"`
	Motivo   string `json:"motivo"`
}

// MigrateResultDTO é a resposta de POST /user-paints/migrate (RN6).
type MigrateResultDTO struct {
	Results    []MigrateLineResultDTO `json:"results"`
	Criadas    int                    `json:"criadas"`
	Mescladas  int                    `json:"mescladas"`
	JaMigradas int                    `json:"jaMigradas"`
	Recusadas  int                    `json:"recusadas"`
}

// MigrateUserPaints grava o estoque de um navegador (RN5), uma linha por vez,
// numa única transação BEGIN IMMEDIATE por lote (padrão de DeleteManufacturer
// em manufacturers_crud.go), em conexão dedicada. Erro de banco desfaz o lote
// inteiro — o chamador HTTP responde 500 com mensagem fixa.
func (s *PaintService) MigrateUserPaints(deviceID string, rawLines []json.RawMessage) (MigrateResultDTO, error) {
	if !migrateDeviceIDRe.MatchString(deviceID) {
		return MigrateResultDTO{}, &UserPaintInputError{Msg: "deviceId inválido"}
	}
	if len(rawLines) > maxMigrateBatchLines {
		return MigrateResultDTO{}, &UserPaintInputError{Msg: "lote grande demais"}
	}

	result := MigrateResultDTO{Results: make([]MigrateLineResultDTO, 0, len(rawLines))}
	if len(rawLines) == 0 {
		return result, nil
	}

	mfrs, err := s.loadStockManufacturers()
	if err != nil {
		return MigrateResultDTO{}, err
	}

	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return MigrateResultDTO{}, err
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return MigrateResultDTO{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
		}
	}()

	seen := make(map[string]bool, len(rawLines))
	for _, raw := range rawLines {
		var line MigrateLineDTO
		if err := json.Unmarshal(raw, &line); err != nil {
			result.Results = append(result.Results, MigrateLineResultDTO{Status: "recusada", Motivo: "linha-invalida"})
			result.Recusadas++
			continue
		}
		lineResult, err := s.migrateOneLine(ctx, conn, deviceID, line, mfrs, seen)
		if err != nil {
			return MigrateResultDTO{}, err
		}
		result.Results = append(result.Results, lineResult)
		switch lineResult.Status {
		case "criada":
			result.Criadas++
		case "mesclada":
			result.Mescladas++
		case "ja-migrada":
			result.JaMigradas++
		case "recusada":
			result.Recusadas++
		}
	}

	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return MigrateResultDTO{}, err
	}
	committed = true
	return result, nil
}

// migrateOneLine aplica RN5, passo a passo, na ordem do contrato: localRef,
// origem já migrada, fabricante, validação, junção ou criação.
func (s *PaintService) migrateOneLine(
	ctx context.Context, conn *sql.Conn, deviceID string, line MigrateLineDTO,
	mfrs []stock.Manufacturer, seen map[string]bool,
) (MigrateLineResultDTO, error) {
	localRef := line.LocalRef
	if !migrateLocalRefRe.MatchString(localRef) {
		return MigrateLineResultDTO{LocalRef: localRef, Status: "recusada", Motivo: "linha-invalida"}, nil
	}
	if seen[localRef] {
		return MigrateLineResultDTO{LocalRef: localRef, Status: "recusada", Motivo: "linha-repetida"}, nil
	}
	seen[localRef] = true

	originRef := deviceID + ":" + localRef
	var originCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_paint_origins WHERE origin_ref = ?`, originRef).Scan(&originCount); err != nil {
		return MigrateLineResultDTO{}, err
	}
	if originCount > 0 {
		return MigrateLineResultDTO{LocalRef: localRef, Status: "ja-migrada"}, nil
	}

	fingerprint := sql.NullString{}
	if migrateFingerprintRe.MatchString(line.Fingerprint) {
		fingerprint = sql.NullString{String: line.Fingerprint, Valid: true}
		var knownPaintID sql.NullInt64
		err := conn.QueryRowContext(ctx,
			`SELECT user_paint_id FROM user_paint_origins WHERE fingerprint = ? ORDER BY rowid LIMIT 1`,
			fingerprint.String,
		).Scan(&knownPaintID)
		switch {
		case err == nil:
			// A mesma linha já subiu por outro caminho: registra este caminho
			// apontando para o mesmo destino (inclusive lápide) e não grava tinta.
			if err := insertUserPaintOrigin(ctx, conn, originRef, deviceID, knownPaintID, fingerprint); err != nil {
				return MigrateLineResultDTO{}, err
			}
			return MigrateLineResultDTO{LocalRef: localRef, Status: "ja-migrada"}, nil
		case !errors.Is(err, sql.ErrNoRows):
			return MigrateLineResultDTO{}, err
		}
	}

	manufacturerID, ok := resolveMigrateManufacturer(mfrs, line.Manufacturer, line.ManufacturerID)
	if !ok {
		return MigrateLineResultDTO{LocalRef: localRef, Status: "recusada", Motivo: "fabricante-nao-encontrado"}, nil
	}

	name, quantity, motivo := validateMigrateLine(line)
	if motivo != "" {
		return MigrateLineResultDTO{LocalRef: localRef, Status: "recusada", Motivo: motivo}, nil
	}

	paintTypeID, err := migratePaintTypeID(ctx, conn, line.PaintTypeID)
	if err != nil {
		return MigrateLineResultDTO{}, err
	}
	catalogID, err := migrateCatalogID(ctx, conn, line.CatalogID)
	if err != nil {
		return MigrateLineResultDTO{}, err
	}

	code := strings.TrimSpace(line.Code)
	mergeable := code != "" && !strings.EqualFold(code, "null") && !migrateNullCodeRe.MatchString(code)

	if mergeable {
		targetID, merged, err := s.tryMergeMigrateLine(ctx, conn, manufacturerID, deviceID, code, name, quantity, paintTypeID, catalogID, line.Volume, line.Notes)
		if err != nil {
			return MigrateLineResultDTO{}, err
		}
		if merged {
			if err := insertUserPaintOrigin(ctx, conn, originRef, deviceID, sql.NullInt64{Int64: targetID, Valid: true}, fingerprint); err != nil {
				return MigrateLineResultDTO{}, err
			}
			return MigrateLineResultDTO{LocalRef: localRef, Status: "mesclada"}, nil
		}
	}

	res, err := conn.ExecContext(ctx, `
		INSERT INTO user_paints (manufacturer_id, name, code, rgb_r, rgb_g, rgb_b, volume, notes, quantity, paint_type_id, catalog_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, manufacturerID, name, line.Code, line.R, line.G, line.B, line.Volume, line.Notes, quantity, paintTypeID, catalogID)
	if err != nil {
		return MigrateLineResultDTO{}, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return MigrateLineResultDTO{}, err
	}
	if err := insertUserPaintOrigin(ctx, conn, originRef, deviceID, sql.NullInt64{Int64: newID, Valid: true}, fingerprint); err != nil {
		return MigrateLineResultDTO{}, err
	}
	return MigrateLineResultDTO{LocalRef: localRef, Status: "criada"}, nil
}

// resolveMigrateManufacturer segue RN5.3: o nome local (limpo, sem
// distinguir caixa) vem primeiro — um manufacturerId antigo pode hoje apontar
// para outro fabricante.
func resolveMigrateManufacturer(mfrs []stock.Manufacturer, localName string, manufacturerID int64) (int64, bool) {
	cleaned := cleanName(localName)
	if cleaned != "" {
		for _, m := range mfrs {
			if strings.EqualFold(cleanName(m.Name), cleaned) {
				return m.ID, true
			}
		}
	}
	for _, m := range mfrs {
		if m.ID == manufacturerID {
			return m.ID, true
		}
	}
	return 0, false
}

// validateMigrateLine aplica RN5.4: quantidade 0 vira 1 (nunca recusa por
// isso); paintTypeId/catalogId nunca recusam (resolvidos à parte para NULL).
func validateMigrateLine(l MigrateLineDTO) (name string, quantity int, motivo string) {
	name = cleanName(l.Name)
	if name == "" {
		return "", 0, "nome-obrigatorio"
	}
	if utf8.RuneCountInString(name) > maxUserPaintNameRunes ||
		utf8.RuneCountInString(l.Code) > maxUserPaintCodeRunes ||
		utf8.RuneCountInString(l.Volume) > maxUserPaintVolumeRunes ||
		utf8.RuneCountInString(l.Notes) > maxUserPaintNotesRunes {
		return "", 0, "texto-longo"
	}
	if !validRGBComponent(l.R) || !validRGBComponent(l.G) || !validRGBComponent(l.B) {
		return "", 0, "cor-invalida"
	}
	quantity = l.Quantity
	if quantity == 0 {
		quantity = 1
	} else if quantity < 0 || quantity > 9999 {
		return "", 0, "quantidade-invalida"
	}
	return name, quantity, ""
}

func migratePaintTypeID(ctx context.Context, conn *sql.Conn, in *int64) (sql.NullInt64, error) {
	if in == nil || *in == 0 {
		return sql.NullInt64{}, nil
	}
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM paint_types WHERE id = ?`, *in).Scan(&n); err != nil {
		return sql.NullInt64{}, err
	}
	if n == 0 {
		return sql.NullInt64{}, nil
	}
	return sql.NullInt64{Int64: *in, Valid: true}, nil
}

func migrateCatalogID(ctx context.Context, conn *sql.Conn, in *int64) (sql.NullInt64, error) {
	if in == nil || *in == 0 {
		return sql.NullInt64{}, nil
	}
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM paints WHERE id = ?`, *in).Scan(&n); err != nil {
		return sql.NullInt64{}, err
	}
	if n == 0 {
		return sql.NullInt64{}, nil
	}
	return sql.NullInt64{Int64: *in, Valid: true}, nil
}

func insertUserPaintOrigin(ctx context.Context, conn *sql.Conn, originRef, deviceID string, userPaintID sql.NullInt64, fingerprint sql.NullString) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO user_paint_origins (origin_ref, device_id, user_paint_id, fingerprint) VALUES (?, ?, ?, ?)`,
		originRef, deviceID, userPaintID, fingerprint)
	return err
}

// tryMergeMigrateLine aplica RN5.5: candidatos do mesmo fabricante sem
// nenhuma origem do mesmo device_id, casando código e nome (aparados, sem
// distinguir caixa) em Go — nunca com lower() do SQLite. Menor id vence.
func (s *PaintService) tryMergeMigrateLine(
	ctx context.Context, conn *sql.Conn, manufacturerID int64, deviceID, code, name string,
	quantity int, paintTypeID, catalogID sql.NullInt64, volume, notes string,
) (int64, bool, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, COALESCE(code, ''), name FROM user_paints up
		WHERE manufacturer_id = ?
		  AND NOT EXISTS (SELECT 1 FROM user_paint_origins o WHERE o.user_paint_id = up.id AND o.device_id = ?)
		ORDER BY id
	`, manufacturerID, deviceID)
	if err != nil {
		return 0, false, err
	}

	var targetID int64
	found := false
	for rows.Next() {
		var id int64
		var existingCode, existingName string
		if err := rows.Scan(&id, &existingCode, &existingName); err != nil {
			rows.Close()
			return 0, false, err
		}
		if strings.EqualFold(strings.TrimSpace(existingCode), code) && strings.EqualFold(strings.TrimSpace(existingName), name) {
			targetID = id
			found = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, false, err
	}
	rows.Close()
	if !found {
		return 0, false, nil
	}

	var currentQty int
	var currentType, currentCatalog sql.NullInt64
	var currentVolume, currentNotes string
	if err := conn.QueryRowContext(ctx, `
		SELECT quantity, paint_type_id, catalog_id, COALESCE(volume, ''), COALESCE(notes, '')
		FROM user_paints WHERE id = ?
	`, targetID).Scan(&currentQty, &currentType, &currentCatalog, &currentVolume, &currentNotes); err != nil {
		return 0, false, err
	}

	newQty := currentQty
	if quantity > newQty {
		newQty = quantity
	}
	newType := currentType
	if !newType.Valid {
		newType = paintTypeID
	}
	newCatalog := currentCatalog
	if !newCatalog.Valid {
		newCatalog = catalogID
	}
	newVolume := currentVolume
	if newVolume == "" {
		newVolume = volume
	}
	newNotes := currentNotes
	if newNotes == "" {
		newNotes = notes
	}

	if _, err := conn.ExecContext(ctx, `
		UPDATE user_paints SET quantity = ?, paint_type_id = ?, catalog_id = ?, volume = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newQty, newType, newCatalog, newVolume, newNotes, targetID); err != nil {
		return 0, false, err
	}
	return targetID, true, nil
}
