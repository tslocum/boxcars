//go:build android

package game

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
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

	// Android timezone issue workaround.
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	tz, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	time.Local = tz
}
