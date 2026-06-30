package seeds

import (
	"database/sql"
	"fmt"
	"log"
)

// LookupData contém os dados para tabelas de lookup
type LookupData struct {
	PaintTypes   []string
	FinishTypes  []string
	CoverageTypes []string
	OpacityTypes  []string
}

// ProductLineData representa uma linha de produto
type ProductLineData struct {
	Manufacturer string
	Name         string
}

// PaintSeed representa uma tinta para importação
type PaintSeed struct {
	Manufacturer string
	Line         string
	Code         string
	Name         string
	PaintType    string
	Finish       string
	Coverage     string
	Opacity      string
	VolumeML     float64
}

// defaultLookupData retorna os dados padrão para tabelas de lookup
func defaultLookupData() LookupData {
	return LookupData{
		PaintTypes:   []string{"Acrylic", "Enamel", "Lacquer", "Oil", "Watercolor"},
		FinishTypes:  []string{"Matte", "Gloss", "Satin", "Metallic", "Fluorescent"},
		CoverageTypes: []string{"Opaque", "Semi-opaque", "Transparent", "Semi-transparent"},
		OpacityTypes:  []string{"High", "Medium", "Low"},
	}
}

// defaultProductLines retorna as linhas de produto padrão
func defaultProductLines() []ProductLineData {
	return []ProductLineData{
		// AK Interactive
		{Manufacturer: "AK Interactive", Name: "3GEN"},
		{Manufacturer: "AK Interactive", Name: "Real Colors"},
		{Manufacturer: "AK Interactive", Name: "Figure Series"},
		{Manufacturer: "AK Interactive", Name: "Interactive"},
		// Vallejo
		{Manufacturer: "Vallejo", Name: "Model Color"},
		{Manufacturer: "Vallejo", Name: "Game Color"},
		{Manufacturer: "Vallejo", Name: "Model Air"},
		{Manufacturer: "Vallejo", Name: "Mecha Color"},
		// Citadel
		{Manufacturer: "Citadel", Name: "Base"},
		{Manufacturer: "Citadel", Name: "Layer"},
		{Manufacturer: "Citadel", Name: "Shade"},
		{Manufacturer: "Citadel", Name: "Dry"},
		{Manufacturer: "Citadel", Name: "Technical"},
		// Army Painter
		{Manufacturer: "Army Painter", Name: "Warpaints"},
		{Manufacturer: "Army Painter", Name: "Speedpaints"},
		{Manufacturer: "Army Painter", Name: "Air"},
		// Scale75
		{Manufacturer: "Scale75", Name: "Scalecolor"},
		{Manufacturer: "Scale75", Name: "Fantasy & Games"},
		{Manufacturer: "Scale75", Name: "Instant Colors"},
		// Tamiya
		{Manufacturer: "Tamiya", Name: "Acrylic"},
		{Manufacturer: "Tamiya", Name: "Lacquer"},
		// Mr Hobby
		{Manufacturer: "Mr Hobby", Name: "Aqueous"},
		{Manufacturer: "Mr Hobby", Name: "Color"},
		{Manufacturer: "Mr Hobby", Name: "GX"},
		// Acrilex
		{Manufacturer: "Acrilex", Name: "Acrílica"},
		{Manufacturer: "Acrilex", Name: "Spray"},
		{Manufacturer: "Acrilex", Name: "Esmalte"},
		// Corfix
		{Manufacturer: "Corfix", Name: "Acrílica"},
		{Manufacturer: "Corfix", Name: "Esmalte"},
		{Manufacturer: "Corfix", Name: "Spray"},
		// Talento
		{Manufacturer: "Talento", Name: "Acrílica"},
		{Manufacturer: "Talento", Name: "Esmalte"},
	}
}

