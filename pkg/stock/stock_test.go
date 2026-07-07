package stock

import "testing"

var mfrs = []Manufacturer{
	{ID: 1, Name: "Vallejo"},
	{ID: 2, Name: "Citadel"},
}

func TestParseCSVHappyPath(t *testing.T) {
	csv := "fabricante,nome,codigo,hex,volume,notas\n" +
		"Vallejo,Model Color Black,70.950,#1c1c1c,17ml,base\n" +
		"citadel,Mephiston Red,,#9a1115,,\n" // fabricante case-insensitive, campos opcionais vazios

	paints, errs := ParseCSV(csv, mfrs)
	if len(errs) != 0 {
		t.Fatalf("não esperava erros, veio %+v", errs)
	}
	if len(paints) != 2 {
		t.Fatalf("esperava 2 tintas, veio %d", len(paints))
	}
	if paints[0].ManufacturerID != 1 || paints[0].R != 0x1c {
		t.Fatalf("linha 1 mal resolvida: %+v", paints[0])
	}
	if paints[1].ManufacturerID != 2 || paints[1].Manufacturer != "Citadel" {
		t.Fatalf("citadel deveria resolver para o fabricante canônico: %+v", paints[1])
	}
}

func TestParseCSVUnknownManufacturerIsRejected(t *testing.T) {
	// A crítica-título do requisito: fabricante que não existe é recusado.
	csv := "MarcaInexistente,Tinta,X,#000000,,\n"
	paints, errs := ParseCSV(csv, mfrs)
	if len(paints) != 0 {
		t.Fatalf("não deveria importar tinta de fabricante inexistente")
	}
	if len(errs) != 1 || errs[0].Line != 1 {
		t.Fatalf("esperava 1 erro na linha 1, veio %+v", errs)
	}
}

func TestParseCSVCollectsAllErrors(t *testing.T) {
	// Sem cabeçalho; três linhas ruins e uma boa — todas as ruins reportadas.
	csv := "Vallejo,,70.9,#111111,,\n" + // nome vazio
		"Vallejo,Boa,,#abc,,\n" + // ok (hex curto)
		"Vallejo,Ruim,,naoehex,,\n" + // hex inválido
		",Sem marca,,#222222,,\n" // fabricante vazio
	paints, errs := ParseCSV(csv, mfrs)
	if len(paints) != 1 {
		t.Fatalf("esperava 1 tinta válida, veio %d", len(paints))
	}
	if len(errs) != 3 {
		t.Fatalf("esperava 3 erros, veio %d: %+v", len(errs), errs)
	}
}

func TestParseHex(t *testing.T) {
	cases := map[string]struct {
		ok      bool
		r, g, b uint8
	}{
		"#1C1C1C": {true, 0x1c, 0x1c, 0x1c},
		"ffffff":  {true, 255, 255, 255},
		"#abc":    {true, 0xaa, 0xbb, 0xcc},
		"":        {false, 0, 0, 0},
		"#12345":  {false, 0, 0, 0},
		"gggggg":  {false, 0, 0, 0},
	}
	for in, want := range cases {
		got, ok := ParseHex(in)
		if ok != want.ok {
			t.Fatalf("ParseHex(%q) ok=%v, esperava %v", in, ok, want.ok)
		}
		if ok && (got.R != want.r || got.G != want.g || got.B != want.b) {
			t.Fatalf("ParseHex(%q) = %+v, esperava %d,%d,%d", in, got, want.r, want.g, want.b)
		}
	}
}

func TestToCSVRoundTrips(t *testing.T) {
	// O que o export gera precisa reentrar pela importação sem perder nada.
	original := []Paint{
		{ManufacturerID: 1, Manufacturer: "Vallejo", Name: "Preto", Code: "70.950", R: 0x1c, G: 0x1c, B: 0x1c, Volume: "17ml", Notes: "base"},
		{ManufacturerID: 2, Manufacturer: "Citadel", Name: "Mephiston, Red", Code: "", R: 0x9a, G: 0x11, B: 0x15, Volume: "", Notes: "com, vírgula"},
	}
	back, errs := ParseCSV(ToCSV(original), mfrs)
	if len(errs) != 0 {
		t.Fatalf("export não reentrou limpo: %+v", errs)
	}
	if len(back) != 2 {
		t.Fatalf("esperava 2 tintas de volta, veio %d", len(back))
	}
	if back[1].Name != "Mephiston, Red" || back[1].Notes != "com, vírgula" {
		t.Fatalf("vírgulas em campos não sobreviveram ao round-trip: %+v", back[1])
	}
	if back[0].R != 0x1c || back[0].B != 0x1c {
		t.Fatalf("cor não sobreviveu ao round-trip: %+v", back[0])
	}
}

func TestCSVTemplateRoundTrips(t *testing.T) {
	// O modelo que oferecemos para download precisa ser importável sem erros
	// (assumindo que os fabricantes de exemplo existam).
	paints, errs := ParseCSV(CSVTemplate(), mfrs)
	if len(errs) != 0 {
		t.Fatalf("o próprio modelo não passou na crítica: %+v", errs)
	}
	if len(paints) != 2 {
		t.Fatalf("modelo deveria ter 2 linhas de exemplo, veio %d", len(paints))
	}
}
