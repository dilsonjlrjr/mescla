package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"

	"paint-match-ai/db/seeds"

	_ "modernc.org/sqlite"
)

func main() {
	// Flags
	dbPath := flag.String("db", "data/paint_knowledge.db", "Caminho para o banco SQLite")
	downloadLogos := flag.Bool("download-logos", false, "Baixar logotipos dos fabricantes")
	downloadAssets := flag.Bool("download-assets", false, "Baixar todos os assets (logos, thumbnails, imagens)")
	importPaints := flag.Bool("import-paints", false, "Importar tintas de demonstração (amostra curada)")
	importCatalog := flag.Bool("import-catalog", false, "Importar catálogo completo dos arquivos de dados (db/data/paints)")
	dataDir := flag.String("data-dir", seeds.DefaultCatalogDir, "Diretório com os .md de catálogo (para -import-catalog)")
	generateSwatches := flag.Bool("generate-swatches", false, "Gerar imagens de swatch para todas as tintas")
	flag.Parse()

	// Garantir que o diretório do banco existe
	dir := filepath.Dir(*dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Erro criando diretório: %v", err)
		}
	}

	// Abrir banco
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("Erro abrindo banco: %v", err)
	}
	defer db.Close()

	// Configurar pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
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

	log.Printf("[seed] Banco: %s (%d tabelas)", *dbPath, tableCount)

	// Seeds de dados (sempre executam)
	dataSeeds := []seeds.Seed{
		seeds.GetManufacturerSeed(),
	}

	// Importar tintas se solicitado
	if *importPaints {
		dataSeeds = append(dataSeeds, seeds.GetPaintImporterSeed())
	}

	// Importar catálogo completo (dataset) se solicitado
	if *importCatalog {
		dataSeeds = append(dataSeeds, seeds.GetCatalogImporterSeed(*dataDir))
	}

	if err := seeds.RunAll(db, dataSeeds); err != nil {
		log.Fatalf("Erro executando seeds de dados: %v", err)
	}

	// Seeds de assets (só se solicitado)
	if *downloadLogos || *downloadAssets {
		log.Println("[asset] Iniciando download de assets...")
		assetSeed := seeds.GetAssetDownloaderSeed()
		if err := seeds.Run(db, assetSeed); err != nil {
			log.Fatalf("Erro baixando assets: %v", err)
		}
	}

	// Gerar swatches se solicitado
	if *generateSwatches {
		log.Println("[swatch] Iniciando geração de swatches...")
		swatchSeed := seeds.GetSwatchGeneratorSeed()
		if err := seeds.Run(db, swatchSeed); err != nil {
			log.Fatalf("Erro gerando swatches: %v", err)
		}
	}

	log.Println("[seed] Concluído com sucesso")
}
