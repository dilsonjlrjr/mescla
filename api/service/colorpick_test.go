package service

import (
	"database/sql"
	"strings"
	"testing"

	"paint-match-ai/api/domain/similarity"

	_ "modernc.org/sqlite"
)

func newColorPickTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}

	// Schema mínimo (compatível com similarity.Engine.FindSimilar e GetPaintByID)
	for _, stmt := range []string{
		`CREATE TABLE manufacturers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`INSERT INTO manufacturers (id, name) VALUES (1, 'Vallejo'), (2, 'Citadel')`,
		`CREATE TABLE product_lines (id INTEGER PRIMARY KEY AUTOINCREMENT, manufacturer_id INTEGER NOT NULL, name TEXT NOT NULL)`,
		`INSERT INTO product_lines (id, manufacturer_id, name) VALUES (1, 1, 'Model Air'), (2, 2, 'Base')`,
		`CREATE TABLE paint_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`INSERT INTO paint_types (id, name) VALUES (1, 'Acrylic')`,
		`CREATE TABLE finish_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`INSERT INTO finish_types (id, name) VALUES (1, 'Matte')`,
		`CREATE TABLE coverage_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`INSERT INTO coverage_types (id, name) VALUES (1, 'Opaque')`,
		`CREATE TABLE opacity_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE)`,
		`INSERT INTO opacity_types (id, name) VALUES (1, 'Opaque')`,
		`CREATE TABLE paints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			manufacturer_id INTEGER NOT NULL,
			product_line_id INTEGER NOT NULL DEFAULT 1,
			code TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			paint_type_id INTEGER DEFAULT 1,
			finish_type_id INTEGER DEFAULT 1,
			coverage_type_id INTEGER DEFAULT 1,
			opacity_type_id INTEGER DEFAULT 1,
			volume_ml REAL DEFAULT 17,
			thumbnail_path TEXT DEFAULT '',
			image_path TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id),
			FOREIGN KEY (product_line_id) REFERENCES product_lines(id)
		)`,
		`CREATE TABLE paint_colors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			paint_id INTEGER NOT NULL UNIQUE,
			rgb_r INTEGER NOT NULL DEFAULT 0,
			rgb_g INTEGER NOT NULL DEFAULT 0,
			rgb_b INTEGER NOT NULL DEFAULT 0,
			hsv_h REAL DEFAULT 0, hsv_s REAL DEFAULT 0, hsv_v REAL DEFAULT 0,
			hsl_h REAL DEFAULT 0, hsl_s REAL DEFAULT 0, hsl_l REAL DEFAULT 0,
			lab_l REAL DEFAULT 0, lab_a REAL DEFAULT 0, lab_b REAL DEFAULT 0,
			lch_l REAL DEFAULT 0, lch_c REAL DEFAULT 0, lch_h REAL DEFAULT 0,
			swatch_path TEXT DEFAULT '',
			FOREIGN KEY (paint_id) REFERENCES paints(id)
		)`,
		// Tintas de exemplo
		`INSERT INTO paints (id, manufacturer_id, product_line_id, code, name) VALUES
			(1, 1, 1, '70.012', 'Scarlet Red'),
			(2, 1, 1, '70.022', 'Ultramarine Blue'),
			(3, 1, 1, '70.006', 'Sun Yellow'),
			(4, 1, 1, '70.001', 'White'),
			(5, 1, 1, '70.002', 'Black'),
			(6, 2, 2, 'Base', 'Abaddon Black'),
			(7, 2, 2, 'Base', 'Mephiston Red'),
			(8, 2, 2, 'Base', 'Macragge Blue'),
			(9, 2, 2, 'Layer', 'Flash Gitz Yellow'),
			(10, 2, 2, 'Layer', 'White Scar')`,
		`INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES
			(1, 228, 45, 45),
			(2, 51, 84, 141),
			(3, 252, 209, 22),
			(4, 255, 255, 255),
			(5, 0, 0, 0),
			(6, 0, 0, 0),
			(7, 153, 12, 12),
			(8, 13, 64, 119),
			(9, 255, 243, 0),
			(10, 255, 255, 255)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	return &PaintService{
		db:  db,
		sim: similarity.NewEngine(db),
	}
}

// ========== CA1: PickColor(R:255, G:0, B:0) → retorna Hex e matches ==========
func TestPickColorReturnsHexAndMatches(t *testing.T) {
	s := newColorPickTestService(t)

	resp, err := s.PickColor(255, 0, 0, 0)
	if err != nil {
		t.Fatalf("PickColor: %v", err)
	}
	if resp.Hex != "#FF0000" {
		t.Fatalf("hex esperado #FF0000, veio %s", resp.Hex)
	}
	if len(resp.Matches) == 0 {
		t.Fatal("esperava pelo menos 1 match")
	}
}

// ========== CA2: Match mais próximo tem ΔE ≤ distância de qualquer outro ==========
func TestPickColorMatchesSorted(t *testing.T) {
	s := newColorPickTestService(t)

	resp, err := s.PickColor(128, 128, 128, 0)
	if err != nil {
		t.Fatalf("PickColor: %v", err)
	}
	if len(resp.Matches) < 2 {
		t.Skip("precisa de pelo menos 2 matches para testar ordenação")
	}
	for i := 1; i < len(resp.Matches); i++ {
		if resp.Matches[i-1].DeltaE > resp.Matches[i].DeltaE {
			t.Fatalf("matches não ordenados: [%d] ΔE=%.2f > [%d] ΔE=%.2f",
				i-1, resp.Matches[i-1].DeltaE, i, resp.Matches[i].DeltaE)
		}
	}
}

// ========== CA3: PickColor com TargetManufacturerID → retorna Recipe ==========
func TestPickColorWithRecipe(t *testing.T) {
	s := newColorPickTestService(t)

	// Testa receita diretamente
	recipe, err := s.SuggestEquivalentRecipe(1, 2)
	if err != nil {
		t.Fatalf("SuggestEquivalentRecipe(1,2): %v", err)
	}
	if recipe.TargetManufacturer == "" {
		t.Fatal("Recipe deveria ter TargetManufacturer preenchido")
	}
	if len(recipe.Ingredients) == 0 {
		t.Fatal("Recipe deveria ter ingredientes")
	}

	// Agora testa via PickColor
	resp, err := s.PickColor(200, 30, 30, 2)
	if err != nil {
		t.Fatalf("PickColor com receita: %v", err)
	}
	if resp.Recipe == nil {
		t.Fatal("esperava Recipe quando TargetManufacturerID informado")
	}
}

// ========== CA4: TargetManufacturerID inválido → erro ==========
func TestPickColorInvalidManufacturer(t *testing.T) {
	s := newColorPickTestService(t)

	_, err := s.PickColor(128, 128, 128, 99999)
	if err == nil {
		t.Fatal("esperava erro para fabricante inexistente")
	}
	if !strings.Contains(err.Error(), "fabricante") {
		t.Fatalf("erro deveria mencionar fabricante: %v", err)
	}
}

// ========== CA5: RGB idêntico a tinta do catálogo → ΔE ≈ 0 ==========
func TestPickColorExactMatch(t *testing.T) {
	s := newColorPickTestService(t)

	// Preto puro deve casar com Abaddon Black ou similar
	resp, err := s.PickColor(0, 0, 0, 0)
	if err != nil {
		t.Fatalf("PickColor: %v", err)
	}
	if len(resp.Matches) == 0 {
		t.Fatal("esperava match para preto")
	}
	if resp.Matches[0].DeltaE > 5.0 {
		t.Fatalf("preto deveria ter ΔE próximo de 0, veio %.2f", resp.Matches[0].DeltaE)
	}
}

// ========== CAN1: Validação de RGB é feita pelo type system (uint8) ==========
func TestPickColorTypeSafety(t *testing.T) {
	// O type system de Go (uint8) garante 0-255 em tempo de compilação.
	// Não é possível passar valor > 255 ou < 0 para PickColor.
	// Este teste documenta que a validação é estrutural, não em runtime.
	s := newColorPickTestService(t)
	resp, err := s.PickColor(255, 255, 255, 0)
	if err != nil {
		t.Fatalf("PickColor com valores válidos: %v", err)
	}
	if resp.Hex != "#FFFFFF" {
		t.Fatalf("hex esperado #FFFFFF, veio %s", resp.Hex)
	}
	_ = s
}
