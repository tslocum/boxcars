package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"syscall"

	"code.rocketnine.space/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

var AutoWatch bool // WASM only

func main() {
	ebiten.SetWindowTitle("Boxcars")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizable(true)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum)
	ebiten.SetMaxTPS(60)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	fullscreenWidth, fullscreenHeight := ebiten.ScreenSizeInFullscreen()
	if fullscreenWidth <= screenWidth || fullscreenHeight <= screenHeight {
		ebiten.SetFullscreen(true)
	}

	g := game.NewGame()

	parseFlags(g)

	if AutoWatch {
		g.Watch = true
	}

	// Auto-connect
	if g.Username != "" && g.Password != "" {
		g.Connect()
	}

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			g.Client.Out <- append(scanner.Bytes())
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM)
	go func() {
		<-sigc

		g.Exit()
	}()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
