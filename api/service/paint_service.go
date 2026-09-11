package service

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"paint-match-ai/api/domain/ai"
	"paint-match-ai/api/domain/color"
	"paint-match-ai/api/domain/equivalence"
	"paint-match-ai/api/domain/mix"
	"paint-match-ai/api/domain/similarity"

	_ "modernc.org/sqlite"
)

type PaintService struct {
	db  *sql.DB
	ret *ai.Retrieval
	sim *similarity.Engine
}

// NewPaintService abre o banco de catálogo. Ordem de resolução:
//  0. MESCLA_DB_PATH, se definida (container do apiserver — volume montado);
//  1. data/paint_knowledge.db no diretório atual (fluxo de desenvolvimento);
//  2. paint_knowledge.db no diretório atual (legado);
//  3. banco já instalado no diretório de dados do usuário;
//  4. primeiro boot: extrai o banco embutido no binário (embeddedSeed)
//     para o diretório de dados — o app é auto-suficiente, sem instalador.
func NewPaintService(embeddedSeed []byte) (*PaintService, error) {
	if envPath := os.Getenv("MESCLA_DB_PATH"); envPath != "" {
		return openPaintService(envPath)
	}

	dbPath := "data/paint_knowledge.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbPath = "paint_knowledge.db"
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			cfgDir, err := os.UserConfigDir()
			if err != nil {
				home, _ := os.UserHomeDir()
				cfgDir = home
			}
			dataDir := filepath.Join(cfgDir, "Mescla")
			dbPath = filepath.Join(dataDir, "paint_knowledge.db")

			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				if len(embeddedSeed) == 0 {
					return nil, fmt.Errorf("banco de catálogo não encontrado (nem no diretório atual, nem em %s, nem embutido no binário)", dataDir)
				}
				if err := os.MkdirAll(dataDir, 0o755); err != nil {
					return nil, fmt.Errorf("criando diretório de dados: %w", err)
				}
				if err := os.WriteFile(dbPath, embeddedSeed, 0o644); err != nil {
					return nil, fmt.Errorf("instalando banco de catálogo: %w", err)
				}
			}
		}
	}

	return openPaintService(dbPath)
}

// openPaintService abre dbPath (já resolvido) e prepara o schema — extraído
// pra ser reaproveitado tanto pela resolução local do desktop/CLI quanto pelo
// caminho direto de MESCLA_DB_PATH (apiserver em container, path do volume).
func openPaintService(dbPath string) (*PaintService, error) {
	// Os pragmas vão no DSN, não em db.Exec: database/sql mantém um POOL, e um
	// `PRAGMA foreign_keys = ON` executado solto vale só para a conexão que o
	// rodou. Com o pragma no DSN o driver o aplica a cada conexão nova — sem
	// isso, um DELETE que caísse noutra conexão não dispararia cascata nenhuma
	// e deixaria abas e regiões órfãs para sempre.
	dsn := dbPath
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	// busy_timeout: escrita concorrente espera o lock em vez de falhar na hora
	// (dois cadastros simultâneos do mesmo fabricante dão 200 e 409, não 500).
	dsn += sep + "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("pragma: %w", err)
	}

	// O estoque do usuário vive no mesmo banco (já no diretório de dados dele),
	// mas numa tabela própria criada sob demanda — o catálogo embutido é
	// read-only e reseedado pelo dev, então não pode carregar dados do usuário.
	if err := ensureUserSchema(db); err != nil {
		return nil, fmt.Errorf("preparando estoque do usuário: %w", err)
	}
	if err := ensurePlanningSchema(db); err != nil {
		return nil, fmt.Errorf("preparando planos de pintura: %w", err)
	}
	if err := ensureSavedRecipesSchema(db); err != nil {
		return nil, fmt.Errorf("preparando receitas salvas: %w", err)
	}

	return &PaintService{
		db:  db,
		ret: ai.NewRetrieval(db),
		sim: similarity.NewEngine(db),
	}, nil
}

func (s *PaintService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

type PaintDTO struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	ProductLine    string `json:"productLine"`
	R              uint8  `json:"r"`
	G              uint8  `json:"g"`
	B              uint8  `json:"b"`
	SwatchPath     string `json:"swatchPath"`
	Thumbnail      string `json:"thumbnail"`
	ImageURL       string `json:"imageUrl"`
	FinishType     string `json:"finishType"`
	PaintType      string `json:"paintType"`
	Coverage       string `json:"coverage"`
	Opacity        string `json:"opacity"`
	Volume         string `json:"volume"`
}

