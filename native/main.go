package main

import (
	"embed"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := &App{}
	err := wails.Run(&options.App{
		Title:            "3MF Bridge",
		Width:            1030,
		Height:           700,
		MinWidth:         640,
		MinHeight:        460,
		BackgroundColour: &options.RGBA{R: 247, G: 246, B: 242, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
	})
	if err != nil {
		fmt.Println("3MF Bridge:", err)
	}
}
