//go:build !js || !wasm

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"codeberg.org/tslocum/bgammon"
	"codeberg.org/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"
)

func fetchMatches(matches []*bgammon.GameListing) []*bgammon.GameListing {
	url := "https://bgammon.org/api/matches.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "bgammon-tv")

	client := http.Client{
		Timeout: time.Second * 10,
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, &matches)
	if err != nil {
		log.Fatal(err)
	}
	return matches
}

func openBoxcars(boxcarsPath string, matchID int) {
	cmd := exec.Command(boxcarsPath, "--join", strconv.Itoa(matchID), "--mute")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() *game.Game {
	var (
		username      string
		password      string
		serverAddress string
		mute          bool
		instant       bool
		fullscreen    bool
		windowed      bool
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
	flag.BoolVar(&fullscreen, "fullscreen", false, "Start in fullscreen mode")
	flag.BoolVar(&windowed, "windowed", false, "Start in windowed mode")
	flag.StringVar(&locale, "locale", "", "Use specified locale for translations")
	flag.IntVar(&join, "join", 0, "Connect as guest and join specified match")
	flag.BoolVar(&tv, "tv", false, "Spectate games continuously")
	flag.IntVar(&debug, "debug", 0, "Debug level")
	flag.Parse()

	if game.DefaultFullscreen() {
		fullscreen = true
	}

	var forceLanguage *language.Tag
	if locale == "" {
		var err error
		locale, err = game.GetLocale()
		if err != nil {
			locale = ""
		}
	}
	if locale != "" {
		dotIndex := strings.IndexByte(locale, '.')
		if dotIndex != -1 {
			locale = locale[:dotIndex]
		}
		tag, err := language.Parse(locale)
		if err == nil {
			forceLanguage = &tag
		}
	}
	game.LoadLocale(forceLanguage)

	g := game.NewGame()
	g.Username = username
	g.Password = password
	g.ServerAddress = serverAddress
	g.Mute = mute
	g.Instant = instant
	g.JoinGame = join

	if fullscreen && !windowed {
		g.Fullscreen = true
		ebiten.SetFullscreen(true)
	}

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

	if tv {
		log.Println("Watching bgammon.org TV...")

		boxcarsPath, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		} else if boxcarsPath == "" {
			boxcarsPath = "boxcars"
		}

		var matches []*bgammon.GameListing
		foundGames := make(map[int]bool)
		t := time.NewTicker(15 * time.Second)
		for {
			matches = fetchMatches(matches[:0])
			for _, match := range matches {
				if match.Password || match.Players != 2 || foundGames[match.ID] {
					continue
				}
				openBoxcars(boxcarsPath, match.ID)
				foundGames[match.ID] = true
			}
			<-t.C
		}
	}

	return g
}