type SearchResultDTO struct {
	PaintID      int64   `json:"paintId"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	R            uint8   `json:"r"`
	G            uint8   `json:"g"`
	B            uint8   `json:"b"`
	DeltaE       float64 `json:"deltaE"`
	SwatchPath   string  `json:"swatchPath"`
	Similarity   float64 `json:"similarity"`
}

type RecipeIngredientDTO struct {
	PaintID    int64   `json:"paintId"`
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Percentage float64 `json:"percentage"`
	R          uint8   `json:"r"`
	G          uint8   `json:"g"`
	B          uint8   `json:"b"`
	// ManufacturerID e Manufacturer identificam a marca deste ingrediente —
	// rf-13, obrigatório para cross-brand (a receita pode combinar potes de
	// fabricantes diferentes).
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
}

// mapIngredients converte os ingredientes de mix.Recipe para o DTO trocado
// com o cliente. Extraído porque o mesmo laço se repetia em
// paint_service.go, universo.go, userstock.go e comparisons.go (rf-13) — os
// quatro precisam do par manufacturerId/manufacturer.
func mapIngredients(ingredients []mix.Ingredient) []RecipeIngredientDTO {
	dto := make([]RecipeIngredientDTO, 0, len(ingredients))
	for _, ing := range ingredients {
		dto = append(dto, RecipeIngredientDTO{
			PaintID:        ing.Paint.ID,
			Name:           ing.Paint.Name,
			Code:           ing.Paint.Code,
			Percentage:     ing.Percentage,
			R:              ing.Paint.R,
			G:              ing.Paint.G,
			B:              ing.Paint.B,
			ManufacturerID: ing.Paint.ManufacturerID,
			Manufacturer:   ing.Paint.Manufacturer,
		})
	}
	return dto
}

// crossBrandInfo decide crossBrand a partir dos ingredientes já montados
// (RN6) — nunca o cliente. manufacturers sai ordenado alfabeticamente e sem
// repetição, só os fabricantes presentes nos ingredientes.
func crossBrandInfo(ingredients []RecipeIngredientDTO) (bool, []string) {
	// Ingrediente com ManufacturerID 0 é fabricante DESCONHECIDO, não uma
	// marca a mais: contá-lo daria crossBrand=true com um nome só em
	// manufacturers, e a tela diria "2 marcas" nomeando uma.
	ids := make(map[int64]bool)
	names := make(map[string]bool)
	for _, ing := range ingredients {
		if ing.ManufacturerID != 0 {
			ids[ing.ManufacturerID] = true
		}
		if ing.Manufacturer != "" {
			names[ing.Manufacturer] = true
		}
	}
	manufacturers := make([]string, 0, len(names))
	for n := range names {
		manufacturers = append(manufacturers, n)
	}
	sort.Strings(manufacturers)
	return len(ids) > 1, manufacturers
}

type EquivalentRecipeDTO struct {
	SourcePaintID      int64                 `json:"sourcePaintId"`
	SourceName         string                `json:"sourceName"`
	SourceManufacturer string                `json:"sourceManufacturer"`
	SourceR            uint8                 `json:"sourceR"`
	SourceG            uint8                 `json:"sourceG"`
	SourceB            uint8                 `json:"sourceB"`
	TargetManufacturer string                `json:"targetManufacturer"`
	Ingredients        []RecipeIngredientDTO `json:"ingredients"`
	ResultR            uint8                 `json:"resultR"`
	ResultG            uint8                 `json:"resultG"`
	ResultB            uint8                 `json:"resultB"`
	DeltaE             float64               `json:"deltaE"`
	Method             string                `json:"method"`
	// Reproducible indica se a mistura chega perto o bastante da cor de origem
	// pra ser considerada uma equivalência de verdade. Quando falso, o fabricante
	// de destino não tem os pigmentos necessários e os "ingredientes" são só a
	// melhor aproximação possível — não uma receita utilizável.
	Reproducible bool     `json:"reproducible"`
	Tips         []string `json:"tips"`
	// Faixa classifica o acerto (rf-11 RN3): otimo, aproximada ou
	// nao-encontrei. É o que a tela usa para o selo e para decidir se abre o
	// diálogo de fallback.
	Faixa FaixaQualidade `json:"faixa,omitempty"`
	// ForaDoUniverso marca a receita resolvida fora do universo que o usuário
	// pediu — só acontece quando ele autorizou a saída no diálogo (rf-11 RN7).
	ForaDoUniverso bool `json:"foraDoUniverso,omitempty"`
	// CrossBrand e Manufacturers (rf-13 RN6) — decididos pelo servidor a
	// partir dos ingredientes, nunca pelo cliente.
	CrossBrand    bool     `json:"crossBrand"`
	Manufacturers []string `json:"manufacturers"`
}

type ManufacturerDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Country    string `json:"country"`
	Website    string `json:"website"`
	LogoPath   string `json:"logoPath"`
	PaintCount int    `json:"paintCount"`
	// UserPaintCount conta as tintas do estoque do servidor: junto com
	// PaintCount, é o que impede excluir o fabricante (RG-18, rf-14).
	UserPaintCount int `json:"userPaintCount"`
}

type StatsDTO struct {
	Manufacturers int `json:"manufacturers"`
	ProductLines  int `json:"productLines"`
	Paints        int `json:"paints"`
	Equivalences  int `json:"equivalences"`
	Recipes       int `json:"recipes"`
}

func (s *PaintService) GetStats() (StatsDTO, error) {
	var stats StatsDTO
	s.db.QueryRow("SELECT COUNT(*) FROM manufacturers").Scan(&stats.Manufacturers)
	s.db.QueryRow("SELECT COUNT(*) FROM product_lines").Scan(&stats.ProductLines)
	s.db.QueryRow("SELECT COUNT(*) FROM paints").Scan(&stats.Paints)
	s.db.QueryRow("SELECT COUNT(*) FROM equivalences").Scan(&stats.Equivalences)
	s.db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&stats.Recipes)
	return stats, nil
}

const manufacturerSelect = `
	SELECT m.id, m.name, COALESCE(m.country, ''), COALESCE(m.website, ''), COALESCE(m.logo_path, ''),
	       (SELECT COUNT(*) FROM paints p WHERE p.manufacturer_id = m.id),
	       (SELECT COUNT(*) FROM user_paints u WHERE u.manufacturer_id = m.id)
	FROM manufacturers m`

func (s *PaintService) GetManufacturers() ([]ManufacturerDTO, error) {
	rows, err := s.db.Query(manufacturerSelect + ` ORDER BY m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ManufacturerDTO
	for rows.Next() {
		var m ManufacturerDTO
		if err := rows.Scan(&m.ID, &m.Name, &m.Country, &m.Website, &m.LogoPath, &m.PaintCount, &m.UserPaintCount); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *PaintService) GetAllPaints() ([]PaintDTO, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), p.manufacturer_id, m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		ORDER BY m.name, p.name
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PaintDTO
	for rows.Next() {
		var p PaintDTO
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.ManufacturerID, &p.Manufacturer,
			&p.ProductLine,
			&p.R, &p.G, &p.B,
			&p.SwatchPath,
			&p.Thumbnail, &p.ImageURL,
			&p.FinishType, &p.PaintType,
			&p.Coverage, &p.Opacity,
			&p.Volume,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// compactCode reduz o código do pote a letras e dígitos minúsculos: "70.951"
// vira "70951", "XF-2" vira "xf2". O SQL de SearchPaints faz o mesmo em p.code
// tirando só '.', '-' e ' ', a única pontuação que o catálogo usa em código.
func compactCode(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *PaintService) SearchPaints(query string) ([]PaintDTO, error) {
	sqlQuery := `
		SELECT p.id, p.name, COALESCE(p.code, ''), p.manufacturer_id, m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE p.name LIKE ? OR p.code LIKE ? OR m.name LIKE ?
		   OR REPLACE(REPLACE(REPLACE(LOWER(p.code), '.', ''), '-', ''), ' ', '') LIKE ?
		ORDER BY (? <> '' AND REPLACE(REPLACE(REPLACE(LOWER(p.code), '.', ''), '-', ''), ' ', '') = ?) DESC,
			m.name, p.name
		LIMIT 50
	`
	like := "%" + query + "%"
	c := compactCode(query)
	codeLike := like
	if c != "" {
		codeLike = "%" + c + "%"
	}
	rows, err := s.db.Query(sqlQuery, like, like, like, codeLike, c, c)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PaintDTO
	for rows.Next() {
		var p PaintDTO
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.ManufacturerID, &p.Manufacturer,
			&p.ProductLine,
			&p.R, &p.G, &p.B,
			&p.SwatchPath,
			&p.Thumbnail, &p.ImageURL,
			&p.FinishType, &p.PaintType,
			&p.Coverage, &p.Opacity,
			&p.Volume,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *PaintService) GetPaintByID(id int64) (PaintDTO, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.code, ''), p.manufacturer_id, m.name,
			   COALESCE(pl.name, ''),
			   COALESCE(pc.rgb_r, 0), COALESCE(pc.rgb_g, 0), COALESCE(pc.rgb_b, 0),
			   COALESCE(pc.swatch_path, ''),
			   COALESCE(p.thumbnail_path, ''), COALESCE(p.image_path, ''),
			   COALESCE(ft.name, ''), COALESCE(pt.name, ''),
			   COALESCE(ct.name, ''), COALESCE(ot.name, ''),
			   COALESCE(p.volume_ml || 'ml', '')
		FROM paints p
		JOIN manufacturers m ON m.id = p.manufacturer_id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		LEFT JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN finish_types ft ON ft.id = p.finish_type_id
		LEFT JOIN paint_types pt ON pt.id = p.paint_type_id
		LEFT JOIN coverage_types ct ON ct.id = p.coverage_type_id
		LEFT JOIN opacity_types ot ON ot.id = p.opacity_type_id
		WHERE p.id = ?
	`
	var p PaintDTO
	err := s.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.Code, &p.ManufacturerID, &p.Manufacturer,
		&p.ProductLine,
		&p.R, &p.G, &p.B,
		&p.SwatchPath,
		&p.Thumbnail, &p.ImageURL,
		&p.FinishType, &p.PaintType,
		&p.Coverage, &p.Opacity,
		&p.Volume,
	)
	return p, err
}

func (s *PaintService) FindSimilar(r, g, b uint8, maxDeltaE float64, maxResults int) ([]SearchResultDTO, error) {
	opts := similarity.SearchOptions{
		MaxResults: maxResults,
		MaxDeltaE:  maxDeltaE,
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}
	if opts.MaxDeltaE <= 0 {
		opts.MaxDeltaE = 10.0
	}

	results, err := s.sim.FindSimilar(r, g, b, opts)
	if err != nil {
		return nil, err
	}

	var dto []SearchResultDTO
	for i, res := range results {
		dto = append(dto, SearchResultDTO{
			PaintID:      res.PaintID,
			Name:         res.Name,
			Manufacturer: res.Manufacturer,
			R:            res.R,
			G:            res.G,
			B:            res.B,
			DeltaE:       res.DeltaE,
			SwatchPath:   res.SwatchPath,
			Similarity:   100.0 - res.DeltaE,
		})
		_ = i
	}
	return dto, nil
}

func (s *PaintService) FindEquivalences(paintID int64) ([]SearchResultDTO, error) {
	results, err := s.sim.FindEquivalent(paintID)
	if err != nil {
		return nil, err
	}

	var dto []SearchResultDTO
	for _, res := range results {
		dto = append(dto, SearchResultDTO{
			PaintID:      res.PaintID,
			Name:         res.Name,
			Manufacturer: res.Manufacturer,
			R:            res.R,
			G:            res.G,
			B:            res.B,
			DeltaE:       res.DeltaE,
			SwatchPath:   res.SwatchPath,
			Similarity:   100.0 - res.DeltaE,
		})
	}
	return dto, nil
}

// SuggestEquivalentRecipe busca a tinta de origem (de qualquer fabricante) e
// monta uma receita de mistura que aproxima a cor da origem, junto com dicas
// de ajuste em texto.
//
// targetManufacturerID == 0 (rf-13) abre o pool para o catálogo inteiro —
// cross-brand real, RG-13. maxIngredients <= 0 é sem teto (RN5).
func (s *PaintService) SuggestEquivalentRecipe(sourcePaintID int64, targetManufacturerID int64, maxIngredients int) (EquivalentRecipeDTO, error) {
	source, err := s.GetPaintByID(sourcePaintID)
	if err != nil {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não encontrada: %w", err)
	}
	if source.R == 0 && source.G == 0 && source.B == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("tinta de origem não possui dados de cor cadastrados")
	}

	var candidates []mix.PaintInput
	var targetMfrName string
	if targetManufacturerID > 0 {
		if err := s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", targetManufacturerID).Scan(&targetMfrName); err != nil {
			return EquivalentRecipeDTO{}, fmt.Errorf("fabricante não encontrado")
		}
		candidates, err = s.loadPaintsByManufacturerID(targetManufacturerID)
	} else {
		candidates, err = s.catalogoInteiroComCor()
	}
	if err != nil {
		return EquivalentRecipeDTO{}, err
	}
	if len(candidates) == 0 {
		if targetManufacturerID > 0 {
			return EquivalentRecipeDTO{}, fmt.Errorf("fabricante de destino não possui tintas cadastradas com cor")
		}
		// Cross-brand não tem fabricante de destino — dizer que "o fabricante"
		// está vazio mentiria sobre a causa.
		return EquivalentRecipeDTO{}, fmt.Errorf("o catálogo não tem nenhuma tinta com cor cadastrada")
	}

	// Toda a regra de negócio (exclusão da tinta-alvo, mistura forçada na mesma
	// marca, dicas, reproduzível) vive em api/domain/equivalence — compartilhada com o
	// módulo WASM do app mobile, que deve produzir a mesma receita.
	sourceInput := mix.PaintInput{ID: source.ID, Name: source.Name, Code: source.Code, R: source.R, G: source.G, B: source.B}
	res, err := equivalence.Suggest(sourceInput, source.Manufacturer, targetMfrName, candidates, maxIngredients)
	if err != nil {
		if errors.Is(err, equivalence.ErrNoCandidates) {
			if targetManufacturerID > 0 {
				return EquivalentRecipeDTO{}, fmt.Errorf("essa marca não tem outras tintas com cor pra montar a mistura")
			}
			return EquivalentRecipeDTO{}, fmt.Errorf("o catálogo não tem outra tinta com cor pra montar a mistura")
		}
		return EquivalentRecipeDTO{}, err
	}
	recipe := res.Recipe

	ingredients := mapIngredients(recipe.Ingredients)
	crossBrand, manufacturers := crossBrandInfo(ingredients)

	return EquivalentRecipeDTO{
		SourcePaintID:      source.ID,
		SourceName:         source.Name,
		SourceManufacturer: source.Manufacturer,
		SourceR:            source.R,
		SourceG:            source.G,
		SourceB:            source.B,
		TargetManufacturer: targetMfrName,
		Ingredients:        ingredients,
		ResultR:            recipe.ResultR,
		ResultG:            recipe.ResultG,
		ResultB:            recipe.ResultB,
		DeltaE:             recipe.DeltaE,
		Method:             recipe.Method,
		Reproducible:       res.Reproducible,
		Tips:               res.Tips,
		CrossBrand:         crossBrand,
		Manufacturers:      manufacturers,
	}, nil
}

// SuggestRecipeForColor monta a receita equivalente para uma COR ARBITRÁRIA
// (r,g,b) dentro do catálogo de um fabricante de destino — usada pela Roda
// cromática, onde o "alvo" é um passo calculado da rampa (não uma tinta do
// catálogo). Reaproveita a mesma orquestração de equivalence.Suggest; a origem
// é sintética (ID 0, sem marca), então nada é excluído do pool de candidatos.
func (s *PaintService) SuggestRecipeForColor(r, g, b uint8, targetManufacturerID int64) (EquivalentRecipeDTO, error) {
	candidates, err := s.loadPaintsByManufacturerID(targetManufacturerID)
	if err != nil {
		return EquivalentRecipeDTO{}, err
	}
	if len(candidates) == 0 {
		return EquivalentRecipeDTO{}, fmt.Errorf("fabricante de destino não possui tintas cadastradas com cor")
	}

	var targetMfrName string
	s.db.QueryRow("SELECT name FROM manufacturers WHERE id = ?", targetManufacturerID).Scan(&targetMfrName)

	sourceInput := mix.PaintInput{ID: 0, Name: "cor alvo", Code: "", R: r, G: g, B: b}
	res, err := equivalence.Suggest(sourceInput, "", targetMfrName, candidates, 0)
	if err != nil {
		if errors.Is(err, equivalence.ErrNoCandidates) {
			return EquivalentRecipeDTO{}, fmt.Errorf("essa marca não tem tintas com cor pra montar a mistura")
		}
		return EquivalentRecipeDTO{}, err
	}
	recipe := res.Recipe

	ingredients := mapIngredients(recipe.Ingredients)

	return EquivalentRecipeDTO{
		SourcePaintID:      0,
		SourceName:         "cor alvo",
		SourceManufacturer: "",
		SourceR:            r,
		SourceG:            g,
		SourceB:            b,
		TargetManufacturer: targetMfrName,
		Ingredients:        ingredients,
		ResultR:            recipe.ResultR,
		ResultG:            recipe.ResultG,
		ResultB:            recipe.ResultB,
		DeltaE:             recipe.DeltaE,
		Method:             recipe.Method,
		Reproducible:       res.Reproducible,
		Tips:               res.Tips,
	}, nil
}

// loadPaintsByManufacturerID carrega as tintas com cor cadastrada de um único
// fabricante — pool de candidatos para SuggestBestSubset. Usa JOIN (não
// LEFT JOIN + COALESCE como o resto do arquivo) de propósito: tinta sem RGB
// não deve virar candidato "preto" silencioso numa receita de mistura.
func (s *PaintService) loadPaintsByManufacturerID(manufacturerID int64) ([]mix.PaintInput, error) {
	query := `
		SELECT p.id, p.name, p.code, pc.rgb_r, pc.rgb_g, pc.rgb_b, p.manufacturer_id, m.name
		FROM paints p
		JOIN paint_colors pc ON pc.paint_id = p.id
		JOIN manufacturers m ON m.id = p.manufacturer_id
		WHERE p.manufacturer_id = ?
	`
	rows, err := s.db.Query(query, manufacturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paints []mix.PaintInput
	for rows.Next() {
		var p mix.PaintInput
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.R, &p.G, &p.B, &p.ManufacturerID, &p.Manufacturer); err != nil {
			return nil, err
		}
		paints = append(paints, p)
	}
	return paints, rows.Err()
}

func (s *PaintService) ProcessQuery(text string) (ai.Response, error) {
	query := ai.Query{
		Text:   text,
		Intent: ai.IntentGeneral,
	}

	paints, _ := s.SearchPaints(text)
	if len(paints) > 0 {
		query.Intent = ai.IntentPaintInfo
	}

	return s.ret.Process(query)
}

func (s *PaintService) CompareColors(paintIDs []int64) ([]SearchResultDTO, error) {
	if len(paintIDs) < 2 {
		return nil, fmt.Errorf("precisa de pelo menos 2 tintas para comparar")
	}

	var paints []PaintDTO
	for _, id := range paintIDs {
		p, err := s.GetPaintByID(id)
		if err != nil {
			continue
		}
		paints = append(paints, p)
	}

	if len(paints) < 2 {
		return nil, fmt.Errorf("poucas tintas encontradas para comparar")
	}

	var results []SearchResultDTO
	for i := 0; i < len(paints); i++ {
		for j := i + 1; j < len(paints); j++ {
			l1, a1, b1 := color.RGBToLab(paints[i].R, paints[i].G, paints[i].B)
			l2, a2, b2 := color.RGBToLab(paints[j].R, paints[j].G, paints[j].B)
			deltaE := color.DeltaE2000(
				[3]float64{l1, a1, b1},
				[3]float64{l2, a2, b2},
			)
			results = append(results, SearchResultDTO{
				PaintID:      paints[j].ID,
				Name:         fmt.Sprintf("%s vs %s", paints[i].Name, paints[j].Name),
				Manufacturer: fmt.Sprintf("%s / %s", paints[i].Manufacturer, paints[j].Manufacturer),
				R:            paints[j].R,
				G:            paints[j].G,
				B:            paints[j].B,
				DeltaE:       deltaE,
				Similarity:   100.0 - deltaE,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DeltaE < results[j].DeltaE
	})

	return results, nil
}
