package mix

import "math"

func CalculateProportions(colors []PaintInput, target [3]float64) []float64 {
	if len(colors) == 0 {
		return nil
	}

	proportions := make([]float64, len(colors))
	for i := range proportions {
		proportions[i] = 100.0 / float64(len(colors))
	}

	return proportions
}

func ValidateProportions(proportions []float64) bool {
	if len(proportions) == 0 {
		return false
	}

	var sum float64
	for _, p := range proportions {
		if p < 0 || p > 100 {
			return false
		}
		sum += p
	}

	return math.Abs(sum-100.0) < 0.01
}

func NormalizeProportions(proportions []float64) []float64 {
	if len(proportions) == 0 {
		return nil
	}

	var sum float64
	for _, p := range proportions {
		sum += p
	}

	if sum == 0 {
		n := 100.0 / float64(len(proportions))
		result := make([]float64, len(proportions))
		for i := range result {
			result[i] = n
		}
		return result
	}

	result := make([]float64, len(proportions))
	for i, p := range proportions {
		result[i] = (p / sum) * 100.0
	}

	return result
}
