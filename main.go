package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	ebiten.SetMaxTPS(60)                // TODO allow users to set custom value
	ebiten.SetRunnableOnUnfocused(true) // Note - this currently does nothing in ebiten
	ebiten.SetWindowClosingHandled(true)

	fullscreenWidth, fullscreenHeight := ebiten.ScreenSizeInFullscreen()
	if fullscreenWidth <= screenWidth || fullscreenHeight <= screenHeight {
		ebiten.SetFullscreen(true)
	}

	g := game.NewGame()

	flag.StringVar(&g.Username, "username", "", "Username")
	flag.StringVar(&g.Password, "password", "", "Password")
	flag.StringVar(&g.ServerAddress, "address", "fibs.com:4321", "Server address")
	flag.BoolVar(&g.Watch, "watch", false, "Watch random game")
	flag.BoolVar(&g.TV, "tv", false, "Watch random games continuously")
	flag.Parse()

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
			// TODO temporary
			if string(scanner.Bytes()) == "/board" {
				g.Client.Out <- []byte("set boardstyle 2")
				time.Sleep(time.Second / 2)
				g.Client.Out <- []byte("board")
				time.Sleep(time.Second / 2)
				g.Client.Out <- []byte("set boardstyle 3")
				continue
			}

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
