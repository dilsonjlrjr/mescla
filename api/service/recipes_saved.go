package service

import (
	"database/sql"
	"fmt"

	"paint-match-ai/api/domain/stock"
)

// ensureSavedRecipesSchema cria a tabela de receitas salvas pelo usuário
// (RF-04, tela Receitas). Não é a mesma coisa que a tabela legada `recipes`
// (cache de equivalência entre tintas do catálogo, ligada a target_paint_id) —
// aqui o que se guarda é o ALVO (hex livre, pode não existir no catálogo),
// nunca a fórmula: reabrir resolve de novo no fabricante escolhido (RG-16 de
// docs/mesclaai-userstory.md). Nome da tabela deliberadamente distinto de
// `recipes` para não colidir com o conceito legado.
func ensureSavedRecipesSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS saved_recipes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			target_hex TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// SavedRecipeDTO é uma receita salva pelo usuário: nome + alvo. A fórmula não
// é persistida — ResolveSavedRecipe recalcula a cada abertura.
type SavedRecipeDTO struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	TargetHex string `json:"targetHex"`
	CreatedAt string `json:"createdAt"`
}

// SaveRecipe grava um alvo com nome. targetHex precisa ser um hex válido
// (#rrggbb) — mesma validação de cadastro de tinta.
func (s *PaintService) SaveRecipe(name, targetHex string) (SavedRecipeDTO, error) {
	if name == "" {
		return SavedRecipeDTO{}, fmt.Errorf("nome da receita é obrigatório")
	}
	if _, ok := stock.ParseHex(targetHex); !ok {
		return SavedRecipeDTO{}, fmt.Errorf("hex inválido: %s", targetHex)
	}

	res, err := s.db.Exec(
		`INSERT INTO saved_recipes (name, target_hex) VALUES (?, ?)`,
		name, targetHex,
	)
	if err != nil {
		return SavedRecipeDTO{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SavedRecipeDTO{}, err
	}
	return s.getSavedRecipe(id)
}

func (s *PaintService) getSavedRecipe(id int64) (SavedRecipeDTO, error) {
	var r SavedRecipeDTO
	err := s.db.QueryRow(
		`SELECT id, name, target_hex, created_at FROM saved_recipes WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.TargetHex, &r.CreatedAt)
	return r, err
}

// ListRecipes lista as receitas salvas, mais recentes primeiro.
func (s *PaintService) ListRecipes() ([]SavedRecipeDTO, error) {
	rows, err := s.db.Query(
		`SELECT id, name, target_hex, created_at FROM saved_recipes ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SavedRecipeDTO, 0)
	for rows.Next() {
		var r SavedRecipeDTO
		if err := rows.Scan(&r.ID, &r.Name, &r.TargetHex, &r.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// DeleteRecipe remove uma receita salva. Idempotente: excluir id inexistente
// não é erro (mesma convenção de DeleteUserPaint).
func (s *PaintService) DeleteRecipe(id int64) error {
	_, err := s.db.Exec(`DELETE FROM saved_recipes WHERE id = ?`, id)
	return err
}

// ResolveSavedRecipe reabre uma receita salva e resolve a fórmula de novo no
// fabricante escolhido — nunca reproduz uma fórmula congelada (RG-16). Reusa
// SuggestRecipeForColor, a mesma orquestração da Roda de cores.
func (s *PaintService) ResolveSavedRecipe(id int64, targetManufacturerID int64) (EquivalentRecipeDTO, error) {
	recipe, err := s.getSavedRecipe(id)
	if err != nil {
		return EquivalentRecipeDTO{}, fmt.Errorf("receita não encontrada: %w", err)
	}
	rgb, ok := stock.ParseHex(recipe.TargetHex)
	if !ok {
		return EquivalentRecipeDTO{}, fmt.Errorf("receita com alvo inválido: %s", recipe.TargetHex)
	}
	return s.SuggestRecipeForColor(rgb.R, rgb.G, rgb.B, targetManufacturerID)
}
