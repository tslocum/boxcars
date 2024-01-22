//go:build !js || !wasm

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"code.rocket9labs.com/tslocum/boxcars/game"
	"golang.org/x/text/language"
)

func parseFlags() *game.Game {
	var (
		username      string
		password      string
		serverAddress string
		locale        string
		tv            bool
		debug         int
	)
	flag.StringVar(&username, "username", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.StringVar(&serverAddress, "address", game.DefaultServerAddress, "Server address")
	flag.StringVar(&locale, "locale", "", "Use specified locale for translations")
	flag.BoolVar(&tv, "tv", false, "Watch random games continuously")
	flag.IntVar(&debug, "debug", 0, "Print debug information and serve pprof on specified port")
	flag.Parse()

	var forceLanguage *language.Tag
	if locale == "" {
		locale = game.DefaultLocale()
	}
	if locale != "" {
		tag, err := language.Parse(locale)
		if err != nil {
			log.Fatalf("unknown locale: %s", locale)
		}
		forceLanguage = &tag
	}
	game.LoadLocale(forceLanguage)

	g := game.NewGame()
	g.Username = username
	g.Password = password
	g.ServerAddress = serverAddress
	g.TV = tv

	if debug > 0 {
		game.Debug = 1
		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", debug), nil))
		}()
	}

	if len(flag.Args()) > 0 {
		replay, err := os.ReadFile(flag.Arg(0))
		if err != nil {
			log.Fatalf("failed to open replay file %s: %s", flag.Arg(0), err)
		}
		g.LoadReplay = replay
	}

	return g
}
