package mix

import (
	"math"
	"testing"
	"time"

	"paint-match-ai/api/domain/color"
)

func tinta(nome string, r, g, b uint8) PaintInput {
	return PaintInput{Name: nome, R: r, G: g, B: b}
}

func lab(r, g, b uint8) [3]float64 {
	l, a, bb := color.RGBToLab(r, g, b)
	return [3]float64{l, a, bb}
}

// CA1 — o teste que justifica o requisito inteiro: amarelo com azul dá VERDE
// na bancada. A média ponderada de RGB previa cinza.
func TestCA1AmareloComAzulDaVerde(t *testing.T) {
	amarelo := tinta("Amarelo", 0xF2, 0xE2, 0x00)
	azul := tinta("Azul", 0x1B, 0x3A, 0x8C)

	r, g, b := MisturaSubtrativa([]PaintInput{amarelo, azul}, []float64{50, 50})

	if !(g > r && g > b) {
		t.Fatalf("mistura deveria ser verde (G maior que R e B), veio #%02X%02X%02X", r, g, b)
	}
}

// CA2 — caracterização da mudança. Com amarelo e azul PUROS a média
// aritmética dá exatamente cinza (128,128,128): é o caso que prova que o
// modelo antigo estava errado por construção, não por imprecisão.
func TestCA2ModeloAntigoDariaCinza(t *testing.T) {
	amarelo := tinta("Amarelo puro", 255, 255, 0)
	azul := tinta("Azul puro", 0, 0, 255)

	// Média ponderada de RGB — exatamente o que o Mix fazia antes do rf-10.
	mediaR := (float64(amarelo.R) + float64(azul.R)) / 2
	mediaG := (float64(amarelo.G) + float64(azul.G)) / 2
	mediaB := (float64(amarelo.B) + float64(azul.B)) / 2

	if mediaR != 127.5 || mediaG != 127.5 || mediaB != 127.5 {
		t.Fatalf("o modelo antigo dava cinza para este par, veio (%.1f,%.1f,%.1f)", mediaR, mediaG, mediaB)
	}

	// O modelo novo NÃO devolve cinza para este par. Com primárias puras — que
	// não existem como pigmento, porque absorvem 100% de um canal — a resposta
	// subtrativa é uma cor bem escura, não o cinza médio da média aritmética.
	// A prova de que amarelo com azul dá verde está na CA1, com valores de
	// tinta real.
	r, g, b := MisturaSubtrativa([]PaintInput{amarelo, azul}, []float64{50, 50})
	if r == g && g == b && r > 100 {
		t.Fatalf("o modelo novo não pode repetir o cinza médio do antigo, veio #%02X%02X%02X", r, g, b)
	}
}

// CA3 — mistura subtrativa escurece mais que a média aritmética.
func TestCA3BrancoComPretoEscurece(t *testing.T) {
	branco := tinta("Branco", 255, 255, 255)
	preto := tinta("Preto", 0, 0, 0)

	r, g, b := MisturaSubtrativa([]PaintInput{branco, preto}, []float64{50, 50})

	if r != g || g != b {
		t.Fatalf("branco com preto tem de sair neutro, veio #%02X%02X%02X", r, g, b)
	}
	if r >= 128 {
		t.Fatalf("subtrativo escurece mais que a média (128), veio %d", r)
	}
}

// CA4 — pool de uma tinta só: 100% dela.
func TestCA4UmaTintaSo(t *testing.T) {
	e := NewEngine()
	unica := tinta("Vermelho", 0x7A, 0x1F, 0x2B)

	rec := e.SuggestRecipe(lab(0x50, 0x30, 0x30), []PaintInput{unica})

	if len(rec.Ingredients) != 1 || rec.Ingredients[0].Percentage != 100 {
		t.Fatalf("esperado 100%% de uma tinta, veio %+v", rec.Ingredients)
	}
}

