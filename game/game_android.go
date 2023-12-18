//go:build android

package game

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

func loadUsername() string {
	config := configPath()
	if config == "" {
		return ""
	}
	buf, err := os.ReadFile(config)
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(buf))
}

func saveUsername(username string) {
	config := configPath()
	if config == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(config), 0700)
	_ = os.WriteFile(config, []byte(username), 0600)
}

func configPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return path.Join(configDir, "boxcars", "config")
}
