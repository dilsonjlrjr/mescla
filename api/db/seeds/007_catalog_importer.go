package seeds

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"paint-match-ai/api/domain/color"
)

// DefaultCatalogDir é o diretório com os arquivos de catálogo (um .md por
// marca) baixados do dataset Miniature Painter Pro.
const DefaultCatalogDir = "db/data/paints"

// brandManufacturer mapeia o nome do arquivo de dados (sem .md) para o nome do
// fabricante no app. As marcas que já existem no seed 002 são reaproveitadas
// (merge por nome); AK e AKRC caem no mesmo fabricante, distintos só pela linha.
var brandManufacturer = map[string]string{
	// Já existentes no app (merge)
	"AK":              "AK Interactive",
	"AKRC":            "AK Interactive",
	"Vallejo":         "Vallejo",
	"Citadel_Colour":  "Citadel",
	"Army_Painter":    "Army Painter",
	"Scale75":         "Scale75",
	"Tamiya":          "Tamiya",
	"MrHobby":         "Mr Hobby",
	"Acrilex":         "Acrilex",
	"Corfix":          "Corfix",
	"Talento":         "Talento",
	"GreenStuffWorld": "Green Stuff World",
	// Novas marcas do dataset
	"AppleBarrel":   "Apple Barrel",
	"Arteza":        "Arteza",
	"CoatDArmes":    "Coat d'Armes",
	"Creature":      "Creature Caster",
	"Duncan":        "Duncan Rhodes",
	"FolkArt":       "FolkArt",
	"Foundry":       "Foundry",
	"Golden":        "Golden",
	"Humbrol":       "Humbrol",
	"Italeri":       "Italeri",
	"KimeraKolors":  "Kimera Kolors",
	"Liquitex":      "Liquitex",
	"Mig":           "AMMO by Mig",
	"MissionModels": "Mission Models",
	"Monument":      "Monument Pro Acryl",
	"MrPaint":       "Mr. Paint",
	"P3":            "P3 Formula",
	"Pantone":       "Pantone",
	"RAL":           "RAL",
	"Reaper":        "Reaper",
	"Revell":        "Revell",
	"TomColor":      "TomColor",
	"TurboDork":     "Turbo Dork",
	"Warcolours":    "Warcolours",
}

// brandMeta traz país/site das marcas novas (as antigas já vêm do seed 002).
var brandMeta = map[string]ManufacturerSeed{
	"Apple Barrel":       {Country: "USA", Website: "https://www.plaidonline.com"},
	"Arteza":             {Country: "USA", Website: "https://arteza.com"},
	"Coat d'Armes":       {Country: "UK", Website: "https://coatdarms.com"},
	"Creature Caster":    {Country: "Canada", Website: "https://creaturecaster.com"},
	"Duncan Rhodes":      {Country: "UK", Website: "https://duncanrhodes.com"},
	"FolkArt":            {Country: "USA", Website: "https://www.plaidonline.com"},
	"Foundry":            {Country: "UK", Website: "https://www.wargamesfoundry.com"},
	"Golden":             {Country: "USA", Website: "https://www.goldenpaints.com"},
	"Humbrol":            {Country: "UK", Website: "https://www.humbrol.com"},
	"Italeri":            {Country: "Italy", Website: "https://www.italeri.com"},
	"Kimera Kolors":      {Country: "Italy", Website: "https://www.kimera-kolors.com"},
	"Liquitex":           {Country: "USA", Website: "https://www.liquitex.com"},
	"AMMO by Mig":        {Country: "Spain", Website: "https://www.migjimenez.com"},
	"Mission Models":     {Country: "USA", Website: "https://missionmodelsus.com"},
	"Monument Pro Acryl": {Country: "USA", Website: "https://monumenthobbies.com"},
	"Mr. Paint":          {Country: "Slovakia", Website: "https://mrpaint.sk"},
	"P3 Formula":         {Country: "USA", Website: "https://privateerpress.com"},
	"Pantone":            {Country: "USA", Website: "https://www.pantone.com"},
	"RAL":                {Country: "Germany", Website: "https://www.ral-farben.de"},
	"Reaper":             {Country: "USA", Website: "https://www.reapermini.com"},
	"Revell":             {Country: "Germany", Website: "https://www.revell.de"},
	"TomColor":           {Country: "China", Website: ""},
	"Turbo Dork":         {Country: "USA", Website: "https://turbodork.com"},
	"Warcolours":         {Country: "Greece", Website: "https://www.warcolours.com"},
}

// catalogPaint é uma tinta lida de um arquivo de catálogo.
type catalogPaint struct {
	Manufacturer string
	Line         string
	Code         string
	Name         string
	R, G, B      uint8
}

