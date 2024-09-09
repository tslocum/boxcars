//go:build js && wasm

package game

import "syscall/js"

func GetLocale() (string, error) {
	return js.Global().Get("navigator").Get("language").String(), nil
}

func DefaultFullscreen() bool {
	return false
}
