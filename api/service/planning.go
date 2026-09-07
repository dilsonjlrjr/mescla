package service

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"
)

// ensurePlanningSchema cria as tabelas de planos de pintura se não existirem.
// É idempotente (roda a cada boot).
func ensurePlanningSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS painting_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			image_data TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS painting_tabs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			image_data TEXT DEFAULT '',
			selected_manufacturer_id INTEGER,
			use_stock_only INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (plan_id) REFERENCES painting_plans(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_painting_tabs_plan ON painting_tabs(plan_id, sort_order);

		CREATE TABLE IF NOT EXISTS painting_regions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tab_id INTEGER NOT NULL,
			x INTEGER NOT NULL,
			y INTEGER NOT NULL,
			r INTEGER NOT NULL CHECK (r >= 0 AND r <= 255),
			g INTEGER NOT NULL CHECK (g >= 0 AND g <= 255),
			b INTEGER NOT NULL CHECK (b >= 0 AND b <= 255),
			hex TEXT NOT NULL,
			region_name TEXT DEFAULT '',
			note TEXT DEFAULT '',
			paint_id INTEGER,
			paint_brand TEXT DEFAULT '',
			paint_name TEXT DEFAULT '',
			paint_code TEXT DEFAULT '',
			delta_e REAL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			painted INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (tab_id) REFERENCES painting_tabs(id) ON DELETE CASCADE
		);

	`)
	if err != nil {
		return err
	}

	// addColumnIfMissing roda sobre o formato ainda presente em painting_regions
	// (legado com plan_id, ou já migrado com tab_id) — a coluna painted é
	// comum aos dois formatos, então isso precisa acontecer antes da migração
	// de dados abaixo, que depende dela para copiar as regiões.
	if err := addColumnIfMissing(db, "painting_regions", "painted", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "painting_plans", "selected_manufacturer_id", "INTEGER"); err != nil {
		return err
	}

	if err := migratePlanningTabsIfNeeded(db); err != nil {
		return err
	}

	// O índice fica DEPOIS da migração de propósito: numa base do formato
	// antigo, painting_regions ainda tem plan_id quando o bloco de DDL acima
	// roda — o CREATE TABLE IF NOT EXISTS é pulado e criar um índice sobre
	// tab_id falharia com "no such column: tab_id", derrubando o boot da API
	// antes de a migração ter chance de rodar.
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_painting_regions_tab ON painting_regions(tab_id, sort_order);`)
	return err
}

