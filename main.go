package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"paint-match-ai/db/seeds"

	_ "modernc.org/sqlite"
)

func main() {
	// Determinar caminho do banco
	dbPath := "paint_knowledge.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	// Garantir que o diretório existe
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Erro criando diretório: %v", err)
		}
	}

	// Abrir banco
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Erro abrindo banco: %v", err)
	}
	defer db.Close()

	// Configurar pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Fatalf("Erro executando pragma: %v", err)
		}
	}

	// Verificar se as tabelas existem
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
	if err != nil {
		log.Fatalf("Erro verificando tabelas: %v", err)
	}
	if tableCount == 0 {
		log.Fatal("Banco vazio. Execute a migração primeiro: sqlite3 paint_knowledge.db < db/migrations/001_initial_schema.sql")
	}

	log.Printf("[seed] Banco: %s (%d tabelas)", dbPath, tableCount)

	// Executar seeds
	allSeeds := []seeds.Seed{
		seeds.GetManufacturerSeed(),
	}

	if err := seeds.RunAll(db, allSeeds); err != nil {
		log.Fatalf("Erro executando seeds: %v", err)
	}

	log.Println("[seed] Todos os seeds concluídos com sucesso")
}
