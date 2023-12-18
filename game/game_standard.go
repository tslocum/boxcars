//go:build (!js || !wasm) && !android

package game

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
)

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
