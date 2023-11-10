package main

//go:generate xgotext -no-locations -default boxcars -in . -out game/locales

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"code.rocket9labs.com/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"
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

	var (
		username      string
		password      string
		serverAddress string
		locale        string
		watch         bool
		tv            bool
		debug         int
		touch         bool
	)
	flag.StringVar(&username, "username", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.StringVar(&serverAddress, "address", game.DefaultServerAddress, "Server address")
	flag.StringVar(&locale, "locale", "", "Use specified locale for translations")
	flag.BoolVar(&watch, "watch", false, "Watch random game")
	flag.BoolVar(&tv, "tv", false, "Watch random games continuously")
	flag.BoolVar(&touch, "touch", false, "Force touch input related interface elements to be displayed")
	flag.IntVar(&debug, "debug", 0, "Print debug information and serve pprof on specified port")
	flag.Parse()

	var forceLanguage *language.Tag
	if locale != "" {
		tag, err := language.Parse(locale)
		if err != nil {
			log.Fatalf("unknown locale: %s", locale)
		}
		forceLanguage = &tag
	}
	game.LoadLocales(forceLanguage)

	g := game.NewGame()
	g.Username = username
	g.Password = password
	g.ServerAddress = serverAddress
	g.Watch = watch
	g.TV = tv

	if touch {
		g.EnableTouchInput()
	}

	if debug > 0 {
		game.Debug = 1
		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", debug), nil))
		}()
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