// migratePlanningTabsIfNeeded aplica a RN10: todo plano do formato antigo
// (foto e regiões no próprio plano, coluna painting_regions.plan_id) ganha
// uma aba "Figura 1" levando a foto e o fabricante do plano, com as regiões
// reapontadas para ela. Roda inteira numa transação própria — interrompida
// no meio (CAN8), reverte por completo; rodada de novo depois de já migrado
// (CAN16), é um no-op, detectado pela ausência da coluna legada plan_id.
func migratePlanningTabsIfNeeded(db *sql.DB) error {
	legacy, err := columnExists(db, "painting_regions", "plan_id")
	if err != nil {
		return err
	}
	if !legacy {
		return nil // já migrado, ou instalação nova (nasceu direto no formato de abas)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("iniciando transação de migração de abas: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT p.id, COALESCE(p.image_data, ''), p.selected_manufacturer_id
		FROM painting_plans p
		LEFT JOIN painting_tabs t ON t.plan_id = p.id
		WHERE t.id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("selecionando planos sem aba: %w", err)
	}
	type legacyPlan struct {
		id             int64
		imageData      string
		manufacturerID sql.NullInt64
	}
	var plans []legacyPlan
	for rows.Next() {
		var lp legacyPlan
		if err := rows.Scan(&lp.id, &lp.imageData, &lp.manufacturerID); err != nil {
			rows.Close()
			return fmt.Errorf("lendo plano legado: %w", err)
		}
		plans = append(plans, lp)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterando planos legados: %w", err)
	}
	rows.Close()

	for _, lp := range plans {
		var manufacturerID interface{}
		if lp.manufacturerID.Valid {
			manufacturerID = lp.manufacturerID.Int64
		}
		if _, err := tx.Exec(
			`INSERT INTO painting_tabs (plan_id, name, image_data, selected_manufacturer_id, use_stock_only, sort_order)
			 VALUES (?, 'Figura 1', ?, ?, 0, 0)`,
			lp.id, lp.imageData, manufacturerID,
		); err != nil {
			return fmt.Errorf("criando aba de migração para o plano %d: %w", lp.id, err)
		}
	}

	if _, err := tx.Exec(`
		CREATE TABLE painting_regions_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tab_id INTEGER NOT NULL,
			x INTEGER NOT NULL,
			y INTEGER NOT NULL,
			r INTEGER NOT NULL CHECK (r >= 0 AND r <= 255),
			g INTEGER NOT NULL CHECK (g >= 0 AND g <= 255),
			b INTEGER NOT NULL CHECK (b >= 0 AND b <= 255),
			hex TEXT NOT NULL,
			region_name TEXT DEFAULT '',
			note TEXT DEFAULT '',
			paint_id INTEGER,
			paint_brand TEXT DEFAULT '',
			paint_name TEXT DEFAULT '',
			paint_code TEXT DEFAULT '',
			delta_e REAL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			painted INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (tab_id) REFERENCES painting_tabs(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("criando painting_regions_new: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO painting_regions_new
			(id, tab_id, x, y, r, g, b, hex, region_name, note, paint_id, paint_brand, paint_name, paint_code, delta_e, sort_order, painted)
		SELECT pr.id, t.id, pr.x, pr.y, pr.r, pr.g, pr.b, pr.hex, pr.region_name, pr.note,
		       pr.paint_id, pr.paint_brand, pr.paint_name, pr.paint_code, pr.delta_e, pr.sort_order, pr.painted
		FROM painting_regions pr
		JOIN painting_tabs t ON t.id = (
			SELECT id FROM painting_tabs WHERE plan_id = pr.plan_id
			ORDER BY sort_order, id LIMIT 1
		)
	`); err != nil {
		return fmt.Errorf("copiando regiões para o formato de abas: %w", err)
	}

	// Região cujo plano não existe mais não entra no JOIN e some sem rastro.
	// Contar antes de trocar as tabelas é o que transforma perda silenciosa em
	// linha de log.
	var antes, depois int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM painting_regions`).Scan(&antes); err != nil {
		return fmt.Errorf("contando regiões antes da migração: %w", err)
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM painting_regions_new`).Scan(&depois); err != nil {
		return fmt.Errorf("contando regiões migradas: %w", err)
	}
	if antes != depois {
		log.Printf("migração rf-09: %d região(ões) órfã(s) descartada(s) — sem plano correspondente", antes-depois)
	}

	if _, err := tx.Exec(`DROP TABLE painting_regions`); err != nil {
		return fmt.Errorf("removendo painting_regions legada: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE painting_regions_new RENAME TO painting_regions`); err != nil {
		return fmt.Errorf("renomeando painting_regions_new: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_painting_regions_tab ON painting_regions(tab_id, sort_order)`); err != nil {
		return fmt.Errorf("recriando índice de painting_regions: %w", err)
	}

	return tx.Commit()
}

// columnExists reporta se column existe em table, via PRAGMA table_info.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("PRAGMA table_info(%s): %w", table, err)
	}
	defer rows.Close()

	var (
		cid       int
		name      string
		colType   string
		notNull   int
		dfltValue sql.NullString
		pk        int
	)
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("lendo table_info(%s): %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterando table_info(%s): %w", table, err)
	}
	return false, nil
}

// addColumnIfMissing roda um ALTER TABLE ADD COLUMN de forma idempotente,
// já que o SQLite não suporta ADD COLUMN IF NOT EXISTS. Verifica via
// PRAGMA table_info se a coluna já existe antes de tentar adicioná-la.
func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	exists, err := columnExists(db, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("adicionando coluna %s.%s: %w", table, column, err)
	}
	return nil
}

// PaintingPlanDTO é um plano de pintura no formato trocado com o frontend.
// A partir do rf-09 o plano é só o envelope: foto, fabricante e regiões
// vivem em cada aba (Tabs).
type PaintingPlanDTO struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	Tabs        []PaintingTabDTO `json:"tabs"`
	CreatedAt   string           `json:"createdAt"`
	UpdatedAt   string           `json:"updatedAt"`
	RegionCount int              `json:"regionCount,omitempty"`
}

// PaintingTabDTO é uma aba de figura dentro de um plano (rf-09): foto,
// fabricante/modo e regiões próprios.
type PaintingTabDTO struct {
	ID                     int64               `json:"id"`
	Name                   string              `json:"name"`
	ImageData              string              `json:"imageData,omitempty"`
	SelectedManufacturerID *int64              `json:"selectedManufacturerId,omitempty"`
	UseStockOnly           int                 `json:"useStockOnly"`
	Regions                []PaintingRegionDTO `json:"regions"`
}

