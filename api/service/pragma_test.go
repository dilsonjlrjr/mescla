package service

import (
	"path/filepath"
	"testing"
)

// A cascata depende de PRAGMA foreign_keys estar ligado em TODA conexão do
// pool. Executá-lo com db.Exec liga só na conexão que o rodou — foi assim que
// abas e regiões órfãs sobreviveram a DeletePlan. O pragma vive no DSN.
func TestForeignKeysLigadoEmTodasAsConexoes(t *testing.T) {
	svc, err := openPaintService(filepath.Join(t.TempDir(), "pragma.db"))
	if err != nil {
		t.Fatalf("openPaintService: %v", err)
	}
	defer svc.db.Close()

	// Mais conexões que uma força o pool a abrir novas.
	svc.db.SetMaxOpenConns(4)

	for i := 0; i < 8; i++ {
		var ligado int
		if err := svc.db.QueryRow("PRAGMA foreign_keys").Scan(&ligado); err != nil {
			t.Fatalf("lendo pragma na iteração %d: %v", i, err)
		}
		if ligado != 1 {
			t.Fatalf("iteração %d: foreign_keys deveria estar ligado, veio %d", i, ligado)
		}
	}
}
