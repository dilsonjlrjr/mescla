package similarity

import "sort"

// RankByDeltaE ordena resultados por Delta E (menor = mais similar)
func RankByDeltaE(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].DeltaE < results[j].DeltaE
	})
}

// FilterByMaxDeltaE remove resultados acima do threshold
func FilterByMaxDeltaE(results []SearchResult, maxDeltaE float64) []SearchResult {
	var filtered []SearchResult
	for _, r := range results {
		if r.DeltaE <= maxDeltaE {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GroupByManufacturer agrupa resultados por fabricante
func GroupByManufacturer(results []SearchResult) map[string][]SearchResult {
	groups := make(map[string][]SearchResult)
	for _, r := range results {
		groups[r.Manufacturer] = append(groups[r.Manufacturer], r)
	}
	return groups
}

// TopN retorna os N melhores resultados
func TopN(results []SearchResult, n int) []SearchResult {
	if n <= 0 || len(results) <= n {
		return results
	}
	return results[:n]
}
