//go:build linux && !android

package game

import (
	"bytes"
	"os"

	"github.com/coder/websocket"
)

const AppName = "boxcars-linux"

var dialOptions = &websocket.DialOptions{
	CompressionMode: websocket.CompressionContextTakeover,
}

func init() {
	if steamDeck() {
		AutoEnableTouchInput = true // Use mobile interface on Steam Deck.
	}
}

func steamDeck() bool {
	buf, err := os.ReadFile("/sys/devices/virtual/dmi/id/board_vendor")
	if err != nil {
		return false
	}
	return bytes.Equal(bytes.ToLower(bytes.TrimSpace(buf)), []byte("valve"))
}

func GetLocale() (string, error) {
	return os.Getenv("LANG"), nil
}

func DefaultFullscreen() bool {
	return steamDeck() // Default to fullscreen mode on Steam Deck.
}
