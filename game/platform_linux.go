//go:build linux && !android

package game

import (
	"bytes"
	"os"
)

func GetLocale() (string, error) {
	return os.Getenv("LANG"), nil
}

func DefaultFullscreen() bool {
	buf, err := os.ReadFile("/sys/devices/virtual/dmi/id/board_vendor")
	if err != nil {
		return false
	}

	steamDevice := bytes.Equal(bytes.ToLower(bytes.TrimSpace(buf)), []byte("valve"))
	return steamDevice // Default to fullscreen mode on Steam Deck.
}