// PaintingRegionDTO é uma região de cor identificada, sempre dentro de uma aba.
type PaintingRegionDTO struct {
	ID         int64   `json:"id"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	R          uint8   `json:"r"`
	G          uint8   `json:"g"`
	B          uint8   `json:"b"`
	Hex        string  `json:"hex"`
	RegionName string  `json:"regionName"`
	Note       string  `json:"note"`
	PaintID    int64   `json:"paintId,omitempty"`
	PaintBrand string  `json:"paintBrand"`
	PaintName  string  `json:"paintName"`
	PaintCode  string  `json:"paintCode"`
	DeltaE     float64 `json:"deltaE"`
	Painted    int     `json:"painted"`
}

const (
	maxTabsPerPlan     = 10
	maxTabNameChars    = 80
	maxRegionsPerTab   = 50
	maxImageDataBytes  = 2 * 1024 * 1024 // 2MB, por aba
	maxNameChars       = 200
	maxRegionNameChars = 100
	maxNoteChars       = 2000
)

// validateAndNormalizeTabs valida limites de entrada por aba (RN1, RN2, RN15)
// e aplica as normalizações derivadas no servidor: nome vazio vira "Figura N"
// pela posição (RN3) e useStockOnly vira 0/1 (CAN10). O id de cada aba
// vindo do cliente é ignorado adiante, em SavePlan (CAN1, CAN9: replace-all).
func validateAndNormalizeTabs(tabs []PaintingTabDTO) ([]PaintingTabDTO, error) {
	if len(tabs) == 0 {
		return nil, fmt.Errorf("o plano precisa de pelo menos uma figura")
	}
	if len(tabs) > maxTabsPerPlan {
		return nil, fmt.Errorf("o plano aceita no máximo %d figuras", maxTabsPerPlan)
	}

	normalized := make([]PaintingTabDTO, len(tabs))
	for i, t := range tabs {
		// Conta RUNA, não byte: o cliente mede em UTF-16 e 60 "ç" passariam na
		// tela para serem recusados aqui, com erro genérico.
		if utf8.RuneCountInString(t.Name) > maxTabNameChars {
			return nil, fmt.Errorf("o nome da figura tem no máximo %d caracteres", maxTabNameChars)
		}
		name := strings.TrimSpace(t.Name)
		if name == "" {
			name = fmt.Sprintf("Figura %d", i+1)
		}
		t.Name = name

		if len(t.Regions) > maxRegionsPerTab {
			return nil, fmt.Errorf("%s: o plano aceita no máximo %d regiões", name, maxRegionsPerTab)
		}
		if len(t.ImageData) > maxImageDataBytes {
			return nil, fmt.Errorf("%s: a foto passa de 2 MB — use uma imagem menor", name)
		}
		for _, r := range t.Regions {
			if len(r.RegionName) > maxRegionNameChars {
				return nil, fmt.Errorf("%s: nome da região excede %d caracteres", name, maxRegionNameChars)
			}
			if len(r.Note) > maxNoteChars {
				return nil, fmt.Errorf("%s: nota excede %d caracteres", name, maxNoteChars)
			}
		}

		t.UseStockOnly = normalizeUseStockOnly(t.UseStockOnly)
		normalized[i] = t
	}
	return normalized, nil
}

// SavePlan cria ou atualiza um plano de pintura.
// Se ID == 0, cria novo. Se ID != 0, atualiza existente — replace-all de
// abas e regiões numa transação só (CAN7): id de aba e de região vindos do
// cliente são ignorados, plan_id da aba vem do plano da rota, tab_id da
// região vem da aba que a contém no payload, e sort_order vem do índice.
func (s *PaintService) SavePlan(plan PaintingPlanDTO) (PaintingPlanDTO, error) {
	tabs, err := validateAndNormalizeTabs(plan.Tabs)
	if err != nil {
		return PaintingPlanDTO{}, err
	}
	plan.Tabs = tabs

	// Nome padrão
	if strings.TrimSpace(plan.Name) == "" {
		plan.Name = "Plano " + time.Now().Format("02/01/2006 15:04")
	}
	if len(plan.Name) > maxNameChars {
		plan.Name = plan.Name[:maxNameChars]
	}

	now := time.Now().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return PaintingPlanDTO{}, fmt.Errorf("iniciando transação: %w", err)
	}
	defer tx.Rollback()

	if plan.ID == 0 {
		// CREATE
		plan.ID = 0 // garantir que ID do cliente não vaze
		plan.CreatedAt = now
		plan.UpdatedAt = now
		res, err := tx.Exec(
			"INSERT INTO painting_plans (name, created_at, updated_at) VALUES (?, ?, ?)",
			plan.Name, plan.CreatedAt, plan.UpdatedAt,
		)
		if err != nil {
			return PaintingPlanDTO{}, fmt.Errorf("inserindo plano: %w", err)
		}
		plan.ID, _ = res.LastInsertId()
	} else {
		// UPDATE ou CREATE se ID do cliente não existe
		var originalCreatedAt string
		err := tx.QueryRow("SELECT created_at FROM painting_plans WHERE id = ?", plan.ID).Scan(&originalCreatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				// ID do cliente não existe → ignora, cria novo (D-003: cai no laço comum de
				// inserção de abas/regiões e no commit único do fim, em vez de sair aqui vazio).
				plan.ID = 0
				plan.CreatedAt = now
				plan.UpdatedAt = now
				res, insErr := tx.Exec(
					"INSERT INTO painting_plans (name, created_at, updated_at) VALUES (?, ?, ?)",
					plan.Name, plan.CreatedAt, plan.UpdatedAt,
				)
				if insErr != nil {
					return PaintingPlanDTO{}, fmt.Errorf("inserindo plano: %w", insErr)
				}
				plan.ID, _ = res.LastInsertId()
			} else {
				return PaintingPlanDTO{}, fmt.Errorf("buscando plano existente: %w", err)
			}
		} else {
			plan.CreatedAt = originalCreatedAt
			plan.UpdatedAt = now

			res, err := tx.Exec(
				"UPDATE painting_plans SET name = ?, updated_at = ? WHERE id = ?",
				plan.Name, plan.UpdatedAt, plan.ID,
			)
			if err != nil {
				return PaintingPlanDTO{}, fmt.Errorf("atualizando plano: %w", err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return PaintingPlanDTO{}, fmt.Errorf("plano não encontrado (id %d)", plan.ID)
			}

			// Replace-all: apaga regiões e abas antigas do plano (nessa ordem,
			// sem depender de cascata) antes de reinserir tudo.
			if _, err := tx.Exec(
				"DELETE FROM painting_regions WHERE tab_id IN (SELECT id FROM painting_tabs WHERE plan_id = ?)",
				plan.ID,
			); err != nil {
				return PaintingPlanDTO{}, fmt.Errorf("removendo regiões antigas: %w", err)
			}
			if _, err := tx.Exec("DELETE FROM painting_tabs WHERE plan_id = ?", plan.ID); err != nil {
				return PaintingPlanDTO{}, fmt.Errorf("removendo abas antigas: %w", err)
			}
		}
	}

	// Insere as abas e, dentro de cada uma, suas regiões.
	for ti, t := range plan.Tabs {
		tabRes, err := tx.Exec(
			`INSERT INTO painting_tabs (plan_id, name, image_data, selected_manufacturer_id, use_stock_only, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			plan.ID, t.Name, t.ImageData, t.SelectedManufacturerID, t.UseStockOnly, ti,
		)
		if err != nil {
			return PaintingPlanDTO{}, fmt.Errorf("inserindo aba %q: %w", t.Name, err)
		}
		tabID, _ := tabRes.LastInsertId()
		t.ID = tabID

		for ri, r := range t.Regions {
			painted := normalizePainted(r.Painted)
			r.Painted = painted
			_, err := tx.Exec(
				`INSERT INTO painting_regions
				 (tab_id, x, y, r, g, b, hex, region_name, note, paint_id, paint_brand, paint_name, paint_code, delta_e, sort_order, painted)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				tabID, r.X, r.Y, r.R, r.G, r.B, r.Hex, r.RegionName, r.Note,
				nullableInt64(r.PaintID), r.PaintBrand, r.PaintName, r.PaintCode, r.DeltaE, ri, painted,
			)
			if err != nil {
				return PaintingPlanDTO{}, fmt.Errorf("inserindo região %d da aba %q: %w", ri+1, t.Name, err)
			}
			t.Regions[ri] = r
		}
		plan.Tabs[ti] = t
	}

	if err := tx.Commit(); err != nil {
		return PaintingPlanDTO{}, fmt.Errorf("commit: %w", err)
	}

	return plan, nil
}

func nullableInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// normalizePainted garante que painted grave só 0 ou 1: qualquer valor
// diferente de 0 vira 1 — nunca grava 2 ou outro valor adulterado (CAN6).
func normalizePainted(v int) int {
	if v == 0 {
		return 0
	}
	return 1
}

// normalizeUseStockOnly garante que useStockOnly grave só 0 ou 1, mesma
// regra de normalizePainted (CAN10).
func normalizeUseStockOnly(v int) int {
	return normalizePainted(v)
}

// ListPlans lista todos os planos (resumo: sem fotos, sem regiões completas).
func (s *PaintService) ListPlans() ([]PaintingPlanDTO, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.created_at, p.updated_at,
		       COALESCE((
		           SELECT COUNT(*)
		           FROM painting_regions pr
		           JOIN painting_tabs t ON t.id = pr.tab_id
		           WHERE t.plan_id = p.id
		       ), 0)
		FROM painting_plans p
		ORDER BY p.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PaintingPlanDTO, 0)
	for rows.Next() {
		var p PaintingPlanDTO
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt, &p.RegionCount); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// LoadPlan carrega um plano completo: todas as abas em sort_order, cada uma
// com suas regiões em sort_order.
func (s *PaintService) LoadPlan(id int64) (PaintingPlanDTO, error) {
	var plan PaintingPlanDTO
	err := s.db.QueryRow(
		"SELECT id, name, created_at, updated_at FROM painting_plans WHERE id = ?",
		id,
	).Scan(&plan.ID, &plan.Name, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return PaintingPlanDTO{}, fmt.Errorf("plano não encontrado (id %d)", id)
		}
		return PaintingPlanDTO{}, err
	}

	tabRows, err := s.db.Query(`
		SELECT id, name, COALESCE(image_data, ''), selected_manufacturer_id, use_stock_only
		FROM painting_tabs
		WHERE plan_id = ?
		ORDER BY sort_order
	`, id)
	if err != nil {
		return PaintingPlanDTO{}, err
	}
	defer tabRows.Close()

	plan.Tabs = make([]PaintingTabDTO, 0)
	for tabRows.Next() {
		var t PaintingTabDTO
		var manufacturerID sql.NullInt64
		if err := tabRows.Scan(&t.ID, &t.Name, &t.ImageData, &manufacturerID, &t.UseStockOnly); err != nil {
			return PaintingPlanDTO{}, err
		}
		if manufacturerID.Valid {
			t.SelectedManufacturerID = &manufacturerID.Int64
		}
		plan.Tabs = append(plan.Tabs, t)
	}
	if err := tabRows.Err(); err != nil {
		return PaintingPlanDTO{}, err
	}

	for i := range plan.Tabs {
		regions, err := s.loadTabRegions(plan.Tabs[i].ID)
		if err != nil {
			return PaintingPlanDTO{}, err
		}
		plan.Tabs[i].Regions = regions
	}

	return plan, nil
}

// loadTabRegions carrega as regiões de uma aba, em sort_order.
func (s *PaintService) loadTabRegions(tabID int64) ([]PaintingRegionDTO, error) {
	rows, err := s.db.Query(`
		SELECT id, x, y, r, g, b, hex,
		       COALESCE(region_name, ''), COALESCE(note, ''),
		       COALESCE(paint_id, 0), COALESCE(paint_brand, ''),
		       COALESCE(paint_name, ''), COALESCE(paint_code, ''),
		       COALESCE(delta_e, 0), COALESCE(painted, 0)
		FROM painting_regions
		WHERE tab_id = ?
		ORDER BY sort_order
	`, tabID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := make([]PaintingRegionDTO, 0)
	for rows.Next() {
		var r PaintingRegionDTO
		if err := rows.Scan(&r.ID, &r.X, &r.Y, &r.R, &r.G, &r.B, &r.Hex,
			&r.RegionName, &r.Note, &r.PaintID, &r.PaintBrand, &r.PaintName, &r.PaintCode, &r.DeltaE,
			&r.Painted,
		); err != nil {
			return nil, err
		}
		regions = append(regions, r)
	}
	return regions, rows.Err()
}

// DeletePlan remove um plano e suas abas/regiões (ON DELETE CASCADE em cadeia).
func (s *PaintService) DeletePlan(id int64) error {
	// Apaga os três níveis explicitamente, numa transação: depender só do
	// ON DELETE CASCADE deixaria abas e regiões órfãs em qualquer conexão do
	// pool onde o PRAGMA foreign_keys não estivesse ligado.
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM painting_regions WHERE tab_id IN
		(SELECT id FROM painting_tabs WHERE plan_id = ?)`, id); err != nil {
		return fmt.Errorf("removendo regiões do plano: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM painting_tabs WHERE plan_id = ?", id); err != nil {
		return fmt.Errorf("removendo abas do plano: %w", err)
	}
	res, err := tx.Exec("DELETE FROM painting_plans WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("plano não encontrado (id %d)", id)
	}
	return tx.Commit()
}
