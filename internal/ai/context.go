package ai

import "paint-match-ai/internal/color"

// QueryIntent tipos de intenção do usuário
type QueryIntent string

const (
	IntentFindSimilar     QueryIntent = "find_similar"
	IntentFindEquivalent  QueryIntent = "find_equivalent"
	IntentMixRecipe       QueryIntent = "mix_recipe"
	IntentPaintInfo       QueryIntent = "paint_info"
	IntentManufacturerInfo QueryIntent = "manufacturer_info"
	IntentCompare         QueryIntent = "compare"
	IntentGeneral         QueryIntent = "general"
)

// Query representa uma pergunta estruturada do usuário
type Query struct {
	Text        string
	Intent      QueryIntent
	TargetColor *color.RGB
	Filters     QueryFilters
}

// QueryFilters filtros opcionais para buscas
type QueryFilters struct {
	ManufacturerID *int64
	PaintTypeID    *int64
	FinishTypeID   *int64
	MaxDeltaE      *float64
	MaxResults     *int
}

// Response representa a resposta estruturada
type Response struct {
	Intent       QueryIntent
	Paints       []PaintInfo
	Equivalences []Equivalence
	Recipes      []Recipe
	Similar      []SimilarResult
	Message      string
	Sources      []string
	IsEstimate   bool
}

// PaintInfo informações completas de uma tinta
type PaintInfo struct {
	ID           int64
	Name         string
	Code         string
	Manufacturer string
	ProductLine  string
	RGB          color.RGB
	Lab          color.Lab
	SwatchPath   string
	Thumbnail    string
	ImageURL     string
	FinishType   string
	PaintType    string
	Coverage     string
	Opacity      string
	Volume       string
}

// Equivalence par de tintas equivalentes
type Equivalence struct {
	SourcePaint PaintInfo
	TargetPaint PaintInfo
	DeltaE      float64
	Similarity  float64
	Notes       string
}

// Recipe receita de mistura
type Recipe struct {
	TargetPaint PaintInfo
	Ingredients []RecipeIngredient
	Method      string
	DeltaE      float64
}

// RecipeIngredient ingrediente de uma receita
type RecipeIngredient struct {
	Paint      PaintInfo
	Percentage float64
}

// SimilarResult resultado de busca por similaridade
type SimilarResult struct {
	Paint  PaintInfo
	DeltaE float64
	Rank   int
}

// RGB alias para color.RGB
type RGB = color.RGB

// Lab alias para color.Lab
type Lab = color.Lab
