//go:build !server

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func setupWindow(app *application.App) {
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TEI Studio — read-only PoC",
		Width: 1100,
		Height: 750,
	})
}
