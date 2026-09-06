package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-pdf/fpdf"
)

// ErrPlanNotFound é devolvido quando o plano pedido não existe (rf-08, CA5).
var ErrPlanNotFound = errors.New("plano não encontrado")

// ErrReportFailed cobre qualquer falha de geração: mensagem fixa, sem texto
// interno do servidor na resposta (rf-08, evita repetir o achado S-001).
var ErrReportFailed = errors.New("não foi possível gerar o relatório")

// maxReportOutputBytes é o teto de saída do relatório (rf-08, seção "Limites").
// Acima disso a geração é abortada com ErrReportFailed.
const maxReportOutputBytes = 10 * 1024 * 1024

// reportRow é uma linha da tabela de áreas marcadas do relatório (RN2, RN4, RN11-tabela).
type reportRow struct {
	N          int
	RegionName string
	Hex        string
	R, G, B    int
	PaintLabel string // "Citadel Khorne Red (22-14)" ou "sem equivalente"
	DeltaE     float64
	Painted    bool
	Note       string
}

// reportPaint é uma entrada da lista de tintas da peça (RN5): uma linha por
// tinta distinta (paintBrand+paintCode), citando os pins onde é usada.
type reportPaint struct {
	Label string // "Khorne Red — Citadel (22-14)"
	Pins  []int
}

// reportData é a composição pronta para desenho, comum ao PDF e ao PNG.
type reportData struct {
	Title     string      // nome do plano
	Generated string      // "06/09/2026"
	Drawing   *image.RGBA // nil quando não há imagem utilizável (RN7)
	Rows      []reportRow
	Paints    []reportPaint
}

// BuildPlanReport carrega o plano, monta o relatório e devolve o corpo pronto
// para download. format é "pdf" ou "png" (já validado pelo handler).
func (s *PaintService) BuildPlanReport(ctx context.Context, id int64, format string) (body []byte, filename string, err error) {
	plan, err := s.LoadPlan(id)
	if err != nil {
		// LoadPlan devolve erro genérico ("plano não encontrado (id %d)": %w"
		// não usado ali, é fmt.Errorf simples) — o único ponto de tradução
		// para o sentinela de rota fica aqui, por prefixo, porque LoadPlan é
		// usado por outras chamadas que não precisam distinguir a causa.
		if strings.HasPrefix(err.Error(), "plano não encontrado") {
			return nil, "", ErrPlanNotFound
		}
		return nil, "", ErrReportFailed
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, "", ctxErr
	}

	agora := time.Now()
	data := reportData{
		Title:     sanitizeText(plan.Name),
		Generated: agora.Format("02/01/2006"),
	}

	if plan.ImageData != "" {
		if img, decErr := decodePlanImage(plan.ImageData); decErr == nil {
			pins := buildPinsForDrawing(plan.Regions)
			data.Drawing = drawPins(img, pins)
		}
		// RN7: decodificação falha é ignorada — Drawing continua nil, sem
		// abortar o relatório.
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, "", ctxErr
	}

	data.Rows = buildReportRows(plan.Regions)
	data.Paints = buildReportPaints(plan.Regions)

	var out []byte
	switch format {
	case "png":
		out, err = renderPNG(ctx, data)
	default:
		out, err = renderPDF(ctx, data)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, "", err
		}
		return nil, "", ErrReportFailed
	}

	if len(out) > maxReportOutputBytes {
		return nil, "", ErrReportFailed
	}

	slug := SlugPlano(plan.Name)
	filename = fmt.Sprintf("mescla-%s-%s.%s", slug, agora.Format("2006-01-02"), format)

	return out, filename, nil
}

// buildPinsForDrawing converte as regiões (já ordenadas por sort_order pelo
// LoadPlan) nos pins numerados a partir de 1 (RN2, RN3).
func buildPinsForDrawing(regions []PaintingRegionDTO) []pinDesenho {
	pins := make([]pinDesenho, 0, len(regions))
	for i, r := range regions {
		pins = append(pins, pinDesenho{
			N:   i + 1,
			X:   r.X,
			Y:   r.Y,
			Hex: r.Hex,
		})
	}
	return pins
}

