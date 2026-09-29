// Command exportar-catalogo gera o banco que o app desktop embute
// (wails/catalogo/paint_knowledge.db) a partir do banco em uso na web.
//
// Leva o catálogo como está cadastrado: fabricantes, linhas, tintas, cores,
// tipos de tinta e a marca de ignorar na mistura. Deixa de fora tudo que é
// do usuário: estoque (user_paints e as origens da migração), projetos de
// pintura e receitas salvas. O desktop recria essas tabelas vazias no boot.
//
//	go run ./wails/cmd/exportar-catalogo -origem <banco da web> -destino wails/catalogo/paint_knowledge.db
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// Tabelas com dado do usuário, esvaziadas no destino. A ordem respeita as
// chaves estrangeiras: filho antes do pai.
var tabelasDoUsuario = []string{
	"user_paint_origins",
	"user_paints",
	"painting_region_ingredients",
	"painting_regions",
	"painting_tabs",
	"painting_plans",
	"saved_recipes",
}

func main() {
	origem := flag.String("origem", "", "banco da web (lido, nunca alterado)")
	destino := flag.String("destino", "wails/catalogo/paint_knowledge.db", "banco do desktop a gerar")
	flag.Parse()
	if *origem == "" {
		log.Fatal("informe -origem")
	}

	if err := exportar(*origem, *destino); err != nil {
		log.Fatal(err)
	}
}

func exportar(origem, destino string) error {
	if _, err := os.Stat(origem); err != nil {
		return fmt.Errorf("banco de origem: %w", err)
	}
	for _, f := range []string{destino, destino + "-wal", destino + "-shm"} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("apagando destino anterior: %w", err)
		}
	}

	src, err := sql.Open("sqlite", "file:"+origem+"?mode=ro")
	if err != nil {
		return err
	}
	defer src.Close()
	// VACUUM INTO copia um retrato consistente, inclusive o que ainda está no WAL.
	if _, err := src.Exec(`VACUUM INTO ?`, destino); err != nil {
		return fmt.Errorf("copiando banco: %w", err)
	}

	db, err := sql.Open("sqlite", destino)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, tabela := range tabelasDoUsuario {
		var existe int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, tabela).Scan(&existe); err != nil {
			return err
		}
		if existe == 0 {
			continue
		}
		if _, err := db.Exec(`DELETE FROM "` + tabela + `"`); err != nil {
			return fmt.Errorf("esvaziando %s: %w", tabela, err)
		}
		if _, err := db.Exec(`DELETE FROM sqlite_sequence WHERE name = ?`, tabela); err != nil {
			return fmt.Errorf("zerando sequência de %s: %w", tabela, err)
		}
	}

	if _, err := db.Exec(`PRAGMA journal_mode = DELETE`); err != nil {
		return err
	}
	if _, err := db.Exec(`VACUUM`); err != nil {
		return err
	}

	for _, tabela := range []string{"manufacturers", "paints", "paint_colors", "paint_types", "user_paints", "painting_plans"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + tabela + `"`).Scan(&n); err != nil {
			return err
		}
		fmt.Printf("%-16s %d\n", tabela, n)
	}
	return nil
}
