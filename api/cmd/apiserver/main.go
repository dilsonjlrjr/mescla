// Command apiserver expõe api/service.PaintService por HTTP (fasthttp) — é o
// que front/ consome pela rede. wails/ NUNCA depende deste binário: continua
// offline, com bind direto ao mesmo PaintService dentro do processo desktop.
package main

import (
	"log"
	"os"

	apidb "paint-match-ai/api/db"
	"paint-match-ai/api/httpapi"
	"paint-match-ai/api/service"
)

func main() {
	addr := os.Getenv("MESCLA_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	svc, err := service.NewPaintService(apidb.EmbeddedSeed())
	if err != nil {
		log.Fatalf("inicializando PaintService: %v", err)
	}
	defer svc.Close()

	server := httpapi.NewServer(svc)
	log.Printf("mescla-api ouvindo em %s", addr)
	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("servidor HTTP: %v", err)
	}
}
