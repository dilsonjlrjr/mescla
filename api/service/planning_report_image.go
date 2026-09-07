package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// pinDesenho é um pin numerado a desenhar sobre o desenho do plano, na cor da
// região que representa (RN2, RN3).
type pinDesenho struct {
	N   int
	X   int
	Y   int
	Hex string
}

// maxDecodeSide e maxDecodePixels barram bomba de descompressão (CAN5): nenhum
// lado acima de 8000 px, nenhuma área acima de 64 MP, verificado com
// image.DecodeConfig antes de qualquer alocação de pixel.
const (
	maxDecodeSide   = 8000
	maxDecodePixels = 64_000_000

	reportWidth = 1240 // RN9: PNG largura fixa, A4 a 150 dpi

	pinRadiusMin = 12
	pinRadiusMax = 48
)

var allowedDataURLPrefixes = []string{
	"data:image/png;base64,",
	"data:image/jpeg;base64,",
}

// decodePlanImage decodifica uma imagem de plano vinda como data URL
// (image/png ou image/jpeg). Qualquer falha — prefixo fora da lista branca,
// base64 inválido, imagem ilegível ou acima dos limites de decodificação —
// devolve erro; quem chama trata isso como "relatório sem desenho" (RN7), sem
// pânico e sem imagem parcial.
func decodePlanImage(dataURL string) (image.Image, error) {
	var payload string
	matched := false
	for _, prefix := range allowedDataURLPrefixes {
		if strings.HasPrefix(dataURL, prefix) {
			payload = dataURL[len(prefix):]
			matched = true
			break
		}
	}
	if !matched {
		return nil, errors.New("prefixo de data URL não reconhecido")
	}

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("base64 inválido: %w", err)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("cabeçalho de imagem ilegível: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errors.New("dimensões de imagem inválidas")
	}
	if cfg.Width > maxDecodeSide || cfg.Height > maxDecodeSide {
		return nil, errors.New("imagem excede o lado máximo permitido")
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxDecodePixels {
		return nil, errors.New("imagem excede a área máxima permitida")
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("imagem ilegível: %w", err)
	}
	return img, nil
}

// larguraMaximaDesenho é o maior lado que o desenho precisa ter em qualquer
// saída: o PNG tem 1240 px de largura e o PDF reduz para a largura útil de A4.
// Reduzir ANTES de desenhar os pins é o que segura a memória — uma foto de
// 7999×7999 copiada em RGBA na resolução cheia custa ~760 MB de pico.
const larguraMaximaDesenho = 1240

// reduzirParaRelatorio devolve a imagem reduzida e o fator aplicado, para as
// coordenadas dos pins acompanharem a redução.
func reduzirParaRelatorio(src image.Image) (image.Image, float64) {
	b := src.Bounds()
	maior := b.Dx()
	if b.Dy() > maior {
		maior = b.Dy()
	}
	if maior <= larguraMaximaDesenho {
		return src, 1
	}
	escala := float64(larguraMaximaDesenho) / float64(maior)
	w, h := int(float64(b.Dx())*escala), int(float64(b.Dy())*escala)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	menor := image.NewRGBA(image.Rect(0, 0, w, h))
	// ApproxBiLinear, não CatmullRom: medido em 7999×7999, CatmullRom leva
	// 9,0 s contra 317 ms — 28× o custo, sem diferença visível num relatório
	// de 1240 px de largura.
	xdraw.ApproxBiLinear.Scale(menor, menor.Bounds(), src, b, xdraw.Over, nil)
	return menor, escala
}

// drawPins copia src para um *image.RGBA novo// drawPins copia src para um *image.RGBA novo (a origem nunca é mutada) e
// desenha, em cada (X, Y) de coordenadas da imagem original, um círculo
// preenchido na cor hex da região, com anel branco de 2 px e o número
// centrado (RN3).
func drawPins(src image.Image, pins []pinDesenho) *image.RGBA {
	src, escala := reduzirParaRelatorio(src)

	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	longSide := bounds.Dx()
	if bounds.Dy() > longSide {
		longSide = bounds.Dy()
	}
	radius := longSide / 40
	if radius < pinRadiusMin {
		radius = pinRadiusMin
	}
	if radius > pinRadiusMax {
		radius = pinRadiusMax
	}

	face := reportFontFace(float64(radius))
	defer face.Close()

	for _, p := range pins {
		fillColor := parseHexColor(p.Hex)
		center := image.Pt(int(float64(p.X)*escala), int(float64(p.Y)*escala))
		drawFilledCircle(dst, center, radius+2, color.White)
		drawFilledCircle(dst, center, radius, fillColor)
		drawCenteredLabel(dst, face, strconv.Itoa(p.N), center, contrastingTextColor(fillColor))
	}

	return dst
}

func drawFilledCircle(dst *image.RGBA, center image.Point, radius int, c color.Color) {
	if radius <= 0 {
		return
	}
	bounds := dst.Bounds()
	rSq := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		y := center.Y + dy
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > rSq {
				continue
			}
			x := center.X + dx
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dst.Set(x, y, c)
		}
	}
}

