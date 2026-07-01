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

func main() {
	paintService, err := NewPaintService()
	if err != nil {
		log.Fatalf("Erro inicializando PaintService: %v", err)
	}
	defer paintService.Close()

	app := application.New(application.Options{
		Name:        "Paint Match AI",
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
		Title:            "Paint Match AI",
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
			SetTitle("Sair do Paint Match AI").
			SetMessage("Tem certeza que deseja sair?")

		sair := dialog.AddButton("Sair")
		sair.OnClick(func() {
			app.Quit()
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
	systray.SetLabel("Paint Match AI")

	trayMenu := app.NewMenu()
	trayMenu.Add("Abrir Paint Match AI").OnClick(func(ctx *application.Context) {
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
