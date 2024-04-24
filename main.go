package main

//go:generate xgotext -no-locations -default boxcars -in . -out game/locales

import (
	"bufio"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

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

	g := parseFlags()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			g.Client.Out <- append(scanner.Bytes(), '\n')
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

	op := &ebiten.RunGameOptions{
		X11ClassName:    "boxcars",
		X11InstanceName: "boxcars",
	}
	if err := ebiten.RunGameWithOptions(g, op); err != nil {
		log.Fatal(err)
	}
}
