package similarity

import (
	"testing"
)

func TestRankByDeltaE(t *testing.T) {
	results := []SearchResult{
		{PaintID: 1, Name: "Red", DeltaE: 5.0},
		{PaintID: 2, Name: "Blue", DeltaE: 2.0},
		{PaintID: 3, Name: "Green", DeltaE: 8.0},
	}

	RankByDeltaE(results)

	if results[0].DeltaE != 2.0 {
		t.Errorf("expected first result DeltaE=2.0, got %f", results[0].DeltaE)
	}
	if results[1].DeltaE != 5.0 {
		t.Errorf("expected second result DeltaE=5.0, got %f", results[1].DeltaE)
	}
	if results[2].DeltaE != 8.0 {
		t.Errorf("expected third result DeltaE=8.0, got %f", results[2].DeltaE)
	}
}

func TestFilterByMaxDeltaE(t *testing.T) {
	results := []SearchResult{
		{PaintID: 1, DeltaE: 2.0},
		{PaintID: 2, DeltaE: 5.0},
		{PaintID: 3, DeltaE: 12.0},
		{PaintID: 4, DeltaE: 8.0},
	}

	filtered := FilterByMaxDeltaE(results, 10.0)

	if len(filtered) != 3 {
		t.Errorf("expected 3 results, got %d", len(filtered))
	}
	for _, r := range filtered {
		if r.DeltaE > 10.0 {
			t.Errorf("result with DeltaE=%f should have been filtered", r.DeltaE)
		}
	}
}

func TestGroupByManufacturer(t *testing.T) {
	results := []SearchResult{
		{PaintID: 1, Manufacturer: "Vallejo", DeltaE: 2.0},
		{PaintID: 2, Manufacturer: "Citadel", DeltaE: 3.0},
		{PaintID: 3, Manufacturer: "Vallejo", DeltaE: 4.0},
	}

	groups := GroupByManufacturer(results)

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["Vallejo"]) != 2 {
		t.Errorf("expected 2 Vallejo results, got %d", len(groups["Vallejo"]))
	}
	if len(groups["Citadel"]) != 1 {
		t.Errorf("expected 1 Citadel result, got %d", len(groups["Citadel"]))
	}
}

func TestTopN(t *testing.T) {
	results := []SearchResult{
		{PaintID: 1, DeltaE: 1.0},
		{PaintID: 2, DeltaE: 2.0},
		{PaintID: 3, DeltaE: 3.0},
		{PaintID: 4, DeltaE: 4.0},
		{PaintID: 5, DeltaE: 5.0},
	}

	top3 := TopN(results, 3)
	if len(top3) != 3 {
		t.Errorf("expected 3 results, got %d", len(top3))
	}

	top10 := TopN(results, 10)
	if len(top10) != 5 {
		t.Errorf("expected 5 results (all), got %d", len(top10))
	}

	top0 := TopN(results, 0)
	if len(top0) != 5 {
		t.Errorf("expected 5 results (all), got %d", len(top0))
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.MaxResults != 10 {
		t.Errorf("expected MaxResults=10, got %d", opts.MaxResults)
	}
	if opts.MaxDeltaE != 10.0 {
		t.Errorf("expected MaxDeltaE=10.0, got %f", opts.MaxDeltaE)
	}
	if opts.ManufacturerID != nil {
		t.Errorf("expected ManufacturerID=nil, got %v", opts.ManufacturerID)
	}
	if opts.PaintTypeID != nil {
		t.Errorf("expected PaintTypeID=nil, got %v", opts.PaintTypeID)
	}
}
