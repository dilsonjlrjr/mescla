package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// newRF13TestService monta um PaintService com o schema completo exigido por
// GetPaintByID/GetAllPaints (catalogpick_test.go tem o mesmo padrão) e roda
// seedSQL por cima — cada teste traz seu próprio catálogo, isolado, porque
// cada chamada abre um banco :memory: novo.
func newRF13TestService(t *testing.T, seedSQL string) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}

	schema := []string{
		`CREATE TABLE manufacturers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE product_lines (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE paint_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE finish_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE coverage_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE opacity_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE paints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			manufacturer_id INTEGER NOT NULL,
			product_line_id INTEGER,
			code TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			paint_type_id INTEGER,
			finish_type_id INTEGER,
			coverage_type_id INTEGER,
			opacity_type_id INTEGER,
			volume_ml REAL DEFAULT 17,
			thumbnail_path TEXT DEFAULT '',
			image_path TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id)
		)`,
		`CREATE TABLE paint_colors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			paint_id INTEGER NOT NULL UNIQUE,
			rgb_r INTEGER NOT NULL DEFAULT 0,
			rgb_g INTEGER NOT NULL DEFAULT 0,
			rgb_b INTEGER NOT NULL DEFAULT 0,
			swatch_path TEXT DEFAULT '',
			FOREIGN KEY (paint_id) REFERENCES paints(id)
		)`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed schema: %v", err)
		}
	}
	if _, err := db.Exec(seedSQL); err != nil {
		t.Fatalf("seed dados: %v", err)
	}
	if err := ensureUserSchema(db); err != nil {
		t.Fatalf("ensureUserSchema: %v", err)
	}

	return &PaintService{db: db}
}

// CA1 — Misturar marcas (targetManufacturerID 0): a receita traz ingredientes
// de dois fabricantes, crossBrand true, manufacturers com os dois nomes
// ordenados e sem repetição.
// CAN4 — a própria tinta de origem nunca aparece entre os ingredientes.
func TestRF13CA1CrossBrandRealComManufacturerIdEManufacturers(t *testing.T) {
	s := newRF13TestService(t, `
		INSERT INTO manufacturers (id, name) VALUES (1, 'Citadel'), (2, 'Mr. Color'), (3, 'Vallejo');
		INSERT INTO paints (id, manufacturer_id, name, code) VALUES
			(1, 3, 'Rosa Claro',  'V-01'),
			(2, 1, 'Vermelho',    'C-01'),
			(3, 2, 'Branco',      'M-01');
		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES
			(1, 238, 154, 138),
			(2, 216, 30, 5),
			(3, 255, 255, 255);
	`)

	rec, err := s.SuggestEquivalentRecipe(1, 0, 0)
	if err != nil {
		t.Fatalf("SuggestEquivalentRecipe: %v", err)
	}
	if len(rec.Ingredients) < 2 {
		t.Fatalf("esperava mistura de 2 ingredientes, veio %d: %+v", len(rec.Ingredients), rec.Ingredients)
	}

	mfrIDs := map[int64]bool{}
	for _, ing := range rec.Ingredients {
		if ing.PaintID == 1 {
			t.Fatal("CAN4: a tinta de origem não pode aparecer como ingrediente")
		}
		if ing.ManufacturerID == 0 || ing.Manufacturer == "" {
			t.Fatalf("ingrediente sem manufacturerId/manufacturer: %+v", ing)
		}
		mfrIDs[ing.ManufacturerID] = true
	}
	if len(mfrIDs) < 2 {
		t.Fatalf("esperava ingredientes de 2 fabricantes distintos, veio %v", mfrIDs)
	}
	if !rec.CrossBrand {
		t.Fatal("crossBrand deveria ser true")
	}
	want := []string{"Citadel", "Mr. Color"}
	got := append([]string(nil), rec.Manufacturers...)
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("manufacturers = %v, esperado %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("manufacturers = %v, esperado %v (ordenado, sem repetição)", got, want)
		}
	}
	if !sort.StringsAreSorted(rec.Manufacturers) {
		t.Fatalf("manufacturers não veio ordenado alfabeticamente: %v", rec.Manufacturers)
	}
}