// buildReportRows monta a tabela de áreas marcadas (RN2, RN4, RN11-estado).
func buildReportRows(regions []PaintingRegionDTO) []reportRow {
	rows := make([]reportRow, 0, len(regions))
	for i, r := range regions {
		// A ausência de tinta é medida pelo texto, não por `paint_id`: receita
		// de mistura chega com paintId nulo e paintName "A + B" (rf-07), e
		// amarrar em paintId apagaria a tinta real do relatório.
		label := formatPaintLabel(r.PaintBrand, r.PaintName, r.PaintCode)
		rows = append(rows, reportRow{
			N:          i + 1,
			RegionName: sanitizeText(r.RegionName),
			Hex:        r.Hex,
			R:          int(r.R),
			G:          int(r.G),
			B:          int(r.B),
			PaintLabel: label,
			DeltaE:     r.DeltaE,
			Painted:    r.Painted != 0,
			Note:       sanitizeText(r.Note),
		})
	}
	return rows
}

// buildReportPaints agrupa por paintBrand+paintCode (RN5): a mesma tinta usada
// em várias regiões aparece uma vez, citando todos os pins.
func buildReportPaints(regions []PaintingRegionDTO) []reportPaint {
	type key struct{ brand, code string }
	order := make([]key, 0)
	byKey := make(map[key]*reportPaint)

	for i, r := range regions {
		label := formatPaintLabel(r.PaintBrand, r.PaintName, r.PaintCode)
		if label == semEquivalente {
			continue
		}
		k := key{brand: r.PaintBrand, code: r.PaintCode + "|" + r.PaintName}
		p, ok := byKey[k]
		if !ok {
			p = &reportPaint{Label: label}
			byKey[k] = p
			order = append(order, k)
		}
		p.Pins = append(p.Pins, i+1)
	}

	paints := make([]reportPaint, 0, len(order))
	for _, k := range order {
		paints = append(paints, *byKey[k])
	}
	return paints
}

// alturaMaximaDesenhoMM limita o bloco do desenho no PDF: sem teto, uma foto
// muito mais alta que larga geraria dezenas de páginas em branco.
const alturaMaximaDesenhoMM = 150

// cortar encurta o texto até caber na largura da célula, terminando em "...".
// CellFormat não corta: o texto invade a coluna seguinte.
func cortar(pdf *fpdf.Fpdf, texto string, larguraMM float64) string {
	limite := larguraMM - 2
	if limite <= 0 || pdf.GetStringWidth(texto) <= limite {
		return texto
	}
	runas := []rune(texto)
	for i := len(runas) - 1; i > 0; i-- {
		corte := string(runas[:i]) + "..."
		if pdf.GetStringWidth(corte) <= limite {
			return corte
		}
	}
	return ""
}

// hexParaRGB lê "#RRGGBB"; hex inválido vira branco, para a amostra nunca
// mentir sobre a cor.
func hexParaRGB(hex string) (int, int, int) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 255, 255, 255
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 255, 255, 255
	}
	return int(v>>16) & 0xFF, int(v>>8) & 0xFF, int(v) & 0xFF
}

// semEquivalente é o texto da região que não tem tinta nenhuma associada.
const semEquivalente = "sem equivalente"

// formatPaintLabel monta "Nome — Marca (Código)", tolerando campos vazios.
func formatPaintLabel(brand, name, code string) string {
	label := sanitizeText(name)
	if brand != "" {
		if label != "" {
			label += " — " + sanitizeText(brand)
		} else {
			label = sanitizeText(brand)
		}
	}
	if code != "" {
		label += " (" + sanitizeText(code) + ")"
	}
	if label == "" {
		return semEquivalente
	}
	return label
}

// sanitizeText remove CR, LF e demais caracteres de controle antes do texto
// ser desenhado no PDF/PNG (superfície de injeção da spec: título, nome de
// região e anotação).
func sanitizeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\r' || r == '\n' || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// transliterateAccents troca as letras acentuadas mais comuns em PT-BR pelo
// equivalente ASCII, antes do restante da RN10 rodar.
func transliterateAccents(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "A", "À", "A", "Ã", "A", "Â", "A", "Ä", "A",
		"É", "E", "È", "E", "Ê", "E", "Ë", "E",
		"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
		"Ó", "O", "Ò", "O", "Õ", "O", "Ô", "O", "Ö", "O",
		"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
		"Ç", "C", "Ñ", "N",
	)
	return replacer.Replace(s)
}

// SlugPlano aplica a RN10/CAN2: translitera acento primeiro, o resto vira
// "-", colapsa repetições, apara "-" das pontas, corta em 60 e, se o
// resultado ficar vazio, devolve "plano".
func SlugPlano(nome string) string {
	s := transliterateAccents(nome)
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	s = b.String()

	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")

	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}

	if s == "" {
		return "plano"
	}
	return s
}

