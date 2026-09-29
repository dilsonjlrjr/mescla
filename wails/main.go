package main

import (
	"embed"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	apidb "paint-match-ai/api/db"
	"paint-match-ai/api/service"
)

//go:embed all:frontend/dist
var assets embed.FS

// Ícone do Mescla (a gema da marca, o mesmo do PWA). No macOS e no Windows
// o ícone do executável vem de build/darwin/icons.icns e build/windows/icon.ico;
// este aqui vale para a caixa "Sobre". No Linux (GTK4) o ícone vem do
// build/linux/org.wails.Mescla.desktop.
//
//go:embed build/appicon.png
var appIcon []byte

func main() {
	seed := apidb.EmbeddedSeed()
	paintService, err := service.NewPaintService(seed)
	if err != nil {
		log.Fatalf("Erro inicializando PaintService: %v", err)
	}
	defer paintService.Close()

	dialogService := NewDialogService()

	app := application.New(application.Options{
		Name:        "Mescla",
		Description: "Ferramenta profissional para pintores de miniaturas",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(paintService),
			application.NewService(dialogService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Mescla",
		Width:            1280,
		Height:           800,
		MinWidth:         1024,
		MinHeight:        700,
		BackgroundColour: application.NewRGB(22, 24, 38), // #161826, fundo da interface
		// Barra de título nativa também no macOS: a interface é a da web, que
		// não reserva espaço para os semáforos nem marca área de arrastar.
		URL: "/",
	})

	// Fechar a janela encerra o app — sem diálogo, sem bandeja. O hook NÃO
	// cancela o fechamento; ele só arma um watchdog: o teardown do Wails v3
	// (alpha) no Windows é assíncrono e às vezes não completa, deixando o
	// processo vivo sem janela. Se em 2s o encerramento normal não terminou,
	// força a saída. Sem risco de dados: o app só lê o banco em runtime.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		go func() {
			time.Sleep(2 * time.Second)
			os.Exit(0)
		}()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
