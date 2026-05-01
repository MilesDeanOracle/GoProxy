package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	err = wails.Run(&options.App{
		Title:            "GoProxy - V1.0.0",
		Width:            1080,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		MaxWidth:         1080,
		BackgroundColour: options.NewRGBA(245, 247, 250, 255),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("run app: %v", err)
	}
}
