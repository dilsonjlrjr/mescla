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

// maxReportOutputBytes é o teto de saída do relatório. Subiu de 10 para 25 MB
// no rf-09 (RN13), porque o relatório passou a ter um bloco de desenho por
// aba em vez de um só. Acima disso a geração é abortada com ErrReportFailed.
const maxReportOutputBytes = 25 * 1024 * 1024

// reportRow é uma linha da tabela de áreas marcadas do relatório (RN2, RN4,
// RN11-tabela do rf-08). TabName é a aba de origem (RN13 do rf-09): a tabela
// do relatório é única e contínua, mas cada linha carrega o nome da figura
// de onde veio.
type reportRow struct {
	N          int
	TabName    string
	RegionName string
	Hex        string
	R, G, B    int
	PaintLabel string // "Citadel Khorne Red (22-14)" ou "sem equivalente"
	DeltaE     float64
	Painted    bool
	// ForaDoUniverso (rf-11 RN7): a tinta veio de fora do fabricante base ou
	// do estoque que o usuário havia pedido — aparece marcada na coluna Estado.
	ForaDoUniverso bool
	Note           string
}

// reportPaintTab é a ocorrência de uma tinta dentro de uma aba específica:
// o nome da aba e os pins, dentro dela, onde a tinta aparece.
type reportPaintTab struct {
	Tab  string
	Pins []int
}

// reportPaint é uma entrada da lista de tintas do plano (RN5 do rf-08, RN7
// do rf-09): uma linha por tinta distinta (paintBrand+paintCode+paintName),
// citando as abas — e, dentro de cada uma, os pins — onde é usada.
type reportPaint struct {
	Label string // "Khorne Red — Citadel (22-14)"
	Tabs  []reportPaintTab
}

// addOccurrence registra que a tinta apareceu no pin da aba dada, agrupando
// por aba (uma tinta usada duas vezes na mesma aba cita os dois pins juntos).
func (p *reportPaint) addOccurrence(tabName string, pin int) {
	for i := range p.Tabs {
		if p.Tabs[i].Tab == tabName {
			p.Tabs[i].Pins = append(p.Tabs[i].Pins, pin)
			return
		}
	}
	p.Tabs = append(p.Tabs, reportPaintTab{Tab: tabName, Pins: []int{pin}})
}

// reportTabBlock é o bloco de relatório de uma aba (RN13 do rf-09): nome da
// figura, seu desenho — nil quando a aba não tem foto ou a foto não decodifica
// (RN7 do rf-08) — e as linhas de tabela da aba, já com os pins reiniciados
// em 1 dentro dela.
type reportTabBlock struct {
	Name    string
	Drawing *image.RGBA
	Rows    []reportRow
}

// reportData é a composição pronta para desenho, comum ao PDF e ao PNG.
type reportData struct {
	Title     string // nome do plano
	Generated string // "06/09/2026"
	Blocks    []reportTabBlock
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

	data.Blocks = make([]reportTabBlock, 0, len(plan.Tabs))
	for _, t := range plan.Tabs {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, "", ctxErr
		}
		data.Blocks = append(data.Blocks, buildTabBlock(t))
	}

	data.Paints = buildReportPaints(plan.Tabs)

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

// buildTabBlock monta o bloco de relatório de uma aba (RN13): o desenho com
// os pins reiniciados em 1 — nil quando a aba não tem foto ou a foto falha ao
// decodificar (RN7 do rf-08, sem abortar o relatório) — e as linhas de tabela
// da aba, já citando o nome dela.
func buildTabBlock(tab PaintingTabDTO) reportTabBlock {
	name := sanitizeText(tab.Name)
	block := reportTabBlock{Name: name}

	if tab.ImageData != "" {
		if img, decErr := decodePlanImage(tab.ImageData); decErr == nil {
			pins := buildPinsForDrawing(tab.Regions)
			block.Drawing = drawPins(img, pins)
		}
	}

	block.Rows = buildReportRows(name, tab.Regions)
	return block
}

// buildPinsForDrawing converte as regiões de uma aba (já ordenadas por
// sort_order pelo LoadPlan) nos pins numerados a partir de 1 (RN2/RN3 do
// rf-08). Chamada uma vez por aba, a numeração reinicia a cada chamada — é
// assim que a RN13 do rf-09 (pins reiniciados por aba) se cumpre.
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

// buildReportRows monta as linhas de tabela de uma aba (RN2, RN4, RN11-estado
// do rf-08), com TabName preenchido (RN13 do rf-09) e a numeração de pin
// reiniciada em 1 para essa aba.
func buildReportRows(tabName string, regions []PaintingRegionDTO) []reportRow {
	rows := make([]reportRow, 0, len(regions))
	for i, r := range regions {
		// A ausência de tinta é medida pelo texto, não por `paint_id`: receita
		// de mistura chega com paintId nulo e paintName "A + B" (rf-07), e
		// amarrar em paintId apagaria a tinta real do relatório.
		label := formatPaintLabel(r.PaintBrand, r.PaintName, r.PaintCode)
		rows = append(rows, reportRow{
			N:              i + 1,
			TabName:        tabName,
			RegionName:     sanitizeText(r.RegionName),
			Hex:            r.Hex,
			R:              int(r.R),
			G:              int(r.G),
			B:              int(r.B),
			PaintLabel:     label,
			DeltaE:         r.DeltaE,
			Painted:        r.Painted != 0,
			ForaDoUniverso: r.ForaDoUniverso != 0,
			Note:           sanitizeText(r.Note),
		})
	}
	return rows
}

