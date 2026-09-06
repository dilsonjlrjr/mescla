package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// newManufacturersTestService monta o mínimo pra exercitar AddManufacturer/
// DeleteManufacturer: manufacturers + paints (catálogo seedado, protege
// contra exclusão) + user_paints (o que deve cascatear).
func newManufacturersTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE manufacturers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
		CREATE TABLE paints (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL);
		INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo');
		INSERT INTO paints (id, manufacturer_id) VALUES (100, 1);
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := ensureUserSchema(db); err != nil {
		t.Fatalf("ensureUserSchema: %v", err)
	}
	return &PaintService{db: db}
}

func TestAddManufacturerRejectsEmptyAndDuplicate(t *testing.T) {
	s := newManufacturersTestService(t)

	if _, err := s.AddManufacturer(""); err == nil {
		t.Fatal("esperava erro para nome vazio")
	}
	if _, err := s.AddManufacturer("Vallejo"); err == nil {
		t.Fatal("esperava erro para nome duplicado")
	}

	got, err := s.AddManufacturer("Mr. Color")
	if err != nil {
		t.Fatalf("add válido falhou: %v", err)
	}
	if got.ID == 0 || got.Name != "Mr. Color" {
		t.Fatalf("add não resolveu campos: %+v", got)
	}
}

func TestDeleteManufacturerRefusesWhenCatalogHasPaints(t *testing.T) {
	s := newManufacturersTestService(t)

	if _, err := s.DeleteManufacturer(1); err == nil {
		t.Fatal("esperava recusa: fabricante 1 tem tinta no catálogo seedado")
	}
}

func TestDeleteManufacturerCascadesUserPaints(t *testing.T) {
	s := newManufacturersTestService(t)

	mfr, err := s.AddManufacturer("Sem Catálogo")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := s.AddUserPaint(UserPaintDTO{ManufacturerID: mfr.ID, Name: "Minha tinta", R: 1, G: 2, B: 3}); err != nil {
		t.Fatalf("add user paint: %v", err)
	}

	result, err := s.DeleteManufacturer(mfr.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if result.RemovedUserPaints != 1 {
		t.Fatalf("esperava 1 tinta removida em cascata, veio %d", result.RemovedUserPaints)
	}

	list, _ := s.GetUserPaints()
	for _, p := range list {
		if p.ManufacturerID == mfr.ID {
			t.Fatal("tinta do fabricante excluído ainda presente no estoque")
		}
	}
}
