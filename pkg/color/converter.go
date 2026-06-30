package color

import "math"

// RGBToHSV converte RGB (0-255) para HSV (h: 0-360, s: 0-1, v: 0-1)
func RGBToHSV(r, g, b uint8) (h, s, v float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	v = max

	if max == 0 {
		s = 0
		h = 0
		return
	}

	s = delta / max

	if delta == 0 {
		h = 0
		return
	}

	switch max {
	case rf:
		h = 60 * (math.Mod(((gf-bf)/delta), 6))
	case gf:
		h = 60 * (((bf - rf) / delta) + 2)
	case bf:
		h = 60 * (((rf - gf) / delta) + 4)
	}

	if h < 0 {
		h += 360
	}

	return
}

// HSVToRGB converte HSV (h: 0-360, s: 0-1, v: 0-1) para RGB (0-255)
func HSVToRGB(h, s, v float64) (r, g, b uint8) {
	if s == 0 {
		val := uint8(v * 255)
		return val, val, val
	}

	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var rf, gf, bf float64

	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	r = uint8((rf + m) * 255)
	g = uint8((gf + m) * 255)
	b = uint8((bf + m) * 255)
	return
}

// RGBToHSL converte RGB (0-255) para HSL (h: 0-360, s: 0-1, l: 0-1)
func RGBToHSL(r, g, b uint8) (h, s, l float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	l = (max + min) / 2

	if delta == 0 {
		h = 0
		s = 0
		return
	}

	if l <= 0.5 {
		s = delta / (max + min)
	} else {
		s = delta / (2 - max - min)
	}

	switch max {
	case rf:
		h = 60 * (math.Mod(((gf-bf)/delta), 6))
	case gf:
		h = 60 * (((bf - rf) / delta) + 2)
	case bf:
		h = 60 * (((rf - gf) / delta) + 4)
	}

	if h < 0 {
		h += 360
	}

	return
}

// HSLToRGB converte HSL (h: 0-360, s: 0-1, l: 0-1) para RGB (0-255)
func HSLToRGB(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		val := uint8(l * 255)
		return val, val, val
	}

	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}

	var c float64
	if l <= 0.5 {
		c = 2 * l * s
	} else {
		c = (2 - 2*l) * s
	}

	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var rf, gf, bf float64

	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	r = uint8((rf + m) * 255)
	g = uint8((gf + m) * 255)
	b = uint8((bf + m) * 255)
	return
}

// RGBToXYZ converte RGB (0-255) para XYZ (D65 illuminant)
func RGBToXYZ(r, g, b uint8) (x, y, z float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	// Inverse sRGB companding
	rf = linearize(rf)
	gf = linearize(gf)
	bf = linearize(bf)

	// sRGB to XYZ (D65)
	x = 0.4124564*rf + 0.3575761*gf + 0.1804375*bf
	y = 0.2126729*rf + 0.7151522*gf + 0.0721750*bf
	z = 0.0193339*rf + 0.1191920*gf + 0.9503041*bf

	return
}

// XYZToRGB converte XYZ para RGB (0-255)
func XYZToRGB(x, y, z float64) (r, g, b uint8) {
	// XYZ to sRGB (D65)
	rf := 3.2404542*x - 1.5371385*y - 0.4985314*z
	gf := -0.9692660*x + 1.8760108*y + 0.0415560*z
	bf := 0.0556434*x - 0.2040259*y + 1.0572252*z

	// Clamp
	rf = clamp01(rf)
	gf = clamp01(gf)
	bf = clamp01(bf)

	// sRGB companding
	rf = delinearize(rf)
	gf = delinearize(gf)
	bf = delinearize(bf)

	r = uint8(rf * 255)
	g = uint8(gf * 255)
	b = uint8(bf * 255)
	return
}

// XYZToLab converte XYZ para CIELAB (D65)
func XYZToLab(x, y, z float64) (l, a, b float64) {
	// D65 reference white
	const xn, yn, zn = 0.95047, 1.00000, 1.08883

	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	l = 116*fy - 16
	a = 500 * (fx - fy)
	b = 200 * (fy - fz)

	return
}

// LabToXYZ converte CIELAB para XYZ (D65)
func LabToXYZ(l, a, b float64) (x, y, z float64) {
	const xn, yn, zn = 0.95047, 1.00000, 1.08883

	fy := (l + 16) / 116
	fx := a/500 + fy
	fz := fy - b/200

	x = xn * labFInv(fx)
	y = yn * labFInv(fy)
	z = zn * labFInv(fz)

	return
}

// LabToLCH converte CIELAB para CIELCH
func LabToLCH(l, a, b float64) (l2, c, h float64) {
	l2 = l
	c = math.Sqrt(a*a + b*b)
	h = math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return
}

// LCHToLab converte CIELCH para CIELAB
func LCHToLab(l, c, h float64) (l2, a, b float64) {
	l2 = l
	hrad := h * math.Pi / 180
	a = c * math.Cos(hrad)
	b = c * math.Sin(hrad)
	return
}

// linearize aplica inverse sRGB companding
func linearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// delinearize aplica sRGB companding
func delinearize(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// labF função auxiliar para CIELAB
func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Pow(t, 1.0/3.0)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

// labFInv inversa da função labF
func labFInv(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta {
		return t * t * t
	}
	return 3 * delta * delta * (t - 4.0/29.0)
}

// clamp01 limita valor entre 0 e 1
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// RGBToLab converte RGB (0-255) diretamente para CIELAB
func RGBToLab(r, g, b uint8) (l, a, bOut float64) {
	x, y, z := RGBToXYZ(r, g, b)
	return XYZToLab(x, y, z)
}
