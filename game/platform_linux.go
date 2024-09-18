//go:build linux && !android

package game

import (
	"bytes"
	"os"
	"path"
)

const (
	targetFPS              = 144
	ShowServerSettings     = false
	ShowQuitDialog         = true
	smallScreen            = false
	enableOnScreenKeyboard = false
	enableRightClick       = true
)

var (
	steamDeck    = isSteamDeck()
	AppName      = appName()
	mobileDevice = steamDeck
)

func userConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return path.Join(configDir, "boxcars")
}

func isSteamDeck() bool {
	buf, err := os.ReadFile("/sys/devices/virtual/dmi/id/board_vendor")
	if err != nil {
		return false
	}
	return bytes.Equal(bytes.ToLower(bytes.TrimSpace(buf)), []byte("valve"))
}

func appName() string {
	platform := "linux"
	if steamDeck {
		platform = "steamdeck"
	}
	return "boxcars-" + platform
}

func GetLocale() (string, error) {
	return os.Getenv("LANG"), nil
}

func DefaultFullscreen() bool {
	return steamDeck // Default to fullscreen mode on Steam Deck.
}

func ReplayDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return path.Join(homeDir, ".local", "share", "boxcars")
}
