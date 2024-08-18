//go:build !js || !wasm

package main

import (
	"flag"
	"log"
	"os"

	"code.rocket9labs.com/tslocum/boxcars/game"
	"golang.org/x/text/language"
)

func parseFlags() *game.Game {
	var (
		username      string
		password      string
		serverAddress string
		mute          bool
		instant       bool
		locale        string
		join          int
		tv            bool
		debug         int
	)
	flag.StringVar(&username, "username", "", "Username")
	flag.StringVar(&password, "password", "", "Password")
	flag.StringVar(&serverAddress, "address", game.DefaultServerAddress, "Server address")
	flag.BoolVar(&mute, "mute", false, "Mute sound effects")
	flag.BoolVar(&instant, "instant", false, "Instant checker movement")
	flag.StringVar(&locale, "locale", "", "Use specified locale for translations")
	flag.IntVar(&join, "join", 0, "Connect as guest and join specified match")
	flag.BoolVar(&tv, "tv", false, "Watch random games continuously")
	flag.IntVar(&debug, "debug", 0, "Debug level")
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
	g.Mute = mute
	g.Instant = instant
	g.JoinGame = join
	g.TV = tv

	if debug > 0 {
		game.Debug = int8(debug)
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
