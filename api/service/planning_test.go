package service

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newPlanningTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	if err := ensurePlanningSchema(db); err != nil {
		t.Fatalf("ensurePlanningSchema: %v", err)
	}
	return &PaintService{db: db}
}

// ========== CA1: SavePlan com nome e 3 regiões → retorna plano com ID, created_at e updated_at ==========
func TestSavePlanCreatesWithTimestamps(t *testing.T) {
	s := newPlanningTestService(t)

	plan := PaintingPlanDTO{
		Name: "Meu Plano",
		Regions: []PaintingRegionDTO{
			{X: 10, Y: 20, R: 255, G: 0, B: 0, Hex: "#FF0000", RegionName: "capa", Note: "base coat"},
			{X: 30, Y: 40, R: 0, G: 255, B: 0, Hex: "#00FF00", RegionName: "ombro"},
			{X: 50, Y: 60, R: 0, G: 0, B: 255, Hex: "#0000FF", RegionName: "peitoral"},
		},
	}

	got, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan falhou: %v", err)
	}
	if got.ID == 0 {
		t.Fatal("esperava ID > 0")
	}
	if got.CreatedAt == "" {
		t.Fatal("esperava CreatedAt preenchido")
	}
	if got.UpdatedAt == "" {
		t.Fatal("esperava UpdatedAt preenchido")
	}
	if got.CreatedAt != got.UpdatedAt {
		t.Fatal("na criação, CreatedAt deve ser igual a UpdatedAt")
	}
}

