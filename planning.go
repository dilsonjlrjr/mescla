package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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

		CREATE TABLE IF NOT EXISTS painting_regions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL,
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
			FOREIGN KEY (plan_id) REFERENCES painting_plans(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_painting_regions_plan ON painting_regions(plan_id, sort_order);
	`)
	return err
}

// PaintingPlanDTO é um plano de pintura no formato trocado com o frontend.
type PaintingPlanDTO struct {
	ID          int64               `json:"id"`
	Name        string              `json:"name"`
	ImageData   string              `json:"imageData,omitempty"`
	Regions     []PaintingRegionDTO `json:"regions"`
	CreatedAt   string              `json:"createdAt"`
	UpdatedAt   string              `json:"updatedAt"`
	RegionCount int                 `json:"regionCount,omitempty"`
}

// PaintingRegionDTO é uma região de cor identificada.
type PaintingRegionDTO struct {
	ID         int64   `json:"id"`
	PlanID     int64   `json:"planId,omitempty"`
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
}

const (
	maxRegionsPerPlan = 50
	maxImageDataBytes = 2 * 1024 * 1024 // 2MB
	maxNameChars      = 200
	maxRegionNameChars = 100
	maxNoteChars      = 2000
)

// validar limites de entrada (CAN1, CAN2: servidor deriva id/created_at, ignora do cliente)
func validatePlanInput(plan PaintingPlanDTO) error {
	if len(plan.Regions) > maxRegionsPerPlan {
		return fmt.Errorf("máximo %d regiões por plano", maxRegionsPerPlan)
	}
	if len(plan.ImageData) > maxImageDataBytes {
		return fmt.Errorf("imagem excede 2MB")
	}
	for i, r := range plan.Regions {
		if len(r.RegionName) > maxRegionNameChars {
			return fmt.Errorf("região %d: nome excede %d caracteres", i+1, maxRegionNameChars)
		}
		if len(r.Note) > maxNoteChars {
			return fmt.Errorf("região %d: nota excede %d caracteres", i+1, maxNoteChars)
		}
	}
	return nil
}

// SavePlan cria ou atualiza um plano de pintura.
// Se ID == 0, cria novo. Se ID != 0, atualiza existente (replace all regions).
func (s *PaintService) SavePlan(plan PaintingPlanDTO) (PaintingPlanDTO, error) {
	if err := validatePlanInput(plan); err != nil {
		return PaintingPlanDTO{}, err
	}

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
			"INSERT INTO painting_plans (name, image_data, created_at, updated_at) VALUES (?, ?, ?, ?)",
			plan.Name, plan.ImageData, plan.CreatedAt, plan.UpdatedAt,
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
				// ID do cliente não existe → ignora, cria novo
				plan.ID = 0
				plan.CreatedAt = now
				plan.UpdatedAt = now
				res, insErr := tx.Exec(
					"INSERT INTO painting_plans (name, image_data, created_at, updated_at) VALUES (?, ?, ?, ?)",
					plan.Name, plan.ImageData, plan.CreatedAt, plan.UpdatedAt,
				)
				if insErr != nil {
					return PaintingPlanDTO{}, fmt.Errorf("inserindo plano: %w", insErr)
				}
				plan.ID, _ = res.LastInsertId()
				// Pula o resto do update (sem delete de regiões, sem update)
				if err := tx.Commit(); err != nil {
					return PaintingPlanDTO{}, fmt.Errorf("commit: %w", err)
				}
				return plan, nil
			}
			return PaintingPlanDTO{}, fmt.Errorf("buscando plano existente: %w", err)
		}
		plan.CreatedAt = originalCreatedAt
		plan.UpdatedAt = now

		res, err := tx.Exec(
			"UPDATE painting_plans SET name = ?, image_data = ?, updated_at = ? WHERE id = ?",
			plan.Name, plan.ImageData, plan.UpdatedAt, plan.ID,
		)
		if err != nil {
			return PaintingPlanDTO{}, fmt.Errorf("atualizando plano: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return PaintingPlanDTO{}, fmt.Errorf("plano não encontrado (id %d)", plan.ID)
		}

		// Remove regiões antigas (replace all)
		if _, err := tx.Exec("DELETE FROM painting_regions WHERE plan_id = ?", plan.ID); err != nil {
			return PaintingPlanDTO{}, fmt.Errorf("removendo regiões antigas: %w", err)
		}
	}

	// Insere novas regiões
	for i, r := range plan.Regions {
		_, err := tx.Exec(
			`INSERT INTO painting_regions
			 (plan_id, x, y, r, g, b, hex, region_name, note, paint_id, paint_brand, paint_name, paint_code, delta_e, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			plan.ID, r.X, r.Y, r.R, r.G, r.B, r.Hex, r.RegionName, r.Note,
			nullableInt64(r.PaintID), r.PaintBrand, r.PaintName, r.PaintCode, r.DeltaE, i,
		)
		if err != nil {
			return PaintingPlanDTO{}, fmt.Errorf("inserindo região %d: %w", i+1, err)
		}
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

// ListPlans lista todos os planos (resumo: sem image_data, sem regiões completas).
func (s *PaintService) ListPlans() ([]PaintingPlanDTO, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.created_at, p.updated_at,
		       COALESCE((SELECT COUNT(*) FROM painting_regions WHERE plan_id = p.id), 0)
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

// LoadPlan carrega um plano completo (imagem + todas as regiões).
func (s *PaintService) LoadPlan(id int64) (PaintingPlanDTO, error) {
	var plan PaintingPlanDTO
	err := s.db.QueryRow(
		"SELECT id, name, COALESCE(image_data, ''), created_at, updated_at FROM painting_plans WHERE id = ?",
		id,
	).Scan(&plan.ID, &plan.Name, &plan.ImageData, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return PaintingPlanDTO{}, fmt.Errorf("plano não encontrado (id %d)", id)
		}
		return PaintingPlanDTO{}, err
	}

	// Carrega regiões
	rows, err := s.db.Query(`
		SELECT id, x, y, r, g, b, hex,
		       COALESCE(region_name, ''), COALESCE(note, ''),
		       COALESCE(paint_id, 0), COALESCE(paint_brand, ''),
		       COALESCE(paint_name, ''), COALESCE(paint_code, ''),
		       COALESCE(delta_e, 0)
		FROM painting_regions
		WHERE plan_id = ?
		ORDER BY sort_order
	`, id)
	if err != nil {
		return PaintingPlanDTO{}, err
	}
	defer rows.Close()

	plan.Regions = make([]PaintingRegionDTO, 0)
	for rows.Next() {
		var r PaintingRegionDTO
		if err := rows.Scan(&r.ID, &r.X, &r.Y, &r.R, &r.G, &r.B, &r.Hex,
			&r.RegionName, &r.Note, &r.PaintID, &r.PaintBrand, &r.PaintName, &r.PaintCode, &r.DeltaE,
		); err != nil {
			return PaintingPlanDTO{}, err
		}
		r.PlanID = id
		plan.Regions = append(plan.Regions, r)
	}

	return plan, rows.Err()
}

// DeletePlan remove um plano e suas regiões (ON DELETE CASCADE).
func (s *PaintService) DeletePlan(id int64) error {
	res, err := s.db.Exec("DELETE FROM painting_plans WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("plano não encontrado (id %d)", id)
	}
	return nil
}
