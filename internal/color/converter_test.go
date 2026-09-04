package color

import (
	"math"
	"testing"
)

func TestRGBToHSV(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		h, s, v float64
	}{
		{255, 0, 0, 0, 1, 1},       // Vermelho
		{0, 255, 0, 120, 1, 1},     // Verde
		{0, 0, 255, 240, 1, 1},     // Azul
		{255, 255, 255, 0, 0, 1},   // Branco
		{0, 0, 0, 0, 0, 0},         // Preto
		{128, 128, 128, 0, 0, 0.5}, // Cinza
	}

	for _, tt := range tests {
		h, s, v := RGBToHSV(tt.r, tt.g, tt.b)
		if !approxEqual(h, tt.h, 1) || !approxEqual(s, tt.s, 2) || !approxEqual(v, tt.v, 2) {
			t.Errorf("RGBToHSV(%d,%d,%d) = (%.2f,%.2f,%.2f), want (%.2f,%.2f,%.2f)",
				tt.r, tt.g, tt.b, h, s, v, tt.h, tt.s, tt.v)
		}
	}
}

func TestHSVToRGB(t *testing.T) {
	tests := []struct {
		h, s, v float64
		r, g, b uint8
	}{
		{0, 1, 1, 255, 0, 0},
		{120, 1, 1, 0, 255, 0},
		{240, 1, 1, 0, 0, 255},
	}

	for _, tt := range tests {
		r, g, b := HSVToRGB(tt.h, tt.s, tt.v)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("HSVToRGB(%.0f,%.0f,%.0f) = (%d,%d,%d), want (%d,%d,%d)",
				tt.h, tt.s, tt.v, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

func TestRGBToHSL(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		h, s, l float64
	}{
		{255, 0, 0, 0, 1, 0.5},
		{0, 255, 0, 120, 1, 0.5},
		{0, 0, 255, 240, 1, 0.5},
	}

	for _, tt := range tests {
		h, s, l := RGBToHSL(tt.r, tt.g, tt.b)
		if !approxEqual(h, tt.h, 1) || !approxEqual(s, tt.s, 2) || !approxEqual(l, tt.l, 2) {
			t.Errorf("RGBToHSL(%d,%d,%d) = (%.2f,%.2f,%.2f), want (%.2f,%.2f,%.2f)",
				tt.r, tt.g, tt.b, h, s, l, tt.h, tt.s, tt.l)
		}
	}
}

func TestHSLToRGB(t *testing.T) {
	tests := []struct {
		h, s, l float64
		r, g, b uint8
	}{
		{0, 1, 0.5, 255, 0, 0},
		{120, 1, 0.5, 0, 255, 0},
		{240, 1, 0.5, 0, 0, 255},
	}

	for _, tt := range tests {
		r, g, b := HSLToRGB(tt.h, tt.s, tt.l)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("HSLToRGB(%.0f,%.0f,%.1f) = (%d,%d,%d), want (%d,%d,%d)",
				tt.h, tt.s, tt.l, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

func TestRGBToXYZ(t *testing.T) {
	// Vermelho puro
	x, y, z := RGBToXYZ(255, 0, 0)
	if !approxEqual(x, 0.4124, 3) || !approxEqual(y, 0.2126, 3) || !approxEqual(z, 0.0193, 3) {
		t.Errorf("RGBToXYZ(255,0,0) = (%.4f,%.4f,%.4f), want (0.4124,0.2126,0.0193)", x, y, z)
	}
}

func TestXYZToRGB(t *testing.T) {
	r, g, b := XYZToRGB(0.4124, 0.2126, 0.0193)
	if !approxEqual(float64(r), 255, 0) || g != 0 || b != 0 {
		t.Errorf("XYZToRGB(0.4124,0.2126,0.0193) = (%d,%d,%d), want (255,0,0)", r, g, b)
	}
}

func TestXYZToLab(t *testing.T) {
	// D65 white point
	l, a, b := XYZToLab(0.95047, 1.0, 1.08883)
	if !approxEqual(l, 100, 2) || !approxEqual(a, 0, 2) || !approxEqual(b, 0, 2) {
		t.Errorf("XYZToLab(D65) = (%.2f,%.2f,%.2f), want (100,0,0)", l, a, b)
	}
}

func TestLabToXYZ(t *testing.T) {
	x, y, z := LabToXYZ(100, 0, 0)
	if !approxEqual(x, 0.95047, 3) || !approxEqual(y, 1.0, 3) || !approxEqual(z, 1.08883, 3) {
		t.Errorf("LabToXYZ(100,0,0) = (%.5f,%.5f,%.5f), want (0.95047,1.0,1.08883)", x, y, z)
	}
}

func TestLabToLCH(t *testing.T) {
	l, c, h := LabToLCH(50, 30, 40)
	if !approxEqual(l, 50, 2) || !approxEqual(c, 50, 2) || !approxEqual(h, 53.13, 2) {
		t.Errorf("LabToLCH(50,30,40) = (%.2f,%.2f,%.2f), want (50,50,53.13)", l, c, h)
	}
}

func TestLCHToLab(t *testing.T) {
	l, a, b := LCHToLab(50, 50, 53.13)
	if !approxEqual(l, 50, 2) || !approxEqual(a, 30, 2) || !approxEqual(b, 40, 2) {
		t.Errorf("LCHToLab(50,50,53.13) = (%.2f,%.2f,%.2f), want (50,30,40)", l, a, b)
	}
}

func TestDeltaE76(t *testing.T) {
	lab1 := [3]float64{50, 0, 0}
	lab2 := [3]float64{50, 0, 0}
	d := DeltaE76(lab1, lab2)
	if !approxEqual(d, 0, 2) {
		t.Errorf("DeltaE76 same color = %.2f, want 0", d)
	}

	lab2 = [3]float64{60, 10, 10}
	d = DeltaE76(lab1, lab2)
	expected := math.Sqrt(100 + 100 + 100)
	if !approxEqual(d, expected, 2) {
		t.Errorf("DeltaE76 = %.2f, want %.2f", d, expected)
	}
}

func TestDeltaE2000(t *testing.T) {
	// Same color should be 0
	lab1 := [3]float64{50, 0, 0}
	lab2 := [3]float64{50, 0, 0}
	d := DeltaE2000(lab1, lab2)
	if !approxEqual(d, 0, 2) {
		t.Errorf("DeltaE2000 same color = %.4f, want 0", d)
	}

	// Known test case
	lab1 = [3]float64{50, 2.6772, -79.7751}
	lab2 = [3]float64{50, -1.0, -80.0}
	d = DeltaE2000(lab1, lab2)
	if d < 0 || d > 10 {
		t.Errorf("DeltaE2000 unexpected value: %.4f", d)
	}
}

func TestRoundTrip(t *testing.T) {
	// RGB -> HSV -> RGB
	r, g, b := uint8(123), uint8(45), uint8(200)
	h, s, v := RGBToHSV(r, g, b)
	r2, g2, b2 := HSVToRGB(h, s, v)
	if !approxEqual(float64(r), float64(r2), 0) || !approxEqual(float64(g), float64(g2), 0) || !approxEqual(float64(b), float64(b2), 0) {
		t.Errorf("RGB->HSV->RGB round trip: (%d,%d,%d) -> (%d,%d,%d)", r, g, b, r2, g2, b2)
	}

	// RGB -> XYZ -> Lab -> XYZ -> RGB
	xf, yf, zf := RGBToXYZ(r, g, b)
	lf, af, bf := XYZToLab(xf, yf, zf)
	xf2, yf2, zf2 := LabToXYZ(lf, af, bf)
	r3, g3, b3 := XYZToRGB(xf2, yf2, zf2)
	if !approxEqual(float64(r), float64(r3), 0) || !approxEqual(float64(g), float64(g3), 0) || !approxEqual(float64(b), float64(b3), 0) {
		t.Errorf("RGB->XYZ->Lab->XYZ->RGB round trip: (%d,%d,%d) -> (%d,%d,%d)", r, g, b, r3, g3, b3)
	}
}

func approxEqual(a, b, precision float64) bool {
	p := math.Pow(10, -precision)
	return math.Abs(a-b) <= p
}
