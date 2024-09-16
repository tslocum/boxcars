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
	AppName                = "boxcars-android"
	targetFPS              = 60
	ShowServerSettings     = true
	ShowQuitDialog         = true
	smallScreen            = true
	mobileDevice           = true
	enableOnScreenKeyboard = true
	enableRightClick       = false
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

	// Load locale early on Android.
	locale, err := GetLocale()
	if err != nil {
		return
	}
	tag, err := language.Parse(strings.TrimSpace(locale))
	if err != nil {
		return
	}
	LoadLocale(&tag)
}

func userConfigDir() string {
	return "/data/data/com.rocket9labs.boxcars"
}

func GetLocale() (string, error) {
	out, err := exec.Command("/system/bin/getprop", "persist.sys.locale").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func DefaultFullscreen() bool {
	return false
}

func ReplayDir() string {
	return ""
}