// defaultPaints retorna uma amostra de tintas para importação
func defaultPaints() []PaintSeed {
	return []PaintSeed{
		// AK Interactive 3GEN
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-001", Name: "White", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-002", Name: "Black", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-003", Name: "Red", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-004", Name: "Blue", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-005", Name: "Green", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-006", Name: "Yellow", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-007", Name: "Orange", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-008", Name: "Purple", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-009", Name: "Brown", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "AK Interactive", Line: "3GEN", Code: "AK-010", Name: "Grey", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},

		// Vallejo Model Color
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.951", Name: "White", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.950", Name: "Black", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.947", Name: "Red", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.925", Name: "Blue", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.970", Name: "Green", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.948", Name: "Yellow", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.910", Name: "Orange", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.960", Name: "Purple", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "Medium", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.983", Name: "Flat Earth", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Vallejo", Line: "Model Color", Code: "70.990", Name: "Light Grey", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},

		// Citadel Base
		{Manufacturer: "Citadel", Line: "Base", Code: "BC-01", Name: "Corax White", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 12},
		{Manufacturer: "Citadel", Line: "Base", Code: "BC-02", Name: "Abaddon Black", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 12},
		{Manufacturer: "Citadel", Line: "Base", Code: "BC-03", Name: "Mephiston Red", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 12},
		{Manufacturer: "Citadel", Line: "Base", Code: "BC-04", Name: "Macragge Blue", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 12},
		{Manufacturer: "Citadel", Line: "Base", Code: "BC-05", Name: "Waaagh! Flesh", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 12},

		// Army Painter Warpaints
		{Manufacturer: "Army Painter", Line: "Warpaints", Code: "WP-001", Name: "Matt White", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 18},
		{Manufacturer: "Army Painter", Line: "Warpaints", Code: "WP-002", Name: "Matt Black", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 18},
		{Manufacturer: "Army Painter", Line: "Warpaints", Code: "WP-003", Name: "Pure Red", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 18},
		{Manufacturer: "Army Painter", Line: "Warpaints", Code: "WP-004", Name: "Crystal Blue", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 18},
		{Manufacturer: "Army Painter", Line: "Warpaints", Code: "WP-005", Name: "Greenskin", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 18},

		// Scale75 Scalecolor
		{Manufacturer: "Scale75", Line: "Scalecolor", Code: "SC-001", Name: "White", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Scale75", Line: "Scalecolor", Code: "SC-002", Name: "Black", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},
		{Manufacturer: "Scale75", Line: "Scalecolor", Code: "SC-003", Name: "Red", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 17},

		// Tamiya Acrylic
		{Manufacturer: "Tamiya", Line: "Acrylic", Code: "X-1", Name: "Black", PaintType: "Acrylic", Finish: "Gloss", Coverage: "Opaque", Opacity: "High", VolumeML: 10},
		{Manufacturer: "Tamiya", Line: "Acrylic", Code: "X-2", Name: "White", PaintType: "Acrylic", Finish: "Gloss", Coverage: "Opaque", Opacity: "High", VolumeML: 10},
		{Manufacturer: "Tamiya", Line: "Acrylic", Code: "X-3", Name: "Cobalt Blue", PaintType: "Acrylic", Finish: "Gloss", Coverage: "Opaque", Opacity: "High", VolumeML: 10},

		// Mr Hobby Aqueous
		{Manufacturer: "Mr Hobby", Line: "Aqueous", Code: "H-01", Name: "White", PaintType: "Acrylic", Finish: "Gloss", Coverage: "Opaque", Opacity: "High", VolumeML: 10},
		{Manufacturer: "Mr Hobby", Line: "Aqueous", Code: "H-02", Name: "Black", PaintType: "Acrylic", Finish: "Gloss", Coverage: "Opaque", Opacity: "High", VolumeML: 10},

		// Acrilex Acrílica
		{Manufacturer: "Acrilex", Line: "Acrílica", Code: "AC-001", Name: "Branco", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},
		{Manufacturer: "Acrilex", Line: "Acrílica", Code: "AC-002", Name: "Preto", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},
		{Manufacturer: "Acrilex", Line: "Acrílica", Code: "AC-003", Name: "Vermelho", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},

		// Corfix Acrílica
		{Manufacturer: "Corfix", Line: "Acrílica", Code: "CF-001", Name: "Branco", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},
		{Manufacturer: "Corfix", Line: "Acrílica", Code: "CF-002", Name: "Preto", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},

		// Talento Acrílica
		{Manufacturer: "Talento", Line: "Acrílica", Code: "TA-001", Name: "Branco", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},
		{Manufacturer: "Talento", Line: "Acrílica", Code: "TA-002", Name: "Preto", PaintType: "Acrylic", Finish: "Matte", Coverage: "Opaque", Opacity: "High", VolumeML: 20},
	}
}

// SeedPaintTypes popula a tabela paint_types
func SeedPaintTypes(db *sql.DB) error {
	return seedLookupTable(db, "paint_types", defaultLookupData().PaintTypes)
}

// SeedFinishTypes popula a tabela finish_types
func SeedFinishTypes(db *sql.DB) error {
	return seedLookupTable(db, "finish_types", defaultLookupData().FinishTypes)
}

