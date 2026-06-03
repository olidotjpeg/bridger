package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct{ ctx context.Context }

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) emitScanDone() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "scan:done")
}

func (a *App) PickFolder() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select photo folder",
	})
	if err != nil {
		return ""
	}
	return path
}