// buildReportPaints agrupa por paintBrand+paintCode+paintName atravessando
// todas as abas do plano (RN7 do rf-09): a mesma tinta usada em várias abas
// aparece uma vez só, citando cada aba e os pins dela.
func buildReportPaints(tabs []PaintingTabDTO) []reportPaint {
	type key struct{ brand, code, name string }
	order := make([]key, 0)
	byKey := make(map[key]*reportPaint)

	for _, t := range tabs {
		tabName := sanitizeText(t.Name)
		for i, r := range t.Regions {
			label := formatPaintLabel(r.PaintBrand, r.PaintName, r.PaintCode)
			if label == semEquivalente {
				continue
			}
			k := key{brand: r.PaintBrand, code: r.PaintCode, name: r.PaintName}
			p, ok := byKey[k]
			if !ok {
				p = &reportPaint{Label: label}
				byKey[k] = p
				order = append(order, k)
			}
			p.addOccurrence(tabName, i+1)
		}
	}

	paints := make([]reportPaint, 0, len(order))
	for _, k := range order {
		// `order` só recebe chave no mesmo passo em que `byKey` recebe o
		// ponteiro, então a busca não falha — mas desreferenciar sem checar
		// deixaria um nil panic latente se alguém mexesse na ordem disso.
		if p, ok := byKey[k]; ok && p != nil {
			paints = append(paints, *p)
		}
	}
	return paints
}

// formatPaintLine monta a linha de uma tinta na lista final do relatório,
// citando as abas — e, dentro de cada uma, os pins — onde ela é usada
// (RN7 do rf-09). Usada pelo PDF e pelo PNG.
func formatPaintLine(p reportPaint) string {
	parts := make([]string, 0, len(p.Tabs))
	for _, t := range p.Tabs {
		parts = append(parts, fmt.Sprintf("%s (%s)", t.Tab, formatPinList(t.Pins)))
	}
	return fmt.Sprintf("%s — %s", p.Label, strings.Join(parts, "; "))
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

// renderPDF monta o PDF A4 retrato do relatório (RN8 do rf-08): margem 15mm,
// cabeçalho com título e data, um bloco de desenho por aba — com o nome dela
// como subtítulo, reduzido preservando proporção — seguido da tabela única e
// contínua (coluna Aba incluída, RN13 do rf-09), paginada e repetindo o
// cabeçalho em cada página (RN8/CA15), e por fim a lista de tintas. Fonte
// core do fpdf é cp1252, então todo texto passa pelo tradutor.
func renderPDF(ctx context.Context, d reportData) ([]byte, error) {
	const margin = 15.0

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")

	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 2*margin

	// #, Aba, Região, Cor, Tinta, Delta E00, Estado, Nota
	widths := []float64{8, 22, 28, 18, 37, 14, 17, 36}
	// A fonte core do fpdf é cp1252, que não tem Δ — "ΔE00" sairia ".E00".
	headers := []string{"#", "Aba", "Região", "Cor", "Tinta", "Delta E00", "Estado", "Nota"}

	drawTableHeader := func() {
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(230, 230, 230)
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

	for bi, block := range d.Blocks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if block.Drawing == nil {
			continue
		}

		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(usableW, 7, tr(block.Name), "", 1, "L", false, 0, "")

		var buf bytes.Buffer
		if err := png.Encode(&buf, block.Drawing); err == nil {
			bounds := block.Drawing.Bounds()
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
				imgName := fmt.Sprintf("plano-desenho-%d", bi)
				pdf.RegisterImageOptionsReader(imgName, opt, &buf)
				pdf.ImageOptions(imgName, pdf.GetX(), pdf.GetY(), drawW, drawH, true, opt, 0, "")
			}
		}
		pdf.Ln(4)
	}

	allRows := make([]reportRow, 0)
	for _, block := range d.Blocks {
		allRows = append(allRows, block.Rows...)
	}

	if len(allRows) == 0 {
		pdf.SetFont("Arial", "", 11)
		pdf.CellFormat(usableW, 7, tr("Nenhuma região marcada"), "", 1, "L", false, 0, "")
	} else {
		drawTableHeader()
		for _, row := range allRows {
			// RN13 do rf-08: checagem a cada linha.
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if pdf.GetY() > 297-margin-14 {
				pdf.AddPage()
				// RN13 do rf-08: checagem a cada página.
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				drawTableHeader()
			}
			estado := "a pintar"
			if row.Painted {
				estado = "pintada"
			}
			if row.ForaDoUniverso {
				// RN7 do rf-11: a lista de compras não pode esconder que essa
				// tinta veio de fora do que o usuário pediu.
				estado += " (fora do pedido)"
			}
			pdf.CellFormat(widths[0], 7, strconv.Itoa(row.N), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[1], 7, cortar(pdf, tr(row.TabName), widths[1]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(widths[2], 7, cortar(pdf, tr(row.RegionName), widths[2]), "1", 0, "L", false, 0, "")
			corX, corY := pdf.GetX(), pdf.GetY()
			pdf.CellFormat(widths[3], 7, cortar(pdf, tr(row.Hex), widths[3]-6), "1", 0, "R", false, 0, "")
			// Amostra da cor alvo (escopo Faz): quadrado à esquerda do hex.
			cr, cg, cb := hexParaRGB(row.Hex)
			pdf.SetFillColor(cr, cg, cb)
			pdf.Rect(corX+1.5, corY+2, 4, 3, "F")
			pdf.SetFillColor(240, 240, 240)
			pdf.CellFormat(widths[4], 7, cortar(pdf, tr(row.PaintLabel), widths[4]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(widths[5], 7, formatDeltaE(row.DeltaE), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[6], 7, tr(estado), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[7], 7, cortar(pdf, tr(row.Note), widths[7]), "1", 0, "L", false, 0, "")
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
			pdf.CellFormat(usableW, 6, tr(formatPaintLine(p)), "", 1, "L", false, 0, "")
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
