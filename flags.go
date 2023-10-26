//go:build !js || !wasm

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"code.rocket9labs.com/tslocum/boxcars/game"
)

func parseFlags(g *game.Game) {
	var (
		debug int
		touch bool
	)
	flag.StringVar(&g.Username, "username", "", "Username")
	flag.StringVar(&g.Password, "password", "", "Password")
	flag.StringVar(&g.ServerAddress, "address", game.DefaultServerAddress, "Server address")
	flag.BoolVar(&g.Watch, "watch", false, "Watch random game")
	flag.BoolVar(&g.TV, "tv", false, "Watch random games continuously")
	flag.BoolVar(&touch, "touch", false, "Force touch input related interface elements to be displayed")
	flag.IntVar(&debug, "debug", 0, "Print debug information and serve pprof on specified port")
	flag.Parse()

	if touch {
		g.TouchInput = true
	}

	if debug > 0 {
		game.Debug = 1
		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", debug), nil))
		}()
	}
}
