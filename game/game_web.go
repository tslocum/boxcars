//go:build js && wasm

package game

import (
	"syscall/js"
)

const (
	DefaultServerAddress = "wss://ws.bgammon.org"
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = false
	ShowServerSettings   = false
	APPNAME              = "boxcars"
	fieldHeight          = 50
	defaultFontSize      = largeFontSize
)

func DefaultLocale() string {
	return js.Global().Get("navigator").Get("language").String()
}

func focused() bool {
	document := js.Global().Get("document")
	hasFocus := document.Call("hasFocus", nil)
	return hasFocus.Truthy()
}
