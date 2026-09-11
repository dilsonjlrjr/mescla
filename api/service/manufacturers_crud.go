package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Cadastro, alteração e exclusão de fabricante (rf-14). A RG-18 foi reescrita
// em 2026-09-11: fabricante com tinta ou em uso não sai, e nenhuma exclusão
// apaga tinta em cascata.

const maxManufacturerNameRunes = 80

var (
	ErrManufacturerNotFound  = errors.New("fabricante não encontrado")
	ErrManufacturerDuplicate = errors.New("nome de fabricante repetido")
	ErrManufacturerInUse     = errors.New("fabricante em uso por receita ou recurso")
)

// NameError recusa o nome pela RN1; Msg é a frase mostrada ao usuário.
type NameError struct{ Msg string }

func (e *NameError) Error() string { return e.Msg }

// ManufacturerHasPaintsError recusa a exclusão: tinta do catálogo ou do
// estoque do servidor prende o fabricante.
type ManufacturerHasPaintsError struct{ Catalog, Stock int }

func (e *ManufacturerHasPaintsError) Error() string {
	return fmt.Sprintf("O fabricante tem %d tintas no catálogo e %d no estoque e não pode ser excluído.", e.Catalog, e.Stock)
}

// cleanName tira os caracteres de formatação invisíveis (U+200B,
// U+FEFF...) que TrimSpace não remove — sem isso nasce um segundo fabricante
// com nome visualmente idêntico — e depois apara.
func cleanName(name string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, name))
}

func normalizeManufacturerName(name string) (string, error) {
	name = cleanName(name)
	if name == "" {
		return "", &NameError{Msg: "Informe o nome do fabricante."}
	}
	if utf8.RuneCountInString(name) > maxManufacturerNameRunes {
		return "", &NameError{Msg: fmt.Sprintf("O nome pode ter no máximo %d caracteres.", maxManufacturerNameRunes)}
	}
	return name, nil
}

// manufacturerNameTaken compara em Go porque o LOWER do SQLite só cobre ASCII:
// "Açaí" e "AÇAÍ" precisam colidir. selfID fica de fora da comparação.
func (s *PaintService) manufacturerNameTaken(name string, selfID int64) (bool, error) {
	rows, err := s.db.Query(`SELECT id, name FROM manufacturers`)
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

// manufacturerWriteError traduz a constraint do banco para o erro do domínio,
// sem deixar o texto do SQLite chegar ao cliente.
func manufacturerWriteError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return ErrManufacturerDuplicate
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		return ErrManufacturerInUse
	}
	return err
}

func (s *PaintService) manufacturerByID(id int64) (ManufacturerDTO, error) {
	var m ManufacturerDTO
	err := s.db.QueryRow(manufacturerSelect+` WHERE m.id = ?`, id).Scan(
		&m.ID, &m.Name, &m.Country, &m.Website, &m.LogoPath, &m.PaintCount, &m.UserPaintCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ManufacturerDTO{}, ErrManufacturerNotFound
	}
	return m, err
}

// AddManufacturer cadastra um fabricante (US-13). Nome repetido, sem
// distinguir caixa, é erro — evita duas linhas "Vallejo".
func (s *PaintService) AddManufacturer(name string) (ManufacturerDTO, error) {
	name, err := normalizeManufacturerName(name)
	if err != nil {
		return ManufacturerDTO{}, err
	}
	taken, err := s.manufacturerNameTaken(name, 0)
	if err != nil {
		return ManufacturerDTO{}, err
	}
	if taken {
		return ManufacturerDTO{}, ErrManufacturerDuplicate
	}
	res, err := s.db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, name)
	if err != nil {
		return ManufacturerDTO{}, manufacturerWriteError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ManufacturerDTO{}, err
	}
	return s.manufacturerByID(id)
}

// UpdateManufacturer troca só o nome. O próprio fabricante não conta como
// repetido, então mudar só a caixa é aceito.
func (s *PaintService) UpdateManufacturer(id int64, name string) (ManufacturerDTO, error) {
	name, err := normalizeManufacturerName(name)
	if err != nil {
		return ManufacturerDTO{}, err
	}
	if _, err := s.manufacturerByID(id); err != nil {
		return ManufacturerDTO{}, err
	}
	taken, err := s.manufacturerNameTaken(name, id)
	if err != nil {
		return ManufacturerDTO{}, err
	}
	if taken {
		return ManufacturerDTO{}, ErrManufacturerDuplicate
	}
	if _, err := s.db.Exec(`UPDATE manufacturers SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, name, id); err != nil {
		return ManufacturerDTO{}, manufacturerWriteError(err)
	}
	return s.manufacturerByID(id)
}

// DeleteManufacturer exclui um fabricante sem tinta e sem uso. Linhas de
// produto vazias saem junto; o DELETE final é condicional, então tinta que
// entre no meio impede a exclusão em vez de ficar órfã.
//
// A transação é BEGIN IMMEDIATE numa conexão dedicada: ela pega o lock de
// escrita antes da primeira leitura. Numa transação deferred, outra escrita
// que comitasse entre a contagem e o DELETE fazia o upgrade falhar com
// SQLITE_BUSY — o busy_timeout não salva snapshot velho — e a exclusão
// concorrente virava 500 em vez de 404 ou 409.
func (s *PaintService) DeleteManufacturer(id int64) error {
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
		}
	}()

	var found int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM manufacturers WHERE id = ?`, id).Scan(&found); err != nil {
		return err
	}
	if found == 0 {
		return ErrManufacturerNotFound
	}

	catalog, stock, err := countManufacturerPaints(ctx, conn, id)
	if err != nil {
		return err
	}
	if catalog > 0 || stock > 0 {
		return &ManufacturerHasPaintsError{Catalog: catalog, Stock: stock}
	}

	var refs int
	if err := conn.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM recipes WHERE source_manufacturer_id = ? OR target_manufacturer_id = ?)
		     + (SELECT COUNT(*) FROM resources WHERE manufacturer_id = ?)
	`, id, id, id).Scan(&refs); err != nil {
		return err
	}
	if refs > 0 {
		return ErrManufacturerInUse
	}

	if _, err := conn.ExecContext(ctx, `
		DELETE FROM product_lines
		WHERE manufacturer_id = ?
		  AND NOT EXISTS (SELECT 1 FROM paints WHERE paints.product_line_id = product_lines.id)
	`, id); err != nil {
		return manufacturerWriteError(err)
	}
	res, err := conn.ExecContext(ctx, `
		DELETE FROM manufacturers
		WHERE id = ?
		  AND NOT EXISTS (SELECT 1 FROM paints WHERE manufacturer_id = ?)
		  AND NOT EXISTS (SELECT 1 FROM user_paints WHERE manufacturer_id = ?)
	`, id, id, id)
	if err != nil {
		return manufacturerWriteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		catalog, stock, err := countManufacturerPaints(ctx, conn, id)
		if err != nil {
			return err
		}
		return &ManufacturerHasPaintsError{Catalog: catalog, Stock: stock}
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	committed = true
	return nil
}

func countManufacturerPaints(ctx context.Context, conn *sql.Conn, id int64) (catalog, stock int, err error) {
	err = conn.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM paints WHERE manufacturer_id = ?),
		       (SELECT COUNT(*) FROM user_paints WHERE manufacturer_id = ?)
	`, id, id).Scan(&catalog, &stock)
	return catalog, stock, err
}
