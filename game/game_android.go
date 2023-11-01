//go:build android

package game

import (
	"log"
	"os"
)

const (
	DefaultServerAddress = "wss://ws.bgammon.org"
	OptimizeDraw         = false
	OptimizeSetRect      = false
	AutoEnableTouchInput = true
	ShowServerSettings   = true
	APPNAME              = "boxcars-android"
)

func init() {
	log.SetOutput(os.Stdout)
}
