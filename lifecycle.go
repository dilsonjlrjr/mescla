package main

import "context"

type App struct {
	paintService *PaintService
}

func NewApp(ps *PaintService) *App {
	return &App{paintService: ps}
}

func (a *App) startup(ctx context.Context) {
}

func (a *App) shutdown(ctx context.Context) {
	a.paintService.Close()
}
