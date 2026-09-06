package service

import "fmt"

// AddManufacturer cadastra um fabricante fora do catálogo seedado (RF-04,
// tela Minhas tintas / aba Fabricantes — US-13). Nome é a chave visível ao
// usuário; duplicata é erro, não silenciosa (evita duas linhas "Vallejo").
func (s *PaintService) AddManufacturer(name string) (ManufacturerDTO, error) {
	if name == "" {
		return ManufacturerDTO{}, fmt.Errorf("nome do fabricante é obrigatório")
	}

	var exists int
	s.db.QueryRow(`SELECT COUNT(*) FROM manufacturers WHERE name = ?`, name).Scan(&exists)
	if exists > 0 {
		return ManufacturerDTO{}, fmt.Errorf("já existe um fabricante com esse nome")
	}

	res, err := s.db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, name)
	if err != nil {
		return ManufacturerDTO{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ManufacturerDTO{}, err
	}

	return ManufacturerDTO{ID: id, Name: name}, nil
}

// DeleteManufacturerResultDTO informa quantas tintas do estoque do usuário
// saíram junto — a confirmação de exclusão (RG-18) precisa declarar esse
// número ANTES de o usuário confirmar, então o fluxo esperado é: o cliente já
// mostrou paintCount de GetManufacturers() na confirmação; esta chamada só
// executa depois que o usuário confirmou.
type DeleteManufacturerResultDTO struct {
	RemovedUserPaints int `json:"removedUserPaints"`
}

// DeleteManufacturer remove um fabricante e, em cascata, as tintas do usuário
// (user_paints) que apontavam pra ele. Recusa a exclusão se o fabricante tem
// tintas no catálogo seedado (paints) — isso é dado de referência
// compartilhado, não propriedade do usuário; um fabricante com catálogo não
// pode ser apagado por aqui (só os fabricantes que o próprio usuário criou
// ficam sem tintas de catálogo).
func (s *PaintService) DeleteManufacturer(id int64) (DeleteManufacturerResultDTO, error) {
	var catalogPaints int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM paints WHERE manufacturer_id = ?`, id).Scan(&catalogPaints); err != nil {
		return DeleteManufacturerResultDTO{}, err
	}
	if catalogPaints > 0 {
		return DeleteManufacturerResultDTO{}, fmt.Errorf("fabricante tem %d tintas no catálogo — não pode ser excluído", catalogPaints)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return DeleteManufacturerResultDTO{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM user_paints WHERE manufacturer_id = ?`, id)
	if err != nil {
		return DeleteManufacturerResultDTO{}, err
	}
	removed, _ := res.RowsAffected()

	if _, err := tx.Exec(`DELETE FROM manufacturers WHERE id = ?`, id); err != nil {
		return DeleteManufacturerResultDTO{}, err
	}

	if err := tx.Commit(); err != nil {
		return DeleteManufacturerResultDTO{}, err
	}

	return DeleteManufacturerResultDTO{RemovedUserPaints: int(removed)}, nil
}
