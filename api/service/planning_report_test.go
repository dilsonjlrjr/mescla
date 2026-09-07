package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"golang.org/x/image/font"
)

// pngDataURL monta uma data URL PNG mínima (w×h, cor sólida) para os testes
// que precisam de uma aba com foto de verdade (decodificável).
func pngDataURL(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("codificando PNG de teste: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func regiaoComTinta(nome, brand, code, paintName, hex string) PaintingRegionDTO {
	return PaintingRegionDTO{
		X: 5, Y: 5, R: 10, G: 20, B: 30, Hex: hex,
		RegionName: nome, PaintBrand: brand, PaintCode: code, PaintName: paintName, DeltaE: 1.2,
	}
}

// TestCA20_DuasAbasTabelaComColunaAbaEPinsReiniciados cobre o CA20: plano com
// 2 abas (3 e 2 regiões) gera tabela de 5 linhas, coluna Aba preenchida com o
// nome de cada figura, e os pins reiniciam em 1 a cada aba (1,2,3 / 1,2).
func TestReportCA20_DuasAbasTabelaComColunaAbaEPinsReiniciados(t *testing.T) {
	tabs := []PaintingTabDTO{
		{
			Name: "Figura 1",
			Regions: []PaintingRegionDTO{
				regiaoComTinta("Capa", "Citadel", "22-14", "Khorne Red", "#7A1F2B"),
				regiaoComTinta("Cinto", "", "", "", "#1B2A3D"),
				regiaoComTinta("Bota", "Vallejo", "70.950", "Black", "#000000"),
			},
		},
		{
			Name: "Costas",
			Regions: []PaintingRegionDTO{
				regiaoComTinta("Manto", "Citadel", "22-14", "Khorne Red", "#7A1F2B"),
				regiaoComTinta("Mochila", "Vallejo", "70.951", "White", "#FFFFFF"),
			},
		},
	}

	blocks := make([]reportTabBlock, 0, len(tabs))
	for _, tab := range tabs {
		blocks = append(blocks, buildTabBlock(tab))
	}

	var allRows []reportRow
	for _, b := range blocks {
		allRows = append(allRows, b.Rows...)
	}
	if len(allRows) != 5 {
		t.Fatalf("esperava 5 linhas na tabela, veio %d", len(allRows))
	}

	wantTab := []string{"Figura 1", "Figura 1", "Figura 1", "Costas", "Costas"}
	wantN := []int{1, 2, 3, 1, 2}
	for i, row := range allRows {
		if row.TabName != wantTab[i] {
			t.Errorf("linha %d: TabName = %q, esperava %q", i, row.TabName, wantTab[i])
		}
		if row.N != wantN[i] {
			t.Errorf("linha %d: N = %d, esperava %d", i, row.N, wantN[i])
		}
	}

	// smoke: os renderizadores aceitam o resultado sem erro.
	data := reportData{Title: "Peça", Generated: "06/09/2026", Blocks: blocks, Paints: buildReportPaints(tabs)}
	if _, err := renderPDF(context.Background(), data); err != nil {
		t.Fatalf("renderPDF: %v", err)
	}
	if _, err := renderPNG(context.Background(), data); err != nil {
		t.Fatalf("renderPNG: %v", err)
	}
}

// TestAbaSemFoto_RelatorioSemBlocoDeDesenho cobre o caso de uma aba sem foto:
// o relatório sai sem o bloco de desenho daquela aba, mas as linhas dela
// continuam na tabela.
func TestReportAbaSemFoto_RelatorioSemBlocoDeDesenho(t *testing.T) {
	tabs := []PaintingTabDTO{
		{
			Name:      "Com foto",
			ImageData: pngDataURL(t, 40, 30),
			Regions:   []PaintingRegionDTO{regiaoComTinta("Capa", "Citadel", "22-14", "Khorne Red", "#7A1F2B")},
		},
		{
			Name:    "Sem foto",
			Regions: []PaintingRegionDTO{regiaoComTinta("Base", "Vallejo", "70.950", "Black", "#000000")},
		},
	}

	var blocks []reportTabBlock
	for _, tab := range tabs {
		blocks = append(blocks, buildTabBlock(tab))
	}

	if blocks[0].Drawing == nil {
		t.Error("aba com foto deveria ter Drawing != nil")
	}
	if blocks[1].Drawing != nil {
		t.Error("aba sem foto deveria ter Drawing == nil")
	}
	if len(blocks[1].Rows) != 1 || blocks[1].Rows[0].TabName != "Sem foto" {
		t.Fatal("aba sem foto deveria manter suas linhas de tabela")
	}

	data := reportData{Title: "Peça", Generated: "06/09/2026", Blocks: blocks, Paints: buildReportPaints(tabs)}
	if _, err := renderPDF(context.Background(), data); err != nil {
		t.Fatalf("renderPDF: %v", err)
	}
	if _, err := renderPNG(context.Background(), data); err != nil {
		t.Fatalf("renderPNG: %v", err)
	}
}

// TestPlanoComUmaAbaSo_ComportamentoIgualAoAnterior cobre o plano com uma
// única aba: um bloco só, pins numerados a partir de 1, sem nenhuma coluna
// ou linha extra quebrando o caminho de antes das abas.
func TestReportPlanoComUmaAbaSo_ComportamentoIgualAoAnterior(t *testing.T) {
	tabs := []PaintingTabDTO{
		{
			Name:      "Figura 1",
			ImageData: pngDataURL(t, 20, 20),
			Regions: []PaintingRegionDTO{
				regiaoComTinta("Capa", "Citadel", "22-14", "Khorne Red", "#7A1F2B"),
				regiaoComTinta("Cinto", "", "", "", "#1B2A3D"),
			},
		},
	}

	blocks := []reportTabBlock{buildTabBlock(tabs[0])}
	if len(blocks) != 1 {
		t.Fatalf("esperava 1 bloco, veio %d", len(blocks))
	}
	if blocks[0].Drawing == nil {
		t.Error("única aba tem foto — deveria ter Drawing")
	}
	if len(blocks[0].Rows) != 2 || blocks[0].Rows[0].N != 1 || blocks[0].Rows[1].N != 2 {
		t.Fatal("pins da aba única deveriam ser 1 e 2")
	}

	data := reportData{Title: "Peça", Generated: "06/09/2026", Blocks: blocks, Paints: buildReportPaints(tabs)}
	if _, err := renderPDF(context.Background(), data); err != nil {
		t.Fatalf("renderPDF: %v", err)
	}
	if _, err := renderPNG(context.Background(), data); err != nil {
		t.Fatalf("renderPNG: %v", err)
	}
}

// TestDedupTintas_CitandoDuasAbas cobre a dedup da lista de tintas por
// paintBrand+paintCode+paintName atravessando abas (RN7 do rf-09): a mesma
// tinta usada em duas abas aparece uma vez, citando as duas.
func TestReportDedupTintas_CitandoDuasAbas(t *testing.T) {
	tabs := []PaintingTabDTO{
		{
			Name: "Figura 1",
			Regions: []PaintingRegionDTO{
				regiaoComTinta("Manto", "Citadel", "22-14", "Khorne Red", "#7A1F2B"),
			},
		},
		{
			Name: "Costas",
			Regions: []PaintingRegionDTO{
				regiaoComTinta("Capa", "Citadel", "22-14", "Khorne Red", "#7A1F2B"),
			},
		},
	}

	paints := buildReportPaints(tabs)
	if len(paints) != 1 {
		t.Fatalf("esperava 1 tinta deduplicada, vieram %d", len(paints))
	}
	p := paints[0]
	if len(p.Tabs) != 2 {
		t.Fatalf("esperava a tinta citando 2 abas, veio %d", len(p.Tabs))
	}
	gotTabs := map[string]bool{p.Tabs[0].Tab: true, p.Tabs[1].Tab: true}
	if !gotTabs["Figura 1"] || !gotTabs["Costas"] {
		t.Fatalf("tinta deveria citar 'Figura 1' e 'Costas', citou %v", gotTabs)
	}
	line := formatPaintLine(p)
	if line == "" {
		t.Fatal("formatPaintLine não deveria devolver vazio")
	}
}

// ========== preserva: BuildPlanReport devolve ErrPlanNotFound para id inexistente ==========
func TestBuildPlanReportPlanNotFound(t *testing.T) {
	s := newPlanningTestService(t)

	_, _, err := s.BuildPlanReport(context.Background(), 99999, "pdf")
	if !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("esperava ErrPlanNotFound, veio %v", err)
	}
}

