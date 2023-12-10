//go:build (!js || !wasm) && !android

package game

const (
	DefaultServerAddress = "tcp://bgammon.org:1337"
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = false
	ShowServerSettings   = false
	APPNAME              = "boxcars"
	fieldHeight          = 50
	defaultFontSize      = largeFontSize
)

func DefaultLocale() string {
	return ""
}

func focused() bool {
	return true
}