func drawCenteredLabel(dst *image.RGBA, face font.Face, label string, center image.Point, c color.Color) {
	advance := font.MeasureString(face, label)
	metrics := face.Metrics()
	textWidth := advance.Ceil()
	textHeight := (metrics.Ascent - metrics.Descent).Ceil()

	origin := fixed.Point26_6{
		X: fixed.I(center.X) - advance/2,
		Y: fixed.I(center.Y) + metrics.Ascent - fixed.I(textHeight/2),
	}

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  origin,
	}
	_ = textWidth
	d.DrawString(label)
}

// parseHexColor lê um hex "#RRGGBB" (ou "RRGGBB"); em erro devolve cinza
// médio para nunca falhar o desenho por causa de uma cor inválida.
func parseHexColor(hex string) color.RGBA {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}
	return color.RGBA{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 255,
	}
}

func contrastingTextColor(c color.RGBA) color.Color {
	luminance := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	if luminance > 140 {
		return color.Black
	}
	return color.White
}

// reportData, reportTabBlock, reportRow e reportPaint são o contrato de
// composição do relatório, fornecido pela outra fatia do pacote
// (planning_report.go). Não redefinidos aqui — apenas usados.

// reportFontCache evita reanalisar o TTF embutido a cada chamada.
var reportFontCollection *opentype.Font

func reportFont() *opentype.Font {
	if reportFontCollection != nil {
		return reportFontCollection
	}
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic(fmt.Sprintf("falha ao carregar fonte embutida: %v", err))
	}
	reportFontCollection = f
	return f
}

func reportFontFace(size float64) font.Face {
	if size < 8 {
		size = 8
	}
	face, err := opentype.NewFace(reportFont(), &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Sprintf("falha ao criar face de fonte: %v", err))
	}
	return face
}

// --- montagem do PNG (RN9, CA12, CA13, CA14) ---

const (
	pagePadding    = 40
	lineHeight     = 22
	headerTitleSz  = 22
	headerMetaSz   = 13
	sectionTitleSz = 16
	subtitleTextSz = 15
	bodyTextSz     = 13
	tableRowH      = 26
	colorSwatchPx  = 16
)

