package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// newPlanningTestService abre um banco :memory: já no schema novo (abas,
// rf-09) e liga PRAGMA foreign_keys — sem isso a cascata de dois níveis
// (plans → tabs → regions) não é exercitada de verdade nos testes.
func newPlanningTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	// Uma conexão só: com ":memory:", cada conexão do pool abre um banco
	// PRÓPRIO e vazio — um teste concorrente cairia em "no such table".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if err := ensurePlanningSchema(db); err != nil {
		t.Fatalf("ensurePlanningSchema: %v", err)
	}
	return &PaintService{db: db}
}

// newLegacyPlanningDB cria à mão o schema anterior ao rf-09 — sem
// painting_tabs, com a foto e o fabricante no próprio painting_plans e
// painting_regions.plan_id em vez de tab_id. newPlanningTestService não serve
// para semear esse formato porque já entrega o schema novo (com abas); este
// helper irmão é o único jeito de exercitar a migração (CA15, CA16, CAN8).
func newLegacyPlanningDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	// Uma conexão só: com ":memory:", cada conexão do pool abre um banco
	// PRÓPRIO e vazio — um teste concorrente cairia em "no such table".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE painting_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			image_data TEXT DEFAULT '',
			selected_manufacturer_id INTEGER,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE painting_regions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL,
			x INTEGER NOT NULL,
			y INTEGER NOT NULL,
			r INTEGER NOT NULL,
			g INTEGER NOT NULL,
			b INTEGER NOT NULL,
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
			FOREIGN KEY (plan_id) REFERENCES painting_plans(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("criando schema legado: %v", err)
	}
	return db
}

