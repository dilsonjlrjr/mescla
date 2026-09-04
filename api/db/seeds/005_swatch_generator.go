package seeds

import (
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

// SwatchGenerator gera imagens de swatch para tintas
type SwatchGenerator struct {
	db        *sql.DB
	assetsDir string
	size      int
}

// NewSwatchGenerator cria um novo gerador
func NewSwatchGenerator(db *sql.DB, assetsDir string) *SwatchGenerator {
	return &SwatchGenerator{
		db:        db,
		assetsDir: assetsDir,
		size:      64,
	}
}

// GenerateAll gera swatches para todas as tintas
func (g *SwatchGenerator) GenerateAll() error {
	dir := filepath.Join(g.assetsDir, "swatches")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("criar diretório %s: %w", dir, err)
	}

	rows, err := g.db.Query(`
		SELECT p.id, p.name, pc.rgb_r, pc.rgb_g, pc.rgb_b, m.name as manufacturer
		FROM paints p
		JOIN manufacturers m ON p.manufacturer_id = m.id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		ORDER BY p.id
	`)
	if err != nil {
		return fmt.Errorf("consultar tintas: %w", err)
	}
	defer rows.Close()

	var count, errors int
	for rows.Next() {
		var id int
		var name, manufacturer string
		var rgbR, rgbG, rgbB sql.NullInt64

		if err := rows.Scan(&id, &name, &rgbR, &rgbG, &rgbB, &manufacturer); err != nil {
			return fmt.Errorf("ler tinta: %w", err)
		}

		// Usar cor padrão se não tiver dados de cor
		c := color.RGBA{R: 128, G: 128, B: 128, A: 255}
		if rgbR.Valid && rgbG.Valid && rgbB.Valid {
			c = color.RGBA{R: uint8(rgbR.Int64), G: uint8(rgbG.Int64), B: uint8(rgbB.Int64), A: 255}
		}

		slug := slugify(fmt.Sprintf("%s_%s", manufacturer, name))
		filename := fmt.Sprintf("%d_%s.png", id, slug)
		destPath := filepath.Join(dir, filename)

		if err := g.generateSwatch(destPath, c); err != nil {
			log.Printf("[swatch] Erro gerando swatch para %s: %v", name, err)
			errors++
			continue
		}

		relPath, _ := filepath.Rel(".", destPath)
		_, err := g.db.Exec(`
			INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b, hsv_h, hsv_s, hsv_v, hsl_h, hsl_s, hsl_l, lab_l, lab_a, lab_b, lch_l, lch_c, lch_h, swatch_path, updated_at)
			VALUES (?, ?, ?, ?, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(paint_id) DO UPDATE SET
				swatch_path = excluded.swatch_path,
				updated_at = CURRENT_TIMESTAMP
		`, id, c.R, c.G, c.B, relPath)
		if err != nil {
			return fmt.Errorf("atualizar swatch_path para %s: %w", name, err)
		}

		count++
		if count%10 == 0 {
			log.Printf("[swatch] Progresso: %d swatches gerados", count)
		}
	}

	log.Printf("[swatch] Concluído: %d swatches gerados, %d erros", count, errors)
	return nil
}

// generateSwatch gera uma imagem de swatch
func (g *SwatchGenerator) generateSwatch(path string, c color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, g.size, g.size))

	// Preencher com a cor
	for y := 0; y < g.size; y++ {
		for x := 0; x < g.size; x++ {
			img.Set(x, y, c)
		}
	}

	// Criar arquivo
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("criar arquivo: %w", err)
	}
	defer f.Close()

	return png.Encode(f, img)
}

// GetSwatchGeneratorSeed retorna o Seed registro
func GetSwatchGeneratorSeed() Seed {
	return Seed{
		Name: "005_swatch_generator",
		Fn: func(db *sql.DB) error {
			gen := NewSwatchGenerator(db, "api/assets")
			return gen.GenerateAll()
		},
	}
}