// renderPNG monta o relatório único em PNG, largura fixa reportWidth, fundo
// branco, altura conforme o conteúdo (RN9 do rf-08): cabeçalho do plano,
// depois um subtítulo + desenho por aba (RN13 do rf-09; abas sem foto não
// entram no laço) e, ao final, a tabela única (com a coluna Aba) e a lista
// de tintas. Retorna ErrReportFailed quando o resultado codificado passa do
// teto (maxReportOutputBytes).
func renderPNG(ctx context.Context, d reportData) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	titleFace := reportFontFace(headerTitleSz)
	defer titleFace.Close()
	metaFace := reportFontFace(headerMetaSz)
	defer metaFace.Close()
	sectionFace := reportFontFace(sectionTitleSz)
	defer sectionFace.Close()
	subtitleFace := reportFontFace(subtitleTextSz)
	defer subtitleFace.Close()
	bodyFace := reportFontFace(bodyTextSz)
	defer bodyFace.Close()

	innerWidth := reportWidth - 2*pagePadding

	type blockLayout struct {
		width, height int
	}
	layouts := make([]blockLayout, len(d.Blocks))

	// 1) mede a altura total antes de desenhar, para alocar a imagem já no
	// tamanho final (RN9: altura conforme o conteúdo).
	y := pagePadding
	y += lineHeight + 8 // título
	y += lineHeight     // "Generated"
	y += 16             // respiro pós-cabeçalho

	for i, block := range d.Blocks {
		if block.Drawing == nil {
			continue
		}
		b := block.Drawing.Bounds()
		w, h := fitWithin(b.Dx(), b.Dy(), innerWidth, 900)
		layouts[i] = blockLayout{width: w, height: h}
		y += lineHeight + 4 // subtítulo da aba
		y += h + 20
	}

	allRows := make([]reportRow, 0)
	for _, block := range d.Blocks {
		allRows = append(allRows, block.Rows...)
	}

	y += lineHeight + 6 // título da tabela
	if len(allRows) == 0 {
		y += lineHeight // "nenhuma região marcada"
	} else {
		y += tableRowH // cabeçalho da tabela
		y += tableRowH * len(allRows)
	}
	y += 24

	y += lineHeight + 6 // título da lista de tintas
	if len(d.Paints) == 0 {
		y += lineHeight
	} else {
		y += lineHeight * len(d.Paints)
	}
	y += pagePadding

	totalHeight := y
	if totalHeight < 200 {
		totalHeight = 200
	}

	img := image.NewRGBA(image.Rect(0, 0, reportWidth, totalHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	cursor := pagePadding

	// cabeçalho
	drawLeftText(img, titleFace, d.Title, pagePadding, cursor+headerTitleSz, color.Black)
	cursor += lineHeight + 8
	drawLeftText(img, metaFace, d.Generated, pagePadding, cursor+headerMetaSz-4, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	cursor += lineHeight
	cursor += 16

	// um subtítulo + desenho por aba com foto
	for i, block := range d.Blocks {
		if block.Drawing == nil {
			continue
		}
		drawLeftText(img, subtitleFace, block.Name, pagePadding, cursor+subtitleTextSz, color.Black)
		cursor += lineHeight + 4

		layout := layouts[i]
		// O teto de tempo só vale se for verificado DENTRO do trabalho: com um
		// check só na entrada, 10 abas de 50 regiões rodavam 12 s depois do
		// prazo estourado.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scaled := image.NewRGBA(image.Rect(0, 0, layout.width, layout.height))
		xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), block.Drawing, block.Drawing.Bounds(), xdraw.Over, nil)
		offsetX := pagePadding + (innerWidth-layout.width)/2
		draw.Draw(img, image.Rect(offsetX, cursor, offsetX+layout.width, cursor+layout.height), scaled, image.Point{}, draw.Over)
		cursor += layout.height + 20
	}

	// tabela de áreas marcadas (única, contínua, com a coluna Aba)
	drawLeftText(img, sectionFace, "Áreas marcadas", pagePadding, cursor+sectionTitleSz, color.Black)
	cursor += lineHeight + 6
	if len(allRows) == 0 {
		drawLeftText(img, bodyFace, "nenhuma região marcada", pagePadding, cursor+bodyTextSz, color.Black)
		cursor += lineHeight
	} else {
		headers := []string{"Pin", "Aba", "Região", "Cor", "Hex", "Tinta", "ΔE00", "Estado", "Anotação"}
		colX := tableColumnOffsets(innerWidth)
		for i, h := range headers {
			drawLeftText(img, bodyFace, h, pagePadding+colX[i], cursor+bodyTextSz, color.Black)
		}
		cursor += tableRowH
		for i, row := range allRows {
			if i%10 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			rowY := cursor + bodyTextSz
			drawLeftText(img, bodyFace, strconv.Itoa(row.N), pagePadding+colX[0], rowY, color.Black)
			abaTxt := truncarNaLargura(bodyFace, sanitizeReportText(row.TabName), colX[2]-colX[1]-colGapPx)
			drawLeftText(img, bodyFace, abaTxt, pagePadding+colX[1], rowY, color.Black)
			nomeRegiao := truncarNaLargura(bodyFace, sanitizeReportText(row.RegionName), colX[3]-colX[2]-colGapPx)
			drawLeftText(img, bodyFace, nomeRegiao, pagePadding+colX[2], rowY, color.Black)
			swatchX := pagePadding + colX[3]
			swatchY := cursor + (tableRowH-colorSwatchPx)/2
			draw.Draw(img, image.Rect(swatchX, swatchY, swatchX+colorSwatchPx, swatchY+colorSwatchPx), image.NewUniform(parseHexColor(row.Hex)), image.Point{}, draw.Src)
			drawLeftText(img, bodyFace, row.Hex, pagePadding+colX[4], rowY, color.Black)
			paintLabel := row.PaintLabel
			if strings.TrimSpace(paintLabel) == "" {
				paintLabel = "sem equivalente"
			}
			paintLabel = truncarNaLargura(bodyFace, sanitizeReportText(paintLabel), colX[6]-colX[5]-colGapPx)
			drawLeftText(img, bodyFace, paintLabel, pagePadding+colX[5], rowY, color.Black)
			drawLeftText(img, bodyFace, formatDeltaE(row.DeltaE), pagePadding+colX[6], rowY, color.Black)
			state := "a pintar"
			if row.Painted {
				state = "pintada"
			}
			drawLeftText(img, bodyFace, state, pagePadding+colX[7], rowY, color.Black)
			nota := truncarNaLargura(bodyFace, sanitizeReportText(row.Note), innerWidth-colX[8])
			drawLeftText(img, bodyFace, nota, pagePadding+colX[8], rowY, color.Black)
			cursor += tableRowH
		}
	}
	cursor += 24

	// lista de tintas da peça
	drawLeftText(img, sectionFace, "Tintas da peça", pagePadding, cursor+sectionTitleSz, color.Black)
	cursor += lineHeight + 6
	if len(d.Paints) == 0 {
		drawLeftText(img, bodyFace, "nenhuma região marcada", pagePadding, cursor+bodyTextSz, color.Black)
	} else {
		for _, paint := range d.Paints {
			line := sanitizeReportText(formatPaintLine(paint))
			drawLeftText(img, bodyFace, line, pagePadding, cursor+bodyTextSz, color.Black)
			cursor += lineHeight
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, ErrReportFailed
	}
	if buf.Len() > maxReportOutputBytes {
		return nil, ErrReportFailed
	}
	return buf.Bytes(), nil
}

func tableColumnOffsets(innerWidth int) [9]int {
	// proporções fixas somando innerWidth; a anotação leva o excedente.
	// Pin, Aba, Região, Cor, Hex, Tinta, ΔE00, Estado, Anotação.
	weights := [9]float64{0.05, 0.09, 0.13, 0.04, 0.09, 0.17, 0.06, 0.09, 0.28}
	var offsets [9]int
	acc := 0.0
	for i, w := range weights {
		offsets[i] = int(acc * float64(innerWidth))
		acc += w
	}
	return offsets
}

func formatDeltaE(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	// ΔE00 não é validado na escrita do plano: valor absurdo vindo do POST
	// viraria string de 300 caracteres e cobriria as colunas seguintes.
	if v < 0 || v > 999 {
		return "-"
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func formatPinList(pins []int) string {
	if len(pins) == 0 {
		return "pin -"
	}
	parts := make([]string, 0, len(pins))
	for _, p := range pins {
		parts = append(parts, "pin "+strconv.Itoa(p))
	}
	return strings.Join(parts, ", ")
}

// sanitizeReportText remove caracteres de controle e quebras de linha antes
// de qualquer texto ser desenhado (superfície de injeção: CAN4).
func sanitizeReportText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(' ')
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// colGapPx é a folga mínima entre o fim de um texto e o início da coluna
// seguinte — sem ela um nome de tinta longo invade a coluna do ΔE00.
const colGapPx = 8

// truncarNaLargura corta o texto no limite de pixels da coluna, terminando em
// reticências. Fonte proporcional não permite corte por contagem de runas.
func truncarNaLargura(face font.Face, texto string, larguraMax int) string {
	if larguraMax <= 0 || font.MeasureString(face, texto).Round() <= larguraMax {
		return texto
	}
	const reticencias = "…"
	larguraRet := font.MeasureString(face, reticencias).Round()
	runas := []rune(texto)
	for i := len(runas) - 1; i > 0; i-- {
		if font.MeasureString(face, string(runas[:i])).Round()+larguraRet <= larguraMax {
			return string(runas[:i]) + reticencias
		}
	}
	return reticencias
}

func drawLeftText(img *image.RGBA, face font.Face, text string, x, baselineY int, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(baselineY),
		},
	}
	d.DrawString(text)
}

// fitWithin calcula largura/altura reduzidas para caber em (maxW, maxH)
// preservando a proporção original (RN8/RN9), sem jamais ampliar.
func fitWithin(w, h, maxW, maxH int) (int, int) {
	if w <= 0 || h <= 0 {
		return maxW, maxH
	}
	scale := 1.0
	if w > maxW {
		scale = float64(maxW) / float64(w)
	}
	if scaledH := float64(h) * scale; scaledH > float64(maxH) {
		scale = float64(maxH) / float64(h)
	}
	newW := int(math.Round(float64(w) * scale))
	newH := int(math.Round(float64(h) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return newW, newH
}
