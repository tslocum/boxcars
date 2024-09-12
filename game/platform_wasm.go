//go:build js && wasm

package game

import (
	"syscall/js"

	"github.com/coder/websocket"
)

var dialOptions = &websocket.DialOptions{}

func GetLocale() (string, error) {
	return js.Global().Get("navigator").Get("language").String(), nil
}

func DefaultFullscreen() bool {
	return false
}
