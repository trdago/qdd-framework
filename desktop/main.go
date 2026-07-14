package main

import (
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/qdd-framework/qdd/ui"
)

func main() {
	// Create an instance of the app structure
	app := NewApp()

	distFs, _ := fs.Sub(ui.StaticFiles, "dist")

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "QDD Desktop",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: distFs,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