// SeedCoverageTypes popula a tabela coverage_types
func SeedCoverageTypes(db *sql.DB) error {
	return seedLookupTable(db, "coverage_types", defaultLookupData().CoverageTypes)
}

// SeedOpacityTypes popula a tabela opacity_types
func SeedOpacityTypes(db *sql.DB) error {
	return seedLookupTable(db, "opacity_types", defaultLookupData().OpacityTypes)
}

// seedLookupTable popula uma tabela de lookup
func seedLookupTable(db *sql.DB, table string, values []string) error {
	stmt, err := db.Prepare(fmt.Sprintf("INSERT OR IGNORE INTO %s (name) VALUES (?)", table))
	if err != nil {
		return fmt.Errorf("prepare %s: %w", table, err)
	}
	defer stmt.Close()

	for _, v := range values {
		if _, err := stmt.Exec(v); err != nil {
			return fmt.Errorf("inserir %s=%s: %w", table, v, err)
		}
	}

	log.Printf("[seed] %s: %d valores inseridos", table, len(values))
	return nil
}

// SeedProductLines popula a tabela product_lines
func SeedProductLines(db *sql.DB) error {
	lines := defaultProductLines()
	inserted := 0

	for _, line := range lines {
		_, err := db.Exec(`
			INSERT INTO product_lines (manufacturer_id, name)
			SELECT id, ? FROM manufacturers WHERE name = ?
			ON CONFLICT DO NOTHING
		`, line.Name, line.Manufacturer)
		if err != nil {
			return fmt.Errorf("inserir linha %s/%s: %w", line.Manufacturer, line.Name, err)
		}
		inserted++
	}

	log.Printf("[seed] product_lines: %d linhas processadas", inserted)
	return nil
}

// SeedPaints popula a tabela paints
func SeedPaints(db *sql.DB) error {
	paints := defaultPaints()
	inserted := 0
	updated := 0

	for _, p := range paints {
		// Verificar se já existe
		var exists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM paints p
			JOIN manufacturers m ON p.manufacturer_id = m.id
			JOIN product_lines pl ON p.product_line_id = pl.id
			WHERE m.name = ? AND pl.name = ? AND p.code = ?
		`, p.Manufacturer, p.Line, p.Code).Scan(&exists)
		if err != nil {
			return fmt.Errorf("verificar existência de %s/%s/%s: %w", p.Manufacturer, p.Line, p.Code, err)
		}

		_, err = db.Exec(`
			INSERT INTO paints (manufacturer_id, product_line_id, code, name, paint_type_id, finish_type_id, coverage_type_id, opacity_type_id, volume_ml)
			SELECT m.id, pl.id, ?, ?, pt.id, ft.id, ct.id, ot.id, ?
			FROM manufacturers m
			JOIN product_lines pl ON pl.manufacturer_id = m.id AND pl.name = ?
			LEFT JOIN paint_types pt ON pt.name = ?
			LEFT JOIN finish_types ft ON ft.name = ?
			LEFT JOIN coverage_types ct ON ct.name = ?
			LEFT JOIN opacity_types ot ON ot.name = ?
			WHERE m.name = ?
			ON CONFLICT(manufacturer_id, product_line_id, code) DO UPDATE SET
				name = excluded.name,
				updated_at = CURRENT_TIMESTAMP
		`, p.Code, p.Name, p.VolumeML, p.Line, p.PaintType, p.Finish, p.Coverage, p.Opacity, p.Manufacturer)
		if err != nil {
			return fmt.Errorf("inserir tinta %s/%s/%s: %w", p.Manufacturer, p.Line, p.Code, err)
		}

		if exists > 0 {
			updated++
		} else {
			inserted++
		}
	}

	log.Printf("[seed] paints: %d inseridas, %d atualizadas", inserted, updated)
	return nil
}

// GetPaintImporterSeed retorna o Seed registro para este pacote
func GetPaintImporterSeed() Seed {
	return Seed{
		Name: "004_paint_importer",
		Fn: func(db *sql.DB) error {
			// Popular tabelas de lookup
			if err := SeedPaintTypes(db); err != nil {
				return err
			}
			if err := SeedFinishTypes(db); err != nil {
				return err
			}
			if err := SeedCoverageTypes(db); err != nil {
				return err
			}
			if err := SeedOpacityTypes(db); err != nil {
				return err
			}

			// Popular linhas de produtos
			if err := SeedProductLines(db); err != nil {
				return err
			}

			// Popular tintas
			return SeedPaints(db)
		},
	}
}
