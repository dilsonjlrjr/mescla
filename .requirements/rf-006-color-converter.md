# RF-006 — Color Converter

## ID
RF-006

## Título
Color Converter — Conversão entre espaços de cor

## Status
Done

## Objetivo
Implementar biblioteca de conversão entre espaços de cor para tintas modelismo.

## Motivação
O sistema precisa converter cores entre diferentes representações (RGB, HSV, HSL, XYZ, CIELAB, CIELCH) para calcular similaridade e DeltaE.

## Escopo
- Conversão RGB ↔ HSV
- Conversão RGB ↔ HSL
- Conversão RGB ↔ XYZ
- Conversão XYZ ↔ CIELAB
- Conversão CIELAB ↔ CIELCH
- Cálculo DeltaE 76
- Cálculo DeltaE 94
- Cálculo DeltaE 2000

## O que faz
- Converte cores entre espaços
- Calcula distância entre cores (DeltaE)
- Fornece funções puras (sem estado)

## O que não faz
- Não modifica banco de dados
- Não gera swatches
- Não calcula misturas

## Dependências
- RF-001 Database Schema

## Pré-requisitos
- Schema com colunas rgb_r, rgb_g, rgb_b, hsv_h, hsv_s, hsv_v, hsl_h, hsl_s, hsl_l, lab_l, lab_a, lab_b, lch_l, lch_c, lch_h

## Gatilhos
- Quando precisar calcular similaridade entre tintas
- Quando precisar converter cores para exibição

## Fluxo esperado
1. Receber cor em um espaço (ex: RGB)
2. Converter para espaço destino (ex: CIELAB)
3. Retornar valores convertidos

## Estrutura técnica
```go
// pkg/color/converter.go
type Color struct {
    R, G, B uint8
    H, S, V float64
    H2, S2, L float64
    X, Y, Z float64
    L, A, B float64
    L2, C, H float64
}

func RGBToHSV(r, g, b uint8) (h, s, v float64)
func HSVToRGB(h, s, v float64) (r, g, b uint8)
func RGBToHSL(r, g, b uint8) (h, s, l float64)
func HSLToRGB(h, s, l float64) (r, g, b uint8)
func RGBToXYZ(r, g, b uint8) (x, y, z float64)
func XYZToRGB(x, y, z float64) (r, g, b uint8)
func XYZToLab(x, y, z float64) (l, a, b float64)
func LabToXYZ(l, a, b float64) (x, y, z float64)
func LabToLCH(l, a, b float64) (l2, c, h float64)
func LCHToLab(l, c, h float64) (l2, a, b float64)
func DeltaE76(lab1, lab2 [3]float64) float64
func DeltaE94(lab1, lab2 [3]float64) float64
func DeltaE2000(lab1, lab2 [3]float64) float64
```

## Arquivos envolvidos
- `pkg/color/converter.go` — funções de conversão
- `pkg/color/deltae.go` — cálculos DeltaE
- `pkg/color/converter_test.go` — testes unitários

## Bibliotecas
- Nenhuma dependência externa (matemática pura)

## Banco de dados
- Leitura: paint_colors (rgb_r, rgb_g, rgb_b)
- Escrita: paint_colors (hsv_*, hsl_*, lab_*, lch_*)

## Critérios de aceite
- [ ] Funções RGB ↔ HSV corretas
- [ ] Funções RGB ↔ HSL corretas
- [ ] Funções RGB ↔ XYZ corretas
- [ ] Funções XYZ ↔ CIELAB corretas
- [ ] Funções CIELAB ↔ CIELCH corretas
- [ ] DeltaE 76 implementado
- [ ] DeltaE 94 implementado
- [ ] DeltaE 2000 implementado
- [ ] Testes unitários passam
- [ ] Precisão < 0.01 para conversões

## Critérios de qualidade
- Funções puras (sem efeitos colaterais)
- Testes com tolerância definida
- Nomes claros e consistentes

## Exemplo de uso
```go
h, s, v := color.RGBToHSV(255, 0, 0) // Vermelho
l, a, b := color.RGBToLab(255, 0, 0)
delta := color.DeltaE2000([3]float64{53, 80, 67}, [3]float64{53, 80, 68})
```

## Próximos requisitos dependentes
- RF-007 Mix Engine
- RF-008 Similarity Engine
