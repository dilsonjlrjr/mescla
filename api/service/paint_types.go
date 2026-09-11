package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Tipo de tinta (rf-15): cadastro compartilhado, como o de fabricante. Tipo
// usado por alguma tinta do catálogo não pode ser excluído.

const maxPaintTypeNameRunes = 60

var (
	ErrPaintTypeNotFound  = errors.New("tipo de tinta não encontrado")
	ErrPaintTypeDuplicate = errors.New("nome de tipo de tinta repetido")
)

// PaintTypeInUseError recusa a exclusão: tinta do catálogo usa o tipo.
type PaintTypeInUseError struct{ Paints int }

func (e *PaintTypeInUseError) Error() string {
	return fmt.Sprintf("O tipo de tinta é usado por %d tintas do catálogo e não pode ser excluído.", e.Paints)
}

type PaintTypeDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PaintCount int    `json:"paintCount"`
}

const paintTypeSelect = `
	SELECT t.id, t.name, (SELECT COUNT(*) FROM paints p WHERE p.paint_type_id = t.id)
	FROM paint_types t`

// ensurePaintTypeDefaults semeia Acrílica e Wash e marca o catálogo inteiro
// como Acrílica (rf-15). Só age com a tabela vazia: depois disso o dono pode
// renomear ou criar tipos sem o boot desfazer nada.
func ensurePaintTypeDefaults(db *sql.DB) error {
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'paint_types'`).Scan(&tables); err != nil {
		return err
	}
	if tables == 0 {
		return nil
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM paint_types`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`INSERT INTO paint_types (name) VALUES ('Acrílica')`)
	if err != nil {
		return err
	}
	acrilica, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO paint_types (name) VALUES ('Wash')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE paints SET paint_type_id = ? WHERE paint_type_id IS NULL`, acrilica); err != nil {
		return err
	}
	return tx.Commit()
}

func normalizePaintTypeName(name string) (string, error) {
	name = cleanName(name)
	if name == "" {
		return "", &NameError{Msg: "Informe o nome do tipo de tinta."}
	}
	if utf8.RuneCountInString(name) > maxPaintTypeNameRunes {
		return "", &NameError{Msg: fmt.Sprintf("O nome pode ter no máximo %d caracteres.", maxPaintTypeNameRunes)}
	}
	return name, nil
}

func (s *PaintService) GetPaintTypes() ([]PaintTypeDTO, error) {
	rows, err := s.db.Query(paintTypeSelect + ` ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []PaintTypeDTO{}
	for rows.Next() {
		var t PaintTypeDTO
		if err := rows.Scan(&t.ID, &t.Name, &t.PaintCount); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (s *PaintService) paintTypeByID(id int64) (PaintTypeDTO, error) {
	var t PaintTypeDTO
	err := s.db.QueryRow(paintTypeSelect+` WHERE t.id = ?`, id).Scan(&t.ID, &t.Name, &t.PaintCount)
	if errors.Is(err, sql.ErrNoRows) {
		return PaintTypeDTO{}, ErrPaintTypeNotFound
	}
	return t, err
}

// paintTypeNameTaken compara em Go pelo mesmo motivo do fabricante: o LOWER do
// SQLite só cobre ASCII.
func (s *PaintService) paintTypeNameTaken(name string, selfID int64) (bool, error) {
	rows, err := s.db.Query(`SELECT id, name FROM paint_types`)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var existing string
		if err := rows.Scan(&id, &existing); err != nil {
			return false, err
		}
		if id != selfID && strings.EqualFold(cleanName(existing), name) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *PaintService) paintTypeWriteError(err error, id int64) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return ErrPaintTypeDuplicate
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		t, lookupErr := s.paintTypeByID(id)
		if lookupErr != nil {
			return lookupErr
		}
		return &PaintTypeInUseError{Paints: t.PaintCount}
	}
	return err
}

func (s *PaintService) AddPaintType(name string) (PaintTypeDTO, error) {
	name, err := normalizePaintTypeName(name)
	if err != nil {
		return PaintTypeDTO{}, err
	}
	taken, err := s.paintTypeNameTaken(name, 0)
	if err != nil {
		return PaintTypeDTO{}, err
	}
	if taken {
		return PaintTypeDTO{}, ErrPaintTypeDuplicate
	}
	res, err := s.db.Exec(`INSERT INTO paint_types (name) VALUES (?)`, name)
	if err != nil {
		return PaintTypeDTO{}, s.paintTypeWriteError(err, 0)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return PaintTypeDTO{}, err
	}
	return s.paintTypeByID(id)
}

func (s *PaintService) UpdatePaintType(id int64, name string) (PaintTypeDTO, error) {
	name, err := normalizePaintTypeName(name)
	if err != nil {
		return PaintTypeDTO{}, err
	}
	if _, err := s.paintTypeByID(id); err != nil {
		return PaintTypeDTO{}, err
	}
	taken, err := s.paintTypeNameTaken(name, id)
	if err != nil {
		return PaintTypeDTO{}, err
	}
	if taken {
		return PaintTypeDTO{}, ErrPaintTypeDuplicate
	}
	if _, err := s.db.Exec(`UPDATE paint_types SET name = ? WHERE id = ?`, name, id); err != nil {
		return PaintTypeDTO{}, s.paintTypeWriteError(err, id)
	}
	return s.paintTypeByID(id)
}

// DeletePaintType exclui um tipo que nenhuma tinta do catálogo usa. O DELETE é
// condicional num comando só: tinta que ganhe o tipo no meio impede a
// exclusão em vez de ficar apontando para um tipo que não existe.
func (s *PaintService) DeletePaintType(id int64) error {
	if _, err := s.paintTypeByID(id); err != nil {
		return err
	}
	res, err := s.db.Exec(`
		DELETE FROM paint_types
		WHERE id = ? AND NOT EXISTS (SELECT 1 FROM paints WHERE paint_type_id = ?)
	`, id, id)
	if err != nil {
		return s.paintTypeWriteError(err, id)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		t, err := s.paintTypeByID(id)
		if err != nil {
			return err
		}
		return &PaintTypeInUseError{Paints: t.PaintCount}
	}
	return nil
}