// CA5 e CA6 — pool grande, sem teto de contagem, nenhum componente abaixo de 5%.
func TestCA5CA6PoolGrandeSemTetoENadaAbaixoDe5(t *testing.T) {
	e := NewEngine()
	pool := poolSintetico(30)

	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), pool, 0)

	if len(rec.Ingredients) == 0 {
		t.Fatal("receita não pode sair vazia com pool de 30 tintas")
	}
	for _, ing := range rec.Ingredients {
		if ing.Percentage < 5 {
			t.Fatalf("componente abaixo do mínimo praticável: %s com %.1f%%", ing.Paint.Name, ing.Percentage)
		}
	}
}

// CA7 — proporções em múltiplos de 5 e soma exata de 100.
func TestCA7ProporcoesEmPassosDe5Somando100(t *testing.T) {
	e := NewEngine()

	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), poolSintetico(12), 0)

	var soma float64
	for _, ing := range rec.Ingredients {
		if math.Mod(ing.Percentage, 5) != 0 {
			t.Errorf("%s com %.2f%% não é múltiplo de 5", ing.Paint.Name, ing.Percentage)
		}
		soma += ing.Percentage
	}
	if math.Abs(soma-100) > 0.001 {
		t.Fatalf("as proporções têm de somar 100, somaram %.2f", soma)
	}
}

// CA8 — tinta idêntica ao alvo vence qualquer mistura de mesmo ΔE00.
func TestCA8TintaExataVenceMistura(t *testing.T) {
	e := NewEngine()
	alvo := tinta("Alvo", 0x4A, 0x7C, 0x2F)
	pool := append(poolSintetico(10), alvo)

	rec := e.SuggestRecipeLimited(lab(alvo.R, alvo.G, alvo.B), pool, 0)

	if len(rec.Ingredients) != 1 {
		t.Fatalf("com a tinta exata no pool a receita é de uma tinta, veio %d", len(rec.Ingredients))
	}
	if rec.Ingredients[0].Paint.Name != "Alvo" {
		t.Fatalf("a tinta escolhida deveria ser a exata, veio %q", rec.Ingredients[0].Paint.Name)
	}
}

// CA9 — teto explícito continua respeitado.
func TestCA9TetoExplicitoRespeitado(t *testing.T) {
	e := NewEngine()

	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), poolSintetico(20), 2)

	if len(rec.Ingredients) > 2 {
		t.Fatalf("teto de 2 componentes desrespeitado: %d", len(rec.Ingredients))
	}
}

// CA10 — pool de 200 tintas resolve dentro do teto de tempo.
func TestCA10PoolDe200RespondeDentroDoTeto(t *testing.T) {
	e := NewEngine()

	ini := time.Now()
	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), poolSintetico(200), 0)
	decorrido := time.Since(ini)

	if decorrido > 3*time.Second {
		t.Fatalf("demorou %v, o teto é 2s (com folga de medição)", decorrido)
	}
	if len(rec.Ingredients) == 0 {
		t.Fatal("receita vazia com pool de 200")
	}
}

// CA11 — pool vazio não estoura.
func TestCA11PoolVazio(t *testing.T) {
	e := NewEngine()

	rec := e.SuggestRecipe(lab(0x4A, 0x7C, 0x2F), nil)

	if rec.Method != "none" {
		t.Fatalf("pool vazio deveria devolver Method none, veio %q", rec.Method)
	}
}

// CA12 — preto absoluto não divide por zero nem gera NaN.
func TestCA12PretoAbsolutoNaoGeraNaN(t *testing.T) {
	preto := tinta("Preto", 0, 0, 0)

	r, g, b := MisturaSubtrativa([]PaintInput{preto, tinta("Outro", 10, 10, 10)}, []float64{50, 50})

	if r > 20 || g > 20 || b > 20 {
		t.Fatalf("mistura de pretos deveria ficar escura, veio #%02X%02X%02X", r, g, b)
	}
}

