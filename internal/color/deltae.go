package color

import "math"

// DeltaE76 calcula Delta E (CIE 1976) entre dois pontos CIELAB
func DeltaE76(lab1, lab2 [3]float64) float64 {
	dl := lab1[0] - lab2[0]
	da := lab1[1] - lab2[1]
	db := lab1[2] - lab2[2]
	return math.Sqrt(dl*dl + da*da + db*db)
}

// DeltaE94 calcula Delta E (CIE 1994) entre dois pontos CIELAB
// Usado para cores de superfície (graphic arts)
func DeltaE94(lab1, lab2 [3]float64) float64 {
	dl := lab1[0] - lab2[0]
	da := lab1[1] - lab2[1]
	db := lab1[2] - lab2[2]

	c1 := math.Sqrt(lab1[1]*lab1[1] + lab1[2]*lab1[2])
	c2 := math.Sqrt(lab2[1]*lab2[1] + lab2[2]*lab2[2])
	dc := c1 - c2

	dh2 := da*da + db*db - dc*dc
	dh := 0.0
	if dh2 > 0 {
		dh = math.Sqrt(dh2)
	}

	// Weighting factors for graphic arts
	const kL = 1.0
	const kC = 1.0
	const kH = 1.0
	const k1 = 0.045
	const k2 = 0.015

	sL := 1.0
	sc := 1 + k1*c1
	sh := 1 + k2*c1

	dl /= (kL * sL)
	dc /= (kC * sc)
	dh /= (kH * sh)

	return math.Sqrt(dl*dl + dc*dc + dh*dh)
}

// DeltaE2000 calcula Delta E (CIEDE2000) entre dois pontos CIELAB
// Mais preciso que DeltaE76 e DeltaE94
func DeltaE2000(lab1, lab2 [3]float64) float64 {
	l1, a1, b1 := lab1[0], lab1[1], lab1[2]
	l2, a2, b2 := lab2[0], lab2[1], lab2[2]

	// Step 1: Calculate Cab'
	c1 := math.Sqrt(a1*a1 + b1*b1)
	c2 := math.Sqrt(a2*a2 + b2*b2)
	cab := (c1 + c2) / 2

	cab7 := math.Pow(cab, 7)
	g := 0.5 * (1 - math.Sqrt(cab7/(cab7+math.Pow(25, 7))))

	a1p := a1 * (1 + g)
	a2p := a2 * (1 + g)

	c1p := math.Sqrt(a1p*a1p + b1*b1)
	c2p := math.Sqrt(a2p*a2p + b2*b2)

	var h1p, h2p float64
	if a1p == 0 && b1 == 0 {
		h1p = 0
	} else {
		h1p = math.Atan2(b1, a1p) * 180 / math.Pi
		if h1p < 0 {
			h1p += 360
		}
	}

	if a2p == 0 && b2 == 0 {
		h2p = 0
	} else {
		h2p = math.Atan2(b2, a2p) * 180 / math.Pi
		if h2p < 0 {
			h2p += 360
		}
	}

	// Step 2: Calculate Delta L', Delta C', Delta H'
	dl := l2 - l1
	dc := c2p - c1p

	var dhp float64
	if c1p*c2p == 0 {
		dhp = 0
	} else if math.Abs(h2p-h1p) <= 180 {
		dhp = h2p - h1p
	} else if h2p-h1p > 180 {
		dhp = h2p - h1p - 360
	} else {
		dhp = h2p - h1p + 360
	}

	dh := 2 * math.Sqrt(c1p*c2p) * math.Sin(dhp*math.Pi/360)

	// Step 3: Calculate CIEDE2000
	lp := (l1 + l2) / 2
	cp := (c1p + c2p) / 2

	var hp float64
	if c1p*c2p == 0 {
		hp = h1p + h2p
	} else if math.Abs(h1p-h2p) <= 180 {
		hp = (h1p + h2p) / 2
	} else if h1p+h2p < 360 {
		hp = (h1p + h2p + 360) / 2
	} else {
		hp = (h1p + h2p - 360) / 2
	}

	t := 1 - 0.17*math.Cos((hp-30)*math.Pi/180) +
		0.24*math.Cos(2*hp*math.Pi/180) +
		0.32*math.Cos((3*hp+6)*math.Pi/180) -
		0.20*math.Cos((4*hp-63)*math.Pi/180)

	lp502 := (lp - 50) * (lp - 50)
	sl := 1 + 0.015*lp502/math.Sqrt(20+lp502)
	sc := 1 + 0.045*cp
	sh := 1 + 0.015*cp*t

	cp7 := math.Pow(cp, 7)
	rt := -2 * math.Sqrt(cp7/(cp7+math.Pow(25, 7))) *
		math.Sin(60*math.Exp(-math.Pow((hp-275)/25, 2))*math.Pi/180)

	// Weighting factors
	const kL = 1.0
	const kC = 1.0
	const kH = 1.0

	return math.Sqrt(
		math.Pow(dl/(kL*sl), 2) +
			math.Pow(dc/(kC*sc), 2) +
			math.Pow(dh/(kH*sh), 2) +
			rt*(dc/(kC*sc))*(dh/(kH*sh)))
}
