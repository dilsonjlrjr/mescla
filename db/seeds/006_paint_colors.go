package seeds

import (
	"database/sql"
	"fmt"
	"log"

	"paint-match-ai/pkg/color"
)

// paintColorSeed associa uma tinta (fabricante + código) a uma cor RGB real.
type paintColorSeed struct {
	Manufacturer string
	Code         string
	Hex          string
}

// paintColorData contém os valores RGB reais das tintas de defaultPaints().
//
// Fontes por grupo:
//   - Vallejo (70.xxx), Citadel (BC-01..05) e Army Painter (WP-001..005): cores
//     oficiais/rastreadas via encycolorpedia.com, thepaintingledger.com e o
//     chart de conversão Warpaints (redgrimm.github.io), cruzando os códigos
//     reais de produto.
//   - Tamiya (X-1..3) e Mr Hobby (H-01/H-02): oficiais via Scalemates.com e
//     encycolorpedia.com.
//   - AK Interactive (AK-0xx), Scale75 (SC-0xx), Acrilex, Corfix e Talento: os
//     códigos usados no seed são fictícios (não correspondem a SKUs reais),
//     então foram usados tons realistas de tinta acrílica de hobby para cada
//     nome de cor, em vez de cores puras de monitor.
var paintColorData = []paintColorSeed{
	// AK Interactive 3GEN
	{"AK Interactive", "AK-001", "d8d4cc"},
	{"AK Interactive", "AK-002", "1c1b1a"},
	{"AK Interactive", "AK-003", "a13228"},
	{"AK Interactive", "AK-004", "2d4a6b"},
	{"AK Interactive", "AK-005", "3f5c3a"},
	{"AK Interactive", "AK-006", "d9b03c"},
	{"AK Interactive", "AK-007", "c96a2e"},
	{"AK Interactive", "AK-008", "5c4570"},
	{"AK Interactive", "AK-009", "6b4a35"},
	{"AK Interactive", "AK-010", "8b8983"},

	// Vallejo Model Color
	{"Vallejo", "70.951", "fdf8f3"},
	{"Vallejo", "70.950", "252527"},
	{"Vallejo", "70.947", "a54336"},
	{"Vallejo", "70.925", "222a46"},
	{"Vallejo", "70.970", "2d4b40"},
	{"Vallejo", "70.948", "f3bb6a"},
	{"Vallejo", "70.910", "ca5536"},
	{"Vallejo", "70.960", "3a2f4b"},
	{"Vallejo", "70.983", "634a32"},
	{"Vallejo", "70.990", "72777b"},

	// Citadel Base
	{"Citadel", "BC-01", "f0efe8"},
	{"Citadel", "BC-02", "231f20"},
	{"Citadel", "BC-03", "9a1115"},
	{"Citadel", "BC-04", "0d407f"},
	{"Citadel", "BC-05", "1f5429"},

	// Army Painter Warpaints
	{"Army Painter", "WP-001", "ffffff"},
	{"Army Painter", "WP-002", "231f20"},
	{"Army Painter", "WP-003", "cf2127"},
	{"Army Painter", "WP-004", "0083c2"},
	{"Army Painter", "WP-005", "136232"},

	// Scale75 Scalecolor
	{"Scale75", "SC-001", "e8e6df"},
	{"Scale75", "SC-002", "1d1c1b"},
	{"Scale75", "SC-003", "9e2b26"},

	// Tamiya Acrylic
	{"Tamiya", "X-1", "000000"},
	{"Tamiya", "X-2", "ffffff"},
	{"Tamiya", "X-3", "1e2d54"},

	// Mr Hobby Aqueous
	{"Mr Hobby", "H-01", "ffffff"},
	{"Mr Hobby", "H-02", "231815"},

	// Acrilex Acrílica
	{"Acrilex", "AC-001", "e9e6dd"},
	{"Acrilex", "AC-002", "201f1d"},
	{"Acrilex", "AC-003", "a3282a"},
	{"Acrilex", "AC-004", "f2c81e"},
	{"Acrilex", "AC-005", "d3a531"},
	{"Acrilex", "AC-006", "27487e"},
	{"Acrilex", "AC-007", "2f7245"},
	{"Acrilex", "AC-008", "d9662b"},

	// Corfix Acrílica
	{"Corfix", "CF-001", "ece8de"},
	{"Corfix", "CF-002", "1e1d1c"},

	// Talento Acrílica
	{"Talento", "TA-001", "eae6dc"},
	{"Talento", "TA-002", "1f1e1c"},
}

