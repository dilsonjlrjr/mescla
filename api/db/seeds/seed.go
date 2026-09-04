package seeds

import (
	"database/sql"
	"fmt"
	"log"
)

// SeedFunc define a assinatura de uma função de seed
type SeedFunc func(db *sql.DB) error

// Seed representa um registro de seed
type Seed struct {
	Name     string
	Fn       SeedFunc
}

// Run executa um seed e reporta resultado
func Run(db *sql.DB, s Seed) error {
	log.Printf("[seed] Executando: %s", s.Name)
	if err := s.Fn(db); err != nil {
		return fmt.Errorf("seed %s falhou: %w", s.Name, err)
	}
	log.Printf("[seed] Concluído: %s", s.Name)
	return nil
}

// RunAll executa múltiplos seeds em sequência
func RunAll(db *sql.DB, seeds []Seed) error {
	for _, s := range seeds {
		if err := Run(db, s); err != nil {
			return err
		}
	}
	return nil
}