// CA13 — o solver é determinístico.
func TestCA13SolverDeterministico(t *testing.T) {
	e := NewEngine()
	pool := poolSintetico(25)
	alvo := lab(0x4A, 0x7C, 0x2F)

	primeira := e.SuggestRecipeLimited(alvo, pool, 0)
	segunda := e.SuggestRecipeLimited(alvo, pool, 0)

	if len(primeira.Ingredients) != len(segunda.Ingredients) {
		t.Fatalf("contagem diferente entre execuções: %d e %d", len(primeira.Ingredients), len(segunda.Ingredients))
	}
	for i := range primeira.Ingredients {
		if primeira.Ingredients[i].Paint.Name != segunda.Ingredients[i].Paint.Name ||
			primeira.Ingredients[i].Percentage != segunda.Ingredients[i].Percentage {
			t.Fatalf("ingrediente %d divergiu entre execuções", i)
		}
	}
}

// CAN1 — proporção NaN ou Inf não contamina o resultado.
func TestCAN1ProporcaoNaNouInfNaoContamina(t *testing.T) {
	paints := []PaintInput{tinta("A", 200, 50, 50), tinta("B", 50, 200, 50)}

	r, g, b := MisturaSubtrativa(paints, []float64{math.NaN(), 50})
	if r == 0 && g == 0 && b == 0 {
		t.Fatal("NaN numa proporção não pode zerar a mistura inteira")
	}

	r2, g2, b2 := MisturaSubtrativa(paints, []float64{math.Inf(1), 50})
	if r2 == 0 && g2 == 0 && b2 == 0 {
		t.Fatal("Inf numa proporção não pode zerar a mistura inteira")
	}
}

// CAN2 — pool absurdo respeita o teto de tempo e não explode.
func TestCAN2PoolDe5000RespeitaOTeto(t *testing.T) {
	e := NewEngine()

	ini := time.Now()
	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), poolSintetico(5000), 0)
	decorrido := time.Since(ini)

	if decorrido > 5*time.Second {
		t.Fatalf("pool de 5000 demorou %v — o teto de 2s não está segurando", decorrido)
	}
	if rec.Method == "" {
		t.Fatal("receita sem método com pool grande")
	}
}

// CAN3 — proporção negativa conta como zero, nunca inverte a mistura.
func TestCAN3ProporcaoNegativaContaZero(t *testing.T) {
	paints := []PaintInput{tinta("A", 200, 50, 50), tinta("B", 50, 200, 50)}

	comNegativo, g1, _ := MisturaSubtrativa(paints, []float64{-50, 50})
	soB, g2, _ := MisturaSubtrativa(paints, []float64{0, 50})

	if comNegativo != soB || g1 != g2 {
		t.Fatalf("proporção negativa deveria contar como zero: #%02X%02X.. contra #%02X%02X..", comNegativo, g1, soB, g2)
	}
}

// CAN4 — tinta duplicada no pool não vira receita com a mesma tinta duas vezes.
func TestCAN4TintaDuplicadaNaoApareceDuasVezes(t *testing.T) {
	e := NewEngine()
	repetida := tinta("Repetida", 0x4A, 0x7C, 0x2F)

	rec := e.SuggestRecipeLimited(lab(0x4A, 0x7C, 0x2F), []PaintInput{repetida, repetida}, 0)

	if len(rec.Ingredients) > 1 {
		t.Fatalf("a mesma tinta não pode aparecer duas vezes na receita: %+v", rec.Ingredients)
	}
}

// poolSintetico gera n tintas espalhadas pelo espaço de cor, de forma
// determinística — nada de aleatório em teste.
func poolSintetico(n int) []PaintInput {
	pool := make([]PaintInput, 0, n)
	for i := 0; i < n; i++ {
		r := uint8((i * 37) % 256)
		g := uint8((i * 91) % 256)
		b := uint8((i * 143) % 256)
		pool = append(pool, PaintInput{ID: int64(i + 1), Name: string(rune('A'+i%26)) + string(rune('0'+i/26)), R: r, G: g, B: b})
	}
	return pool
}
