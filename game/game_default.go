//go:build !windows

package game

import (
	"os"
	"path"
)

func ReplayDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return path.Join(homeDir, ".local", "share", "boxcars")
}