// ========== CA2: ListPlans após salvar 2 planos → retorna array com 2 itens, cada um com name, created_at, region_count ==========
func TestListPlansAfterSave(t *testing.T) {
	s := newPlanningTestService(t)

	// Salva 2 planos
	s.SavePlan(PaintingPlanDTO{Name: "Plano A", Regions: []PaintingRegionDTO{{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"}}})
	s.SavePlan(PaintingPlanDTO{Name: "Plano B", Regions: []PaintingRegionDTO{
		{X: 1, Y: 1, R: 255, G: 0, B: 0, Hex: "#FF0000"},
		{X: 2, Y: 2, R: 0, G: 255, B: 0, Hex: "#00FF00"},
	}})

	list, err := s.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("esperava 2 planos, veio %d", len(list))
	}
	if list[0].Name != "Plano A" {
		t.Fatalf("primeiro plano deveria ser 'Plano A', veio '%s'", list[0].Name)
	}
	if list[0].RegionCount != 1 {
		t.Fatalf("Plano A deveria ter 1 região, veio %d", list[0].RegionCount)
	}
	if list[1].RegionCount != 2 {
		t.Fatalf("Plano B deveria ter 2 regiões, veio %d", list[1].RegionCount)
	}
	if list[0].ImageData != "" {
		t.Fatal("listagem não deve incluir image_data")
	}
}

// ========== CA3: LoadPlan(id) → retorna plano completo com image_data e array de regiões ==========
func TestLoadPlanComplete(t *testing.T) {
	s := newPlanningTestService(t)

	plan, _ := s.SavePlan(PaintingPlanDTO{
		Name:      "Plano Completo",
		ImageData: "data:image/png;base64,abc123",
		Regions: []PaintingRegionDTO{
			{X: 10, Y: 20, R: 255, G: 0, B: 0, Hex: "#FF0000", RegionName: "capa", Note: "nota 1",
				PaintID: 42, PaintBrand: "Citadel", PaintName: "Mephiston Red", PaintCode: "Base", DeltaE: 1.5},
		},
	})

	got, err := s.LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if got.Name != "Plano Completo" {
		t.Fatalf("nome esperado 'Plano Completo', veio '%s'", got.Name)
	}
	if got.ImageData != "data:image/png;base64,abc123" {
		t.Fatalf("image_data não preservado")
	}
	if len(got.Regions) != 1 {
		t.Fatalf("esperava 1 região, veio %d", len(got.Regions))
	}
	r := got.Regions[0]
	if r.RegionName != "capa" || r.Note != "nota 1" || r.PaintBrand != "Citadel" || r.DeltaE != 1.5 {
		t.Fatalf("dados da região não preservados: %+v", r)
	}
}

// ========== CA4: SavePlan com mesmo ID → atualiza nome, imagem e regiões (antigas substituídas) ==========
func TestSavePlanUpdateReplacesRegions(t *testing.T) {
	s := newPlanningTestService(t)

	plan, _ := s.SavePlan(PaintingPlanDTO{
		Name: "V1",
		Regions: []PaintingRegionDTO{
			{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000", RegionName: "antiga"},
		},
	})

	// Atualiza
	plan.Name = "V2"
	plan.ImageData = "data:image/png;base64,novo"
	plan.Regions = []PaintingRegionDTO{
		{X: 1, Y: 1, R: 255, G: 255, B: 255, Hex: "#FFFFFF", RegionName: "nova1"},
		{X: 2, Y: 2, R: 128, G: 128, B: 128, Hex: "#808080", RegionName: "nova2"},
	}

	updated, err := s.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan (update): %v", err)
	}
	if updated.ID != plan.ID {
		t.Fatal("ID deve ser mantido no update")
	}
	if updated.Name != "V2" {
		t.Fatalf("nome não atualizado: '%s'", updated.Name)
	}

	loaded, _ := s.LoadPlan(plan.ID)
	if len(loaded.Regions) != 2 {
		t.Fatalf("esperava 2 regiões após update, veio %d", len(loaded.Regions))
	}
	if loaded.Regions[0].RegionName != "nova1" {
		t.Fatal("região antiga não foi substituída")
	}
	if loaded.ImageData != "data:image/png;base64,novo" {
		t.Fatal("image_data não foi atualizado")
	}
}

// ========== CA5: DeletePlan(id) → plano e regiões removidos. LoadPlan(id) retorna erro ==========
func TestDeletePlanCascade(t *testing.T) {
	s := newPlanningTestService(t)

	plan, _ := s.SavePlan(PaintingPlanDTO{
		Name: "Para Deletar",
		Regions: []PaintingRegionDTO{
			{X: 0, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"},
			{X: 1, Y: 1, R: 1, G: 1, B: 1, Hex: "#010101"},
		},
	})

	err := s.DeletePlan(plan.ID)
	if err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}

	_, err = s.LoadPlan(plan.ID)
	if err == nil {
		t.Fatal("LoadPlan deveria falhar após delete")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("erro deveria conter 'não encontrado', veio: %v", err)
	}
}

// ========== CA6: Salvar com 51 regiões → erro ==========
func TestSavePlanMaxRegions(t *testing.T) {
	s := newPlanningTestService(t)

	regions := make([]PaintingRegionDTO, 51)
	for i := range regions {
		regions[i] = PaintingRegionDTO{X: i, Y: 0, R: 0, G: 0, B: 0, Hex: "#000000"}
	}

	_, err := s.SavePlan(PaintingPlanDTO{Name: "Muitas", Regions: regions})
	if err == nil {
		t.Fatal("esperava erro para 51 regiões")
	}
	if !strings.Contains(err.Error(), "50") {
		t.Fatalf("erro deveria mencionar limite de 50: %v", err)
	}
}

// ========== CA7: Salvar com image_data > 2MB → erro ==========
func TestSavePlanMaxImageSize(t *testing.T) {
	s := newPlanningTestService(t)

	big := PaintingPlanDTO{
		Name:      "Grande",
		ImageData: strings.Repeat("x", 2*1024*1024+1), // 2MB + 1 byte
		Regions:   []PaintingRegionDTO{},
	}

	_, err := s.SavePlan(big)
	if err == nil {
		t.Fatal("esperava erro para imagem > 2MB")
	}
	if !strings.Contains(err.Error(), "2MB") {
		t.Fatalf("erro deveria mencionar 2MB: %v", err)
	}
}

// ========== CA8: Salvar sem nome → nome gerado automaticamente ==========
func TestSavePlanDefaultName(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{Name: "", Regions: []PaintingRegionDTO{}})
	if err != nil {
		t.Fatalf("SavePlan sem nome: %v", err)
	}
	if plan.Name == "" {
		t.Fatal("esperava nome gerado automaticamente")
	}
	if !strings.HasPrefix(plan.Name, "Plano ") {
		t.Fatalf("nome deveria começar com 'Plano ', veio '%s'", plan.Name)
	}
}

// ========== CAN1: Enviar ID no payload → ignorado ==========
func TestSavePlanIgnoresClientID(t *testing.T) {
	s := newPlanningTestService(t)

	// Tenta forçar ID = 999
	plan, err := s.SavePlan(PaintingPlanDTO{ID: 999, Name: "Hack", Regions: []PaintingRegionDTO{}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if plan.ID == 999 {
		t.Fatal("ID do cliente não deve ser aceito — servidor gera")
	}
}

// ========== CAN2: Enviar created_at no payload → ignorado ==========
func TestSavePlanIgnoresCreatedAt(t *testing.T) {
	s := newPlanningTestService(t)

	plan1, _ := s.SavePlan(PaintingPlanDTO{Name: "Ref1", Regions: []PaintingRegionDTO{}})

	// Tenta sobrescrever created_at
	plan1.Regions = []PaintingRegionDTO{}
	updated, _ := s.SavePlan(plan1)

	// created_at não deve mudar no update
	loaded, _ := s.LoadPlan(updated.ID)
	if loaded.CreatedAt != plan1.CreatedAt {
		t.Fatal("created_at não deve ser alterado pelo cliente")
	}
}

// ========== CAN3: LoadPlan com id inexistente → erro, não panic ==========
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

// ========== CAN4: DeletePlan com id inexistente → erro, sem efeito colateral ==========
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

// ========== Teste adicional: plano sem imagem ==========
func TestSavePlanNoImage(t *testing.T) {
	s := newPlanningTestService(t)

	plan, err := s.SavePlan(PaintingPlanDTO{
		Name:    "Sem Imagem",
		Regions: []PaintingRegionDTO{{X: 0, Y: 0, R: 128, G: 128, B: 128, Hex: "#808080"}},
	})
	if err != nil {
		t.Fatalf("SavePlan sem imagem: %v", err)
	}
	loaded, _ := s.LoadPlan(plan.ID)
	if loaded.ImageData != "" {
		t.Fatal("image_data deveria ser vazio")
	}
}
