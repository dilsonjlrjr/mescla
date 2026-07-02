package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/tray/icon.png
var trayIcon []byte

// Banco de catálogo embutido no binário (gerado pelo scripts/build-all.sh).
// Em dev o diretório fica vazio e o app usa o paint_knowledge.db do CWD.
//go:embed all:db/embedded
var embeddedDB embed.FS

func main() {
	seed, _ := embeddedDB.ReadFile("db/embedded/paint_knowledge.db")
	paintService, err := NewPaintService(seed)
	if err != nil {
		log.Fatalf("Erro inicializando PaintService: %v", err)
	}
	defer paintService.Close()

	app := application.New(application.Options{
		Name:        "Mescla",
		Description: "Ferramenta profissional para pintores de miniaturas",
		Services: []application.Service{
			application.NewService(paintService),
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
		BackgroundColour: application.NewRGB(10, 10, 15),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})

	confirmQuit := func() {
		dialog := app.Dialog.Question().
			SetTitle("Sair da Mescla").
			SetMessage("Tem certeza que deseja sair?")

		sair := dialog.AddButton("Sair")
		sair.OnClick(func() {
			// Quit fora do callback do dialog: chamado inline ele roda no meio
			// do event-loop do próprio dialog e o processo não termina (macOS).
			go app.Quit()
		})

		cancelar := dialog.AddButton("Cancelar")
		dialog.SetDefaultButton(cancelar)
		dialog.SetCancelButton(cancelar)
		dialog.Show()
	}

	// Fechar a janela (botão nativo) pede confirmação antes de sair.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		confirmQuit()
	})

	systray := app.SystemTray.New()
	systray.SetIcon(trayIcon)
	systray.SetLabel("Mescla")

	trayMenu := app.NewMenu()
	trayMenu.Add("Abrir Mescla").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Sair").OnClick(func(ctx *application.Context) {
		confirmQuit()
	})
	systray.SetMenu(trayMenu)

	systray.OnClick(func() {
		window.Show()
		window.Focus()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
