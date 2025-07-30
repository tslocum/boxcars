//go:build js && wasm

package main

import (
	"codeberg.org/tslocum/boxcars/game"
)

func parseFlags() *game.Game {
	locale, err := game.GetLocale()
	if err != nil {
		locale = ""
	}
	game.LoadLocale(locale)

	return game.NewGame()
}