// ImportCatalog lê todos os .md de dataDir e popula manufacturers, product_lines,
// paints e paint_colors com o catálogo completo. Idempotente: reexecutar
// atualiza em vez de duplicar (upsert por nome de fabricante, por
// manufacturer_id+name de linha e por manufacturer_id+product_line_id+code).
func ImportCatalog(db *sql.DB, dataDir string) error {
	files, err := filepath.Glob(filepath.Join(dataDir, "*.md"))
	if err != nil {
		return fmt.Errorf("listar arquivos em %s: %w", dataDir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("nenhum arquivo .md encontrado em %s (rode o download do dataset primeiro)", dataDir)
	}
	sort.Strings(files)

	var paints []catalogPaint
	for _, f := range files {
		brand := strings.TrimSuffix(filepath.Base(f), ".md")
		mfr, ok := brandManufacturer[brand]
		if !ok {
			log.Printf("[catalog] marca sem mapeamento, ignorando: %s", brand)
			continue
		}
		parsed, err := parseCatalogFile(f, mfr)
		if err != nil {
			return fmt.Errorf("parsear %s: %w", f, err)
		}
		paints = append(paints, parsed...)
	}
	log.Printf("[catalog] %d tintas lidas de %d arquivos", len(paints), len(files))

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	mfrIDs := map[string]int64{}
	lineIDs := map[string]int64{}   // chave: mfrName|lineName
	usedCodes := map[string]bool{}  // chave: mfrID|lineID|code
	var nMfr, nLine, nPaint int

	for _, p := range paints {
		// Fabricante
		mfrID, ok := mfrIDs[p.Manufacturer]
		if !ok {
			mfrID, err = ensureManufacturer(tx, p.Manufacturer)
			if err != nil {
				return err
			}
			mfrIDs[p.Manufacturer] = mfrID
			nMfr++
		}

		// Linha de produto
		lineKey := p.Manufacturer + "|" + p.Line
		lineID, ok := lineIDs[lineKey]
		if !ok {
			lineID, err = ensureProductLine(tx, mfrID, p.Line)
			if err != nil {
				return err
			}
			lineIDs[lineKey] = lineID
			nLine++
		}

		// Código único dentro da (marca, linha)
		code := uniqueCode(usedCodes, mfrID, lineID, p.Code, p.Name)

		paintID, err := upsertPaint(tx, mfrID, lineID, code, p.Name)
		if err != nil {
			return fmt.Errorf("gravar tinta %s/%s/%s: %w", p.Manufacturer, p.Line, p.Name, err)
		}
		if err := upsertPaintColor(tx, paintID, p.R, p.G, p.B); err != nil {
			return fmt.Errorf("gravar cor de %s/%s/%s: %w", p.Manufacturer, p.Line, p.Name, err)
		}
		nPaint++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("[catalog] concluído: %d fabricantes, %d linhas, %d tintas", nMfr, nLine, nPaint)
	return nil
}

// parseCatalogFile lê um .md de marca e devolve suas tintas. Mapeia colunas
// pelo cabeçalho (algumas marcas não têm coluna Code).
func parseCatalogFile(path, mfr string) ([]catalogPaint, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var out []catalogPaint
	var cols map[string]int

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitRow(line)
		if cols == nil {
			// Primeira linha de tabela = cabeçalho
			if idxOf(cells, "Name") >= 0 && idxOf(cells, "R") >= 0 {
				cols = map[string]int{}
				for i, c := range cells {
					cols[c] = i
				}
			}
			continue
		}
		// Pular linha separadora |---|---|
		if len(cells) > 0 && strings.HasPrefix(cells[0], "-") {
			continue
		}
		r, okR := atoiCell(cells, cols["R"])
		g, okG := atoiCell(cells, cols["G"])
		b, okB := atoiCell(cells, cols["B"])
		if !okR || !okG || !okB {
			continue
		}
		name := cellAt(cells, cols["Name"])
		if name == "" {
			continue
		}
		lineName := strings.TrimSpace(cellAt(cells, colOr(cols, "Set", -1)))
		if lineName == "" {
			lineName = "Geral"
		}
		code := ""
		if ci, ok := cols["Code"]; ok {
			code = cellAt(cells, ci)
		}
		out = append(out, catalogPaint{
			Manufacturer: mfr,
			Line:         lineName,
			Code:         code,
			Name:         name,
			R:            clamp8(r),
			G:            clamp8(g),
			B:            clamp8(b),
		})
	}
	return out, scanner.Err()
}

func ensureManufacturer(tx *sql.Tx, name string) (int64, error) {
	meta := brandMeta[name] // vazio para as marcas já seedadas
	_, err := tx.Exec(`
		INSERT INTO manufacturers (name, country, website)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO NOTHING
	`, name, meta.Country, meta.Website)
	if err != nil {
		return 0, fmt.Errorf("inserir fabricante %s: %w", name, err)
	}
	var id int64
	if err := tx.QueryRow("SELECT id FROM manufacturers WHERE name = ?", name).Scan(&id); err != nil {
		return 0, fmt.Errorf("buscar id do fabricante %s: %w", name, err)
	}
	return id, nil
}

func ensureProductLine(tx *sql.Tx, mfrID int64, name string) (int64, error) {
	_, err := tx.Exec(`
		INSERT INTO product_lines (manufacturer_id, name)
		VALUES (?, ?)
		ON CONFLICT(manufacturer_id, name) DO NOTHING
	`, mfrID, name)
	if err != nil {
		return 0, fmt.Errorf("inserir linha %s: %w", name, err)
	}
	var id int64
	if err := tx.QueryRow("SELECT id FROM product_lines WHERE manufacturer_id = ? AND name = ?", mfrID, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("buscar id da linha %s: %w", name, err)
	}
	return id, nil
}

func upsertPaint(tx *sql.Tx, mfrID, lineID int64, code, name string) (int64, error) {
	_, err := tx.Exec(`
		INSERT INTO paints (manufacturer_id, product_line_id, code, name)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(manufacturer_id, product_line_id, code) DO UPDATE SET
			name = excluded.name,
			updated_at = CURRENT_TIMESTAMP
	`, mfrID, lineID, code, name)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := tx.QueryRow(
		"SELECT id FROM paints WHERE manufacturer_id = ? AND product_line_id = ? AND code = ?",
		mfrID, lineID, code,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertPaintColor(tx *sql.Tx, paintID int64, r, g, b uint8) error {
	h, s, v := color.RGBToHSV(r, g, b)
	hslH, hslS, hslL := color.RGBToHSL(r, g, b)
	labL, labA, labB := color.RGBToLab(r, g, b)
	_, lchC, lchH := color.LabToLCH(labL, labA, labB)

	_, err := tx.Exec(`
		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b, hsv_h, hsv_s, hsv_v, hsl_h, hsl_s, hsl_l, lab_l, lab_a, lab_b, lch_l, lch_c, lch_h, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(paint_id) DO UPDATE SET
			rgb_r = excluded.rgb_r, rgb_g = excluded.rgb_g, rgb_b = excluded.rgb_b,
			hsv_h = excluded.hsv_h, hsv_s = excluded.hsv_s, hsv_v = excluded.hsv_v,
			hsl_h = excluded.hsl_h, hsl_s = excluded.hsl_s, hsl_l = excluded.hsl_l,
			lab_l = excluded.lab_l, lab_a = excluded.lab_a, lab_b = excluded.lab_b,
			lch_l = excluded.lch_l, lch_c = excluded.lch_c, lch_h = excluded.lch_h,
			updated_at = CURRENT_TIMESTAMP
	`, paintID, r, g, b, h, s, v, hslH, hslS, hslL, labL, labA, labB, labL, lchC, lchH)
	return err
}

// uniqueCode garante um código não-vazio e único dentro de (marca, linha).
// Usa o código do dataset quando presente e livre; senão deriva do nome; em
// colisão, acrescenta sufixo numérico.
func uniqueCode(used map[string]bool, mfrID, lineID int64, code, name string) string {
	base := strings.TrimSpace(code)
	if base == "" {
		base = catalogSlug(name)
	}
	if base == "" {
		base = "x"
	}
	prefix := fmt.Sprintf("%d|%d|", mfrID, lineID)
	candidate := base
	for i := 2; used[prefix+candidate]; i++ {
		candidate = base + "-" + strconv.Itoa(i)
	}
	used[prefix+candidate] = true
	return candidate
}

func catalogSlug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// splitRow divide "|a|b|c|" em ["a","b","c"] (sem as bordas vazias), já trimado.
func splitRow(line string) []string {
	parts := strings.Split(line, "|")
	if len(parts) >= 2 {
		parts = parts[1 : len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func idxOf(cells []string, name string) int {
	for i, c := range cells {
		if c == name {
			return i
		}
	}
	return -1
}

func colOr(cols map[string]int, name string, def int) int {
	if i, ok := cols[name]; ok {
		return i
	}
	return def
}

func cellAt(cells []string, idx int) string {
	if idx < 0 || idx >= len(cells) {
		return ""
	}
	return cells[idx]
}

func atoiCell(cells []string, idx int) (int, bool) {
	s := cellAt(cells, idx)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func clamp8(n int) uint8 {
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

// GetCatalogImporterSeed retorna o Seed que importa o catálogo completo.
func GetCatalogImporterSeed(dataDir string) Seed {
	if dataDir == "" {
		dataDir = DefaultCatalogDir
	}
	return Seed{
		Name: "007_catalog_importer",
		Fn: func(db *sql.DB) error {
			return ImportCatalog(db, dataDir)
		},
	}
}
