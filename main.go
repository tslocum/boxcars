package main

//go:generate xgotext -no-locations -default boxcars -in . -out game/locales

import (
	"image"
	"log"
	"os"
	"os/signal"
	"syscall"

	"code.rocket9labs.com/tslocum/boxcars/game"
	"code.rocket9labs.com/tslocum/gotext"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

func main() {
	g := parseFlags()

	ebiten.SetWindowTitle(gotext.Get("%s - Free Online Backgammon", "bgammon.org"))
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowIcon([]image.Image{game.ImgIconAlt})

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM)
	go func() {
		<-sigc

		g.Exit()
	}()

	op := &ebiten.RunGameOptions{
		X11ClassName:    "boxcars",
		X11InstanceName: "boxcars",
	}
	if err := ebiten.RunGameWithOptions(g, op); err != nil {
		log.Fatal(err)
	}
}
