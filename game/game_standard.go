//go:build (!js || !wasm) && !android

package game

const (
	DefaultServerAddress = "tcp://bgammon.org:1337"
	OptimizeDraws        = true
	AutoEnableTouchInput = false
	APPNAME              = "boxcars"
)
