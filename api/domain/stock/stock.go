// Package stock é a fonte única da lógica do "meu estoque" — as tintas que o
// próprio pintor possui. É neutra de plataforma e de persistência: o desktop
// guarda as tintas em SQLite e o mobile em localStorage, mas AMBOS usam este
// pacote para o parsing/crítica do CSV de importação e para converter o estoque
// no formato que o motor de mistura (api/domain/mix) entende — assim a validação de um
// CSV e a receita "priorizando o estoque" são idênticas nos dois alvos.
//
// Uma tinta de estoque é livre: o pintor digita nome, código e cor, escolhendo
// um fabricante que precisa existir no catálogo. Ela não referencia uma tinta
// do catálogo (pode ser um tom que nem está lá).
package stock

import (
	"encoding/csv"
	"fmt"
	"strings"

	"paint-match-ai/api/domain/color"
	"paint-match-ai/api/domain/mix"
)

// Paint é uma tinta do estoque do usuário. ID é opcional (o desktop usa o
// rowid do SQLite; o mobile um id local) — o parsing de CSV não o preenche.
type Paint struct {
	ID             int64  `json:"id"`
	ManufacturerID int64  `json:"manufacturerId"`
	Manufacturer   string `json:"manufacturer"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	R              uint8  `json:"r"`
	G              uint8  `json:"g"`
	B              uint8  `json:"b"`
	Volume         string `json:"volume"`
	Notes          string `json:"notes"`
}

// Manufacturer é o mínimo que o parser precisa para validar/resolver o
// fabricante de uma linha do CSV.
type Manufacturer struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RowError descreve uma linha do CSV que não passou na crítica. Line é o número
// da linha no arquivo (1-based, contando o cabeçalho), para o usuário localizar.
type RowError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
}

// csvHeader é a ordem canônica das colunas do modelo.
var csvHeader = []string{"fabricante", "nome", "codigo", "hex", "volume", "notas"}

// CSVTemplate devolve o conteúdo do arquivo-modelo, pronto para download. Traz o
// cabeçalho e duas linhas de exemplo para o pintor só substituir os valores.
func CSVTemplate() string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(csvHeader)
	_ = w.Write([]string{"Vallejo", "Model Color Black", "70.950", "#1c1c1c", "17ml", "meu preto de base"})
	_ = w.Write([]string{"Citadel", "Mephiston Red", "", "#9a1115", "12ml", ""})
	w.Flush()
	return b.String()
}

// ToCSV serializa um estoque no mesmo formato do modelo — o inverso de ParseCSV.
// O que sai daqui reentra sem erros (round-trip), então o export serve de backup
// e de base para reimportar em outra máquina.
func ToCSV(paints []Paint) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(csvHeader)
	for _, p := range paints {
		_ = w.Write([]string{p.Manufacturer, p.Name, p.Code, hexOf(p.R, p.G, p.B), p.Volume, p.Notes})
	}
	w.Flush()
	return b.String()
}

// hexOf formata a cor como "#rrggbb".
func hexOf(r, g, b uint8) string {
	const digits = "0123456789abcdef"
	return string([]byte{
		'#',
		digits[r>>4], digits[r&0xf],
		digits[g>>4], digits[g&0xf],
		digits[b>>4], digits[b&0xf],
	})
}

// ParseCSV lê o CSV de importação e valida cada linha contra os fabricantes
// conhecidos. Devolve as tintas que passaram e a lista de erros por linha —
// nunca aborta no primeiro erro, para o usuário ver tudo o que precisa corrigir
// de uma vez. O cabeçalho é opcional: se a primeira linha casar com o modelo,
// é ignorada.
func ParseCSV(csvText string, manufacturers []Manufacturer) ([]Paint, []RowError) {
	// Índice case-insensitive nome→id, sem acento, para casar "vallejo" com
	// "Vallejo" e tolerar espaços extras.
	byName := make(map[string]Manufacturer, len(manufacturers))
	for _, m := range manufacturers {
		byName[normalizeName(m.Name)] = m
	}

	r := csv.NewReader(strings.NewReader(csvText))
	r.FieldsPerRecord = -1 // linhas com nº de colunas variável não abortam o Read
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, []RowError{{Line: 0, Message: "arquivo CSV inválido: " + err.Error()}}
	}

	var paints []Paint
	var errs []RowError
	for i, rec := range records {
		lineNo := i + 1
		// Pula cabeçalho (só na primeira linha) e linhas totalmente vazias.
		if i == 0 && looksLikeHeader(rec) {
			continue
		}
		if isBlank(rec) {
			continue
		}

		p, rowErr := parseRow(rec, byName)
		if rowErr != "" {
			errs = append(errs, RowError{Line: lineNo, Message: rowErr, Raw: strings.Join(rec, ",")})
			continue
		}
		paints = append(paints, p)
	}
	return paints, errs
}

// parseRow valida e monta uma tinta a partir de um registro do CSV. Devolve uma
// mensagem não-vazia quando a linha é inválida.
func parseRow(rec []string, byName map[string]Manufacturer) (Paint, string) {
	get := func(i int) string {
		if i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}

	mfrName, name, code, hex, volume, notes := get(0), get(1), get(2), get(3), get(4), get(5)

	if mfrName == "" {
		return Paint{}, "fabricante em branco"
	}
	mfr, ok := byName[normalizeName(mfrName)]
	if !ok {
		return Paint{}, fmt.Sprintf("fabricante %q não existe no catálogo", mfrName)
	}
	if name == "" {
		return Paint{}, "nome da tinta em branco"
	}
	rgb, ok := ParseHex(hex)
	if !ok {
		return Paint{}, fmt.Sprintf("cor %q inválida — use hex como #1c1c1c", hex)
	}

	return Paint{
		ManufacturerID: mfr.ID,
		Manufacturer:   mfr.Name,
		Name:           name,
		Code:           code,
		R:              rgb.R,
		G:              rgb.G,
		B:              rgb.B,
		Volume:         volume,
		Notes:          notes,
	}, ""
}

// ParseHex aceita "#RRGGBB", "RRGGBB", "#RGB" ou "RGB" e devolve a cor. O
// segundo retorno é falso quando a string não é um hex de cor válido.
func ParseHex(s string) (color.RGB, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		// Forma curta #abc → #aabbcc.
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.RGB{}, false
	}
	var vals [3]uint8
	for i := 0; i < 3; i++ {
		hi, ok1 := hexNibble(s[i*2])
		lo, ok2 := hexNibble(s[i*2+1])
		if !ok1 || !ok2 {
			return color.RGB{}, false
		}
		vals[i] = hi<<4 | lo
	}
	return color.RGB{R: vals[0], G: vals[1], B: vals[2]}, true
}

func hexNibble(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// ToMixInputs converte tintas de estoque no formato do motor de mistura. Cada
// PaintInput carrega o ID da tinta de estoque, para a UI reencontrar a tinta
// (marca, volume) a partir do ingrediente da receita.
func ToMixInputs(paints []Paint) []mix.PaintInput {
	out := make([]mix.PaintInput, 0, len(paints))
	for _, p := range paints {
		out = append(out, mix.PaintInput{
			ID: p.ID, Name: p.Name, Code: p.Code, R: p.R, G: p.G, B: p.B,
			ManufacturerID: p.ManufacturerID, Manufacturer: p.Manufacturer,
		})
	}
	return out
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func looksLikeHeader(rec []string) bool {
	if len(rec) == 0 {
		return false
	}
	return normalizeName(rec[0]) == "fabricante"
}

func isBlank(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}
