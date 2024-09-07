//go:build android

package game

import (
	"os/exec"
	"strings"

	"golang.org/x/text/language"
)

func init() {
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

func GetLocale() (string, error) {
	out, err := exec.Command("/system/bin/getprop", "persist.sys.locale").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