// SeedPaintColors popula paint_colors com RGB real e todos os espaços de
// cor derivados (HSV, HSL, Lab, LCH), substituindo o placeholder cinza
// (128,128,128) usado como fallback pelo gerador de swatches.
func SeedPaintColors(db *sql.DB) error {
	var matched, unmatched int

	for _, pc := range paintColorData {
		r, g, b, err := hexToRGB(pc.Hex)
		if err != nil {
			return fmt.Errorf("cor inválida %q para %s/%s: %w", pc.Hex, pc.Manufacturer, pc.Code, err)
		}

		h, s, v := color.RGBToHSV(r, g, b)
		hslH, hslS, hslL := color.RGBToHSL(r, g, b)
		labL, labA, labB := color.RGBToLab(r, g, b)
		_, lchC, lchH := color.LabToLCH(labL, labA, labB)

		res, err := db.Exec(`
			INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b, hsv_h, hsv_s, hsv_v, hsl_h, hsl_s, hsl_l, lab_l, lab_a, lab_b, lch_l, lch_c, lch_h, updated_at)
			SELECT p.id, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP
			FROM paints p
			JOIN manufacturers m ON m.id = p.manufacturer_id
			WHERE m.name = ? AND p.code = ?
			ON CONFLICT(paint_id) DO UPDATE SET
				rgb_r = excluded.rgb_r, rgb_g = excluded.rgb_g, rgb_b = excluded.rgb_b,
				hsv_h = excluded.hsv_h, hsv_s = excluded.hsv_s, hsv_v = excluded.hsv_v,
				hsl_h = excluded.hsl_h, hsl_s = excluded.hsl_s, hsl_l = excluded.hsl_l,
				lab_l = excluded.lab_l, lab_a = excluded.lab_a, lab_b = excluded.lab_b,
				lch_l = excluded.lch_l, lch_c = excluded.lch_c, lch_h = excluded.lch_h,
				updated_at = CURRENT_TIMESTAMP
		`, r, g, b, h, s, v, hslH, hslS, hslL, labL, labA, labB, labL, lchC, lchH, pc.Manufacturer, pc.Code)
		if err != nil {
			return fmt.Errorf("gravar cor de %s/%s: %w", pc.Manufacturer, pc.Code, err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("verificar gravação de %s/%s: %w", pc.Manufacturer, pc.Code, err)
		}
		if affected == 0 {
			log.Printf("[seed] paint_colors: nenhuma tinta encontrada para %s/%s (verifique se --import-paints já rodou)", pc.Manufacturer, pc.Code)
			unmatched++
			continue
		}
		matched++
	}

	log.Printf("[seed] paint_colors: %d cores gravadas, %d códigos não encontrados", matched, unmatched)
	return nil
}

// hexToRGB converte uma string hex "RRGGBB" (sem #) para componentes RGB.
func hexToRGB(hex string) (r, g, b uint8, err error) {
	var ri, gi, bi int
	if n, scanErr := fmt.Sscanf(hex, "%02x%02x%02x", &ri, &gi, &bi); scanErr != nil || n != 3 {
		return 0, 0, 0, fmt.Errorf("formato hex inválido: %q", hex)
	}
	return uint8(ri), uint8(gi), uint8(bi), nil
}

// GetPaintColorsSeed retorna o Seed registro para este pacote
func GetPaintColorsSeed() Seed {
	return Seed{
		Name: "006_paint_colors",
		Fn:   SeedPaintColors,
	}
}
