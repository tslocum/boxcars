//go:build js && wasm

package main

import (
	"code.rocket9labs.com/tslocum/boxcars/game"
	"golang.org/x/text/language"
)

func parseFlags() *game.Game {
	var forceLanguage *language.Tag
	locale, err := game.GetLocale()
	if err == nil && locale != "" {
		tag, err := language.Parse(locale)
		if err == nil {
			forceLanguage = &tag
		}
	}
	game.LoadLocale(forceLanguage)

	return game.NewGame()
}