// insertLegacyPlan grava um plano direto no formato antigo (foto e
// fabricante no próprio plano), sem passar por SavePlan.
func insertLegacyPlan(t *testing.T, db *sql.DB, name, imageData string, manufacturerID *int64) int64 {
	t.Helper()
	var mfg interface{}
	if manufacturerID != nil {
		mfg = *manufacturerID
	}
	res, err := db.Exec(
		"INSERT INTO painting_plans (name, image_data, selected_manufacturer_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		name, imageData, mfg, "2020-01-01T00:00:00Z", "2020-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("inserindo plano legado: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// insertLegacyRegion grava uma região direto no formato antigo
// (painting_regions.plan_id), sem passar por SavePlan.
func insertLegacyRegion(t *testing.T, db *sql.DB, planID int64, r PaintingRegionDTO, sortOrder int) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO painting_regions
		 (plan_id, x, y, r, g, b, hex, region_name, note, paint_id, paint_brand, paint_name, paint_code, delta_e, sort_order, painted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		planID, r.X, r.Y, r.R, r.G, r.B, r.Hex, r.RegionName, r.Note,
		nullableInt64(r.PaintID), r.PaintBrand, r.PaintName, r.PaintCode, r.DeltaE, sortOrder, r.Painted,
	)
	if err != nil {
		t.Fatalf("inserindo região legada: %v", err)
	}
}

// ========== CA15: plano no formato legado ganha aba Figura 1 com a foto e as regiões reapontadas ==========
func TestMigratePlanningTabsCreatesFigura1FromLegacyPlan(t *testing.T) {
	db := newLegacyPlanningDB(t)

	manufacturerID := int64(7)
	planID := insertLegacyPlan(t, db, "Plano Legado", "data:image/png;base64,legacyimg", &manufacturerID)
	insertLegacyRegion(t, db, planID, PaintingRegionDTO{
		X: 10, Y: 20, R: 255, G: 0, B: 0, Hex: "#FF0000",
		RegionName: "capa", Note: "nota legada",
		PaintID: 42, PaintBrand: "Citadel", PaintName: "Mephiston Red", PaintCode: "Base",
		DeltaE: 1.5, Painted: 1,
	}, 0)

	if err := ensurePlanningSchema(db); err != nil {
		t.Fatalf("ensurePlanningSchema: %v", err)
	}

	s := &PaintService{db: db}
	plan, err := s.LoadPlan(planID)
	if err != nil {
		t.Fatalf("LoadPlan após migração: %v", err)
	}
	if len(plan.Tabs) != 1 {
		t.Fatalf("esperava 1 aba após migração, veio %d", len(plan.Tabs))
	}
	tab := plan.Tabs[0]
	if tab.Name != "Figura 1" {
		t.Fatalf("aba migrada deveria se chamar 'Figura 1', veio '%s'", tab.Name)
	}
	if tab.ImageData != "data:image/png;base64,legacyimg" {
		t.Fatal("foto do plano legado não foi levada para a aba")
	}
	if tab.SelectedManufacturerID == nil || *tab.SelectedManufacturerID != 7 {
		t.Fatalf("fabricante do plano legado não foi levado para a aba: %+v", tab.SelectedManufacturerID)
	}
	if len(tab.Regions) != 1 {
		t.Fatalf("esperava 1 região reapontada, veio %d", len(tab.Regions))
	}
	r := tab.Regions[0]
	if r.RegionName != "capa" || r.Note != "nota legada" || r.PaintBrand != "Citadel" || r.DeltaE != 1.5 || r.Painted != 1 {
		t.Fatalf("dados da região não preservados na migração: %+v", r)
	}
}

// ========== CA16: rodar ensurePlanningSchema de novo não cria aba duplicada ==========
func TestMigratePlanningTabsIsIdempotent(t *testing.T) {
	db := newLegacyPlanningDB(t)

	planID := insertLegacyPlan(t, db, "Plano Legado", "", nil)
	insertLegacyRegion(t, db, planID, PaintingRegionDTO{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"}, 0)

	if err := ensurePlanningSchema(db); err != nil {
		t.Fatalf("primeira ensurePlanningSchema: %v", err)
	}
	if err := ensurePlanningSchema(db); err != nil {
		t.Fatalf("segunda ensurePlanningSchema: %v", err)
	}

	var tabCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM painting_tabs WHERE plan_id = ?", planID).Scan(&tabCount); err != nil {
		t.Fatalf("contando abas: %v", err)
	}
	if tabCount != 1 {
		t.Fatalf("esperava 1 aba após rodar a migração duas vezes, veio %d", tabCount)
	}
}

// ========== CAN8: migração interrompida reverte inteira ==========
// Chama migratePlanningTabsIfNeeded diretamente (não ensurePlanningSchema):
// o defeito de planning.go:33 (ver retorno) faz ensurePlanningSchema falhar
// antes mesmo de chegar na transação de migração, contra um banco legado de
// verdade — testar a função da migração isolada é o único jeito de provar o
// rollback em si sem tocar em planning.go.
func TestMigratePlanningTabsRevertsOnFailure(t *testing.T) {
	db := newLegacyPlanningDB(t)

	planID := insertLegacyPlan(t, db, "Plano Legado", "foto", nil)
	insertLegacyRegion(t, db, planID, PaintingRegionDTO{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"}, 0)

	// migratePlanningTabsIfNeeded pressupõe painting_tabs já existente
	// (normalmente criada antes dela, em ensurePlanningSchema).
	if _, err := db.Exec(`
		CREATE TABLE painting_tabs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			image_data TEXT DEFAULT '',
			selected_manufacturer_id INTEGER,
			use_stock_only INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (plan_id) REFERENCES painting_plans(id) ON DELETE CASCADE
		)
	`); err != nil {
		t.Fatalf("criando painting_tabs: %v", err)
	}

	// Força a falha do CREATE TABLE painting_regions_new no meio da
	// transação de migração, pré-criando a tabela que ela tentaria criar.
	if _, err := db.Exec(`CREATE TABLE painting_regions_new (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("pré-criando tabela para forçar falha: %v", err)
	}

	if err := migratePlanningTabsIfNeeded(db); err == nil {
		t.Fatal("esperava erro na migração interrompida")
	}

	var tabCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM painting_tabs").Scan(&tabCount); err != nil {
		t.Fatalf("contando abas: %v", err)
	}
	if tabCount != 0 {
		t.Fatalf("migração interrompida deveria reverter a aba criada, veio %d aba(s)", tabCount)
	}

	legacy, err := columnExists(db, "painting_regions", "plan_id")
	if err != nil {
		t.Fatalf("columnExists: %v", err)
	}
	if !legacy {
		t.Fatal("painting_regions deveria continuar no formato legado (plan_id) após reverter")
	}

	var regionCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM painting_regions WHERE plan_id = ?", planID).Scan(&regionCount); err != nil {
		t.Fatalf("contando regiões legadas: %v", err)
	}
	if regionCount != 1 {
		t.Fatalf("região legada deveria continuar intacta, veio %d", regionCount)
	}
}

// ========== CA5: ordem das abas persiste depois de reordenar, salvar e recarregar ==========
func TestSavePlanTabOrderPersists(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Ordem",
		Tabs: []PaintingTabDTO{
			{Name: "Primeira"},
			{Name: "Segunda"},
			{Name: "Terceira"},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	// Reordena: segunda vai para o fim.
	plan.Tabs = []PaintingTabDTO{plan.Tabs[0], plan.Tabs[2], plan.Tabs[1]}
	if _, err := s.SavePlan(plan); err != nil {
		t.Fatalf("SavePlan (reordenar): %v", err)
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 3 {
		t.Fatalf("esperava 3 abas, veio %d", len(loaded.Tabs))
	}
	got := []string{loaded.Tabs[0].Name, loaded.Tabs[1].Name, loaded.Tabs[2].Name}
	want := []string{"Primeira", "Terceira", "Segunda"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordem das abas não persistiu: got %v, want %v", got, want)
		}
	}
}

// ========== CA6: excluir aba leva as regiões e não afeta outra aba ==========
func TestSavePlanDeleteTabRemovesItsRegionsOnly(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Duas Abas",
		Tabs: []PaintingTabDTO{
			{Name: "Figura 1", Regions: []PaintingRegionDTO{
				{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"},
				{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"},
			}},
			{Name: "Figura 2", Regions: []PaintingRegionDTO{
				{X: 2, Y: 2, R: 2, G: 2, B: 2, Hex: "#020202"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	// Exclui a primeira aba (replace-all: reenvia só a que sobrou).
	plan.Tabs = []PaintingTabDTO{plan.Tabs[1]}
	updated, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan (excluir aba): %v", err)
	}

	loaded, err := s.LoadPlan(updated.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 1 {
		t.Fatalf("esperava 1 aba após excluir, veio %d", len(loaded.Tabs))
	}
	if loaded.Tabs[0].Name != "Figura 2" {
		t.Fatalf("aba remanescente deveria ser 'Figura 2', veio '%s'", loaded.Tabs[0].Name)
	}
	if len(loaded.Tabs[0].Regions) != 1 {
		t.Fatalf("regiões da aba remanescente não deveriam mudar, veio %d", len(loaded.Tabs[0].Regions))
	}

	var totalRegions int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM painting_regions").Scan(&totalRegions); err != nil {
		t.Fatalf("contando regiões: %v", err)
	}
	if totalRegions != 1 {
		t.Fatalf("regiões da aba excluída deveriam ter sido removidas, restaram %d no total", totalRegions)
	}
}

// ========== CA19: round-trip de 2 abas com foto, regiões e fabricante ==========
func TestSavePlanRoundTripsTwoTabs(t *testing.T) {
	s := newPlanningTestService(t)
	mfg := int64(3)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Duas Figuras",
		Tabs: []PaintingTabDTO{
			{
				Name:                   "Figura 1",
				ImageData:              "data:image/png;base64,foto1",
				SelectedManufacturerID: &mfg,
				Regions: []PaintingRegionDTO{
					{X: 10, Y: 20, R: 255, G: 0, B: 0, Hex: "#FF0000", RegionName: "capa"},
				},
			},
			{
				Name:      "Costas",
				ImageData: "data:image/png;base64,foto2",
				Regions: []PaintingRegionDTO{
					{X: 5, Y: 5, R: 0, G: 0, B: 255, Hex: "#0000FF", RegionName: "cinto"},
					{X: 6, Y: 6, R: 0, G: 255, B: 0, Hex: "#00FF00", RegionName: "manto"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 2 {
		t.Fatalf("esperava 2 abas, veio %d", len(loaded.Tabs))
	}
	t1, t2 := loaded.Tabs[0], loaded.Tabs[1]
	if t1.Name != "Figura 1" || t1.ImageData != "data:image/png;base64,foto1" {
		t.Fatalf("aba 1 não preservada: %+v", t1)
	}
	if t1.SelectedManufacturerID == nil || *t1.SelectedManufacturerID != 3 {
		t.Fatalf("fabricante da aba 1 não preservado: %+v", t1.SelectedManufacturerID)
	}
	if len(t1.Regions) != 1 {
		t.Fatalf("aba 1 deveria ter 1 região, veio %d", len(t1.Regions))
	}
	if t2.Name != "Costas" || t2.ImageData != "data:image/png;base64,foto2" {
		t.Fatalf("aba 2 não preservada: %+v", t2)
	}
	if len(t2.Regions) != 2 {
		t.Fatalf("aba 2 deveria ter 2 regiões, veio %d", len(t2.Regions))
	}
}

// ========== CA26: regionCount de ListPlans soma as regiões de todas as abas ==========
func TestListPlansRegionCountSumsAllTabs(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Soma",
		Tabs: []PaintingTabDTO{
			{Name: "Figura 1", Regions: []PaintingRegionDTO{
				{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"},
				{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"},
				{X: 2, Y: 2, R: 2, G: 2, B: 2, Hex: "#020202"},
			}},
			{Name: "Figura 2", Regions: []PaintingRegionDTO{
				{X: 3, Y: 3, R: 3, G: 3, B: 3, Hex: "#030303"},
				{X: 4, Y: 4, R: 4, G: 4, B: 4, Hex: "#040404"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	list, err := s.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("esperava 1 plano, veio %d", len(list))
	}
	if list[0].ID != plan.ID {
		t.Fatal("plano errado na listagem")
	}
	if list[0].RegionCount != 5 {
		t.Fatalf("regionCount deveria somar as duas abas (5), veio %d", list[0].RegionCount)
	}
}

// ========== CA23: nome de aba vazio vira Figura N pela posição ==========
func TestSavePlanEmptyTabNameBecomesFiguraByPosition(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Nomes",
		Tabs: []PaintingTabDTO{
			{Name: "Capa"},
			{Name: "   "}, // vazio após trim
			{Name: ""},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.Tabs[0].Name != "Capa" {
		t.Fatalf("nome preenchido não deveria mudar: '%s'", plan.Tabs[0].Name)
	}
	if plan.Tabs[1].Name != "Figura 2" {
		t.Fatalf("2ª aba vazia deveria virar 'Figura 2', veio '%s'", plan.Tabs[1].Name)
	}
	if plan.Tabs[2].Name != "Figura 3" {
		t.Fatalf("3ª aba vazia deveria virar 'Figura 3', veio '%s'", plan.Tabs[2].Name)
	}
}

// ========== CA24: nome de aba > 80 caracteres é recusado ==========
func TestSavePlanTabNameTooLongRefused(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.SavePlan(PaintingPlanDTO{
		Name: "Nome Longo",
		Tabs: []PaintingTabDTO{{Name: strings.Repeat("a", maxTabNameChars+1)}},
	})
	if err == nil {
		t.Fatal("esperava erro para nome de aba com 81 caracteres")
	}
	if !strings.Contains(err.Error(), fmt.Sprint(maxTabNameChars)) {
		t.Fatalf("erro deveria mencionar o limite de %d: %v", maxTabNameChars, err)
	}
}

// ========== CA25: foto de aba > 2 MB é recusada citando o nome da aba ==========
func TestSavePlanTabImageTooLargeRefusedCitingTabName(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.SavePlan(PaintingPlanDTO{
		Name: "Foto Grande",
		Tabs: []PaintingTabDTO{{
			Name:      "Costas",
			ImageData: strings.Repeat("x", maxImageDataBytes+1),
		}},
	})
	if err == nil {
		t.Fatal("esperava erro para foto > 2MB")
	}
	if !strings.Contains(err.Error(), "Costas") {
		t.Fatalf("erro deveria citar o nome da aba 'Costas': %v", err)
	}
}

// ========== CA8: 11ª aba é recusada ==========
func TestSavePlanEleventhTabRefused(t *testing.T) {
	s := newPlanningTestService(t)

	tabs := make([]PaintingTabDTO, maxTabsPerPlan+1)
	for i := range tabs {
		tabs[i] = PaintingTabDTO{Name: fmt.Sprintf("Figura %d", i+1)}
	}

	_, err := s.SavePlan(PaintingPlanDTO{Name: "Muitas Abas", Tabs: tabs})
	if err == nil {
		t.Fatal("esperava erro para 11 abas")
	}
	if !strings.Contains(err.Error(), fmt.Sprint(maxTabsPerPlan)) {
		t.Fatalf("erro deveria mencionar o limite de %d: %v", maxTabsPerPlan, err)
	}
}

// ========== CA9: 51ª região numa aba é recusada, com a mensagem citando a aba ==========
func TestSavePlanFiftyFirstRegionInTabRefusedCitingTab(t *testing.T) {
	s := newPlanningTestService(t)

	regions := make([]PaintingRegionDTO, maxRegionsPerTab+1)
	for i := range regions {
		regions[i] = PaintingRegionDTO{X: i, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"}
	}

	_, err := s.SavePlan(PaintingPlanDTO{
		Name: "Muitas Regiões",
		Tabs: []PaintingTabDTO{{Name: "Detalhe", Regions: regions}},
	})
	if err == nil {
		t.Fatal("esperava erro para 51 regiões numa aba")
	}
	if !strings.Contains(err.Error(), "Detalhe") || !strings.Contains(err.Error(), fmt.Sprint(maxRegionsPerTab)) {
		t.Fatalf("erro deveria citar a aba 'Detalhe' e o limite de %d: %v", maxRegionsPerTab, err)
	}
}

// ========== CA27: useStockOnly e selectedManufacturerId sobrevivem ao round-trip ==========
func TestSavePlanRoundTripsUseStockOnlyAndManufacturer(t *testing.T) {
	s := newPlanningTestService(t)
	mfg := int64(9)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Estoque",
		Tabs: []PaintingTabDTO{{
			Name:                   "Figura 1",
			SelectedManufacturerID: &mfg,
			UseStockOnly:           1,
		}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	tab := loaded.Tabs[0]
	if tab.UseStockOnly != 1 {
		t.Fatalf("useStockOnly deveria sobreviver ao round-trip, veio %d", tab.UseStockOnly)
	}
	if tab.SelectedManufacturerID == nil || *tab.SelectedManufacturerID != 9 {
		t.Fatalf("selectedManufacturerId deveria sobreviver ao round-trip: %+v", tab.SelectedManufacturerID)
	}
}

// ========== CAN1: id/planId de aba vindos do cliente são ignorados ==========
func TestSavePlanIgnoresClientTabIDAndPlanID(t *testing.T) {
	s := newPlanningTestService(t)

	other, err := s.SavePlan(PaintingPlanDTO{Name: "Outro Plano", Tabs: []PaintingTabDTO{{Name: "X"}}})
	if err != nil {
		t.Fatalf("SavePlan (outro plano): %v", err)
	}

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Plano Alvo",
		Tabs: []PaintingTabDTO{{
			ID:   other.Tabs[0].ID, // tenta forçar id de aba de outro plano
			Name: "Figura 1",
		}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.Tabs[0].ID == other.Tabs[0].ID {
		t.Fatal("id de aba vindo do cliente não deveria ser aceito")
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 1 || loaded.Tabs[0].Name != "Figura 1" {
		t.Fatalf("aba do plano alvo não ficou correta: %+v", loaded.Tabs)
	}

	// A aba original não pode ter mudado de plano (planId derivado da rota,
	// nunca do id de aba enviado pelo cliente).
	var hijackedPlanID int64
	if err := s.db.QueryRow("SELECT plan_id FROM painting_tabs WHERE id = ?", other.Tabs[0].ID).Scan(&hijackedPlanID); err != nil {
		t.Fatalf("consultando aba original: %v", err)
	}
	if hijackedPlanID != other.ID {
		t.Fatalf("aba original não deveria mudar de plano: agora pertence a %d", hijackedPlanID)
	}
}

// ========== CAN3: sortOrder do cliente é ignorado — a ordem vem do índice ==========
func TestSavePlanIgnoresClientSortOrder(t *testing.T) {
	s := newPlanningTestService(t)

	raw := `{
		"name": "Ordem Cliente",
		"tabs": [
			{"name": "Primeira", "sortOrder": 5, "regions": [
				{"x":0,"y":0,"r":0,"g":0,"b":0,"hex":"#000000","sortOrder":9}
			]},
			{"name": "Segunda", "sortOrder": 1}
		]
	}`
	var plan PaintingPlanDTO
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	saved, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	loaded, err := s.LoadPlan(saved.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 2 || loaded.Tabs[0].Name != "Primeira" || loaded.Tabs[1].Name != "Segunda" {
		t.Fatalf("ordem das abas deveria vir do índice do payload, não de 'sortOrder': %+v", loaded.Tabs)
	}
}

// ========== CAN9: região com id de outra aba é gravada sob a aba que a contém ==========
func TestSavePlanRegionWithForeignTabIDStaysUnderContainingTab(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Duas Abas",
		Tabs: []PaintingTabDTO{
			{Name: "Figura 1", Regions: []PaintingRegionDTO{
				{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"},
			}},
			{Name: "Figura 2", Regions: []PaintingRegionDTO{
				{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	foreignRegionID := plan.Tabs[0].Regions[0].ID

	// Reenvia com a região da aba 2 carregando o id de uma região da aba 1.
	plan.Tabs[1].Regions[0].ID = foreignRegionID
	updated, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan (id cruzado): %v", err)
	}

	loaded, err := s.LoadPlan(updated.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs[0].Regions) != 1 || loaded.Tabs[0].Regions[0].X != 0 {
		t.Fatalf("região da aba 1 não deveria ser afetada: %+v", loaded.Tabs[0].Regions)
	}
	if len(loaded.Tabs[1].Regions) != 1 || loaded.Tabs[1].Regions[0].X != 1 {
		t.Fatalf("região com id emprestado deveria ficar sob a aba 2: %+v", loaded.Tabs[1].Regions)
	}
}

// ========== CAN10: useStockOnly fora de 0/1 é normalizado ==========
func TestSavePlanNormalizesUseStockOnlyOutsideRange(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Normaliza",
		Tabs: []PaintingTabDTO{{Name: "Figura 1", UseStockOnly: 2}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.Tabs[0].UseStockOnly != 1 {
		t.Fatalf("useStockOnly=2 deveria normalizar para 1, veio %d", plan.Tabs[0].UseStockOnly)
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if loaded.Tabs[0].UseStockOnly != 1 {
		t.Fatalf("useStockOnly normalizado deveria persistir, veio %d", loaded.Tabs[0].UseStockOnly)
	}
}

// ========== D-003: id inexistente cria novo plano e mantém o conteúdo enviado ==========
func TestSavePlanUnknownIDCreatesAndKeepsContent(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		ID:   999,
		Name: "Fantasma",
		Tabs: []PaintingTabDTO{{Name: "Figura 1", Regions: []PaintingRegionDTO{
			{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101", RegionName: "único"},
		}}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.ID == 999 || plan.ID == 0 {
		t.Fatalf("esperava novo ID gerado pelo servidor, veio %d", plan.ID)
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if loaded.Name != "Fantasma" {
		t.Fatalf("conteúdo do plano deveria ser mantido, veio nome '%s'", loaded.Name)
	}
	if len(loaded.Tabs) != 1 || len(loaded.Tabs[0].Regions) != 1 || loaded.Tabs[0].Regions[0].RegionName != "único" {
		t.Fatalf("conteúdo (abas/regiões) deveria ser mantido: %+v", loaded.Tabs)
	}
}

// ========== preserva: truncagem do nome do plano acima do limite ==========
func TestSavePlanTruncatesLongName(t *testing.T) {
	s := newPlanningTestService(t)

	longName := strings.Repeat("a", maxNameChars+50)
	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: longName,
		Tabs: []PaintingTabDTO{{Name: "Figura 1"}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if len(plan.Name) != maxNameChars {
		t.Fatalf("nome do plano deveria ser truncado em %d, veio %d", maxNameChars, len(plan.Name))
	}
}

// ========== preserva: nome vazio de plano gera nome padrão ==========
func TestSavePlanDefaultName(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{Name: "", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan sem nome: %v", err)
	}
	if !strings.HasPrefix(plan.Name, "Plano ") {
		t.Fatalf("nome deveria começar com 'Plano ', veio '%s'", plan.Name)
	}
}

// ========== preserva: plano sem nenhuma aba é recusado ==========
func TestSavePlanRequiresAtLeastOneTab(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.SavePlan(PaintingPlanDTO{Name: "Sem Abas"})
	if err == nil {
		t.Fatal("esperava erro para plano sem nenhuma aba")
	}
	if !strings.Contains(err.Error(), "pelo menos uma figura") {
		t.Fatalf("erro deveria mencionar 'pelo menos uma figura': %v", err)
	}
}

// ========== preserva: DeletePlan remove em cascata (plans → tabs → regions) ==========
func TestDeletePlanCascadesThroughTabsAndRegions(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Para Deletar",
		Tabs: []PaintingTabDTO{{Name: "Figura 1", Regions: []PaintingRegionDTO{
			{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"},
			{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"},
		}}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	if err := s.DeletePlan(plan.ID); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}

	_, err = s.LoadPlan(plan.ID)
	if err == nil || !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("LoadPlan deveria falhar após delete, veio: %v", err)
	}

	var tabCount, regionCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM painting_tabs").Scan(&tabCount); err != nil {
		t.Fatalf("contando abas: %v", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM painting_regions").Scan(&regionCount); err != nil {
		t.Fatalf("contando regiões: %v", err)
	}
	if tabCount != 0 || regionCount != 0 {
		t.Fatalf("cascata de dois níveis deveria remover abas e regiões, veio %d abas e %d regiões", tabCount, regionCount)
	}
}

// ========== preserva: update faz replace-all de abas e regiões ==========
func TestSavePlanUpdateReplacesTabsAndRegions(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "V1",
		Tabs: []PaintingTabDTO{{Name: "Antiga", Regions: []PaintingRegionDTO{
			{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000", RegionName: "antiga"},
		}}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	plan.Name = "V2"
	plan.Tabs = []PaintingTabDTO{{Name: "Nova", Regions: []PaintingRegionDTO{
		{X: 1, Y: 1, R: 255, G: 255, B: 255, Hex: "#FFFFFF", RegionName: "nova1"},
		{X: 2, Y: 2, R: 128, G: 128, B: 128, Hex: "#808080", RegionName: "nova2"},
	}}}

	updated, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan (update): %v", err)
	}
	if updated.ID != plan.ID {
		t.Fatal("ID deve ser mantido no update")
	}

	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(loaded.Tabs) != 1 || loaded.Tabs[0].Name != "Nova" {
		t.Fatalf("aba antiga deveria ter sido substituída: %+v", loaded.Tabs)
	}
	if len(loaded.Tabs[0].Regions) != 2 || loaded.Tabs[0].Regions[0].RegionName != "nova1" {
		t.Fatal("regiões antigas não foram substituídas")
	}
}

// ========== preserva: aba sem foto fica com image_data vazio ==========
func TestSavePlanTabWithoutImage(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name: "Sem Imagem",
		Tabs: []PaintingTabDTO{{Name: "Figura 1", Regions: []PaintingRegionDTO{
			{X: 0, Y: 0, R: 128, G: 128, B: 128, Hex: "#808080"},
		}}},
	})
	if err != nil {
		t.Fatalf("SavePlan sem imagem: %v", err)
	}
	loaded, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if loaded.Tabs[0].ImageData != "" {
		t.Fatal("image_data da aba deveria ser vazio")
	}
}

// ========== preserva: id de plano vindo do cliente é ignorado ==========
func TestSavePlanIgnoresClientPlanID(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{ID: 999, Name: "Hack", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.ID == 999 {
		t.Fatal("ID do cliente não deve ser aceito — servidor gera")
	}
}

// ========== preserva: created_at não muda com um update ==========
func TestSavePlanIgnoresClientCreatedAt(t *testing.T) {
	s := newPlanningTestService(t)

	plan1, err := s.SavePlan(PaintingPlanDTO{Name: "Ref1", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	updated, err := s.SavePlan(plan1)
	if err != nil {
		t.Fatalf("SavePlan (update): %v", err)
	}
	loaded, err := s.LoadPlan(updated.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if loaded.CreatedAt != plan1.CreatedAt {
		t.Fatal("created_at não deve ser alterado pelo cliente")
	}
}

// ========== preserva: LoadPlan com id inexistente retorna erro ==========
func TestLoadPlanNotFound(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.LoadPlan(99999)
	if err == nil {
		t.Fatal("esperava erro para id inexistente")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("erro deveria conter 'não encontrado', veio: %v", err)
	}
}

// ========== preserva: DeletePlan com id inexistente retorna erro ==========
func TestDeletePlanNotFound(t *testing.T) {
	s := newPlanningTestService(t)

	err := s.DeletePlan(99999)
	if err == nil {
		t.Fatal("esperava erro para id inexistente")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("erro deveria conter 'não encontrado', veio: %v", err)
	}
}

// CAN2 — payload com 11 abas é recusado e nada é gravado.
func TestSavePlanCAN2ElevenTabsRefusedAndNothingWritten(t *testing.T) {
	svc := newPlanningTestService(t)

	tabs := make([]PaintingTabDTO, 11)
	for i := range tabs {
		tabs[i] = PaintingTabDTO{Name: fmt.Sprintf("Figura %d", i+1)}
	}
	if _, err := svc.SavePlan(PaintingPlanDTO{Name: "Onze abas", Tabs: tabs}); err == nil {
		t.Fatal("11 abas deveriam ser recusadas")
	}

	planos, err := svc.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(planos) != 0 {
		t.Fatalf("nada podia ter sido gravado, veio %d plano(s)", len(planos))
	}
}

// CAN4 — nome de aba com script sobrevive como texto, sem virar markup.
func TestSavePlanCAN4TabNameWithScriptStaysText(t *testing.T) {
	svc := newPlanningTestService(t)

	const bruto = `<script>alert(1)</script>`
	salvo, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Injeção",
		Tabs: []PaintingTabDTO{{Name: bruto}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	lido, err := svc.LoadPlan(salvo.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if lido.Tabs[0].Name != bruto {
		t.Fatalf("o nome tem de voltar literal, veio %q", lido.Tabs[0].Name)
	}
}

// CAN7 — duas gravações simultâneas do mesmo plano deixam um conjunto só de
// abas, sem duplicação: o replace-all roda numa transação.
func TestSavePlanCAN7ConcurrentSavesLeaveOneSetOfTabs(t *testing.T) {
	svc := newPlanningTestService(t)

	base, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Concorrente",
		Tabs: []PaintingTabDTO{{Name: "Frente", Regions: []PaintingRegionDTO{{Hex: "#7A1F2B"}}}},
	})
	if err != nil {
		t.Fatalf("SavePlan inicial: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = svc.SavePlan(PaintingPlanDTO{
				ID:   base.ID,
				Name: "Concorrente",
				Tabs: []PaintingTabDTO{{
					Name:    fmt.Sprintf("Aba %d", n),
					Regions: []PaintingRegionDTO{{Hex: "#1B2A3D"}},
				}},
			})
		}(i)
	}
	wg.Wait()

	lido, err := svc.LoadPlan(base.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if len(lido.Tabs) != 1 {
		t.Fatalf("uma gravação vence: esperado 1 aba, veio %d", len(lido.Tabs))
	}
	if len(lido.Tabs[0].Regions) != 1 {
		t.Fatalf("esperado 1 região na aba vencedora, veio %d", len(lido.Tabs[0].Regions))
	}
}

// ========== preserva: nome de região > 100 caracteres é recusado citando a aba ==========
func TestSavePlanRegionNameTooLongRefusedCitingTab(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.SavePlan(PaintingPlanDTO{
		Name: "Nome de Região Longo",
		Tabs: []PaintingTabDTO{{
			Name: "Detalhe",
			Regions: []PaintingRegionDTO{
				{X: 0, Y: 0, Hex: "#000000", RegionName: strings.Repeat("a", maxRegionNameChars+1)},
			},
		}},
	})
	if err == nil {
		t.Fatal("esperava erro para nome de região com 101 caracteres")
	}
	if !strings.Contains(err.Error(), "Detalhe") || !strings.Contains(err.Error(), fmt.Sprint(maxRegionNameChars)) {
		t.Fatalf("erro deveria citar a aba 'Detalhe' e o limite de %d: %v", maxRegionNameChars, err)
	}
}

// ========== preserva: nota de região > 2000 caracteres é recusada citando a aba ==========
func TestSavePlanRegionNoteTooLongRefusedCitingTab(t *testing.T) {
	s := newPlanningTestService(t)

	_, err := s.SavePlan(PaintingPlanDTO{
		Name: "Nota Longa",
		Tabs: []PaintingTabDTO{{
			Name: "Detalhe",
			Regions: []PaintingRegionDTO{
				{X: 0, Y: 0, Hex: "#000000", Note: strings.Repeat("a", maxNoteChars+1)},
			},
		}},
	})
	if err == nil {
		t.Fatal("esperava erro para nota com 2001 caracteres")
	}
	if !strings.Contains(err.Error(), "Detalhe") || !strings.Contains(err.Error(), fmt.Sprint(maxNoteChars)) {
		t.Fatalf("erro deveria citar a aba 'Detalhe' e o limite de %d: %v", maxNoteChars, err)
	}
}

// ========== preserva: ListPlans devolve lista vazia sem erro quando não há planos ==========
func TestListPlansEmptyReturnsNoError(t *testing.T) {
	s := newPlanningTestService(t)

	list, err := s.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("esperava lista vazia, veio %d", len(list))
	}
}

// A migração é no-op numa base que já nasceu no formato de abas: nada a
// converter, nenhuma aba criada.
func TestMigratePlanningTabsNoOpEmBaseNova(t *testing.T) {
	svc := newPlanningTestService(t)

	if err := migratePlanningTabsIfNeeded(svc.db); err != nil {
		t.Fatalf("migração em base nova deveria ser no-op: %v", err)
	}
	var abas int
	if err := svc.db.QueryRow("SELECT COUNT(*) FROM painting_tabs").Scan(&abas); err != nil {
		t.Fatalf("contando abas: %v", err)
	}
	if abas != 0 {
		t.Fatalf("base nova não podia ganhar aba, veio %d", abas)
	}
}

// columnExists responde falso para tabela inexistente em vez de estourar.
func TestColumnExistsTabelaInexistente(t *testing.T) {
	svc := newPlanningTestService(t)

	// PRAGMA table_info de tabela inexistente devolve conjunto VAZIO, não erro:
	// a resposta certa é "não tem a coluna", sem estourar.
	existe, err := columnExists(svc.db, "tabela_que_nao_existe", "qualquer")
	if err != nil {
		t.Fatalf("tabela inexistente não deveria virar erro: %v", err)
	}
	if existe {
		t.Fatal("tabela inexistente não pode reportar coluna existente")
	}
	ok, err := columnExists(svc.db, "painting_tabs", "coluna_que_nao_existe")
	if err != nil {
		t.Fatalf("columnExists: %v", err)
	}
	if ok {
		t.Fatal("coluna inexistente não pode ser reportada como existente")
	}
}

// addColumnIfMissing é idempotente: chamada duas vezes não falha nem duplica.
func TestAddColumnIfMissingIdempotente(t *testing.T) {
	svc := newPlanningTestService(t)

	for i := 0; i < 2; i++ {
		if err := addColumnIfMissing(svc.db, "painting_tabs", "coluna_de_teste", "TEXT"); err != nil {
			t.Fatalf("chamada %d: %v", i+1, err)
		}
	}
	ok, err := columnExists(svc.db, "painting_tabs", "coluna_de_teste")
	if err != nil || !ok {
		t.Fatalf("coluna deveria existir depois das duas chamadas (ok=%v err=%v)", ok, err)
	}
}

// DeletePlan apaga os três níveis de uma vez, sem depender de cascata.
func TestDeletePlanApagaAbasERegioes(t *testing.T) {
	svc := newPlanningTestService(t)

	salvo, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Para apagar",
		Tabs: []PaintingTabDTO{
			{Name: "Frente", Regions: []PaintingRegionDTO{{Hex: "#7A1F2B"}, {Hex: "#C8B15A"}}},
			{Name: "Costas", Regions: []PaintingRegionDTO{{Hex: "#1B2A3D"}}},
		},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := svc.DeletePlan(salvo.ID); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}

	for _, q := range []struct {
		nome string
		sql  string
	}{
		{"abas", "SELECT COUNT(*) FROM painting_tabs WHERE plan_id = ?"},
		{"regiões", `SELECT COUNT(*) FROM painting_regions WHERE tab_id IN
			(SELECT id FROM painting_tabs WHERE plan_id = ?)`},
	} {
		var n int
		if err := svc.db.QueryRow(q.sql, salvo.ID).Scan(&n); err != nil {
			t.Fatalf("contando %s: %v", q.nome, err)
		}
		if n != 0 {
			t.Fatalf("%s órfã(s) depois do DeletePlan: %d", q.nome, n)
		}
	}
}

// Banco fechado: as leituras devolvem erro em vez de estourar. Cobre os ramos
// de falha de query que nenhum caminho feliz alcança.
func TestConsultasComBancoFechadoDevolvemErro(t *testing.T) {
	svc := newPlanningTestService(t)
	if _, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Antes de fechar",
		Tabs: []PaintingTabDTO{{Name: "Frente"}},
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := svc.db.Close(); err != nil {
		t.Fatalf("fechando banco: %v", err)
	}

	if _, err := svc.ListPlans(); err == nil {
		t.Error("ListPlans com banco fechado deveria devolver erro")
	}
	if _, err := svc.LoadPlan(1); err == nil {
		t.Error("LoadPlan com banco fechado deveria devolver erro")
	}
	if err := svc.DeletePlan(1); err == nil {
		t.Error("DeletePlan com banco fechado deveria devolver erro")
	}
	if _, err := svc.SavePlan(PaintingPlanDTO{Name: "x", Tabs: []PaintingTabDTO{{Name: "y"}}}); err == nil {
		t.Error("SavePlan com banco fechado deveria devolver erro")
	}
}