// renderPDF monta o PDF A4 retrato do relatório (RN8): margem 15mm, cabeçalho
// com título e data, desenho reduzido preservando proporção, tabela paginada
// repetindo o cabeçalho em cada página (RN8/CA15), seguida da lista de
// tintas. Fonte core do fpdf é cp1252, então todo texto passa pelo tradutor.
func renderPDF(ctx context.Context, d reportData) ([]byte, error) {
	const margin = 15.0

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")

	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 2*margin

	drawTableHeader := func() {
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		widths := []float64{10, 35, 20, 45, 15, 20, 35}
		// A fonte core do fpdf é cp1252, que não tem Δ — "ΔE00" sairia ".E00".
		headers := []string{"#", "Região", "Cor", "Tinta", "Delta E00", "Estado", "Nota"}
		for i, h := range headers {
			pdf.CellFormat(widths[i], 7, tr(h), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Arial", "", 8)
	}

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(usableW, 8, tr(d.Title), "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(usableW, 6, tr(d.Generated), "", 1, "L", false, 0, "")
		pdf.Ln(2)
	})

	pdf.AddPage()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if d.Drawing != nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, d.Drawing); err == nil {
			bounds := d.Drawing.Bounds()
			imgW := float64(bounds.Dx())
			imgH := float64(bounds.Dy())
			if imgW > 0 && imgH > 0 {
				drawW := usableW
				drawH := drawW * imgH / imgW
				// Sem teto, uma foto muito alta (100×7999) viraria 14 metros
				// de página. Cabe na página, mantendo a proporção.
				if drawH > alturaMaximaDesenhoMM {
					drawH = alturaMaximaDesenhoMM
					drawW = drawH * imgW / imgH
				}
				opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
				pdf.RegisterImageOptionsReader("plano-desenho", opt, &buf)
				pdf.ImageOptions("plano-desenho", pdf.GetX(), pdf.GetY(), drawW, drawH, true, opt, 0, "")
			}
		}
	}

	pdf.Ln(4)

	if len(d.Rows) == 0 {
		pdf.SetFont("Arial", "", 11)
		pdf.CellFormat(usableW, 7, tr("Nenhuma região marcada"), "", 1, "L", false, 0, "")
	} else {
		drawTableHeader()
		for _, row := range d.Rows {
			// RN13: checagem a cada linha.
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if pdf.GetY() > 297-margin-14 {
				pdf.AddPage()
				// RN13: checagem a cada página.
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				drawTableHeader()
			}
			estado := "a pintar"
			if row.Painted {
				estado = "pintada"
			}
			widths := []float64{10, 35, 20, 45, 15, 20, 35}
			pdf.CellFormat(widths[0], 7, strconv.Itoa(row.N), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[1], 7, cortar(pdf, tr(row.RegionName), widths[1]), "1", 0, "L", false, 0, "")
			corX, corY := pdf.GetX(), pdf.GetY()
			pdf.CellFormat(widths[2], 7, cortar(pdf, tr(row.Hex), widths[2]-6), "1", 0, "R", false, 0, "")
			// Amostra da cor alvo (escopo Faz): quadrado à esquerda do hex.
			cr, cg, cb := hexParaRGB(row.Hex)
			pdf.SetFillColor(cr, cg, cb)
			pdf.Rect(corX+1.5, corY+2, 4, 3, "F")
			pdf.SetFillColor(240, 240, 240)
			pdf.CellFormat(widths[3], 7, cortar(pdf, tr(row.PaintLabel), widths[3]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(widths[4], 7, formatDeltaE(row.DeltaE), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[5], 7, tr(estado), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[6], 7, cortar(pdf, tr(row.Note), widths[6]), "1", 0, "L", false, 0, "")
			pdf.Ln(-1)
		}
	}

	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(usableW, 7, tr("Tintas da peça"), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(d.Paints) == 0 {
		pdf.CellFormat(usableW, 7, tr("Nenhuma região marcada"), "", 1, "L", false, 0, "")
	} else {
		for _, p := range d.Paints {
			pins := make([]string, 0, len(p.Pins))
			for _, n := range p.Pins {
				pins = append(pins, strconv.Itoa(n))
			}
			line := fmt.Sprintf("%s — pins %s", p.Label, strings.Join(pins, ", "))
			pdf.CellFormat(usableW, 6, tr(line), "", 1, "L", false, 0, "")
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
