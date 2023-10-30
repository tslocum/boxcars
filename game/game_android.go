//go:build android

package game

import (
	"log"
	"os"
)

const (
	DefaultServerAddress = "wss://ws.bgammon.org"
	OptimizeDraws        = false
	AutoEnableTouchInput = true
	ShowServerSettings   = true
	APPNAME              = "boxcars-android"
)

func init() {
	log.SetOutput(os.Stdout)
}
