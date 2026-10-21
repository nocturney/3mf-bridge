package main

import (
	"context"
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
	preferences, prefErr := loadSettings()
	if prefErr != nil {
		fmt.Println("3MF Bridge: could not restore settings:", prefErr)
		preferences = defaultSettings()
	}
	err := wails.Run(&options.App{
		Title:            "3MF Bridge",
		Width:            preferences.Window.Width,
		Height:           preferences.Window.Height,
		MinWidth:         720,
		MinHeight:        500,
		BackgroundColour: &options.RGBA{R: 247, G: 246, B: 242, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnBeforeClose: func(_ context.Context) bool {
			// Flush the final user-chosen size even if the 900ms JS debounce
			// has not fired yet; never prevent the OS close command.
			_ = app.RememberWindowSize()
			return false
		},
		Bind:        []interface{}{app},
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true},
	})
	if err != nil {
		fmt.Println("3MF Bridge:", err)
	}
}
