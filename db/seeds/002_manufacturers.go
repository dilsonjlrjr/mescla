package seeds

import (
	"database/sql"
	"fmt"
	"log"
)

// ManufacturerSeed representa os dados de um fabricante
type ManufacturerSeed struct {
	Name    string
	Country string
	Website string
}

// manufacturers contém a lista completa de fabricantes conhecidos
var manufacturers = []ManufacturerSeed{
	{Name: "AK Interactive", Country: "Spain", Website: "https://ak-interactive.com"},
	{Name: "Vallejo", Country: "Spain", Website: "https://acrylicosvallejo.com"},
	{Name: "Citadel", Country: "UK", Website: "https://www.games-workshop.com"},
	{Name: "Army Painter", Country: "Denmark", Website: "https://www.thearmypainter.com"},
	{Name: "Scale75", Country: "Spain", Website: "https://scale75.com"},
	{Name: "Tamiya", Country: "Japan", Website: "https://www.tamiya.com"},
	{Name: "Mr Hobby", Country: "Japan", Website: "https://www.gsiCreos.co.jp"},
	{Name: "Acrilex", Country: "Brazil", Website: "https://www.acrilex.com.br"},
	{Name: "Corfix", Country: "Brazil", Website: "https://www.corfix.com.br"},
	{Name: "Talento", Country: "Brazil", Website: "https://www.talento.com.br"},
}

// SeedManufacturers popula a tabela manufacturers com dados reais
func SeedManufacturers(db *sql.DB) error {
	stmt, err := db.Prepare(`
		INSERT INTO manufacturers (name, country, website)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			country = excluded.country,
			website = excluded.website,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	updated := 0

	for _, m := range manufacturers {
		// Verificar se já existe
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM manufacturers WHERE name = ?", m.Name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("verificar existência de %s: %w", m.Name, err)
		}

		_, err = stmt.Exec(m.Name, m.Country, m.Website)
		if err != nil {
			return fmt.Errorf("inserir %s: %w", m.Name, err)
		}

		if exists > 0 {
			updated++
			log.Printf("[seed] Atualizado: %s (%s)", m.Name, m.Country)
		} else {
			inserted++
			log.Printf("[seed] Inserido: %s (%s)", m.Name, m.Country)
		}
	}

	log.Printf("[seed] Manufacturers: %d inseridos, %d atualizados", inserted, updated)
	return nil
}

// GetManufacturerSeed retorna o Seed registro para este pacote
func GetManufacturerSeed() Seed {
	return Seed{
		Name: "002_manufacturers",
		Fn:   SeedManufacturers,
	}
}
