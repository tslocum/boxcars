//go:build windows

package game

import (
	"os"
	"path"
)

const (
	AppName            = "boxcars-windows"
	ShowServerSettings = false
	ShowQuitDialog     = true
	targetFPS          = 144
)

var (
	smallScreen            = false
	mobileDevice           = false
	enableOnScreenKeyboard = false
	enableRightClick       = true
)

func userConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return path.Join(configDir, "boxcars")
}

func DefaultFullscreen() bool {
	return false
}

func ReplayDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return path.Join(homeDir, "boxcars")
}
