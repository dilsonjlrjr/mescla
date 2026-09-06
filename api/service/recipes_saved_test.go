package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newRecipesTestService(t *testing.T) *PaintService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("abrindo banco: %v", err)
	}
	if err := ensureSavedRecipesSchema(db); err != nil {
		t.Fatalf("ensureSavedRecipesSchema: %v", err)
	}
	return &PaintService{db: db}
}

func TestSaveRecipeValidatesInput(t *testing.T) {
	s := newRecipesTestService(t)

	if _, err := s.SaveRecipe("", "#6E2A4A"); err == nil {
		t.Fatal("esperava erro para nome vazio")
	}
	if _, err := s.SaveRecipe("Mephiston", "não é hex"); err == nil {
		t.Fatal("esperava erro para hex inválido")
	}

	got, err := s.SaveRecipe("Mephiston Red", "#6E2A4A")
	if err != nil {
		t.Fatalf("save válido falhou: %v", err)
	}
	if got.ID == 0 || got.Name != "Mephiston Red" || got.TargetHex != "#6E2A4A" {
		t.Fatalf("save não resolveu campos: %+v", got)
	}
}

func TestRecipeCRUDRoundTrip(t *testing.T) {
	s := newRecipesTestService(t)

	r1, _ := s.SaveRecipe("Alvo A", "#112233")
	_, _ = s.SaveRecipe("Alvo B", "#445566")

	list, err := s.ListRecipes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("esperava 2 receitas, veio %d", len(list))
	}

	if err := s.DeleteRecipe(r1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = s.ListRecipes()
	if len(list) != 1 {
		t.Fatalf("esperava 1 receita após delete, veio %d", len(list))
	}

	// Idempotente: excluir de novo não é erro.
	if err := s.DeleteRecipe(r1.ID); err != nil {
		t.Fatalf("delete idempotente falhou: %v", err)
	}
}

func TestResolveSavedRecipeRejectsMissingOrInvalid(t *testing.T) {
	s := newRecipesTestService(t)

	if _, err := s.ResolveSavedRecipe(999, 1); err == nil {
		t.Fatal("esperava erro para receita inexistente")
	}
}