// CA4 — maxIngredients=3 limita a receita a no máximo 3 ingredientes, mesmo
// quando o catálogo permitiria mais (calibrado: sem teto essa entrada dá 7).
func TestRF13CA4MaxIngredientsLimitaAReceita(t *testing.T) {
	var b strings.Builder
	b.WriteString("INSERT INTO manufacturers (id, name) VALUES (1, 'Origem'), (2, 'Catalogo');\n")
	b.WriteString("INSERT INTO paints (id, manufacturer_id, name, code) VALUES (100, 1, 'Origem', 'O-01');\n")
	b.WriteString("INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES (100, 120, 90, 140);\n")
	for i := 0; i < 12; i++ {
		id, r, g, bl := i+1, (i*20)%256, (i*40)%256, (i*60)%256
		fmt.Fprintf(&b, "INSERT INTO paints (id, manufacturer_id, name, code) VALUES (%d, 2, 'Candidato %d', '');\n", id, id)
		fmt.Fprintf(&b, "INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES (%d, %d, %d, %d);\n", id, r, g, bl)
	}

	s := newRF13TestService(t, b.String())

	uncapped, err := s.SuggestEquivalentRecipe(100, 0, 0)
	if err != nil {
		t.Fatalf("SuggestEquivalentRecipe(sem teto): %v", err)
	}
	if len(uncapped.Ingredients) <= 3 {
		t.Skipf("cenário calibrado esperava >3 sem teto, veio %d — recalibrar", len(uncapped.Ingredients))
	}

	capped, err := s.SuggestEquivalentRecipe(100, 0, 3)
	if err != nil {
		t.Fatalf("SuggestEquivalentRecipe(maxIngredients=3): %v", err)
	}
	if len(capped.Ingredients) > 3 {
		t.Fatalf("maxIngredients=3 desrespeitado: veio %d ingredientes", len(capped.Ingredients))
	}
}

// CAN3 — sourcePaintId inexistente ou de tinta sem cor cadastrada dá 400 com
// mensagem específica, nunca receita vazia.
func TestRF13CAN3SourcePaintIdInvalido(t *testing.T) {
	s := newRF13TestService(t, `
		INSERT INTO manufacturers (id, name) VALUES (1, 'Citadel');
		INSERT INTO paints (id, manufacturer_id, name, code) VALUES (1, 1, 'Sem Cor', 'S-01');
	`)

	if _, err := s.SuggestEquivalentRecipe(999, 0, 0); err == nil {
		t.Fatal("esperava erro para sourcePaintId inexistente")
	} else if !strings.Contains(err.Error(), "tinta de origem não encontrada") {
		t.Fatalf("mensagem inesperada: %v", err)
	}

	if _, err := s.SuggestEquivalentRecipe(1, 0, 0); err == nil {
		t.Fatal("esperava erro para tinta sem cor cadastrada")
	} else if !strings.Contains(err.Error(), "tinta de origem não possui dados de cor cadastrados") {
		t.Fatalf("mensagem inesperada: %v", err)
	}
}

// CAN6 — o ingrediente serializado tem exatamente os campos declarados na
// spec (paintId, name, code, percentage, r, g, b, manufacturerId,
// manufacturer), nenhum campo interno de paints a mais.
func TestRF13CAN6IngredienteSerializaSoOsCamposDeclarados(t *testing.T) {
	ing := RecipeIngredientDTO{
		PaintID:        1,
		Name:           "Vermelho",
		Code:           "C-01",
		Percentage:     60,
		R:              200,
		G:              10,
		B:              10,
		ManufacturerID: 1,
		Manufacturer:   "Citadel",
	}
	body, err := json.Marshal(ing)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	esperado := map[string]bool{
		"paintId": true, "name": true, "code": true, "percentage": true,
		"r": true, "g": true, "b": true, "manufacturerId": true, "manufacturer": true,
	}
	if len(m) != len(esperado) {
		t.Fatalf("campos = %v, esperado exatamente %v", keys(m), keys(esperado))
	}
	for k := range m {
		if !esperado[k] {
			t.Errorf("campo inesperado no JSON do ingrediente: %s", k)
		}
	}
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
