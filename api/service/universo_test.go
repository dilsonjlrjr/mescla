package service

import (
	"database/sql"
	"testing"
)

// newUniversoTestService monta um banco com catálogo mínimo (duas marcas com
// tintas coloridas) e estoque do usuário, para exercitar as quatro combinações
// do rf-11.
func newUniversoTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
		CREATE TABLE manufacturers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
		CREATE TABLE paints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			manufacturer_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			code TEXT DEFAULT ''
		);
		CREATE TABLE paint_colors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			paint_id INTEGER NOT NULL,
			rgb_r INTEGER, rgb_g INTEGER, rgb_b INTEGER
		);
		CREATE TABLE paint_types (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
		INSERT INTO manufacturers (id, name) VALUES (1, 'Acrilex'), (2, 'Citadel');

		INSERT INTO paints (id, manufacturer_id, name, code) VALUES
			(1, 1, 'Acrilex Vermelho', 'A-01'),
			(2, 1, 'Acrilex Amarelo',  'A-02'),
			(3, 1, 'Acrilex Sem Cor',  'A-03'),
			(4, 2, 'Citadel Azul',     'C-01'),
			(5, 2, 'Citadel Branco',   'C-02');

		INSERT INTO paint_colors (paint_id, rgb_r, rgb_g, rgb_b) VALUES
			(1, 216, 30, 5),
			(2, 242, 226, 0),
			(4, 63, 72, 204),
			(5, 255, 255, 255);
	`); err != nil {
		t.Fatalf("seed catálogo: %v", err)
	}
	if err := ensureUserSchema(db); err != nil {
		t.Fatalf("ensureUserSchema: %v", err)
	}
	return &PaintService{db: db}
}

func comEstoque(t *testing.T, s *PaintService, tintas ...UserPaintDTO) {
	t.Helper()
	for _, p := range tintas {
		if _, err := s.AddUserPaint(p); err != nil {
			t.Fatalf("AddUserPaint(%s): %v", p.Name, err)
		}
	}
}

// CA1 — nenhum interruptor ligado: o universo é o catálogo inteiro.
func TestCA1UniversoCatalogoInteiro(t *testing.T) {
	s := newUniversoTestService(t)

	pool, err := s.montarUniverso(0, false)
	if err != nil {
		t.Fatalf("montarUniverso: %v", err)
	}
	// 4 tintas COM cor; a "Acrilex Sem Cor" não entra.
	if len(pool) != 4 {
		t.Fatalf("esperado 4 tintas com cor no catálogo, veio %d", len(pool))
	}
	for _, p := range pool {
		if p.Name == "Acrilex Sem Cor" {
			t.Fatal("tinta sem cor cadastrada não pode virar candidato")
		}
	}
}

// CA2 — só estoque: o universo é o que o usuário tem.
func TestCA2UniversoSoEstoque(t *testing.T) {
	s := newUniversoTestService(t)
	comEstoque(t, s,
		UserPaintDTO{ManufacturerID: 1, Name: "Minha Acrilex", R: 200, G: 30, B: 30},
		UserPaintDTO{ManufacturerID: 2, Name: "Minha Citadel", R: 30, G: 30, B: 200},
	)

	pool, err := s.montarUniverso(0, true)
	if err != nil {
		t.Fatalf("montarUniverso: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("esperado 2 tintas do estoque, veio %d", len(pool))
	}
}

// CA3 — só fabricante base: o universo é o catálogo daquela marca.
func TestCA3UniversoSoFabricanteBase(t *testing.T) {
	s := newUniversoTestService(t)

	pool, err := s.montarUniverso(1, false)
	if err != nil {
		t.Fatalf("montarUniverso: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("Acrilex tem 2 tintas com cor, veio %d", len(pool))
	}
}

// CA4 — os dois ligados: INTERSEÇÃO, não união.
func TestCA4UniversoIntersecaoEstoqueEFabricante(t *testing.T) {
	s := newUniversoTestService(t)
	comEstoque(t, s,
		UserPaintDTO{ManufacturerID: 1, Name: "Minha Acrilex A", R: 200, G: 30, B: 30},
		UserPaintDTO{ManufacturerID: 1, Name: "Minha Acrilex B", R: 240, G: 220, B: 10},
		UserPaintDTO{ManufacturerID: 2, Name: "Minha Citadel", R: 30, G: 30, B: 200},
	)

	pool, err := s.montarUniverso(1, true)
	if err != nil {
		t.Fatalf("montarUniverso: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("interseção deveria ter as 2 Acrilex do estoque, veio %d", len(pool))
	}
	for _, p := range pool {
		if p.Name == "Minha Citadel" {
			t.Fatal("tinta de outra marca vazou para a interseção")
		}
	}
}

// CA15 — estoque ligado e vazio não é erro de servidor.
func TestCA15EstoqueVazioNaoEErro(t *testing.T) {
	s := newUniversoTestService(t)

	_, err := s.montarUniverso(0, true)
	var vazio ErrUniversoVazio
	if !asUniversoVazio(err, &vazio) {
		t.Fatalf("esperava ErrUniversoVazio, veio %v", err)
	}
	if vazio.Motivo != "Seu estoque está vazio" {
		t.Fatalf("mensagem inesperada: %q", vazio.Motivo)
	}
}

// CA16 — interseção vazia cita a marca.
func TestCA16IntersecaoVaziaCitaAMarca(t *testing.T) {
	s := newUniversoTestService(t)
	comEstoque(t, s, UserPaintDTO{ManufacturerID: 2, Name: "Só Citadel", R: 30, G: 30, B: 200})

	_, err := s.montarUniverso(1, true)
	var vazio ErrUniversoVazio
	if !asUniversoVazio(err, &vazio) {
		t.Fatalf("esperava ErrUniversoVazio, veio %v", err)
	}
	if vazio.Motivo != "Você não tem tintas Acrilex no estoque" {
		t.Fatalf("a mensagem tem de citar a marca, veio %q", vazio.Motivo)
	}
}

// CA18 / CAN2 — fabricante inexistente é recusado, sem ecoar o valor.
func TestCA18FabricanteInexistenteRecusado(t *testing.T) {
	s := newUniversoTestService(t)

	_, err := s.montarUniverso(999, false)
	if err == nil {
		t.Fatal("fabricante inexistente deveria dar erro")
	}
	if err.Error() != "fabricante não encontrado" {
		t.Fatalf("mensagem não pode ecoar o id recebido, veio %q", err.Error())
	}
}

// CA6, CA7, CA8 — as três faixas de qualidade.
func TestCA6CA7CA8FaixasDeQualidade(t *testing.T) {
	casos := []struct {
		delta    float64
		esperada FaixaQualidade
	}{
		{0, FaixaOtima},
		{2.4, FaixaOtima},
		{3.0, FaixaOtima},
		{3.1, FaixaAproximada},
		{4.5, FaixaAproximada},
		{6.0, FaixaAproximada},
		{6.1, FaixaNaoEncontrei},
		{7.9, FaixaNaoEncontrei},
	}
	for _, c := range casos {
		if got := ClassificarFaixa(c.delta); got != c.esperada {
			t.Errorf("ΔE00 %.1f: esperava %q, veio %q", c.delta, c.esperada, got)
		}
	}
}

// CA5 / CAN3 — a receita sai do universo pedido, e só dele.
func TestCA5CAN3ReceitaSoUsaOUniversoPedido(t *testing.T) {
	s := newUniversoTestService(t)

	// Alvo azul: existe só na Citadel. Com fabricante base Acrilex, nenhum
	// ingrediente pode ser Citadel.
	rec, err := s.ResolverCorNoUniverso(63, 72, 204, 1, false, false, 0)
	if err != nil {
		t.Fatalf("ResolverCorNoUniverso: %v", err)
	}
	if len(rec.Ingredients) == 0 {
		t.Fatal("receita vazia")
	}
	for _, ing := range rec.Ingredients {
		if ing.PaintID == 4 || ing.PaintID == 5 {
			t.Fatalf("ingrediente de fora do universo pedido: %s", ing.Name)
		}
	}
	if rec.ForaDoUniverso {
		t.Error("sem autorização do usuário, a receita não pode ser marcada como fora do universo")
	}
}

// CA13 — a marcação de "fora do universo" é decidida por quem chama.
func TestCA13MarcacaoForaDoUniverso(t *testing.T) {
	s := newUniversoTestService(t)

	rec, err := s.ResolverCorNoUniverso(63, 72, 204, 0, false, true, 0)
	if err != nil {
		t.Fatalf("ResolverCorNoUniverso: %v", err)
	}
	if !rec.ForaDoUniverso {
		t.Fatal("a marcação pedida por quem chama tem de aparecer na resposta")
	}
}

// CA12 — o melhor ΔE00 de cada saída é calculável sem montar a receita.
func TestCA12MelhorDeltaEPorUniverso(t *testing.T) {
	s := newUniversoTestService(t)

	soAcrilex, err := s.MelhorDeltaENoUniverso(63, 72, 204, 1, false)
	if err != nil {
		t.Fatalf("MelhorDeltaENoUniverso(Acrilex): %v", err)
	}
	catalogo, err := s.MelhorDeltaENoUniverso(63, 72, 204, 0, false)
	if err != nil {
		t.Fatalf("MelhorDeltaENoUniverso(catálogo): %v", err)
	}
	if catalogo > soAcrilex {
		t.Fatalf("o catálogo inteiro tem a tinta exata: deveria ser melhor (%.2f) que só Acrilex (%.2f)", catalogo, soAcrilex)
	}
}

// CA20 — duas resoluções iguais dão a mesma receita.
func TestCA20ResolucaoDeterministica(t *testing.T) {
	s := newUniversoTestService(t)

	a, err := s.ResolverCorNoUniverso(100, 140, 60, 0, false, false, 0)
	if err != nil {
		t.Fatalf("primeira: %v", err)
	}
	b, err := s.ResolverCorNoUniverso(100, 140, 60, 0, false, false, 0)
	if err != nil {
		t.Fatalf("segunda: %v", err)
	}
	if len(a.Ingredients) != len(b.Ingredients) || a.DeltaE != b.DeltaE {
		t.Fatal("a mesma entrada tem de dar a mesma receita")
	}
}

func asUniversoVazio(err error, alvo *ErrUniversoVazio) bool {
	v, ok := err.(ErrUniversoVazio)
	if ok {
		*alvo = v
	}
	return ok
}

// CA13 — a marcação persiste no banco: round-trip por SavePlan/LoadPlan.
func TestCA13ForaDoUniversoPersisteNoRoundTrip(t *testing.T) {
	svc := newPlanningTestService(t)

	salvo, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Persistência",
		Tabs: []PaintingTabDTO{{
			Name: "Frente",
			Regions: []PaintingRegionDTO{
				{Hex: "#7A1F2B", PaintBrand: "Citadel", PaintName: "Khorne Red", ForaDoUniverso: 1},
				{Hex: "#C8B15A", PaintBrand: "Citadel", PaintName: "Zamesi Desert", ForaDoUniverso: 0},
			},
		}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	lido, err := svc.LoadPlan(salvo.ID)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	regioes := lido.Tabs[0].Regions
	if regioes[0].ForaDoUniverso != 1 {
		t.Errorf("primeira região deveria voltar marcada, veio %d", regioes[0].ForaDoUniverso)
	}
	if regioes[1].ForaDoUniverso != 0 {
		t.Errorf("segunda região não deveria estar marcada, veio %d", regioes[1].ForaDoUniverso)
	}
}

// CAN10 — valor fora de 0/1 é normalizado, nunca grava lixo.
func TestCAN10ForaDoUniversoNormalizado(t *testing.T) {
	svc := newPlanningTestService(t)

	salvo, err := svc.SavePlan(PaintingPlanDTO{
		Name: "Normalização",
		Tabs: []PaintingTabDTO{{
			Name: "Frente",
			Regions: []PaintingRegionDTO{
				{Hex: "#000000", ForaDoUniverso: 7},
			},
		}},
	})
	if err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if v := salvo.Tabs[0].Regions[0].ForaDoUniverso; v != 1 {
		t.Fatalf("esperava normalização para 1, veio %d", v)
	}
}
