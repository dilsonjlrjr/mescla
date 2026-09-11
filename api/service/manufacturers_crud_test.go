package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// newManufacturersTestService monta o mínimo que o CRUD de fabricante toca:
// catálogo (paints, product_lines), estoque do servidor (user_paints) e as
// tabelas que prendem o fabricante por uso (recipes, resources).
func newManufacturersTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE,
			country TEXT, website TEXT, logo_path TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE product_lines (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL REFERENCES manufacturers(id), name TEXT NOT NULL);
		CREATE TABLE paints (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL REFERENCES manufacturers(id), product_line_id INTEGER REFERENCES product_lines(id));
		CREATE TABLE recipes (id INTEGER PRIMARY KEY AUTOINCREMENT, source_manufacturer_id INTEGER REFERENCES manufacturers(id), target_manufacturer_id INTEGER REFERENCES manufacturers(id));
		CREATE TABLE resources (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER REFERENCES manufacturers(id));
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

func mustAddManufacturer(t *testing.T, s *PaintService, name string) ManufacturerDTO {
	t.Helper()
	m, err := s.AddManufacturer(name)
	if err != nil {
		t.Fatalf("add %q: %v", name, err)
	}
	return m
}

func TestAddManufacturerValidatesName(t *testing.T) {
	s := newManufacturersTestService(t)

	var nameErr *NameError
	for _, bad := range []string{"", "   ", strings.Repeat("á", 81)} {
		if _, err := s.AddManufacturer(bad); !errors.As(err, &nameErr) {
			t.Fatalf("nome %q: esperava NameError, veio %v", bad, err)
		}
	}
	if _, err := s.AddManufacturer("vallejo"); !errors.Is(err, ErrManufacturerDuplicate) {
		t.Fatalf("esperava duplicata sem distinguir caixa, veio %v", err)
	}

	got := mustAddManufacturer(t, s, "  Pro Acryl  ")
	if got.ID == 0 || got.Name != "Pro Acryl" || got.PaintCount != 0 || got.UserPaintCount != 0 {
		t.Fatalf("add não resolveu campos: %+v", got)
	}

	mustAddManufacturer(t, s, "Açaí")
	if _, err := s.AddManufacturer("AÇAÍ"); !errors.Is(err, ErrManufacturerDuplicate) {
		t.Fatalf("esperava duplicata fora do ASCII, veio %v", err)
	}

	// Caractere invisível não pode criar um segundo "Vallejo" (guardrail rf-14).
	zwsp, bom := string(rune(0x200b)), string(rune(0xfeff))
	for _, hidden := range []string{"Vallejo" + zwsp, bom + "Vallejo"} {
		if _, err := s.AddManufacturer(hidden); !errors.Is(err, ErrManufacturerDuplicate) {
			t.Fatalf("nome %q: esperava duplicata, veio %v", hidden, err)
		}
	}
	if got := mustAddManufacturer(t, s, "Novo"+zwsp); got.Name != "Novo" {
		t.Fatalf("invisível deveria sair do nome gravado: %q", got.Name)
	}

	injection := "x'); DROP TABLE manufacturers;--"
	if got := mustAddManufacturer(t, s, injection); got.Name != injection {
		t.Fatalf("nome não gravado literal: %q", got.Name)
	}
	if _, err := s.GetManufacturers(); err != nil {
		t.Fatalf("tabela deveria continuar íntegra: %v", err)
	}
}

func TestUpdateManufacturer(t *testing.T) {
	s := newManufacturersTestService(t)
	pro := mustAddManufacturer(t, s, "Pro Acryl")

	got, err := s.UpdateManufacturer(pro.ID, "Pro Acryl Signature")
	if err != nil || got.ID != pro.ID || got.Name != "Pro Acryl Signature" {
		t.Fatalf("rename: %+v, %v", got, err)
	}
	if _, err := s.UpdateManufacturer(pro.ID, "VALLEJO"); !errors.Is(err, ErrManufacturerDuplicate) {
		t.Fatalf("esperava duplicata, veio %v", err)
	}
	if got, err := s.UpdateManufacturer(pro.ID, "pro acryl signature"); err != nil || got.Name != "pro acryl signature" {
		t.Fatalf("troca só de caixa deveria passar: %+v, %v", got, err)
	}
	if _, err := s.UpdateManufacturer(999, "Qualquer"); !errors.Is(err, ErrManufacturerNotFound) {
		t.Fatalf("esperava não encontrado, veio %v", err)
	}
}

func TestDeleteManufacturerRefusesWhenCatalogHasPaints(t *testing.T) {
	s := newManufacturersTestService(t)

	var paintsErr *ManufacturerHasPaintsError
	if err := s.DeleteManufacturer(1); !errors.As(err, &paintsErr) || paintsErr.Catalog != 1 || paintsErr.Stock != 0 {
		t.Fatalf("esperava recusa com 1 tinta no catálogo, veio %v", err)
	}
}

func TestDeleteManufacturerRefusesWhenStockHasPaints(t *testing.T) {
	s := newManufacturersTestService(t)
	mfr := mustAddManufacturer(t, s, "Sem Catálogo")
	if _, err := s.AddUserPaint(UserPaintDTO{ManufacturerID: mfr.ID, Name: "Minha tinta", R: 1, G: 2, B: 3}); err != nil {
		t.Fatalf("add user paint: %v", err)
	}

	var paintsErr *ManufacturerHasPaintsError
	if err := s.DeleteManufacturer(mfr.ID); !errors.As(err, &paintsErr) || paintsErr.Stock != 1 {
		t.Fatalf("esperava recusa com 1 tinta no estoque, veio %v", err)
	}
	list, err := s.GetUserPaints()
	if err != nil || len(list) != 1 {
		t.Fatalf("a tinta do estoque deveria continuar: %d, %v", len(list), err)
	}
	mfrs, err := s.GetManufacturers()
	if err != nil {
		t.Fatalf("get manufacturers: %v", err)
	}
	for _, m := range mfrs {
		if m.ID == mfr.ID && m.UserPaintCount != 1 {
			t.Fatalf("userPaintCount esperado 1, veio %d", m.UserPaintCount)
		}
	}
}

func TestDeleteManufacturerRefusesWhenInUse(t *testing.T) {
	s := newManufacturersTestService(t)
	mfr := mustAddManufacturer(t, s, "Usado em Receita")
	if _, err := s.db.Exec(`INSERT INTO recipes (target_manufacturer_id) VALUES (?)`, mfr.ID); err != nil {
		t.Fatalf("seed recipe: %v", err)
	}
	if err := s.DeleteManufacturer(mfr.ID); !errors.Is(err, ErrManufacturerInUse) {
		t.Fatalf("esperava em uso, veio %v", err)
	}
}

func TestDeleteManufacturerRemovesEmptyLinesOnce(t *testing.T) {
	s := newManufacturersTestService(t)
	mfr := mustAddManufacturer(t, s, "Vazio")
	if _, err := s.db.Exec(`INSERT INTO product_lines (manufacturer_id, name) VALUES (?, 'Geral')`, mfr.ID); err != nil {
		t.Fatalf("seed line: %v", err)
	}

	if err := s.DeleteManufacturer(mfr.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var lines int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM product_lines WHERE manufacturer_id = ?`, mfr.ID).Scan(&lines); err != nil || lines != 0 {
		t.Fatalf("linha vazia deveria sair junto: %d, %v", lines, err)
	}
	if err := s.DeleteManufacturer(mfr.ID); !errors.Is(err, ErrManufacturerNotFound) {
		t.Fatalf("segunda exclusão deveria dar não encontrado, veio %v", err)
	}
}
