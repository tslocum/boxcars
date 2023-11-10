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
)

func DefaultLocale() string {
	return js.Global().Get("navigator").Get("language").String()
}