// ========== preserva: ctx já cancelado na entrada aborta antes de renderizar ==========
func TestBuildPlanReportCtxJaCanceladoNaEntrada(t *testing.T) {
	s := newPlanningTestService(t)
	plan, err := s.SavePlan(PaintingPlanDTO{Name: "Cancelado", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err = s.BuildPlanReport(ctx, plan.ID, "pdf")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, veio %v", err)
	}
}

// ========== preserva: format "png" gera um PNG de verdade ==========
func TestBuildPlanReportFormatoPNG(t *testing.T) {
	s := newPlanningTestService(t)
	plan, err := s.SavePlan(PaintingPlanDTO{Name: "PNG", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	body, filename, err := s.BuildPlanReport(context.Background(), plan.ID, "png")
	if err != nil {
		t.Fatalf("BuildPlanReport: %v", err)
	}
	if !strings.HasSuffix(filename, ".png") {
		t.Fatalf("nome do arquivo deveria terminar em .png, veio %q", filename)
	}
	if !bytes.HasPrefix(body, []byte("\x89PNG")) {
		t.Fatal("corpo deveria ser um PNG válido")
	}
}

// ========== preserva: formato desconhecido cai no renderizador PDF (default) ==========
func TestBuildPlanReportFormatoInvalidoCaiParaPDF(t *testing.T) {
	s := newPlanningTestService(t)
	plan, err := s.SavePlan(PaintingPlanDTO{Name: "Formato", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	body, filename, err := s.BuildPlanReport(context.Background(), plan.ID, "xml")
	if err != nil {
		t.Fatalf("BuildPlanReport: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Fatal("formato desconhecido deveria cair no renderizador PDF (default do switch)")
	}
	if !strings.HasSuffix(filename, ".xml") {
		t.Fatalf("a extensão do arquivo segue o valor recebido sem validar (validação é do handler): %q", filename)
	}
}

// ========== preserva: nome de plano que vira slug vazio cai para "plano" ==========
func TestBuildPlanReportNomeVirandoSlugVazioCaiParaPlano(t *testing.T) {
	s := newPlanningTestService(t)
	plan, err := s.SavePlan(PaintingPlanDTO{Name: "!!!", Tabs: []PaintingTabDTO{{Name: "Figura 1"}}})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	_, filename, err := s.BuildPlanReport(context.Background(), plan.ID, "pdf")
	if err != nil {
		t.Fatalf("BuildPlanReport: %v", err)
	}
	if !strings.HasPrefix(filename, "mescla-plano-") {
		t.Fatalf("nome de plano que vira slug vazio deveria cair para 'plano', veio %q", filename)
	}
}

// ========== preserva: plano sem nenhuma região em nenhuma aba sai como "Nenhuma região marcada" ==========
func TestReportZeroRegionsAcrossTabs_NenhumaRegiaoMarcada(t *testing.T) {
	tabs := []PaintingTabDTO{{Name: "Figura 1"}, {Name: "Figura 2"}}
	var blocks []reportTabBlock
	for _, tab := range tabs {
		blocks = append(blocks, buildTabBlock(tab))
	}
	data := reportData{Title: "Vazio", Generated: "06/09/2026", Blocks: blocks, Paints: buildReportPaints(tabs)}
	if len(data.Paints) != 0 {
		t.Fatalf("plano sem nenhuma região não deveria ter tintas, veio %d", len(data.Paints))
	}

	pdfOut, err := renderPDF(context.Background(), data)
	if err != nil || len(pdfOut) == 0 {
		t.Fatalf("renderPDF com plano vazio: err=%v len=%d", err, len(pdfOut))
	}
	pngOut, err := renderPNG(context.Background(), data)
	if err != nil || len(pngOut) == 0 {
		t.Fatalf("renderPNG com plano vazio: err=%v len=%d", err, len(pngOut))
	}
}

// ========== preserva RN7: imagem que não decodifica sai sem desenho, não vira erro ==========
func TestBuildTabBlockRN7_ImagemQueNaoDecodificaSaiSemDesenho(t *testing.T) {
	garbage := base64.StdEncoding.EncodeToString([]byte("bytes que nao sao imagem"))
	tab := PaintingTabDTO{
		Name:      "Figura 1",
		ImageData: "data:image/png;base64," + garbage,
		Regions:   []PaintingRegionDTO{regiaoComTinta("Capa", "Citadel", "22-14", "Khorne Red", "#7A1F2B")},
	}
	block := buildTabBlock(tab)
	if block.Drawing != nil {
		t.Fatal("RN7: imagem que não decodifica deveria sair sem desenho, não vira erro")
	}
	if len(block.Rows) != 1 {
		t.Fatal("linhas da tabela deveriam continuar mesmo sem desenho")
	}
}

func TestDecodePlanImageRejectsUnknownPrefix(t *testing.T) {
	if _, err := decodePlanImage("data:image/gif;base64,abc"); err == nil {
		t.Fatal("esperava erro para prefixo fora da lista branca")
	}
}

func TestDecodePlanImageRejectsInvalidBase64(t *testing.T) {
	if _, err := decodePlanImage("data:image/png;base64,%%%not-base64%%%"); err == nil {
		t.Fatal("esperava erro para base64 inválido")
	}
}

func TestDecodePlanImageRejectsUndecodableBytes(t *testing.T) {
	garbage := base64.StdEncoding.EncodeToString([]byte("não é uma imagem"))
	if _, err := decodePlanImage("data:image/png;base64," + garbage); err == nil {
		t.Fatal("esperava erro para bytes que não formam uma imagem")
	}
}

func TestFormatDeltaEHandlesOutOfRangeAndInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		v    float64
		want string
	}{
		{"NaN", math.NaN(), "-"},
		{"infinito", math.Inf(1), "-"},
		{"negativo", -0.1, "-"},
		{"acima de 999", 1000, "-"},
		{"válido", 12.34, "12.3"},
	}
	for _, c := range cases {
		if got := formatDeltaE(c.v); got != c.want {
			t.Errorf("%s: formatDeltaE(%v) = %q, esperava %q", c.name, c.v, got, c.want)
		}
	}
}

func TestFormatPinListHandlesEmptyAndMultiple(t *testing.T) {
	if got := formatPinList(nil); got != "pin -" {
		t.Fatalf("pins vazio deveria ser 'pin -', veio %q", got)
	}
	if got := formatPinList([]int{1, 3}); got != "pin 1, pin 3" {
		t.Fatalf("lista de pins formatada incorretamente: %q", got)
	}
}

func TestHexParaRGBFallsBackToWhiteOnInvalidHex(t *testing.T) {
	for _, hex := range []string{"curto", "#GGGGGG", ""} {
		r, g, b := hexParaRGB(hex)
		if r != 255 || g != 255 || b != 255 {
			t.Errorf("hexParaRGB(%q) deveria cair para branco, veio %d,%d,%d", hex, r, g, b)
		}
	}
	r, g, b := hexParaRGB("#7A1F2B")
	if r != 0x7A || g != 0x1F || b != 0x2B {
		t.Errorf("hexParaRGB válido decodificado incorretamente: %d,%d,%d", r, g, b)
	}
}

func TestParseHexColorFallsBackToGrayOnInvalidHex(t *testing.T) {
	c := parseHexColor("nope")
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("parseHexColor inválido deveria cair para cinza médio, veio %+v", c)
	}
	valid := parseHexColor("#00FF80")
	if valid.R != 0 || valid.G != 255 || valid.B != 0x80 {
		t.Errorf("parseHexColor válido decodificado incorretamente: %+v", valid)
	}
}

func TestContrastingTextColorPicksReadableColor(t *testing.T) {
	// Contraste: sobre fundo escuro o texto é branco, sobre fundo claro é
	// preto. É o que o pin numerado precisa para ser legível em qualquer cor.
	if got := contrastingTextColor(color.RGBA{R: 0, G: 0, B: 0, A: 255}); got != color.White {
		t.Errorf("fundo escuro deveria ter texto branco, veio %v", got)
	}
	if got := contrastingTextColor(color.RGBA{R: 255, G: 255, B: 255, A: 255}); got != color.Black {
		t.Errorf("fundo claro deveria ter texto preto, veio %v", got)
	}
}

func TestSanitizeReportTextRemovesControlCharsAndTrims(t *testing.T) {
	got := sanitizeReportText("  linha1\nlinha2\ttab\r\x07fim  ")
	if strings.ContainsAny(got, "\n\r\t\x07") {
		t.Fatalf("texto sanitizado não deveria conter caracteres de controle: %q", got)
	}
	if got != strings.TrimSpace(got) {
		t.Fatalf("texto sanitizado deveria vir sem espaço nas pontas: %q", got)
	}
}

func TestTruncarNaLarguraCutsLongTextWithEllipsis(t *testing.T) {
	face := reportFontFace(13)
	defer face.Close()

	curto := truncarNaLargura(face, "abc", 1000)
	if curto != "abc" {
		t.Fatalf("texto que cabe não deveria ser cortado, veio %q", curto)
	}

	longo := truncarNaLargura(face, strings.Repeat("mescla ", 30), 50)
	if !strings.HasSuffix(longo, "…") {
		t.Fatalf("texto que não cabe deveria terminar em reticências, veio %q", longo)
	}
	if font.MeasureString(face, longo).Round() > 50 {
		t.Fatalf("texto truncado ainda ultrapassa a largura máxima: %q", longo)
	}
}

func TestFitWithinPreservesAspectRatioAndNeverUpscales(t *testing.T) {
	if w, h := fitWithin(0, 0, 100, 50); w != 100 || h != 50 {
		t.Fatalf("dimensão inválida deveria cair para o máximo, veio %d,%d", w, h)
	}
	if w, h := fitWithin(2000, 1000, 500, 500); w != 500 || h != 250 {
		t.Fatalf("deveria reduzir preservando a proporção 2:1, veio %d,%d", w, h)
	}
	if _, h := fitWithin(100, 1000, 500, 200); h != 200 {
		t.Fatalf("altura deveria ser limitada a 200, veio %d", h)
	}
}

func TestReduzirParaRelatorioScalesDownWhenLargerThanLimit(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2000, 100))
	out, escala := reduzirParaRelatorio(src)
	if escala >= 1 {
		t.Fatalf("imagem maior que o limite deveria ser reduzida, escala veio %v", escala)
	}
	if b := out.Bounds(); b.Dx() != larguraMaximaDesenho {
		t.Fatalf("largura reduzida deveria ser %d, veio %d", larguraMaximaDesenho, b.Dx())
	}
}

func TestReduzirParaRelatorioKeepsSmallImageUnchanged(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	out, escala := reduzirParaRelatorio(src)
	if escala != 1 {
		t.Fatalf("imagem dentro do limite não deveria ser reduzida, escala veio %v", escala)
	}
	if out != image.Image(src) {
		t.Fatal("imagem dentro do limite deveria ser devolvida sem cópia")
	}
}

func TestSlugPlanoFallsBackToPlanoWhenResultEmpty(t *testing.T) {
	if got := SlugPlano("!!!"); got != "plano" {
		t.Fatalf("slug de nome só com símbolos deveria ser 'plano', veio %q", got)
	}
}

func TestSlugPlanoTransliteratesAccentsAndCollapsesDashes(t *testing.T) {
	got := SlugPlano("Peça  Ação — Espírito!!")
	if got != "peca-acao-espirito" {
		t.Fatalf("slug deveria transliterar acentos e colapsar separadores, veio %q", got)
	}
}

func TestSlugPlanoTruncatesAtSixtyChars(t *testing.T) {
	got := SlugPlano(strings.Repeat("a", 100))
	if len(got) > 60 {
		t.Fatalf("slug deveria ser truncado em 60 caracteres, veio %d", len(got))
	}
}
