package main

//go:generate xgotext -no-locations -default boxcars -in . -out game/locales

import (
	"bufio"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"code.rocket9labs.com/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

func main() {
	ebiten.SetWindowTitle("bgammon.org - Free Online Backgammon")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(144)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	g := game.NewGame()

	parseFlags(g)

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
