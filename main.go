package main

import (
	"log"

	"code.rocketnine.space/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

func main() {
	ebiten.SetWindowTitle("Boxcars")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizable(true)
	ebiten.SetMaxTPS(60)                // TODO allow users to set custom value
	ebiten.SetRunnableOnUnfocused(true) // Note - this currently does nothing in ebiten

	//ebiten.SetWindowClosingHandled(true) TODO implement

	fullscreenWidth, fullscreenHeight := ebiten.ScreenSizeInFullscreen()
	if fullscreenWidth <= screenWidth || fullscreenHeight <= screenHeight {
		ebiten.SetFullscreen(true)
	}

	if err := ebiten.RunGame(game.NewGame()); err != nil {
		log.Fatal(err)
	}
}
