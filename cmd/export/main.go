// Command export lê o banco de catálogo (paint_knowledge.db) e emite o
// catálogo compacto em JSON para o app mobile (frontend-mobile). O JSON é
// carregado inteiro em memória no navegador; por isso as tintas vão como
// array-de-arrays (menos bytes que objetos com chaves repetidas 11.932 vezes).
//
// Formato:
//
//	{
//	  "manufacturers": [{"id":8,"name":"Acrilex","paintCount":95}, ...],
//	  "paints": [[id, mfrId, "name", "code", "line", r, g, b], ...]
//	}
//
// Lab/HSL não são exportados de propósito: o módulo WASM computa RGBToLab no
// init (instantâneo) — evita ~700KB de floats no payload.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type manufacturerJSON struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PaintCount int    `json:"paintCount"`
}

type catalogJSON struct {
	Manufacturers []manufacturerJSON `json:"manufacturers"`
	Paints        [][]any            `json:"paints"`
}

func main() {
	dbPath := flag.String("db", "paint_knowledge.db", "Caminho para o banco SQLite")
	outPath := flag.String("out", "frontend-mobile/public/data/catalog.json", "Arquivo JSON de saída")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("abrindo banco: %v", err)
	}
	defer db.Close()

	catalog, err := buildCatalog(db)
	if err != nil {
		log.Fatalf("montando catálogo: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		log.Fatalf("criando diretório de saída: %v", err)
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		log.Fatalf("serializando: %v", err)
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		log.Fatalf("gravando %s: %v", *outPath, err)
	}

	log.Printf("[export] %s: %d fabricantes, %d tintas, %d KB",
		*outPath, len(catalog.Manufacturers), len(catalog.Paints), len(data)/1024)
}

func buildCatalog(db *sql.DB) (catalogJSON, error) {
	var out catalogJSON

	// Só fabricantes com tintas coloridas entram — um fabricante sem cor não
	// serve nem como origem nem como destino no mobile.
	mfrRows, err := db.Query(`
		SELECT m.id, m.name, COUNT(pc.id)
		FROM manufacturers m
		JOIN paints p ON p.manufacturer_id = m.id
		JOIN paint_colors pc ON pc.paint_id = p.id
		GROUP BY m.id, m.name
		HAVING COUNT(pc.id) > 0
		ORDER BY m.name
	`)
	if err != nil {
		return out, fmt.Errorf("query manufacturers: %w", err)
	}
	defer mfrRows.Close()
	for mfrRows.Next() {
		var m manufacturerJSON
		if err := mfrRows.Scan(&m.ID, &m.Name, &m.PaintCount); err != nil {
			return out, err
		}
		out.Manufacturers = append(out.Manufacturers, m)
	}
	if err := mfrRows.Err(); err != nil {
		return out, err
	}

	// JOIN em paint_colors (não LEFT): tinta sem RGB não serve no mobile —
	// mesma regra do loadPaintsByManufacturerID do desktop.
	paintRows, err := db.Query(`
		SELECT p.id, p.manufacturer_id, p.name, COALESCE(p.code, ''),
		       COALESCE(pl.name, ''), pc.rgb_r, pc.rgb_g, pc.rgb_b
		FROM paints p
		JOIN paint_colors pc ON pc.paint_id = p.id
		LEFT JOIN product_lines pl ON pl.id = p.product_line_id
		ORDER BY p.manufacturer_id, p.name
	`)
	if err != nil {
		return out, fmt.Errorf("query paints: %w", err)
	}
	defer paintRows.Close()
	for paintRows.Next() {
		var (
			id, mfrID        int64
			name, code, line string
			r, g, b          uint8
		)
		if err := paintRows.Scan(&id, &mfrID, &name, &code, &line, &r, &g, &b); err != nil {
			return out, err
		}
		out.Paints = append(out.Paints, []any{id, mfrID, name, code, line, r, g, b})
	}
	return out, paintRows.Err()
}
