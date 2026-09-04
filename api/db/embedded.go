package db

import "embed"

// Banco de catálogo embutido no binário (gerado por scripts/build-all.sh e
// scripts/build-mobile.sh). Em dev o diretório fica vazio (só .gitkeep) e o
// app usa o paint_knowledge.db do CWD.
//
//go:embed all:embedded
var embeddedDB embed.FS

// EmbeddedSeed retorna os bytes do banco de catálogo embutido, ou nil se
// ainda não foi gerado (dev sem build).
func EmbeddedSeed() []byte {
	data, _ := embeddedDB.ReadFile("embedded/paint_knowledge.db")
	return data
}
