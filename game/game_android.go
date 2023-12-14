//go:build android

package game

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/text/language"
)

const (
	DefaultServerAddress = "wss://ws.bgammon.org"
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = true
	ShowServerSettings   = true
	APPNAME              = "boxcars-android"
	fieldHeight          = 100
	defaultFontSize      = extraLargeFontSize
)

func init() {
	log.SetOutput(os.Stdout)

	// Detect timezone.
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	tz, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	time.Local = tz

	// Detect locale.
	out, err = exec.Command("/system/bin/getprop", "persist.sys.locale").Output()
	if err != nil {
		return
	}
	tag, err := language.Parse(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	LoadLocale(&tag)
}

func DefaultLocale() string {
	return ""
}

func focused() bool {
	return true
}
