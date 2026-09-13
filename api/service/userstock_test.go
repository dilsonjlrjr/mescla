package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// newStockTestService monta um PaintService sobre um banco em memória com o
// mínimo necessário (manufacturers + user_paints) para exercitar o CRUD e a
// importação de CSV do estoque, sem depender do catálogo completo.
func newStockTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE manufacturers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
		CREATE TABLE paint_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
		CREATE TABLE paints (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL REFERENCES manufacturers(id));
		INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo'), (2, 'Citadel');
	`); err != nil {
		t.Fatalf("seed manufacturers: %v", err)
	}
	if err := ensureUserSchema(db); err != nil {
		t.Fatalf("ensureUserSchema: %v", err)
	}
	return &PaintService{db: db}
}

func TestAddUserPaintValidatesManufacturer(t *testing.T) {
	s := newStockTestService(t)

	if _, err := s.AddUserPaint(UserPaintDTO{ManufacturerID: 999, Name: "Fantasma", R: 10}); err == nil {
		t.Fatal("esperava erro para fabricante inexistente")
	}
	if _, err := s.AddUserPaint(UserPaintDTO{ManufacturerID: 1, Name: ""}); err == nil {
		t.Fatal("esperava erro para nome vazio")
	}

	got, err := s.AddUserPaint(UserPaintDTO{ManufacturerID: 1, Name: "Meu Preto", Code: "70.950", R: 28, G: 28, B: 28, Volume: "17ml"})
	if err != nil {
		t.Fatalf("add válido falhou: %v", err)
	}
	if got.ID == 0 || got.Manufacturer != "Vallejo" {
		t.Fatalf("add não resolveu id/marca: %+v", got)
	}
}

func TestUserPaintCRUDRoundTrip(t *testing.T) {
	s := newStockTestService(t)

	p, _ := s.AddUserPaint(UserPaintDTO{ManufacturerID: 1, Name: "Azul", R: 10, G: 20, B: 200})
	p.Name = "Azul Marinho"
	p.ManufacturerID = 2
	if _, err := s.UpdateUserPaint(p); err != nil {
		t.Fatalf("update: %v", err)
	}

	list, err := s.GetUserPaints()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Azul Marinho" || list[0].Manufacturer != "Citadel" {
		t.Fatalf("update não persistiu: %+v", list)
	}

	if err := s.DeleteUserPaint(p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = s.GetUserPaints()
	if len(list) != 0 {
		t.Fatalf("delete não removeu: %+v", list)
	}
	if err := s.DeleteUserPaint(p.ID); err == nil {
		t.Fatal("deletar id inexistente deveria falhar")
	}
}

func TestImportUserPaintsCSV(t *testing.T) {
	s := newStockTestService(t)

	csv := "fabricante,nome,codigo,hex,volume,notas\n" +
		"Vallejo,Preto,70.950,#1c1c1c,17ml,base\n" +
		"MarcaX,Ruim,,#000000,,\n" + // fabricante inexistente → erro
		"Citadel,Vermelho,,#9a1115,,\n"

	res, err := s.ImportUserPaintsCSV(csv)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Imported != 2 {
		t.Fatalf("esperava 2 importadas, veio %d", res.Imported)
	}
	if len(res.Errors) != 1 || res.Errors[0].Line != 3 {
		t.Fatalf("esperava 1 erro na linha 3, veio %+v", res.Errors)
	}

	list, _ := s.GetUserPaints()
	if len(list) != 2 {
		t.Fatalf("esperava 2 tintas no banco, veio %d", len(list))
	}
}
